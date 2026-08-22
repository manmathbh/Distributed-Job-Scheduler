package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorker_ProcessesJobsAndHeartbeats(t *testing.T) {
	st := store.NewMemoryStore()
	svc := service.New(st, metrics.NewRegistry())
	ctx := context.Background()

	p, err := svc.CreateProject(ctx, "owner", "P", "")
	require.NoError(t, err)
	q, err := svc.CreateQueue(ctx, p.ID, service.QueueConfig{Name: "q", Concurrency: 10, MaxAttempts: 3})
	require.NoError(t, err)

	job, err := svc.SubmitJob(ctx, service.SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID})
	require.NoError(t, err)

	w := New(st, svc, metrics.NewRegistry(), Config{
		ID:                "test-worker",
		Concurrency:       2,
		PollInterval:      10 * time.Millisecond,
		LeaseDuration:     time.Minute,
		HeartbeatInterval: 20 * time.Millisecond,
		Handler: HandlerFunc(func(ctx context.Context, j *models.Job) (json.RawMessage, error) {
			return j.Payload, nil
		}),
	})

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- w.Run(runCtx) }()

	// Wait for completion.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j, err := svc.GetJob(ctx, job.ID)
		if err == nil && j.Status == models.JobStatusCompleted {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	got, err := svc.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, got.Status)

	// Heartbeat should have registered the worker.
	wkr, err := st.GetWorker(ctx, "test-worker")
	require.NoError(t, err)
	assert.Equal(t, models.WorkerStatusActive, wkr.Status)

	// Graceful shutdown.
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not shut down")
	}
}

func TestWorker_RespectsQueueConcurrency(t *testing.T) {
	st := store.NewMemoryStore()
	svc := service.New(st, metrics.NewRegistry())
	ctx := context.Background()

	p, _ := svc.CreateProject(ctx, "owner", "P", "")
	q, _ := svc.CreateQueue(ctx, p.ID, service.QueueConfig{Name: "q", Concurrency: 1, MaxAttempts: 3})

	for i := 0; i < 5; i++ {
		_, err := svc.SubmitJob(ctx, service.SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID})
		require.NoError(t, err)
	}

	// Claim one job directly; queue concurrency of 1 means the worker cannot
	// claim a second until the first completes.
	claimed, err := svc.ClaimJob(ctx, q.ID, "manual-worker")
	require.NoError(t, err)

	// A second claim attempt must fail (concurrency exhausted).
	_, err = svc.ClaimJob(ctx, q.ID, "another-worker")
	require.ErrorIs(t, err, store.ErrNoJobs)

	// Complete the first job, freeing capacity.
	exec, err := svc.RecordExecutionStart(ctx, claimed)
	require.NoError(t, err)
	require.NoError(t, svc.CompleteJob(ctx, claimed, exec, json.RawMessage(`{}`)))

	_, err = svc.ClaimJob(ctx, q.ID, "another-worker")
	require.NoError(t, err)
}

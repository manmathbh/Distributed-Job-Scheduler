package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService() (*Service, *store.MemoryStore) {
	st := store.NewMemoryStore()
	return New(st, metrics.NewRegistry()), st
}

func mustCreateProjectQueue(t *testing.T, svc *Service, owner string) (*models.Project, *models.Queue) {
	t.Helper()
	ctx := context.Background()
	p, err := svc.CreateProject(ctx, owner, "Test Project", "")
	require.NoError(t, err)
	q, err := svc.CreateQueue(ctx, p.ID, QueueConfig{
		Name: "default", Concurrency: 5, MaxAttempts: 3,
		RetryStrategy: models.RetryStrategyFixed, InitialDelay: time.Second, MaxDelay: time.Second, Multiplier: 1,
	})
	require.NoError(t, err)
	return p, q
}

func TestSubmitJob_Immediate(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")

	job, err := svc.SubmitJob(context.Background(), SubmitJobRequest{
		ProjectID: p.ID, QueueID: q.ID, Type: models.JobTypeImmediate, Payload: json.RawMessage(`{"x":1}`),
	})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusQueued, job.Status)
	assert.Equal(t, 3, job.MaxAttempts)
	assert.Equal(t, models.RetryStrategyFixed, job.RetryStrategy)
}

func TestSubmitJob_DelayedAndScheduled(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	ctx := context.Background()

	delayed, err := svc.SubmitJob(ctx, SubmitJobRequest{
		ProjectID: p.ID, QueueID: q.ID, Type: models.JobTypeDelayed, Delay: time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, delayed.Status)

	future := time.Now().Add(time.Hour)
	scheduled, err := svc.SubmitJob(ctx, SubmitJobRequest{
		ProjectID: p.ID, QueueID: q.ID, Type: models.JobTypeScheduled, ScheduledAt: &future,
	})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, scheduled.Status)
}

func TestSubmitJob_PausedQueue(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	_, err := svc.SetQueueStatus(context.Background(), q.ID, models.QueueStatusPaused)
	require.NoError(t, err)

	_, err = svc.SubmitJob(context.Background(), SubmitJobRequest{
		ProjectID: p.ID, QueueID: q.ID, Type: models.JobTypeImmediate,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "paused")
}

func TestSubmitJob_InvalidPayload(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")

	_, err := svc.SubmitJob(context.Background(), SubmitJobRequest{
		ProjectID: p.ID, QueueID: q.ID, Payload: json.RawMessage(`not-json`),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid JSON")
}

func TestJobLifecycle_Complete(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	ctx := context.Background()

	job, err := svc.SubmitJob(ctx, SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID})
	require.NoError(t, err)

	claimed, err := svc.ClaimJob(ctx, q.ID, "worker-1")
	require.NoError(t, err)
	assert.Equal(t, job.ID, claimed.ID)
	assert.Equal(t, models.JobStatusClaimed, claimed.Status)
	assert.Equal(t, 1, claimed.Attempts)

	exec, err := svc.RecordExecutionStart(ctx, claimed)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusRunning, exec.Status)

	err = svc.CompleteJob(ctx, claimed, exec, json.RawMessage(`{"done":true}`))
	require.NoError(t, err)

	got, err := svc.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, got.Status)
	assert.NotNil(t, got.CompletedAt)

	execs, err := svc.Store().ListExecutions(ctx, job.ID)
	require.NoError(t, err)
	assert.Len(t, execs, 1)
	assert.Equal(t, models.JobStatusCompleted, execs[0].Status)
}

func TestJobLifecycle_Fail_Retry(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	ctx := context.Background()

	job, err := svc.SubmitJob(ctx, SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID})
	require.NoError(t, err)

	claimed, _ := svc.ClaimJob(ctx, q.ID, "worker-1")
	exec, _ := svc.RecordExecutionStart(ctx, claimed)

	// First failure -> retry (attempts=1 < max=3)
	err = svc.FailJob(ctx, claimed, exec, "boom")
	require.NoError(t, err)

	got, err := svc.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, got.Status, "failed job with attempts remaining should be retry-scheduled")
	assert.Equal(t, "boom", got.LastError)
	assert.False(t, got.AvailableAt.Before(time.Now()), "retry should be scheduled in the future")
}

func TestJobLifecycle_Fail_DLQ(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	ctx := context.Background()

	// max_attempts=1 means first failure is permanent.
	job, err := svc.SubmitJob(ctx, SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID, MaxAttempts: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, job.MaxAttempts)

	claimed, _ := svc.ClaimJob(ctx, q.ID, "worker-1")
	exec, _ := svc.RecordExecutionStart(ctx, claimed)

	err = svc.FailJob(ctx, claimed, exec, "permanent failure")
	require.NoError(t, err)

	got, err := svc.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusFailed, got.Status)

	entries, _, err := svc.ListDLQ(ctx, store.ListDLQFilter{ProjectID: p.ID, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "permanent failure", entries[0].Reason)
	assert.Equal(t, 1, entries[0].Attempts)
}

func TestRetryJob_And_RequeueDLQ(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	ctx := context.Background()

	job, _ := svc.SubmitJob(ctx, SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID, MaxAttempts: 1})
	claimed, _ := svc.ClaimJob(ctx, q.ID, "worker-1")
	exec, _ := svc.RecordExecutionStart(ctx, claimed)
	require.NoError(t, svc.FailJob(ctx, claimed, exec, "dead"))

	// Retry via job endpoint.
	retried, err := svc.RetryJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusQueued, retried.Status)
	assert.Equal(t, 0, retried.Attempts)

	// Requeue via DLQ.
	entries, _, _ := svc.ListDLQ(ctx, store.ListDLQFilter{ProjectID: p.ID, Limit: 10})
	require.Len(t, entries, 1)
	requeued, err := svc.RequeueDLQ(ctx, entries[0].ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusQueued, requeued.Status)
	assert.NotEqual(t, job.ID, requeued.ID, "requeue creates a fresh job")
}

func TestCreateSchedule(t *testing.T) {
	svc, _ := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")

	sj, err := svc.CreateSchedule(context.Background(), SubmitJobRequest{
		ProjectID: p.ID, QueueID: q.ID, Type: models.JobTypeRecurring,
		CronExpr: "*/5 * * * *", Payload: json.RawMessage(`{"cron":true}`),
	})
	require.NoError(t, err)
	assert.NotNil(t, sj.NextRunAt)
	assert.Equal(t, "*/5 * * * *", sj.CronExpr)

	schedules, err := svc.ListSchedules(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Len(t, schedules, 1)
}

func TestFailJob_StaleWorkerCannotModifyReclaimedJob(t *testing.T) {
	svc, st := newTestService()
	p, q := mustCreateProjectQueue(t, svc, "owner")
	ctx := context.Background()

	job, err := svc.SubmitJob(ctx, SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID, MaxAttempts: 5})
	require.NoError(t, err)

	claimedA, err := svc.ClaimJob(ctx, q.ID, "worker-A")
	require.NoError(t, err)
	execA, err := svc.RecordExecutionStart(ctx, claimedA)
	require.NoError(t, err)

	// Simulate lease expiry, recovery, and re-claim by worker B.
	j, err := st.GetJob(ctx, job.ID)
	require.NoError(t, err)
	past := time.Now().Add(-time.Minute)
	j.LeaseExpiresAt = &past
	require.NoError(t, st.UpdateJob(ctx, j))
	_, err = st.RecoverExpiredLeases(ctx, time.Now())
	require.NoError(t, err)

	claimedB, err := svc.ClaimJob(ctx, q.ID, "worker-B")
	require.NoError(t, err)
	_, err = svc.RecordExecutionStart(ctx, claimedB)
	require.NoError(t, err)

	// Stale worker A must not be able to fail the job it no longer owns.
	err = svc.FailJob(ctx, claimedA, execA, "stale failure")
	require.Error(t, err)

	got, err := st.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusRunning, got.Status, "job must remain running under worker B")
	require.NotNil(t, got.WorkerID)
	assert.Equal(t, "worker-B", *got.WorkerID)
}

func TestMaterializeScheduledJob_InheritsRetryPolicy(t *testing.T) {
	svc, st := newTestService()
	ctx := context.Background()
	p, err := svc.CreateProject(ctx, "owner", "P", "")
	require.NoError(t, err)
	q, err := svc.CreateQueue(ctx, p.ID, QueueConfig{
		Name:          "q",
		Concurrency:   3,
		MaxAttempts:   7,
		RetryStrategy: models.RetryStrategyLinear,
		InitialDelay:  2 * time.Second,
		MaxDelay:      45 * time.Second,
		Multiplier:    1,
	})
	require.NoError(t, err)

	sj := &models.ScheduledJob{
		ID: "sched-1", ProjectID: p.ID, QueueID: q.ID, Name: "cron",
		CronExpr: "*/5 * * * *", Timezone: "UTC", Payload: []byte(`{}`),
		Priority: 5, Enabled: true,
	}
	require.NoError(t, st.CreateScheduledJob(ctx, sj))

	job, err := svc.MaterializeScheduledJob(ctx, sj)
	require.NoError(t, err)
	assert.Equal(t, models.JobTypeRecurring, job.Type)
	assert.Equal(t, 7, job.MaxAttempts)
	assert.Equal(t, models.RetryStrategyLinear, job.RetryStrategy)
	assert.Equal(t, 2*time.Second, job.RetryInitialDelay)
	assert.Equal(t, 45*time.Second, job.RetryMaxDelay)
	assert.Equal(t, 5, job.Priority)

	// A recurring job must retry (not permanently fail) using the inherited policy.
	claimed, err := svc.ClaimJob(ctx, q.ID, "worker")
	require.NoError(t, err)
	exec, err := svc.RecordExecutionStart(ctx, claimed)
	require.NoError(t, err)
	require.NoError(t, svc.FailJob(ctx, claimed, exec, "transient error"))

	got, err := svc.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, got.Status, "recurring job should be retry-scheduled, not failed")
}

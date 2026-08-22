package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/db"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/stretchr/testify/require"
)

func postgresService(t *testing.T) *Service {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	require.NoError(t, err)
	require.NoError(t, db.Migrate(ctx, pool))
	t.Cleanup(pool.Close)
	return New(store.NewPostgresStore(pool), metrics.NewRegistry())
}

// TestService_CreateProject_Postgres regresses the organization foreign-key
// bug: project creation must ensure its owning organization exists, and the
// full happy path must work against real PostgreSQL.
func TestService_CreateProject_Postgres(t *testing.T) {
	svc := postgresService(t)
	ctx := context.Background()
	ownerID := fmt.Sprintf("org-%d", time.Now().UnixNano())

	p, err := svc.CreateProject(ctx, ownerID, "PG Project", "desc")
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.DeleteProject(ctx, p.ID) })

	q, err := svc.CreateQueue(ctx, p.ID, QueueConfig{
		Name: "default", Concurrency: 4, MaxAttempts: 3,
		RetryStrategy: models.RetryStrategyExponential,
	})
	require.NoError(t, err)

	job, err := svc.SubmitJob(ctx, SubmitJobRequest{ProjectID: p.ID, QueueID: q.ID, Payload: json.RawMessage(`{"k":"v"}`)})
	require.NoError(t, err)

	claimed, err := svc.ClaimJob(ctx, q.ID, "worker-1")
	require.NoError(t, err)
	exec, err := svc.RecordExecutionStart(ctx, claimed)
	require.NoError(t, err)
	require.NoError(t, svc.CompleteJob(ctx, claimed, exec, json.RawMessage(`{"ok":true}`)))

	got, err := svc.GetJob(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, models.JobStatusCompleted, got.Status)
}

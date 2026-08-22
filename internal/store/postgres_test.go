package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/db"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postgresStore(t *testing.T) *PostgresStore {
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
	return NewPostgresStore(pool)
}

// mustCreateOrg inserts a unique organization row (required by the
// projects.organization_id foreign key) and cleans it up afterward.
func mustCreateOrg(t *testing.T, s *PostgresStore) string {
	t.Helper()
	ctx := context.Background()
	id := fmt.Sprintf("org-%d", time.Now().UnixNano())
	_, err := s.Pool().Exec(ctx,
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, now(), now())`,
		id, "test-org", id)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = s.Pool().Exec(context.Background(), `DELETE FROM organizations WHERE id=$1`, id)
	})
	return id
}

func TestPostgres_RetryAttempt_ClaimTokenGuard(t *testing.T) {
	s := postgresStore(t)
	ctx := context.Background()

	projID := fmt.Sprintf("proj-%d", time.Now().UnixNano())
	queueID := fmt.Sprintf("q-%d", time.Now().UnixNano())
	now := time.Now()
	orgID := mustCreateOrg(t, s)
	require.NoError(t, s.CreateProject(ctx, &models.Project{ID: projID, OrganizationID: orgID, OwnerID: orgID, Name: "p", Slug: projID, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateQueue(ctx, &models.Queue{ID: queueID, ProjectID: projID, Name: "q", Concurrency: 10, Status: models.QueueStatusActive, CreatedAt: now, UpdatedAt: now}))
	t.Cleanup(func() { _ = s.DeleteProject(ctx, projID) })

	require.NoError(t, s.CreateJob(ctx, &models.Job{
		ID: projID + "-job", ProjectID: projID, QueueID: queueID,
		Type: models.JobTypeImmediate, Payload: []byte(`{}`), Status: models.JobStatusQueued,
		AvailableAt: now, MaxAttempts: 5, RetryStrategy: models.RetryStrategyExponential,
		RetryInitialDelay: time.Second, RetryMaxDelay: time.Minute, RetryMultiplier: 2,
		CreatedAt: now, UpdatedAt: now,
	}))

	claimed, err := s.ClaimJob(ctx, queueID, "worker-A", "token-A", time.Minute)
	require.NoError(t, err)
	_, err = s.StartJob(ctx, claimed.ID, "token-A")
	require.NoError(t, err)

	// A different claim token must be rejected.
	_, err = s.RetryAttempt(ctx, claimed.ID, "token-B", models.JobStatusScheduled, now.Add(time.Second), "stale")
	require.ErrorIs(t, err, ErrConflict)

	got, err := s.GetJob(ctx, claimed.ID)
	require.NoError(t, err)
	require.Equal(t, models.JobStatusRunning, got.Status)

	// The correct token succeeds.
	retried, err := s.RetryAttempt(ctx, claimed.ID, "token-A", models.JobStatusScheduled, now.Add(time.Second), "boom")
	require.NoError(t, err)
	require.Equal(t, models.JobStatusScheduled, retried.Status)
}

func TestPostgres_AtomicClaimConcurrency(t *testing.T) {
	s := postgresStore(t)
	ctx := context.Background()

	projID := fmt.Sprintf("proj-%d", time.Now().UnixNano())
	queueID := fmt.Sprintf("q-%d", time.Now().UnixNano())
	now := time.Now()
	orgID := mustCreateOrg(t, s)
	require.NoError(t, s.CreateProject(ctx, &models.Project{ID: projID, OrganizationID: orgID, OwnerID: orgID, Name: "p", Slug: projID, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateQueue(ctx, &models.Queue{ID: queueID, ProjectID: projID, Name: "q", Concurrency: 1000, Status: models.QueueStatusActive, CreatedAt: now, UpdatedAt: now}))
	t.Cleanup(func() { _ = s.DeleteProject(ctx, projID) })

	const numJobs = 100
	for i := 0; i < numJobs; i++ {
		j := &models.Job{
			ID: fmt.Sprintf("%s-job-%d", projID, i), ProjectID: projID, QueueID: queueID,
			Type: models.JobTypeImmediate, Payload: []byte(`{}`), Status: models.JobStatusQueued,
			AvailableAt: now, MaxAttempts: 3, RetryStrategy: models.RetryStrategyExponential,
			RetryInitialDelay: time.Second, RetryMaxDelay: time.Minute, RetryMultiplier: 2,
			CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, s.CreateJob(ctx, j))
	}

	const numWorkers = 16
	var mu sync.Mutex
	claimed := map[string]int{}
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for {
				job, err := s.ClaimJob(ctx, queueID, fmt.Sprintf("w-%d", worker), fmt.Sprintf("tok-%d-%d", worker, time.Now().UnixNano()), time.Minute)
				if err == ErrNoJobs {
					return
				}
				if err != nil {
					t.Errorf("claim error: %v", err)
					return
				}
				mu.Lock()
				claimed[job.ID]++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	assert.Equal(t, numJobs, len(claimed), "all jobs must be claimed")
	for id, c := range claimed {
		assert.Equal(t, 1, c, "job %s double-claimed", id)
	}
}

func TestPostgres_QueueConcurrencyLimit(t *testing.T) {
	s := postgresStore(t)
	ctx := context.Background()

	projID := fmt.Sprintf("proj-%d", time.Now().UnixNano())
	queueID := fmt.Sprintf("q-%d", time.Now().UnixNano())
	now := time.Now()
	orgID := mustCreateOrg(t, s)
	require.NoError(t, s.CreateProject(ctx, &models.Project{ID: projID, OrganizationID: orgID, OwnerID: orgID, Name: "p", Slug: projID, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateQueue(ctx, &models.Queue{ID: queueID, ProjectID: projID, Name: "q", Concurrency: 2, Status: models.QueueStatusActive, CreatedAt: now, UpdatedAt: now}))
	t.Cleanup(func() { _ = s.DeleteProject(ctx, projID) })

	for i := 0; i < 10; i++ {
		require.NoError(t, s.CreateJob(ctx, &models.Job{
			ID: fmt.Sprintf("%s-job-%d", projID, i), ProjectID: projID, QueueID: queueID,
			Type: models.JobTypeImmediate, Payload: []byte(`{}`), Status: models.JobStatusQueued,
			AvailableAt: now, MaxAttempts: 3, RetryStrategy: models.RetryStrategyExponential,
			RetryInitialDelay: time.Second, RetryMaxDelay: time.Minute, RetryMultiplier: 2,
			CreatedAt: now, UpdatedAt: now,
		}))
	}

	for i := 0; i < 2; i++ {
		_, err := s.ClaimJob(ctx, queueID, "w", fmt.Sprintf("t%d", i), time.Minute)
		require.NoError(t, err)
	}
	_, err := s.ClaimJob(ctx, queueID, "w", "t3", time.Minute)
	assert.ErrorIs(t, err, ErrNoJobs)
}

func TestPostgres_LifecycleAndPagination(t *testing.T) {
	s := postgresStore(t)
	ctx := context.Background()

	projID := fmt.Sprintf("proj-%d", time.Now().UnixNano())
	queueID := fmt.Sprintf("q-%d", time.Now().UnixNano())
	now := time.Now()
	orgID := mustCreateOrg(t, s)
	require.NoError(t, s.CreateProject(ctx, &models.Project{ID: projID, OrganizationID: orgID, OwnerID: orgID, Name: "p", Slug: projID, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.CreateQueue(ctx, &models.Queue{ID: queueID, ProjectID: projID, Name: "q", Concurrency: 100, Status: models.QueueStatusActive, CreatedAt: now, UpdatedAt: now}))
	t.Cleanup(func() { _ = s.DeleteProject(ctx, projID) })

	for i := 0; i < 12; i++ {
		require.NoError(t, s.CreateJob(ctx, &models.Job{
			ID: fmt.Sprintf("%s-job-%02d", projID, i), ProjectID: projID, QueueID: queueID,
			Type: models.JobTypeImmediate, Payload: []byte(`{}`), Status: models.JobStatusQueued,
			AvailableAt: now, MaxAttempts: 3, RetryStrategy: models.RetryStrategyExponential,
			RetryInitialDelay: time.Second, RetryMaxDelay: time.Minute, RetryMultiplier: 2,
			CreatedAt: now.Add(time.Duration(i) * time.Millisecond), UpdatedAt: now,
		}))
	}

	seen := map[string]bool{}
	cursor := ""
	for {
		page, err := s.ListJobs(ctx, ListJobsFilter{ProjectID: projID, Limit: 5, Cursor: cursor})
		require.NoError(t, err)
		for _, j := range page.Items {
			assert.False(t, seen[j.ID], "duplicate job %s", j.ID)
			seen[j.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	assert.Equal(t, 12, len(seen))
}

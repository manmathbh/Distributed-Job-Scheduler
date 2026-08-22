package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMemProject(t *testing.T, s *MemoryStore, id, owner string) *models.Project {
	t.Helper()
	p := &models.Project{ID: id, OrganizationID: owner, OwnerID: owner, Name: "p", Slug: "p-" + id, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.CreateProject(context.Background(), p))
	return p
}

func newMemQueue(t *testing.T, s *MemoryStore, id, projectID string, concurrency int) *models.Queue {
	t.Helper()
	q := &models.Queue{ID: id, ProjectID: projectID, Name: "q-" + id, Priority: 0, Concurrency: concurrency, Status: models.QueueStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.CreateQueue(context.Background(), q))
	return q
}

func newMemJob(t *testing.T, s *MemoryStore, id, projectID, queueID string) *models.Job {
	t.Helper()
	j := &models.Job{
		ID: id, ProjectID: projectID, QueueID: queueID, Type: models.JobTypeImmediate,
		Payload: []byte(`{}`), Priority: 0, Status: models.JobStatusQueued, AvailableAt: time.Now(),
		MaxAttempts: 3, RetryStrategy: models.RetryStrategyExponential, RetryInitialDelay: time.Second, RetryMaxDelay: time.Minute, RetryMultiplier: 2,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateJob(context.Background(), j))
	return j
}

// TestMemoryStore_ConcurrentClaimAtomicity is the critical test: many workers
// claim many jobs concurrently and each job must be claimed at most once.
func TestMemoryStore_ConcurrentClaimAtomicity(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 1000)

	const numJobs = 200
	for i := 0; i < numJobs; i++ {
		newMemJob(t, s, fmt.Sprintf("job-%d", i), "proj", "q")
	}

	const numWorkers = 20
	var mu sync.Mutex
	claimed := make(map[string]int)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for {
				job, err := s.ClaimJob(ctx, "q", fmt.Sprintf("worker-%d", worker), fmt.Sprintf("token-%d-%d", worker, time.Now().UnixNano()), time.Minute)
				if err == ErrNoJobs {
					return
				}
				if err != nil {
					t.Errorf("unexpected claim error: %v", err)
					return
				}
				mu.Lock()
				claimed[job.ID]++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	assert.Equal(t, numJobs, len(claimed), "every job must be claimed exactly once")
	for id, count := range claimed {
		assert.Equal(t, 1, count, "job %s claimed %d times (double-claim!)", id, count)
	}
}

// TestMemoryStore_QueueConcurrencyLimit ensures a queue never exceeds its
// concurrency limit with concurrent claimers.
func TestMemoryStore_QueueConcurrencyLimit(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 3)

	for i := 0; i < 50; i++ {
		newMemJob(t, s, fmt.Sprintf("job-%d", i), "proj", "q")
	}

	// Claim 3 jobs; the 4th must fail with ErrNoJobs.
	for i := 0; i < 3; i++ {
		_, err := s.ClaimJob(ctx, "q", "worker", fmt.Sprintf("t%d", i), time.Minute)
		require.NoError(t, err)
	}
	_, err := s.ClaimJob(ctx, "q", "worker", "t9", time.Minute)
	assert.ErrorIs(t, err, ErrNoJobs, "queue concurrency limit must be enforced")
}

// TestMemoryStore_ClaimPriorityOrder verifies priority ordering.
func TestMemoryStore_ClaimPriorityOrder(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 100)

	j1 := newMemJob(t, s, "low", "proj", "q")
	j1.Priority = 1
	require.NoError(t, s.UpdateJob(ctx, j1))
	j2 := newMemJob(t, s, "high", "proj", "q")
	j2.Priority = 100
	require.NoError(t, s.UpdateJob(ctx, j2))

	job, err := s.ClaimJob(ctx, "q", "worker", "t", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "high", job.ID, "highest priority job should be claimed first")
}

func TestMemoryStore_LeaseRecovery_DLQ(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)

	j := newMemJob(t, s, "job-1", "proj", "q")
	j.MaxAttempts = 1
	require.NoError(t, s.UpdateJob(ctx, j))

	claimed, err := s.ClaimJob(ctx, "q", "worker", "tok", 10*time.Millisecond)
	require.NoError(t, err)
	// lease expires after 10ms
	time.Sleep(20 * time.Millisecond)

	n, err := s.RecoverExpiredLeases(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	got, err := s.GetJob(ctx, claimed.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusFailed, got.Status, "expired lease with exhausted attempts -> failed")

	entries, _, err := s.ListDLQ(ctx, ListDLQFilter{ProjectID: "proj", Limit: 10})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestMemoryStore_LeaseRecovery_Requeue(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)

	j := newMemJob(t, s, "job-1", "proj", "q")
	j.MaxAttempts = 5
	require.NoError(t, s.UpdateJob(ctx, j))

	_, err := s.ClaimJob(ctx, "q", "worker", "tok", 10*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)

	_, err = s.RecoverExpiredLeases(ctx, time.Now())
	require.NoError(t, err)

	got, err := s.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusQueued, got.Status, "expired lease with attempts remaining -> requeued")
}

func TestMemoryStore_Pagination(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)

	for i := 0; i < 25; i++ {
		j := newMemJob(t, s, fmt.Sprintf("job-%02d", i), "proj", "q")
		j.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		require.NoError(t, s.UpdateJob(ctx, j))
	}

	var seen []string
	cursor := ""
	for {
		page, err := s.ListJobs(ctx, ListJobsFilter{ProjectID: "proj", Limit: 10, Cursor: cursor})
		require.NoError(t, err)
		for _, j := range page.Items {
			seen = append(seen, j.ID)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	assert.Len(t, seen, 25, "pagination must return all jobs")
	assert.Equal(t, len(seen), len(unique(seen)), "no duplicates across pages")
}

func TestMemoryStore_QueueStats(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)

	j := newMemJob(t, s, "a", "proj", "q")
	j.Status = models.JobStatusCompleted
	require.NoError(t, s.UpdateJob(ctx, j))
	newMemJob(t, s, "b", "proj", "q")
	newMemJob(t, s, "c", "proj", "q")

	stats, err := s.QueueStats(ctx, "q")
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(2), stats.Queued)
	assert.Equal(t, int64(1), stats.Completed)
}

func TestMemoryStore_TransitionJob_ClaimTokenGuard(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)
	newMemJob(t, s, "job-1", "proj", "q")

	claimed, err := s.ClaimJob(ctx, "q", "worker", "token-A", time.Minute)
	require.NoError(t, err)

	// Wrong claim token must be rejected.
	_, err = s.TransitionJob(ctx, "job-1", models.JobStatusClaimed, models.JobStatusRunning, "token-B")
	assert.ErrorIs(t, err, ErrConflict, "stale worker with wrong token must not transition")

	// Correct token succeeds.
	_, err = s.TransitionJob(ctx, "job-1", models.JobStatusClaimed, models.JobStatusRunning, "token-A")
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusClaimed, claimed.Status)
}

func unique(in []string) []string {
	m := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func TestMemoryStore_RetryAttempt_ClaimTokenGuard(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)

	j := newMemJob(t, s, "job-1", "proj", "q")
	j.MaxAttempts = 5
	require.NoError(t, s.UpdateJob(ctx, j))

	// Worker A claims and starts the job.
	claimed, err := s.ClaimJob(ctx, "q", "worker-A", "token-A", 10*time.Millisecond)
	require.NoError(t, err)
	_, err = s.StartJob(ctx, claimed.ID, "token-A")
	require.NoError(t, err)

	// Lease expires and the job is recovered, then re-claimed by worker B.
	time.Sleep(20 * time.Millisecond)
	_, err = s.RecoverExpiredLeases(ctx, time.Now())
	require.NoError(t, err)
	reclaimed, err := s.ClaimJob(ctx, "q", "worker-B", "token-B", time.Minute)
	require.NoError(t, err)
	require.Equal(t, claimed.ID, reclaimed.ID)
	_, err = s.StartJob(ctx, reclaimed.ID, "token-B")
	require.NoError(t, err)

	// Stale worker A must not be able to retry/fail the job it no longer owns.
	_, err = s.RetryAttempt(ctx, claimed.ID, "token-A", models.JobStatusScheduled, time.Now(), "stale failure")
	assert.ErrorIs(t, err, ErrConflict, "stale worker must not modify a re-claimed job")

	// The job remains running under worker B.
	got, err := s.GetJob(ctx, claimed.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusRunning, got.Status)
	assert.NotNil(t, got.ClaimToken)
	assert.Equal(t, "token-B", *got.ClaimToken)
}

func TestMemoryStore_RetryAttempt_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	newMemProject(t, s, "proj", "owner")
	newMemQueue(t, s, "q", "proj", 10)

	j := newMemJob(t, s, "job-1", "proj", "q")
	j.MaxAttempts = 5
	require.NoError(t, s.UpdateJob(ctx, j))

	claimed, err := s.ClaimJob(ctx, "q", "worker-A", "token-A", time.Minute)
	require.NoError(t, err)
	_, err = s.StartJob(ctx, claimed.ID, "token-A")
	require.NoError(t, err)

	next := time.Now().Add(time.Second)
	retried, err := s.RetryAttempt(ctx, claimed.ID, "token-A", models.JobStatusScheduled, next, "boom")
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, retried.Status)
	assert.Equal(t, "boom", retried.LastError)
	assert.Nil(t, retried.ClaimToken)
	assert.Nil(t, retried.WorkerID)
}

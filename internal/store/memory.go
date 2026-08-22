package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
)

// MemoryStore is an in-memory, mutex-guarded Store used for unit tests and as
// an offline fallback. It implements the same semantics as PostgresStore
// (including atomic single-claim guarantees) but is not durable.
type MemoryStore struct {
	mu sync.Mutex

	organizations map[string]*models.Organization
	projects      map[string]*models.Project
	retryPolicies map[string]*models.RetryPolicy
	queues        map[string]*models.Queue
	jobs          map[string]*models.Job
	executions    map[string]*models.JobExecution
	workers       map[string]*models.Worker
	logs          map[string][]*models.JobLog
	scheduled     map[string]*models.ScheduledJob
	dlq           map[string]*models.DeadLetterJob
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		organizations: make(map[string]*models.Organization),
		projects:      make(map[string]*models.Project),
		retryPolicies: make(map[string]*models.RetryPolicy),
		queues:        make(map[string]*models.Queue),
		jobs:          make(map[string]*models.Job),
		executions:    make(map[string]*models.JobExecution),
		workers:       make(map[string]*models.Worker),
		logs:          make(map[string][]*models.JobLog),
		scheduled:     make(map[string]*models.ScheduledJob),
		dlq:           make(map[string]*models.DeadLetterJob),
	}
}

// ---- Organizations ----

func (s *MemoryStore) EnsureOrganization(_ context.Context, org *models.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.organizations[org.ID]; !ok {
		s.organizations[org.ID] = cloneOrg(org)
	}
	return nil
}

// ---- Projects ----

func (s *MemoryStore) CreateProject(_ context.Context, p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = cloneProject(p)
	return nil
}

func (s *MemoryStore) GetProject(_ context.Context, id string) (*models.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneProject(p), nil
}

func (s *MemoryStore) ListProjects(_ context.Context, ownerID string, limit int, cursor string) ([]*models.Project, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > DefaultMaxPageSize {
		limit = DefaultMaxPageSize
	}
	items := make([]*models.Project, 0)
	for _, p := range s.projects {
		if ownerID != "" && p.OwnerID != ownerID {
			continue
		}
		items = append(items, cloneProject(p))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	out, next := paginateProjects(items, limit, cursor)
	return out, next, nil
}

func (s *MemoryStore) UpdateProject(_ context.Context, p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[p.ID]; !ok {
		return ErrNotFound
	}
	s.projects[p.ID] = cloneProject(p)
	return nil
}

func (s *MemoryStore) DeleteProject(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[id]; !ok {
		return ErrNotFound
	}
	delete(s.projects, id)
	return nil
}

// ---- Retry policies + queues ----

func (s *MemoryStore) CreateRetryPolicy(_ context.Context, rp *models.RetryPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retryPolicies[rp.ID] = cloneRetryPolicy(rp)
	return nil
}

func (s *MemoryStore) GetRetryPolicy(_ context.Context, id string) (*models.RetryPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rp, ok := s.retryPolicies[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneRetryPolicy(rp), nil
}

func (s *MemoryStore) CreateQueue(_ context.Context, q *models.Queue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[q.ID] = cloneQueue(q)
	return nil
}

func (s *MemoryStore) GetQueue(_ context.Context, id string) (*models.Queue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.queues[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneQueue(q), nil
}

func (s *MemoryStore) ListQueues(_ context.Context, projectID string) ([]*models.Queue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*models.Queue, 0)
	for _, q := range s.queues {
		if projectID != "" && q.ProjectID != projectID {
			continue
		}
		out = append(out, cloneQueue(q))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	return out, nil
}

func (s *MemoryStore) UpdateQueue(_ context.Context, q *models.Queue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.queues[q.ID]; !ok {
		return ErrNotFound
	}
	s.queues[q.ID] = cloneQueue(q)
	return nil
}

func (s *MemoryStore) DeleteQueue(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.queues[id]; !ok {
		return ErrNotFound
	}
	delete(s.queues, id)
	return nil
}

func (s *MemoryStore) QueueStats(_ context.Context, queueID string) (*models.QueueStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := &models.QueueStats{QueueID: queueID}
	for _, j := range s.jobs {
		if j.QueueID != queueID {
			continue
		}
		stats.Total++
		switch j.Status {
		case models.JobStatusQueued:
			stats.Queued++
		case models.JobStatusScheduled:
			stats.Scheduled++
		case models.JobStatusClaimed, models.JobStatusRunning:
			stats.Running++
		case models.JobStatusCompleted:
			stats.Completed++
		case models.JobStatusFailed:
			stats.Failed++
		}
	}
	for _, d := range s.dlq {
		if d.QueueID == queueID {
			stats.DeadLettered++
		}
	}
	return stats, nil
}

// ---- Jobs ----

func (s *MemoryStore) CreateJob(_ context.Context, j *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[j.ID] = cloneJob(j)
	return nil
}

func (s *MemoryStore) BatchCreateJobs(_ context.Context, jobs []*models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range jobs {
		s.jobs[j.ID] = cloneJob(j)
	}
	return nil
}

func (s *MemoryStore) GetJob(_ context.Context, id string) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneJob(j), nil
}

func (s *MemoryStore) ListJobs(_ context.Context, f ListJobsFilter) (*Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Limit <= 0 || f.Limit > DefaultMaxPageSize {
		f.Limit = DefaultMaxPageSize
	}
	items := make([]*models.Job, 0)
	for _, j := range s.jobs {
		if j.ProjectID != f.ProjectID {
			continue
		}
		if f.QueueID != "" && j.QueueID != f.QueueID {
			continue
		}
		if f.Status != "" && string(j.Status) != f.Status {
			continue
		}
		if f.Type != "" && string(j.Type) != f.Type {
			continue
		}
		items = append(items, cloneJob(j))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	out, next := paginateJobs(items, f.Limit, f.Cursor)
	return &Page{Items: out, NextCursor: next}, nil
}

func (s *MemoryStore) UpdateJob(_ context.Context, j *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[j.ID]; !ok {
		return ErrNotFound
	}
	s.jobs[j.ID] = cloneJob(j)
	return nil
}

func (s *MemoryStore) TransitionJob(_ context.Context, id string, from, to models.JobStatus, claimToken string) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	if claimToken != "" && (j.ClaimToken == nil || *j.ClaimToken != claimToken) {
		return nil, ErrConflict
	}
	if err := models.Transition(j.Status, to); err != nil {
		return nil, err
	}
	if j.Status != from && from != "" {
		return nil, ErrConflict
	}
	j.Status = to
	j.UpdatedAt = time.Now()
	return cloneJob(j), nil
}

func (s *MemoryStore) RetryAttempt(_ context.Context, id, claimToken string, to models.JobStatus, availableAt time.Time, errMsg string) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	if claimToken != "" && (j.ClaimToken == nil || *j.ClaimToken != claimToken) {
		return nil, ErrConflict
	}
	if j.Status != models.JobStatusRunning {
		return nil, ErrConflict
	}
	if err := models.Transition(models.JobStatusRunning, to); err != nil {
		return nil, err
	}
	now := time.Now()
	j.Status = to
	j.AvailableAt = availableAt
	j.LastError = errMsg
	j.ClaimToken = nil
	j.LeaseExpiresAt = nil
	j.WorkerID = nil
	j.UpdatedAt = now
	return cloneJob(j), nil
}

func (s *MemoryStore) ClaimJob(_ context.Context, queueID, workerID, claimToken string, lease time.Duration) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.queues[queueID]
	if !ok {
		return nil, ErrNotFound
	}
	if q.Status == models.QueueStatusPaused {
		return nil, ErrNoJobs
	}
	running := 0
	for _, j := range s.jobs {
		if j.QueueID == queueID && (j.Status == models.JobStatusClaimed || j.Status == models.JobStatusRunning) {
			running++
		}
	}
	if running >= q.Concurrency {
		return nil, ErrNoJobs
	}
	now := time.Now()
	candidates := make([]*models.Job, 0)
	for _, j := range s.jobs {
		if j.QueueID == queueID && j.Status == models.JobStatusQueued && !j.AvailableAt.After(now) {
			candidates = append(candidates, j)
		}
	}
	if len(candidates) == 0 {
		return nil, ErrNoJobs
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
		}
		return candidates[i].Priority > candidates[j].Priority
	})
	job := candidates[0]
	job.Status = models.JobStatusClaimed
	job.Attempts++
	job.WorkerID = &workerID
	job.ClaimToken = &claimToken
	leaseUntil := now.Add(lease)
	job.LeaseExpiresAt = &leaseUntil
	job.ClaimedAt = timePtr(now)
	job.UpdatedAt = now
	return cloneJob(job), nil
}

func (s *MemoryStore) StartJob(_ context.Context, id, claimToken string) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	if claimToken != "" && (j.ClaimToken == nil || *j.ClaimToken != claimToken) {
		return nil, ErrConflict
	}
	if err := models.Transition(j.Status, models.JobStatusRunning); err != nil {
		return nil, err
	}
	now := time.Now()
	j.Status = models.JobStatusRunning
	j.StartedAt = &now
	j.UpdatedAt = now
	return cloneJob(j), nil
}

func (s *MemoryStore) RecoverExpiredLeases(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	recovered := 0
	for _, j := range s.jobs {
		if j.Status != models.JobStatusClaimed && j.Status != models.JobStatusRunning {
			continue
		}
		if j.LeaseExpiresAt == nil || j.LeaseExpiresAt.After(now) {
			continue
		}
		if j.Attempts < j.MaxAttempts {
			j.Status = models.JobStatusQueued
			j.AvailableAt = now
		} else {
			j.Status = models.JobStatusFailed
			j.FailedAt = timePtr(now)
			s.dlq[j.ID] = &models.DeadLetterJob{
				ID:        "dlq-" + j.ID,
				JobID:     j.ID,
				ProjectID: j.ProjectID,
				QueueID:   j.QueueID,
				Payload:   j.Payload,
				Reason:    "lease expired after max attempts",
				Attempts:  j.Attempts,
				WorkerID:  j.WorkerID,
				FailedAt:  now,
				CreatedAt: now,
			}
		}
		j.ClaimToken = nil
		j.LeaseExpiresAt = nil
		j.WorkerID = nil
		j.UpdatedAt = now
		recovered++
	}
	return recovered, nil
}

func (s *MemoryStore) PromoteDueJobs(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	promoted := 0
	for _, j := range s.jobs {
		if j.Status == models.JobStatusScheduled && !j.AvailableAt.After(now) {
			j.Status = models.JobStatusQueued
			j.UpdatedAt = now
			promoted++
		}
	}
	return promoted, nil
}

func (s *MemoryStore) CountJobs(_ context.Context, projectID string) (map[models.JobStatus]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[models.JobStatus]int64)
	for _, j := range s.jobs {
		if j.ProjectID != projectID {
			continue
		}
		out[j.Status]++
	}
	return out, nil
}

// ---- Executions + logs ----

func (s *MemoryStore) CreateExecution(_ context.Context, e *models.JobExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[e.ID] = cloneExecution(e)
	return nil
}

func (s *MemoryStore) UpdateExecution(_ context.Context, e *models.JobExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.executions[e.ID]; !ok {
		return ErrNotFound
	}
	s.executions[e.ID] = cloneExecution(e)
	return nil
}

func (s *MemoryStore) ListExecutions(_ context.Context, jobID string) ([]*models.JobExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*models.JobExecution, 0)
	for _, e := range s.executions {
		if e.JobID == jobID {
			out = append(out, cloneExecution(e))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Attempt < out[j].Attempt })
	return out, nil
}

func (s *MemoryStore) AppendLog(_ context.Context, l *models.JobLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs[l.JobID] = append(s.logs[l.JobID], cloneLog(l))
	return nil
}

func (s *MemoryStore) ListLogs(_ context.Context, jobID string, limit int) ([]*models.JobLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	all := s.logs[jobID]
	if len(all) <= limit {
		out := make([]*models.JobLog, len(all))
		for i, l := range all {
			out[i] = cloneLog(l)
		}
		return out, nil
	}
	out := make([]*models.JobLog, limit)
	copy(out, all[len(all)-limit:])
	return out, nil
}

// ---- Workers ----

func (s *MemoryStore) RegisterWorker(_ context.Context, w *models.Worker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workers[w.ID] = cloneWorker(w)
	return nil
}

func (s *MemoryStore) Heartbeat(_ context.Context, workerID string, running int, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.workers[workerID]
	if !ok {
		return ErrNotFound
	}
	w.LastHeartbeat = now
	w.LastSeenAt = now
	w.Status = models.WorkerStatusActive
	return nil
}

func (s *MemoryStore) GetWorker(_ context.Context, id string) (*models.Worker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.workers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneWorker(w), nil
}

func (s *MemoryStore) ListWorkers(_ context.Context, projectID string) ([]*models.Worker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*models.Worker, 0)
	for _, w := range s.workers {
		if projectID == "" || w.ProjectID == projectID {
			out = append(out, cloneWorker(w))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastHeartbeat.After(out[j].LastHeartbeat) })
	return out, nil
}

func (s *MemoryStore) MarkStaleWorkers(_ context.Context, staleAfter time.Duration, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	marked := 0
	for _, w := range s.workers {
		if w.Status == models.WorkerStatusDead {
			continue
		}
		if now.Sub(w.LastHeartbeat) > staleAfter {
			w.Status = models.WorkerStatusDead
			marked++
		}
	}
	return marked, nil
}

// ---- Scheduled jobs ----

func (s *MemoryStore) CreateScheduledJob(_ context.Context, sj *models.ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scheduled[sj.ID] = cloneScheduled(sj)
	return nil
}

func (s *MemoryStore) GetScheduledJob(_ context.Context, id string) (*models.ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sj, ok := s.scheduled[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneScheduled(sj), nil
}

func (s *MemoryStore) ListScheduledJobs(_ context.Context, projectID string) ([]*models.ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*models.ScheduledJob, 0)
	for _, sj := range s.scheduled {
		if projectID == "" || sj.ProjectID == projectID {
			out = append(out, cloneScheduled(sj))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateScheduledJob(_ context.Context, sj *models.ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scheduled[sj.ID]; !ok {
		return ErrNotFound
	}
	s.scheduled[sj.ID] = cloneScheduled(sj)
	return nil
}

func (s *MemoryStore) FireDueScheduledJob(_ context.Context, now time.Time, nextRunFn func(*models.ScheduledJob) (time.Time, error)) (*models.ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *models.ScheduledJob
	for _, sj := range s.scheduled {
		if !sj.Enabled || sj.NextRunAt == nil || sj.NextRunAt.After(now) {
			continue
		}
		if best == nil || sj.NextRunAt.Before(*best.NextRunAt) {
			best = sj
		}
	}
	if best == nil {
		return nil, nil
	}
	next, err := nextRunFn(cloneScheduled(best))
	if err != nil {
		return nil, err
	}
	best.LastRunAt = timePtr(now)
	best.NextRunAt = timePtr(next)
	best.UpdatedAt = now
	return cloneScheduled(best), nil
}

// ---- DLQ ----

func (s *MemoryStore) CreateDLQEntry(_ context.Context, d *models.DeadLetterJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dlq[d.ID] = cloneDLQ(d)
	return nil
}

func (s *MemoryStore) GetDLQEntry(_ context.Context, id string) (*models.DeadLetterJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dlq[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneDLQ(d), nil
}

func (s *MemoryStore) ListDLQ(_ context.Context, f ListDLQFilter) ([]*models.DeadLetterJob, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Limit <= 0 || f.Limit > DefaultMaxPageSize {
		f.Limit = DefaultMaxPageSize
	}
	items := make([]*models.DeadLetterJob, 0)
	for _, d := range s.dlq {
		if d.ProjectID != f.ProjectID {
			continue
		}
		if f.QueueID != "" && d.QueueID != f.QueueID {
			continue
		}
		items = append(items, cloneDLQ(d))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FailedAt.Equal(items[j].FailedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].FailedAt.After(items[j].FailedAt)
	})
	return paginateDLQ(items, f.Limit, f.Cursor)
}

func (s *MemoryStore) MarkRequeued(_ context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dlq[id]
	if !ok {
		return ErrNotFound
	}
	d.RequeuedAt = timePtr(now)
	return nil
}

// ---- Clone helpers ----

func timePtr(t time.Time) *time.Time { return &t }

func cloneProject(p *models.Project) *models.Project             { c := *p; return &c }
func cloneOrg(o *models.Organization) *models.Organization       { c := *o; return &c }
func cloneRetryPolicy(r *models.RetryPolicy) *models.RetryPolicy { c := *r; return &c }
func cloneQueue(q *models.Queue) *models.Queue                   { c := *q; return &c }
func cloneWorker(w *models.Worker) *models.Worker                { c := *w; return &c }
func cloneExecution(e *models.JobExecution) *models.JobExecution { c := *e; return &c }
func cloneLog(l *models.JobLog) *models.JobLog                   { c := *l; return &c }
func cloneScheduled(s *models.ScheduledJob) *models.ScheduledJob { c := *s; return &c }
func cloneDLQ(d *models.DeadLetterJob) *models.DeadLetterJob     { c := *d; return &c }

func cloneJob(j *models.Job) *models.Job {
	c := *j
	c.Payload = append([]byte(nil), j.Payload...)
	return &c
}

func paginateJobs(items []*models.Job, limit int, cursor string) ([]*models.Job, string) {
	idx := 0
	if cursor != "" {
		ct, cid, err := DecodeCursor(cursor)
		if err == nil {
			idx = len(items)
			for i, it := range items {
				if it.CreatedAt.Before(ct) || (it.CreatedAt.Equal(ct) && it.ID < cid) {
					idx = i
					break
				}
			}
		}
	}
	if idx >= len(items) {
		return []*models.Job{}, ""
	}
	end := idx + limit
	if end > len(items) {
		end = len(items)
	}
	out := items[idx:end]
	next := ""
	if end < len(items) {
		last := items[end-1]
		next = EncodeCursor(last.CreatedAt, last.ID)
	}
	return out, next
}

func paginateProjects(items []*models.Project, limit int, cursor string) ([]*models.Project, string) {
	idx := 0
	if cursor != "" {
		ct, cid, err := DecodeCursor(cursor)
		if err == nil {
			idx = len(items)
			for i, it := range items {
				if it.CreatedAt.Before(ct) || (it.CreatedAt.Equal(ct) && it.ID < cid) {
					idx = i
					break
				}
			}
		}
	}
	if idx >= len(items) {
		return []*models.Project{}, ""
	}
	end := idx + limit
	if end > len(items) {
		end = len(items)
	}
	out := items[idx:end]
	next := ""
	if end < len(items) {
		last := items[end-1]
		next = EncodeCursor(last.CreatedAt, last.ID)
	}
	return out, next
}

func paginateDLQ(items []*models.DeadLetterJob, limit int, cursor string) ([]*models.DeadLetterJob, string, error) {
	idx := 0
	if cursor != "" {
		ct, cid, err := DecodeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		idx = len(items)
		for i, it := range items {
			if it.FailedAt.Before(ct) || (it.FailedAt.Equal(ct) && it.ID < cid) {
				idx = i
				break
			}
		}
	}
	if idx >= len(items) {
		return []*models.DeadLetterJob{}, "", nil
	}
	end := idx + limit
	if end > len(items) {
		end = len(items)
	}
	out := items[idx:end]
	next := ""
	if end < len(items) {
		last := items[end-1]
		next = EncodeCursor(last.FailedAt, last.ID)
	}
	return out, next, nil
}

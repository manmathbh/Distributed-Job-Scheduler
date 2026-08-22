package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/id"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/retry"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
)

// Service orchestrates business rules on top of the Store.
type Service struct {
	store   store.Store
	metrics *metrics.Registry
}

// New constructs a Service.
func New(st store.Store, m *metrics.Registry) *Service {
	return &Service{store: st, metrics: m}
}

// Store exposes the underlying store for the scheduler/worker.
func (s *Service) Store() store.Store { return s.store }

// ---- Projects ----

func (s *Service) CreateProject(ctx context.Context, ownerID, name, description string) (*models.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	now := time.Now()

	// Ensure the owning organization exists to satisfy the
	// projects.organization_id foreign key. The owner (API key principal) is
	// modeled as a single-owner organization.
	if err := s.store.EnsureOrganization(ctx, &models.Organization{
		ID:        ownerID,
		Name:      ownerID,
		Slug:      ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return nil, err
	}

	p := &models.Project{
		ID:             id.New(),
		OrganizationID: ownerID, // org scoping derived from key owner
		OwnerID:        ownerID,
		Name:           name,
		Slug:           slugify(name) + "-" + id.New()[:8],
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.store.CreateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) ListProjects(ctx context.Context, ownerID string, limit int, cursor string) ([]*models.Project, string, error) {
	return s.store.ListProjects(ctx, ownerID, limit, cursor)
}

func (s *Service) GetProject(ctx context.Context, id string) (*models.Project, error) {
	return s.store.GetProject(ctx, id)
}

func (s *Service) UpdateProject(ctx context.Context, id, name, description string) (*models.Project, error) {
	p, err := s.store.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		p.Name = strings.TrimSpace(name)
	}
	if description != "" {
		p.Description = description
	}
	p.UpdatedAt = time.Now()
	if err := s.store.UpdateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) DeleteProject(ctx context.Context, id string) error {
	return s.store.DeleteProject(ctx, id)
}

// ---- Queues ----

// QueueConfig carries queue creation inputs.
type QueueConfig struct {
	Name          string
	Description   string
	Priority      int
	Concurrency   int
	RetryStrategy models.RetryStrategy
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	Multiplier    float64
}

func (s *Service) CreateQueue(ctx context.Context, projectID string, cfg QueueConfig) (*models.Queue, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	if cfg.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.MaxAttempts < 0 {
		return nil, fmt.Errorf("max_attempts must be >= 0")
	}
	if cfg.RetryStrategy == "" {
		cfg.RetryStrategy = retry.Default().Strategy
	}
	if !cfg.RetryStrategy.IsValid() {
		return nil, fmt.Errorf("invalid retry strategy %q", cfg.RetryStrategy)
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = retry.Default().InitialDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = retry.Default().MaxDelay
	}
	if cfg.Multiplier < 1 {
		cfg.Multiplier = retry.Default().Multiplier
	}

	now := time.Now()
	rp := &models.RetryPolicy{
		ID:           id.New(),
		Strategy:     cfg.RetryStrategy,
		MaxAttempts:  cfg.MaxAttempts,
		InitialDelay: cfg.InitialDelay,
		MaxDelay:     cfg.MaxDelay,
		Multiplier:   cfg.Multiplier,
		CreatedAt:    now,
	}
	if err := s.store.CreateRetryPolicy(ctx, rp); err != nil {
		return nil, err
	}

	q := &models.Queue{
		ID:            id.New(),
		ProjectID:     projectID,
		Name:          cfg.Name,
		Description:   cfg.Description,
		Priority:      cfg.Priority,
		Concurrency:   cfg.Concurrency,
		Status:        models.QueueStatusActive,
		RetryPolicyID: &rp.ID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.store.CreateQueue(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) GetQueue(ctx context.Context, id string) (*models.Queue, error) {
	return s.store.GetQueue(ctx, id)
}

func (s *Service) ListQueues(ctx context.Context, projectID string) ([]*models.Queue, error) {
	return s.store.ListQueues(ctx, projectID)
}

func (s *Service) UpdateQueue(ctx context.Context, id string, cfg QueueConfig) (*models.Queue, error) {
	q, err := s.store.GetQueue(ctx, id)
	if err != nil {
		return nil, err
	}
	if cfg.Name != "" {
		q.Name = strings.TrimSpace(cfg.Name)
	}
	if cfg.Description != "" {
		q.Description = cfg.Description
	}
	q.Priority = cfg.Priority
	if cfg.Concurrency > 0 {
		q.Concurrency = cfg.Concurrency
	}
	q.UpdatedAt = time.Now()
	if err := s.store.UpdateQueue(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) SetQueueStatus(ctx context.Context, id string, status models.QueueStatus) (*models.Queue, error) {
	q, err := s.store.GetQueue(ctx, id)
	if err != nil {
		return nil, err
	}
	q.Status = status
	q.UpdatedAt = time.Now()
	if err := s.store.UpdateQueue(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) DeleteQueue(ctx context.Context, id string) error {
	return s.store.DeleteQueue(ctx, id)
}

func (s *Service) QueueStats(ctx context.Context, id string) (*models.QueueStats, error) {
	return s.store.QueueStats(ctx, id)
}

// ---- Jobs ----

// SubmitJobRequest captures all job-submission inputs.
type SubmitJobRequest struct {
	ProjectID     string
	QueueID       string
	Type          models.JobType
	Payload       json.RawMessage
	Priority      int
	ScheduledAt   *time.Time
	Delay         time.Duration
	MaxAttempts   int
	RetryStrategy models.RetryStrategy
	CronExpr      string
	Timezone      string
	ScheduleName  string
}

// resolveRetryPolicy returns the effective retry policy for a new job.
func (s *Service) resolveRetryPolicy(ctx context.Context, q *models.Queue, req SubmitJobRequest) retry.Policy {
	policy := retry.Default()
	if req.MaxAttempts > 0 {
		policy.MaxAttempts = req.MaxAttempts
	}
	if req.RetryStrategy != "" {
		policy.Strategy = req.RetryStrategy
	}
	if q != nil && q.RetryPolicyID != nil {
		if rp, err := s.store.GetRetryPolicy(ctx, *q.RetryPolicyID); err == nil {
			if req.MaxAttempts <= 0 {
				policy.MaxAttempts = rp.MaxAttempts
			}
			if req.RetryStrategy == "" {
				policy.Strategy = rp.Strategy
			}
			policy.InitialDelay = rp.InitialDelay
			policy.MaxDelay = rp.MaxDelay
			policy.Multiplier = rp.Multiplier
		}
	}
	return policy
}

func (s *Service) SubmitJob(ctx context.Context, req SubmitJobRequest) (*models.Job, error) {
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if req.QueueID == "" {
		return nil, fmt.Errorf("queue_id is required")
	}
	if req.Type == "" {
		req.Type = models.JobTypeImmediate
	}
	if !req.Type.IsValid() {
		return nil, fmt.Errorf("invalid job type %q", req.Type)
	}
	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage("{}")
	}
	if !json.Valid(req.Payload) {
		return nil, fmt.Errorf("payload must be valid JSON")
	}

	q, err := s.store.GetQueue(ctx, req.QueueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("queue does not belong to project")
	}
	if q.Status == models.QueueStatusPaused && req.Type != models.JobTypeScheduled && req.Type != models.JobTypeRecurring {
		return nil, fmt.Errorf("queue is paused")
	}

	policy := s.resolveRetryPolicy(ctx, q, req)

	now := time.Now()
	job := &models.Job{
		ID:                id.New(),
		ProjectID:         req.ProjectID,
		QueueID:           req.QueueID,
		Type:              req.Type,
		Payload:           req.Payload,
		Priority:          req.Priority,
		MaxAttempts:       policy.MaxAttempts,
		RetryStrategy:     policy.Strategy,
		RetryInitialDelay: policy.InitialDelay,
		RetryMaxDelay:     policy.MaxDelay,
		RetryMultiplier:   policy.Multiplier,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	switch req.Type {
	case models.JobTypeImmediate:
		job.Status = models.JobStatusQueued
		job.AvailableAt = now
	case models.JobTypeDelayed:
		job.AvailableAt = now.Add(req.Delay)
		if job.AvailableAt.After(now) {
			job.Status = models.JobStatusScheduled
		} else {
			job.Status = models.JobStatusQueued
		}
	case models.JobTypeScheduled:
		if req.ScheduledAt == nil {
			return nil, fmt.Errorf("scheduled_at is required for scheduled jobs")
		}
		job.ScheduledAt = req.ScheduledAt
		job.AvailableAt = *req.ScheduledAt
		if job.AvailableAt.After(now) {
			job.Status = models.JobStatusScheduled
		} else {
			job.Status = models.JobStatusQueued
		}
	case models.JobTypeRecurring:
		// Recurring jobs are represented as schedules; handled by SubmitSchedule.
		return nil, fmt.Errorf("recurring jobs must be submitted via the schedule endpoint")
	}

	if err := s.store.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	s.metrics.IncSubmitted()
	return job, nil
}

// BatchJob is a single item in a batch submission.
type BatchJob struct {
	Type        models.JobType
	Payload     json.RawMessage
	Priority    int
	Delay       time.Duration
	ScheduledAt *time.Time
}

func (s *Service) SubmitBatch(ctx context.Context, projectID, queueID string, items []BatchJob) ([]*models.Job, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("batch must contain at least one job")
	}
	if len(items) > 1000 {
		return nil, fmt.Errorf("batch size exceeds maximum of 1000")
	}
	q, err := s.store.GetQueue(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != projectID {
		return nil, fmt.Errorf("queue does not belong to project")
	}

	jobs := make([]*models.Job, 0, len(items))
	now := time.Now()
	for _, it := range items {
		req := SubmitJobRequest{
			ProjectID: projectID, QueueID: queueID, Type: it.Type,
			Payload: it.Payload, Priority: it.Priority, Delay: it.Delay, ScheduledAt: it.ScheduledAt,
		}
		if req.Type == "" {
			req.Type = models.JobTypeImmediate
		}
		if len(req.Payload) == 0 {
			req.Payload = json.RawMessage("{}")
		}
		if !json.Valid(req.Payload) {
			return nil, fmt.Errorf("payload must be valid JSON")
		}
		policy := s.resolveRetryPolicy(ctx, q, req)
		j := &models.Job{
			ID:                id.New(),
			ProjectID:         projectID,
			QueueID:           queueID,
			Type:              req.Type,
			Payload:           req.Payload,
			Priority:          req.Priority,
			MaxAttempts:       policy.MaxAttempts,
			RetryStrategy:     policy.Strategy,
			RetryInitialDelay: policy.InitialDelay,
			RetryMaxDelay:     policy.MaxDelay,
			RetryMultiplier:   policy.Multiplier,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		switch j.Type {
		case models.JobTypeDelayed:
			j.AvailableAt = now.Add(it.Delay)
		case models.JobTypeScheduled:
			if it.ScheduledAt == nil {
				return nil, fmt.Errorf("scheduled_at is required for scheduled jobs")
			}
			j.ScheduledAt = it.ScheduledAt
			j.AvailableAt = *it.ScheduledAt
		default:
			j.AvailableAt = now
		}
		if j.AvailableAt.After(now) {
			j.Status = models.JobStatusScheduled
		} else {
			j.Status = models.JobStatusQueued
		}
		jobs = append(jobs, j)
	}

	if err := s.store.BatchCreateJobs(ctx, jobs); err != nil {
		return nil, err
	}
	s.metrics.Counters.JobsSubmitted.Add(int64(len(jobs)))
	return jobs, nil
}

func (s *Service) GetJob(ctx context.Context, id string) (*models.Job, error) {
	return s.store.GetJob(ctx, id)
}

func (s *Service) ListJobs(ctx context.Context, f store.ListJobsFilter) (*store.Page, error) {
	return s.store.ListJobs(ctx, f)
}

func (s *Service) CancelJob(ctx context.Context, id string) (*models.Job, error) {
	j, err := s.store.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if j.Status.IsTerminal() {
		return nil, fmt.Errorf("job already in terminal state %s", j.Status)
	}
	return s.store.TransitionJob(ctx, id, j.Status, models.JobStatusCancelled, "")
}

func (s *Service) RetryJob(ctx context.Context, id string) (*models.Job, error) {
	j, err := s.store.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if j.Status != models.JobStatusFailed {
		return nil, fmt.Errorf("only failed jobs can be retried")
	}
	now := time.Now()
	j.Status = models.JobStatusQueued
	j.AvailableAt = now
	j.Attempts = 0
	j.LastError = ""
	j.ClaimToken = nil
	j.LeaseExpiresAt = nil
	j.WorkerID = nil
	j.FailedAt = nil
	j.UpdatedAt = now
	if err := s.store.UpdateJob(ctx, j); err != nil {
		return nil, err
	}
	s.metrics.IncRetried()
	return j, nil
}

// ---- Worker-side lifecycle (claim / run / complete / fail) ----

func (s *Service) ClaimJob(ctx context.Context, queueID, workerID string) (*models.Job, error) {
	j, err := s.store.ClaimJob(ctx, queueID, workerID, id.New(), 30*time.Second)
	if err != nil {
		return nil, err
	}
	s.metrics.IncClaims()
	return j, nil
}

// RecordExecutionStart transitions a claimed job to running and opens an
// execution record. Returns the execution to be finalized later.
func (s *Service) RecordExecutionStart(ctx context.Context, job *models.Job) (*models.JobExecution, error) {
	claimToken := ""
	if job.ClaimToken != nil {
		claimToken = *job.ClaimToken
	}
	started, err := s.store.StartJob(ctx, job.ID, claimToken)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	exec := &models.JobExecution{
		ID:        id.New(),
		JobID:     job.ID,
		Attempt:   started.Attempts,
		WorkerID:  started.WorkerID,
		Status:    models.JobStatusRunning,
		StartedAt: now,
	}
	if err := s.store.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}
	return exec, nil
}

func (s *Service) CompleteJob(ctx context.Context, job *models.Job, exec *models.JobExecution, result json.RawMessage) error {
	claimToken := ""
	if job.ClaimToken != nil {
		claimToken = *job.ClaimToken
	}
	now := time.Now()
	if _, err := s.store.TransitionJob(ctx, job.ID, models.JobStatusRunning, models.JobStatusCompleted, claimToken); err != nil {
		return err
	}
	completed, err := s.store.GetJob(ctx, job.ID)
	if err != nil {
		return err
	}
	completed.CompletedAt = &now
	completed.UpdatedAt = now
	if err := s.store.UpdateJob(ctx, completed); err != nil {
		return err
	}
	exec.Status = models.JobStatusCompleted
	exec.CompletedAt = &now
	exec.DurationMS = now.Sub(exec.StartedAt).Milliseconds()
	exec.Metadata = result
	if err := s.store.UpdateExecution(ctx, exec); err != nil {
		return err
	}
	s.metrics.IncCompleted()
	s.metrics.AddExecution(now.Sub(exec.StartedAt))
	return nil
}

// FailJob applies retry policy: either schedules a retry (queued/scheduled) or
// dead-letters the job.
func (s *Service) FailJob(ctx context.Context, job *models.Job, exec *models.JobExecution, errMsg string) error {
	claimToken := ""
	if job.ClaimToken != nil {
		claimToken = *job.ClaimToken
	}
	now := time.Now()
	policy := retry.Policy{
		Strategy:     job.RetryStrategy,
		MaxAttempts:  job.MaxAttempts,
		InitialDelay: job.RetryInitialDelay,
		MaxDelay:     job.RetryMaxDelay,
		Multiplier:   job.RetryMultiplier,
	}

	failed, err := s.store.GetJob(ctx, job.ID)
	if err != nil {
		return err
	}
	if failed.Status != models.JobStatusRunning {
		return fmt.Errorf("job not running (status=%s)", failed.Status)
	}

	exec.Status = models.JobStatusFailed
	exec.CompletedAt = &now
	exec.DurationMS = now.Sub(exec.StartedAt).Milliseconds()
	exec.Error = errMsg
	exec.Retryable = policy.ShouldRetry(failed.Attempts)
	if err := s.store.UpdateExecution(ctx, exec); err != nil {
		return err
	}
	s.metrics.IncFailed()
	s.metrics.AddExecution(now.Sub(exec.StartedAt))

	if policy.ShouldRetry(failed.Attempts) {
		delay := policy.NextDelay(failed.Attempts)
		next := now.Add(delay)
		if _, err := s.store.RetryAttempt(ctx, job.ID, claimToken, models.JobStatusScheduled, next, errMsg); err != nil {
			return err
		}
		s.metrics.IncRetried()
		return nil
	}

	// Permanent failure -> dead letter queue.
	if _, err := s.store.TransitionJob(ctx, job.ID, models.JobStatusRunning, models.JobStatusFailed, claimToken); err != nil {
		return err
	}
	dead, err := s.store.GetJob(ctx, job.ID)
	if err != nil {
		return err
	}
	dead.FailedAt = &now
	dead.LastError = errMsg
	dead.ClaimToken = nil
	dead.LeaseExpiresAt = nil
	dead.UpdatedAt = now
	if err := s.store.UpdateJob(ctx, dead); err != nil {
		return err
	}
	dlq := &models.DeadLetterJob{
		ID:        id.New(),
		JobID:     dead.ID,
		ProjectID: dead.ProjectID,
		QueueID:   dead.QueueID,
		Payload:   dead.Payload,
		Reason:    errMsg,
		Attempts:  dead.Attempts,
		WorkerID:  dead.WorkerID,
		FailedAt:  now,
		CreatedAt: now,
	}
	if err := s.store.CreateDLQEntry(ctx, dlq); err != nil {
		return err
	}
	s.metrics.IncDeadLettered()
	return nil
}

// ---- Scheduled (recurring) jobs ----

func (s *Service) CreateSchedule(ctx context.Context, req SubmitJobRequest) (*models.ScheduledJob, error) {
	if req.CronExpr == "" {
		return nil, fmt.Errorf("cron_expr is required for recurring jobs")
	}
	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage("{}")
	}
	if !json.Valid(req.Payload) {
		return nil, fmt.Errorf("payload must be valid JSON")
	}
	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}
	nextRun, err := NextCronRun(req.CronExpr, tz, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	now := time.Now()
	sj := &models.ScheduledJob{
		ID:        id.New(),
		ProjectID: req.ProjectID,
		QueueID:   req.QueueID,
		Name:      req.ScheduleName,
		CronExpr:  req.CronExpr,
		Timezone:  tz,
		Payload:   req.Payload,
		Priority:  req.Priority,
		Enabled:   true,
		NextRunAt: &nextRun,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateScheduledJob(ctx, sj); err != nil {
		return nil, err
	}
	s.metrics.IncSubmitted()
	return sj, nil
}

func (s *Service) ListSchedules(ctx context.Context, projectID string) ([]*models.ScheduledJob, error) {
	return s.store.ListScheduledJobs(ctx, projectID)
}

func (s *Service) GetSchedule(ctx context.Context, id string) (*models.ScheduledJob, error) {
	return s.store.GetScheduledJob(ctx, id)
}

// MaterializeScheduledJob creates a queued job from a due recurring schedule,
// snapshotting the queue's configured retry policy onto the job (mirroring
// normal job submission).
func (s *Service) MaterializeScheduledJob(ctx context.Context, sj *models.ScheduledJob) (*models.Job, error) {
	q, err := s.store.GetQueue(ctx, sj.QueueID)
	if err != nil {
		return nil, err
	}
	if q.ProjectID != sj.ProjectID {
		return nil, fmt.Errorf("schedule queue does not belong to project")
	}
	policy := s.resolveRetryPolicy(ctx, q, SubmitJobRequest{ProjectID: sj.ProjectID, QueueID: sj.QueueID})

	now := time.Now()
	job := &models.Job{
		ID:                id.New(),
		ProjectID:         sj.ProjectID,
		QueueID:           sj.QueueID,
		Type:              models.JobTypeRecurring,
		Payload:           sj.Payload,
		Priority:          sj.Priority,
		Status:            models.JobStatusQueued,
		AvailableAt:       now,
		MaxAttempts:       policy.MaxAttempts,
		RetryStrategy:     policy.Strategy,
		RetryInitialDelay: policy.InitialDelay,
		RetryMaxDelay:     policy.MaxDelay,
		RetryMultiplier:   policy.Multiplier,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	s.metrics.IncSubmitted()
	return job, nil
}

// ---- DLQ ----

func (s *Service) ListDLQ(ctx context.Context, f store.ListDLQFilter) ([]*models.DeadLetterJob, string, error) {
	return s.store.ListDLQ(ctx, f)
}

// RequeueDLQ re-enqueues a dead-lettered job for a fresh attempt.
func (s *Service) RequeueDLQ(ctx context.Context, dlqID string) (*models.Job, error) {
	entry, err := s.store.GetDLQEntry(ctx, dlqID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	job := &models.Job{
		ID:                id.New(),
		ProjectID:         entry.ProjectID,
		QueueID:           entry.QueueID,
		Type:              models.JobTypeImmediate,
		Payload:           entry.Payload,
		Priority:          0,
		Status:            models.JobStatusQueued,
		AvailableAt:       now,
		MaxAttempts:       3,
		RetryStrategy:     models.RetryStrategyExponential,
		RetryInitialDelay: time.Second,
		RetryMaxDelay:     time.Minute,
		RetryMultiplier:   2.0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if err := s.store.MarkRequeued(ctx, dlqID, now); err != nil {
		return nil, err
	}
	return job, nil
}

// ---- Workers ----

func (s *Service) RegisterWorker(ctx context.Context, w *models.Worker) error {
	return s.store.RegisterWorker(ctx, w)
}

func (s *Service) Heartbeat(ctx context.Context, workerID string, running int) error {
	return s.store.Heartbeat(ctx, workerID, running, time.Now())
}

func (s *Service) ListWorkers(ctx context.Context, projectID string) ([]*models.Worker, error) {
	return s.store.ListWorkers(ctx, projectID)
}

// ---- Overview / dashboard ----

// Overview is the dashboard summary payload.
type Overview struct {
	Projects      int64                      `json:"projects"`
	Queues        int64                      `json:"queues"`
	Workers       int64                      `json:"workers"`
	ActiveWorkers int64                      `json:"active_workers"`
	Jobs          map[models.JobStatus]int64 `json:"jobs"`
	DeadLettered  int64                      `json:"dead_lettered"`
	Metrics       metrics.Snapshot           `json:"metrics"`
}

func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	projects, _, err := s.store.ListProjects(ctx, "", DefaultMaxPageSize, "")
	if err != nil {
		return nil, err
	}
	workers, err := s.store.ListWorkers(ctx, "")
	if err != nil {
		return nil, err
	}
	ov := &Overview{
		Projects: int64(len(projects)),
		Workers:  int64(len(workers)),
		Metrics:  s.metrics.Snapshot(),
	}
	for _, w := range workers {
		if w.Status != models.WorkerStatusDead {
			ov.ActiveWorkers++
		}
	}
	// Count jobs across all projects by listing queues is expensive; approximate
	// via per-project counts using the first page of projects we already have.
	ov.Jobs = map[models.JobStatus]int64{}
	for _, p := range projects {
		counts, err := s.store.CountJobs(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		for st, c := range counts {
			ov.Jobs[st] += c
		}
		dlq, _, err := s.store.ListDLQ(ctx, store.ListDLQFilter{ProjectID: p.ID, Limit: DefaultMaxPageSize})
		if err != nil {
			return nil, err
		}
		ov.DeadLettered += int64(len(dlq))
	}
	return ov, nil
}

// DefaultMaxPageSize mirrors the store constant.
const DefaultMaxPageSize = store.DefaultMaxPageSize

// slugify converts a name to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "project"
	}
	return out
}

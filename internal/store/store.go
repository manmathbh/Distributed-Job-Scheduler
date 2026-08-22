package store

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = fmt.Errorf("not found")

// ErrConflict is returned when an operation violates a concurrency constraint.
var ErrConflict = fmt.Errorf("conflict")

// ErrNoJobs is returned when no jobs are available to claim.
var ErrNoJobs = fmt.Errorf("no jobs available")

// DefaultMaxPageSize caps collection page sizes.
const DefaultMaxPageSize = 100

// Page is a generic cursor-paginated result.
type Page struct {
	Items      []*models.Job
	NextCursor string
}

// ListJobsFilter narrows job listings.
type ListJobsFilter struct {
	ProjectID string
	QueueID   string
	Status    string
	Type      string
	Limit     int
	Cursor    string
}

// ListDLQFilter narrows dead-letter listings.
type ListDLQFilter struct {
	ProjectID string
	QueueID   string
	Limit     int
	Cursor    string
}

// OrganizationStore persists organizations.
type OrganizationStore interface {
	// EnsureOrganization idempotently creates an organization if it does not
	// already exist. Used before creating projects to satisfy the
	// projects.organization_id foreign key.
	EnsureOrganization(ctx context.Context, org *models.Organization) error
}

// ProjectStore persists projects.
type ProjectStore interface {
	CreateProject(ctx context.Context, p *models.Project) error
	GetProject(ctx context.Context, id string) (*models.Project, error)
	ListProjects(ctx context.Context, ownerID string, limit int, cursor string) ([]*models.Project, string, error)
	UpdateProject(ctx context.Context, p *models.Project) error
	DeleteProject(ctx context.Context, id string) error
}

// QueueStore persists queues and retry policies.
type QueueStore interface {
	CreateRetryPolicy(ctx context.Context, rp *models.RetryPolicy) error
	GetRetryPolicy(ctx context.Context, id string) (*models.RetryPolicy, error)
	CreateQueue(ctx context.Context, q *models.Queue) error
	GetQueue(ctx context.Context, id string) (*models.Queue, error)
	ListQueues(ctx context.Context, projectID string) ([]*models.Queue, error)
	UpdateQueue(ctx context.Context, q *models.Queue) error
	DeleteQueue(ctx context.Context, id string) error
	QueueStats(ctx context.Context, queueID string) (*models.QueueStats, error)
}

// JobStore persists jobs and executes the atomic claim.
type JobStore interface {
	CreateJob(ctx context.Context, j *models.Job) error
	BatchCreateJobs(ctx context.Context, jobs []*models.Job) error
	GetJob(ctx context.Context, id string) (*models.Job, error)
	ListJobs(ctx context.Context, f ListJobsFilter) (*Page, error)
	UpdateJob(ctx context.Context, j *models.Job) error
	// TransitionJob applies a validated state change. If claimToken is non-empty
	// it must match the job's current claim token (idempotency guard).
	TransitionJob(ctx context.Context, id string, from, to models.JobStatus, claimToken string) (*models.Job, error)
	// ClaimJob atomically claims a queued job for a worker, respecting the
	// queue's concurrency limit. Returns ErrNoJobs when nothing is claimable.
	ClaimJob(ctx context.Context, queueID, workerID, claimToken string, lease time.Duration) (*models.Job, error)
	// StartJob confirms execution started for a claimed job (claim -> running).
	StartJob(ctx context.Context, id, claimToken string) (*models.Job, error)
	// RetryAttempt atomically moves a running job back to a retryable state
	// (e.g. scheduled) with a new available_at and error message, guarded by the
	// claim token so a stale worker cannot modify a job it no longer owns.
	RetryAttempt(ctx context.Context, id, claimToken string, to models.JobStatus, availableAt time.Time, errMsg string) (*models.Job, error)
	// RecoverExpiredLeases returns claimed/running jobs whose lease expired back
	// to a retryable state (or dead-letters them).
	RecoverExpiredLeases(ctx context.Context, now time.Time) (int, error)
	// PromoteDueJobs moves scheduled jobs whose available_at <= now into queued.
	PromoteDueJobs(ctx context.Context, now time.Time) (int, error)
	CountJobs(ctx context.Context, projectID string) (map[models.JobStatus]int64, error)
}

// ExecutionStore persists execution history and logs.
type ExecutionStore interface {
	CreateExecution(ctx context.Context, e *models.JobExecution) error
	UpdateExecution(ctx context.Context, e *models.JobExecution) error
	ListExecutions(ctx context.Context, jobID string) ([]*models.JobExecution, error)
	AppendLog(ctx context.Context, l *models.JobLog) error
	ListLogs(ctx context.Context, jobID string, limit int) ([]*models.JobLog, error)
}

// WorkerStore persists worker registration and heartbeats.
type WorkerStore interface {
	RegisterWorker(ctx context.Context, w *models.Worker) error
	Heartbeat(ctx context.Context, workerID string, running int, now time.Time) error
	GetWorker(ctx context.Context, id string) (*models.Worker, error)
	ListWorkers(ctx context.Context, projectID string) ([]*models.Worker, error)
	MarkStaleWorkers(ctx context.Context, staleAfter time.Duration, now time.Time) (int, error)
}

// SchedulerStore persists recurring schedules and supports single-coordinator
// claim of due schedules.
type SchedulerStore interface {
	CreateScheduledJob(ctx context.Context, s *models.ScheduledJob) error
	GetScheduledJob(ctx context.Context, id string) (*models.ScheduledJob, error)
	ListScheduledJobs(ctx context.Context, projectID string) ([]*models.ScheduledJob, error)
	UpdateScheduledJob(ctx context.Context, s *models.ScheduledJob) error
	// FireDueScheduledJob atomically locks the earliest due schedule, advances
	// next_run_at via nextRunFn (invoked while locked), and returns it for
	// firing. Returns (nil, nil) when nothing is due.
	FireDueScheduledJob(ctx context.Context, now time.Time, nextRunFn func(*models.ScheduledJob) (time.Time, error)) (*models.ScheduledJob, error)
}

// DLQStore persists dead-lettered jobs.
type DLQStore interface {
	CreateDLQEntry(ctx context.Context, d *models.DeadLetterJob) error
	GetDLQEntry(ctx context.Context, id string) (*models.DeadLetterJob, error)
	ListDLQ(ctx context.Context, f ListDLQFilter) ([]*models.DeadLetterJob, string, error)
	MarkRequeued(ctx context.Context, id string, now time.Time) error
}

// Store is the composite persistence interface for the scheduler.
type Store interface {
	OrganizationStore
	ProjectStore
	QueueStore
	JobStore
	ExecutionStore
	WorkerStore
	SchedulerStore
	DLQStore
}

// EncodeCursor encodes a keyset cursor from (created_at, id).
func EncodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%d|%s", createdAt.UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a keyset cursor into (created_at, id).
func DecodeCursor(cursor string) (time.Time, string, error) {
	if cursor == "" {
		return time.Time{}, "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}
	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return time.Unix(0, ns), parts[1], nil
}

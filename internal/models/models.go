package models

import (
	"time"
)

// JobType classifies how a job is scheduled.
type JobType string

const (
	JobTypeImmediate JobType = "immediate"
	JobTypeDelayed   JobType = "delayed"
	JobTypeScheduled JobType = "scheduled"
	JobTypeRecurring JobType = "recurring"
)

func (t JobType) IsValid() bool {
	switch t {
	case JobTypeImmediate, JobTypeDelayed, JobTypeScheduled, JobTypeRecurring:
		return true
	default:
		return false
	}
}

// JobStatus is the lifecycle state of a job.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"    // available_at <= now, ready to claim
	JobStatusScheduled JobStatus = "scheduled" // waiting until available_at (future)
	JobStatusClaimed   JobStatus = "claimed"   // claimed by a worker (lease held)
	JobStatusRunning   JobStatus = "running"   // actively executing
	JobStatusCompleted JobStatus = "completed" // success
	JobStatusFailed    JobStatus = "failed"    // permanent failure (dead-lettered)
	JobStatusCancelled JobStatus = "cancelled" // cancelled by user
)

func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed || s == JobStatusCancelled
}

// QueueStatus represents whether a queue is accepting work.
type QueueStatus string

const (
	QueueStatusActive QueueStatus = "active"
	QueueStatusPaused QueueStatus = "paused"
)

func (s QueueStatus) IsValid() bool {
	return s == QueueStatusActive || s == QueueStatusPaused
}

// RetryStrategy defines the backoff algorithm used between attempts.
type RetryStrategy string

const (
	RetryStrategyFixed       RetryStrategy = "fixed"
	RetryStrategyLinear      RetryStrategy = "linear"
	RetryStrategyExponential RetryStrategy = "exponential"
)

func (s RetryStrategy) IsValid() bool {
	switch s {
	case RetryStrategyFixed, RetryStrategyLinear, RetryStrategyExponential:
		return true
	default:
		return false
	}
}

// WorkerStatus represents the liveness of a worker.
type WorkerStatus string

const (
	WorkerStatusActive WorkerStatus = "active"
	WorkerStatusIdle   WorkerStatus = "idle"
	WorkerStatusDead   WorkerStatus = "dead"
)

// User is an account in the system.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Organization groups users and owns projects.
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrganizationMember is a user's membership in an organization.
type OrganizationMember struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

// Project is a scoped unit of work that owns queues.
type Project struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	OwnerID        string    `json:"owner_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RetryPolicy configures retry behavior for a queue (or a job).
type RetryPolicy struct {
	ID           string        `json:"id"`
	Strategy     RetryStrategy `json:"strategy"`
	MaxAttempts  int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
	CreatedAt    time.Time     `json:"created_at"`
}

// Queue is a named job queue belonging to a project.
type Queue struct {
	ID            string      `json:"id"`
	ProjectID     string      `json:"project_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Priority      int         `json:"priority"`
	Concurrency   int         `json:"concurrency"`
	Status        QueueStatus `json:"status"`
	RetryPolicyID *string     `json:"retry_policy_id,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// QueueStats is a computed snapshot of queue health.
type QueueStats struct {
	QueueID       string `json:"queue_id"`
	Total         int64  `json:"total"`
	Queued        int64  `json:"queued"`
	Scheduled     int64  `json:"scheduled"`
	Running       int64  `json:"running"`
	Completed     int64  `json:"completed"`
	Failed        int64  `json:"failed"`
	DeadLettered  int64  `json:"dead_lettered"`
	ActiveWorkers int64  `json:"active_workers"`
}

// Job is the core unit of work.
type Job struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	QueueID   string    `json:"queue_id"`
	Type      JobType   `json:"type"`
	Payload   []byte    `json:"payload"`
	Priority  int       `json:"priority"`
	Status    JobStatus `json:"status"`

	// Scheduling
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	AvailableAt time.Time  `json:"available_at"`

	// Retry snapshot (copied from queue policy at submit time)
	MaxAttempts       int           `json:"max_attempts"`
	RetryStrategy     RetryStrategy `json:"retry_strategy"`
	RetryInitialDelay time.Duration `json:"retry_initial_delay"`
	RetryMaxDelay     time.Duration `json:"retry_max_delay"`
	RetryMultiplier   float64       `json:"retry_multiplier"`

	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error,omitempty"`

	// Lease / claim bookkeeping
	ClaimToken     *string    `json:"claim_token,omitempty"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	WorkerID       *string    `json:"worker_id,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
}

// JobExecution records a single attempt of a job.
type JobExecution struct {
	ID          string     `json:"id"`
	JobID       string     `json:"job_id"`
	Attempt     int        `json:"attempt"`
	WorkerID    *string    `json:"worker_id,omitempty"`
	Status      JobStatus  `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMS  int64      `json:"duration_ms"`
	Error       string     `json:"error,omitempty"`
	Retryable   bool       `json:"retryable"`
	Metadata    []byte     `json:"metadata,omitempty"`
}

// Worker is a registered job processor.
type Worker struct {
	ID            string       `json:"id"`
	ProjectID     string       `json:"project_id"`
	Hostname      string       `json:"hostname"`
	Status        WorkerStatus `json:"status"`
	Concurrency   int          `json:"concurrency"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	StartedAt     time.Time    `json:"started_at"`
	LastSeenAt    time.Time    `json:"last_seen_at"`
	Metadata      []byte       `json:"metadata,omitempty"`
}

// WorkerHeartbeat records a heartbeat event.
type WorkerHeartbeat struct {
	ID       string    `json:"id"`
	WorkerID string    `json:"worker_id"`
	SentAt   time.Time `json:"sent_at"`
	Running  int       `json:"running"`
}

// JobLog is a single log line emitted during job execution.
type JobLog struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	Attempt   int       `json:"attempt"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// ScheduledJob is a recurring (cron) schedule that materializes jobs.
type ScheduledJob struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	QueueID   string     `json:"queue_id"`
	Name      string     `json:"name"`
	CronExpr  string     `json:"cron_expr"`
	Timezone  string     `json:"timezone"`
	Payload   []byte     `json:"payload"`
	Priority  int        `json:"priority"`
	Enabled   bool       `json:"enabled"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// DeadLetterJob is a permanently failed job retained for inspection.
type DeadLetterJob struct {
	ID         string     `json:"id"`
	JobID      string     `json:"job_id"`
	ProjectID  string     `json:"project_id"`
	QueueID    string     `json:"queue_id"`
	Payload    []byte     `json:"payload"`
	Reason     string     `json:"reason"`
	Attempts   int        `json:"attempts"`
	WorkerID   *string    `json:"worker_id,omitempty"`
	FailedAt   time.Time  `json:"failed_at"`
	CreatedAt  time.Time  `json:"created_at"`
	RequeuedAt *time.Time `json:"requeued_at,omitempty"`
}

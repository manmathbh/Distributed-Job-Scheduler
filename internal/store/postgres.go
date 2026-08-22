package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/manmathbh/distributed-job-scheduler/internal/id"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
)

func newID() string { return id.New() }

// PostgresStore implements Store on top of PostgreSQL. It is the
// authoritative persistence layer for scheduler state.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore wraps an existing pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Pool exposes the underlying pool for health checks.
func (s *PostgresStore) Pool() *pgxpool.Pool { return s.pool }

// ---- Organizations ----

func (s *PostgresStore) EnsureOrganization(ctx context.Context, org *models.Organization) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (id) DO NOTHING`,
		org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	return err
}

// ---- Projects ----

func (s *PostgresStore) CreateProject(ctx context.Context, p *models.Project) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO projects (id, organization_id, owner_id, name, slug, description, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.OrganizationID, p.OwnerID, p.Name, p.Slug, p.Description, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *PostgresStore) GetProject(ctx context.Context, id string) (*models.Project, error) {
	p := &models.Project{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, organization_id, owner_id, name, slug, description, created_at, updated_at
		FROM projects WHERE id=$1`, id).
		Scan(&p.ID, &p.OrganizationID, &p.OwnerID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PostgresStore) ListProjects(ctx context.Context, ownerID string, limit int, cursor string) ([]*models.Project, string, error) {
	if limit <= 0 || limit > DefaultMaxPageSize {
		limit = DefaultMaxPageSize
	}
	ct, cid, err := DecodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	args := []interface{}{limit + 1}
	where := ""
	if ownerID != "" {
		args = append(args, ownerID)
		where = " WHERE owner_id=$2"
	}
	if cursor != "" {
		if where == "" {
			where = " WHERE (created_at, id) < ($2::timestamptz, $3)"
		} else {
			where += " AND (created_at, id) < ($3::timestamptz, $4)"
		}
		args = append(args, ct, cid)
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, organization_id, owner_id, name, slug, description, created_at, updated_at
		FROM projects%s ORDER BY created_at DESC, id DESC LIMIT $1`, where), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*models.Project, 0, limit)
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.OwnerID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, "", err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = EncodeCursor(last.CreatedAt, last.ID)
	}
	return items, next, nil
}

func (s *PostgresStore) UpdateProject(ctx context.Context, p *models.Project) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE projects SET name=$2, slug=$3, description=$4, updated_at=$5 WHERE id=$1`,
		p.ID, p.Name, p.Slug, p.Description, p.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteProject(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Retry policies + queues ----

func (s *PostgresStore) CreateRetryPolicy(ctx context.Context, rp *models.RetryPolicy) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO retry_policies (id, strategy, max_attempts, initial_delay_ms, max_delay_ms, multiplier, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		rp.ID, string(rp.Strategy), rp.MaxAttempts, rp.InitialDelay.Milliseconds(), rp.MaxDelay.Milliseconds(), rp.Multiplier, rp.CreatedAt)
	return err
}

func (s *PostgresStore) GetRetryPolicy(ctx context.Context, id string) (*models.RetryPolicy, error) {
	rp := &models.RetryPolicy{}
	var initialMS, maxMS int64
	err := s.pool.QueryRow(ctx, `
		SELECT id, strategy, max_attempts, initial_delay_ms, max_delay_ms, multiplier, created_at
		FROM retry_policies WHERE id=$1`, id).
		Scan(&rp.ID, &rp.Strategy, &rp.MaxAttempts, &initialMS, &maxMS, &rp.Multiplier, &rp.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rp.InitialDelay = time.Duration(initialMS) * time.Millisecond
	rp.MaxDelay = time.Duration(maxMS) * time.Millisecond
	return rp, nil
}

func (s *PostgresStore) CreateQueue(ctx context.Context, q *models.Queue) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO queues (id, project_id, name, description, priority, concurrency, status, retry_policy_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		q.ID, q.ProjectID, q.Name, q.Description, q.Priority, q.Concurrency, string(q.Status), q.RetryPolicyID, q.CreatedAt, q.UpdatedAt)
	return err
}

func (s *PostgresStore) GetQueue(ctx context.Context, id string) (*models.Queue, error) {
	q := &models.Queue{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, priority, concurrency, status, retry_policy_id, created_at, updated_at
		FROM queues WHERE id=$1`, id).
		Scan(&q.ID, &q.ProjectID, &q.Name, &q.Description, &q.Priority, &q.Concurrency, &q.Status, &q.RetryPolicyID, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (s *PostgresStore) ListQueues(ctx context.Context, projectID string) ([]*models.Queue, error) {
	query := `SELECT id, project_id, name, description, priority, concurrency, status, retry_policy_id, created_at, updated_at FROM queues`
	args := []interface{}{}
	if projectID != "" {
		query += ` WHERE project_id=$1`
		args = append(args, projectID)
	}
	query += ` ORDER BY priority DESC, name ASC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.Queue, 0)
	for rows.Next() {
		q := &models.Queue{}
		if err := rows.Scan(&q.ID, &q.ProjectID, &q.Name, &q.Description, &q.Priority, &q.Concurrency, &q.Status, &q.RetryPolicyID, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, q)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateQueue(ctx context.Context, q *models.Queue) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE queues SET name=$2, description=$3, priority=$4, concurrency=$5, status=$6, retry_policy_id=$7, updated_at=$8
		WHERE id=$1`,
		q.ID, q.Name, q.Description, q.Priority, q.Concurrency, string(q.Status), q.RetryPolicyID, q.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteQueue(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM queues WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) QueueStats(ctx context.Context, queueID string) (*models.QueueStats, error) {
	stats := &models.QueueStats{QueueID: queueID}
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status='queued')   AS queued,
			COUNT(*) FILTER (WHERE status='scheduled') AS scheduled,
			COUNT(*) FILTER (WHERE status IN ('claimed','running')) AS running,
			COUNT(*) FILTER (WHERE status='completed') AS completed,
			COUNT(*) FILTER (WHERE status='failed')    AS failed
		FROM jobs WHERE queue_id=$1`, queueID).
		Scan(&stats.Queued, &stats.Scheduled, &stats.Running, &stats.Completed, &stats.Failed)
	if err != nil {
		return nil, err
	}
	stats.Total = stats.Queued + stats.Scheduled + stats.Running + stats.Completed + stats.Failed
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM dead_letter_jobs WHERE queue_id=$1`, queueID).Scan(&stats.DeadLettered)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// ---- Jobs ----

func (s *PostgresStore) CreateJob(ctx context.Context, j *models.Job) error {
	return s.insertJob(ctx, j)
}

func (s *PostgresStore) BatchCreateJobs(ctx context.Context, jobs []*models.Job) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, j := range jobs {
		if err := s.insertJobTx(ctx, tx, j); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) insertJob(ctx context.Context, j *models.Job) error {
	_, err := s.pool.Exec(ctx, jobInsertSQL,
		j.ID, j.ProjectID, j.QueueID, string(j.Type), jobPayload(j.Payload), j.Priority, string(j.Status),
		j.ScheduledAt, j.AvailableAt,
		j.MaxAttempts, string(j.RetryStrategy), j.RetryInitialDelay.Milliseconds(), j.RetryMaxDelay.Milliseconds(), j.RetryMultiplier,
		j.Attempts, j.LastError, j.ClaimToken, j.LeaseExpiresAt, j.WorkerID,
		j.CreatedAt, j.UpdatedAt, j.ClaimedAt, j.StartedAt, j.CompletedAt, j.FailedAt)
	return err
}

func (s *PostgresStore) insertJobTx(ctx context.Context, tx pgx.Tx, j *models.Job) error {
	_, err := tx.Exec(ctx, jobInsertSQL,
		j.ID, j.ProjectID, j.QueueID, string(j.Type), jobPayload(j.Payload), j.Priority, string(j.Status),
		j.ScheduledAt, j.AvailableAt,
		j.MaxAttempts, string(j.RetryStrategy), j.RetryInitialDelay.Milliseconds(), j.RetryMaxDelay.Milliseconds(), j.RetryMultiplier,
		j.Attempts, j.LastError, j.ClaimToken, j.LeaseExpiresAt, j.WorkerID,
		j.CreatedAt, j.UpdatedAt, j.ClaimedAt, j.StartedAt, j.CompletedAt, j.FailedAt)
	return err
}

const jobInsertSQL = `
	INSERT INTO jobs (
		id, project_id, queue_id, type, payload, priority, status,
		scheduled_at, available_at,
		max_attempts, retry_strategy, retry_initial_delay_ms, retry_max_delay_ms, retry_multiplier,
		attempts, last_error, claim_token, lease_expires_at, worker_id,
		created_at, updated_at, claimed_at, started_at, completed_at, failed_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`

const jobColumns = `id, project_id, queue_id, type, payload, priority, status,
	scheduled_at, available_at, max_attempts, retry_strategy, retry_initial_delay_ms, retry_max_delay_ms,
	retry_multiplier, attempts, last_error, claim_token, lease_expires_at, worker_id,
	created_at, updated_at, claimed_at, started_at, completed_at, failed_at`

func scanJob(row pgx.Row) (*models.Job, error) {
	j := &models.Job{}
	var retryInitialMS, retryMaxMS int64
	err := row.Scan(
		&j.ID, &j.ProjectID, &j.QueueID, &j.Type, &j.Payload, &j.Priority, &j.Status,
		&j.ScheduledAt, &j.AvailableAt, &j.MaxAttempts, &j.RetryStrategy, &retryInitialMS, &retryMaxMS,
		&j.RetryMultiplier, &j.Attempts, &j.LastError, &j.ClaimToken, &j.LeaseExpiresAt, &j.WorkerID,
		&j.CreatedAt, &j.UpdatedAt, &j.ClaimedAt, &j.StartedAt, &j.CompletedAt, &j.FailedAt)
	if err != nil {
		return nil, err
	}
	j.RetryInitialDelay = time.Duration(retryInitialMS) * time.Millisecond
	j.RetryMaxDelay = time.Duration(retryMaxMS) * time.Millisecond
	return j, nil
}

func (s *PostgresStore) GetJob(ctx context.Context, id string) (*models.Job, error) {
	j, err := scanJob(s.pool.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (s *PostgresStore) ListJobs(ctx context.Context, f ListJobsFilter) (*Page, error) {
	if f.Limit <= 0 || f.Limit > DefaultMaxPageSize {
		f.Limit = DefaultMaxPageSize
	}
	ct, cid, err := DecodeCursor(f.Cursor)
	if err != nil {
		return nil, err
	}
	where := " WHERE project_id=$1"
	args := []interface{}{f.ProjectID}
	idx := 2
	if f.QueueID != "" {
		args = append(args, f.QueueID)
		where += fmt.Sprintf(" AND queue_id=$%d", idx)
		idx++
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND status=$%d", idx)
		idx++
	}
	if f.Type != "" {
		args = append(args, f.Type)
		where += fmt.Sprintf(" AND type=$%d", idx)
		idx++
	}
	if f.Cursor != "" {
		args = append(args, ct, cid)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d::timestamptz, $%d)", idx, idx+1)
	}
	args = append(args, f.Limit+1)
	query := fmt.Sprintf(`SELECT %s FROM jobs%s ORDER BY created_at DESC, id DESC LIMIT $%d`, jobColumns, where, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.Job, 0, f.Limit)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	next := ""
	if len(items) > f.Limit {
		items = items[:f.Limit]
		last := items[len(items)-1]
		next = EncodeCursor(last.CreatedAt, last.ID)
	}
	return &Page{Items: items, NextCursor: next}, nil
}

func (s *PostgresStore) UpdateJob(ctx context.Context, j *models.Job) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE jobs SET
			status=$2, available_at=$3, attempts=$4, last_error=$5,
			claim_token=$6, lease_expires_at=$7, worker_id=$8,
			updated_at=$9, claimed_at=$10, started_at=$11, completed_at=$12, failed_at=$13
		WHERE id=$1`,
		j.ID, string(j.Status), j.AvailableAt, j.Attempts, j.LastError,
		j.ClaimToken, j.LeaseExpiresAt, j.WorkerID,
		j.UpdatedAt, j.ClaimedAt, j.StartedAt, j.CompletedAt, j.FailedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) TransitionJob(ctx context.Context, id string, from, to models.JobStatus, claimToken string) (*models.Job, error) {
	if err := models.Transition(from, to); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	j, err := scanJob(tx.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if j.Status != from {
		return nil, ErrConflict
	}
	if claimToken != "" && (j.ClaimToken == nil || *j.ClaimToken != claimToken) {
		return nil, ErrConflict
	}
	now := time.Now()
	_, err = tx.Exec(ctx, `UPDATE jobs SET status=$2, updated_at=$3 WHERE id=$1`, id, string(to), now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	j.Status = to
	j.UpdatedAt = now
	return j, nil
}

// ClaimJob atomically claims the highest-priority available job in a queue
// using SELECT ... FOR UPDATE SKIP LOCKED so concurrent workers can never
// claim the same row.
func (s *PostgresStore) ClaimJob(ctx context.Context, queueID, workerID, claimToken string, lease time.Duration) (*models.Job, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize on the queue row to enforce its concurrency limit.
	var concurrency int
	var qstatus models.QueueStatus
	err = tx.QueryRow(ctx, `SELECT concurrency, status FROM queues WHERE id=$1 FOR UPDATE`, queueID).
		Scan(&concurrency, &qstatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if qstatus == models.QueueStatusPaused {
		return nil, ErrNoJobs
	}

	var running int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE queue_id=$1 AND status IN ('claimed','running')`, queueID).Scan(&running); err != nil {
		return nil, err
	}
	if running >= concurrency {
		return nil, ErrNoJobs
	}

	now := time.Now()
	j, err := scanJob(tx.QueryRow(ctx, `
		SELECT `+jobColumns+` FROM jobs
		WHERE queue_id=$1 AND status='queued' AND available_at <= $2
		ORDER BY priority DESC, created_at ASC, id ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, queueID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoJobs
	}
	if err != nil {
		return nil, err
	}

	leaseUntil := now.Add(lease)
	_, err = tx.Exec(ctx, `
		UPDATE jobs SET status='claimed', claim_token=$2, worker_id=$3, attempts=attempts+1,
			lease_expires_at=$4, claimed_at=$5, updated_at=$5
		WHERE id=$1`,
		j.ID, claimToken, workerID, leaseUntil, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	j.Status = models.JobStatusClaimed
	j.ClaimToken = &claimToken
	j.WorkerID = &workerID
	j.LeaseExpiresAt = &leaseUntil
	j.ClaimedAt = &now
	j.Attempts++
	j.UpdatedAt = now
	return j, nil
}

func (s *PostgresStore) StartJob(ctx context.Context, id, claimToken string) (*models.Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	j, err := scanJob(tx.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if j.Status != models.JobStatusClaimed {
		return nil, ErrConflict
	}
	if claimToken != "" && (j.ClaimToken == nil || *j.ClaimToken != claimToken) {
		return nil, ErrConflict
	}
	now := time.Now()
	_, err = tx.Exec(ctx, `UPDATE jobs SET status='running', started_at=$2, updated_at=$2 WHERE id=$1`, id, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	j.Status = models.JobStatusRunning
	j.StartedAt = &now
	j.UpdatedAt = now
	return j, nil
}

func (s *PostgresStore) RetryAttempt(ctx context.Context, id, claimToken string, to models.JobStatus, availableAt time.Time, errMsg string) (*models.Job, error) {
	if err := models.Transition(models.JobStatusRunning, to); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	j, err := scanJob(tx.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if j.Status != models.JobStatusRunning {
		return nil, ErrConflict
	}
	if claimToken != "" && (j.ClaimToken == nil || *j.ClaimToken != claimToken) {
		return nil, ErrConflict
	}
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE jobs SET status=$2, available_at=$3, last_error=$4,
			claim_token=NULL, lease_expires_at=NULL, worker_id=NULL, updated_at=$5
		WHERE id=$1`, id, string(to), availableAt, errMsg, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	j.Status = to
	j.AvailableAt = availableAt
	j.LastError = errMsg
	j.ClaimToken = nil
	j.LeaseExpiresAt = nil
	j.WorkerID = nil
	j.UpdatedAt = now
	return j, nil
}

// RecoverExpiredLeases requeues (or dead-letters) jobs whose leases expired.
func (s *PostgresStore) RecoverExpiredLeases(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT `+jobColumns+` FROM jobs
		WHERE status IN ('claimed','running') AND lease_expires_at <= $1
		FOR UPDATE SKIP LOCKED`, now)
	if err != nil {
		return 0, err
	}
	var expired []*models.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			rows.Close()
			return 0, err
		}
		expired = append(expired, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, j := range expired {
		if j.Attempts < j.MaxAttempts {
			_, err = tx.Exec(ctx, `
				UPDATE jobs SET status='queued', available_at=$2, claim_token=NULL, lease_expires_at=NULL, worker_id=NULL, updated_at=$2
				WHERE id=$1`, j.ID, now)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE jobs SET status='failed', failed_at=$2, updated_at=$2, claim_token=NULL, lease_expires_at=NULL
				WHERE id=$1`, j.ID, now)
			if err == nil {
				err = s.insertDLQTx(ctx, tx, &models.DeadLetterJob{
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
				})
			}
		}
		if err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(expired), nil
}

func (s *PostgresStore) PromoteDueJobs(ctx context.Context, now time.Time) (int, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE jobs SET status='queued', updated_at=$1
		WHERE status='scheduled' AND available_at <= $1`, now)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *PostgresStore) CountJobs(ctx context.Context, projectID string) (map[models.JobStatus]int64, error) {
	rows, err := s.pool.Query(ctx, `SELECT status, COUNT(*) FROM jobs WHERE project_id=$1 GROUP BY status`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[models.JobStatus]int64)
	for rows.Next() {
		var st string
		var c int64
		if err := rows.Scan(&st, &c); err != nil {
			return nil, err
		}
		out[models.JobStatus(st)] = c
	}
	return out, rows.Err()
}

// ---- Executions + logs ----

func (s *PostgresStore) CreateExecution(ctx context.Context, e *models.JobExecution) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_executions (id, job_id, attempt, worker_id, status, started_at, completed_at, duration_ms, error, retryable, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		e.ID, e.JobID, e.Attempt, e.WorkerID, string(e.Status), e.StartedAt, e.CompletedAt, e.DurationMS, e.Error, e.Retryable, e.Metadata)
	return err
}

func (s *PostgresStore) UpdateExecution(ctx context.Context, e *models.JobExecution) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE job_executions SET status=$2, completed_at=$3, duration_ms=$4, error=$5, retryable=$6, metadata=$7
		WHERE id=$1`,
		e.ID, string(e.Status), e.CompletedAt, e.DurationMS, e.Error, e.Retryable, e.Metadata)
	return err
}

func (s *PostgresStore) ListExecutions(ctx context.Context, jobID string) ([]*models.JobExecution, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, job_id, attempt, worker_id, status, started_at, completed_at, duration_ms, error, retryable, metadata
		FROM job_executions WHERE job_id=$1 ORDER BY attempt ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.JobExecution, 0)
	for rows.Next() {
		e := &models.JobExecution{}
		if err := rows.Scan(&e.ID, &e.JobID, &e.Attempt, &e.WorkerID, &e.Status, &e.StartedAt, &e.CompletedAt, &e.DurationMS, &e.Error, &e.Retryable, &e.Metadata); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (s *PostgresStore) AppendLog(ctx context.Context, l *models.JobLog) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_logs (id, job_id, attempt, level, message, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`, l.ID, l.JobID, l.Attempt, l.Level, l.Message, l.CreatedAt)
	return err
}

func (s *PostgresStore) ListLogs(ctx context.Context, jobID string, limit int) ([]*models.JobLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, job_id, attempt, level, message, created_at
		FROM job_logs WHERE job_id=$1 ORDER BY created_at DESC LIMIT $2`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.JobLog, 0)
	for rows.Next() {
		l := &models.JobLog{}
		if err := rows.Scan(&l.ID, &l.JobID, &l.Attempt, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	// Reverse to ascending order for readability.
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, rows.Err()
}

// ---- Workers ----

func (s *PostgresStore) RegisterWorker(ctx context.Context, w *models.Worker) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workers (id, project_id, hostname, status, concurrency, last_heartbeat, started_at, last_seen_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET last_heartbeat=$6, last_seen_at=$8, status=$4, concurrency=$5`,
		w.ID, w.ProjectID, w.Hostname, string(w.Status), w.Concurrency, w.LastHeartbeat, w.StartedAt, w.LastSeenAt, w.Metadata)
	return err
}

func (s *PostgresStore) Heartbeat(ctx context.Context, workerID string, running int, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE workers SET last_heartbeat=$2, last_seen_at=$2, status='active' WHERE id=$1`, workerID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO worker_heartbeats (id, worker_id, sent_at, running) VALUES ($1,$2,$3,$4)`,
		newID(), workerID, now, running)
	return err
}

func (s *PostgresStore) GetWorker(ctx context.Context, id string) (*models.Worker, error) {
	w := &models.Worker{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, hostname, status, concurrency, last_heartbeat, started_at, last_seen_at, metadata
		FROM workers WHERE id=$1`, id).
		Scan(&w.ID, &w.ProjectID, &w.Hostname, &w.Status, &w.Concurrency, &w.LastHeartbeat, &w.StartedAt, &w.LastSeenAt, &w.Metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (s *PostgresStore) ListWorkers(ctx context.Context, projectID string) ([]*models.Worker, error) {
	query := `SELECT id, project_id, hostname, status, concurrency, last_heartbeat, started_at, last_seen_at, metadata FROM workers`
	args := []interface{}{}
	if projectID != "" {
		query += ` WHERE project_id=$1`
		args = append(args, projectID)
	}
	query += ` ORDER BY last_heartbeat DESC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.Worker, 0)
	for rows.Next() {
		w := &models.Worker{}
		if err := rows.Scan(&w.ID, &w.ProjectID, &w.Hostname, &w.Status, &w.Concurrency, &w.LastHeartbeat, &w.StartedAt, &w.LastSeenAt, &w.Metadata); err != nil {
			return nil, err
		}
		items = append(items, w)
	}
	return items, rows.Err()
}

func (s *PostgresStore) MarkStaleWorkers(ctx context.Context, staleAfter time.Duration, now time.Time) (int, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE workers SET status='dead' WHERE status <> 'dead' AND last_heartbeat < $1`, now.Add(-staleAfter))
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ---- Scheduled jobs ----

func (s *PostgresStore) CreateScheduledJob(ctx context.Context, sj *models.ScheduledJob) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO scheduled_jobs (id, project_id, queue_id, name, cron_expr, timezone, payload, priority, enabled, next_run_at, last_run_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		sj.ID, sj.ProjectID, sj.QueueID, sj.Name, sj.CronExpr, sj.Timezone, jobPayload(sj.Payload), sj.Priority, sj.Enabled, sj.NextRunAt, sj.LastRunAt, sj.CreatedAt, sj.UpdatedAt)
	return err
}

func (s *PostgresStore) GetScheduledJob(ctx context.Context, id string) (*models.ScheduledJob, error) {
	sj := &models.ScheduledJob{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, queue_id, name, cron_expr, timezone, payload, priority, enabled, next_run_at, last_run_at, created_at, updated_at
		FROM scheduled_jobs WHERE id=$1`, id).
		Scan(&sj.ID, &sj.ProjectID, &sj.QueueID, &sj.Name, &sj.CronExpr, &sj.Timezone, &sj.Payload, &sj.Priority, &sj.Enabled, &sj.NextRunAt, &sj.LastRunAt, &sj.CreatedAt, &sj.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return sj, nil
}

func (s *PostgresStore) ListScheduledJobs(ctx context.Context, projectID string) ([]*models.ScheduledJob, error) {
	query := `SELECT id, project_id, queue_id, name, cron_expr, timezone, payload, priority, enabled, next_run_at, last_run_at, created_at, updated_at FROM scheduled_jobs`
	args := []interface{}{}
	if projectID != "" {
		query += ` WHERE project_id=$1`
		args = append(args, projectID)
	}
	query += ` ORDER BY name ASC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.ScheduledJob, 0)
	for rows.Next() {
		sj := &models.ScheduledJob{}
		if err := rows.Scan(&sj.ID, &sj.ProjectID, &sj.QueueID, &sj.Name, &sj.CronExpr, &sj.Timezone, &sj.Payload, &sj.Priority, &sj.Enabled, &sj.NextRunAt, &sj.LastRunAt, &sj.CreatedAt, &sj.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, sj)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateScheduledJob(ctx context.Context, sj *models.ScheduledJob) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE scheduled_jobs SET name=$2, cron_expr=$3, timezone=$4, payload=$5, priority=$6, enabled=$7, next_run_at=$8, last_run_at=$9, updated_at=$10
		WHERE id=$1`,
		sj.ID, sj.Name, sj.CronExpr, sj.Timezone, jobPayload(sj.Payload), sj.Priority, sj.Enabled, sj.NextRunAt, sj.LastRunAt, sj.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) FireDueScheduledJob(ctx context.Context, now time.Time, nextRunFn func(*models.ScheduledJob) (time.Time, error)) (*models.ScheduledJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sj := &models.ScheduledJob{}
	err = tx.QueryRow(ctx, `
		SELECT id, project_id, queue_id, name, cron_expr, timezone, payload, priority, enabled, next_run_at, last_run_at, created_at, updated_at
		FROM scheduled_jobs
		WHERE enabled AND next_run_at IS NOT NULL AND next_run_at <= $1
		ORDER BY next_run_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, now).
		Scan(&sj.ID, &sj.ProjectID, &sj.QueueID, &sj.Name, &sj.CronExpr, &sj.Timezone, &sj.Payload, &sj.Priority, &sj.Enabled, &sj.NextRunAt, &sj.LastRunAt, &sj.CreatedAt, &sj.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	next, err := nextRunFn(sj)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE scheduled_jobs SET last_run_at=$2, next_run_at=$3, updated_at=$2 WHERE id=$1`,
		sj.ID, now, next)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	sj.LastRunAt = &now
	sj.NextRunAt = &next
	sj.UpdatedAt = now
	return sj, nil
}

// ---- DLQ ----

func (s *PostgresStore) CreateDLQEntry(ctx context.Context, d *models.DeadLetterJob) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.insertDLQTx(ctx, tx, d); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) insertDLQTx(ctx context.Context, tx pgx.Tx, d *models.DeadLetterJob) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO dead_letter_jobs (id, job_id, project_id, queue_id, payload, reason, attempts, worker_id, failed_at, created_at, requeued_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (job_id) DO NOTHING`,
		d.ID, d.JobID, d.ProjectID, d.QueueID, jobPayload(d.Payload), d.Reason, d.Attempts, d.WorkerID, d.FailedAt, d.CreatedAt, d.RequeuedAt)
	return err
}

func (s *PostgresStore) GetDLQEntry(ctx context.Context, id string) (*models.DeadLetterJob, error) {
	d := &models.DeadLetterJob{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, job_id, project_id, queue_id, payload, reason, attempts, worker_id, failed_at, created_at, requeued_at
		FROM dead_letter_jobs WHERE id=$1`, id).
		Scan(&d.ID, &d.JobID, &d.ProjectID, &d.QueueID, &d.Payload, &d.Reason, &d.Attempts, &d.WorkerID, &d.FailedAt, &d.CreatedAt, &d.RequeuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *PostgresStore) ListDLQ(ctx context.Context, f ListDLQFilter) ([]*models.DeadLetterJob, string, error) {
	if f.Limit <= 0 || f.Limit > DefaultMaxPageSize {
		f.Limit = DefaultMaxPageSize
	}
	ct, cid, err := DecodeCursor(f.Cursor)
	if err != nil {
		return nil, "", err
	}
	where := " WHERE project_id=$1"
	args := []interface{}{f.ProjectID}
	idx := 2
	if f.QueueID != "" {
		args = append(args, f.QueueID)
		where += fmt.Sprintf(" AND queue_id=$%d", idx)
		idx++
	}
	if f.Cursor != "" {
		args = append(args, ct, cid)
		where += fmt.Sprintf(" AND (failed_at, id) < ($%d::timestamptz, $%d)", idx, idx+1)
	}
	args = append(args, f.Limit+1)
	query := fmt.Sprintf(`
		SELECT id, job_id, project_id, queue_id, payload, reason, attempts, worker_id, failed_at, created_at, requeued_at
		FROM dead_letter_jobs%s ORDER BY failed_at DESC, id DESC LIMIT $%d`, where, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]*models.DeadLetterJob, 0, f.Limit)
	for rows.Next() {
		d := &models.DeadLetterJob{}
		if err := rows.Scan(&d.ID, &d.JobID, &d.ProjectID, &d.QueueID, &d.Payload, &d.Reason, &d.Attempts, &d.WorkerID, &d.FailedAt, &d.CreatedAt, &d.RequeuedAt); err != nil {
			return nil, "", err
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > f.Limit {
		items = items[:f.Limit]
		last := items[len(items)-1]
		next = EncodeCursor(last.FailedAt, last.ID)
	}
	return items, next, nil
}

func (s *PostgresStore) MarkRequeued(ctx context.Context, id string, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE dead_letter_jobs SET requeued_at=$2 WHERE id=$1`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// jobPayload returns a non-nil JSONB payload.
func jobPayload(b []byte) []byte {
	if len(b) == 0 {
		return []byte("{}")
	}
	return b
}

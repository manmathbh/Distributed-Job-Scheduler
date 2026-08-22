-- 0001_init.sql
-- Authoritative relational schema for the distributed job scheduler.

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organizations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organization_members (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL DEFAULT 'member',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, user_id)
);

CREATE TABLE IF NOT EXISTS projects (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    owner_id        TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL,
    slug            TEXT UNIQUE NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_projects_organization ON projects(organization_id);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id);

CREATE TABLE IF NOT EXISTS retry_policies (
    id            TEXT PRIMARY KEY,
    strategy      TEXT NOT NULL DEFAULT 'exponential',
    max_attempts  INT  NOT NULL DEFAULT 3,
    initial_delay_ms BIGINT NOT NULL DEFAULT 1000,
    max_delay_ms     BIGINT NOT NULL DEFAULT 60000,
    multiplier    DOUBLE PRECISION NOT NULL DEFAULT 2.0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_retry_strategy CHECK (strategy IN ('fixed','linear','exponential')),
    CONSTRAINT chk_retry_max_attempts CHECK (max_attempts >= 0),
    CONSTRAINT chk_retry_multiplier CHECK (multiplier >= 1.0)
);

CREATE TABLE IF NOT EXISTS queues (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    priority        INT NOT NULL DEFAULT 0,
    concurrency     INT NOT NULL DEFAULT 1,
    status          TEXT NOT NULL DEFAULT 'active',
    retry_policy_id TEXT REFERENCES retry_policies(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name),
    CONSTRAINT chk_queue_concurrency CHECK (concurrency > 0),
    CONSTRAINT chk_queue_status CHECK (status IN ('active','paused'))
);
CREATE INDEX IF NOT EXISTS idx_queues_project ON queues(project_id);

CREATE TABLE IF NOT EXISTS jobs (
    id                 TEXT PRIMARY KEY,
    project_id         TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    queue_id           TEXT NOT NULL REFERENCES queues(id) ON DELETE CASCADE,
    type               TEXT NOT NULL DEFAULT 'immediate',
    payload            JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority           INT NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'queued',
    scheduled_at       TIMESTAMPTZ,
    available_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    max_attempts       INT NOT NULL DEFAULT 3,
    retry_strategy     TEXT NOT NULL DEFAULT 'exponential',
    retry_initial_delay_ms BIGINT NOT NULL DEFAULT 1000,
    retry_max_delay_ms     BIGINT NOT NULL DEFAULT 60000,
    retry_multiplier   DOUBLE PRECISION NOT NULL DEFAULT 2.0,
    attempts           INT NOT NULL DEFAULT 0,
    last_error         TEXT NOT NULL DEFAULT '',
    claim_token        TEXT,
    lease_expires_at   TIMESTAMPTZ,
    worker_id          TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    claimed_at         TIMESTAMPTZ,
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    failed_at          TIMESTAMPTZ,
    CONSTRAINT chk_job_status CHECK (status IN ('queued','scheduled','claimed','running','completed','failed','cancelled')),
    CONSTRAINT chk_job_type CHECK (type IN ('immediate','delayed','scheduled','recurring'))
);

-- Claiming query index: (queue_id, status, available_at) with priority ordering.
CREATE INDEX IF NOT EXISTS idx_jobs_claim ON jobs(queue_id, status, available_at);
CREATE INDEX IF NOT EXISTS idx_jobs_claim_priority ON jobs(queue_id, status, priority DESC, created_at ASC);
-- Status lookup + project pagination.
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_project_created ON jobs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_queue_status ON jobs(queue_id, status);
CREATE INDEX IF NOT EXISTS idx_jobs_worker ON jobs(worker_id);
CREATE INDEX IF NOT EXISTS idx_jobs_available ON jobs(available_at);

CREATE TABLE IF NOT EXISTS job_executions (
    id           TEXT PRIMARY KEY,
    job_id       TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    attempt      INT NOT NULL DEFAULT 0,
    worker_id    TEXT,
    status       TEXT NOT NULL DEFAULT 'running',
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms  BIGINT NOT NULL DEFAULT 0,
    error        TEXT NOT NULL DEFAULT '',
    retryable    BOOLEAN NOT NULL DEFAULT false,
    metadata     JSONB
);
CREATE INDEX IF NOT EXISTS idx_job_executions_job ON job_executions(job_id, attempt);
CREATE INDEX IF NOT EXISTS idx_job_executions_worker ON job_executions(worker_id);

CREATE TABLE IF NOT EXISTS workers (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL DEFAULT '',
    hostname       TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'active',
    concurrency    INT NOT NULL DEFAULT 1,
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata       JSONB
);
CREATE INDEX IF NOT EXISTS idx_workers_project ON workers(project_id);
CREATE INDEX IF NOT EXISTS idx_workers_heartbeat ON workers(last_heartbeat);

CREATE TABLE IF NOT EXISTS worker_heartbeats (
    id         TEXT PRIMARY KEY,
    worker_id  TEXT NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    running    INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_worker_heartbeats_worker ON worker_heartbeats(worker_id, sent_at);

CREATE TABLE IF NOT EXISTS job_logs (
    id         TEXT PRIMARY KEY,
    job_id     TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    attempt    INT NOT NULL DEFAULT 0,
    level      TEXT NOT NULL DEFAULT 'info',
    message    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_job_logs_job ON job_logs(job_id, created_at);

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    queue_id    TEXT NOT NULL REFERENCES queues(id) ON DELETE CASCADE,
    name        TEXT NOT NULL DEFAULT '',
    cron_expr   TEXT NOT NULL,
    timezone    TEXT NOT NULL DEFAULT 'UTC',
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority    INT NOT NULL DEFAULT 0,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next ON scheduled_jobs(next_run_at);

CREATE TABLE IF NOT EXISTS dead_letter_jobs (
    id          TEXT PRIMARY KEY,
    job_id      TEXT UNIQUE NOT NULL,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    queue_id    TEXT NOT NULL REFERENCES queues(id) ON DELETE CASCADE,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason      TEXT NOT NULL DEFAULT '',
    attempts    INT NOT NULL DEFAULT 0,
    worker_id   TEXT,
    failed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    requeued_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_dlq_project ON dead_letter_jobs(project_id, failed_at);
CREATE INDEX IF NOT EXISTS idx_dlq_queue ON dead_letter_jobs(queue_id);

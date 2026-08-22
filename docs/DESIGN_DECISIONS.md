# Design Decisions

## Why PostgreSQL

The assignment requires a relational, authoritative data model (users, organizations, projects,
queues, jobs, executions, workers, heartbeats, logs, schedules, retry policies, dead letters).
PostgreSQL provides:

- **ACID transactions** and row-level locking, which make atomic job claiming straightforward and
  correct (`SELECT … FOR UPDATE SKIP LOCKED`).
- **Rich indexing** for the claiming and pagination hot paths.
- **Durability** out of the box (its own WAL), which is what a production scheduler needs.

The previous implementation was a single-node in-memory queue; it could not provide the required
multi-worker, cross-instance atomicity. PostgreSQL becomes the single source of truth.

## Why Redis

Redis is retained for **API-key authentication** (the existing `auth.RedisStore`) because API keys
are high-frequency, low-cardinality reads where Redis's `O(1)` lookups are a good fit. Redis is
deliberately **not** used for job state — introducing a second source of truth for jobs would
require a consistency protocol with no clear benefit here.

## Role of the WAL

The existing Write-Ahead Log (`internal/wal`) provided durability for the legacy in-memory queue.
It is **kept** for backward compatibility: the original `/jobs`, `/jobs/lease`, `/jobs/{id}/ack`
endpoints and their tests continue to work exactly as before. For the new scheduler, PostgreSQL's
own WAL is the durability mechanism, so the application-level WAL is not duplicated there.

## Atomic claiming

`ClaimJob` runs, in a single transaction:

1. `SELECT concurrency, status FROM queues WHERE id=$1 FOR UPDATE` (serialize on the queue).
2. Count `claimed`/`running` jobs; bail with `ErrNoJobs` if the queue is at its concurrency limit.
3. `SELECT … WHERE queue_id=$1 AND status='queued' AND available_at <= now() ORDER BY priority DESC,
   created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED`.
4. `UPDATE … SET status='claimed', claim_token=…, worker_id=…, lease_expires_at=now()+lease`.

`FOR UPDATE SKIP LOCKED` means a second worker cannot see (and therefore cannot claim) a row another
transaction is claiming, guaranteeing no double-claim. The concurrency test in
`internal/store` launches many goroutines against the same jobs and asserts each job is claimed
exactly once.

## Concurrency model

- Claims are lease-based: `lease_expires_at` bounds how long a worker may hold a job.
- A `claim_token` (random UUID) is set on claim; completion/failure must present the matching token,
  so a worker whose lease expired cannot clobber a job re-claimed by another worker.
- Queue concurrency limits are enforced inside the claim transaction.

## Retry strategy

`internal/retry` computes the next delay deterministically for three strategies:

- **fixed**: constant `initial_delay`.
- **linear**: `attempt × initial_delay`.
- **exponential**: `initial_delay × multiplier^(attempt-1)`.

All delays are capped by `max_delay`, and attempts are capped by `max_attempts` (no infinite
retries). A failed job with attempts remaining becomes `scheduled` with `available_at = now + delay`;
otherwise it is dead-lettered.

## Heartbeat strategy

Workers write a heartbeat row and update `last_heartbeat` every `HEARTBEAT_INTERVAL`. The scheduler
marks workers `dead` when their heartbeat is older than 60s. Heartbeat records are retained for
history.

## Scheduling strategy

- Delayed/scheduled jobs are stored as `scheduled` with a future `available_at`; the scheduler
  promotes them to `queued` when due.
- Recurring jobs are stored in `scheduled_jobs` with a `next_run_at`. Firing uses
  `FOR UPDATE SKIP LOCKED` and advances `next_run_at` inside the lock, so multiple scheduler
  instances never materialize a duplicate job.
- Timezone: cron expressions are evaluated in the schedule's IANA timezone (default `UTC`); invalid
  timezones fall back to UTC.

## Consistency model & failure recovery

Delivery is **at-least-once**. Execution should be **idempotent** (the handler contract). Exactly-once
execution is *not* claimed. Recovery paths:

- Worker crash / lease expiry → scheduler requeues the job (or dead-letters it if attempts are
  exhausted).
- Stale worker → marked `dead`.
- DB outage → API returns structured errors; the in-process worker pauses until the DB recovers.

## Indexing choices

- `jobs(queue_id, status, available_at)` and `jobs(queue_id, status, priority DESC, created_at ASC)`
  — the claiming query.
- `jobs(project_id, created_at DESC)` — cursor pagination by project.
- `jobs(status)`, `jobs(worker_id)`, `jobs(available_at)` — filtering/lookup.
- `job_executions(job_id, attempt)`, `job_logs(job_id, created_at)`,
  `scheduled_jobs(next_run_at)`, `dead_letter_jobs(project_id, failed_at)`.

## Trade-offs

- The legacy `/jobs` in-memory queue is retained rather than deleted, yielding two job paths. This
  preserves the existing SDK, examples, and 159 pre-existing tests. The `/api/v1` PostgreSQL path is
  the authoritative system; the legacy path is explicitly a compatibility layer.
- `MemoryStore` exists for unit tests and offline dev (`STORE_MODE=memory`). It is not a "fake"
  persistence layer — PostgreSQL is the real path; the memory implementation is the test double that
  shares the exact same `Store` contract.
- Execution exactly-once is intentionally not promised; at-least-once + idempotent handlers is the
  honest, production-standard guarantee.

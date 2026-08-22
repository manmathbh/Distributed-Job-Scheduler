# Architecture

## Overview

The distributed job scheduler is a single Go module with a layered, modular architecture. The core
value proposition is **reliable, concurrent job execution** backed by PostgreSQL as the single
source of truth.

```
Client (SDK / curl / dashboard)
   │  Authorization: Bearer <api-key>   (Redis-backed API keys)
   ▼
HTTP API  (internal/api)
   │
   ├── Legacy /jobs, /jobs/lease, /jobs/{id}/ack, /admin/keys, /keys
   │       └── queue.Core  (in-memory + WAL)         [backward compatibility]
   │
   └── /api/v1/*  (versioned scheduler API)
           ├── authz (owner/project scoping + admin bypass)
           ├── service.Service  (business rules)
           │     ├── ProjectService / QueueService / JobService
           │     ├── Worker lifecycle (claim → run → complete/fail)
           │     └── DLQ requeue
           └── store.Store (interface)
                 ├── PostgresStore  ← AUTHORITATIVE
                 └── MemoryStore    ← tests / offline dev
   ▼
PostgreSQL
   │
   ├── Scheduler (internal/scheduler)  — due-job promotion, lease recovery,
   │       stale-worker marking, cron firing (atomic, skip-locked)
   │
   └── Worker (internal/worker)  — poll → atomic claim → execute → heartbeat
           └── graceful shutdown (stop claiming, drain in-flight)
```

## Components

| Component | Package | Responsibility |
|-----------|---------|----------------|
| Models | `internal/models` | Domain structs, enums, centralized state machine |
| Config | `internal/config` | Environment-driven configuration |
| DB | `internal/db` | Connection pool + embedded migration runner |
| Store | `internal/store` | `Store` interface; `PostgresStore`, `MemoryStore` |
| Retry | `internal/retry` | Deterministic backoff (fixed/linear/exponential) |
| Service | `internal/service` | Business rules, submission, lifecycle orchestration |
| Scheduler | `internal/scheduler` | Periodic maintenance + cron firing |
| Worker | `internal/worker` | In-process job executor |
| Metrics | `internal/metrics` | Atomic counters + snapshot |
| API | `internal/api` | HTTP handlers, authz, pagination |
| Auth | `internal/auth` | Redis API keys (client/worker/admin) |
| Web | `internal/web` | Embedded dashboard SPA |
| SDK | `pkg/sdk` | Legacy HTTP client + polling worker |

## Data flow (job lifecycle)

1. Client `POST /api/v1/projects/{p}/queues/{q}/jobs` → `service.SubmitJob` snapshots the queue's
   retry policy and writes a `queued` (or `scheduled`) job row.
2. Scheduler promotes `scheduled` jobs whose `available_at <= now` to `queued`.
3. Worker claims via `ClaimJob` (`FOR UPDATE SKIP LOCKED`), enforcing the queue concurrency limit.
4. Worker marks `running` (creates a `job_executions` row), runs the handler.
5. Success → `completed`; failure → retry (backoff, `scheduled`) or dead-letter (`failed` + DLQ row).
6. Scheduler recovers expired leases (worker crash) and marks stale workers dead.

## Consistency model

- **PostgreSQL** owns all durable scheduler state.
- **Redis** stores API keys and transient data only (never authoritative job state).
- **WAL** (legacy) provides durability for the in-process queue retained for compatibility; the
  new scheduler relies on PostgreSQL's own WAL.
- Delivery semantics are **at-least-once** with idempotent execution via claim tokens; a stale
  worker cannot overwrite a job claimed by another (claim-token guard + lease expiry).

## Concurrency model

- Atomic claim: `SELECT … FOR UPDATE SKIP LOCKED` in a transaction, serialized on the queue row to
  enforce its concurrency limit.
- Claim tokens (`claim_token`) guard completion/failure so an expired lease's owner cannot clobber a
  re-claimed job.
- Scheduler cron firing uses `FOR UPDATE SKIP LOCKED` + advancing `next_run_at` inside the lock so
  multiple instances cannot duplicate a schedule.

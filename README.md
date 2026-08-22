# Distributed Job Scheduler

A production-inspired, distributed job scheduling platform in Go. It reliably executes asynchronous
background jobs across multiple workers with PostgreSQL as the authoritative store, Redis for API-key
authentication, and an embedded dashboard.

## Overview

- **Projects & queues**: multi-tenant projects, each owning multiple named queues with priority,
  concurrency limits, retry policy, and pause/resume.
- **Job types**: immediate, delayed, scheduled, recurring (cron), and batch submission.
- **Reliability**: atomic job claiming (`FOR UPDATE SKIP LOCKED`), leases, worker heartbeats, stale
  worker/lease recovery, configurable retries, and a Dead Letter Queue.
- **Observability**: execution history, per-job logs, queue statistics, throughput metrics, and a
  web dashboard.
- **Backward compatibility**: the original WAL-backed in-memory queue (`/jobs`, `/jobs/lease`,
  `/jobs/{id}/ack`) is preserved alongside the new `/api/v1` surface.

## Architecture

```
Client (SDK / curl / dashboard)
   │  Authorization: Bearer <api-key>
   ▼
HTTP API (internal/api)
   ├── Legacy /jobs, /admin/keys, /keys   → queue.Core (WAL, in-memory)   [compat]
   └── /api/v1/*                          → services → store.Store
        ├── PostgresStore  (authoritative)
        └── MemoryStore    (tests)
   ▼
PostgreSQL  (projects, queues, jobs, executions, workers, heartbeats, logs,
             schedules, retry policies, dead letters)
   ▲
Scheduler  → delayed/scheduled/cron promotion, lease recovery, stale workers
Worker     → poll → atomic claim → execute → heartbeat → ack/fail

Redis  → API keys + transient state (non-authoritative)
WAL    → durability for the legacy in-process queue (kept for compatibility)
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and
[docs/DESIGN_DECISIONS.md](docs/DESIGN_DECISIONS.md).

## Prerequisites

- Go 1.25+
- Docker (recommended) — runs PostgreSQL, Redis, API, and worker.

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable` | PostgreSQL connection |
| `STORE_MODE` | `postgres` | `postgres` or `memory` (tests/dev only) |
| `REDIS_URL` | `localhost:6379` | Redis for API keys |
| `WAL_DIR` | `./data` | legacy WAL directory |
| `LEASE_DURATION` | `30s` | worker lease duration |
| `LEASE_CHECK_INTERVAL` | `5s` | legacy lease sweep interval |
| `HEARTBEAT_INTERVAL` | `10s` | worker heartbeat interval |
| `SCHEDULER_ENABLED` | `true` | run the scheduler loop |
| `SCHEDULER_INTERVAL` | `2s` | scheduler tick interval |
| `WORKER_ENABLED` | `true` | run the in-process worker |
| `WORKER_CONCURRENCY` | `4` | in-process worker concurrency |
| `WORKER_ID` | auto | worker identifier |
| `SEED_DEMO` | `true` | seed a demo project/queues on first boot |
| `TEST_DATABASE_URL` | *(unset)* | enables PostgreSQL integration tests |

## Docker Setup

```bash
docker compose up --build
```

Services: `api` (server + scheduler + in-process worker), `worker` (standalone worker),
`postgres`, `redis`. The dashboard is served by `api` at http://localhost:8080/dashboard/.

On first boot, development API keys are seeded and logged:

- client: `client_…` (owner `dev-client`)
- worker: `worker_…` (owner `dev-worker`)
- admin:  `admin_…`  (owner `dev-admin`)

## Local Setup

```bash
# start dependencies
docker compose up -d postgres redis

# run the server (auto-migrates the schema)
go run ./cmd/server

# run a standalone worker in another terminal
go run ./cmd/worker

# open the dashboard
open http://localhost:8080/dashboard/
```

## Database Migration

Migrations are embedded and applied automatically at startup (`internal/db/migrations`). No manual
step is required.

## API Examples

```bash
# create a project (uses your API key's owner)
curl -s -X POST http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" \
  -d '{"name":"My Project"}'

# create a queue
curl -s -X POST http://localhost:8080/api/v1/projects/$PROJECT_ID/queues \
  -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" \
  -d '{"name":"default","concurrency":4,"retry_strategy":"exponential","max_attempts":3}'

# submit an immediate job
curl -s -X POST http://localhost:8080/api/v1/projects/$PROJECT_ID/queues/$QUEUE_ID/jobs \
  -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" \
  -d '{"type":"immediate","payload":{"url":"https://example.com"}}'

# submit a recurring (cron) job
curl -s -X POST http://localhost:8080/api/v1/projects/$PROJECT_ID/queues/$QUEUE_ID/jobs \
  -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" \
  -d '{"type":"recurring","cron_expr":"*/5 * * * *","payload":{"task":"sync"}}'

# get a job
curl -s http://localhost:8080/api/v1/jobs/$JOB_ID \
  -H "Authorization: Bearer $API_KEY"
```

Full API reference: [docs/API.md](docs/API.md).

## Tests

```bash
go test ./...          # unit + integration tests
go test -race ./...    # race detector
go vet ./...

# PostgreSQL integration tests (requires a running Postgres)
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable" go test ./internal/store/
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Design decisions](docs/DESIGN_DECISIONS.md)
- [API reference](docs/API.md)
- [ER diagram](docs/ER_DIAGRAM.md)

## License

MIT — see [LICENSE](LICENSE).

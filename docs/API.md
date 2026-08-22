# API Reference

All `/api/v1` endpoints require `Authorization: Bearer <api-key>`. Errors are structured as
`{"error": "...", "code": "..."}` with appropriate HTTP status codes. Collections use cursor
pagination (`limit` + `cursor`) with stable `created_at DESC, id DESC` ordering.

Authorization model: **admin** keys access everything; **client**/**worker** keys are scoped to
projects whose `owner_id` matches the key's owner.

---

## Projects

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/projects` | Create a project (owned by the caller) |
| `GET` | `/api/v1/projects` | List projects (`limit`, `cursor`) |
| `GET` | `/api/v1/projects/{projectID}` | Get a project |
| `PATCH` | `/api/v1/projects/{projectID}` | Update name/description |
| `DELETE` | `/api/v1/projects/{projectID}` | Delete a project (cascades) |

Create body: `{"name": "...", "description": "..."}`

## Queues

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/projects/{projectID}/queues` | Create a queue |
| `GET` | `/api/v1/projects/{projectID}/queues` | List queues |
| `GET` | `/api/v1/queues/{queueID}` | Get a queue |
| `PATCH` | `/api/v1/queues/{queueID}` | Update a queue |
| `POST` | `/api/v1/queues/{queueID}/pause` | Pause |
| `POST` | `/api/v1/queues/{queueID}/resume` | Resume |
| `DELETE` | `/api/v1/queues/{queueID}` | Delete |
| `GET` | `/api/v1/queues/{queueID}/stats` | Queue statistics |

Create/update body:

```json
{
  "name": "default",
  "description": "...",
  "priority": 0,
  "concurrency": 4,
  "retry_strategy": "exponential",
  "max_attempts": 3,
  "initial_delay_ms": 1000,
  "max_delay_ms": 60000,
  "multiplier": 2.0
}
```

## Jobs

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/projects/{p}/queues/{q}/jobs` | Submit a job |
| `POST` | `/api/v1/projects/{p}/queues/{q}/jobs/batch` | Batch submit |
| `GET` | `/api/v1/projects/{p}/jobs` | List jobs (`queue_id`, `status`, `type`, `limit`, `cursor`) |
| `GET` | `/api/v1/jobs/{jobID}` | Get a job |
| `POST` | `/api/v1/jobs/{jobID}/retry` | Re-queue a failed job |
| `POST` | `/api/v1/jobs/{jobID}/cancel` | Cancel a job |
| `GET` | `/api/v1/jobs/{jobID}/executions` | Execution history |
| `GET` | `/api/v1/jobs/{jobID}/logs` | Job logs (`limit`) |

Submit body (type one of `immediate`, `delayed`, `scheduled`, `recurring`):

```json
{
  "type": "immediate",
  "payload": {"url": "https://example.com"},
  "priority": 0,
  "scheduled_at": "2026-01-01T00:00:00Z",
  "delay_ms": 5000,
  "max_attempts": 3,
  "retry_strategy": "exponential",
  "cron_expr": "*/5 * * * *",
  "timezone": "UTC",
  "name": "sync"
}
```

Batch body: `{"jobs": [ { "type": "immediate", "payload": {...} }, ... ]}`

## Schedules (recurring)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/projects/{p}/schedules` | Create a recurring schedule (`cron_expr` + `queue_id` query) |
| `GET` | `/api/v1/projects/{p}/schedules` | List schedules |

## Dead Letter Queue

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/projects/{p}/dead-letter` | List dead-lettered jobs (`queue_id`, `limit`, `cursor`) |
| `POST` | `/api/v1/dead-letter/{dlqID}/requeue` | Re-enqueue a dead job |

## Workers

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/workers` | List workers (`project_id` query, admin) |
| `GET` | `/api/v1/workers/{workerID}` | Get a worker |
| `POST` | `/api/v1/workers/{workerID}/heartbeat` | Record a heartbeat (`{"running": n}`) |

## Metrics & overview

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/metrics` | Throughput counters, uptime, avg execution |
| `GET` | `/api/v1/overview` | Dashboard summary (counts, workers, DLQ) |

## Job states

`queued → scheduled → claimed → running → completed`

Failure path: `running → scheduled (retry)` or `running → failed (dead-lettered)`. `cancelled` is a
terminal state from any non-terminal state.

## Legacy endpoints (backward compatible)

`POST /jobs`, `POST /jobs/lease`, `POST /jobs/{id}/ack`, `GET /jobs/{id}`, `/health`,
`POST/GET /admin/keys`, `GET /keys`, `DELETE /keys/{key}` behave as in the original WAL-backed
queue.

# Distributed Job Scheduler

A distributed job scheduling platform built in Go. It lets you create
projects and queues, submit background jobs, and process them with
workers. PostgreSQL stores the main scheduler data, Redis handles
API-key authentication and transient state, and the built-in dashboard
gives you a simple way to manage and monitor everything.

The project is designed to cover the parts you would expect from a real
job scheduler: retries, delayed and scheduled jobs, worker heartbeats,
leases, recovery, dead-letter handling, execution history, and basic
monitoring.

## What the project does

The scheduler is organized around projects, queues, jobs, and workers.

-   **Projects** keep different applications or users separated.
-   **Queues** group related jobs and control concurrency and retry
    behavior.
-   **Jobs** can run immediately, after a delay, at a scheduled time, or
    repeatedly using cron.
-   **Workers** pick up available jobs, execute them, send heartbeats,
    and acknowledge or fail them.
-   **PostgreSQL** is the main source of truth for projects, queues,
    jobs, executions, workers, schedules, and dead letters.
-   **Redis** is used for API-key authentication and transient state.
-   **The dashboard** provides a web interface for connecting with an
    API key, creating projects and queues, submitting jobs, and checking
    their status.
-   **The legacy WAL-backed queue API** is still available for backward
    compatibility.

## Architecture

The system consists of an HTTP API, PostgreSQL, Redis, schedulers, workers,
and the web dashboard.

![Distributed Job Scheduler Architecture](docs/images/Architecture.png)

The API accepts requests from clients and the dashboard using Bearer API
keys. PostgreSQL acts as the authoritative store for projects, queues, jobs,
workers, executions, schedules, and dead-letter records.

Redis is used for API-key authentication and transient state, while the WAL
continues to support the legacy queue implementation.

### Job Lifecycle

Jobs are submitted to PostgreSQL and remain queued until a worker atomically
claims them. The worker executes the job and either completes it, retries it
when attempts remain, or moves it to the Dead Letter Queue after all attempts
are exhausted.

![Job Lifecycle](docs/images/Flow.png)
The main flow is:

1.  A client connects using an API key.
2.  The API checks the key and identifies its owner.
3.  The owner creates a project and one or more queues.
4.  Jobs are submitted to a queue.
5.  PostgreSQL stores the job and its current state.
6.  Workers poll for available jobs and claim them atomically.
7.  A worker executes the job and periodically sends heartbeats.
8.  The worker acknowledges successful jobs or reports failures.
9.  Failed jobs can be retried according to the queue's retry policy.
10. Jobs that exceed their retry limit can be moved to the Dead Letter
    Queue.
11. The scheduler promotes delayed, scheduled, and recurring jobs and
    also helps recover stale leases.

## Main features

### Projects and queues

Each API-key owner can create projects. A project can contain multiple
queues.

Queues can define:

-   Queue name
-   Concurrency limit
-   Retry strategy
-   Maximum attempts
-   Pause and resume state
-   Job priority

This makes it possible to keep different types of background work
separated.

### Job types

The scheduler supports:

-   Immediate jobs
-   Delayed jobs
-   Scheduled jobs
-   Recurring jobs using cron expressions
-   Batch submission

### Reliability

The scheduler includes several mechanisms to handle failures and worker
crashes:

-   Atomic job claiming using PostgreSQL transactions and
    `FOR UPDATE SKIP LOCKED`
-   Worker leases
-   Worker heartbeats
-   Expired lease recovery
-   Stale worker recovery
-   Configurable retry policies
-   Dead Letter Queue
-   Execution history

### Dashboard

The API serves a built-in dashboard at:

`http://localhost:8080/dashboard/`

The dashboard can be used to:

-   Register a client
-   Connect with an API key
-   Create and switch between projects
-   Create queues
-   Submit jobs
-   View job status
-   View workers
-   Check dead-lettered jobs
-   View metrics and scheduler activity

## Getting started

There are two ways to run the project:

1.  Run everything locally with Docker.
2.  Use a deployed instance and start from the dashboard.

The second option is useful if you only want to try the application
without setting up PostgreSQL and Redis locally.

## Starting your own project from the dashboard

If you are using a deployed version of the scheduler, you can start
without creating anything from the command line.

### 1. Open the dashboard

Open the dashboard URL provided by the deployment.

For a local setup, it is:

``` text
http://localhost:8080/dashboard/
```

You should see the Job Scheduler dashboard with an API-key field and a
`Register` button.

### 2. Register a client

Click **Register**.

Enter your name and email address and create a client.

After registration, the application gives you a client API key.

Keep this key somewhere safe. It is what the dashboard and API use to
identify your client.

### 3. Connect your client

Paste the client API key into the API-key field at the top of the
dashboard and click **Connect**.

After a successful connection, the dashboard loads the projects
belonging to that client.

If this is a new account, you may not have any projects yet.

### 4. Create your first project

Use the project selector and the `+` button to create a project.

For example:

``` text
Project name: My First Project
```

Select the project after it is created.

### 5. Create a queue

Open the **Queues** section and create a queue.

For a first test, something simple is enough:

``` text
Queue name: default
Concurrency: 4
Retry strategy: exponential
Maximum attempts: 3
```

The queue is where jobs will be placed.

### 6. Submit your first job

Open **Jobs**.

Click **Submit Job**.

Select the queue you just created and use:

``` json
{
  "message": "Hello from my first scheduled job"
}
```

For the first test, keep the job type as `immediate` and leave the delay
at `0`.

Click **Save**.

### 7. Check the job

After submitting the job, the job should appear in the Jobs page.

Depending on the worker state, you should see the job move through
states such as:

``` text
queued -> running -> completed
```

If the worker is not available or the job fails, the status will show
the corresponding state.

You can also open the **Overview**, **Workers**, **Dead Letter**, and
**Metrics** pages to check what is happening in the system.

## Dashboard workflow

The normal flow for a new user is:

``` mermaid
flowchart LR
    A[Open Dashboard] --> B[Register]
    B --> C[Get Client API Key]
    C --> D[Connect]
    D --> E[Create Project]
    E --> F[Create Queue]
    F --> G[Submit Job]
    G --> H[Worker Claims Job]
    H --> I[Job Executes]
    I --> J[Completed / Failed]
    J --> K[View Status & Metrics]
```

## Running with Docker

Docker Compose starts the main dependencies and application services.

``` bash
docker compose up --build
```

The setup includes:

-   `api` - HTTP API, scheduler, and in-process worker
-   `worker` - standalone worker
-   `postgres` - PostgreSQL database
-   `redis` - Redis instance

After the containers start, open:

``` text
http://localhost:8080/dashboard/
```

The application automatically applies the database migrations when it
starts.

For development, the application can also seed demo data when
`SEED_DEMO=true`.

## Local development

If you want to run the Go services directly, start PostgreSQL and Redis
first:

``` bash
docker compose up -d postgres redis
```

Then start the API server:

``` bash
go run ./cmd/server
```

In another terminal, start a standalone worker:

``` bash
go run ./cmd/worker
```

Open the dashboard:

``` text
http://localhost:8080/dashboard/
```

The server automatically applies the embedded database migrations during
startup.

## Environment variables

  -----------------------------------------------------------------------------------------------------------------------------
  Variable                 Default                                                                      Purpose
  ------------------------ ---------------------------------------------------------------------------- -----------------------
  `PORT`                   `8080`                                                                       HTTP server port

  `DATABASE_URL`           `postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable`   PostgreSQL connection

  `STORE_MODE`             `postgres`                                                                   `postgres` or `memory`

  `REDIS_URL`              `localhost:6379`                                                             Redis connection used
                                                                                                        for API keys

  `WAL_DIR`                `./data`                                                                     Legacy WAL directory

  `LEASE_DURATION`         `30s`                                                                        Worker lease duration

  `LEASE_CHECK_INTERVAL`   `5s`                                                                         Legacy lease sweep
                                                                                                        interval

  `HEARTBEAT_INTERVAL`     `10s`                                                                        Worker heartbeat
                                                                                                        interval

  `SCHEDULER_ENABLED`      `true`                                                                       Enables the scheduler

  `SCHEDULER_INTERVAL`     `2s`                                                                         Scheduler tick interval

  `WORKER_ENABLED`         `true`                                                                       Enables the in-process
                                                                                                        worker

  `WORKER_CONCURRENCY`     `4`                                                                          In-process worker
                                                                                                        concurrency

  `WORKER_ID`              `auto`                                                                       Worker identifier

  `SEED_DEMO`              `true`                                                                       Seeds demo data on
                                                                                                        first boot

  `TEST_DATABASE_URL`      unset                                                                        Enables PostgreSQL
                                                                                                        integration tests
  -----------------------------------------------------------------------------------------------------------------------------

## API usage

The API can also be used directly instead of the dashboard.

Set your API key first:

``` bash
export API_KEY="your-client-api-key"
```

### Create a project

``` bash
curl -s -X POST http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project"}'
```

Save the returned project ID.

### Create a queue

Replace `$PROJECT_ID` with your project ID:

``` bash
curl -s -X POST \
  http://localhost:8080/api/v1/projects/$PROJECT_ID/queues \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"default",
    "concurrency":4,
    "retry_strategy":"exponential",
    "max_attempts":3
  }'
```

Save the returned queue ID.

### Submit an immediate job

``` bash
curl -s -X POST \
  http://localhost:8080/api/v1/projects/$PROJECT_ID/queues/$QUEUE_ID/jobs \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"immediate",
    "payload":{
      "message":"hello from the scheduler"
    }
  }'
```

### Submit a recurring job

``` bash
curl -s -X POST \
  http://localhost:8080/api/v1/projects/$PROJECT_ID/queues/$QUEUE_ID/jobs \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"recurring",
    "cron_expr":"*/5 * * * *",
    "payload":{
      "task":"sync"
    }
  }'
```

### Get a job

``` bash
curl -s \
  http://localhost:8080/api/v1/jobs/$JOB_ID \
  -H "Authorization: Bearer $API_KEY"
```

More endpoints and request/response details are available in the API
documentation.

## How a job moves through the system

``` mermaid
sequenceDiagram
    participant C as Client
    participant API as HTTP API
    participant DB as PostgreSQL
    participant W as Worker
    participant S as Scheduler

    C->>API: Submit job
    API->>DB: Store job
    DB-->>API: Job created
    API-->>C: Job ID

    loop Worker polling
        W->>DB: Find available job
        DB-->>W: Atomically claim job
    end

    W->>DB: Mark running
    W->>DB: Send heartbeat
    W->>W: Execute job

    alt Job succeeds
        W->>DB: Acknowledge / complete
    else Job fails
        W->>DB: Record failure
        DB-->>W: Retry if attempts remain
    end

    S->>DB: Promote delayed / scheduled / recurring jobs
    S->>DB: Recover expired leases
```

## Database

The database schema is managed through embedded migrations under:

``` text
internal/db/migrations
```

Migrations are applied automatically when the application starts, so
there is no separate migration command required for the normal setup.

The PostgreSQL database stores the main scheduler state, including:

-   Projects
-   Queues
-   Jobs
-   Job executions
-   Workers
-   Worker heartbeats
-   Schedules
-   Retry information
-   Logs
-   Dead-lettered jobs

## Reliability and recovery

A worker does not simply take a job and assume it will finish.

When a worker claims a job, the scheduler gives it a lease. The worker
sends heartbeats while processing the job. If the worker disappears or
stops sending heartbeats, the scheduler can detect the stale lease and
make the job available for recovery.

Job claiming uses PostgreSQL locking so multiple workers can safely
compete for work without processing the same available job at the same
time.

A simplified view is:

``` text
Available job
     |
     v
Worker claims job
     |
     v
Lease + heartbeat
     |
     +---- success ----> Completed
     |
     +---- failure -----> Retry
                              |
                              +---- attempts remaining --> Available again
                              |
                              +---- attempts exhausted -> Dead Letter Queue
```

## Legacy API

The original WAL-backed in-memory queue is kept for backward
compatibility.

The legacy endpoints include:

``` text
/jobs
/jobs/lease
/jobs/{id}/ack
/admin/keys
/keys
```

The newer scheduler API is available under:

``` text
/api/v1/*
```

The legacy implementation uses the WAL for durability of the in-process
queue, while the newer scheduler uses PostgreSQL as its authoritative
store.

## Testing

Run the complete test suite:

``` bash
go test ./...
```

Run tests with the race detector:

``` bash
go test -race ./...
```

Run static checks:

``` bash
go vet ./...
```

For PostgreSQL integration tests, provide a test database:

``` bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable" \
go test ./internal/store/
```

## Project structure

A simplified view of the repository:

``` text
.
├── cmd/
│   ├── server/
│   └── worker/
├── internal/
│   ├── api/
│   ├── config/
│   ├── db/
│   │   └── migrations/
│   ├── metrics/
│   ├── queue/
│   ├── scheduler/
│   ├── services/
│   ├── store/
│   ├── worker/
│   └── web/
│       └── static/
├── docs/
│   ├── API.md
│   ├── ARCHITECTURE.md
│   ├── DESIGN_DECISIONS.md
│   └── ER_DIAGRAM.md
├── docker-compose.yml
└── README.md
```

## Documentation

-   [Architecture](docs/ARCHITECTURE.md)
-   [Design decisions](docs/DESIGN_DECISIONS.md)
-   [API reference](docs/API.md)
-   [ER diagram](docs/ER_DIAGRAM.md)


## License

MIT. See [LICENSE](LICENSE).

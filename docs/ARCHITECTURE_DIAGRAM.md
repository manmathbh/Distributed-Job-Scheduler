# Architecture Diagram

```mermaid
flowchart TB
    subgraph Clients
        SDK["Go SDK<br/>(pkg/sdk)"]
        CURL["curl / REST"]
        WEB["Web Dashboard<br/>(embedded SPA)"]
    end

    subgraph "HTTP API (internal/api)"
        AUTH["Auth middleware<br/>Bearer API key"]
        LEGACY["Legacy endpoints<br/>/jobs, /admin/keys, /keys"]
        V1["/api/v1 endpoints<br/>projects, queues, jobs,<br/>schedules, DLQ, workers, metrics"]
        STATIC["Static handler<br/>/dashboard"]
    end

    subgraph Services
        SVC["service.Service<br/>(business rules)"]
        RETRY["retry<br/>(backoff math)"]
    end

    subgraph Store
        IFACE["store.Store interface"]
        PG["PostgresStore<br/>(authoritative)"]
        MEM["MemoryStore<br/>(tests/dev)"]
    end

    subgraph Runtime
        SCHED["Scheduler<br/>(promote/recover/stale/cron)"]
        WORKER["Worker<br/>(poll→claim→execute→heartbeat)"]
        METRICS["Metrics registry"]
    end

    subgraph Infra
        POSTGRES[("PostgreSQL")]
        REDIS[("Redis<br/>API keys")]
        WAL[("WAL<br/>legacy queue durability")]
    end

    SDK --> AUTH
    CURL --> AUTH
    WEB --> STATIC

    AUTH --> LEGACY
    AUTH --> V1
    V1 --> SVC
    SVC --> RETRY
    SVC --> IFACE

    IFACE --> PG
    IFACE --> MEM
    PG --> POSTGRES

    SCHED --> IFACE
    WORKER --> IFACE
    WORKER --> METRICS
    SCHED --> METRICS
    SVC --> METRICS

    LEGACY --> WAL
    AUTH --> REDIS

    WEB -. polling .-> V1
```

## Flow

1. Clients authenticate with `Authorization: Bearer <api-key>` (Redis-backed).
2. `/api/v1` requests hit `service.Service`, which enforces project/owner authorization and
   delegates persistence to the `Store`.
3. `PostgresStore` performs atomic claiming (`FOR UPDATE SKIP LOCKED`), pagination, and lifecycle
   transitions against PostgreSQL.
4. The `Scheduler` and `Worker` background loops drive due-job promotion, lease recovery, stale
   worker detection, cron firing, and job execution.
5. The dashboard polls `/api/v1` for live updates (no WebSockets required).

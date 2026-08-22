# ER Diagram

```mermaid
erDiagram
    users ||--o{ organization_members : "belongs"
    organizations ||--o{ organization_members : "has"
    organizations ||--o{ projects : "owns"
    projects ||--o{ queues : "has"
    projects ||--o{ jobs : "contains"
    projects ||--o{ scheduled_jobs : "contains"
    projects ||--o{ dead_letter_jobs : "contains"
    queues ||--o{ jobs : "queues"
    queues }o--o| retry_policies : "uses"
    jobs ||--o{ job_executions : "executed by"
    jobs ||--o{ job_logs : "logs"
    workers ||--o{ worker_heartbeats : "sends"
    jobs }o--o| workers : "claimed by"

    users {
        text id PK
        text email UK
        text name
        timestamptz created_at
        timestamptz updated_at
    }
    organizations {
        text id PK
        text name
        text slug UK
        timestamptz created_at
        timestamptz updated_at
    }
    organization_members {
        text id PK
        text organization_id FK
        text user_id FK
        text role
        timestamptz created_at
    }
    projects {
        text id PK
        text organization_id FK
        text owner_id
        text name
        text slug UK
        text description
        timestamptz created_at
        timestamptz updated_at
    }
    retry_policies {
        text id PK
        text strategy
        int max_attempts
        bigint initial_delay_ms
        bigint max_delay_ms
        float multiplier
        timestamptz created_at
    }
    queues {
        text id PK
        text project_id FK
        text name
        text description
        int priority
        int concurrency
        text status
        text retry_policy_id FK
        timestamptz created_at
        timestamptz updated_at
    }
    jobs {
        text id PK
        text project_id FK
        text queue_id FK
        text type
        jsonb payload
        int priority
        text status
        timestamptz scheduled_at
        timestamptz available_at
        int max_attempts
        text retry_strategy
        bigint retry_initial_delay_ms
        bigint retry_max_delay_ms
        float retry_multiplier
        int attempts
        text last_error
        text claim_token
        timestamptz lease_expires_at
        text worker_id
        timestamptz created_at
        timestamptz updated_at
        timestamptz claimed_at
        timestamptz started_at
        timestamptz completed_at
        timestamptz failed_at
    }
    job_executions {
        text id PK
        text job_id FK
        int attempt
        text worker_id
        text status
        timestamptz started_at
        timestamptz completed_at
        bigint duration_ms
        text error
        bool retryable
        jsonb metadata
    }
    workers {
        text id PK
        text project_id
        text hostname
        text status
        int concurrency
        timestamptz last_heartbeat
        timestamptz started_at
        timestamptz last_seen_at
        jsonb metadata
    }
    worker_heartbeats {
        text id PK
        text worker_id FK
        timestamptz sent_at
        int running
    }
    job_logs {
        text id PK
        text job_id FK
        int attempt
        text level
        text message
        timestamptz created_at
    }
    scheduled_jobs {
        text id PK
        text project_id FK
        text queue_id FK
        text name
        text cron_expr
        text timezone
        jsonb payload
        int priority
        bool enabled
        timestamptz next_run_at
        timestamptz last_run_at
        timestamptz created_at
        timestamptz updated_at
    }
    dead_letter_jobs {
        text id PK
        text job_id UK
        text project_id FK
        text queue_id FK
        jsonb payload
        text reason
        int attempts
        text worker_id
        timestamptz failed_at
        timestamptz created_at
        timestamptz requeued_at
    }
```

## Key relationships

- A **project** owns many **queues**; a queue owns many **jobs**.
- A **queue** references a **retry_policy**; jobs snapshot the policy at submission time so history
  is preserved if the queue's policy later changes.
- Each **job** has many **job_executions** (one per attempt) and many **job_logs**.
- A **worker** records many **worker_heartbeats**; jobs reference the worker that claimed them.
- **dead_letter_jobs** is keyed by `job_id` (one entry per permanently-failed job) and retains the
  payload, failure reason, attempt count, worker, and timestamps.

## Notes

- `ON DELETE CASCADE` is used for child-of-project tables (`queues`, `jobs`, `scheduled_jobs`,
  `dead_letter_jobs`) and child-of-job tables (`job_executions`, `job_logs`).
- `jobs.payload` is `jsonb`; timestamps are `timestamptz`; delays are stored as milliseconds
  (`bigint`) for deterministic retry math.

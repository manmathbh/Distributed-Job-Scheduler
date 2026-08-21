# WAL Job Queue

A high-performance, distributed job queue system written in Go, featuring Write-Ahead Logging (WAL) for durability, Redis-based caching, and comprehensive SDK support.

## Overview

WAL Job Queue is a production-ready task queue implementation designed for reliability and scalability. It uses Write-Ahead Logging to ensure zero data loss during crashes or system failures.

## Core Features

- ** Persistent Storage**: WAL-based persistence guarantees no job loss
- **⚡ High Performance**: Optimized for throughput with Redis caching
- ** Auto Recovery**: Seamless state restoration after crashes
- ** Secure**: Built-in authentication with scoped API keys
- ** Go SDK**: Production-ready client libraries
- ** Retry Logic**: Configurable retry mechanisms with exponential backoff
- ** Monitoring**: Job state tracking and metrics

## Architecture

The system consists of:
- **Queue Manager**: Handles job lifecycle and state transitions
- **WAL Engine**: Ensures durability through append-only logging
- **Worker Pool**: Processes jobs with lease-based concurrency control
- **Auth Service**: Manages API keys and permissions
- **REST API**: HTTP endpoints for job submission and monitoring

## Getting Started

### Using Docker (Recommended)

```bash
# Start all services
docker-compose up -d

# Verify deployment
curl http://localhost:8080/health
```

### Local Development

Requirements:
- Go 1.23 or higher
- Redis 6.0+

```bash
# Install dependencies
go mod download

# Run server
go run cmd/server/main.go
```

## API Usage

### Submit a Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "task.process",
    "payload": {"data": "your-data"},
    "priority": 1
  }'
```

### Query Job Status

```bash
curl http://localhost:8080/api/v1/jobs/{job-id} \
  -H "X-API-Key: your-api-key"
```

## SDK Documentation

See [Go SDK Guide](docs/GO-SDK.md) for detailed client and worker implementation examples.

## Configuration

Environment variables:
- `PORT`: HTTP server port (default: 8080)
- `REDIS_URL`: Redis connection string
- `WAL_DIR`: Write-Ahead Log directory path
- `MAX_RETRIES`: Maximum job retry attempts (default: 3)
- `LEASE_TIMEOUT`: Job lease duration in seconds (default: 300)

## Job States

1. **PENDING**: Job queued, waiting for worker
2. **PROCESSING**: Leased by worker
3. **COMPLETED**: Successfully finished
4. **FAILED**: Retryable failure
5. **DEAD**: Max retries exceeded

## Development

```bash
# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Build binary
go build -o queue-server cmd/server/main.go
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions welcome! Please open an issue first to discuss proposed changes.

## Author

Developed by Manmath Hatte

---

For more information, see the [documentation](docs/) directory.

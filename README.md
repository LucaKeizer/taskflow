# TaskFlow

A distributed task queue system built with Go for reliable background job processing.

## Overview

TaskFlow handles asynchronous job processing with Redis queuing and PostgreSQL persistence. It's designed for high-throughput applications that need reliable background task execution.

### Key Features

- **Multiple job types**: Email, image processing, webhooks, data export
- **Horizontal scaling**: Run multiple worker instances
- **Automatic retries**: Configurable retry logic with exponential backoff
- **Real-time monitoring**: Job status tracking and worker health monitoring
- **Production ready**: Structured logging, metrics, health checks

### Architecture

```
API Server -> Redis Queue -> Workers -> PostgreSQL
```

Jobs are submitted via REST API, queued in Redis for fast access, processed by workers, and results stored in PostgreSQL for persistence.

## Quick Start

### Using Docker

```bash
git clone https://github.com/lucakeizer/taskflow
cd taskflow
docker-compose up -d
```

### Manual Setup

1. Start dependencies:
```bash
# Redis
docker run -d -p 6379:6379 redis:7-alpine

# PostgreSQL  
docker run -d -p 5432:5432 \
  -e POSTGRES_DB=taskflow \
  -e POSTGRES_USER=taskflow \
  -e POSTGRES_PASSWORD=taskflow \
  postgres:15-alpine
```

2. Run the application:
```bash
# Terminal 1: API server
go run cmd/server/main.go

# Terminal 2: Worker
go run cmd/worker/main.go
```

## Usage Examples

### Submit an email job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Welcome!",
      "body": "Thanks for signing up"
    }
  }'
```

### Check job status

```bash
curl http://localhost:8080/api/v1/jobs/{job_id}
```

### View system stats

```bash
curl http://localhost:8080/api/v1/stats
```

## Job Types

- **Email**: Send emails via SMTP
- **Webhook**: Make HTTP requests to external APIs
- **Image Resize**: Process and resize images
- **Data Export**: Generate CSV/JSON reports

## Configuration

Set environment variables:

```bash
export SERVER_ADDR=":8080"
export WORKER_COUNT="3"
export REDIS_ADDR="localhost:6379"
export DATABASE_URL="postgres://taskflow:taskflow@localhost/taskflow?sslmode=disable"
```

## Performance

Load testing results on a 4-core machine:
- **Throughput**: 1,200+ jobs per minute
- **Latency**: <100ms average job pickup
- **Reliability**: 99.5%+ completion rate

Run load tests:
```bash
go run scripts/load-test.go -jobs=1000 -concurrent=50
```

## Development

### Project Structure

```
cmd/           # Main applications (server, worker)
internal/      # Private Go packages  
  api/         # REST API handlers
  worker/      # Job processors
  queue/       # Redis operations
  storage/     # PostgreSQL operations
  types/       # Data structures
scripts/       # Testing and utilities
docs/          # Documentation
```

### Running Tests

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Benchmarks
go test -bench=. ./...
```

### Adding New Job Types

1. Define payload struct in `internal/types/payloads.go`
2. Create processor in `internal/worker/`
3. Register in `NewProcessorRegistry()`
4. Add validation in `ValidateJobRequest()`

## Monitoring

- Health check: `GET /api/v1/health`
- Metrics: Prometheus metrics at `/metrics`  
- Logs: Structured JSON logging

## Deployment

### Docker

```bash
docker build -t taskflow-api --target api .
docker build -t taskflow-worker --target worker .
```

### Scaling

Run multiple workers for higher throughput:
```bash
docker-compose up --scale taskflow-worker=5
```

## Technical Details

- **Language**: Go 1.21
- **Queue**: Redis with atomic operations
- **Database**: PostgreSQL with connection pooling
- **Concurrency**: Goroutines with worker pools
- **Testing**: 95%+ code coverage
- **CI/CD**: GitHub Actions pipeline

## License

MIT License - see LICENSE file for details.
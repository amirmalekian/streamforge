# StreamForge

A concurrent media processing platform built with Go, demonstrating production-grade backend engineering practices.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  Gin API    │────▶│  RabbitMQ   │
└─────────────┘     └─────────────┘     └──────┬──────┘
                          │                    │
                          ▼                    ▼
                   ┌─────────────┐     ┌─────────────┐
                   │ PostgreSQL  │     │  Workers    │
                   │  (Jobs)     │     │  (Pool)     │
                   └─────────────┘     └──────┬──────┘
                          │                    │
                          ▼                    ▼
                   ┌─────────────────────────────┐
                   │         Redis               │
                   │  (Progress + Rate Limit)    │
                   └─────────────────────────────┘
```

## Tech Stack

- **Language**: Go 1.23
- **Framework**: Gin (HTTP)
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **Message Queue**: RabbitMQ 3.13
- **Auth**: JWT (HS256)
- **Real-time**: Server-Sent Events (SSE)
- **Config**: Viper
- **Migrations**: golang-migrate
- **Container**: Docker + Docker Compose

## Features

- **Concurrent Worker Pool**: Fixed-size worker pool with bounded channels, context cancellation, and graceful shutdown
- **Job Lifecycle**: CREATED → QUEUED → PROCESSING → COMPLETED/FAILED/CANCELLED
- **Real-time Progress**: SSE streaming with Redis pub/sub
- **Rate Limiting**: Per-IP rate limiting via Redis
- **Authentication**: JWT with bcrypt password hashing
- **Clean Architecture**: Modular monolith with clear domain boundaries

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.23+ (for local development)

### Using Docker Compose

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f api

# Stop services
docker compose down
```

The API will be available at `http://localhost:8080`

### Local Development

```bash
# Start dependencies only
docker compose up -d postgres redis rabbitmq

# Run migrations
go run github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path migrations -database "postgres://streamforge:streamforge@localhost:5432/streamforge?sslmode=disable" up

# Run API
go run ./cmd/api

# Run Worker (in separate terminal)
go run ./cmd/worker
```

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login and get JWT |

### Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/jobs` | Create new job |
| GET | `/api/v1/jobs` | List user's jobs |
| GET | `/api/v1/jobs/:id` | Get job details |
| DELETE | `/api/v1/jobs/:id` | Cancel job |
| GET | `/api/v1/jobs/:id/events` | SSE progress stream |
| GET | `/api/v1/jobs/:id/items` | List media items |

### Example Usage

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securepass123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securepass123"}'

# Create job (with token from login)
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"source_url":"https://example.com/playlist.m3u8"}'

# Stream progress
curl -N http://localhost:8080/api/v1/jobs/<job-id>/events \
  -H "Authorization: Bearer <token>"
```

## Project Structure

```
streamforge/
├── cmd/
│   ├── api/          # API server entry point
│   └── worker/       # Worker process entry point
├── internal/
│   ├── api/          # HTTP handlers, routes, middleware
│   ├── auth/         # Authentication service
│   ├── config/       # Configuration management
│   ├── database/     # PostgreSQL repository
│   ├── jobs/         # Job domain logic
│   ├── middleware/   # HTTP middleware
│   ├── queue/        # RabbitMQ client
│   ├── redis/        # Redis client + progress tracking
│   └── worker/       # Worker pool implementation
├── migrations/       # SQL migrations
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

## Configuration

Environment variables (with defaults):

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | development | Application environment |
| `PORT` | 8080 | HTTP server port |
| `POSTGRES_HOST` | localhost | PostgreSQL host |
| `POSTGRES_PORT` | 5432 | PostgreSQL port |
| `POSTGRES_USER` | streamforge | PostgreSQL user |
| `POSTGRES_PASSWORD` | streamforge | PostgreSQL password |
| `POSTGRES_DB` | streamforge | PostgreSQL database |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `RABBITMQ_HOST` | localhost | RabbitMQ host |
| `RABBITMQ_PORT` | 5672 | RabbitMQ port |
| `RABBITMQ_USER` | streamforge | RabbitMQ user |
| `RABBITMQ_PASSWORD` | streamforge | RabbitMQ password |
| `JWT_SECRET` | dev-secret | JWT signing secret |
| `JWT_EXPIRY` | 24h | JWT token expiry |
| `RATE_LIMIT` | 100 | Requests per minute per IP |
| `WORKER_COUNT` | 4 | Number of worker goroutines |

## Concurrency Model

- **Worker Pool**: Fixed number of workers (configurable, default: CPU count)
- **Task Queue**: Buffered channel (size = 2× workers) for backpressure
- **Context Propagation**: Cancellation signals flow through all layers
- **Graceful Shutdown**: Workers drain queue on SIGTERM (30s timeout)
- **Error Handling**: Failed tasks nacked with requeue, max retries via RabbitMQ

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires services)
go test -tags=integration ./...
```

## Roadmap

- [ ] Real media processing with ffmpeg/yt-dlp
- [ ] Webhook callbacks for job completion
- [ ] Prometheus metrics + Grafana dashboards
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Horizontal worker scaling
- [ ] Multi-part upload support
- [ ] Admin dashboard
# Go Microservice Boilerplate

Production-ready Go microservice template with Clean Architecture, designed for OpenShift/Kubernetes deployment.

## Stack

| Layer | Technology |
|---|---|
| Framework | Fiber v2 |
| ORM | GORM (PostgreSQL / MySQL) |
| Cache | Redis |
| Messaging | Kafka + RabbitMQ |
| Config | Viper (ENV-first) |
| Logger | zerolog |
| CLI | Cobra |
| HTTP Client | net/http + resilience wrapper |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                OpenShift / Kubernetes                    │
│         ConfigMap + Secrets → Viper (ENV priority)      │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │              Fiber v2 — HTTP Layer                │  │
│  │   recover · logger · CORS · request-id · /health  │  │
│  └──────────────────────┬────────────────────────────┘  │
│                         │                               │
│  ┌──────────────────────▼────────────────────────────┐  │
│  │         Handler  (delivery/http/handler)           │  │
│  │      bind request → call usecase → response        │  │
│  └──────────────────────┬────────────────────────────┘  │
│                         │                               │
│  ┌──────────────────────▼────────────────────────────┐  │
│  │              Usecase  (internal/usecase)           │  │
│  │        business logic · depends on interfaces      │  │
│  └────────────┬─────────────────────────┬────────────┘  │
│               │                         │               │
│  ┌────────────▼──────────┐  ┌───────────▼────────────┐  │
│  │  Repository Interface  │  │  ExternalSvc Interface │  │
│  │  (domain/repository)   │  │  (domain/service)      │  │
│  └────────────┬──────────┘  └───────────┬────────────┘  │
│               │                         │               │
│  ┌────────────▼──────────┐  ┌───────────▼────────────┐  │
│  │  Postgres / Redis impl │  │  HTTP Client impl      │  │
│  │  (internal/repository) │  │  (pkg/httpclient)      │  │
│  └────────────┬──────────┘  └───────────┬────────────┘  │
│               │                         │               │
│  ┌────────────▼──────────┐  ┌───────────▼────────────┐  │
│  │  PostgreSQL / MySQL   │  │  External Microservice  │  │
│  │  Redis cache          │  │  via HTTP/REST          │  │
│  └───────────────────────┘  └────────────────────────┘  │
│                                                         │
│  ┌──────────────────┐  ┌──────────────────────────────┐ │
│  │  Kafka           │  │  RabbitMQ                    │ │
│  │  producer+consumer│  │  publisher+consumer+DLQ     │ │
│  └──────────────────┘  └──────────────────────────────┘ │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │  zerolog · logs/ folder · daily rotation         │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## Clean Architecture Rules

```
Handler
  └── calls Usecase interface only
        └── calls Repository interface (persistence)
              └── impl: internal/repository/postgres/
                        internal/repository/redis/
        └── calls ExternalService interface (inter-service HTTP)
              └── impl: pkg/httpclient/
```

Each layer depends only on the interface above it. No concrete imports across layers. All wired in `cmd/api/main.go` via constructor injection.

---

## Inter-service HTTP Communication

```
Usecase
  └── ExternalUserService (interface in domain/service/)
        └── user_client.go (impl in pkg/httpclient/)
              └── base_client.go
                    ├── inject X-Request-ID header (tracing)
                    ├── inject Authorization header
                    └── middleware.go
                          ├── retry (max 3, exponential backoff)
                          ├── timeout (5s default, configurable)
                          └── circuit breaker (open after 5 failures)
```

External service URLs are injected via ENV (`USER_SERVICE_URL`, `ORDER_SERVICE_URL`), managed by OpenShift ConfigMap per environment.

---

## Request Tracing

Every request gets a unique `X-Request-ID` (UUID) injected by middleware. This ID propagates through:

- Response header `X-Request-ID`
- All log entries (zerolog context)
- Outgoing HTTP calls to other services (request header)
- Kafka message headers
- RabbitMQ message properties

---

## Project Structure

```
.
├── cmd/
│   ├── api/main.go          ← entrypoint, wire all deps
│   └── cli/main.go          ← CLI entrypoint
├── config/                  ← Viper config loader
├── internal/
│   ├── domain/
│   │   ├── entity/          ← GORM models
│   │   ├── repository/      ← repository interfaces
│   │   └── service/         ← external HTTP service interfaces
│   ├── usecase/             ← business logic
│   ├── delivery/http/       ← handler, middleware, route
│   └── repository/          ← postgres + redis implementations
├── pkg/
│   ├── logger/              ← zerolog setup + file rotation
│   ├── messaging/
│   │   ├── kafka/           ← producer + group consumer
│   │   └── rabbitmq/        ← publisher + consumer + DLQ
│   ├── httpclient/          ← base client + resilience + service impls
│   ├── response/            ← centralized HTTP response struct
│   └── validator/
├── cli/commands/            ← cobra generators
├── migrations/
├── logs/                    ← log files (gitignored)
├── Dockerfile               ← multi-stage build
├── docker-compose.yml
└── Makefile
```

---

## CLI Generator

```bash
# Build CLI first
make cli-build

# Then use it
./bin/cli make:all <name>      # generates all layers at once
./bin/cli make:model <name>    # entity only
./bin/cli make:repo <name>     # interface + postgres impl
./bin/cli make:usecase <name>  # interface + implementation
./bin/cli make:controller <name>
./bin/cli make:route <name>
./bin/cli make:client <name>   # external HTTP service interface + impl
```

---

## Environment Variables

All config is driven by ENV — no hardcoded values, required for OpenShift ConfigMap/Secret injection.

```env
# App
APP_ENV=development
APP_PORT=8080
APP_NAME=my-service

# Database (switchable: postgres | mysql)
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=user
DB_PASSWORD=password
DB_AUTO_MIGRATE=true

# Redis
REDIS_URL=redis://localhost:6379

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=my-service-group

# RabbitMQ
RABBITMQ_DSN=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=my-exchange

# Logger
LOG_LEVEL=info
LOG_TO_FILE=true
LOG_FILE_PATH=logs/app.log

# External services (managed per-env via ConfigMap)
USER_SERVICE_URL=http://user-service:8080
ORDER_SERVICE_URL=http://order-service:8080
```

---

## Quick Start

```bash
# 1. Start infrastructure
make docker-up

# 2. Copy env
cp .env.example .env

# 3. Run
make run

# 4. Health check
curl http://localhost:8080/health
```

---

## Makefile Commands

```bash
make run          # go run cmd/api/main.go
make build        # build binary to bin/api
make cli-build    # build CLI to bin/cli
make docker-up    # docker-compose up -d
make docker-down  # docker-compose down
make test         # go test ./...
make lint         # golangci-lint run
```

---

## Health Check

```
GET /health

200 OK — all services healthy
503 Service Unavailable — one or more services down

{
  "status": "ok",
  "services": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "rabbitmq": "ok"
  },
  "version": "1.0.0",
  "uptime": "2h30m"
}
```

---

## Graceful Shutdown

Handles `SIGTERM` (OpenShift pod termination) and `SIGINT`:

1. Stop accepting new HTTP requests
2. Wait max 30s for in-flight requests to complete
3. Close Kafka producer
4. Close RabbitMQ connection
5. Close Redis connection
6. Close DB connection pool

---

## Reference Implementation

The `user` domain is included as a complete reference:

- `internal/domain/entity/user.go`
- `internal/domain/repository/user_repository.go`
- `internal/usecase/user_usecase.go`
- `internal/delivery/http/handler/user_handler.go`
- `internal/delivery/http/route/user_route.go`
- `internal/repository/postgres/user_repository.go`
- `pkg/httpclient/user_client.go`

Use `make:all <name>` to generate the same structure for any new domain.

# Prompt: Go Microservice Boilerplate Init

Gunakan prompt ini dengan Claude Code CLI:
```
claude "$(cat prompt-go-microservice.md)"
```
Atau jalankan langsung dari direktori project yang sudah di-`git init`.

---

## Prompt

```
Initialize a production-ready Go microservice boilerplate. Generate ALL files with complete, working code. No placeholders, no TODOs.

---

## Stack

- Framework: Fiber v2
- ORM: GORM (PostgreSQL default, switchable to MySQL via DB_DRIVER env)
- Cache: Redis
- Messaging: Kafka + RabbitMQ (separate packages, independently usable)
- Config: Viper — ENV variable is highest priority (required for OpenShift/Kubernetes)
- Logger: zerolog (JSON in production, pretty console in development, file rotation)
- CLI generator: Cobra
- HTTP client: net/http with resilience wrapper (for inter-service communication)
- DI: manual constructor injection in main.go
- Architecture: Clean Architecture

---

## Project Structure

Generate this exact folder structure with all files:

```
.
├── cmd/
│   ├── api/
│   │   └── main.go                  ← app entrypoint, wire all dependencies
│   └── cli/
│       └── main.go                  ← CLI entrypoint (cobra root)
├── config/
│   ├── config.go                    ← Viper loader, struct mapping
│   ├── config.yaml                  ← default values (overridden by ENV)
│   └── service_urls.go              ← external service URL config
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   │   └── user.go              ← GORM model + domain entity
│   │   ├── repository/
│   │   │   └── user_repository.go   ← repository interface
│   │   └── service/
│   │       └── external_user.go     ← external HTTP service interface
│   ├── usecase/
│   │   └── user_usecase.go          ← interface + implementation
│   ├── delivery/
│   │   └── http/
│   │       ├── handler/
│   │       │   └── user_handler.go
│   │       ├── middleware/
│   │       │   ├── logger.go        ← request logging + request-id injection
│   │       │   ├── recover.go
│   │       │   └── cors.go
│   │       └── route/
│   │           ├── router.go        ← register all routes
│   │           └── user_route.go
│   └── repository/
│       ├── postgres/
│       │   └── user_repository.go   ← GORM postgres impl
│       └── redis/
│           └── cache.go             ← generic Redis cache helper with TTL
├── pkg/
│   ├── logger/
│   │   └── logger.go                ← zerolog setup, file rotation, levels
│   ├── messaging/
│   │   ├── kafka/
│   │   │   ├── producer.go          ← producer with context + reconnect
│   │   │   └── consumer.go          ← group consumer
│   │   └── rabbitmq/
│   │       ├── publisher.go         ← exchange declaration + publish
│   │       └── consumer.go          ← queue binding + DLQ support
│   ├── httpclient/
│   │   ├── base_client.go           ← net/http wrapper, header injection (request-id, auth)
│   │   ├── middleware.go            ← retry + timeout + circuit breaker
│   │   └── user_client.go           ← impl of external_user service interface
│   ├── response/
│   │   └── response.go              ← centralized HTTP response struct
│   └── validator/
│       └── validator.go             ← input validation helper
├── cli/
│   └── commands/
│       ├── root.go
│       ├── make_route.go
│       ├── make_controller.go
│       ├── make_model.go
│       ├── make_repo.go
│       ├── make_usecase.go
│       ├── make_client.go           ← generates interface + HTTP client impl
│       └── make_all.go              ← runs all generators in sequence
├── migrations/
│   └── .gitkeep
├── logs/
│   └── .gitkeep                     ← log files stored here
├── Dockerfile                       ← multi-stage: builder + minimal runtime
├── docker-compose.yml               ← postgres, mysql, redis, kafka, zookeeper, rabbitmq
├── Makefile
├── .env.example
├── .gitignore
└── go.mod
```

---

## Config (Viper)

Priority order (highest to lowest): ENV variable → config.yaml → hardcoded default

Required ENV variables — all must be readable from OpenShift ConfigMap and Secrets:

```
APP_ENV           = development | production
APP_PORT          = 8080
APP_NAME          = my-service

DB_DRIVER         = postgres | mysql
DB_HOST           = localhost
DB_PORT           = 5432
DB_NAME           = mydb
DB_USER           = user
DB_PASSWORD       = password
DB_MAX_OPEN_CONNS = 25
DB_MAX_IDLE_CONNS = 5
DB_CONN_MAX_LIFE  = 5m
DB_AUTO_MIGRATE   = true

REDIS_URL         = redis://localhost:6379
REDIS_PASSWORD    =
REDIS_DB          = 0

KAFKA_BROKERS     = localhost:9092
KAFKA_GROUP_ID    = my-service-group

RABBITMQ_DSN      = amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE = my-exchange

LOG_LEVEL         = info
LOG_TO_FILE       = true
LOG_FILE_PATH     = logs/app.log

USER_SERVICE_URL  = http://user-service:8080
ORDER_SERVICE_URL = http://order-service:8080
```

---

## Logger (zerolog)

- JSON output in production, pretty console in development (based on APP_ENV)
- Write to stdout always
- Write to file logs/app.log when LOG_TO_FILE=true, with daily rotation (lumberjack)
- Log levels: trace, debug, info, warn, error, fatal — controlled by LOG_LEVEL env
- Every log entry includes: timestamp, level, service name, request-id (from context)
- Request-id must be propagated: injected by middleware, stored in context, extracted in logger

---

## Database

- GORM with PostgreSQL as default driver
- Switchable to MySQL by setting DB_DRIVER=mysql — no code change required
- Auto-migrate on startup when DB_AUTO_MIGRATE=true
- Connection pool fully configurable via ENV
- Migrations folder for manual SQL files

---

## HTTP Layer (Fiber v2)

Middleware stack (in order):
1. recover — catch panics, return 500
2. request-id — generate UUID, set in context + response header X-Request-ID
3. logger — log method, path, status, latency, request-id
4. CORS — allow configurable origins

Endpoints:
- GET  /health         → return status of DB, Redis, Kafka, RabbitMQ connections
- GET  /users/:id
- POST /users
- PUT  /users/:id
- DELETE /users/:id

---

## Clean Architecture Rules

- Handler → calls Usecase interface only
- Usecase → calls Repository interface + ExternalService interface only
- Repository interface lives in internal/domain/repository/
- ExternalService interface lives in internal/domain/service/
- Implementation lives in internal/repository/ and pkg/httpclient/
- No direct import across layers — only through interfaces
- All dependencies injected via constructor in cmd/api/main.go

---

## Inter-service HTTP Client (pkg/httpclient)

base_client.go:
- Wraps net/http
- Injects X-Request-ID header from context on every outgoing request
- Injects Authorization header if token available in context
- Configurable timeout per client (default 5s, override via config)

middleware.go:
- Retry: max 3 attempts, exponential backoff, retry only on 5xx and network errors
- Timeout: context-based, respects caller's deadline
- Circuit breaker: open after 5 consecutive failures, half-open after 30s

user_client.go:
- Implements internal/domain/service/external_user.go interface
- Methods: GetUser(ctx, id) (*entity.User, error)
- URL read from config USER_SERVICE_URL

---

## Messaging

### Kafka (pkg/messaging/kafka)

producer.go:
- Produce message with context support
- JSON serialize payload
- Include request-id in message headers
- Auto-reconnect on connection failure

consumer.go:
- Group consumer with configurable group ID
- Handler func(ctx, message) error pattern
- Graceful shutdown on SIGTERM

### RabbitMQ (pkg/messaging/rabbitmq)

publisher.go:
- Declare exchange on init
- Publish with routing key
- Include request-id in message properties

consumer.go:
- Declare queue + binding on init
- Dead letter queue support (x-dead-letter-exchange header)
- Graceful shutdown

---

## CLI Generator (cli/commands using Cobra)

All commands accept a <name> argument in snake_case. Generated files use the name to populate struct names, interface names, and method stubs.

Commands:

```
go run cmd/cli/main.go make:route <name>
  → internal/delivery/http/route/<name>_route.go
  → registers CRUD routes for <name>

go run cmd/cli/main.go make:controller <name>
  → internal/delivery/http/handler/<name>_handler.go
  → handler struct with injected usecase interface

go run cmd/cli/main.go make:model <name>
  → internal/domain/entity/<name>.go
  → GORM model + CreatedAt, UpdatedAt, DeletedAt

go run cmd/cli/main.go make:repo <name>
  → internal/domain/repository/<name>_repository.go  (interface)
  → internal/repository/postgres/<name>_repository.go (GORM impl)

go run cmd/cli/main.go make:usecase <name>
  → internal/usecase/<name>_usecase.go
  → interface + struct + constructor + CRUD method stubs

go run cmd/cli/main.go make:client <name>
  → internal/domain/service/external_<name>.go  (interface)
  → pkg/httpclient/<name>_client.go              (HTTP impl)

go run cmd/cli/main.go make:all <name>
  → runs: make:model, make:repo, make:usecase, make:controller, make:route in sequence
  → prints summary of generated files
```

Each generated file must:
- Use the correct package name based on its folder
- Define proper interface in the domain layer
- Implement that interface in the infrastructure layer
- Follow the same pattern as the reference user implementation

---

## Graceful Shutdown

In cmd/api/main.go:
- Listen for SIGTERM and SIGINT (required for OpenShift pod termination)
- On signal: stop accepting new requests, wait max 30s for in-flight requests
- Close DB connection, Redis, Kafka producer, RabbitMQ connection in order
- Log each shutdown step

---

## Docker

Dockerfile (multi-stage):
- Stage 1 (builder): golang:1.22-alpine, build binary with CGO_ENABLED=0
- Stage 2 (runtime): gcr.io/distroless/static or alpine:3.19, copy binary only
- EXPOSE 8080
- CMD ["/app"]
- No hardcoded config — all via ENV

docker-compose.yml services:
- postgres:16-alpine
- mysql:8
- redis:7-alpine
- zookeeper:3.8 (required by kafka)
- kafka:3.5 (bitnami/kafka)
- rabbitmq:3.12-management

---

## Makefile

```makefile
run:         go run cmd/api/main.go
build:       go build -o bin/api cmd/api/main.go
cli-build:   go build -o bin/cli cmd/cli/main.go
migrate:     go run cmd/api/main.go migrate
docker-up:   docker-compose up -d
docker-down: docker-compose down
test:        go test ./...
lint:        golangci-lint run
```

---

## Health Check Response

GET /health must return:

```json
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

Return HTTP 200 if all ok, HTTP 503 if any service is down.

---

## Reference Implementation: User Domain

Generate complete working code for the user domain as the reference:

Entity (internal/domain/entity/user.go):
- ID (uuid), Name, Email, CreatedAt, UpdatedAt, DeletedAt

Repository interface: FindByID, FindAll, Create, Update, Delete

Usecase: GetUser, ListUsers, CreateUser, UpdateUser, DeleteUser

Handler: GET /users/:id, GET /users, POST /users, PUT /users/:id, DELETE /users/:id

All layers wired in cmd/api/main.go.

---

Generate all files now. Start with go.mod, then config, then pkg, then internal, then cmd, then cli, then Dockerfile and docker-compose. Print each file path before its content.
```

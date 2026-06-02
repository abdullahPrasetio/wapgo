# Changelog

All notable changes to wapgo are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

---

## [0.10.0] — 2026-06-02

### Added
- `pkg/observability` — **OTel → Elastic APM bridge**: when `OBSERVABILITY_PROVIDER=elastic_apm`, an
  `apmotel` TracerProvider is installed as the global OTel provider, so spans created via
  `otel.Tracer(...)` (including in generated usecases) are forwarded to Elastic APM and nested under the
  apmfiber transaction. Previously these spans were silently dropped under the Elastic backend.
- `pkg/observability.StartSpan(ctx, name) (context.Context, func())` — provider-agnostic helper for
  manual spans; works for OTel, Elastic APM (via the bridge), and `none` (no-op).
- CLI skeleton now vendors `pkg/observability` and `pkg/auth` as source, and `go.mod.tmpl` gained the
  OpenTelemetry, Elastic APM (`apm/v2`, `apmfiber`, `apmhttp`, `apmotel`), Prometheus, `redisotel`,
  `otelgorm`, and `golang-jwt/jwt/v5` dependencies — **generated projects now compile out of the box**
  (`go mod tidy && go build ./...`) for every `--apm` choice.

### Fixed
- `pkg/observability` setup — `OBSERVABILITY_PROVIDER=none` now returns the no-op provider instead of
  falling through to the OTel branch, so tracing is genuinely disabled.
- Generated usecases — the package-level OTel `tracer` var is now named per-domain
  (`<domain>Tracer`), so multiple usecases generated into the same package
  (e.g. `make:all user` + `make:all product`) no longer fail to compile with `tracer redeclared`.
- CLI `make:client` template — documented how to wire `Options.TransportWrapper = obsProvider.WrapTransport`
  so outgoing third-party calls are recorded as APM/OTel child spans.

---

## [0.9.0] — 2026-06-02

### Added
- CLI `wapgo new` — interactive project wizard (powered by `charmbracelet/huh`): prompts for project name, module path, database (PostgreSQL/MySQL), observability provider (Elastic APM / OpenTelemetry / None), and optional features (Redis, Kafka, RabbitMQ) via a multi-select. Falls back to flag/default-driven non-interactive mode with `--yes` (or automatically when stdin is not a TTY) for CI use.
- CLI `wapgo add <feature>` — add an optional capability (`redis`, `kafka`, `rabbitmq`) to an existing project after scaffolding; copies the feature's source files, never overwrites existing files (idempotent), and prints the manual wiring steps for `cmd/api/main.go`, `.env`, and `docker-compose.yml`.
- Conditional scaffolding — `wapgo new` now generates only the files for selected features. Skeleton files for disabled features (Redis cache, Kafka, RabbitMQ) are omitted, and `cmd/api/main.go`, `docker-compose.yml`, and `.env.example` are rendered with feature-aware conditionals. Only the chosen database service appears in `docker-compose.yml`, and `DB_PORT` defaults to the right port (3306 for MySQL, 5432 for PostgreSQL).
- Styled CLI output — `wapgo new` / `wapgo add` print a colored summary and next-steps block via `charmbracelet/lipgloss`.

### Changed
- Scaffold now strips the `//go:build ignore` guard from generated Go files (the guard exists only to keep skeleton sources out of the CLI module build).
- Skeleton `internal/delivery/http/handler/health_handler.go` — refactored to be Redis-agnostic: the Redis ping is registered as a regular `AddChecker("redis", …)` from `main.go` only when Redis is enabled, instead of being a hard-coded dependency of the handler.
- `wapgo new` flags: added `--apm`, `--redis`, `--kafka`, `--rabbitmq`, `--yes/-y`; `--db` no longer defaults eagerly (the wizard/`--yes` path applies the default).

---

## [0.8.0] — 2026-06-02

### Added
- CLI `make:migration <name>` — generate a timestamped up/down SQL migration file pair in `migrations/` following the golang-migrate convention (`{timestamp}_{name}.up.sql` / `.down.sql`). Supports snake_case, PascalCase, and kebab-case input; generates a GORM-compatible CREATE TABLE skeleton (UUID PK, soft-delete column, index).
- `pkg/response` — `Paginated()` helper and `PaginatedResponse` / `PageMeta` types for list endpoints; `total_pages` computed automatically from `total` and `per_page`; zero-safe (perPage=0 → totalPages=0).
- CLI skeleton (`wapgo new`) — generates `README.md` with project name, module path, quick-start, make targets, project structure, and configuration reference; `DB` driver value injected from `--db` flag.

---

## [0.7.0] — 2026-06-02

### Added
- `pkg/response` — expanded test coverage: status code, message body, nil-data omission
- `internal/delivery/http/middleware` — unit tests for SecurityHeaders, CORS, RateLimiter, RequestID, Recover (11 cases)
- `internal/repository/db` — unit tests with go-sqlmock: FindByID, FindAll, Create, Update, Delete, ExistsByEmail (11 cases)
- `pkg/database` — unit tests: buildDialector (mysql/postgres/unsupported), configurePool (all fields/invalid duration), NewConnection error path (9 cases)
- `internal/integration/mysql_test.go` — integration test for MySQL via testcontainers (Create, FindByID, FindAll, ExistsByEmail, Update, Delete)
- `CHANGELOG.md` — this file, following Keep a Changelog format
- `HIGHLIGHTS.md` — framework strengths and comparison table vs generic boilerplate
- `.github/workflows/release.yml` — GitHub Actions release workflow: builds CLI for linux/darwin × amd64/arm64, uploads tarballs + checksums to GitHub Releases

### Changed
- `internal/repository/postgres/` → `internal/repository/db/` — renamed to driver-agnostic name; GORM query code is identical for MySQL and Postgres, driver selection remains in `pkg/database` via `DB_DRIVER` env var
- `config/config.go` — default `DB_DRIVER` changed `postgres` → `mysql`; default `DB_PORT` changed `5432` → `3306`; default `OBSERVABILITY_PROVIDER` changed `otel` → `elastic_apm`
- CLI `make:repo` — output path updated to `internal/repository/db/`, template renamed to `repository.go.tmpl`
- `internal/domain/repository/doc.go` — dependency diagram updated to reflect `repository/db`

---

## [0.7.0-beta.1] — 2026-06-02

### Added
- `pkg/response` — expanded test coverage: status code, message body, nil-data omission
- `internal/delivery/http/middleware` — unit tests for SecurityHeaders, CORS, RateLimiter, RequestID, Recover (11 cases)
- `internal/repository/db` — unit tests with go-sqlmock: FindByID, FindAll, Create, Update, Delete, ExistsByEmail (11 cases)
- `CHANGELOG.md` — this file, following Keep a Changelog format
- `HIGHLIGHTS.md` — framework strengths and comparison table vs generic boilerplate
- `.github/workflows/release.yml` — GitHub Actions release workflow: builds CLI for linux/darwin × amd64/arm64, uploads tarballs + checksums to GitHub Releases

### Changed
- `internal/repository/postgres/` → `internal/repository/db/` — renamed to driver-agnostic name; GORM query code is identical for MySQL and Postgres, driver selection remains in `pkg/database` via `DB_DRIVER` env var
- `config/config.go` — default `DB_DRIVER` changed `postgres` → `mysql`; default `DB_PORT` changed `5432` → `3306`; default `OBSERVABILITY_PROVIDER` changed `otel` → `elastic_apm`
- CLI `make:repo` — output path updated to `internal/repository/db/`, template renamed to `repository.go.tmpl`
- `internal/domain/repository/doc.go` — dependency diagram updated to reflect `repository/db`

---

## [0.6.0] — 2026-06-02

### Added
- `ARCHITECTURE.md` — architecture reference document: layer diagram, two `repository` folder explanation, interface/impl pattern, DI wiring, mock pattern, import rules
- `HIGHLIGHTS.md` — technical strengths of the framework
- `SECURITY.md` — security policy and vulnerability reporting process
- `CONTRIBUTING.md` — contributing guide
- `install.sh` — one-line installer for the CLI binary (auto-detect OS/arch, GitHub Releases)
- `examples/` — runnable examples for JWT, HTTP client, messaging (Kafka + RabbitMQ)
- `kubernetes/` — production-ready Kubernetes manifests (Deployment, Service, ConfigMap, Secret, NetworkPolicy)
- `doc.go` — package documentation for `domain/repository`, `repository/postgres`, `repository/redis`, `usecase`

### Changed
- `internal/domain/repository/cache.go` — moved `Cacher` interface here from redis package (Clean Architecture: interface belongs in domain, not in implementation)
- `internal/repository/redis/cache.go` — removed `Cacher` interface, implementation now references `repository.Cacher`
- `internal/repository/postgres/user_repository.go` — added inline comments explaining unexported struct and interface-returning constructor
- `GUIDE.md` — fixed Kafka and RabbitMQ API examples to match actual function signatures; added links to ARCHITECTURE.md
- `README.md` — added documentation nav links
- CLI templates — updated to match Cacher interface relocation

### Removed
- Swaggo annotations from `user_handler.go` (swagger feature cancelled)

---

## [0.5.0] — 2025-XX-XX

### Added
- `pkg/auth` — JWT middleware with `GenerateToken`, `ValidateToken`, `ExtractClaims`
- `pkg/observability` — pluggable observability provider:
  - OpenTelemetry (OTLP HTTP exporter)
  - Elastic APM
  - Prometheus RED metrics middleware (`MetricsMiddleware`, `MetricsHandler`)
  - GORM query instrumentation (`InstrumentGORM`)
  - Redis command instrumentation (`InstrumentRedis`)
- `config/` — `OBSERVABILITY_PROVIDER` env var to switch provider without code changes
- Graceful shutdown sequence: HTTP server → observability provider → Redis → DB

---

## [0.4.0] — 2025-XX-XX

### Added
- CLI multi-module monorepo: `cli/` is a separate Go module linked via `go.work`
- `wapgo new <name>` — scaffold a new service from embedded skeleton template
- `wapgo make:model`, `make:repo`, `make:usecase`, `make:controller`, `make:route`, `make:client`, `make:all` — code generators for each layer
- `.github/workflows/ci.yml` — CI pipeline: lint, SAST (gosec), CVE (govulncheck), secret scan (gitleaks), test + coverage gate (≥80%), integration tests, build, Docker build + Trivy scan

---

## [0.3.0] — 2025-XX-XX

### Added
- `pkg/httpclient` — resilient inter-service HTTP client:
  - Circuit breaker (gobreaker)
  - Retry with exponential backoff
  - Timeout per request
  - Request/response logging middleware
- `internal/domain/service/external_user.go` — example domain service using the HTTP client
- `config/service_urls.go` — typed service URL config

---

## [0.2.0] — 2025-XX-XX

### Added
- `internal/repository/redis` — Redis cache implementation (`RedisCacher`) with `Set`, `Get`, `Del`, `Exists`; `ErrCacheMiss` sentinel
- `pkg/messaging/kafka` — Kafka producer/consumer with `Publish`, `Start` (consumer loop), `HealthCheck`
- `pkg/messaging/rabbitmq` — RabbitMQ publisher/consumer with `Publish`, `Subscribe`, `HealthCheck`
- `internal/delivery/http/handler/health_handler.go` — `/health` endpoint with pluggable checkers (DB, Redis, Kafka, RabbitMQ)
- `docker-compose.yml` — local infrastructure: Postgres, Redis, Kafka, RabbitMQ

---

## [0.1.0] — 2025-XX-XX

### Added
- Core skeleton: Fiber v2, GORM, Postgres driver, Zerolog logger
- Clean Architecture layers: `internal/domain`, `internal/repository`, `internal/usecase`, `internal/delivery/http`
- `entity.User` with UUID primary key, soft delete, `BeforeCreate` hook
- `UserRepository` interface + Postgres implementation (CRUD + `ExistsByEmail`)
- `UserUseCase` interface + implementation with domain error types (`ErrNotFound`, `ErrEmailConflict`, `ErrInvalidUUID`)
- `UserHandler` with `mapError` — maps domain errors to HTTP status codes
- Middleware stack: `SecurityHeaders` (helmet), `RateLimiter`, `CORS`, `Recover`, `RequestID`, `RequestLogger`
- `pkg/validator` — go-playground/validator wrapper with JSON field names in error messages
- `pkg/response` — typed response helpers (`Success`, `Created`, `Error`, `BadRequest`, `NotFound`, `InternalError`, `Unauthorized`)
- `pkg/logger` — zerolog setup with file rotation (lumberjack), request-ID context injection
- `config/` — ENV-first config via Viper, fully compatible with Kubernetes ConfigMap + Secret
- `Makefile` — `run`, `build`, `test`, `docker-up`, `docker-down`, `migrate`, `lint` targets
- `Dockerfile` — multi-stage build with distroless final image

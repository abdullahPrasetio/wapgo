# Changelog

All notable changes to wapgo are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [1.7.0] — 2026-06-15

### Added
- **`wapgo add google-auth`** — Google OAuth2 login/register add-on. Scaffolds three subtrees:
  `pkg/auth/google` (Provider, AuthURL, Exchange, UserInfo), handler (`GoogleAuthHandler` with
  Login + Callback), and route (`RegisterGoogleAuthRoutes`). Requires `golang.org/x/oauth2`.
- **`pkg/auth/google`** — framework-level OAuth2 helper: `New(Config)`, `AuthURL(state)`,
  `Exchange(ctx, code)` fetches Google userinfo after token exchange.
- **`FindByEmail`** added to skeleton `UserRepository` interface and GORM implementation.
  Returns `gorm.ErrRecordNotFound` (not wrapped) for not-found so callers can sentinel-check.
- **Kafka `ConsumerConfig` struct** — `NewConsumer` signature changed from
  `(brokers, groupID, topic string, log)` to `(cfg ConsumerConfig, topic string, log)`.
  New fields: `HeartbeatInterval`, `SessionTimeout`, `RebalanceTimeout time.Duration`
  (zero value falls back to the previous hardcoded defaults: 3s / 30s / 30s).
  Three new ENV vars: `KAFKA_HEARTBEAT_INTERVAL`, `KAFKA_SESSION_TIMEOUT`,
  `KAFKA_REBALANCE_TIMEOUT` — with defaults set in both `config.go` and skeleton
  `config.go.tmpl`. `worker_main.go.tmpl` updated to `ConsumerConfig{}` struct literal.

### Fixed
- **`upsertUser` error swallow** — `GoogleAuthHandler.upsertUser` now distinguishes
  `gorm.ErrRecordNotFound` from other DB errors via `errors.Is`. A DB timeout or network
  error no longer silently falls through to `Create`, preventing ghost-account creation
  under partial outage.
- **`FindByEmail`/`ExistsByEmail` case-sensitivity** — skeleton DB implementation now uses
  `LOWER(email) = LOWER(?)` instead of exact-match, preventing duplicate accounts when an
  existing user registered with mixed-case email (e.g. `User@Gmail.com`) logs in via Google
  (which always returns lowercase email).
- **Google-auth JWT missing roles** — `auth.Sign` now passes `[]string{"user"}` (was `nil`).
  Users who logged in via Google were rejected by any role-protected route even though
  their token was valid.
- **`io.LimitReader` missing in `Exchange()`** — `Provider.Exchange` now wraps
  `resp.Body` with `io.LimitReader(resp.Body, 4*1024)` before decoding, consistent with
  the pattern already used in `pkg/notification/firebase`.
- **`worker_main.go.tmpl` missing `"time"` import** — when `.Kafka = true`, the generated
  worker used `time.ParseDuration` without importing `"time"`. Fixed by adding a conditional
  `"time"` import guarded by `[[- if .Kafka ]]`.
- **Skeleton `config.go.tmpl` missing Kafka timeout fields** — framework `config.go` was
  updated (3 fields + ENV bindings + defaults) but the skeleton template was not, causing
  `wapgo new` to generate a config that didn't expose the new Kafka ENV vars.

### Changed (Breaking)
- **`kafka.NewConsumer` signature** — callers must switch to `ConsumerConfig{}` struct literal.
  Zero-valued duration fields fall back to the previous defaults so no behavioural change
  if the new fields are left at zero.

---

## [Unreleased]

### Added
- **`wapgo make:worker [name] [--broker kafka|rabbitmq|both]`** — new CLI command that generates
  a standalone worker binary entrypoint. Without a name: `cmd/worker/main.go`. With a name:
  `cmd/worker-{name}/main.go`. The `--broker` flag selects which messaging integration to scaffold;
  omitting it auto-detects from `go.mod` (checks for `segmentio/kafka-go` and `rabbitmq/amqp091-go`).
  Two Makefile targets (`run-{suffix}` and `build-{suffix}`) are appended automatically when a
  `Makefile` is detected. Generated worker uses the shared `*Connection` pattern for RabbitMQ
  so a single TCP socket serves all consumers.
- **RabbitMQ — shared connection type** (`pkg/messaging/rabbitmq.Connection`): a single
  `*Connection` is created once per process and passed to all `Publisher` and `Consumer` instances.
  Eliminates the connection-explosion problem where N consumers/publishers previously created N TCP
  sockets to the broker, crashing RabbitMQ under load.
  - `NewConnection(dsn, log)` — dials with 5 s timeout + 10 s heartbeat; double-checked locking
    for reconnect to prevent concurrent races.
  - `Connection.Channel()` — opens a new AMQP channel; auto-reconnects the underlying connection
    transparently on channel failure.
  - `Connection.Close()` — shuts down the shared TCP connection; call once during shutdown.
- **RabbitMQ — blocking `Subscribe` with auto-reconnect**: `Consumer.Subscribe(ctx, queue, rk, handler)`
  now blocks until `ctx` is cancelled, reconnecting the channel automatically with exponential
  backoff (1 s → 30 s cap) when the broker restarts. Channel death is detected via `amqp.NotifyClose`.
  > ⚠️ **BREAKING CHANGE** — `Subscribe` signature changed: added `ctx context.Context` as first
  > parameter, and the method now **blocks** (previously returned immediately and spun an internal
  > goroutine). Callers must wrap in a goroutine: `go consumer.Subscribe(ctx, queue, rk, handler)`.
  > The `cmd/worker-*/main.go` scaffold and all skeleton templates already use this pattern.
- **RabbitMQ — publisher channel retry**: `Publisher.Publish` reopens its channel once on error
  before returning, so a broker restart is transparent to callers in most cases.
- **Kafka — exponential backoff on fetch error**: `Consumer.Start` now waits 1 s before the first
  retry after a fetch failure, doubling each attempt up to 30 s, using a `select` on `ctx.Done`
  so shutdown is still immediate. Backoff resets to 1 s on the next successful fetch.
- **Kafka — session configuration**: `HeartbeatInterval: 3s`, `SessionTimeout: 30s`,
  `RebalanceTimeout: 30s`, and `ErrorLogger` (surfaces internal reader errors via zerolog) added
  to `NewConsumer`'s `ReaderConfig`.
- **Redis — connection pool configuration**: six new fields in `RedisConfig` with explicit
  ENV bindings and sane defaults:
  `REDIS_POOL_SIZE` (20), `REDIS_MIN_IDLE_CONNS` (5), `REDIS_DIAL_TIMEOUT` (5s),
  `REDIS_READ_TIMEOUT` (3s), `REDIS_WRITE_TIMEOUT` (3s), `REDIS_MAX_RETRIES` (3).
  `REDIS_URL` replaces the old `REDIS_HOST`/`REDIS_PORT` pair (URL format: `redis://host:port`).

### Fixed
- **Skeleton IDE false positives** — all 44 plain `.go` skeleton files under
  `cli/generator/templates/skeleton/` renamed to `.go.tmpl`. The `//go:build ignore` guard
  (needed to exclude skeleton sources from the CLI build) caused gopls to analyze each file
  independently, making cross-file references (e.g. `Connection` in `connection.go` referenced
  from `consumer.go`) appear as "undefined" in the IDE. With `.go.tmpl` extension gopls skips
  these files entirely; the generated user project is unaffected.
  `scaffold.go` updated: template files are now rendered to a `strings.Builder` first so the
  module-path placeholder (`github.com/abdullahPrasetio/wapgo`) is replaced after rendering,
  keeping `AddFeatureFiles` consistent.

---

## [1.2.0] — 2026-06-12

### Added
- **`wapgo upgrade`** — new CLI command to check for a newer release and self-update via
  `go install`. Flags: `--check` (report only, no install). Offline-safe: network errors
  are non-fatal warnings.

### Fixed
- **`wapgo new` scaffold** — `go mod tidy` no longer fails after scaffolding. Root cause:
  `cmd/api/main.go.tmpl` imported `[[.Module]]/docs` (Swagger generated package) which
  does not exist until the user runs `swag init`. The import has been removed.
  `router.go` template also had a `github.com/gofiber/swagger` import that is not included
  in the generated `go.mod`; the swagger route and its dependency have been removed from
  the skeleton (users can add it manually via `wapgo add` or by generating docs separately).

---

## [1.4.0] — 2026-06-12

### Security

- **Token confusion eliminated** — `Claims` struct gains `TokenType string` field (`"access"` | `"refresh"`).
  `auth.Sign()` signature changed to `Sign(subject, roles, tokenType, cfg) (token, jti string, err error)` — JTI
  returned directly, no round-trip `Verify` needed. `Middleware` now rejects any token with `token_type != "access"`,
  preventing refresh tokens from being used as Bearer access tokens.
- **Reset token env-gated** — `AuthHandler` gains an `env string` field; `reset_token` is only included in
  `ForgotPassword` responses when `APP_ENV != "production"`. Prevents account takeover via leaked token in
  production API responses.
- **Logout blacklists refresh JTI** — `Logout` now revokes both the access token JTI and the refresh token JTI in
  the Redis blacklist. Previously only the access JTI was revoked.
- **Session versioning on password reset** — `ResetPassword` increments a per-user version counter
  (`auth:ver:{userID}`) in Redis. `issueTokenPair` embeds the current version in the stored `refreshSession`.
  `Refresh` rejects any session whose version is lower than the current counter, invalidating all pre-reset sessions.
- **bcrypt minimum floor** — `NewAuthUseCase` now enforces `bcryptCost >= 10` (was `>= bcrypt.MinCost` = 4),
  consistent with `APP_BCRYPT_COST` documentation and `main.go` enforcement.

### Reliability

- **Redis error logged before degradation** — `RedisCacher.Get` emits a `zerolog` `Warn` with the raw error and
  key before returning `ErrCacheMiss`. Operators can now distinguish Redis outage from normal cache misses.

### Changed

- `auth.Sign` — signature is now `(subject string, roles []string, tokenType string, cfg *Config) (string, string, error)`.
  All callers (`examples/jwt`, `examples/auth`, `examples/shop`, benchmarks, tests) updated.
- `Refresh` — validates user still exists in DB (`userRepo.FindByID`) after session verification; returns
  `ErrInvalidToken` for soft-deleted or hard-deleted accounts.
- `mockCacher` in `auth_usecase_test.go` rewritten to use JSON marshal/unmarshal, matching real Redis behaviour
  and supporting struct-typed values.

### Tests added

- `TestSign_ReturnsJTI` — verifies JTI in returned string matches the `ID` claim in the parsed token.
- `TestMiddleware_RefreshTokenRejected` — verifies middleware returns 401 for a token signed with `"refresh"` type.
- `TestAuthUseCase_Refresh_InvalidAfterPasswordReset` — end-to-end: login → reset password → pre-reset refresh
  token rejected.
- `TestAuthUseCase_Logout_BlacklistsRefreshJTI` — verifies both access and refresh JTIs are in the blacklist
  after logout.

---

## [1.3.0] — 2026-06-12

### Added

- **`pkg/database/timeout.go`** — `QueryTimeoutPlugin` GORM plugin; enforces per-query deadline on all
  CRUD + raw operations. Configurable via `DB_QUERY_TIMEOUT` (default `5s`).
- **`mw.WithBodyLimit(maxBytes)`** — per-route semantic body size limit in
  `internal/delivery/http/middleware/security.go`; enforces a tighter limit post-Fiber-buffer on
  individual endpoints (e.g. login, password reset).
- **Graceful Redis degradation** — `RedisCacher.Get` returns `ErrCacheMiss` instead of a hard error when
  Redis is unavailable, allowing callers to fall back to DB without crashing.
- **bcrypt cost configurable** — `BCRYPT_COST` ENV → `cfg.App.BcryptCost`; minimum `10`, default `12`;
  passed into `NewAuthUseCase` to control hashing work factor.

---

## [1.2.0] — 2026-06-12

### Added

- **`pkg/auth/blacklist.go`** — `Blacklist` interface + `RedisBlacklist` implementation (JTI revocation via
  Redis SET with TTL). `auth.Middleware` accepts an optional `Blacklist` variadic argument; checks JTI on
  every request when provided.
- **Full authentication endpoints** wired via `internal/usecase/auth_usecase.go` and
  `internal/delivery/http/handler/auth_handler.go`:
  - `POST /api/v1/auth/login` — bcrypt verify, issue access + refresh JWT, store refresh session in Redis.
  - `POST /api/v1/auth/refresh` — verify + rotate refresh session (token rotation).
  - `POST /api/v1/auth/logout` — revoke access token JTI, delete refresh session.
  - `POST /api/v1/auth/forgot-password` — generate single-use reset token (Redis TTL 15 min).
  - `POST /api/v1/auth/reset-password` — consume reset token, update password hash.
- **`pkg/auditlog/`** — `Logger`, `Entry`, `Action` constants; context-propagated via
  `WithAuditLogger` / `FromContext`.
- **`pkg/crypto/`** — `Encryptor` with AES-256-GCM; non-deterministic (fresh nonce per call).
- **JWT middleware per-route** — `auth.Middleware(jwtCfg, bl)` attached to all `/users` routes via
  `RegisterUserRoutes`; `auth.RequireRole("admin")` on `DELETE /users/:id`.
- **Trusted proxy config** — `APP_TRUSTED_PROXIES` (comma-separated) wired to
  `fiber.Config.TrustedProxies + EnableTrustedProxyCheck`.

### Security

- `Password` field on `entity.User` tagged `json:"-"` — never serialized to API responses.

---

## [1.1.0] — 2026-06-03

### Added
- **Swagger/OpenAPI docs** (`/docs`) — `github.com/gofiber/swagger` + `swaggo/swag`; full `@Summary`,
  `@Router`, `@Param`, `@Success`, `@Failure` annotations on all user and health handlers; `docs/`
  directory generated via `swag init -g cmd/api/main.go --parseDependency --parseInternal`.
  Route `/docs/*` is guarded by `prodGuard` — returns 404 in production, Swagger UI in all other envs.
- **Welcome page** (`GET /`) — lightweight JSON landing: `{service, version, env, links}`.
  In production `links` only exposes `/health`; in dev/staging also exposes `/docs` and `/metrics`.
- **`wapgo make:test`** — new CLI generator with two layers:
  - `--layer usecase` (default) — usecase unit tests with a hand-written `mock.Mock` repository;
    covers `Get*_OK`, `Get*_NotFound`, `Get*_InvalidUUID`, `List*_OK`, `Create*_OK`, `Delete*_OK`.
  - `--layer handler` — HTTP handler tests using a struct-based mock usecase + `httptest`;
    covers all five CRUD operations, invalid body, validation failure, and each error status code.
  - Both layers are generated automatically by `wapgo make:all <name>`.
- CLI skeleton `internal/delivery/http/route/router.go` updated: includes `GET /`, `GET /docs/*`
  (prodGuard), and `github.com/gofiber/swagger` import; `cmd/api/main.go.tmpl` imports `docs` package.

---

## [0.11.1] — 2026-06-03

### Fixed
- `pkg/logger` — data race on global sink variables: concurrent calls to `SetupSinks` (writes) and
  `API()`/`Consumer()`/`ThirdParty()`/`Trace()` (reads) caused a detected race under `-race` on Linux
  CI. Fixed by replacing plain `zerolog.Logger` vars with `atomic.Pointer[zerolog.Logger]`-backed
  `sinkHolder`, making all reads and writes lock-free and race-safe.
- CI — `golangci-lint-action@v6` rejects version strings of the form `v2.x.x` (golangci-lint v2 not
  supported); upgraded to `golangci-lint-action@v7`.
- CI — Go toolchain upgraded from `1.25.8` to `1.25.11`; Go 1.25.0–1.25.10 carry 28+ active stdlib
  CVEs that `govulncheck` flags; all are resolved in 1.25.11.
- CI — test command now excludes packages without test files (`examples/`, `cmd/`) to prevent the
  `go: no such tool "covdata"` error on downloaded toolchains.

---

## [0.11.0] — 2026-06-03

### Added
- `pkg/logger` — **four structured log sinks** written to `logs/`: `api.log`, `consumer.log`,
  `thirdparty.log`, `trace.log`. `SetupSinks(SinkConfig)` plus `API()` / `Consumer()` / `ThirdParty()` /
  `Trace()` accessors. Rotation is selectable via `LOG_ROTATION`: `size` (lumberjack) or `daily`
  (date-stamped `api-2006-01-02.log`, midnight rollover, `LOG_MAX_AGE_DAYS` retention).
- `pkg/journal` — request/message-scoped journal stored in context. `AddThirdParty` and `AddTrace`
  append to the parent record **and** write a standalone JSON line to `thirdparty.log` / `trace.log`
  (dual-write), all sharing `request_id` + `trace_id`. Includes header redaction (`RedactHeaders`) and
  body capping (`CapBody`, skips binary content).
- `internal/delivery/http/middleware` — `AccessLog` middleware writes one JSON line per request to
  `api.log` with the **full request** (method, url, query, all headers, body) and **full response**
  (status, headers, body, latency), the correlating `trace_id`, and the embedded `thirdparty[]` /
  `trace[]` arrays. Sensitive headers are redacted; bodies are size-capped and gated by `LOG_HTTP_BODIES`.
- `pkg/httpclient` — `Client.Do` now records each outbound call into the request journal (method, url,
  host, status, latency, capped request/response bodies) when a journal is present in context — this is
  the source of the per-request "which third parties were hit" list. New `Options.LogBodyMaxBytes`.
- `pkg/messaging/kafka` & `pkg/messaging/rabbitmq` — consumers now start a per-message journal + APM/OTel
  span (continuing any propagated trace via the existing carriers) and write one structured line to
  `consumer.log` with `thirdparty[]` / `trace[]`, request_id, trace_id, latency, and status.
- `pkg/observability.TraceID(ctx)` — backend-agnostic trace-id extraction (OTel span context, then
  Elastic APM transaction) for log correlation.
- CLI skeleton — vendors `pkg/logger/sinks.go` and `pkg/journal`, the `AccessLog` middleware, and the
  previously-missing Kafka/RabbitMQ `carrier.go`; wires `SetupSinks` + `AccessLog` in `cmd/api/main.go`;
  adds `deploy/filebeat.yml` + `deploy/README.md` for shipping the four JSON logs to Elasticsearch.
- `config` — `LOG_DIR`, `LOG_ROTATION`, `LOG_MAX_AGE_DAYS`, `LOG_BODY_MAX_BYTES`, `LOG_HTTP_BODIES`.

### Fixed
- CLI skeleton — the Kafka/RabbitMQ packages were missing `carrier.go`, so projects generated with
  `--kafka`/`--rabbitmq` failed to compile; the carriers are now vendored.

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

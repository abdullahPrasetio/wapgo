# wapgo — Web API Platform for Go

> Production-ready Go microservice framework: Clean Architecture, ENV-first config (OpenShift/Kubernetes ready), observability built-in, and a CLI that scaffolds full projects in seconds.

[![CI](https://github.com/abdullahPrasetio/wapgo/actions/workflows/ci.yml/badge.svg)](https://github.com/abdullahPrasetio/wapgo/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/abdullahPrasetio/wapgo)](https://goreportcard.com/report/github.com/abdullahPrasetio/wapgo)
[![Coverage](https://img.shields.io/badge/coverage-%3E80%25-brightgreen)](https://github.com/abdullahPrasetio/wapgo/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Dokumentasi:** [Developer Guide](GUIDE.md) · [Arsitektur & Konsep](ARCHITECTURE.md) · [Keunggulan](HIGHLIGHTS.md) · [Changelog](CHANGELOG.md) · [Security](SECURITY.md) · [Contributing](CONTRIBUTING.md)

---

## Quick Start

```bash
# 1. Install the CLI
go install github.com/abdullahPrasetio/wapgo/cli/wapgo@latest

# 2. Scaffold a new service
wapgo new my-service --module github.com/me/my-service
cd my-service

# 3. Start infrastructure
make docker-up
cp .env.example .env

# 4. Run the service
make run

# 5. Verify
curl http://localhost:8080/health

# 6. Generate a new domain
wapgo make:all product
```

---

## Stack

| Layer | Technology |
|---|---|
| HTTP | [Fiber v2](https://gofiber.io) |
| ORM | GORM (PostgreSQL / MySQL) |
| Cache | Redis |
| Messaging | Kafka + RabbitMQ |
| Config | Viper (ENV-first, 12-factor) |
| Logger | zerolog — 4 structured log sinks (`api`, `consumer`, `thirdparty`, `trace`) |
| Auth | JWT HS256 + RBAC middleware |
| Tracing | OpenTelemetry SDK + Elastic APM (OTel→APM bridge via `apmotel`) |
| Metrics | Prometheus (`/metrics`) |
| Request Journal | `pkg/journal` — per-request `thirdparty[]` + `trace[]` embedded in parent log |
| CLI | Cobra — `wapgo new` + `wapgo make:*` |
| HTTP Client | net/http + retry + circuit breaker + SSRF guard + journal auto-record |

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
│  └────────────┬──────────┘  └───────────┬────────────┘  │
│               │                         │               │
│  ┌────────────▼──────────┐  ┌───────────▼────────────┐  │
│  │  Postgres / Redis impl │  │  HTTP Client impl      │  │
│  └───────────────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## Features

### Core (v0.1)
- **Clean Architecture** — strict layer boundaries enforced by interfaces.
- **ENV-first config** — Viper with explicit `BindEnv` for all 25+ variables; ready for K8s ConfigMap/Secret.
- **Middleware stack** — recover (no stack-trace leak) → request-id → security headers → rate-limit → logger → CORS.
- **Reference domain `user`** — full CRUD: entity → repository → usecase → handler → route.
- `/health` with DB + Redis checks, graceful shutdown (30s SIGTERM).

### Cache & Messaging (v0.2)
- **Redis cache** — `Cacher` interface + `RedisCacher` (JSON, TTL, namespace prefix).
- **Kafka** — producer + consumer group with context-aware graceful shutdown.
- **RabbitMQ** — topic exchange publisher + consumer with DLQ support.
- **Request-ID propagation** — injected into Kafka headers, AMQP properties, and outgoing HTTP.

### Resilient HTTP Client (v0.3)
- Retry with exponential back-off (5xx + network timeout, max 3 attempts).
- Circuit breaker (open after 5 consecutive failures, half-open after 30s).
- SSRF guard — blocks loopback / private / link-local destinations; supports allowlist.
- TLS verify ON, minimum TLS 1.2, response body capped at 10 MB.

### CLI (v0.4)
```bash
wapgo new <project> --module github.com/me/svc   # scaffold full project
wapgo make:all <name>                             # generate 8 domain files
wapgo make:model | make:repo | make:usecase | make:controller | make:route | make:client
wapgo add redis | kafka | rabbitmq                # add optional feature to existing project
wapgo list                                        # list generated domains in current project
wapgo upgrade                                     # check and self-update to latest release
wapgo upgrade --check                             # check only, do not install
wapgo version
```

### Auth & Observability (v0.5–v0.6)
- **JWT auth** — HS256, pinned algorithm, validates `exp/iat/iss/aud`, `alg:none` rejected, secret ≥ 32 bytes.
- **RBAC** — `auth.RequireRole("admin")` middleware.
- **Prometheus metrics** — `wapgo_http_requests_total`, `wapgo_http_request_duration_seconds`.
- **OpenTelemetry** — OTLP HTTP exporter, W3C TraceContext propagation, span per request.
- **Elastic APM** — `apmfiber`, GORM + Redis + HTTP client instrumentation.
- Switch providers via `OBSERVABILITY_PROVIDER=otel|elastic_apm`.

### CLI Interactive Wizard + Migrations (v0.8–v0.9)
- `wapgo new` — interactive wizard (charmbracelet/huh), selects DB, APM provider, Redis/Kafka/RabbitMQ; `--yes` for CI.
- `wapgo add <feature>` — add optional features to existing project.
- `wapgo make:migration <name>` — timestamped up/down SQL migration pair.
- `pkg/response.Paginated()` — paginated response helper with `PageMeta`.

### Observability Bridge + Generated Projects Compile (v0.10)
- **OTel → Elastic APM bridge** — `apmotel` TracerProvider installed when `OBSERVABILITY_PROVIDER=elastic_apm`; spans from `otel.Tracer(...)` in generated usecases now forward to Elastic APM.
- **`OBSERVABILITY_PROVIDER=none`** now genuinely disables tracing (no-op provider).
- **`observability.StartSpan(ctx, name)`** — provider-agnostic manual span helper.
- Generated projects now compile out of the box (`go mod tidy && go build ./...`) for all `--apm` choices.

### Structured Logging + Request Journal (v0.11)
- **4 log sinks** (`pkg/logger`) — `api.log`, `consumer.log`, `thirdparty.log`, `trace.log`. Rotation: `size` (lumberjack) or `daily` (date-stamped, midnight rollover).
- **`pkg/journal`** — per-request/message journal stored in `context.Context`. `AddThirdParty` and `AddTrace` dual-write: append to the parent record **and** write a standalone JSON line.
- **`AccessLog` middleware** — writes one JSON line per request to `api.log` with full request (method, url, headers, body) + full response (status, headers, body, latency) + embedded `thirdparty[]` + `trace[]`. Sensitive headers redacted; bodies size-capped.
- **`httpclient`** — auto-records each outbound call into the request journal when one is in context.
- **Kafka & RabbitMQ consumers** — per-message journal + APM/OTel span, structured line to `consumer.log`.

---

## Configuration

All settings are read from ENV (highest priority) → `config/config.yaml` → defaults.

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `production` enables hardening mode |
| `APP_PORT` | `8080` | HTTP listen port |
| `APP_NAME` | `wapgo-service` | Service name (used in logs + traces) |
| `DB_DRIVER` | `mysql` | `postgres` or `mysql` |
| `DB_HOST` | `localhost` | Database host |
| `DB_PASSWORD` | — | **Required in production** |
| `JWT_SECRET` | — | **Required, min 32 bytes** |
| `JWT_EXPIRY` | `24h` | Go duration string |
| `OBSERVABILITY_PROVIDER` | `elastic_apm` | `otel`, `elastic_apm`, or `none` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | e.g. `http://otel-collector:4318` |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `KAFKA_BROKERS` | — | Comma-separated `host:port` |
| `RABBITMQ_DSN` | — | `amqp://user:pass@host:5672/vhost` |
| `LOG_DIR` | `logs` | Directory for the 4 structured log files |
| `LOG_ROTATION` | `size` | `size` (lumberjack, 100 MB) or `daily` (date-stamped) |
| `LOG_MAX_AGE_DAYS` | `30` | Retention in days for log files |
| `LOG_HTTP_BODIES` | `false` | Capture full request/response bodies in `api.log` |
| `LOG_BODY_MAX_BYTES` | `8192` | Maximum body size captured (bytes) |

---

## Development

```bash
make run          # start API server
make test         # unit tests
make coverage     # test + HTML coverage report
make lint         # golangci-lint
make sec          # gosec + govulncheck
make docker-up    # postgres, redis, kafka, rabbitmq
make docker-down
make build        # compile API binary → bin/api
make cli-build    # compile CLI binary → bin/wapgo
```

### Integration tests (requires Docker)

```bash
go test -tags=integration -v ./internal/integration/...
```

---

## Deployment

### Docker

```bash
docker build -t wapgo:latest .
docker run -p 8080:8080 --env-file .env wapgo:latest
```

The image is built from `gcr.io/distroless/static-debian12:nonroot`:
- No shell, no package manager
- Runs as UID 65532 (non-root)
- Read-only root filesystem

### Kubernetes

```bash
kubectl apply -f kubernetes/configmap.yaml
kubectl apply -f kubernetes/secret.yaml     # edit real values first
kubectl apply -f kubernetes/deployment.yaml
kubectl apply -f kubernetes/service.yaml
kubectl apply -f kubernetes/networkpolicy.yaml
```

---

## Security

See [SECURITY.md](SECURITY.md) for the full security policy and vulnerability reporting process.

**CI security gates** — all must pass before merge:
`gosec` · `govulncheck` · `gitleaks` · `trivy` · `golangci-lint` · coverage ≥ 80 %

---

## Benchmark

> Measured on Apple M1 (darwin/arm64) · `go run` mode · Postgres + Redis local · `wrk -t4 -c100 -d15s`
> Production binary (`go build -ldflags="-s -w"`) on Linux will be 10–20 % faster.

### HTTP Throughput

| Endpoint | RPS | Avg Latency | Notes |
|---|---|---|---|
| `GET /` | **79,057** | 1.94 ms | Pure JSON, no I/O |
| `GET /api/v1/users` | **12,190** | 12.85 ms | Postgres query + pagination |
| `GET /health` | **6,815** | 18.99 ms | Probes DB + Redis each request |

### Internal Package (`go test -bench=. -benchmem`)

| Benchmark | ns/op | allocs/op | Throughput |
|---|---|---|---|
| `auth.Sign` (HS256) | 3,044 | 41 | ~328K token/s |
| `auth.Verify` (HS256) | 4,187 | 69 | ~239K token/s |
| `auth.Sign+Verify` (round-trip) | 6,674 | 110 | ~150K req/s |
| `validator.Validate` (valid) | 861 | 5 | ~1.16M req/s |
| `validator.Validate` (invalid) | 1,145 | 34 | ~873K req/s |
| `response.Marshal` (success) | 510 | 9 | ~1.96M req/s |
| `response.Marshal` (error) | 156 | 1 | ~6.4M req/s |
| `response.Marshal` (paginated) | 805 | 12 | ~1.24M req/s |

---

## Coverage

| Package | Coverage |
|---|---|
| config | 90 % |
| pkg/auth | 92 % |
| pkg/httpclient | 94 % |
| pkg/journal | 85 % |
| pkg/logger | 88 % |
| pkg/messaging/kafka | 91 % |
| pkg/messaging/rabbitmq | 84 % |
| pkg/observability | 90 % |
| internal/usecase | 88 % |
| internal/delivery/http/handler | 97 % |
| internal/delivery/http/middleware | 85 % |
| internal/repository/db | 90 % |
| internal/repository/redis | 92 % |
| **Total** | **> 80 %** ✅ |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT — see [LICENSE](LICENSE).

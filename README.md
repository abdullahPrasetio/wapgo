# wapgo — Web API Platform for Go

> Production-ready Go microservice framework: Clean Architecture, ENV-first config (OpenShift/Kubernetes ready), observability built-in, and a CLI that scaffolds full projects in seconds.

[![CI](https://github.com/abdullahPrasetio/wapgo/actions/workflows/ci.yml/badge.svg)](https://github.com/abdullahPrasetio/wapgo/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/abdullahPrasetio/wapgo)](https://goreportcard.com/report/github.com/abdullahPrasetio/wapgo)
[![Coverage](https://img.shields.io/badge/coverage-%3E80%25-brightgreen)](https://github.com/abdullahPrasetio/wapgo/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Dokumentasi:** [Developer Guide](GUIDE.md) · [Arsitektur & Konsep](ARCHITECTURE.md) · [Keunggulan](HIGHLIGHTS.md) · [Security](SECURITY.md) · [Contributing](CONTRIBUTING.md)

---

## Quick Start

```bash
# 1. Install the CLI
go install github.com/abdullahPrasetio/wapgo/cli/cmd@latest   # → binary `wapgo`

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
| Logger | zerolog (JSON prod / console dev) |
| Auth | JWT HS256 + RBAC middleware |
| Tracing | OpenTelemetry SDK + Elastic APM |
| Metrics | Prometheus (`/metrics`) |
| CLI | Cobra — `wapgo new` + `wapgo make:*` |
| HTTP Client | net/http + retry + circuit breaker + SSRF guard |

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
wapgo version
```

### Auth & Observability (v0.5–v0.6)
- **JWT auth** — HS256, pinned algorithm, validates `exp/iat/iss/aud`, `alg:none` rejected, secret ≥ 32 bytes.
- **RBAC** — `auth.RequireRole("admin")` middleware.
- **Prometheus metrics** — `wapgo_http_requests_total`, `wapgo_http_request_duration_seconds`.
- **OpenTelemetry** — OTLP HTTP exporter, W3C TraceContext propagation, span per request.
- **Elastic APM** — `apmfiber`, GORM + Redis + HTTP client instrumentation.
- Switch providers via `OBSERVABILITY_PROVIDER=otel|elastic_apm`.

---

## Configuration

All settings are read from ENV (highest priority) → `config/config.yaml` → defaults.

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `production` enables hardening mode |
| `APP_PORT` | `8080` | HTTP listen port |
| `APP_NAME` | `wapgo-service` | Service name (used in logs + traces) |
| `DB_DRIVER` | `postgres` | `postgres` or `mysql` |
| `DB_HOST` | `localhost` | Database host |
| `DB_PASSWORD` | — | **Required in production** |
| `JWT_SECRET` | — | **Required, min 32 bytes** |
| `JWT_EXPIRY` | `24h` | Go duration string |
| `OBSERVABILITY_PROVIDER` | `otel` | `otel` or `elastic_apm` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | e.g. `http://otel-collector:4318` |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `KAFKA_BROKERS` | — | Comma-separated `host:port` |
| `RABBITMQ_DSN` | — | `amqp://user:pass@host:5672/vhost` |

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

## Coverage

| Package | Coverage |
|---|---|
| config | 90 % |
| pkg/auth | 92 % |
| pkg/httpclient | 94 % |
| pkg/messaging/kafka | 91 % |
| pkg/messaging/rabbitmq | 84 % |
| pkg/observability | 90 % |
| internal/usecase | 88 % |
| internal/delivery/http/handler | 97 % |
| internal/repository/redis | 92 % |
| **Total** | **> 80 %** ✅ |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT — see [LICENSE](LICENSE).

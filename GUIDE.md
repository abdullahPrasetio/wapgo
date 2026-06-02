# wapgo — Developer Guide

> **wapgo** (*Web API Platform for Go*) adalah framework microservice Go yang production-ready: Clean Architecture, ENV-first config, observability & JWT bawaan, dan CLI generator ala Laravel artisan.

**Dokumen lain:** [README](README.md) · [Arsitektur & Konsep](ARCHITECTURE.md) · [Security](SECURITY.md) · [Contributing](CONTRIBUTING.md)

---

## Daftar Isi

1. [Instalasi CLI](#1-instalasi-cli)
2. [Membuat Project Baru](#2-membuat-project-baru)
3. [Menjalankan Service](#3-menjalankan-service)
4. [Struktur Project](#4-struktur-project)
5. [Generator CLI — Make Commands](#5-generator-cli--make-commands)
6. [Konfigurasi ENV](#6-konfigurasi-env)
7. [Paket pkg/ — Referensi Cepat](#7-paket-pkg--referensi-cepat)
   - [logger](#71-logger)
   - [auth (JWT)](#72-auth-jwt)
   - [httpclient](#73-httpclient)
   - [messaging — Kafka](#74-messaging--kafka)
   - [messaging — RabbitMQ](#75-messaging--rabbitmq)
   - [observability](#76-observability)
8. [Observability: OTel vs Elastic APM](#8-observability-otel-vs-elastic-apm)
9. [Health Check](#9-health-check)
10. [Makefile Commands](#10-makefile-commands)
11. [Fitur Security Bawaan](#11-fitur-security-bawaan)
12. [Fase yang Sudah Selesai](#12-fase-yang-sudah-selesai)

---

## 1. Instalasi CLI

```bash
go install github.com/abdullahPrasetio/wapgo/cli/wapgo@latest
```

Setelah install, binary `wapgo` tersedia di `$GOPATH/bin`. Verifikasi:

```bash
wapgo version
```

Untuk build dari source (development):

```bash
make cli-build      # output → bin/wapgo
make cli-install    # install ke $GOPATH/bin
```

---

## 2. Membuat Project Baru

```bash
wapgo new my-service --module github.com/yourorg/my-service
```

Perintah ini men-scaffold project lengkap ke folder `my-service/`:

```
my-service/
├── cmd/api/main.go          ← entrypoint, wiring lengkap
├── config/                  ← Viper loader
├── internal/
│   ├── domain/              ← entity, repository interface, service interface
│   ├── usecase/             ← business logic + OTel spans bawaan
│   ├── delivery/http/       ← handler, middleware, route
│   └── repository/          ← postgres impl, redis cache
├── pkg/                     ← shared packages (logger, auth, httpclient, …)
├── migrations/
├── Makefile
├── docker-compose.yml       ← postgres, mysql, redis, kafka, rabbitmq
└── .env.example
```

---

## 3. Menjalankan Service

```bash
cd my-service

# 1. Jalankan infrastruktur (Docker)
make docker-up

# 2. Copy dan sesuaikan config
cp .env.example .env

# 3. Jalankan service
make run

# 4. Cek health
curl http://localhost:8080/health
```

Output health yang normal:

```json
{
  "status": "ok",
  "services": {
    "database": "ok",
    "redis": "ok",
    "kafka": "not_configured",
    "rabbitmq": "not_configured"
  }
}
```

---

## 4. Struktur Project

Clean Architecture dengan batas layer yang ketat:

```
Handler (HTTP) → Usecase (bisnis) → Repository/ExternalService (interface)
```

- Tidak ada import konkret lintas layer — semua lewat interface.
- Wiring (injeksi dependensi) hanya di `cmd/api/main.go`.
- Interface domain didefinisikan di `internal/domain/`, implementasi di `internal/repository/` atau `pkg/`.

---

## 5. Generator CLI — Make Commands

Setelah project dibuat, tambahkan domain baru dengan satu perintah:

```bash
# Generate semua layer sekaligus (paling sering dipakai)
wapgo make:all product

# Atau generate layer satu per satu
wapgo make:model product
wapgo make:repo product
wapgo make:usecase product
wapgo make:controller product
wapgo make:route product
wapgo make:client product      # external HTTP client
```

### Apa yang dihasilkan `make:all <domain>`

| File yang dibuat | Isi |
|---|---|
| `internal/domain/entity/<domain>.go` | Struct entity |
| `internal/domain/repository/<domain>_repository.go` | Repository interface |
| `internal/domain/service/external_<domain>.go` | External service interface |
| `internal/repository/postgres/<domain>_repository.go` | Implementasi GORM |
| `internal/usecase/<domain>_usecase.go` | Bisnis logic + OTel span per method |
| `internal/delivery/http/handler/<domain>_handler.go` | Fiber handler |
| `internal/delivery/http/route/<domain>_route.go` | Route registration |
| `pkg/httpclient/<domain>_client.go` | External HTTP client |

Usecase yang di-generate sudah memiliki tracing otomatis:

```go
var tracer = otel.Tracer("product-usecase")

func (uc *ProductUsecase) GetByID(ctx context.Context, id string) (*entity.Product, error) {
    ctx, span := tracer.Start(ctx, "GetByID")
    defer span.End()

    result, err := uc.repo.FindByID(ctx, id)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    return result, nil
}
```

---

## 6. Konfigurasi ENV

Semua konfigurasi dibaca dari ENV (atau `.env`). Prioritas: `ENV → config.yaml → default`.

### App

| ENV | Default | Keterangan |
|---|---|---|
| `APP_NAME` | `wapgo-service` | Nama service |
| `APP_ENV` | `development` | `development` / `production` |
| `APP_PORT` | `8080` | Port HTTP |

### Database

| ENV | Default | Keterangan |
|---|---|---|
| `DB_DRIVER` | `postgres` | `postgres` / `mysql` |
| `DB_HOST` | `localhost` | |
| `DB_PORT` | `5432` | |
| `DB_NAME` | `wapgo_db` | |
| `DB_USER` | `postgres` | |
| `DB_PASSWORD` | *(kosong)* | |
| `DB_MAX_OPEN_CONNS` | `25` | Connection pool |
| `DB_MAX_IDLE_CONNS` | `5` | |

### Redis

| ENV | Default | Keterangan |
|---|---|---|
| `REDIS_HOST` | `localhost` | |
| `REDIS_PORT` | `6379` | |
| `REDIS_PASSWORD` | *(kosong)* | |
| `REDIS_DB` | `0` | |

### JWT

| ENV | Default | Keterangan |
|---|---|---|
| `JWT_SECRET` | *(wajib diisi, ≥ 32 karakter)* | Secret signing HS256 |
| `JWT_EXPIRY` | `24h` | Durasi token |
| `JWT_ISSUER` | `wapgo-service` | Klaim `iss` |
| `JWT_AUDIENCE` | `wapgo-client` | Klaim `aud` |

### Kafka

| ENV | Default | Keterangan |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated |
| `KAFKA_GROUP_ID` | `wapgo-group` | Consumer group |

### RabbitMQ

| ENV | Default | Keterangan |
|---|---|---|
| `RABBITMQ_DSN` | `amqp://guest:guest@localhost:5672/` | |

### Observability

| ENV | Default | Keterangan |
|---|---|---|
| `OBSERVABILITY_PROVIDER` | `otel` | `otel` / `elastic_apm` |
| `OTEL_TRACING_ENABLED` | `false` | Aktifkan OTel tracing |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | *(kosong)* | OTLP endpoint (Jaeger, Tempo, dll) |

Untuk Elastic APM (jika `OBSERVABILITY_PROVIDER=elastic_apm`):

| ENV | Keterangan |
|---|---|
| `ELASTIC_APM_SERVER_URL` | URL APM server (mis. `http://apm-server:8200`) |
| `ELASTIC_APM_SERVICE_NAME` | Nama service di Kibana APM |
| `ELASTIC_APM_SECRET_TOKEN` | Token autentikasi |
| `ELASTIC_APM_ENVIRONMENT` | `production` / `staging` / dll |
| `ELASTIC_APM_ACTIVE` | `true` untuk mengaktifkan agent |

---

## 7. Paket `pkg/` — Referensi Cepat

### 7.1 logger

```go
import "github.com/abdullahPrasetio/wapgo/pkg/logger"

log := logger.New(cfg.Logger)

// Log dengan request ID dari context
log.Info().Str("user_id", id).Msg("user fetched")

// Gunakan logger request-scoped (dari middleware)
logger.FromContext(ctx).Error().Err(err).Msg("operation failed")
```

Output JSON di produksi, console di development. Rotasi file otomatis via lumberjack.

### 7.2 auth (JWT)

```go
import "github.com/abdullahPrasetio/wapgo/pkg/auth"

// Sign token
token, err := auth.Sign(auth.Claims{
    UserID: "user-123",
    Role:   "admin",
}, cfg.JWT)

// Verify token
claims, err := auth.Verify(token, cfg.JWT)

// Middleware di Fiber route
app.Use(auth.Middleware(cfg.JWT))
app.Use(auth.RequireRole("admin"))

// Ambil claims dari handler
func handler(c *fiber.Ctx) error {
    claims := auth.GetClaims(c)
    fmt.Println(claims.UserID, claims.Role)
    return nil
}
```

Hardening: algoritma di-pin ke HS256, validasi `exp`/`iat`/`iss`/`aud`, `alg:none` ditolak, secret ≥ 32 byte.

### 7.3 httpclient

```go
import "github.com/abdullahPrasetio/wapgo/pkg/httpclient"

client := httpclient.New(httpclient.Options{
    BaseURL: "https://api.example.com",
    Timeout: 5 * time.Second,
    AllowedHosts: []string{"api.example.com"},  // SSRF guard
    TransportWrapper: obsProvider.WrapTransport, // tracing otomatis
})

// Request dengan context (propagasi X-Request-ID & Authorization otomatis)
resp, err := client.Do(ctx, http.MethodGet, "/users/123", nil)
```

Bawaan: retry (3x, exponential backoff), circuit breaker (open setelah 5 gagal), TLS verify ON, SSRF guard, timeout per-request.

### 7.4 messaging — Kafka

```go
import "github.com/abdullahPrasetio/wapgo/pkg/messaging/kafka"

// Producer
// brokers: comma-separated "host:port"
producer := kafka.NewProducer("localhost:9092", logger)
err := producer.Publish(ctx, kafka.Message{
    Topic: "user.events",
    Key:   []byte("user-123"),
    Value: []byte(`{"event":"created","id":"123"}`),
})
defer producer.Close()

// Consumer
// brokers: comma-separated, groupID: consumer group, topic: satu topic
consumer := kafka.NewConsumer("localhost:9092", "my-service-group", "user.events", logger)
err = consumer.Start(ctx, func(ctx context.Context, msg kafka.Message) error {
    // proses pesan; return non-nil = offset tidak di-commit (re-delivered)
    return nil
})
defer consumer.Close()

// Health check (untuk /health endpoint)
kafka.HealthCheck("localhost:9092")  // return func(ctx) string
```

`X-Request-ID` dipropagasi otomatis via Kafka header `x-request-id`.

### 7.5 messaging — RabbitMQ

```go
import "github.com/abdullahPrasetio/wapgo/pkg/messaging/rabbitmq"

// Publisher
pub, err := rabbitmq.NewPublisher("amqp://guest:guest@localhost:5672/", "user.events", logger)
if err != nil { ... }
defer pub.Close()

err = pub.Publish(ctx, rabbitmq.Message{
    RoutingKey: "user.created",
    Body:       []byte(`{"id":"123"}`),
})

// Consumer dengan Dead Letter Queue otomatis
cons, err := rabbitmq.NewConsumer("amqp://guest:guest@localhost:5672/", "user.events", logger)
if err != nil { ... }
defer cons.Close()

// Subscribe: declare queue + bind routing key + mulai goroutine drain
err = cons.Subscribe("user.events.created", "user.created",
    func(ctx context.Context, msg rabbitmq.Message) error {
        // return non-nil = Nack → pesan masuk ke DLQ otomatis
        return nil
    },
)

// Health check (untuk /health endpoint)
rabbitmq.HealthCheck("amqp://guest:guest@localhost:5672/")  // return func(ctx) string
```

DLQ (`user.events.created.dlq`) dikonfigurasi otomatis via `x-dead-letter-exchange`.

### 7.6 observability

```go
import "github.com/abdullahPrasetio/wapgo/pkg/observability"

// Setup provider (di main.go)
obsProvider, err := observability.New(ctx, &cfg.Observability, cfg.App.Name, version)

// Instrument dependencies
obsProvider.InstrumentGORM(db)
obsProvider.InstrumentRedis(redisClient)

// Pasang middleware ke Fiber
app.Use(obsProvider.HTTPMiddleware())       // tracing server span
app.Use(observability.MetricsMiddleware())  // Prometheus RED metrics

// Di handler / usecase: ambil context dengan span aktif
ctx := observability.TraceContext(c)

// Shutdown bersih
defer obsProvider.Shutdown(shutCtx)
```

Prometheus metrics tersedia di `GET /metrics` (404 di `APP_ENV=production`).

---

## 8. Observability: OTel vs Elastic APM

Set satu ENV untuk memilih:

```bash
OBSERVABILITY_PROVIDER=otel         # default — kirim ke Jaeger/Tempo/dll via OTLP
OBSERVABILITY_PROVIDER=elastic_apm  # kirim ke Kibana APM
```

### Coverage tracing end-to-end

| Layer | OTel | Elastic APM |
|---|---|---|
| HTTP Server span | ✅ W3C TraceContext | ✅ `apmfiber.Middleware` |
| GORM (query DB) | ✅ `otelgorm.NewPlugin()` | ✅ Custom GORM callback plugin |
| Redis commands | ✅ `redisotel.InstrumentTracing()` | ✅ Custom `ProcessHook` |
| Outgoing HTTP | ✅ `otelhttp.NewTransport()` | ✅ `apmhttp.WrapRoundTripper()` |
| Usecase layer | ✅ Manual span per method | ✅ Child span dari server span |
| Kafka / RabbitMQ | ✅ W3C header propagation | ✅ W3C header propagation |

### Setup OTel (Jaeger / Grafana Tempo)

```bash
OBSERVABILITY_PROVIDER=otel
OTEL_TRACING_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
```

### Setup Elastic APM (Kibana)

```bash
OBSERVABILITY_PROVIDER=elastic_apm
ELASTIC_APM_SERVER_URL=http://apm-server:8200
ELASTIC_APM_SERVICE_NAME=my-service
ELASTIC_APM_SECRET_TOKEN=your-token
ELASTIC_APM_ENVIRONMENT=production
ELASTIC_APM_ACTIVE=true
```

Tidak perlu kode tambahan — framework otomatis memilih provider dan menginstru-men semua layer.

---

## 9. Health Check

`GET /health` mengembalikan status semua dependensi:

```json
{
  "status": "ok",
  "services": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "rabbitmq": "ok"
  }
}
```

Status `"not_configured"` muncul bila ENV service tidak diset. `"error: ..."` bila service gagal dicapai. HTTP status 503 bila ada yang error.

Tambahkan checker custom:

```go
// di main.go
healthHandler.AddChecker("my-service", func(ctx context.Context) error {
    return myClient.Ping(ctx)
})
```

---

## 10. Makefile Commands

| Command | Fungsi |
|---|---|
| `make run` | Jalankan service (hot-reload manual) |
| `make build` | Build binary ke `bin/api` |
| `make cli-build` | Build CLI ke `bin/wapgo` |
| `make cli-install` | Install CLI ke `$GOPATH/bin` |
| `make test` | Run semua test (dengan `-race`) |
| `make coverage` | Test + coverage HTML report |
| `make lint` | golangci-lint |
| `make sec` | gosec + govulncheck |
| `make docker-up` | Jalankan semua infrastruktur (Docker) |
| `make docker-down` | Stop Docker |
| `make migrate` | Jalankan auto-migrate GORM |
| `make tidy` | `go mod tidy` |

---

## 11. Fitur Security Bawaan

Semua aktif tanpa konfigurasi tambahan:

| Fitur | Detail |
|---|---|
| **Security headers** | HSTS, X-Content-Type-Options, X-Frame-Options=DENY, Referrer-Policy, CSP |
| **Rate limiting** | Per-IP, konfigurabel via `RATE_LIMIT_MAX` dan `RATE_LIMIT_WINDOW` |
| **Body limit** | 4MB default |
| **CORS** | Allowlist ketat via `CORS_ALLOWED_ORIGINS` |
| **Recover** | Panic dicatch tanpa bocor stack trace ke response |
| **TLS verify** | ON by default di httpclient (`InsecureSkipVerify=false`, min TLS 1.2) |
| **SSRF guard** | Allowlist host tujuan, tolak redirect ke internal/loopback/link-local |
| **JWT hardening** | Algo di-pin HS256, validasi `exp`/`iat`/`iss`/`aud`, `alg:none` ditolak |
| **SQL injection** | GORM parameterized query — tidak ada raw string concat |
| **Input validation** | `go-playground/validator` di semua DTO |
| **Secret redaction** | Field sensitif tidak pernah muncul di log |
| **Metrics guard** | `/metrics` mengembalikan 404 di `APP_ENV=production` |

---

## 12. Fase yang Sudah Selesai

| Fase | Fitur | Status |
|---|---|---|
| **v0.1** | Core skeleton: CRUD users, Postgres, Fiber, middleware stack, health check | ✅ |
| **v0.2** | Redis cache, Kafka producer/consumer, RabbitMQ publisher/consumer, DLQ | ✅ |
| **v0.3** | HTTP client: retry, circuit breaker, TLS, SSRF guard | ✅ |
| **v0.4** | CLI `wapgo new` + `make:all` + `make:*` generator | ✅ |
| **v0.5** | JWT auth + RBAC, Prometheus metrics, OTel tracing dasar | ✅ |
| **v0.6** | Provider abstraction OTel / Elastic APM, full end-to-end tracing semua layer | ✅ |

Coverage semua paket > 80%. `go build ./...` dan `go vet ./...` bersih.

---

## Contoh `.env` Lengkap

```dotenv
# App
APP_NAME=my-service
APP_ENV=development
APP_PORT=8080

# Database
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASSWORD=secret

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT (wajib ≥ 32 karakter)
JWT_SECRET=supersecretkey-that-is-at-least-32-chars
JWT_EXPIRY=24h
JWT_ISSUER=my-service
JWT_AUDIENCE=my-client

# Kafka (opsional)
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=my-service-group

# RabbitMQ (opsional)
RABBITMQ_DSN=amqp://guest:guest@localhost:5672/

# Observability — pilih salah satu

# Opsi 1: OTel (Jaeger / Grafana Tempo)
OBSERVABILITY_PROVIDER=otel
OTEL_TRACING_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Opsi 2: Elastic APM (Kibana)
# OBSERVABILITY_PROVIDER=elastic_apm
# ELASTIC_APM_SERVER_URL=http://localhost:8200
# ELASTIC_APM_SERVICE_NAME=my-service
# ELASTIC_APM_SECRET_TOKEN=
# ELASTIC_APM_ENVIRONMENT=development
# ELASTIC_APM_ACTIVE=true
```

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
   - [logger (4 sinks)](#71-logger-4-sinks)
   - [journal (request journal)](#72-journal-request-journal)
   - [auth (JWT)](#73-auth-jwt)
   - [httpclient](#74-httpclient)
   - [messaging — Kafka](#75-messaging--kafka)
   - [messaging — RabbitMQ](#76-messaging--rabbitmq)
   - [observability](#77-observability)
   - [Worker binary](#78-worker-binary)
   - [notification — SMTP (email)](#79-notification--smtp-email)
   - [notification — Firebase FCM](#710-notification--firebase-fcm-push-notification)
   - [auth/google (Google OAuth2)](#711-authgoogle-google-oauth2)
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

### Upgrade CLI

Cek dan upgrade ke versi terbaru:

```bash
wapgo upgrade           # cek GitHub release dan jalankan go install jika ada yang baru
wapgo upgrade --check   # hanya cek, tidak install
```

Output contoh saat ada update:

```
  ✦ wapgo upgrade

  →  installed : v1.4.0
  →  latest    : v1.4.2

  ↑  update available: v1.4.0 → v1.4.2

  →  running: go install github.com/abdullahPrasetio/wapgo/cli/wapgo@v1.4.2

  ✓ upgraded to v1.4.2  run wapgo version to confirm
```

Jika sudah up to date:

```
  ✦ wapgo upgrade

  →  installed : v1.4.2
  →  latest    : v1.4.2

  ✓ already up to date
```

> **Catatan:** Command ini membutuhkan koneksi internet untuk mengakses GitHub API.
> Jika offline, peringatan ditampilkan namun tidak menyebabkan error fatal.

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

### `make:worker` — Worker Binary

Generate standalone consumer binary terpisah dari HTTP server:

```bash
# Worker tunggal (auto-detect broker dari go.mod)
wapgo make:worker

# Worker dengan nama (untuk multi-domain worker terpisah)
wapgo make:worker order
wapgo make:worker payment --broker kafka
wapgo make:worker notification --broker rabbitmq
wapgo make:worker sync --broker both
```

Output:
- `cmd/worker/main.go` (tanpa nama) atau `cmd/worker-{name}/main.go` (dengan nama)
- Makefile: `run-worker-{name}` dan `build-worker-{name}` ditambahkan otomatis

Jalankan worker terpisah dari API:

```bash
make run-worker-order     # via Makefile
# atau langsung:
go run ./cmd/worker-order
```

---

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
| `DB_DRIVER` | `mysql` | `postgres` / `mysql` |
| `DB_HOST` | `localhost` | |
| `DB_PORT` | `3306` | |
| `DB_NAME` | `wapgo_db` | |
| `DB_USER` | `root` | |
| `DB_PASSWORD` | *(kosong)* | |
| `DB_MAX_OPEN_CONNS` | `25` | Connection pool |
| `DB_MAX_IDLE_CONNS` | `5` | |

### Redis

| ENV | Default | Keterangan |
|---|---|---|
| `REDIS_URL` | `redis://localhost:6379` | URL koneksi (menggantikan HOST/PORT) |
| `REDIS_PASSWORD` | *(kosong)* | Password auth |
| `REDIS_DB` | `0` | Nomor database |
| `REDIS_POOL_SIZE` | `20` | Jumlah maksimum koneksi di pool |
| `REDIS_MIN_IDLE_CONNS` | `5` | Koneksi idle minimal yang selalu siap |
| `REDIS_DIAL_TIMEOUT` | `5s` | Timeout saat membuka koneksi baru |
| `REDIS_READ_TIMEOUT` | `3s` | Timeout baca per perintah |
| `REDIS_WRITE_TIMEOUT` | `3s` | Timeout tulis per perintah |
| `REDIS_MAX_RETRIES` | `3` | Jumlah retry otomatis pada error sementara |

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
| `KAFKA_HEARTBEAT_INTERVAL` | `3s` | Interval heartbeat ke broker; naikkan ke `5s`–`10s` di jaringan lambat |
| `KAFKA_SESSION_TIMEOUT` | `30s` | Batas waktu sebelum broker anggap consumer mati dan trigger rebalance |
| `KAFKA_REBALANCE_TIMEOUT` | `30s` | Waktu maksimal rebalance grup; naikkan untuk cluster besar |

### RabbitMQ

| ENV | Default | Keterangan |
|---|---|---|
| `RABBITMQ_DSN` | `amqp://guest:guest@localhost:5672/` | URL koneksi AMQP |
| `RABBITMQ_EXCHANGE` | `{app-name}-exchange` | Nama topic exchange |

### SMTP (add-on opsional)

Hanya diperlukan jika `pkg/notification/email` digunakan. Biarkan kosong jika tidak.

| ENV | Default | Keterangan |
|---|---|---|
| `SMTP_HOST` | *(kosong)* | Hostname server SMTP |
| `SMTP_PORT` | `587` | `587`=STARTTLS, `465`=implicit TLS, `25`=plain |
| `SMTP_USERNAME` | *(kosong)* | Username autentikasi |
| `SMTP_PASSWORD` | *(kosong)* | Password autentikasi |
| `SMTP_FROM` | *(kosong)* | Alamat pengirim (`From:` header) |
| `SMTP_TIMEOUT` | `10s` | Timeout koneksi TCP + transaksi SMTP |

### Firebase FCM (add-on opsional)

Hanya diperlukan jika `pkg/notification/firebase` digunakan. Biarkan kosong jika tidak.

| ENV | Default | Keterangan |
|---|---|---|
| `FIREBASE_CREDENTIALS_JSON` | *(kosong)* | Konten JSON service account key (Firebase Console → Project Settings → Service Accounts → Generate new private key) |

### Google OAuth2 (add-on opsional — `wapgo add google-auth`)

Hanya diperlukan jika fitur Google login/register diaktifkan. Wajib diisi semua tiga ENV di bawah jika fitur ini dipakai.

| ENV | Default | Keterangan |
|---|---|---|
| `GOOGLE_CLIENT_ID` | *(kosong, wajib)* | OAuth2 Client ID — dari Google Cloud Console → APIs & Services → Credentials |
| `GOOGLE_CLIENT_SECRET` | *(kosong, wajib)* | OAuth2 Client Secret — dari halaman yang sama |
| `GOOGLE_REDIRECT_URL` | *(kosong, wajib)* | Callback URL yang **sama persis** dengan yang didaftarkan di Google Console, mis. `http://localhost:8080/auth/google/callback` |

> **Cara dapat Client ID & Secret:**
> 1. Buka [console.cloud.google.com](https://console.cloud.google.com) → APIs & Services → Credentials
> 2. Create Credentials → OAuth 2.0 Client IDs → Application type: **Web application**
> 3. Tambah Authorized redirect URIs: URL callback service kamu
> 4. Copy Client ID dan Client Secret

### Health Check

| ENV | Default | Keterangan |
|---|---|---|
| `HEALTH_PROBE_TIMEOUT` | `2s` | Timeout per probe dependensi (DB, Redis, dll) |

### Observability

| ENV | Default | Keterangan |
|---|---|---|
| `OBSERVABILITY_PROVIDER` | `elastic_apm` | `otel` / `elastic_apm` / `none` |
| `OTEL_TRACING_ENABLED` | `false` | Aktifkan OTel tracing |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | *(kosong)* | OTLP endpoint (Jaeger, Tempo, dll) |

### Logging (pkg/logger)

| ENV | Default | Keterangan |
|---|---|---|
| `LOG_DIR` | `logs` | Direktori 4 file log terstruktur |
| `LOG_ROTATION` | `size` | `size` (lumberjack 100 MB) atau `daily` (per tanggal) |
| `LOG_MAX_AGE_DAYS` | `30` | Retensi file log dalam hari |
| `LOG_HTTP_BODIES` | `false` | Catat body request/response di `api.log` |
| `LOG_BODY_MAX_BYTES` | `8192` | Batas ukuran body yang dicatat (byte) |

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

### 7.1 logger (4 sinks)

`pkg/logger` mengelola empat file log terstruktur secara bersamaan: `api.log`, `consumer.log`,
`thirdparty.log`, dan `trace.log`. Setiap file adalah JSON line-delimited.

```go
import "github.com/abdullahPrasetio/wapgo/pkg/logger"

// Inisialisasi di main.go (satu kali)
if err := logger.SetupSinks(logger.SinkConfig{
    Dir:        "logs",       // default "logs"
    Rotation:   "size",       // "size" (lumberjack) atau "daily" (date-stamped)
    MaxSizeMB:  100,
    MaxAgeDays: 30,
    Console:    true,         // juga echo ke stdout (dev)
}); err != nil {
    log.Fatal().Err(err).Msg("failed to setup log sinks")
}

// Akses logger per kategori
logger.API().Info().Str("method", "GET").Str("path", "/users").Msg("request")
logger.Consumer().Info().Str("topic", "user.events").Msg("message received")
logger.ThirdParty().Info().Str("url", "https://api.example.com").Msg("outbound call")
logger.Trace().Info().Str("name", "risk-score").Msg("custom trace")
```

Mode rotasi dikontrol via ENV `LOG_ROTATION`:
- `size` (default) — lumberjack, rotasi per `LOG_MAX_SIZE_MB`, retensi `LOG_MAX_AGE_DAYS` hari.
- `daily` — file berstempel tanggal (`api-2026-06-03.log`), rotasi tengah malam, retensi `LOG_MAX_AGE_DAYS` hari.

### 7.2 journal (request journal)

`pkg/journal` mengumpulkan semua hit thirdparty dan custom trace yang terjadi selama satu request/pesan
ke dalam satu record induk. Semua entry juga ditulis ke file sink masing-masing (dual-write).

```go
import "github.com/abdullahPrasetio/wapgo/pkg/journal"

// Di middleware (dilakukan otomatis oleh AccessLog middleware):
ctx, j := journal.Start(ctx, "api")
defer j.Finish() // menulis 1 baris JSON ke api.log berisi thirdparty[] + trace[]

// Di httpclient (dilakukan otomatis bila journal ada di ctx):
// → AddThirdParty dipanggil otomatis, masuk ke thirdparty[] induk + thirdparty.log

// Di usecase / handler (custom trace):
journal.FromContext(ctx).AddTrace("risk-score", map[string]any{
    "score": 0.87,
    "user":  userID,
})
// → masuk ke trace[] induk + trace.log
```

Redaksi header sensitif (Authorization, Cookie, Set-Cookie) dan pembatasan ukuran body dilakukan
secara otomatis. Gate `LOG_HTTP_BODIES=true` diperlukan agar body request/response dicatat.

### 7.3 auth (JWT)

```go
import "github.com/abdullahPrasetio/wapgo/pkg/auth"

// Sign access token (token_type:"access", JTI otomatis, expiry dari cfg.JWT.Expiry)
token, err := auth.Sign("user-123", []string{"admin"}, &cfg.JWT)

// Sign refresh token (token_type:"refresh", expiry biasanya lebih panjang)
refresh, err := auth.SignRefresh("user-123", &cfg.JWT)

// Verify token → kembalikan *Claims
claims, err := auth.Verify(token, &cfg.JWT)
fmt.Println(claims.Subject, claims.Roles, claims.TokenType, claims.ID) // JTI ada di ID

// Middleware di Fiber route
app.Use(auth.Middleware(&cfg.JWT))          // tolak token bukan access (refresh ditolak)
app.Use(auth.RequireRole("admin"))          // RBAC

// Ambil claims dari handler
func handler(c *fiber.Ctx) error {
    claims := auth.GetClaims(c)
    fmt.Println(claims.Subject, claims.Roles)
    return nil
}
```

**Hardening yang aktif:**
- Algoritma di-pin ke HS256 — `alg:none` dan algoritma lain ditolak
- Validasi `exp` / `iat` / `iss` / `aud` ketat
- Secret ≥ 32 byte (gagal saat Sign/Verify jika kurang)
- **JTI (JWT ID):** setiap token dapat ID random 16-byte hex — siap untuk token blacklist
- **TokenType guard:** Middleware menolak token dengan `token_type != "access"` — refresh token tidak bisa dipakai di endpoint API

### 7.4 httpclient

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

### 7.5 messaging — Kafka

```go
import "github.com/abdullahPrasetio/wapgo/pkg/messaging/kafka"

// Producer
producer := kafka.NewProducer("localhost:9092", logger)
err := producer.Publish(ctx, kafka.Message{
    Topic: "user.events",
    Key:   []byte("user-123"),
    Value: []byte(`{"event":"created","id":"123"}`),
})
defer producer.Close()

// Consumer — Start() blocks; cancel ctx untuk stop graceful
consumer := kafka.NewConsumer("localhost:9092", "my-service-group", "user.events", logger)
go func() {
    if err := consumer.Start(ctx, func(ctx context.Context, msg kafka.Message) error {
        // proses pesan; return non-nil = offset tidak di-commit (re-delivered)
        return nil
    }); err != nil {
        log.Error().Err(err).Msg("kafka consumer stopped")
    }
}()
defer consumer.Close()

// Health check
kafka.HealthCheck("localhost:9092")  // return func(ctx) string
```

**Resiliency bawaan:**
- Fetch error → exponential backoff 1 s → 30 s (context-aware: shutdown tetap instan)
- `HeartbeatInterval: 3s`, `SessionTimeout: 30s`, `RebalanceTimeout: 30s` — mencegah rebalancing macet
- Internal reader error diredirect ke zerolog via `ErrorLogger`
- `X-Request-ID` dipropagasi otomatis via Kafka header `x-request-id`

### 7.6 messaging — RabbitMQ

> **Penting:** Buat **satu** `Connection` per proses dan bagikan ke semua publisher dan consumer.
> Jangan pernah panggil `NewPublisher`/`NewConsumer` dengan DSN langsung — itu pola lama yang
> menyebabkan satu publisher/consumer = satu koneksi TCP, yang akan crash broker saat ada ratusan consumer.

```go
import "github.com/abdullahPrasetio/wapgo/pkg/messaging/rabbitmq"

// ── 1. SATU koneksi per proses ──────────────────────────────────────────────
conn, err := rabbitmq.NewConnection(cfg.RabbitMQ.DSN, log.Logger)
if err != nil {
    log.Fatal().Err(err).Msg("rabbitmq connection failed")
}
defer conn.Close() // tutup saat shutdown

// ── 2. Publisher — share conn ────────────────────────────────────────────────
pub, err := rabbitmq.NewPublisher(conn, "user.events", log.Logger)
if err != nil { ... }
defer pub.Close() // hanya menutup channel, bukan koneksi

err = pub.Publish(ctx, rabbitmq.Message{
    RoutingKey: "user.created",
    Body:       []byte(`{"id":"123"}`),
})
// Jika channel mati (broker restart), Publish membuka ulang channel sekali
// lalu retry otomatis sebelum mengembalikan error.

// ── 3. Consumer — share conn, Subscribe BLOCKS ───────────────────────────────
consumer := rabbitmq.NewConsumer(conn, "user.events", log.Logger)

go func() {
    // Subscribe blocks sampai ctx dibatalkan.
    // Jika channel/koneksi mati, Subscribe reconnect otomatis dengan
    // exponential backoff (1s → 30s) — tidak perlu restart pod.
    if err := consumer.Subscribe(ctx, "user.events.created", "user.created",
        func(ctx context.Context, msg rabbitmq.Message) error {
            // return non-nil = Nack → pesan ke DLQ otomatis
            return nil
        },
    ); err != nil {
        log.Error().Err(err).Msg("rabbitmq consumer stopped")
    }
}()

// ── 4. Health check (untuk /health endpoint) ─────────────────────────────────
rabbitmq.HealthCheck(cfg.RabbitMQ.DSN)  // return func(ctx) string
// HealthCheck menggunakan koneksi sementara tersendiri — tidak mengganggu conn di atas.
```

**Resiliency bawaan:**
- Satu TCP socket untuk semua consumer + publisher dalam satu proses
- `NewConnection` dial dengan 5 s timeout + 10 s heartbeat AMQP
- `Channel()` auto-reconnect dengan double-checked locking (aman untuk concurrent goroutine)
- `Subscribe` blocking + auto-reconnect channel dengan exponential backoff 1 s → 30 s
- Channel death deteksi via `amqp.NotifyClose` — reaktif, tidak polling
- DLQ (`{queue}.dlq`) dikonfigurasi otomatis via `x-dead-letter-exchange`

### 7.7 observability

```go
import "github.com/abdullahPrasetio/wapgo/pkg/observability"

// Setup provider (di main.go) — environment diambil dari cfg.App.Env ("development"/"production"/dll)
obsProvider, err := observability.New(ctx, &cfg.Observability, cfg.App.Name, version, cfg.App.Env)

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

### 7.8 Worker binary

Worker adalah binary terpisah dari HTTP server. Dibuat via `wapgo make:worker [name]`.

**Kapan pakai worker terpisah?**
- Consumer domain besar dengan logika berat (order processing, payment, notification)
- Scaling consumer secara independen dari API
- Isolasi fault: crash consumer tidak mematikan HTTP server

**Pola standar dalam generated worker:**

```go
// cmd/worker-order/main.go (contoh output make:worker order --broker both)

func main() {
    cfg, _ := config.Load()
    // ... setup logger, observability ...

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // ── Kafka consumer ────────────────────────────────────────────────────
    if cfg.Kafka.Brokers != "" {
        kConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, "orders", log.Logger)
        defer kConsumer.Close()
        go func() {
            kConsumer.Start(ctx, func(ctx context.Context, msg kafka.Message) error {
                // TODO: dispatch ke use case
                return nil
            })
        }()
    }

    // ── RabbitMQ consumer — SATU koneksi per proses ───────────────────────
    if cfg.RabbitMQ.DSN != "" {
        rmqConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.DSN, log.Logger)
        if err != nil { log.Fatal().Err(err).Msg("rabbitmq failed") }
        defer rmqConn.Close()

        rConsumer := rabbitmq.NewConsumer(rmqConn, cfg.RabbitMQ.Exchange, log.Logger)
        go func() {
            rConsumer.Subscribe(ctx, "orders", "order.*", func(ctx context.Context, msg rabbitmq.Message) error {
                // TODO: dispatch ke use case
                return nil
            })
        }()
    }

    log.Info().Msg("worker ready — waiting for messages")
    <-ctx.Done() // tunggu SIGTERM / SIGINT
    log.Info().Msg("shutdown complete")
}
```

**Multiple worker untuk satu service:**

```
cmd/
├── api/main.go           ← HTTP server
├── worker-order/main.go  ← order consumer
└── worker-notif/main.go  ← notification consumer
```

Makefile yang di-generate otomatis:

```makefile
run-worker-order:
    go run ./cmd/worker-order

build-worker-order:
    go build -o bin/worker-order ./cmd/worker-order
```

---

### 7.9 notification — SMTP (email)

> Add-on opsional. Import hanya jika service perlu kirim email.

```go
import (
    "github.com/abdullahPrasetio/wapgo/pkg/notification/email"
)

smtpTimeout, _ := time.ParseDuration(cfg.Notification.SMTP.Timeout) // "10s" default

mailer := email.NewSMTPMailer(email.Config{
    Host:     cfg.Notification.SMTP.Host,
    Port:     cfg.Notification.SMTP.Port,    // 587 default
    Username: cfg.Notification.SMTP.Username,
    Password: cfg.Notification.SMTP.Password,
    From:     cfg.Notification.SMTP.From,
    Timeout:  smtpTimeout,                   // zero → NewSMTPMailer defaults to 10s
}, logger)

err := mailer.Send(ctx, email.Message{
    To:      []string{"user@example.com"},
    Subject: "Order Confirmed",
    Body:    "<h1>Pesanan #123 dikonfirmasi</h1>",
    IsHTML:  true,
})
```

**Integrasi otomatis:**
- Setiap `Send` mencatat span OTel `notification.email.send`
- Entry `ThirdParty{name:"smtp"}` otomatis muncul di `thirdparty.log` dan embed di `api.log` / `consumer.log`
- Koneksi TCP baru per-Send — stateless, aman concurrent

**Health check:**

```go
health.Register("smtp", email.HealthCheck(cfg.Notification.SMTP.Host, cfg.Notification.SMTP.Port))
```

---

### 7.10 notification — Firebase FCM (push notification)

> Add-on opsional. Import hanya jika service perlu kirim push notification.
> Tidak membutuhkan Firebase Admin SDK — auth dilakukan via FCM v1 HTTP API
> dengan service account JWT (RS256), menggunakan `golang-jwt/jwt/v5` yang sudah ada di go.mod.

```go
import (
    "github.com/abdullahPrasetio/wapgo/pkg/notification/firebase"
)

pusher, err := firebase.NewFCMClient(cfg.Notification.Firebase.CredentialsJSON, logger)
if err != nil {
    log.Fatal().Err(err).Msg("firebase init failed")
}

// Kirim ke satu device
err = pusher.Send(ctx, firebase.Message{
    Token: "device-registration-token",
    Title: "Pesanan Dikirim",
    Body:  "Paket Anda sedang dalam perjalanan",
    Data:  map[string]string{"order_id": "123", "type": "order_shipped"},
})

// Kirim ke topic
err = pusher.Send(ctx, firebase.Message{
    Topic: "promo",
    Title: "Flash Sale!",
    Body:  "Diskon 50% hanya 2 jam",
})
```

**Integrasi otomatis:**
- Access token di-cache ~1 jam — tidak re-fetch setiap request
- Setiap `Send` mencatat span OTel `notification.firebase.send`
- Entry `ThirdParty{name:"firebase-fcm"}` otomatis muncul di `thirdparty.log` dan embed di `api.log`
- Device token di-mask di log (`abcd***wxyz`) — tidak bocor ke log file

**Health check:**

```go
health.Register("firebase", firebase.HealthCheck(cfg.Notification.Firebase.CredentialsJSON))
```

**Cara dapat `FIREBASE_CREDENTIALS_JSON`:**
1. Firebase Console → Project Settings → Service Accounts
2. Klik "Generate new private key" → download file JSON
3. Set ENV: `FIREBASE_CREDENTIALS_JSON=$(cat path/to/key.json)` atau paste konten JSON ke Kubernetes Secret

**Contoh struktur JSON service account key:**

```json
{
  "type": "service_account",
  "project_id": "my-app-12345",
  "private_key_id": "a1b2c3d4e5f6a1b2c3d4e5f6",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE...(base64)...\n-----END PRIVATE KEY-----\n",
  "client_email": "firebase-adminsdk-xxxxx@my-app-12345.iam.gserviceaccount.com",
  "client_id": "123456789012345678901",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk%40my-app-12345.iam.gserviceaccount.com",
  "universe_domain": "googleapis.com"
}
```

Field yang **wajib** ada: `project_id`, `private_key`, `client_email`, `token_uri`.
Field lain diabaikan oleh `pkg/notification/firebase`.

**Set ke ENV (3 cara):**

```bash
# Cara 1 — export dari file (lokal/CI)
export FIREBASE_CREDENTIALS_JSON=$(cat serviceAccountKey.json)

# Cara 2 — inline di .env (satu baris, \n harus di-escape manual)
# FIREBASE_CREDENTIALS_JSON={"type":"service_account","project_id":"my-app-12345",...}

# Cara 3 — Kubernetes Secret (production)
kubectl create secret generic firebase-creds \
  --from-file=FIREBASE_CREDENTIALS_JSON=serviceAccountKey.json
```

---

### 7.11 auth/google (Google OAuth2)

> Add-on opsional. Aktifkan dengan `wapgo add google-auth`. Membutuhkan Redis (state CSRF) dan user repository sudah tersedia di project.

```go
import (
    googleauth "github.com/abdullahPrasetio/wapgo/pkg/auth/google"
)

// Inisialisasi provider (di main.go)
provider := googleauth.New(googleauth.Config{
    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
    RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
})

// Buat handler dan daftarkan route
googleHandler := handler.NewGoogleAuthHandler(provider, userRepo, &jwtCfg, redisClient)
route.RegisterGoogleAuthRoutes(api, googleHandler)
```

**Endpoint yang dihasilkan:**

| Method | Path | Fungsi |
|---|---|---|
| `GET` | `/auth/google` | Redirect ke Google consent page |
| `GET` | `/auth/google/callback` | Terima code dari Google, upsert user, return JWT |

**Alur lengkap:**

```
Browser
  → GET /auth/google
    ← 307 redirect ke accounts.google.com/o/oauth2/...?state=<random>

Google consent page
  → user klik "Allow"
  → GET /auth/google/callback?code=xxx&state=yyy

Server
  → verify state di Redis (CSRF, one-time, TTL 5m)
  → exchange code dengan Google → UserInfo {id, email, name, picture}
  → FindByEmail → user ada? return user : create user baru (password kosong)
  → auth.Sign(user.ID, roles, jwtCfg) → JWT
  ← 200 { "access_token": "...", "user": { id, name, email } }
```

**Security bawaan:**
- State CSRF 128-bit random, simpan di Redis dengan TTL 5 menit — one-time (di-delete saat verify)
- `GetDel` Redis bersifat atomik — tidak ada race condition verifikasi ulang
- User yang login via Google tidak bisa login dengan email+password (password field kosong, bcrypt tidak cocok)
- OAuth2 scope minimal: `openid email profile`

**Contoh response sukses:**

```json
{
  "status": true,
  "message": "google login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Budi Santoso",
      "email": "budi@gmail.com"
    }
  }
}
```

**Instalasi ke project yang sudah ada:**

```bash
wapgo add google-auth
go get golang.org/x/oauth2
```

Jika project sudah dibuat sebelum versi ini, tambahkan `FindByEmail` manual ke:
- `internal/domain/repository/user_repository.go` — interface
- `internal/repository/db/user_repository.go` — implementasi GORM

(Project baru dari `wapgo new` sudah include keduanya otomatis.)

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
| **JWT hardening** | Algo di-pin HS256, validasi `exp`/`iat`/`iss`/`aud`, `alg:none` ditolak, JTI per-token, `token_type` guard (refresh ditolak di endpoint API) |
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
| **v0.7** | Test coverage ≥ 80% semua layer, release workflow CI/CD | ✅ |
| **v0.8** | `make:migration`, `pkg/response.Paginated`, skeleton README | ✅ |
| **v0.9** | CLI wizard interaktif, `wapgo add <feature>`, conditional scaffolding | ✅ |
| **v0.10** | OTel → Elastic APM bridge (`apmotel`), `none` provider, `StartSpan` helper, skeleton compile | ✅ |
| **v0.11** | 4 log sinks, `pkg/journal` (dual-write), `AccessLog` middleware, consumer journal, Filebeat example | ✅ |
| **v1.1** | Swagger UI, welcome page, `make:test` (usecase + handler layer), `wapgo upgrade` | ✅ |
| **v1.2** | **Connection management**: RabbitMQ shared `Connection`, blocking `Subscribe` + auto-reconnect; Redis pool config (6 ENV vars); Kafka backoff + session config; `make:worker` | ✅ |
| **v1.3** | **Auth hardening + Notification add-ons**: JTI per-token, `SignRefresh()`, `token_type` guard (refresh ditolak di API); `pkg/notification/email` (SMTP OTel); `pkg/notification/firebase` (FCM v1 OTel, token cache); `wapgo add email/firebase`; skeleton sync penuh (config, kafka OTel, observability env) | ✅ |
| **v1.4** | **Google OAuth2**: `pkg/auth/google` (Provider, AuthURL, Exchange); `wapgo add google-auth` scaffold handler + route; Redis CSRF state (one-time, TTL 5m); auto-register user baru; `FindByEmail` di skeleton UserRepository | ✅ |
| **v1.5** | **Security hardening + Kafka tuning**: `upsertUser` bedakan `ErrRecordNotFound` dari DB error; `FindByEmail`/`ExistsByEmail` case-insensitive (`LOWER`); Google-auth JWT default role `"user"`; `io.LimitReader` di `Exchange()`; Kafka `ConsumerConfig` struct + 3 ENV baru (`KAFKA_HEARTBEAT_INTERVAL`, `KAFKA_SESSION_TIMEOUT`, `KAFKA_REBALANCE_TIMEOUT`) | ✅ |

Coverage semua paket > 80%. `go build ./...` dan `go vet ./...` bersih.

---

## Contoh `.env` Lengkap

```dotenv
# App
APP_NAME=my-service
APP_ENV=development
APP_PORT=8080
APP_CORS_ALLOWED_ORIGINS=http://localhost:3000

# Database
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASSWORD=secret
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFE=5m
DB_AUTO_MIGRATE=true
DB_SSL_MODE=disable

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=20        # max koneksi ke Redis
REDIS_MIN_IDLE_CONNS=5    # koneksi idle minimal
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s
REDIS_MAX_RETRIES=3

# JWT (wajib ≥ 32 karakter)
JWT_SECRET=supersecretkey-that-is-at-least-32-chars
JWT_EXPIRY=24h
JWT_ISSUER=my-service
JWT_AUDIENCE=my-client

# Kafka (opsional — biarkan kosong jika tidak dipakai)
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=my-service-group
# KAFKA_HEARTBEAT_INTERVAL=3s   # default 3s
# KAFKA_SESSION_TIMEOUT=30s     # default 30s
# KAFKA_REBALANCE_TIMEOUT=30s   # default 30s

# RabbitMQ (opsional — biarkan kosong jika tidak dipakai)
RABBITMQ_DSN=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=my-service-exchange

# Observability — pilih salah satu

# Opsi 1: Elastic APM (Kibana) — default
OBSERVABILITY_PROVIDER=elastic_apm
ELASTIC_APM_SERVER_URL=http://localhost:8200
ELASTIC_APM_SERVICE_NAME=my-service
ELASTIC_APM_SECRET_TOKEN=
ELASTIC_APM_ENVIRONMENT=development
ELASTIC_APM_ACTIVE=true

# Opsi 2: OTel (Jaeger / Grafana Tempo)
# OBSERVABILITY_PROVIDER=otel
# OTEL_TRACING_ENABLED=true
# OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Opsi 3: matikan tracing
# OBSERVABILITY_PROVIDER=none

# Logging (4 structured log sinks → logs/)
LOG_LEVEL=info
LOG_DIR=logs
LOG_ROTATION=size         # size | daily
LOG_MAX_AGE_DAYS=30
LOG_HTTP_BODIES=false     # true = catat body request/response di api.log
LOG_BODY_MAX_BYTES=16384

# Health check
HEALTH_PROBE_TIMEOUT=2s      # timeout per probe (DB ping, Redis ping, dll)

# ── Notification add-ons (opsional — hapus atau biarkan kosong jika tidak dipakai) ──

# SMTP (pkg/notification/email)
SMTP_HOST=smtp.example.com
SMTP_PORT=587              # 587=STARTTLS, 465=implicit TLS, 25=plain
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=
SMTP_FROM=noreply@example.com
SMTP_TIMEOUT=10s

# Firebase FCM (pkg/notification/firebase)
# Isi dengan konten JSON dari Firebase Console → Project Settings → Service Accounts
FIREBASE_CREDENTIALS_JSON=

# Google OAuth2 (pkg/auth/google — wapgo add google-auth)
# Dari Google Cloud Console → APIs & Services → Credentials → OAuth 2.0 Client IDs
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
```

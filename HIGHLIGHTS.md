# Keunggulan wapgo

> Ringkasan kelebihan teknis framework ini dibanding membangun service dari nol atau memakai boilerplate generik.

---

## 1. Clean Architecture yang Benar-Benar Dijaga di Code Level

Kebanyakan proyek Go mengaku pakai Clean Architecture tapi hanya di struktur folder. wapgo menegakkannya lewat kode:

- Interface `UserRepository` dan `Cacher` didefinisikan di **domain layer** (`internal/domain/repository/`) — bukan di package implementasi
- Struct implementasi (`userRepository`) sengaja **unexported** sehingga caller di luar package tidak bisa bergantung ke tipe konkret
- Constructor selalu **return interface**, bukan pointer ke struct

```go
// Caller hanya tahu kontrak ini, bukan bahwa di baliknya ada Postgres
func NewUserRepository(db *gorm.DB) domainrepo.UserRepository
```

Efeknya: kalau suatu hari ingin ganti Postgres ke MySQL, cukup tukar satu file implementasi — usecase dan handler tidak berubah sama sekali.

---

## 2. Testability by Design — Semua Layer Bisa Di-mock

Karena semua koneksi antar layer memakai interface:

| Layer | Depend ke |
|---|---|
| Handler | `usecase.UserUseCase` (interface) |
| Usecase | `repository.UserRepository` (interface) |
| Repository | `*gorm.DB` (bisa diganti test DB) |

Unit test bisa ditulis tanpa database, tanpa Redis, tanpa Kafka benar-benar nyala:

```go
uc := &mockUserUseCase{user: &entity.User{...}}
handler := NewUserHandler(uc, validator.New())
// test HTTP response tanpa hit database
```

Ini artinya test suite berjalan dalam milidetik, bukan menit.

---

## 3. Structured Logging End-to-End — 4 Sink + Request Journal

wapgo menulis log ke **empat file terstruktur** sekaligus, semuanya JSON line-delimited:

| File | Isi |
|---|---|
| `api.log` | 1 baris per request HTTP — full req/resp headers + body + `thirdparty[]` + `trace[]` |
| `consumer.log` | 1 baris per pesan Kafka/RabbitMQ — full payload + `thirdparty[]` + `trace[]` |
| `thirdparty.log` | 1 baris per hit thirdparty — full outbound req/resp dengan latency |
| `trace.log` | 1 baris per custom trace yang diinject dari usecase |

**Dual-write**: thirdparty & trace ditulis ke file tersendiri **sekaligus** diembed di dalam record induk `api`/`consumer`. Tidak perlu korelasi manual — satu record `api.log` sudah memuat semua konteks.

```go
// Di usecase: inject custom trace
journal.FromContext(ctx).AddTrace("fraud-score", map[string]any{"score": 0.92})

// Hasilnya di api.log:
// {"method":"POST","path":"/payment","thirdparty":[{"url":"https://bank.api/charge","status":200}],
//  "trace":[{"name":"fraud-score","data":{"score":0.92}}],"latency_ms":43}
```

Semua header sensitif diredaksi otomatis (Authorization, Cookie, Set-Cookie). Body size dicap via `LOG_BODY_MAX_BYTES`. File siap dikirim ke Elasticsearch via Filebeat (contoh `deploy/filebeat.yml` sudah tersedia di skeleton).

---

## 4. Observability Production-Grade — Dua Provider, Satu Interface

Bukan sekadar "ada logging". wapgo menyediakan tiga lapisan observability sekaligus:

| Lapisan | Tool | Keterangan |
|---|---|---|
| Distributed tracing | OpenTelemetry **atau** Elastic APM | Dipilih via `OBSERVABILITY_PROVIDER` env |
| Query tracing | GORM + Redis instrumentation | Setiap query otomatis jadi child span |
| RED metrics | Prometheus | Rate, Error, Duration per endpoint |

Satu env var cukup untuk pindah dari OTel ke Elastic APM — tidak ada perubahan kode aplikasi.

```bash
OBSERVABILITY_PROVIDER=otel     # kirim trace ke OTLP collector
OBSERVABILITY_PROVIDER=elastic  # kirim trace ke Elastic APM server
```

---

## 5. CLI Code Generator — Scaffold Lengkap dalam Satu Perintah

Framework lain sering menyediakan struktur folder saja. wapgo menyertakan CLI yang generate **semua layer sekaligus**:

```bash
wapgo make:all User
```

Menghasilkan dalam hitungan detik:
- `internal/domain/entity/user.go`
- `internal/domain/repository/user_repository.go` (interface)
- `internal/repository/postgres/user_repository.go` (implementasi)
- `internal/usecase/user_usecase.go` (interface + impl + DTOs)
- `internal/delivery/http/handler/user_handler.go`
- `internal/delivery/http/route/user_route.go`

Developer langsung bisa fokus ke business logic, bukan konfigurasi boilerplate.

---

## 6. Security Defaults — Aktif Tanpa Konfigurasi

Semua middleware keamanan aktif by default saat `Setup()` dipanggil:

| Middleware | Fungsi |
|---|---|
| `SecurityHeaders` | HSTS, X-Frame-Options, X-Content-Type-Options, CSP |
| `RateLimiter` | Batasi jumlah request per IP |
| `CORS` | Whitelist origin via env `CORS_ALLOWED_ORIGINS` |
| `Recover` | Panic tidak crash server, dicatat ke log |
| `prodGuard` | Endpoint `/metrics` otomatis 404 di production |

JWT middleware siap pakai dengan `ExtractClaims` helper — tidak perlu parse token manual.

---

## 7. Messaging dengan Connection Management Production-Grade

### Kafka — Backoff + Session Config
Consumer Kafka memiliki exponential backoff (1 s → 30 s) pada fetch error, sehingga tidak membanjiri broker saat terjadi gangguan sementara. Shutdown tetap instan karena backoff menggunakan `select` pada `ctx.Done`. `HeartbeatInterval`, `SessionTimeout`, dan `RebalanceTimeout` dikonfigurasi untuk mencegah rebalancing macet.

### RabbitMQ — Connection Pool per Proses

**Sebelum (pola lama yang salah):**
```go
// Setiap baris ini membuka satu koneksi TCP → 50 consumer = 50 koneksi = crash
pub, _ := rabbitmq.NewPublisher(dsn, exchange, logger)
cons, _ := rabbitmq.NewConsumer(dsn, exchange, logger)
```

**Sekarang (shared connection):**
```go
// Satu koneksi TCP untuk semua publisher + consumer dalam proses
conn, _ := rabbitmq.NewConnection(dsn, logger)
defer conn.Close()

pub, _ := rabbitmq.NewPublisher(conn, exchange, logger)
consumer := rabbitmq.NewConsumer(conn, exchange, logger)
go consumer.Subscribe(ctx, "queue", "rk.*", handler)  // auto-reconnect
```

`Subscribe` bersifat blocking dan auto-reconnect ketika channel atau koneksi mati — pod tidak perlu direstart saat broker restart. Channel death dideteksi via `amqp.NotifyClose`, bukan polling.

### Redis — Pool yang Dikonfigurasi

```bash
REDIS_POOL_SIZE=20        # default 10 → ditingkatkan ke 20
REDIS_MIN_IDLE_CONNS=5    # selalu ada 5 koneksi siap
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_MAX_RETRIES=3
```

### Worker Binary (`wapgo make:worker`)

```bash
wapgo make:worker order --broker rabbitmq
# → cmd/worker-order/main.go + Makefile targets: run-worker-order, build-worker-order
```

Consumer bisa dijalankan sebagai binary terpisah dari HTTP server, dengan scaling dan fault isolation independen.

---

## 8. ENV-First Config — Siap Kubernetes/OpenShift dari Hari Pertama

Tidak ada config file yang perlu di-mount ke container. Semua konfigurasi dibaca dari environment variable:

```bash
DB_DSN=postgres://...
REDIS_URL=redis://...
OBSERVABILITY_PROVIDER=otel
APP_ENV=production
```

Ini selaras langsung dengan Kubernetes `ConfigMap` + `Secret`, Helm values, dan OpenShift deployment config — tanpa adapter tambahan.

---

## 9. Multi-module Monorepo yang Bersih

CLI dan framework core hidup di module terpisah, dihubungkan via `go.work`:

```
wapgo/           → github.com/abdullahPrasetio/wapgo
└── cli/         → github.com/abdullahPrasetio/wapgo/cli
```

Keuntungannya:
- **Binary framework** dan **binary CLI** bisa di-release dan di-version secara independent
- User yang hanya butuh pakai framework sebagai library tidak terpaksa download dependency CLI
- `go install .../cli/cmd@latest` install hanya CLI, tidak seluruh framework

---

## Perbandingan Singkat

| Aspek | Boilerplate generik | wapgo |
|---|---|---|
| Layer separation | Folder saja | Dijaga di code level (interface + unexported) |
| Testability | Perlu setup manual | Interface-first, mock langsung bisa |
| Observability | Tidak ada / tambah sendiri | OTel + Elastic APM (bridge) + Prometheus built-in |
| Logging | `log.Printf` / satu file | 4 sink terstruktur JSON, dual-write, rotation |
| Request Journal | Tidak ada | Thirdparty[] + trace[] otomatis embed di record induk |
| Code generation | Tidak ada | CLI wizard interaktif + scaffold semua layer |
| Security | Tambah sendiri | Default aktif semua + header redaction bawaan |
| Messaging | Tambah sendiri | Kafka + RabbitMQ: shared connection, auto-reconnect, backoff, DLQ, journal |
| Worker binary | Tambah sendiri | `make:worker` scaffold multi-domain consumer binary terpisah |
| Redis pool | Default bare-minimum | Pool size, idle conns, timeout, retry — semua via ENV |
| Container-ready | Perlu konfigurasi | ENV-first by design, Filebeat example tersedia |

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

## 3. Observability Production-Grade — Dua Provider, Satu Interface

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

## 4. CLI Code Generator — Scaffold Lengkap dalam Satu Perintah

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

## 5. Security Defaults — Aktif Tanpa Konfigurasi

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

## 6. Messaging dengan API yang Konsisten

Kafka dan RabbitMQ punya interface yang seragam:

```go
// Kafka
producer := kafka.NewProducer(brokers, logger)
producer.Publish(ctx, kafka.Message{Topic: "...", Value: payload})

// RabbitMQ
pub, _ := rabbitmq.NewPublisher(dsn, exchange, logger)
pub.Publish(ctx, rabbitmq.Message{RoutingKey: "...", Body: payload})
```

Keduanya menyediakan `HealthCheck()` yang kompatibel dengan `/health` endpoint. Pindah provider messaging tidak mengubah cara usecase berinteraksi.

---

## 7. ENV-First Config — Siap Kubernetes/OpenShift dari Hari Pertama

Tidak ada config file yang perlu di-mount ke container. Semua konfigurasi dibaca dari environment variable:

```bash
DB_DSN=postgres://...
REDIS_URL=redis://...
OBSERVABILITY_PROVIDER=otel
APP_ENV=production
```

Ini selaras langsung dengan Kubernetes `ConfigMap` + `Secret`, Helm values, dan OpenShift deployment config — tanpa adapter tambahan.

---

## 8. Multi-module Monorepo yang Bersih

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
| Observability | Tidak ada / tambah sendiri | OTel + Elastic APM + Prometheus built-in |
| Code generation | Tidak ada | CLI scaffold semua layer sekaligus |
| Security | Tambah sendiri | Default aktif semua |
| Messaging | Tambah sendiri | Kafka + RabbitMQ dengan API konsisten |
| Container-ready | Perlu konfigurasi | ENV-first by design |

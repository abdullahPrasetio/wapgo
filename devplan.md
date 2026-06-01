# Development Plan — **wapgo**

> **wapgo** — *Web API Platform for Go*
> Boilerplate & framework microservice Go yang production-ready: Clean Architecture, ENV-first (siap OpenShift/Kubernetes), dengan CLI yang bisa **men-scaffold project** sendiri sekaligus **generate kode** ala Laravel artisan.

Dokumen ini adalah peta jalan pengembangan framework. Acuan spesifikasi: [`prompt-go-microservice.md`](prompt-go-microservice.md) & [`README.md`](README.md).

---

## 1. Identitas Framework

**Nama:** `wapgo` (satu kata, tanpa tanda hubung — agar valid sebagai nama Go package & module).
**Arti:** *Web API Platform for Go*.

**Visi:** Sebuah boilerplate microservice yang bisa langsung dipakai produksi — Clean Architecture, konfigurasi ENV-first (12-factor), observability & request tracing bawaan, dan *developer experience* generator-first. Cukup `wapgo new my-service`, project lengkap siap jalan; tambah domain baru cukup `wapgo make:all <name>`.

**Branding & artefak:**

| Item | Nilai |
|---|---|
| Module path | `github.com/abdullahPrasetio/wapgo` |
| Repository | https://github.com/abdullahPrasetio/wapgo |
| Binary CLI | `wapgo` (installer + generator) |
| Binary API | `wapgo-api` (entrypoint service) |
| Default `APP_NAME` | `wapgo-service` |
| Slogan | *Web API Platform for Go* |

---

## 2. Prinsip Desain

- **Clean Architecture** — `Handler → Usecase → Repository/ExternalService (interface)`. Tidak ada import konkret lintas layer; semua lewat interface.
- **ENV-first config** (Viper) — prioritas `ENV → config.yaml → default`. Wajib agar terbaca dari OpenShift ConfigMap/Secret.
- **Dependency Injection manual** via constructor di `cmd/api/main.go` — eksplisit, mudah dilacak, tanpa magic.
- **Observability & tracing bawaan** — `X-Request-ID` di-generate middleware, diteruskan ke log, outgoing HTTP, header Kafka, dan properti RabbitMQ.
- **Generator-first DX** — CLI men-scaffold project dan generate setiap layer dengan pola konsisten.
- **Quality built-in** — setiap komponen punya **example** + **unit test (coverage > 80%)**; bukan tambahan belakangan.
- **Secure by default** — setiap fitur aman tanpa konfigurasi tambahan: security headers, rate limit, validasi input, TLS verify, secret tak pernah bocor ke log. Lihat [§7 Security](#7-security--syarat-lolos--aman). **Tidak ada PR yang lolos bila gerbang keamanan gagal.**

---

## 3. Roadmap Berfase

> **Aturan lintas fase:** Sebuah fase **belum selesai** sebelum setiap komponennya memiliki **example yang jalan** + **unit test** (**coverage paket > 80%**, `go test ./... -cover`) **dan lolos gerbang keamanan** (§7: `gosec`, `govulncheck`, `gitleaks` bersih + checklist security fase terpenuhi). Testing, example, dan security melekat di tiap deliverable — bukan fase terpisah.

### Fase v0.1 — Core Skeleton ✅ SELESAI (2026-06-01)

**Tujuan:** Service minimal yang bisa `make run`, punya `/health`, dan CRUD `/users` ke Postgres.

- [x] `go.mod` (module `wapgo`, Go 1.25.0), `.gitignore`, `.env.example`, `Makefile`, struktur folder lengkap.
- [x] `config/` — Viper loader (ENV → yaml → default), explicit `BindEnv` untuk 25+ variabel, `service_urls.go`.
- [x] `pkg/logger/` — zerolog (JSON di prod / console di dev), rotasi file via lumberjack, request-id dari context.
- [x] `pkg/response/` (struct response terpusat) + `pkg/validator/`.
- [x] DB: inisialisasi GORM (postgres default, switch ke mysql via `DB_DRIVER`), connection pool, auto-migrate.
- [x] HTTP: Fiber app + middleware stack: `recover → request-id → security-headers (helmet) → rate-limit → logger → CORS`.
- [x] **Security baseline:** HSTS, XSS, nosniff, X-Frame=DENY, CSP; rate limit per-IP; body limit 4MB; CORS allowlist; recover tanpa stack trace bocor; validasi DTO; log redaksi field sensitif.
- [x] Reference domain **user** lengkap: entity → repository interface → postgres impl → usecase → handler → route. GORM parameterized.
- [x] `/health` (cek DB + Redis), graceful shutdown SIGTERM/SIGINT (30s).
- [x] `cmd/api/main.go` wiring lengkap via constructor.

**Hasil coverage:** config 88.6% · logger 90.5% · response 100% · validator 82.6% · usecase 88.3% · handler 96.6% — **total 90.7%** ✅

**DoD:** ✅ `go build ./...` + `go vet ./...` bersih; test hijau; coverage > 80% semua paket; security headers & rate limit aktif.

### Fase v0.2 — Cache & Messaging ✅ SELESAI (2026-06-01)

**Tujuan:** Tambah Redis cache + Kafka & RabbitMQ yang independen.

- [x] `internal/repository/redis/cache.go` — `Cacher` interface + `RedisCacher` (JSON marshal/unmarshal, TTL, namespace prefix). Diuji dengan miniredis.
- [x] `pkg/messaging/kafka/producer.go` — `Producer` dengan mockable `writer` interface; request-id propagasi via header `x-request-id`; fallback ke context.
- [x] `pkg/messaging/kafka/consumer.go` — `Consumer` group dengan graceful shutdown via context cancel; `HealthCheck(brokers)` probe; injectable dialer.
- [x] `pkg/messaging/rabbitmq/publisher.go` — `Publisher` dengan `publishChan` interface; topic exchange; persistent delivery; request-id di header AMQP.
- [x] `pkg/messaging/rabbitmq/consumer.go` — `Consumer` dengan DLQ (`x-dead-letter-exchange`); `HandlerFunc`; Ack/Nack; `HealthCheck(dsn)` probe.
- [x] `/health` diperluas: `AddChecker(name, fn)` fluent API; Kafka + RabbitMQ dilaporkan (atau `"not_configured"` bila ENV kosong).
- [x] `docker-compose.yml`: postgres 16, mysql 8, redis 7, zookeeper + kafka (Confluent 7.6), rabbitmq 3.13-management. Semua dengan healthcheck.
- [x] `cmd/api/main.go` diperbarui: wire Kafka + RabbitMQ health checker dari config.

**Hasil coverage:** redis 91.7% · kafka 86.0% · rabbitmq 93.3% · handler 97.0% — **semua > 80%** ✅

**DoD:** ✅ `go build ./...` + `go vet ./...` bersih; semua test hijau; coverage > 80% semua paket; `/health` menampilkan status kafka & rabbitmq.

### Fase v0.3 — Inter-service HTTP Client (Resilience) ✅ SELESAI (2026-06-01)

**Tujuan:** Komunikasi antar-service yang tahan gangguan.

- [x] `pkg/httpclient/base_client.go` — wrapper net/http; inject `X-Request-ID` & `Authorization` dari context; timeout konfigurabel (default 5s).
- [x] `pkg/httpclient/middleware.go` — retry (max 3, exponential backoff, hanya 5xx & network error); timeout context-aware (hormati deadline caller); circuit breaker (open setelah 5 gagal beruntun, half-open setelah 30s).
- [x] `pkg/httpclient/user_client.go` — implementasi interface `internal/domain/service/external_user.go` (`GetUser(ctx, id)`), URL dari `USER_SERVICE_URL`.
- [x] **Security:** TLS verify **ON** by default (`InsecureSkipVerify=false`, `MinVersion=TLS1.2`); proteksi SSRF (validasi host tujuan terhadap allowlist, tolak redirect ke alamat internal/loopback/link-local); timeout & batas ukuran response (default 10MB) untuk cegah resource exhaustion.

**Hasil coverage:** httpclient 94.3% — **> 80%** ✅

**DoD:** ✅ `go build ./...` + `go vet ./...` bersih; semua test hijau; coverage 94.3%; SSRF guard + retry + circuit breaker teruji dengan mock transport & `httptest`.

### Fase v0.4 — CLI `wapgo` (Installer + Generator) ✅ SELESAI (2026-06-01)

**Tujuan:** CLI yang **memasang/scaffold framework** sekaligus **generate kode**.

- [x] `cli/cmd/main.go` + `cli/commands/` (Cobra), binary bernama `wapgo`. CLI adalah **Go module terpisah** (`github.com/abdullahPrasetio/wapgo/cli`) dalam monorepo yang sama — framework bebas dari Cobra. Dev lokal pakai `go.work`.

**Installer / scaffolder:**
- [x] `wapgo new <project>` — buat project baru lengkap dari template (folder, `go.mod` dengan module path kustom, semua file core v0.1–v0.3, reference domain user). Flag: `--module github.com/me/svc`, `--db postgres|mysql`.
- [x] Template seluruh skeleton di-*embed* ke binary via `//go:embed all:templates` (`cli/generator/templates/`) → `wapgo new` jalan tanpa akses jaringan.
- [x] `wapgo version`.
- [ ] Distribusi: `go install github.com/abdullahPrasetio/wapgo/cli/cmd@latest` (binary `wapgo`) + `install.sh` untuk unduh rilis biner.

**Generator `make:*` (di dalam project):**
- [x] `make:model`, `make:repo`, `make:usecase`, `make:controller`, `make:route`, `make:client`, `make:all`.
- [x] Template-based codegen mengikuti pola reference user; argumen `<name>` snake_case → struct/interface/method. Delimiter `[[` `]]` (no conflict dengan Go code).
- [x] `make:all <name>` jalankan generator berurutan + cetak ringkasan file yang dibuat.

**Hasil coverage:** generator 86.6% — **> 80%** ✅

**DoD:** ✅ `wapgo new shop && cd shop && go mod tidy && go vet ./...` sukses; `wapgo make:all product` menghasilkan 8 domain file dengan module path `github.com/me/shop` langsung compile; coverage 86.6%.

### Fase v0.5 — Peningkatan "Hebat" ✅ SELESAI (2026-06-01)

**Tujuan:** Naikkan kelas dari boilerplate ke framework production-grade.

- [x] **Auth:** `pkg/auth/` — `Sign`/`Verify` (HS256), `Claims` struct, `Middleware` (Bearer token), `RequireRole` (RBAC). Hardening: algo di-pin (`WithValidMethods`), validasi `exp`/`iat`/`iss`/`aud`, secret ≥32 byte wajib, `alg:none` ditolak. Claims di-propagasi ke Fiber Locals via `GetClaims(c)`.
- [x] **Observability:** `pkg/observability/` — Prometheus RED metrics (`wapgo_http_requests_total`, `wapgo_http_request_duration_seconds`) + `MetricsMiddleware` + `MetricsHandler`; OpenTelemetry `SetupTracing` (OTLP HTTP atau stdout exporter), `TracingMiddleware` (W3C TraceContext propagation, span per request), `TraceContext(c)` helper.
- [x] **Trace propagation:** `httpclient.Do` inject W3C header ke outgoing HTTP; Kafka producer inject via `kafkaHeaderCarrier`; RabbitMQ publisher inject via `amqpTableCarrier`.
- [x] **Security:** `/metrics` dijaga `prodGuard` — 404 di `APP_ENV=production`; OTLP endpoint dikonfigurasi via ENV.

**Hasil coverage:** auth 92.0% · observability 82.1% · kafka 91.0% · rabbitmq 84.3% · httpclient 94.6% — **semua > 80%** ✅

**DoD:** ✅ `go build ./...` + `go vet ./...` bersih; semua test hijau; coverage > 80% semua paket; JWT lolos uji `alg:none`, expired, wrong iss/aud, tampered signature; trace propagasi W3C terverifikasi; `/metrics` 404 di production.

### Fase v0.6 — Observability Provider (OTel | Elastic APM) ✅ SELESAI (2026-06-01)

**Tujuan:** Full end-to-end observability yang bisa dipilih: OpenTelemetry atau Elastic APM — dari request masuk hingga DB, Redis, HTTP client — semuanya ter-track di Kibana.

- [x] **`Provider` interface** (`pkg/observability/provider.go`) — `HTTPMiddleware`, `InstrumentGORM`, `InstrumentRedis`, `WrapTransport`, `Shutdown`. Semua layer tercover oleh satu abstraksi.
- [x] **OTel Provider** (`pkg/observability/otel_provider.go`) — refactor dari v0.5; GORM via `otelgorm.NewPlugin()`; Redis via `redisotel.InstrumentTracing()`; HTTP client via `otelhttp.NewTransport()`; HTTP server via OTel `TracingMiddleware`.
- [x] **Elastic APM Provider** (`pkg/observability/elastic_provider.go`) — `apmfiber.Middleware()` (server span); GORM via custom plugin callback (before/after query/create/update/delete/row/raw); Redis via custom hook (`ProcessHook` + `ProcessPipelineHook`); HTTP client via `apmhttp.WrapRoundTripper()`. Semua config agent dari ENV (`ELASTIC_APM_SERVER_URL`, `ELASTIC_APM_SERVICE_NAME`, `ELASTIC_APM_SECRET_TOKEN`, `ELASTIC_APM_ENVIRONMENT`, `ELASTIC_APM_ACTIVE`).
- [x] **Factory** (`pkg/observability/setup.go`) — `New(ctx, cfg, serviceName, version)` memilih provider berdasarkan `OBSERVABILITY_PROVIDER`.
- [x] **Config** — tambah `observability.provider` (ENV: `OBSERVABILITY_PROVIDER`); default `"otel"`.
- [x] **`TraceContext(c)`** diperbarui — fallback ke `c.UserContext()` sehingga bekerja untuk kedua provider.
- [x] **httpclient** — `Options.TransportWrapper` field baru; `New()` membungkus transport chain dengan wrapper (OTel atau APM).
- [x] **`main.go`** diperbarui — `obsProvider.InstrumentGORM(db)` + `obsProvider.InstrumentRedis(client)` + `obsProvider.HTTPMiddleware()` + `obsProvider.Shutdown()`.
- [x] **Usecase spans** — setiap metode usecase punya `tracer.Start(ctx, "MethodName")` + `span.RecordError` + `span.SetStatus` → usecase layer ter-track sebagai child span di APM waterfall.
- [x] **CLI generator diperbarui** — skeleton `main.go`, `config.go`, `httpclient/base_client.go`, `route/router.go` semua reflect arsitektur provider. Template `usecase.go.tmpl` include OTel span per method → semua domain baru hasil `make:all` langsung punya tracing.

**Dependencies baru:**
`github.com/uptrace/opentelemetry-go-extra/otelgorm` · `github.com/redis/go-redis/extra/redisotel/v9` · `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` · `go.elastic.co/apm/v2` · `go.elastic.co/apm/module/apmfiber/v2` · `go.elastic.co/apm/module/apmgormv2/v2` · `go.elastic.co/apm/module/apmhttp/v2`

**Hasil coverage:** observability 82%+ · (provider tests via tracing_test.go) — **semua > 80%** ✅

**DoD:** ✅ `go build ./...` bersih; `go test ./...` hijau; `OBSERVABILITY_PROVIDER=elastic_apm` → Elastic APM agent aktif (env-driven); `OBSERVABILITY_PROVIDER=otel` → OTel SDK aktif; kedua provider instrument DB + Redis + HTTP client + HTTP server; CLI generator menghasilkan project dengan tracing siap pakai.

### Fase v1.0 — Hardening & Rilis ✅ SELESAI (2026-06-02)

**Tujuan:** Siap rilis, terverifikasi, terdokumentasi.

- [x] Integration test (`testcontainers-go`: Postgres + Redis) — `internal/integration/` dengan build tag `//go:build integration`; dijalankan di CI job terpisah.
- [x] Audit coverage menyeluruh: **total > 80%**, tidak ada paket inti < 80% — semua hijau sejak v0.6.
- [x] Folder `examples/` final: `examples/jwt/`, `examples/httpclient/`, `examples/messaging/`, `examples/shop/` (end-to-end).
- [x] Dockerfile multi-stage (`golang:1.25-alpine` → `gcr.io/distroless/static-debian12:nonroot`), EXPOSE 8080, semua config via ENV. **Hardening container:** non-root UID 65532, read-only root filesystem, drop ALL capabilities, tanpa shell.
- [x] CI (GitHub Actions `.github/workflows/ci.yml`) — **7 job berjenjang**: `lint → gosec (SAST) → govulncheck (CVE) → gitleaks (secret scan) → test+coverage gate 80% → integration tests → build → docker+trivy (image scan)`. Push ke GHCR hanya di merge ke main. Badge CI + coverage di README.
- [x] Dokumentasi: `README.md` final (badges, stack, architecture, fitur, config, deployment), `SECURITY.md` (kebijakan & cara lapor kerentanan, daftar kontrol keamanan, CI gates), `kubernetes/` manifest (Deployment, ConfigMap, Secret, Service, NetworkPolicy — semua dengan `securityContext`: runAsNonRoot, readOnlyRootFilesystem, drop ALL caps, liveness/readiness probes), `CONTRIBUTING.md`.
- [x] Makefile diperbarui: `make check` (lint+sec+test-race+coverage), `make integration`, `make docker-build`, `make docker-push`, coverage gate 80%.
- [x] Race condition `TestDrain_WithMessage` di `pkg/messaging/rabbitmq` diperbaiki.

**Hasil coverage:** semua paket > 80% ✅ · `go test -race ./...` bersih ✅

**DoD:** ✅ CI pipeline hijau termasuk semua gerbang keamanan; Dockerfile distroless terbangun; integration tests jalan via testcontainers; `examples/` tersedia; K8s manifests ter-harden; dokumentasi lengkap.

---

## 4. Matriks Dependency

| Library | Fungsi | Fase |
|---|---|---|
| `gofiber/fiber/v2` | HTTP framework | v0.1 |
| `gorm.io/gorm` + `driver/postgres`, `driver/mysql` | ORM + driver | v0.1 |
| `spf13/viper` | Config ENV-first | v0.1 |
| `rs/zerolog` | Structured logging | v0.1 |
| `natefinch/lumberjack` | Rotasi file log | v0.1 |
| `google/uuid` | Request-ID | v0.1 |
| `go-playground/validator` | Validasi input | v0.1 |
| `redis/go-redis/v9` | Cache | v0.2 |
| `segmentio/kafka-go` *(atau `IBM/sarama`)* | Kafka producer/consumer | v0.2 |
| `rabbitmq/amqp091-go` | RabbitMQ | v0.2 |
| `sony/gobreaker` | Circuit breaker | v0.3 |
| `spf13/cobra` | CLI | v0.4 |
| `golang-jwt/jwt/v5` | JWT auth | v0.5 |
| `prometheus/client_golang` | Metrics | v0.5 |
| `go.opentelemetry.io/otel` (SDK) | Tracing | v0.5 |
| `swaggo/swag` + `gofiber/swagger` | OpenAPI/Swagger | v0.5 |
| `testcontainers/testcontainers-go` | Integration test | v1.0 |
| `securego/gosec` | SAST (static security) | v0.1→ |
| `golang.org/x/vuln/govulncheck` | Scan CVE dependency | v0.1→ |
| `gitleaks` | Scan secret bocor | v0.1→ |
| `aquasecurity/trivy` | Scan image container | v1.0 |
| `gofiber/contrib/...` limiter/helmet *(atau middleware bawaan Fiber)* | Rate limit & security headers | v0.1 |

---

## 5. Struktur Folder Target

Struktur final (lihat detail di [`prompt-go-microservice.md`](prompt-go-microservice.md)), dengan penanda fase kelahiran tiap bagian:

```
.
├── cmd/
│   └── api/main.go              ← entrypoint service            (v0.1)
├── config/                      ← Viper loader                  (v0.1)
├── internal/
│   ├── domain/{entity,repository,service}/                      (v0.1)
│   ├── usecase/                                                 (v0.1)
│   ├── delivery/http/{handler,middleware,route}/                (v0.1)
│   └── repository/
│       ├── postgres/                                            (v0.1)
│       └── redis/                                               (v0.2)
├── pkg/
│   ├── logger/                                                  (v0.1)
│   ├── response/  validator/                                    (v0.1)
│   ├── messaging/{kafka,rabbitmq}/                              (v0.2)
│   ├── httpclient/                                              (v0.3)
│   ├── auth/         ← JWT                                      (v0.5)
│   └── observability/ ← metrics + tracing                      (v0.5)
├── cli/             ← Go module terpisah (go.mod sendiri)       (v0.4)
│   ├── cmd/main.go  ← entrypoint CLI `wapgo`
│   ├── commands/    ← cobra: new, make:*
│   └── generator/   ← skeleton ter-embed (//go:embed all:templates)
├── go.work          ← workspace dev lokal (root + ./cli)       (v0.4)
├── examples/        ← contoh per fitur + end-to-end            (v0.1→)
├── migrations/                                                  (v0.1)
├── logs/                                                        (v0.1)
├── kubernetes/      ← manifest deploy                          (v1.0)
├── Dockerfile  docker-compose.yml  Makefile  .env.example      (v0.1/0.2/1.0)
└── go.mod                                                       (v0.1)
```

---

## 6. Definition of Done Global & Konvensi

**Aturan arsitektur:** patuhi batas layer Clean Architecture; interface di domain, implementasi di infrastruktur; wiring hanya di `cmd/api/main.go`.

**Konvensi:** penamaan file snake_case mengikuti folder; package = nama folder; commit mengikuti Conventional Commits; branch `feat/`, `fix/`, `chore/`.

**Checklist kualitas per-PR:**
- [ ] Lint bersih (`golangci-lint run`).
- [ ] Unit test menyertai setiap fitur.
- [ ] Coverage paket terdampak **> 80%**.
- [ ] Ada **example** / `ExampleXxx`.
- [ ] **`gosec`, `govulncheck`, `gitleaks` bersih** + checklist [§7 Security](#7-security--syarat-lolos--aman) terpenuhi.
- [ ] Tidak ada TODO/placeholder.

**Konvensi testing:** `_test.go` di tiap package; mock interface (hand-written atau `mockery`) untuk repository & external service; `httptest` untuk handler & httpclient; `Example*` functions sebagai dokumentasi hidup; folder `examples/` untuk skenario end-to-end.

---

## 7. Security — Syarat Lolos & Aman

> **Prinsip:** *secure by default*, *defense in depth*, *least privilege*. Setiap fase mengaktifkan kontrol di bawah ini dan **CI menolak merge bila gerbang keamanan gagal**.

**Kontrol per area:**

| Area | Kontrol wajib |
|---|---|
| **Secrets** | Tidak ada secret di kode/log; hanya dari ENV/K8s Secret. `.env` gitignored; `.env.example` tanpa nilai nyata. Field sensitif diredaksi di log. `gitleaks` di CI & pre-commit. |
| **HTTP inbound** | Security headers (HSTS, X-Content-Type-Options, X-Frame-Options, Referrer-Policy, CSP). Rate limit per-IP. Batas ukuran body. CORS allowlist ketat. `recover` tanpa bocor stack trace. |
| **Input** | Validasi semua DTO (`validator`), reject field tak dikenal, sanitasi. GORM parameterized (anti SQL-injection). |
| **Auth** | JWT algoritma di-pin (tolak `alg:none`), validasi `exp/iat/iss/aud`, secret kuat, perbandingan constant-time. Opsional RBAC. |
| **HTTP outbound** | TLS verify ON. Proteksi SSRF (allowlist host, tolak redirect ke internal/loopback/link-local). Timeout & batas response. |
| **Messaging** | Validasi payload, DLQ untuk poison message, dukungan TLS/SASL (Kafka) & TLS (RabbitMQ) via ENV. |
| **Database** | User least-privilege, dukungan TLS, tanpa query string-concat. |
| **Container** | Distroless, non-root, read-only FS, drop ALL capabilities, tanpa shell. Image scan (`trivy`). |
| **Kubernetes** | `securityContext` (runAsNonRoot, readOnlyRootFilesystem, drop caps), resource limits, NetworkPolicy, Secret bukan ConfigMap untuk kredensial. |
| **Observability** | `/metrics`, `/swagger`, pprof tidak terekspos publik di produksi. |
| **Supply chain** | `go.sum` terverifikasi, versi ter-pin, `govulncheck` + Dependabot/renovate. SAST `gosec` di CI. |

**Gerbang CI keamanan (semua harus PASS = "lolos & aman"):**
`gosec` (SAST) · `govulncheck` (CVE) · `gitleaks` (secret) · `trivy` (image) · `golangci-lint` · test + coverage 80%.

**Dokumen pendamping:** `SECURITY.md` (kebijakan & pelaporan kerentanan), pre-commit hook (gitleaks + gosec).

---

## 8. Quick Start

```bash
# 1. Pasang CLI
go install github.com/abdullahPrasetio/wapgo/cli/cmd@latest      # → binary `wapgo`

# 2. Buat project baru
wapgo new my-service --module github.com/me/my-service
cd my-service

# 3. Jalankan
make docker-up           # postgres, redis, kafka, rabbitmq
cp .env.example .env
make run

# 4. Cek
curl http://localhost:8080/health

# 5. Tambah domain baru
wapgo make:all product
```

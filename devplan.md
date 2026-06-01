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

### Fase v0.1 — Core Skeleton (fondasi yang bisa jalan)

**Tujuan:** Service minimal yang bisa `make run`, punya `/health`, dan CRUD `/users` ke Postgres.

- [ ] `go.mod` (module `wapgo`, Go 1.22), `.gitignore`, `.env.example`, `Makefile`, struktur folder lengkap.
- [ ] `config/` — Viper loader (ENV → yaml → default), struct mapping, `service_urls.go`.
- [ ] `pkg/logger/` — zerolog (JSON di prod / console di dev berdasar `APP_ENV`), rotasi file via lumberjack, ambil request-id dari context.
- [ ] `pkg/response/` (struct response terpusat) + `pkg/validator/`.
- [ ] DB: inisialisasi GORM (postgres default, switch ke mysql via `DB_DRIVER`), connection pool dari ENV, auto-migrate saat `DB_AUTO_MIGRATE=true`.
- [ ] HTTP: Fiber app + middleware stack berurutan: `recover → request-id → security-headers → rate-limit → body-limit → logger → CORS`.
- [ ] **Security baseline:** middleware security headers (HSTS, X-Content-Type-Options, X-Frame-Options=DENY, Referrer-Policy, CSP minimal); rate limiter per-IP; batas ukuran body request; CORS allowlist ketat (tanpa `*` saat credentials); `recover` **tidak membocorkan stack trace** ke response; validasi input wajib di semua DTO (reject field tak dikenal); log **me-redaksi** field sensitif (password, token, `Authorization`).
- [ ] Reference domain **user** lengkap: entity → repository interface → postgres impl → usecase → handler → route. Query lewat GORM parameterized (anti SQL-injection).
- [ ] `/health` (cek DB + Redis), graceful shutdown (SIGTERM/SIGINT, tunggu max 30s).
- [ ] `cmd/api/main.go` mem-wire semua dependency via constructor.

**DoD:** `make run` jalan; `GET /health` → 200; CRUD `/users` berfungsi terhadap Postgres; security headers & rate limit aktif; `gosec`/`gitleaks` bersih; test + example terpasang, coverage > 80%.

### Fase v0.2 — Cache & Messaging

**Tujuan:** Tambah Redis cache + Kafka & RabbitMQ yang independen.

- [ ] `internal/repository/redis/cache.go` — helper cache generic + TTL; integrasi ke usecase user (cache-aside).
- [ ] `pkg/messaging/kafka/` — `producer.go` (context, JSON serialize, request-id di header, auto-reconnect) + `consumer.go` (group consumer, pola `handler func(ctx, msg) error`, graceful shutdown).
- [ ] `pkg/messaging/rabbitmq/` — `publisher.go` (declare exchange, publish dengan routing key, request-id di properties) + `consumer.go` (declare queue + binding, DLQ via `x-dead-letter-exchange`, graceful shutdown).
- [ ] `/health` diperluas: tambahkan status Kafka & RabbitMQ.
- [ ] `docker-compose.yml`: postgres, mysql, redis, zookeeper, kafka, rabbitmq-management.

**DoD:** produce/consume pesan sukses lokal via docker-compose; `/health` melaporkan 4 service; test + example, coverage > 80%.

### Fase v0.3 — Inter-service HTTP Client (Resilience)

**Tujuan:** Komunikasi antar-service yang tahan gangguan.

- [ ] `pkg/httpclient/base_client.go` — wrapper net/http; inject `X-Request-ID` & `Authorization` dari context; timeout konfigurabel (default 5s).
- [ ] `pkg/httpclient/middleware.go` — retry (max 3, exponential backoff, hanya 5xx & network error); timeout context-aware (hormati deadline caller); circuit breaker (open setelah 5 gagal beruntun, half-open setelah 30s).
- [ ] `pkg/httpclient/user_client.go` — implementasi interface `internal/domain/service/external_user.go` (`GetUser(ctx, id)`), URL dari `USER_SERVICE_URL`.
- [ ] **Security:** TLS verify **ON** by default (`InsecureSkipVerify=false`); proteksi SSRF (validasi host tujuan terhadap allowlist, tolak redirect ke alamat internal/loopback/link-local); timeout & batas ukuran response untuk cegah resource exhaustion.

**DoD:** unit test resilience (retry & circuit breaker) + test SSRF guard lulus dengan server mock (`httptest`); example pemakaian client; coverage > 80%.

### Fase v0.4 — CLI `wapgo` (Installer + Generator)

**Tujuan:** CLI yang **memasang/scaffold framework** sekaligus **generate kode**.

- [ ] `cmd/cli/main.go` + `cli/commands/` (Cobra), binary bernama `wapgo`.

**Installer / scaffolder:**
- [ ] `wapgo new <project>` — buat project baru lengkap dari template (folder, `go.mod` dengan module path kustom, semua file core v0.1–v0.3, reference domain user). Flag: `--module github.com/me/svc`, `--db postgres|mysql`.
- [ ] Template seluruh skeleton di-*embed* ke binary via `//go:embed` (`cli/templates/`) → `wapgo new` jalan tanpa akses jaringan.
- [ ] `wapgo version`, dan (opsional) `wapgo upgrade`.
- [ ] Distribusi: `go install github.com/abdullahPrasetio/wapgo/cmd/cli@latest` (binary `wapgo`) + `install.sh` untuk unduh rilis biner.

**Generator `make:*` (di dalam project):**
- [ ] `make:model`, `make:repo`, `make:usecase`, `make:controller`, `make:route`, `make:client`, `make:all`.
- [ ] Template-based codegen mengikuti pola reference user; argumen `<name>` snake_case → struct/interface/method.
- [ ] `make:all <name>` jalankan generator berurutan + cetak ringkasan file yang dibuat.

**DoD:** `wapgo new shop && cd shop && go build ./...` sukses tanpa edit manual; lalu `wapgo make:all product` menghasilkan domain yang langsung compile; test untuk logika codegen, coverage > 80%.

### Fase v0.5 — Peningkatan "Hebat" (di luar spec)

**Tujuan:** Naikkan kelas dari boilerplate ke framework production-grade.

- [ ] **Auth:** JWT middleware + helper (sign/verify). **Hardening:** algoritma di-pin (tolak `alg:none`), validasi `exp`/`iat`/`iss`/`aud`, secret/kunci kuat dari ENV, perbandingan token constant-time; opsional RBAC middleware (role/scope). Propagasi klaim ke context.
- [ ] **Observability:** Prometheus metrics (`/metrics`, RED metrics per route) + OpenTelemetry tracing (span dari request-id, propagasi ke httpclient & messaging).
- [ ] **Security:** `/metrics`, `/swagger`, dan pprof **tidak terekspos publik di produksi** (dinonaktifkan atau di balik auth/jaringan internal saat `APP_ENV=production`).

**DoD:** `/metrics` & `/swagger` aktif di dev namun terlindung di prod; JWT lolos uji serangan umum (`alg:none`, token kedaluwarsa, signature palsu); trace tampil end-to-end; test + example, coverage > 80%.

### Fase v1.0 — Hardening & Rilis

**Tujuan:** Siap rilis, terverifikasi, terdokumentasi.

- [ ] Integration test (testcontainers: Postgres/Redis/Kafka/RabbitMQ) melengkapi unit test tiap fase.
- [ ] Audit coverage menyeluruh: **total > 80%**, tidak ada paket inti < 80% — tambal yang kurang.
- [ ] Folder `examples/` final: contoh end-to-end (service `shop` hasil `wapgo new`) + example per fitur (messaging, httpclient resilience, JWT).
- [ ] Dockerfile multi-stage (builder `golang:1.22-alpine` → distroless), EXPOSE 8080, semua config via ENV. **Hardening container:** non-root user, read-only root filesystem, drop capabilities, tanpa shell.
- [ ] CI (GitHub Actions) — **semua wajib lolos (gate "lolos & aman")**: `lint (golangci-lint) → gosec (SAST) → govulncheck (CVE deps) → gitleaks (secret scan) → trivy (image scan) → go test ./... -coverprofile (gate coverage 80%) → build → docker`. Badge coverage + security di README. Dependabot/renovate aktif.
- [ ] Dokumentasi: README final, `SECURITY.md` (kebijakan & cara lapor kerentanan), `kubernetes/` manifest (Deployment, ConfigMap, Secret, Service, liveness/readiness probes, `securityContext`: runAsNonRoot, readOnlyRootFilesystem, drop ALL caps, NetworkPolicy), CONTRIBUTING.

**DoD:** CI hijau **termasuk semua gerbang keamanan** + gate coverage 80%; image terbangun & lolos scan; contoh deploy K8s ter-harden & `examples/` tersedia.

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
│   ├── api/main.go              ← entrypoint service            (v0.1)
│   └── cli/main.go              ← entrypoint CLI `wapgo`        (v0.4)
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
├── cli/
│   ├── commands/    ← cobra: new, make:*                        (v0.4)
│   └── templates/   ← skeleton ter-embed (//go:embed)          (v0.4)
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
go install github.com/abdullahPrasetio/wapgo/cmd/cli@latest      # → binary `wapgo`

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

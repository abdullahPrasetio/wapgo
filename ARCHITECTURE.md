# wapgo — Arsitektur & Konsep

Dokumen ini menjelaskan keputusan desain yang sering membingungkan:
dua folder bernama `repository`, kenapa struct sengaja lowercase, kenapa
constructor return interface bukan struct, dan cara kerja mocking.

---

## Daftar Isi

1. [Gambaran Layer](#1-gambaran-layer)
2. [Kenapa Ada Dua Folder `repository`?](#2-kenapa-ada-dua-folder-repository)
3. [Interface vs Implementasi — Pola Standar](#3-interface-vs-implementasi--pola-standar)
4. [Kenapa Constructor Return Interface, Bukan Struct?](#4-kenapa-constructor-return-interface-bukan-struct)
5. [Kenapa Struct Implementasi Lowercase (Unexported)?](#5-kenapa-struct-implementasi-lowercase-unexported)
6. [Dependency Injection — Wiring di `main.go`](#6-dependency-injection--wiring-di-maingo)
7. [Cara Kerja Mock di Test](#7-cara-kerja-mock-di-test)
8. [Aturan Import Antar Layer](#8-aturan-import-antar-layer)

---

## 1. Gambaran Layer

```
┌──────────────────────────────────────────────┐
│  cmd/api/main.go  ← satu-satunya tempat wiring│
└───────────────────┬──────────────────────────┘
                    │ inject via constructor
        ┌───────────▼───────────┐
        │   delivery/http/      │  ← HTTP: handler, middleware, route
        │   (Handler layer)     │     AccessLog middleware: start journal,
        │                       │     capture req/resp, call Finish()
        └───────────┬───────────┘
                    │ panggil via interface UserUseCase
        ┌───────────▼───────────┐
        │   internal/usecase/   │  ← Business logic
        │   (UseCase layer)     │     journal.FromContext(ctx).AddTrace(...)
        └───────────┬───────────┘
                    │ panggil via interface UserRepository / Cacher
        ┌───────────▼───────────┐
        │  internal/domain/     │  ← Kontrak (interface only)
        │  repository/          │     TIDAK tahu teknologi apapun
        └───────────┬───────────┘
                    │ diimplementasi oleh
        ┌───────────┴──────────────────┐
        │                              │
┌───────▼──────────┐    ┌─────────────▼──────┐
│ repository/      │    │ repository/redis/   │
│ db/              │    │ (RedisCacher)       │
│ (userRepository) │    │                     │
└──────────────────┘    └─────────────────────┘

Paket lintas-layer (boleh dipakai semua layer di atas):
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐
│  pkg/logger      │  │  pkg/journal     │  │  pkg/observability   │
│  (4 log sinks)   │  │  (request scope, │  │  (OTel / Elastic APM │
│  SetupSinks()    │  │   ctx-stored)    │  │   bridge, StartSpan) │
└──────────────────┘  └──────────────────┘  └──────────────────────┘
```

**Aturan utama:** Panah dependency selalu mengarah ke dalam (ke domain).
Layer luar boleh tahu layer dalam, tapi tidak sebaliknya.

---

## 2. Kenapa Ada Dua Folder `repository`?

Ini yang paling sering membingungkan. Ini bedanya:

| | `internal/domain/repository/` | `internal/repository/db/` |
|---|---|---|
| **Isi** | Interface saja | Implementasi konkret (GORM, MySQL/Postgres) |
| **Tahu teknologi** | Tidak (tidak ada import gorm/redis) | Ya (import gorm, sql) |
| **Siapa yang pakai** | Usecase — depend ke sini | main.go — inject ke usecase |
| **Bisa di-mock** | Ya, cukup buat struct yang implement | Tidak perlu di-mock |

Analoginya seperti **kontrak kerja** vs **orang yang mengerjakan**:

```
domain/repository/user_repository.go
= "Saya butuh seseorang yang bisa FindByID, Create, Update, Delete"
  (tidak peduli caranya pakai apa)

repository/db/user_repository.go
= "Saya sanggup memenuhi kontrak itu, caranya pakai GORM (MySQL atau Postgres)"
```

Usecase hanya memegang kontrak. Ia tidak tahu — dan tidak perlu tahu —
bahwa di baliknya ada Postgres, atau MySQL, atau bahkan in-memory map.

---

## 3. Interface vs Implementasi — Pola Standar

Pola ini dipakai konsisten di semua layer:

```
internal/domain/repository/user_repository.go   ← interface UserRepository
internal/repository/db/user_repository.go       ← implementasi konkret (GORM, MySQL/Postgres)

internal/domain/repository/cache.go             ← interface Cacher
internal/repository/redis/cache.go              ← implementasi konkret

internal/usecase/user_usecase.go                ← interface UserUseCase (+ impl)
internal/delivery/http/handler/user_handler.go  ← bergantung ke UserUseCase (interface)
```

Setiap interface didefinisikan di **sisi pemakai** (domain/usecase),
bukan di sisi implementasi.

---

## 4. Kenapa Constructor Return Interface, Bukan Struct?

```go
// Di internal/repository/db/user_repository.go

// ❌ Kalau return *userRepository (concrete):
func NewUserRepository(db *gorm.DB) *userRepository { ... }
// → caller harus import package db
// → caller jadi tahu ini implementasi GORM, bukan kontrak abstrak
// → tidak bisa swap implementasi tanpa ubah caller

// ✅ Yang ada sekarang — return interface:
func NewUserRepository(db *gorm.DB) domainrepo.UserRepository { ... }
// → caller hanya tahu tipe UserRepository (interface)
// → caller tidak perlu import package db
// → bisa diganti MySQL/Postgres/in-memory tanpa ubah usecase sama sekali
```

Prinsipnya: **return tipe seluas mungkin, terima parameter sesempit mungkin.**
Interface adalah tipe yang paling luas — caller tidak terikat ke implementasi apapun.

---

## 5. Kenapa Struct Implementasi Lowercase (Unexported)?

```go
// Di internal/repository/db/user_repository.go

type userRepository struct {   // ← huruf kecil = unexported
    db *gorm.DB
}
```

Karena struct ini adalah **detail implementasi** yang tidak boleh bocor keluar.

Kalau di-export (`UserRepository` huruf besar), caller bisa langsung pakai
struct-nya — membypass interface dan membuat coupling langsung ke GORM.
Dengan tetap unexported, satu-satunya cara pakai adalah lewat constructor
yang return interface:

```go
// Satu-satunya pintu masuk yang tersedia dari luar package:
repo := db.NewUserRepository(gormDB)   // tipe: domainrepo.UserRepository
                                       // bukan *db.userRepository
```

---

## 6. Dependency Injection — Wiring di `main.go`

Semua "sambungan" antara interface dan implementasi dilakukan **hanya** di
`cmd/api/main.go`. Tidak ada tempat lain yang boleh melakukan ini.

```go
// cmd/api/main.go (disederhanakan)

// 1. Buat implementasi konkret
db         := database.Connect(cfg)
redisClient := cache.Connect(cfg)

// 2. Bungkus dengan implementasi repository
userRepo  := db.NewUserRepository(gormDB)        // return UserRepository (interface)
cacher    := redis.New(redisClient, "users")     // return *RedisCacher (implements Cacher)

// 3. Inject ke usecase — usecase hanya menerima interface
userUC := usecase.NewUserUseCase(userRepo)       // return UserUseCase (interface)

// 4. Inject ke handler — handler hanya menerima interface
userHandler := handler.NewUserHandler(userUC, validator.New())

// 5. Daftarkan route
route.RegisterUserRoutes(app, userHandler, cfg)
```

Kalau suatu saat ingin ganti MySQL ke Postgres (atau sebaliknya), hanya
konfigurasi `DB_DRIVER` yang berubah — implementasi GORM di `repository/db/`
sudah driver-agnostic. Usecase, handler, dan route tidak perlu disentuh sama sekali.

---

## 7. Cara Kerja Mock di Test

Karena setiap layer bergantung ke interface, mock cukup dibuat dengan
membuat struct yang memenuhi interface tersebut — tanpa library tambahan,
tanpa database nyata.

### Mock repository untuk test usecase

```go
// internal/usecase/user_usecase_test.go

// Struct ini memenuhi interface domainrepo.UserRepository
type mockUserRepo struct {
    users    map[uuid.UUID]*entity.User
    forceErr error
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
    if m.forceErr != nil {
        return nil, m.forceErr
    }
    u, ok := m.users[id]
    if !ok {
        return nil, gorm.ErrRecordNotFound
    }
    return u, nil
}
// ... implementasi method lain ...

func TestGetUser_Success(t *testing.T) {
    repo := &mockUserRepo{users: make(map[uuid.UUID]*entity.User)}
    uc   := usecase.NewUserUseCase(repo)   // inject mock, bukan Postgres nyata
    
    // test logic ...
}
```

### Mock usecase untuk test handler

```go
// internal/delivery/http/handler/user_handler_test.go

// Struct ini memenuhi interface usecase.UserUseCase
type mockUserUC struct {
    user *entity.User
    err  error
}

func (m *mockUserUC) GetUser(_ context.Context, _ string) (*entity.User, error) {
    return m.user, m.err
}
// ... implementasi method lain ...

func TestGetUser_NotFound(t *testing.T) {
    h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrNotFound}, validator.New())
    // test via httptest tanpa usecase nyata, tanpa database
}
```

Hasilnya: unit test berjalan dalam milidetik, tidak butuh Docker, tidak
butuh koneksi jaringan. Integration test (yang butuh database nyata) ada
di `internal/integration/` dengan build tag `//go:build integration`.

---

## 8. Aturan Import Antar Layer

| Package | Boleh import | Tidak boleh import |
|---|---|---|
| `internal/domain/` | `entity` saja | Semua paket lain di `internal/` |
| `internal/usecase/` | `domain/entity`, `domain/repository`, `domain/service`, `pkg/journal` | `delivery/`, `repository/db`, `repository/redis` |
| `internal/delivery/` | `usecase` (interface), `domain/entity`, `pkg/` | `repository/db`, `repository/redis` |
| `internal/repository/db/` | `domain/entity`, `domain/repository`, `gorm` | `usecase/`, `delivery/` |
| `internal/repository/redis/` | `domain/repository` | `usecase/`, `delivery/` |
| `pkg/journal` | `pkg/logger` | `internal/` apapun |
| `pkg/logger` | `zerolog`, `lumberjack` | `internal/` apapun |
| `cmd/api/main.go` | Semua | — |

Kalau ada import yang melanggar tabel di atas, itu tanda ada layer yang
"mengetahui terlalu banyak" dan perlu direfactor.

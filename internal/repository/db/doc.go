// Package db berisi implementasi relasional (GORM) untuk interface-interface
// yang didefinisikan di internal/domain/repository.
//
// Saat ini satu folder ini mendukung semua driver relasional (MySQL, Postgres,
// dll) karena kode GORM bersifat driver-agnostic — pemilihan driver dilakukan
// di pkg/database/database.go via DB_DRIVER env var.
//
// Jika ke depan dibutuhkan implementasi yang benar-benar spesifik per driver
// (misalnya memanfaatkan fitur eksklusif MySQL atau Postgres), struktur folder
// bisa dipecah menjadi:
//
//	internal/repository/mysql/
//	internal/repository/postgres/
//	internal/repository/sqlite/   ← untuk testing / embedded
//
// Aturan utama package ini:
//   - Setiap struct bersifat unexported. Caller hanya menerima tipe interface
//     dari constructor-nya, bukan struct konkretnya.
//   - Constructor selalu return tipe interface (bukan *struct).
//   - Boleh import gorm dan driver DB. Layer di atasnya (usecase, handler)
//     tidak boleh import package ini langsung.
package db

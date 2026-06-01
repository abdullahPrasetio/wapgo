// Package postgres berisi implementasi Postgres untuk interface-interface yang
// didefinisikan di internal/domain/repository.
//
// Aturan utama package ini:
//   - Setiap struct di sini bersifat unexported (huruf kecil). Caller luar
//     hanya menerima tipe interface dari constructor-nya, bukan struct konkretnya.
//     Ini memastikan tidak ada yang bisa bergantung ke detail implementasi.
//   - Constructor selalu return tipe interface (bukan *struct), sehingga
//     caller tidak perlu tahu bahwa di baliknya ada Postgres.
//   - Boleh import gorm, driver Postgres, dan paket infrastruktur lain.
//     Layer di atasnya (usecase, handler) tidak boleh import package ini langsung.
package postgres

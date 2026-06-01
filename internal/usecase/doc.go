// Package usecase berisi business logic aplikasi.
//
// Setiap usecase terdiri dari tiga bagian dalam satu file:
//
//  1. DTOs — struct request/response yang menjadi batas input usecase.
//     Validasi dilakukan di layer handler sebelum DTO dikirim ke sini.
//
//  2. Interface — kontrak yang diekspos ke handler. Handler bergantung ke
//     interface ini, bukan ke struct implementasinya. Tujuannya sama seperti
//     di layer repository: memudahkan mocking saat unit test handler.
//
//  3. Implementasi — struct unexported yang memenuhi interface di atas.
//     Constructor-nya selalu return tipe interface, bukan *struct.
//
// Aturan utama package ini:
//   - Boleh import domain/entity dan domain/repository (interface).
//   - Tidak boleh import package delivery (handler, route) atau infrastruktur
//     secara langsung (gorm, redis, dsb) — kecuali error sentinel seperti
//     gorm.ErrRecordNotFound yang dipakai untuk mapping ke domain error.
//   - Seluruh akses ke database dan cache dilakukan lewat interface dari
//     domain/repository, bukan lewat implementasi konkretnya.
package usecase

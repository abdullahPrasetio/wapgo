// Package repository berisi kontrak (interface) untuk semua operasi persistence
// dalam domain ini.
//
// Aturan utama package ini:
//   - Hanya berisi interface, bukan implementasi.
//   - Tidak boleh import package infrastruktur (gorm, redis, sql, dsb).
//   - Usecase bergantung ke package ini, bukan ke implementasinya.
//
// Alur dependency yang benar:
//
//	usecase  ──depends on──▶  domain/repository (interface)
//	                                  ▲
//	repository/db    ──implements─┘
//	repository/redis ──implements─┘
//
// Dengan pola ini, usecase bisa di-test tanpa database nyata — cukup buat
// struct dummy yang memenuhi interface, lalu inject ke usecase.
package repository

// Package redis berisi implementasi Redis untuk interface Cacher yang
// didefinisikan di internal/domain/repository.
//
// ErrCacheMiss diekspor dari package ini karena digunakan oleh caller untuk
// membedakan "key tidak ada" dari error koneksi. Gunakan errors.Is() untuk
// memeriksanya:
//
//	err := cacher.Get(ctx, key, &dest)
//	if errors.Is(err, redis.ErrCacheMiss) {
//	    // key tidak ditemukan, bukan error fatal
//	}
package redis

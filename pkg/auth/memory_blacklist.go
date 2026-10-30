package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryBlacklist stores revoked JTIs in process memory.
//
// It is the fallback used when Redis is not configured. Revocation therefore
// only applies to the current process: a revoked token is still accepted by
// other replicas and after a restart. Use RedisBlacklist for deployments with
// more than one instance or a need for durable revocation.
//
// It is safe for concurrent use.
type MemoryBlacklist struct {
	mu  sync.Mutex
	now func() time.Time
	m   map[string]time.Time // jti → expiry
}

// NewMemoryBlacklist creates an empty in-memory blacklist.
func NewMemoryBlacklist() *MemoryBlacklist {
	return &MemoryBlacklist{now: time.Now, m: make(map[string]time.Time)}
}

// Revoke marks a JTI as revoked for the given duration.
func (b *MemoryBlacklist) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if jti == "" {
		return fmt.Errorf("revoke: empty jti")
	}
	now := b.now()

	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.m) >= 1024 {
		for k, exp := range b.m {
			if !now.Before(exp) {
				delete(b.m, k)
			}
		}
	}
	b.m[jti] = now.Add(ttl)
	return nil
}

// IsRevoked reports whether the JTI is currently revoked.
func (b *MemoryBlacklist) IsRevoked(_ context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	exp, ok := b.m[jti]
	if !ok {
		return false, nil
	}
	if !b.now().Before(exp) {
		delete(b.m, jti)
		return false, nil
	}
	return true, nil
}

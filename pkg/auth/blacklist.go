package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const blacklistKeyPrefix = "jwt:bl:"

// Blacklist defines token revocation storage.
// Implementations must be safe for concurrent use.
type Blacklist interface {
	// Revoke stores the JTI (jwt.ID) until the token's natural expiry.
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	// IsRevoked returns true when the JTI has been revoked.
	// On storage error it returns (false, err); callers decide whether to fail open or closed.
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// RedisBlacklist stores revoked JTIs in Redis.
// Keys are stored with a TTL equal to the remaining token lifetime so they
// are cleaned up automatically — no separate sweeper is needed.
type RedisBlacklist struct {
	client *redis.Client
}

// NewRedisBlacklist creates a RedisBlacklist backed by the given Redis client.
func NewRedisBlacklist(client *redis.Client) *RedisBlacklist {
	return &RedisBlacklist{client: client}
}

// Revoke marks a JTI as revoked for the given duration.
func (b *RedisBlacklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" {
		return fmt.Errorf("revoke: empty jti")
	}
	key := blacklistKeyPrefix + jti
	if err := b.client.Set(ctx, key, 1, ttl).Err(); err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// IsRevoked reports whether the JTI exists in the blacklist.
func (b *RedisBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	key := blacklistKeyPrefix + jti
	n, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("blacklist check: %w", err)
	}
	return n > 0, nil
}

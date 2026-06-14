package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

// RedisCacher implements repository.Cacher using a Redis client.
type RedisCacher struct {
	client *redis.Client
	ns     string
}

func New(client *redis.Client, ns string) *RedisCacher {
	return &RedisCacher{client: client, ns: ns}
}

func (r *RedisCacher) prefixed(key string) string {
	if r.ns == "" {
		return key
	}
	return r.ns + ":" + key
}

func (r *RedisCacher) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache set marshal: %w", err)
	}
	return r.client.Set(ctx, r.prefixed(key), b, ttl).Err()
}

func (r *RedisCacher) Get(ctx context.Context, key string, dest interface{}) error {
	b, err := r.client.Get(ctx, r.prefixed(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("cache get: %w", err)
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("cache get unmarshal: %w", err)
	}
	return nil
}

func (r *RedisCacher) Del(ctx context.Context, keys ...string) error {
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = r.prefixed(k)
	}
	return r.client.Del(ctx, full...).Err()
}

func (r *RedisCacher) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, r.prefixed(key)).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists: %w", err)
	}
	return n > 0, nil
}

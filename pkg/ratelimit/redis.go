package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// incrWindowScript increments the window counter and sets its TTL atomically on
// the first hit. Returns {count, pttl_ms}.
const incrWindowScript = `
local c = redis.call('INCR', KEYS[1])
if c == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return {c, redis.call('PTTL', KEYS[1])}
`

// RedisLimiter is a fixed-window limiter whose counters live in Redis, so all
// replicas share the same quota. Requires a reachable Redis instance; callers
// should fail open on error (see the package docs).
type RedisLimiter struct {
	client *redis.Client
	prefix string
	max    int
	window time.Duration
}

// NewRedis creates a Redis-backed limiter. prefix namespaces the counters
// (e.g. "myapp:rl"); pass "" to use "ratelimit".
func NewRedis(client *redis.Client, prefix string, max int, window time.Duration) *RedisLimiter {
	if max <= 0 {
		max = 100
	}
	if window <= 0 {
		window = time.Minute
	}
	if prefix == "" {
		prefix = "ratelimit"
	}
	return &RedisLimiter{client: client, prefix: prefix, max: max, window: window}
}

// Allow increments the shared counter for key and reports whether it is within quota.
func (r *RedisLimiter) Allow(ctx context.Context, key string) (Result, error) {
	res, err := r.client.Eval(ctx, incrWindowScript,
		[]string{r.prefix + ":" + key},
		r.window.Milliseconds(),
	).Slice()
	if err != nil {
		return Result{}, fmt.Errorf("rate limit: %w", err)
	}
	if len(res) != 2 {
		return Result{}, fmt.Errorf("rate limit: unexpected script reply %v", res)
	}
	count, _ := res[0].(int64)
	ttlMs, _ := res[1].(int64)

	if count > int64(r.max) {
		retryAfter := time.Duration(ttlMs) * time.Millisecond
		if ttlMs <= 0 {
			retryAfter = r.window
		}
		return Result{Allowed: false, Remaining: 0, RetryAfter: retryAfter}, nil
	}
	return Result{Allowed: true, Remaining: r.max - int(count)}, nil
}

// Package ratelimit provides request-rate limiting that works without any
// external dependency and can optionally be shared across replicas via Redis.
//
// Two implementations are provided:
//
//   - MemoryLimiter: fixed-window counter held in process memory. It is
//     per-instance: with N replicas behind a load balancer each replica enforces
//     its own quota, so the effective limit is up to N × Max. State resets on
//     restart. Use it for single-instance deployments or as a safe fallback.
//
//   - RedisLimiter: fixed-window counter stored in Redis so every replica shares
//     one quota. It degrades safely: when Redis is unreachable the caller gets an
//     error and the HTTP middleware fails open (logs a warning, allows the
//     request) so a cache outage cannot take the API down.
package ratelimit

import (
	"context"
	"time"
)

// Result describes the outcome of one Allow call.
type Result struct {
	// Allowed is false when the caller exceeded the configured quota.
	Allowed bool
	// Remaining is the number of requests left in the current window (>= 0).
	Remaining int
	// RetryAfter is how long the caller should wait before retrying.
	// It is zero when Allowed is true.
	RetryAfter time.Duration
}

// Limiter decides whether a request identified by key may proceed.
// Implementations must be safe for concurrent use.
type Limiter interface {
	Allow(ctx context.Context, key string) (Result, error)
}

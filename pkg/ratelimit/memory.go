package ratelimit

import (
	"context"
	"sync"
	"time"
)

// sweepThreshold bounds the entry map; expired windows are purged once the map
// grows past it, so a long-lived process with many client IPs cannot leak.
const sweepThreshold = 4096

type windowState struct {
	count   int
	resetAt time.Time
}

// MemoryLimiter is a fixed-window limiter backed by process memory.
// It is per-instance and not shared between replicas; see the package docs.
type MemoryLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	m      map[string]windowState
}

// NewMemory creates a MemoryLimiter allowing max requests per window per key.
// Non-positive values fall back to 100 requests per minute.
func NewMemory(max int, window time.Duration) *MemoryLimiter {
	if max <= 0 {
		max = 100
	}
	if window <= 0 {
		window = time.Minute
	}
	return &MemoryLimiter{
		max:    max,
		window: window,
		now:    time.Now,
		m:      make(map[string]windowState),
	}
}

// Allow increments the counter for key and reports whether it is within quota.
func (m *MemoryLimiter) Allow(_ context.Context, key string) (Result, error) {
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.m) >= sweepThreshold {
		m.sweepLocked(now)
	}

	w, ok := m.m[key]
	if !ok || !now.Before(w.resetAt) {
		w = windowState{resetAt: now.Add(m.window)}
	}
	w.count++
	m.m[key] = w

	if w.count > m.max {
		return Result{Allowed: false, Remaining: 0, RetryAfter: w.resetAt.Sub(now)}, nil
	}
	remaining := m.max - w.count
	return Result{Allowed: true, Remaining: remaining}, nil
}

// sweepLocked drops windows whose reset time has passed. Caller holds m.mu.
func (m *MemoryLimiter) sweepLocked(now time.Time) {
	for k, w := range m.m {
		if !now.Before(w.resetAt) {
			delete(m.m, k)
		}
	}
}

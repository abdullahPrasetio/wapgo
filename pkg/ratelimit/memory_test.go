package ratelimit

import (
	"context"
	"testing"
	"time"
)

func newTestMemory(max int, window time.Duration) (*MemoryLimiter, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m := NewMemory(max, window)
	m.now = func() time.Time { return now }
	return m, &now
}

func TestMemory_AllowsUpToMax(t *testing.T) {
	m, _ := newTestMemory(3, time.Minute)
	ctx := context.Background()

	for i := range 3 {
		res, err := m.Allow(ctx, "ip-1")
		if err != nil {
			t.Fatalf("Allow: %v", err)
		}
		if !res.Allowed {
			t.Fatalf("request %d unexpectedly blocked", i+1)
		}
	}
	res, err := m.Allow(ctx, "ip-1")
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	if res.Allowed {
		t.Fatal("expected the 4th request to be blocked")
	}
	if res.RetryAfter <= 0 || res.RetryAfter > time.Minute {
		t.Fatalf("unexpected RetryAfter %v", res.RetryAfter)
	}
}

func TestMemory_WindowResets(t *testing.T) {
	m, now := newTestMemory(2, time.Minute)
	ctx := context.Background()

	for range 2 {
		if _, err := m.Allow(ctx, "ip"); err != nil {
			t.Fatal(err)
		}
	}
	if res, _ := m.Allow(ctx, "ip"); res.Allowed {
		t.Fatal("expected blocked within window")
	}

	*now = now.Add(61 * time.Second)
	res, err := m.Allow(ctx, "ip")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Allowed {
		t.Fatal("expected allowed after window reset")
	}
}

func TestMemory_KeysAreIndependent(t *testing.T) {
	m, _ := newTestMemory(1, time.Minute)
	ctx := context.Background()

	if res, _ := m.Allow(ctx, "a"); !res.Allowed {
		t.Fatal("expected a allowed")
	}
	if res, _ := m.Allow(ctx, "b"); !res.Allowed {
		t.Fatal("expected b allowed independently")
	}
	if res, _ := m.Allow(ctx, "a"); res.Allowed {
		t.Fatal("expected a blocked")
	}
}

func TestNewMemory_Defaults(t *testing.T) {
	m := NewMemory(0, 0)
	if m.max != 100 || m.window != time.Minute {
		t.Fatalf("unexpected defaults: max=%d window=%v", m.max, m.window)
	}
}

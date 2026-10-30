package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedisLimiter(t *testing.T, prefix string, max int, window time.Duration) (*RedisLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedis(client, prefix, max, window), mr
}

// TestRedis_SharedAcrossInstances proves two limiters backed by the same Redis
// share one quota — the behaviour the in-memory limiter cannot provide.
func TestRedis_SharedAcrossInstances(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	a := NewRedis(client, "rl", 2, time.Minute)
	b := NewRedis(client, "rl", 2, time.Minute)
	ctx := context.Background()

	if res, err := a.Allow(ctx, "1.2.3.4"); err != nil || !res.Allowed {
		t.Fatalf("instance a request 1: res=%+v err=%v", res, err)
	}
	if res, err := b.Allow(ctx, "1.2.3.4"); err != nil || !res.Allowed {
		t.Fatalf("instance b request 1: res=%+v err=%v", res, err)
	}
	res, err := a.Allow(ctx, "1.2.3.4")
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	if res.Allowed {
		t.Fatal("expected the shared quota to block the third request")
	}
	if res.RetryAfter <= 0 {
		t.Fatalf("expected positive RetryAfter, got %v", res.RetryAfter)
	}
}

func TestRedis_WindowExpires(t *testing.T) {
	l, mr := newTestRedisLimiter(t, "rl", 1, time.Minute)
	ctx := context.Background()

	if res, _ := l.Allow(ctx, "ip"); !res.Allowed {
		t.Fatal("expected first request allowed")
	}
	if res, _ := l.Allow(ctx, "ip"); res.Allowed {
		t.Fatal("expected second request blocked")
	}

	mr.FastForward(61 * time.Second)

	if res, _ := l.Allow(ctx, "ip"); !res.Allowed {
		t.Fatal("expected allowed after window expiry")
	}
}

func TestRedis_KeysAreIndependent(t *testing.T) {
	l, _ := newTestRedisLimiter(t, "rl", 1, time.Minute)
	ctx := context.Background()

	if res, _ := l.Allow(ctx, "a"); !res.Allowed {
		t.Fatal("expected a allowed")
	}
	if res, _ := l.Allow(ctx, "b"); !res.Allowed {
		t.Fatal("expected b allowed independently")
	}
}

func TestRedis_ErrorOnUnavailableBackend(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	l := NewRedis(client, "rl", 10, time.Minute)

	mr.Close()

	if _, err := l.Allow(context.Background(), "ip"); err == nil {
		t.Fatal("expected an error when Redis is unavailable")
	}
	_ = client.Close()
}

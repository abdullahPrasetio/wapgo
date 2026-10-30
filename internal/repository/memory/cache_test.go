package memory

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/abdullahPrasetio/wapgo/internal/domain/repository"
)

type payload struct {
	Name string
}

func newTestCacher(ns string) (*Cacher, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	c := New(ns)
	c.now = func() time.Time { return now }
	return c, &now
}

func TestSetGet(t *testing.T) {
	c, _ := newTestCacher("users")
	ctx := context.Background()

	if err := c.Set(ctx, "1", payload{Name: "Alice"}, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got payload
	if err := c.Get(ctx, "1", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Alice" {
		t.Fatalf("got %q, want Alice", got.Name)
	}
}

func TestGet_CacheMiss(t *testing.T) {
	c, _ := newTestCacher("")
	if err := c.Get(context.Background(), "missing", &payload{}); err != domainrepo.ErrCacheMiss {
		t.Fatalf("got %v, want ErrCacheMiss", err)
	}
}

func TestTTLExpiry(t *testing.T) {
	c, now := newTestCacher("")
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	*now = now.Add(59 * time.Second)
	if err := c.Get(ctx, "k", new(string)); err != nil {
		t.Fatalf("expected hit before expiry, got %v", err)
	}

	*now = now.Add(2 * time.Second)
	if err := c.Get(ctx, "k", new(string)); err != domainrepo.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss after expiry, got %v", err)
	}
	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Fatal("expected expired key to be gone")
	}
}

func TestNoTTL_NeverExpires(t *testing.T) {
	c, now := newTestCacher("")
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	*now = now.Add(365 * 24 * time.Hour)
	if err := c.Get(ctx, "k", new(string)); err != nil {
		t.Fatalf("expected hit without TTL, got %v", err)
	}
}

func TestDelAndExists(t *testing.T) {
	c, _ := newTestCacher("ns")
	ctx := context.Background()

	if err := c.Set(ctx, "a", "v1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "b", "v2", time.Minute); err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Exists(ctx, "a"); !ok {
		t.Fatal("expected a to exist")
	}
	if err := c.Del(ctx, "a", "b"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if err := c.Get(ctx, "a", new(string)); err != domainrepo.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss after Del, got %v", err)
	}
	if ok, _ := c.Exists(ctx, "b"); ok {
		t.Fatal("expected b to be gone")
	}
}

func TestSet_MarshalError(t *testing.T) {
	c, _ := newTestCacher("")
	if err := c.Set(context.Background(), "k", make(chan int), time.Minute); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestNamespaceIsolation(t *testing.T) {
	ctx := context.Background()
	a := New("a")
	b := New("b")

	if err := a.Set(ctx, "k", "from-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := b.Get(ctx, "k", new(string)); err != domainrepo.ErrCacheMiss {
		t.Fatalf("expected namespace isolation, got %v", err)
	}
}

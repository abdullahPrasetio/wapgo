package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedisBlacklist(t *testing.T) (*RedisBlacklist, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return NewRedisBlacklist(client), mr
}

func TestRedisBlacklist_RevokeAndIsRevoked(t *testing.T) {
	bl, _ := newTestRedisBlacklist(t)
	ctx := context.Background()

	jti := "test-jti-1"

	revoked, err := bl.IsRevoked(ctx, jti)
	if err != nil || revoked {
		t.Fatalf("expected not revoked, got revoked=%v err=%v", revoked, err)
	}

	if err := bl.Revoke(ctx, jti, time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	revoked, err = bl.IsRevoked(ctx, jti)
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("expected token to be revoked")
	}
}

func TestRedisBlacklist_Expiry(t *testing.T) {
	bl, mr := newTestRedisBlacklist(t)
	ctx := context.Background()

	jti := "expiring-jti"
	if err := bl.Revoke(ctx, jti, time.Second); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	revoked, _ := bl.IsRevoked(ctx, jti)
	if !revoked {
		t.Fatal("expected revoked before expiry")
	}

	mr.FastForward(2 * time.Second)

	revoked, _ = bl.IsRevoked(ctx, jti)
	if revoked {
		t.Fatal("expected not revoked after expiry")
	}
}

func TestRedisBlacklist_EmptyJTI(t *testing.T) {
	bl, _ := newTestRedisBlacklist(t)
	ctx := context.Background()

	if err := bl.Revoke(ctx, "", time.Minute); err == nil {
		t.Fatal("expected error for empty jti")
	}

	revoked, err := bl.IsRevoked(ctx, "")
	if err != nil || revoked {
		t.Fatalf("empty jti: revoked=%v err=%v", revoked, err)
	}
}

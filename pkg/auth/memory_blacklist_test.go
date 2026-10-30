package auth

import (
	"context"
	"testing"
	"time"
)

func newTestMemoryBlacklist() (*MemoryBlacklist, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	bl := NewMemoryBlacklist()
	bl.now = func() time.Time { return now }
	return bl, &now
}

func TestMemoryBlacklist_RevokeAndIsRevoked(t *testing.T) {
	bl, _ := newTestMemoryBlacklist()
	ctx := context.Background()

	revoked, err := bl.IsRevoked(ctx, "jti-1")
	if err != nil || revoked {
		t.Fatalf("expected not revoked, got revoked=%v err=%v", revoked, err)
	}

	if err := bl.Revoke(ctx, "jti-1", time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	revoked, err = bl.IsRevoked(ctx, "jti-1")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("expected token to be revoked")
	}
}

func TestMemoryBlacklist_Expiry(t *testing.T) {
	bl, now := newTestMemoryBlacklist()
	ctx := context.Background()

	if err := bl.Revoke(ctx, "jti", time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	*now = now.Add(30 * time.Second)
	if revoked, _ := bl.IsRevoked(ctx, "jti"); !revoked {
		t.Fatal("expected revoked before expiry")
	}

	*now = now.Add(31 * time.Second)
	if revoked, _ := bl.IsRevoked(ctx, "jti"); revoked {
		t.Fatal("expected not revoked after expiry")
	}
}

func TestMemoryBlacklist_EmptyJTI(t *testing.T) {
	bl, _ := newTestMemoryBlacklist()
	ctx := context.Background()

	if err := bl.Revoke(ctx, "", time.Minute); err == nil {
		t.Fatal("expected error for empty jti")
	}
	revoked, err := bl.IsRevoked(ctx, "")
	if err != nil || revoked {
		t.Fatalf("empty jti: revoked=%v err=%v", revoked, err)
	}
}

func TestMemoryBlacklist_IndependentJTIs(t *testing.T) {
	bl, _ := newTestMemoryBlacklist()
	ctx := context.Background()

	if err := bl.Revoke(ctx, "a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if revoked, _ := bl.IsRevoked(ctx, "b"); revoked {
		t.Fatal("expected separate JTIs to be independent")
	}
}

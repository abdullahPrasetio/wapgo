//go:build integration

// Integration tests for the Redis cache repository.
// Requires Docker. Run with: go test -tags=integration ./internal/integration/...
package integration

import (
	"context"
	"testing"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	redisrepo "github.com/abdullahPrasetio/wapgo/internal/repository/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCacher(t *testing.T) {
	ctx := context.Background()

	// ── Spin up Redis container ───────────────────────────────────────────────
	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { redisContainer.Terminate(ctx) }) //nolint:errcheck

	addr, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := redisclient.NewClient(&redisclient.Options{Addr: addr})
	t.Cleanup(func() { client.Close() }) //nolint:errcheck

	require.NoError(t, client.Ping(ctx).Err())

	cacher := redisrepo.New(client, "test")

	// ── Set + Get ─────────────────────────────────────────────────────────────
	t.Run("set and get", func(t *testing.T) {
		type payload struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		want := payload{ID: 1, Name: "Alice"}
		require.NoError(t, cacher.Set(ctx, "user:1", want, time.Minute))

		var got payload
		require.NoError(t, cacher.Get(ctx, "user:1", &got))
		assert.Equal(t, want, got)
	})

	// ── Cache miss ────────────────────────────────────────────────────────────
	t.Run("cache miss", func(t *testing.T) {
		var dest any
		err := cacher.Get(ctx, "nonexistent", &dest)
		assert.ErrorIs(t, err, redisrepo.ErrCacheMiss)
	})

	// ── Exists ────────────────────────────────────────────────────────────────
	t.Run("exists", func(t *testing.T) {
		require.NoError(t, cacher.Set(ctx, "exists-key", "val", time.Minute))

		ok, err := cacher.Exists(ctx, "exists-key")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = cacher.Exists(ctx, "missing-key")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	// ── Del ───────────────────────────────────────────────────────────────────
	t.Run("del", func(t *testing.T) {
		require.NoError(t, cacher.Set(ctx, "del-key", "val", time.Minute))
		require.NoError(t, cacher.Del(ctx, "del-key"))

		ok, err := cacher.Exists(ctx, "del-key")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	// ── TTL expiry ────────────────────────────────────────────────────────────
	t.Run("ttl expiry", func(t *testing.T) {
		require.NoError(t, cacher.Set(ctx, "ttl-key", "value", 50*time.Millisecond))

		ok, err := cacher.Exists(ctx, "ttl-key")
		require.NoError(t, err)
		assert.True(t, ok)

		time.Sleep(100 * time.Millisecond)

		ok, err = cacher.Exists(ctx, "ttl-key")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

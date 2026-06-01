package redis_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redistore "github.com/abdullahPrasetio/wapgo/internal/repository/redis"
)

func newTestCache(t *testing.T, ns string) *redistore.RedisCacher {
	t.Helper()
	s := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	return redistore.New(client, ns)
}

func TestSetGet(t *testing.T) {
	c := newTestCache(t, "users")
	ctx := context.Background()

	type payload struct{ Name string }
	require.NoError(t, c.Set(ctx, "1", payload{Name: "Alice"}, time.Minute))

	var got payload
	require.NoError(t, c.Get(ctx, "1", &got))
	assert.Equal(t, "Alice", got.Name)
}

func TestGet_CacheMiss(t *testing.T) {
	c := newTestCache(t, "")
	err := c.Get(context.Background(), "missing", &struct{}{})
	assert.ErrorIs(t, err, redistore.ErrCacheMiss)
}

func TestSet_MarshalError(t *testing.T) {
	c := newTestCache(t, "")
	// channels cannot be marshaled to JSON
	err := c.Set(context.Background(), "k", make(chan int), time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache set marshal")
}

func TestGet_UnmarshalError(t *testing.T) {
	s := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	// store raw invalid JSON directly, bypassing the cache
	require.NoError(t, client.Set(context.Background(), "broken", "not-json{{{", 0).Err())

	c := redistore.New(client, "")
	var dest map[string]interface{}
	err := c.Get(context.Background(), "broken", &dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache get unmarshal")
}

func TestDel(t *testing.T) {
	c := newTestCache(t, "ns")
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "a", "v1", time.Minute))
	require.NoError(t, c.Set(ctx, "b", "v2", time.Minute))
	require.NoError(t, c.Del(ctx, "a", "b"))

	assert.ErrorIs(t, c.Get(ctx, "a", new(string)), redistore.ErrCacheMiss)
	assert.ErrorIs(t, c.Get(ctx, "b", new(string)), redistore.ErrCacheMiss)
}

func TestExists(t *testing.T) {
	c := newTestCache(t, "")
	ctx := context.Background()

	ok, err := c.Exists(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, c.Set(ctx, "present", 1, time.Minute))
	ok, err = c.Exists(ctx, "present")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestKeyPrefixing(t *testing.T) {
	s := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	c := redistore.New(client, "myns")

	require.NoError(t, c.Set(context.Background(), "key", "hello", 0))

	// Verify the actual Redis key carries the namespace prefix.
	raw, err := client.Get(context.Background(), "myns:key").Result()
	require.NoError(t, err)

	var got string
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	assert.Equal(t, "hello", got)
}

func TestSet_TTL_Expiry(t *testing.T) {
	s := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	c := redistore.New(client, "")
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "exp", "val", 5*time.Second))

	ok, err := c.Exists(ctx, "exp")
	require.NoError(t, err)
	assert.True(t, ok)

	s.FastForward(6 * time.Second)

	assert.ErrorIs(t, c.Get(ctx, "exp", new(string)), redistore.ErrCacheMiss)
}

func TestNoPrefix(t *testing.T) {
	c := newTestCache(t, "")
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "bare", 42, 0))
	var got int
	require.NoError(t, c.Get(ctx, "bare", &got))
	assert.Equal(t, 42, got)
}

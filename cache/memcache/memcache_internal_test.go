package memcache

import (
	"context"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemcacheBatch_NoServer(t *testing.T) {
	// Internal test (package memcache) that constructs a memcacheCache
	// directly with no servers. All operations fail — exercises error
	// accumulation and goroutine patterns without needing a real memcache.
	t.Parallel()

	c := &memcacheCache[string, string]{
		client: memcache.New(), // client with no servers — all ops will fail
	}

	t.Run("mget_returns_empty_on_error", func(t *testing.T) {
		result := c.MGetFunc(context.Background(), "a", "b")
		assert.Empty(t, result)
	})

	t.Run("mset_returns_error", func(t *testing.T) {
		err := c.MSetFunc(context.Background(), map[string]string{"a": "1"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "cache/memcache")
	})

	t.Run("mdel_returns_error", func(t *testing.T) {
		err := c.MDelFunc(context.Background(), "a")
		require.Error(t, err)
		assert.ErrorContains(t, err, "cache/memcache")
	})

	t.Run("mset_empty_map_no_error", func(t *testing.T) {
		err := c.MSetFunc(context.Background(), map[string]string{})
		require.NoError(t, err)
	})

	t.Run("mdel_empty_keys_no_error", func(t *testing.T) {
		err := c.MDelFunc(context.Background())
		require.NoError(t, err)
	})
}

func TestResolveTTL(t *testing.T) {
	t.Parallel()

	t.Run("positive_ttl_returns_seconds", func(t *testing.T) {
		t.Parallel()

		c := &memcacheCache[string, string]{}
		got := c.resolveTTL(5 * time.Second)

		assert.Equal(t, int32(5), got)
	})

	t.Run("zero_ttl_falls_to_default", func(t *testing.T) {
		t.Parallel()

		c := &memcacheCache[string, string]{defaultTTL: 10 * time.Second}
		got := c.resolveTTL(0)

		assert.Equal(t, int32(10), got)
	})

	t.Run("no_ttl_and_no_default_returns_zero", func(t *testing.T) {
		t.Parallel()

		c := &memcacheCache[string, string]{}
		got := c.resolveTTL()

		assert.Equal(t, int32(0), got)
	})

	t.Run("sub_second_ttl_returns_1", func(t *testing.T) {
		t.Parallel()

		c := &memcacheCache[string, string]{}
		got := c.resolveTTL(100 * time.Millisecond)

		assert.Equal(t, int32(1), got)
	})

	t.Run("sub_second_default_ttl_returns_1", func(t *testing.T) {
		t.Parallel()

		c := &memcacheCache[string, string]{defaultTTL: 500 * time.Millisecond}
		got := c.resolveTTL()

		assert.Equal(t, int32(1), got)
	})

	t.Run("no_ttl_with_default_returns_seconds", func(t *testing.T) {
		t.Parallel()

		c := &memcacheCache[string, string]{defaultTTL: 30 * time.Second}
		got := c.resolveTTL()

		assert.Equal(t, int32(30), got)
	})
}

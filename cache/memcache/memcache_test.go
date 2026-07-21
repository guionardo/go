package memcache_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/guionardo/go/cache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoMemcache(t *testing.T) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", "localhost:11211", 100*time.Millisecond)
	if err != nil {
		t.Skip("memcache not available on localhost:11211")
	}
	conn.Close()
}

func TestMemcacheCache_SetGet(t *testing.T) {
	skipIfNoMemcache(t)

	t.Run("set_and_get_returns_value", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		err := c.Set(context.Background(), "memcache_test_set_get", "v")
		require.NoError(t, err)

		got, err := c.Get(context.Background(), "memcache_test_set_get")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		_, err := c.Get(context.Background(), "nonexistent_key")
		require.Error(t, err)
		assert.ErrorContains(t, err, "key not found")
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		_ = c.Set(context.Background(), "memcache_test_delete", "v")

		err := c.Delete(context.Background(), "memcache_test_delete")
		require.NoError(t, err)

		_, err = c.Get(context.Background(), "memcache_test_delete")
		require.Error(t, err)
	})

	t.Run("get_or_set_computes", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		got, err := c.GetOrSet(context.Background(), "memcache_test_gos", func() (string, error) {
			return "computed", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "computed", got)
	})
}

func TestMemcacheCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("close_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := memcache.New[string, string]()
		err := c.Close()
		require.NoError(t, err)
	})

	t.Run("close_is_idempotent", func(t *testing.T) {
		t.Parallel()

		c := memcache.New[string, string]()
		_ = c.Close()
		err := c.Close()
		require.NoError(t, err)
	})
}

func TestMemcacheCache_WithOptions(t *testing.T) {
	t.Parallel()

	c := memcache.New[string, string](
		memcache.WithTimeout(200*time.Millisecond),
		memcache.WithMaxIdleConns(5),
	)
	defer c.Close()
	err := c.Close()
	require.NoError(t, err)
}

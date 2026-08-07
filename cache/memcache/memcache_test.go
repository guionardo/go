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

		err := c.Set(t.Context(), "memcache_test_set_get", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "memcache_test_set_get")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		_, err := c.Get(t.Context(), "nonexistent_key")
		require.Error(t, err)
		assert.ErrorContains(t, err, "key not found")
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		_ = c.Set(t.Context(), "memcache_test_delete", "v")

		err := c.Delete(t.Context(), "memcache_test_delete")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "memcache_test_delete")
		require.Error(t, err)
	})

	t.Run("get_or_set_computes", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		got, err := c.GetOrSet(t.Context(), "memcache_test_gos", func(context.Context) (string, error) {
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

func TestMemcacheCache_Batch(t *testing.T) {
	skipIfNoMemcache(t)

	t.Run("mget_returns_found", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		require.NoError(t, c.Set(t.Context(), "mc_mget_a", "v1"))
		require.NoError(t, c.Set(t.Context(), "mc_mget_b", "v2"))

		result := c.MGet(t.Context(), "mc_mget_a", "mc_mget_b", "mc_mget_missing")
		assert.Equal(t, "v1", result["mc_mget_a"])
		assert.Equal(t, "v2", result["mc_mget_b"])
		assert.NotContains(t, result, "mc_mget_missing")
	})

	t.Run("mget_empty_keys", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		result := c.MGet(t.Context())
		assert.Empty(t, result)
	})

	t.Run("mset_stores_all", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		err := c.MSet(t.Context(), map[string]string{"mc_mset_a": "va", "mc_mset_b": "vb"})
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "mc_mset_a")
		require.NoError(t, err)
		assert.Equal(t, "va", got)

		got, err = c.Get(t.Context(), "mc_mset_b")
		require.NoError(t, err)
		assert.Equal(t, "vb", got)
	})

	t.Run("mdel_deletes", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		require.NoError(t, c.Set(t.Context(), "mc_mdel_a", "va"))
		require.NoError(t, c.Set(t.Context(), "mc_mdel_b", "vb"))

		err := c.MDel(t.Context(), "mc_mdel_a", "mc_mdel_b")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "mc_mdel_a")
		require.Error(t, err)

		_, err = c.Get(t.Context(), "mc_mdel_b")
		require.Error(t, err)
	})

	t.Run("mdel_idempotent", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		err := c.MDel(t.Context(), "mc_mdel_nonexistent")
		require.NoError(t, err)
	})

	t.Run("mdel_empty_keys", func(t *testing.T) {
		c := memcache.New[string, string]()
		defer c.Close()

		err := c.MDel(t.Context())
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

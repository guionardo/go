package redis_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guionardo/go/cache/redis"
)

func skipIfNoRedis(t *testing.T) {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	c := redis.New[string, string](redis.WithAddr(addr))
	err := c.Set(t.Context(), "_test_ping", "pong")
	if err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
	}
	_ = c.Close()
}

func TestRedisCache_SetGet(t *testing.T) {
	t.Parallel()
	skipIfNoRedis(t)

	t.Run("set_and_get_returns_value", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		err := c.Set(t.Context(), "redis_test_set_get", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "redis_test_set_get")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		_, err := c.Get(t.Context(), "redis_test_nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache/redis")
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		_ = c.Set(t.Context(), "redis_test_delete", "v")
		_ = c.Delete(t.Context(), "redis_test_delete")

		_, err := c.Get(t.Context(), "redis_test_delete")
		require.Error(t, err)
	})

	t.Run("get_or_set_computes", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		got, err := c.GetOrSet(
			t.Context(),
			"redis_test_getorset",
			func(context.Context) (string, error) { return "computed", nil },
		)
		require.NoError(t, err)
		assert.Equal(t, "computed", got)
	})

	t.Run("set_and_get_with_ttl", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		err := c.Set(t.Context(), "redis_test_ttl", "ttl-value", 0)
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "redis_test_ttl")
		require.NoError(t, err)
		assert.Equal(t, "ttl-value", got)
	})
}

func TestRedisCache_Batch(t *testing.T) {
	t.Parallel()
	skipIfNoRedis(t)

	t.Run("mget_returns_found", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		require.NoError(t, c.Set(t.Context(), "redis_mget_a", "v1"))
		require.NoError(t, c.Set(t.Context(), "redis_mget_b", "v2"))

		result := c.MGet(t.Context(), "redis_mget_a", "redis_mget_b", "redis_mget_missing")
		assert.Equal(t, "v1", result["redis_mget_a"])
		assert.Equal(t, "v2", result["redis_mget_b"])
		assert.NotContains(t, result, "redis_mget_missing")
	})

	t.Run("mget_empty_keys", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		result := c.MGet(t.Context())
		assert.Empty(t, result)
	})

	t.Run("mset_stores_all", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		err := c.MSet(t.Context(), map[string]string{
			"redis_mset_a": "va",
			"redis_mset_b": "vb",
		})
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "redis_mset_a")
		require.NoError(t, err)
		assert.Equal(t, "va", got)

		got, err = c.Get(t.Context(), "redis_mset_b")
		require.NoError(t, err)
		assert.Equal(t, "vb", got)
	})

	t.Run("mset_with_ttl", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		err := c.MSet(t.Context(), map[string]string{"k": "v"}, 0)
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("mdel_deletes", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		_ = c.Set(t.Context(), "redis_mdel_a", "a")
		_ = c.Set(t.Context(), "redis_mdel_b", "b")

		err := c.MDel(t.Context(), "redis_mdel_a", "redis_mdel_b")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "redis_mdel_a")
		require.Error(t, err)
		_, err = c.Get(t.Context(), "redis_mdel_b")
		require.Error(t, err)
	})

	t.Run("mdel_empty_keys", func(t *testing.T) {
		t.Parallel()

		c := redis.New[string, string]()
		err := c.MDel(t.Context())
		require.NoError(t, err)
	})
}

func TestRedisCache_Close(t *testing.T) {
	t.Parallel()

	c := redis.New[string, string]()
	require.NoError(t, c.Close())
}

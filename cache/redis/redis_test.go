package redis_test

import (
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
			func() (string, error) { return "computed", nil },
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

func TestRedisCache_Close(t *testing.T) {
	t.Parallel()

	c := redis.New[string, string]()
	require.NoError(t, c.Close())
}

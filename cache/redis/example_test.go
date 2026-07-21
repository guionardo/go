package redis_test

import (
	"os"
	"testing"

	"github.com/guionardo/go/cache/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoExampleRedis(t *testing.T) {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	c := redis.New[string, string](redis.WithAddr(addr))
	err := c.Set(t.Context(), "_example_ping", "pong")
	if err != nil {
		t.Skip("Redis not available")
	}
	_ = c.Close()
}

func TestRedisExample_SetGet(t *testing.T) {
	skipIfNoExampleRedis(t)

	c := redis.New[string, string](redis.WithAddr("localhost:6379"))

	err := c.Set(t.Context(), "example", "redis-value")
	require.NoError(t, err)

	value, err := c.Get(t.Context(), "example")
	require.NoError(t, err)
	assert.Equal(t, "redis-value", value)

	err = c.Close()
	require.NoError(t, err)
}

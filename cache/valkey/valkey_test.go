package valkey_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guionardo/go/cache/valkey"
)

func skipIfNoValkey(t *testing.T) {
	t.Helper()

	addr := os.Getenv("VALKEY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	c := valkey.New[string, string](valkey.WithAddr(addr))
	err := c.Set(t.Context(), "_test_ping", "pong")
	if err != nil {
		t.Skipf("Valkey not available at %s: %v", addr, err)
	}
	_ = c.Close()
}

func TestValkeyCache_SetGet(t *testing.T) {
	t.Parallel()
	skipIfNoValkey(t)

	t.Run("set_and_get_returns_value", func(t *testing.T) {
		t.Parallel()

		c := valkey.New[string, string]()
		err := c.Set(t.Context(), "valkey_test_set_get", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "valkey_test_set_get")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		t.Parallel()

		c := valkey.New[string, string]()
		_, err := c.Get(t.Context(), "valkey_test_nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache/valkey")
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		t.Parallel()

		c := valkey.New[string, string]()
		_ = c.Set(t.Context(), "valkey_test_delete", "v")
		_ = c.Delete(t.Context(), "valkey_test_delete")

		_, err := c.Get(t.Context(), "valkey_test_delete")
		require.Error(t, err)
	})

	t.Run("get_or_set_computes", func(t *testing.T) {
		t.Parallel()

		c := valkey.New[string, string]()
		got, err := c.GetOrSet(
			t.Context(),
			"valkey_test_getorset",
			func() (string, error) { return "computed", nil },
		)
		require.NoError(t, err)
		assert.Equal(t, "computed", got)
	})

	t.Run("set_and_get_with_ttl", func(t *testing.T) {
		t.Parallel()

		c := valkey.New[string, string]()
		err := c.Set(t.Context(), "valkey_test_ttl", "ttl-value", 0)
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "valkey_test_ttl")
		require.NoError(t, err)
		assert.Equal(t, "ttl-value", got)
	})
}

func TestValkeyCache_Close(t *testing.T) {
	t.Parallel()

	c := valkey.New[string, string]()
	require.NoError(t, c.Close())
}

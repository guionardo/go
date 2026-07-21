package valkey_test

import (
	"os"
	"testing"

	"github.com/guionardo/go/cache/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoExampleValkey(t *testing.T) {
	t.Helper()

	addr := os.Getenv("VALKEY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	c := valkey.New[string, string](valkey.WithAddr(addr))
	err := c.Set(t.Context(), "_example_ping", "pong")
	if err != nil {
		t.Skip("Valkey not available")
	}
	_ = c.Close()
}

func TestValkeyExample_SetGet(t *testing.T) {
	skipIfNoExampleValkey(t)

	c := valkey.New[string, string](valkey.WithAddr("localhost:6379"))

	err := c.Set(t.Context(), "example", "valkey-value")
	require.NoError(t, err)

	value, err := c.Get(t.Context(), "example")
	require.NoError(t, err)
	assert.Equal(t, "valkey-value", value)

	err = c.Close()
	require.NoError(t, err)
}

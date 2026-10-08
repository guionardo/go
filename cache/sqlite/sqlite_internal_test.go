package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTTL(t *testing.T) {
	t.Parallel()

	t.Run("per_key_ttl_overrides_default", func(t *testing.T) {
		t.Parallel()

		c := &sqliteCache[string, string]{defaultTTL: 10 * time.Second}
		expiresAt, ok := c.resolveTTL(30 * time.Second)

		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(30*time.Second), time.Unix(0, expiresAt), time.Second)
	})

	t.Run("zero_ttl_uses_default", func(t *testing.T) {
		t.Parallel()

		c := &sqliteCache[string, string]{defaultTTL: 10 * time.Second}
		expiresAt, ok := c.resolveTTL(0)

		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(10*time.Second), time.Unix(0, expiresAt), time.Second)
	})

	t.Run("no_ttl_uses_default", func(t *testing.T) {
		t.Parallel()

		c := &sqliteCache[string, string]{defaultTTL: 30 * time.Second}
		expiresAt, ok := c.resolveTTL()

		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(30*time.Second), time.Unix(0, expiresAt), time.Second)
	})

	t.Run("no_ttl_no_default_returns_none", func(t *testing.T) {
		t.Parallel()

		c := &sqliteCache[string, string]{}
		expiresAt, ok := c.resolveTTL()

		assert.False(t, ok)
		assert.Zero(t, expiresAt)
	})
}

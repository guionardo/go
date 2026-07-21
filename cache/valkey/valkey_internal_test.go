package valkey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveTTL(t *testing.T) {
	t.Parallel()

	t.Run("per_key_ttl_overrides_default", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 10 * time.Second}
		got := c.resolveTTL(30 * time.Second)

		assert.Equal(t, 30*time.Second, got)
	})

	t.Run("zero_ttl_falls_to_default", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 10 * time.Second}
		got := c.resolveTTL(0)

		assert.Equal(t, 10*time.Second, got)
	})

	t.Run("no_ttl_with_default", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 30 * time.Second}
		got := c.resolveTTL()

		assert.Equal(t, 30*time.Second, got)
	})

	t.Run("no_ttl_and_no_default_returns_zero", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{}
		got := c.resolveTTL()

		assert.Equal(t, time.Duration(0), got)
	})
}

package postgres

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

		assert.NotNil(t, got)
		assert.WithinDuration(t, time.Now().Add(30*time.Second), *got, time.Second)
	})

	t.Run("zero_ttl_uses_default", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 10 * time.Second}
		got := c.resolveTTL(0)

		assert.NotNil(t, got)
		assert.WithinDuration(t, time.Now().Add(10*time.Second), *got, time.Second)
	})

	t.Run("no_ttl_uses_default", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 30 * time.Second}
		got := c.resolveTTL()

		assert.NotNil(t, got)
		assert.WithinDuration(t, time.Now().Add(30*time.Second), *got, time.Second)
	})

	t.Run("no_ttl_no_default_returns_nil", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{}
		got := c.resolveTTL()

		assert.Nil(t, got)
	})
}

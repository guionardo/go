package memcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveTTL(t *testing.T) {
	t.Parallel()

	t.Run("positive_ttl_returns_seconds", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{}
		got := c.resolveTTL(5 * time.Second)

		assert.Equal(t, int32(5), got)
	})

	t.Run("zero_ttl_falls_to_default", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 10 * time.Second}
		got := c.resolveTTL(0)

		assert.Equal(t, int32(10), got)
	})

	t.Run("no_ttl_and_no_default_returns_zero", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{}
		got := c.resolveTTL()

		assert.Equal(t, int32(0), got)
	})

	t.Run("sub_second_ttl_returns_1", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{}
		got := c.resolveTTL(100 * time.Millisecond)

		assert.Equal(t, int32(1), got)
	})

	t.Run("sub_second_default_ttl_returns_1", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 500 * time.Millisecond}
		got := c.resolveTTL()

		assert.Equal(t, int32(1), got)
	})

	t.Run("no_ttl_with_default_returns_seconds", func(t *testing.T) {
		t.Parallel()

		c := &Cache[string, string]{defaultTTL: 30 * time.Second}
		got := c.resolveTTL()

		assert.Equal(t, int32(30), got)
	})
}

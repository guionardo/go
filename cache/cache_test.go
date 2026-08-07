package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/stretchr/testify/assert"
)

func TestCacheInterface(t *testing.T) {
	t.Parallel()

	t.Run("type_assertion_string_string", func(t *testing.T) {
		t.Parallel()

		var _ cache.Cache[string, string]
	})

	t.Run("type_assertion_int_bytes", func(t *testing.T) {
		t.Parallel()

		var _ cache.Cache[int, []byte]
	})

	t.Run("type_assertion_struct_struct", func(t *testing.T) {
		t.Parallel()

		type X struct{}
		var _ cache.Cache[string, X]
	})

	_ = context.Background // suppress unused import warning
}

func TestCacheSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("err_miss", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, cache.ErrMiss)
		assert.Contains(t, cache.ErrMiss.Error(), "not found")
	})

	t.Run("err_closed", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, cache.ErrClosed)
		assert.Contains(t, cache.ErrClosed.Error(), "closed")
	})
}

func TestCacheOption(t *testing.T) {
	t.Parallel()

	t.Run("with_default_ttl_compiles", func(t *testing.T) {
		t.Parallel()

		opt := cache.WithDefaultTTL(5 * time.Minute)
		assert.NotNil(t, opt)
	})
}

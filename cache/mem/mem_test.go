package mem_test

import (
	"sync"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemCache_SetGet(t *testing.T) { //nolint:funlen
	t.Parallel()

	t.Run("set_and_get_returns_value", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()

		err := c.Set(t.Context(), "k", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()

		_, err := c.Get(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorContains(t, err, "key not found")
	})

	t.Run("get_expired_returns_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](cache.WithDefaultTTL(1 * time.Millisecond))
		_ = c.Set(t.Context(), "k", "v")
		time.Sleep(10 * time.Millisecond)

		_, err := c.Get(t.Context(), "k")
		require.Error(t, err)
	})

	t.Run("per_key_ttl_overrides_default", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](cache.WithDefaultTTL(1 * time.Hour))
		err := c.Set(t.Context(), "k", "v", 1*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)

		_, err = c.Get(t.Context(), "k")
		require.Error(t, err)
	})

	t.Run("set_without_ttl_no_expiry", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()

		err := c.Set(t.Context(), "k", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})
}

func TestMemCache_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete_removes_key", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()
		_ = c.Set(t.Context(), "k", "v")

		err := c.Delete(t.Context(), "k")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "k")
		require.Error(t, err)
	})

	t.Run("delete_missing_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()

		err := c.Delete(t.Context(), "nonexistent")
		require.NoError(t, err)
	})
}

func TestMemCache_GetOrSet(t *testing.T) {
	t.Parallel()

	t.Run("get_or_set_returns_existing", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()
		_ = c.Set(t.Context(), "k", "v")

		got, err := c.GetOrSet(t.Context(), "k", func() (string, error) {
			return "computed", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_or_set_computes_when_missing", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()

		got, err := c.GetOrSet(t.Context(), "k", func() (string, error) {
			return "computed", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "computed", got)
	})
}

func TestMemCache_GetOrSet_SetterError(t *testing.T) {
	t.Parallel()

	c := mem.New[string, string]()

	_, err := c.GetOrSet(t.Context(), "k", func() (string, error) {
		return "", assert.AnError
	})
	require.Error(t, err)
}

func TestMemCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("close_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()

		err := c.Close()
		require.NoError(t, err)
	})

	t.Run("close_is_idempotent", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string]()
		_ = c.Close()

		err := c.Close()
		require.NoError(t, err)
	})
}

func TestMemCache_Concurrent(t *testing.T) {
	t.Parallel()

	c := mem.New[int, int]()
	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range 100 {
				key := i*1000 + j
				err := c.Set(t.Context(), key, j)
				assert.NoError(t, err)

				got, err := c.Get(t.Context(), key)
				if err == nil {
					assert.Equal(t, j, got)
				}
			}
		}()
	}

	wg.Wait()
}

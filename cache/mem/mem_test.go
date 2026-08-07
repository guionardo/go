package mem_test

import (
	"context"
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

		c := mem.New[string, string](t.Context())

		err := c.Set(t.Context(), "k", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())

		_, err := c.Get(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorContains(t, err, "key not found")
	})

	t.Run("get_expired_returns_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context(), mem.WithDefaultTTL(1*time.Millisecond))
		_ = c.Set(t.Context(), "k", "v")
		time.Sleep(10 * time.Millisecond)

		_, err := c.Get(t.Context(), "k")
		require.Error(t, err)
	})

	t.Run("per_key_ttl_overrides_default", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context(), mem.WithDefaultTTL(1*time.Hour))
		err := c.Set(t.Context(), "k", "v", 1*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)

		_, err = c.Get(t.Context(), "k")
		require.Error(t, err)
	})

	t.Run("set_without_ttl_no_expiry", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())

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

		c := mem.New[string, string](t.Context())
		_ = c.Set(t.Context(), "k", "v")

		err := c.Delete(t.Context(), "k")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "k")
		require.Error(t, err)
	})

	t.Run("delete_missing_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())

		err := c.Delete(t.Context(), "nonexistent")
		require.NoError(t, err)
	})
}

func TestMemCache_GetOrSet(t *testing.T) {
	t.Parallel()

	t.Run("get_or_set_returns_existing", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())
		_ = c.Set(t.Context(), "k", "v")

		got, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
			return "computed", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_or_set_computes_when_missing", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())

		got, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
			return "computed", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "computed", got)
	})
}

func TestMemCache_GetOrSet_SetterError(t *testing.T) {
	t.Parallel()

	c := mem.New[string, string](t.Context())

	_, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
		return "", assert.AnError
	})
	require.Error(t, err)
}

func TestMemCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("close_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())

		err := c.Close()
		require.NoError(t, err)
	})

	t.Run("close_is_idempotent", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context())
		_ = c.Close()

		err := c.Close()
		require.NoError(t, err)
	})
}

func TestMemCache_Batch(t *testing.T) {
	t.Parallel()

	t.Run("mget_returns_only_found_keys", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context()).(cache.BatchCache[string, string])
		require.NoError(t, c.Set(t.Context(), "a", "1"))
		require.NoError(t, c.Set(t.Context(), "b", "2"))

		result := c.MGet(t.Context(), "a", "b", "missing")
		assert.Equal(t, "1", result["a"])
		assert.Equal(t, "2", result["b"])
		assert.NotContains(t, result, "missing")
	})

	t.Run("mget_empty_keys_returns_empty_map", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context()).(cache.BatchCache[string, string])
		result := c.MGet(t.Context())
		assert.Empty(t, result)
	})

	t.Run("mset_stores_all_keys", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context()).(cache.BatchCache[string, string])
		err := c.MSet(t.Context(), map[string]string{"a": "1", "b": "2"})
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "a")
		require.NoError(t, err)
		assert.Equal(t, "1", got)

		got, err = c.Get(t.Context(), "b")
		require.NoError(t, err)
		assert.Equal(t, "2", got)
	})

	t.Run("mset_empty_map_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context()).(cache.BatchCache[string, string])
		err := c.MSet(t.Context(), map[string]string{})
		require.NoError(t, err)
	})

	t.Run("mdel_deletes_all_keys", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context()).(cache.BatchCache[string, string])
		_ = c.Set(t.Context(), "a", "1")
		_ = c.Set(t.Context(), "b", "2")

		err := c.MDel(t.Context(), "a", "b")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "a")
		require.Error(t, err)
		_, err = c.Get(t.Context(), "b")
		require.Error(t, err)
	})

	t.Run("mset_with_ttl_stores_keys", func(t *testing.T) {
		t.Parallel()

		c := mem.New[string, string](t.Context()).(cache.BatchCache[string, string])
		err := c.MSet(t.Context(), map[string]string{"a": "1"}, time.Minute)
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "a")
		require.NoError(t, err)
		assert.Equal(t, "1", got)
	})
}

func TestMemCache_Concurrent(t *testing.T) {
	t.Parallel()

	c := mem.New[int, int](t.Context())
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

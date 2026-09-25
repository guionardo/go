package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCacher is a minimal in-memory implementation of the cache.cacher
// primitive interface, used to exercise concreteCache without a network
// provider.
type fakeCacher[K comparable, V any] struct {
	mu   sync.Mutex
	data map[K]V
	err  error
}

func newFakeCacher[K comparable, V any]() *fakeCacher[K, V] {
	return &fakeCacher[K, V]{data: make(map[K]V)}
}

func (f *fakeCacher[K, V]) GetFunc(_ context.Context, key K) (V, error) {
	var zero V

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return zero, f.err
	}

	if v, ok := f.data[key]; ok {
		return v, nil
	}

	return zero, cache.ErrMiss
}

func (f *fakeCacher[K, V]) SetFunc(_ context.Context, key K, value V, _ ...time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}

	f.data[key] = value

	return nil
}

func (f *fakeCacher[K, V]) DeleteFunc(_ context.Context, key K) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}

	delete(f.data, key)

	return nil
}

func (f *fakeCacher[K, V]) CloseFunc() error {
	return nil
}

func (f *fakeCacher[K, V]) MGetFunc(_ context.Context, keys ...K) map[K]V {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make(map[K]V, len(keys))
	for _, key := range keys {
		if v, ok := f.data[key]; ok {
			result[key] = v
		}
	}
	return result
}

func (f *fakeCacher[K, V]) MSetFunc(_ context.Context, items map[K]V, _ ...time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}

	for key, value := range items {
		f.data[key] = value
	}
	return nil
}

func (f *fakeCacher[K, V]) MDelFunc(_ context.Context, keys ...K) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}

	for _, key := range keys {
		delete(f.data, key)
	}
	return nil
}

func TestConcreteCache_GetSetDelete(t *testing.T) {
	t.Parallel()

	c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())

	t.Run("get_miss_returns_err_miss", func(t *testing.T) {
		t.Parallel()

		_, err := c.Get(t.Context(), "missing")
		require.ErrorIs(t, err, cache.ErrMiss)
	})

	t.Run("set_then_get_returns_value", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		require.NoError(t, c.Set(t.Context(), "k", "v"))

		got, err := c.Get(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("set_with_ttl_is_accepted", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		require.NoError(t, c.Set(t.Context(), "k", "v", time.Minute))

		got, err := c.Get(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		require.NoError(t, c.Set(t.Context(), "k", "v"))
		require.NoError(t, c.Delete(t.Context(), "k"))

		_, err := c.Get(t.Context(), "k")
		require.ErrorIs(t, err, cache.ErrMiss)
	})

	t.Run("provider_error_is_propagated", func(t *testing.T) {
		t.Parallel()

		f := newFakeCacher[string, string]()
		f.err = assert.AnError
		c := cache.NewConcreteCache[string, string](f)

		_, err := c.Get(t.Context(), "k")
		require.ErrorIs(t, err, assert.AnError)
	})
}

func TestConcreteCache_GetOrSet(t *testing.T) {
	t.Parallel()

	t.Run("get_or_set_computes_on_miss", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		got, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
			return computed, nil
		})
		require.NoError(t, err)
		assert.Equal(t, computed, got)
	})

	t.Run("get_or_set_returns_existing", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		require.NoError(t, c.Set(t.Context(), "k", "existing"))

		got, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
			return computed, nil
		})
		require.NoError(t, err)
		assert.Equal(t, "existing", got)
	})

	t.Run("get_or_set_propagates_setter_error", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		_, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
			return "", assert.AnError
		})
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("get_or_set_with_ttl_stores", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		got, err := c.GetOrSet(t.Context(), "k", func(context.Context) (string, error) {
			return computed, nil
		}, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, computed, got)
	})
}

func TestConcreteCache_Close(t *testing.T) {
	t.Parallel()

	c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
	require.NoError(t, c.Close())
}

func TestConcreteCache_Batch(t *testing.T) {
	t.Parallel()

	t.Run("mget_returns_only_found_keys", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		require.NoError(t, c.Set(t.Context(), "a", "1"))
		require.NoError(t, c.Set(t.Context(), "b", "2"))

		result := c.MGet(t.Context(), "a", "b", "missing")
		assert.Equal(t, "1", result["a"])
		assert.Equal(t, "2", result["b"])
		assert.NotContains(t, result, "missing")
	})

	t.Run("mget_empty_keys_returns_empty_map", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		result := c.MGet(t.Context())
		assert.Empty(t, result)
	})

	t.Run("mset_stores_all_keys", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
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

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		err := c.MSet(t.Context(), map[string]string{})
		require.NoError(t, err)
	})

	t.Run("mdel_deletes_all_keys", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		_ = c.Set(t.Context(), "a", "1")
		_ = c.Set(t.Context(), "b", "2")

		err := c.MDel(t.Context(), "a", "b")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "a")
		require.ErrorIs(t, err, cache.ErrMiss)
		_, err = c.Get(t.Context(), "b")
		require.ErrorIs(t, err, cache.ErrMiss)
	})

	t.Run("mdel_empty_keys_does_not_error", func(t *testing.T) {
		t.Parallel()

		c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
		err := c.MDel(t.Context())
		require.NoError(t, err)
	})

	t.Run("batch_with_provider_error_returns_aggregated", func(t *testing.T) {
		t.Parallel()

		f := newFakeCacher[string, string]()
		f.err = assert.AnError
		c := cache.NewConcreteCache[string, string](f)

		err := c.MSet(t.Context(), map[string]string{"a": "1"})
		require.Error(t, err)
	})
}

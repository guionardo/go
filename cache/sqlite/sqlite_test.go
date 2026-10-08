package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRoundTrip(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	err := c.Set(t.Context(), "k", "v")
	require.NoError(t, err)

	got, err := c.Get(t.Context(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	err = c.Delete(t.Context(), "k")
	require.NoError(t, err)

	_, err = c.Get(t.Context(), "k")
	require.Error(t, err)
	require.ErrorIs(t, err, cache.ErrMiss)

	require.NoError(t, c.Close())

	_, err = c.Get(t.Context(), "k")
	require.Error(t, err)
	require.ErrorIs(t, err, sqlite.ErrClosed)
}

func TestBatchPlaceholderSmoke(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	err := c.MSet(t.Context(), map[string]string{"a": "1", "b": "2"})
	require.NoError(t, err)

	got := c.MGet(t.Context(), "a", "b", "missing")
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, got)

	err = c.MDel(t.Context(), "a")
	require.NoError(t, err)

	got = c.MGet(t.Context(), "a", "b")
	assert.Equal(t, map[string]string{"b": "2"}, got)
}

func TestFileCRUD(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithPath(filepath.Join(t.TempDir(), "cache.db")))
	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "file-key", "file-value"))

	got, err := c.Get(t.Context(), "file-key")
	require.NoError(t, err)
	assert.Equal(t, "file-value", got)

	require.NoError(t, c.Delete(t.Context(), "file-key"))

	_, err = c.Get(t.Context(), "file-key")
	require.Error(t, err)
	require.ErrorIs(t, err, cache.ErrMiss)
}

func TestFilePathWithSpaces(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "cache dir with spaces")
	c := sqlite.New[string, string](sqlite.WithPath(filepath.Join(dir, "cache.db")))

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "k", "v"))

	got, err := c.Get(t.Context(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}

func TestTTLExpiry(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "ttl", "v", 40*time.Millisecond))

	got, err := c.Get(t.Context(), "ttl")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	require.Eventually(t, func() bool {
		_, err := c.Get(t.Context(), "ttl")

		return errors.Is(err, cache.ErrMiss)
	}, time.Second, 5*time.Millisecond)
}

func TestDefaultTTLExpiry(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory(), sqlite.WithDefaultTTL(40*time.Millisecond))

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "ttl", "v"))

	got, err := c.Get(t.Context(), "ttl")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	require.Eventually(t, func() bool {
		_, err := c.Get(t.Context(), "ttl")

		return errors.Is(err, cache.ErrMiss)
	}, time.Second, 5*time.Millisecond)
}

func TestNoExpiry(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "forever", "v"))

	// No TTL and no default means no expiry: assert presence twice rather than
	// sleeping out an await that does not exist.
	for range 2 {
		got, err := c.Get(t.Context(), "forever")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	}
}

func TestGetOrSetDedup(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	const goroutines = 20

	var calls atomic.Int64

	start := make(chan struct{})
	results := make([]string, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup

	for i := range goroutines {
		wg.Go(func() {
			<-start

			setter := func(context.Context) (string, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond) // widen the overlap window

				return "deduped", nil
			}

			results[i], errs[i] = c.GetOrSet(context.Background(), "hot", setter)
		})
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), calls.Load(), "setter must run exactly once")

	for i := range goroutines {
		require.NoError(t, errs[i])
		assert.Equal(t, "deduped", results[i])
	}
}

func TestMemoryIsolation(t *testing.T) {
	t.Parallel()

	c1 := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c1.Close() })

	c2 := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c2.Close() })

	require.NoError(t, c1.Set(t.Context(), "k", "v"))

	_, err := c2.Get(t.Context(), "k")
	require.Error(t, err)
	require.ErrorIs(t, err, cache.ErrMiss)
}

func TestParity(t *testing.T) {
	t.Parallel()

	t.Run("int_key_reads_via_stringified_form", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[any, string](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		require.NoError(t, c.Set(t.Context(), 42, "answer"))

		got, err := c.Get(t.Context(), "42")
		require.NoError(t, err)
		assert.Equal(t, "answer", got)
	})

	t.Run("unicode_and_empty_string_values", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		require.NoError(t, c.Set(t.Context(), "unicode", "acentuação 日本語"))

		got, err := c.Get(t.Context(), "unicode")
		require.NoError(t, err)
		assert.Equal(t, "acentuação 日本語", got)

		require.NoError(t, c.Set(t.Context(), "empty", ""))

		got, err = c.Get(t.Context(), "empty")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("struct_values", func(t *testing.T) {
		t.Parallel()

		type payload struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		}

		c := sqlite.New[string, payload](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		want := payload{Name: "app", Tags: []string{"a", "b"}}
		require.NoError(t, c.Set(t.Context(), "struct", want))

		got, err := c.Get(t.Context(), "struct")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestMarshalFailure(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, any](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	err := c.Set(t.Context(), "chan", make(chan int))
	require.Error(t, err)
	require.ErrorContains(t, err, "cache/sqlite")

	// Nothing was stored.
	_, err = c.Get(t.Context(), "chan")
	require.ErrorIs(t, err, cache.ErrMiss)
}

func TestClosedOps(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())
	require.NoError(t, c.Close())

	_, err := c.Get(t.Context(), "k")
	require.ErrorIs(t, err, sqlite.ErrClosed)

	err = c.Set(t.Context(), "k", "v")
	require.ErrorIs(t, err, sqlite.ErrClosed)

	err = c.Delete(t.Context(), "k")
	require.ErrorIs(t, err, sqlite.ErrClosed)

	require.NoError(t, c.Close()) // idempotent
}

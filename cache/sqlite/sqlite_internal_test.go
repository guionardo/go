package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newInternalProvider builds a provider directly (bypassing New) so tests can
// assert on the underlying *sql.DB. The handle is closed via t.Cleanup.
func newInternalProvider(t *testing.T, opts ...Option) *sqliteCache[string, string] {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	c := &sqliteCache[string, string]{defaultTTL: cfg.DefaultTTL}
	c.initErr = c.open(t.Context(), cfg)

	t.Cleanup(func() { _ = c.CloseFunc() })

	return c
}

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

func TestPoolPinned(t *testing.T) {
	t.Parallel()

	t.Run("memory_mode", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		assert.Equal(t, 1, c.db.Stats().MaxOpenConnections)
	})

	t.Run("file_mode", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "cache.db")))
		require.NoError(t, c.initErr)

		assert.Equal(t, 1, c.db.Stats().MaxOpenConnections)
	})
}

func TestPragmaReadBack(t *testing.T) {
	t.Parallel()

	t.Run("file_mode_wal_busy_timeout_synchronous", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "cache.db")))
		require.NoError(t, c.initErr)

		var mode string
		require.NoError(t, c.db.QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&mode))
		assert.Equal(t, "wal", mode)

		var busyTimeout int
		require.NoError(t, c.db.QueryRowContext(t.Context(), "PRAGMA busy_timeout").Scan(&busyTimeout))
		assert.Equal(t, 5000, busyTimeout)

		var synchronous int
		require.NoError(t, c.db.QueryRowContext(t.Context(), "PRAGMA synchronous").Scan(&synchronous))
		assert.Equal(t, 1, synchronous)
	})

	t.Run("memory_mode_busy_timeout", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		var busyTimeout int
		require.NoError(t, c.db.QueryRowContext(t.Context(), "PRAGMA busy_timeout").Scan(&busyTimeout))
		assert.Equal(t, 5000, busyTimeout)
	})
}

// countCacheTables returns the number of sqlite_master entries named
// cache_entries on the provider's pinned connection.
func countCacheTables(t *testing.T, c *sqliteCache[string, string]) int {
	t.Helper()

	var count int
	require.NoError(t, c.db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'cache_entries'").Scan(&count))

	return count
}

// probeConcurrentOpen opens the cache at path, stores and reads back a probe
// value, and returns the first error encountered.
func probeConcurrentOpen(t *testing.T, path string) error {
	t.Helper()

	c := New[string, string](WithPath(path))
	defer func() { _ = c.Close() }()

	if err := c.Set(t.Context(), "probe", "ok"); err != nil {
		return err
	}

	value, err := c.Get(t.Context(), "probe")
	if err != nil {
		return err
	}

	if value != "ok" {
		return assert.AnError
	}

	return nil
}

func TestBootstrapIdempotence(t *testing.T) {
	t.Parallel()

	t.Run("sequential_reopen_leaves_one_table", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "cache.db")

		c1 := newInternalProvider(t, WithPath(path))
		require.NoError(t, c1.initErr)
		require.NoError(t, c1.CloseFunc())

		c2 := newInternalProvider(t, WithPath(path))
		require.NoError(t, c2.initErr)

		assert.Equal(t, 1, countCacheTables(t, c2))
	})

	t.Run("concurrent_first_open_all_usable", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "concurrent.db")

		const workers = 4

		var wg sync.WaitGroup

		errs := make([]error, workers)

		for i := range errs {
			wg.Go(func() {
				errs[i] = probeConcurrentOpen(t, path)
			})
		}

		wg.Wait()

		for i, err := range errs {
			require.NoErrorf(t, err, "worker %d", i)
		}

		c := newInternalProvider(t, WithPath(path))
		require.NoError(t, c.initErr)
		assert.Equal(t, 1, countCacheTables(t, c))
	})
}

func TestDeferredErr(t *testing.T) {
	t.Parallel()

	c := New[string, string](WithPath("bad?path"))
	require.NotNil(t, c)

	t.Cleanup(func() { _ = c.Close() })

	_, err := c.Get(t.Context(), "k")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidPath)
	require.ErrorContains(t, err, "cache/sqlite")

	err = c.Set(t.Context(), "k", "v")
	require.ErrorIs(t, err, ErrInvalidPath)

	err = c.Delete(t.Context(), "k")
	require.ErrorIs(t, err, ErrInvalidPath)

	// Close is cleanup: it returns nil even when the handle deferred an open error.
	require.NoError(t, c.Close())
}

func TestClosedTaxonomy(t *testing.T) {
	t.Parallel()

	c := newInternalProvider(t, WithMemory())
	require.NoError(t, c.initErr)
	require.NoError(t, c.CloseFunc())

	_, err := c.GetFunc(t.Context(), "k")
	require.ErrorIs(t, err, ErrClosed)

	err = c.SetFunc(t.Context(), "k", "v")
	require.ErrorIs(t, err, ErrClosed)

	err = c.DeleteFunc(t.Context(), "k")
	require.ErrorIs(t, err, ErrClosed)

	require.NoError(t, c.CloseFunc()) // idempotent
}

func TestCorruptedRowNotMiss(t *testing.T) {
	t.Parallel()

	c := newInternalProvider(t, WithMemory())
	require.NoError(t, c.initErr)

	// Seed a row whose value is not valid JSON straight through the handle.
	_, err := c.db.ExecContext(t.Context(), UpsertSQL, "corrupt", "{not-json", nil)
	require.NoError(t, err)

	_, err = c.GetFunc(t.Context(), "corrupt")
	require.Error(t, err)
	require.NotErrorIs(t, err, cache.ErrMiss)
	require.ErrorContains(t, err, "cache/sqlite")
}

func TestCanceledContext(t *testing.T) {
	t.Parallel()

	c := newInternalProvider(t, WithMemory())
	require.NoError(t, c.initErr)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.GetFunc(ctx, "k")
	require.Error(t, err)
}

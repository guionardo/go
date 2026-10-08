package sqlite

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

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
			wg.Add(1)

			go func() {
				defer wg.Done()

				c := New[string, string](WithPath(path))
				defer func() { _ = c.Close() }()

				if err := c.Set(t.Context(), "probe", "ok"); err != nil {
					errs[i] = err

					return
				}

				value, err := c.Get(t.Context(), "probe")
				if err != nil {
					errs[i] = err

					return
				}

				if value != "ok" {
					errs[i] = assert.AnError
				}
			}()
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

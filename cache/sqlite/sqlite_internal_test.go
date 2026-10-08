package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ttlCase is one row of the resolveTTL matrix exercised by TestResolveTTL.
// A zero wantUntil means no expiry is expected.
type ttlCase struct {
	name       string
	defaultTTL time.Duration
	ttl        []time.Duration
	wantUntil  time.Duration
}

// ttlMatrix mirrors the postgres resolveTTL contract: a per-call ttl > 0 wins;
// else defaultTTL > 0 applies; else no expiry. Negative and multiple-ttl
// inputs behave per the same predicate (only ttl[0] > 0 counts).
var ttlMatrix = []ttlCase{
	{
		name:       "per_key_ttl_overrides_default",
		defaultTTL: 10 * time.Second,
		ttl:        []time.Duration{30 * time.Second},
		wantUntil:  30 * time.Second,
	},
	{name: "zero_ttl_uses_default", defaultTTL: 10 * time.Second, ttl: []time.Duration{0}, wantUntil: 10 * time.Second},
	{name: "no_ttl_uses_default", defaultTTL: 30 * time.Second, wantUntil: 30 * time.Second},
	{name: "no_ttl_no_default_returns_none", wantUntil: 0},
	{
		name:       "negative_ttl_falls_back_to_default",
		defaultTTL: 10 * time.Second,
		ttl:        []time.Duration{-30 * time.Second},
		wantUntil:  10 * time.Second,
	},
	{name: "negative_ttl_no_default_returns_none", ttl: []time.Duration{-30 * time.Second}, wantUntil: 0},
	{
		name:      "multiple_ttl_args_use_the_first",
		ttl:       []time.Duration{30 * time.Second, time.Hour},
		wantUntil: 30 * time.Second,
	},
}

// newInternalProvider builds a provider directly (bypassing New) so tests can
// assert on the underlying *sql.DB. The handle is closed via t.Cleanup.
func newInternalProvider(t *testing.T, opts ...Option) *sqliteCache[string, string] {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	c := &sqliteCache[string, string]{
		defaultTTL:    cfg.DefaultTTL,
		sweepInterval: cfg.SweepInterval,
	}
	c.initErr = c.open(t.Context(), cfg)

	t.Cleanup(func() { _ = c.CloseFunc() })

	return c
}

func TestResolveTTL(t *testing.T) {
	t.Parallel()

	for _, tc := range ttlMatrix {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &sqliteCache[string, string]{defaultTTL: tc.defaultTTL}
			expiresAt, ok := c.resolveTTL(tc.ttl...)

			if tc.wantUntil == 0 {
				assert.False(t, ok)
				assert.Zero(t, expiresAt)

				return
			}

			require.True(t, ok)
			assert.WithinDuration(t, time.Now().Add(tc.wantUntil), time.Unix(0, expiresAt), time.Second)
		})
	}
}

// TestReadsNeverDeleteRows proves the read path is filter-only (TTL-02): an
// expired Get reports ErrMiss without removing the seeded row.
func TestReadsNeverDeleteRows(t *testing.T) {
	t.Parallel()

	c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "read.db")))
	require.NoError(t, c.initErr)

	_, err := c.db.ExecContext(t.Context(), UpsertSQL, "expired", `"v"`, time.Now().Add(-time.Hour).UnixNano())
	require.NoError(t, err)

	_, err = c.GetFunc(t.Context(), "expired")
	require.ErrorIs(t, err, cache.ErrMiss)

	assert.Equal(t, 1, countKeyRows(t, c.db, "expired"), "reads must never delete rows (TTL-02)")
}

// TestSweepOnOpen proves the open-time sweep is synchronous (TTL-03, D-10):
// expired rows present before New are gone by the time New returns, while
// unexpired rows survive the sweep predicate.
func TestSweepOnOpen(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "sweep.db")

	seed := newInternalProvider(t, WithPath(path))
	require.NoError(t, seed.initErr)

	past := time.Now().Add(-time.Hour).UnixNano()
	future := time.Now().Add(time.Hour).UnixNano()

	_, err := seed.db.ExecContext(t.Context(), UpsertSQL, "stale", `"v"`, past)
	require.NoError(t, err)

	_, err = seed.db.ExecContext(t.Context(), UpsertSQL, "fresh", `"v"`, future)
	require.NoError(t, err)

	require.Equal(t, 1, countKeyRows(t, seed.db, "stale"))
	require.NoError(t, seed.CloseFunc())

	// Reopen through the public constructor: New runs the open-time sweep
	// synchronously before it returns (D-10).
	c := New[string, string](WithPath(path))

	t.Cleanup(func() { _ = c.Close() })

	observer, err := sql.Open("sqlite", path)
	require.NoError(t, err)

	t.Cleanup(func() { _ = observer.Close() })

	assert.Equal(t, 0, countKeyRows(t, observer, "stale"), "expired row must be reclaimed by the open sweep")
	assert.Equal(t, 1, countKeyRows(t, observer, "fresh"), "unexpired row must survive the open sweep")

	// No deferred open error surfaced on the reopened handle.
	require.NoError(t, c.Set(t.Context(), "alive", "ok"))
}

// TestSweepFailureSwallowed proves the best-effort contract (D-09): a sweep
// against a broken handle logs and returns — nothing propagates to callers.
func TestSweepFailureSwallowed(t *testing.T) {
	t.Parallel()

	c := newInternalProvider(t, WithMemory())
	require.NoError(t, c.initErr)

	require.NoError(t, c.db.Close())

	// No panic and no error to assert: the failure is logged and swallowed.
	c.sweep(t.Context())
}

// TestPeriodicSweep proves the opt-in ticker reclaims rows seeded after open
// (TTL-04): only the periodic sweep can remove a row that arrived post-open.
func TestPeriodicSweep(t *testing.T) {
	t.Parallel()

	c := newInternalProvider(t, WithMemory(), WithSweepInterval(10*time.Millisecond))
	require.NoError(t, c.initErr)

	assert.NotNil(t, c.stop)
	assert.NotNil(t, c.done)

	_, err := c.db.ExecContext(t.Context(), UpsertSQL, "tick", `"v"`, time.Now().Add(-time.Hour).UnixNano())
	require.NoError(t, err)
	require.Equal(t, 1, countKeyRows(t, c.db, "tick"))

	require.Eventually(t, func() bool {
		count, err := keyRowCount(c.db, "tick")

		return err == nil && count == 0
	}, time.Second, 5*time.Millisecond)

	require.NoError(t, c.CloseFunc())
	require.NoError(t, c.CloseFunc()) // idempotent
}

// TestSweeperDisabled proves the deliberate divergence from mem/postgres
// (D-08): an unset or non-positive interval creates no sweeper at all.
func TestSweeperDisabled(t *testing.T) {
	t.Parallel()

	t.Run("unset_interval", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		assert.Nil(t, c.stop)
		assert.Nil(t, c.done)
	})

	t.Run("zero_and_negative_intervals", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory(), WithSweepInterval(0), WithSweepInterval(-time.Second))
		require.NoError(t, c.initErr)

		assert.Nil(t, c.stop)
		assert.Nil(t, c.done)
	})
}

// TestOptions_WithSweepInterval covers the WithSweepInterval option and its
// zero-value default (no sweeper) inside this plan's file ownership.
func TestOptions_WithSweepInterval(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.Zero(t, cfg.SweepInterval, "default is no sweeper (D-08)")

	WithSweepInterval(10 * time.Millisecond)(cfg)
	assert.Equal(t, 10*time.Millisecond, cfg.SweepInterval)
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

// TestAutoCheckpointReadBack proves WithAutoCheckpoint rides in the file-mode
// DSN and reaches the pinned connection: the pragma read-back equals the
// configured page count. The zero value keeps SQLite's default of 1000 pages
// (D-07, no sentinel semantics), and memory mode ignores the option.
func TestAutoCheckpointReadBack(t *testing.T) {
	t.Parallel()

	t.Run("configured_pages_read_back", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "autockpt.db")), WithAutoCheckpoint(200))
		require.NoError(t, c.initErr)

		assert.Equal(t, 200, readAutoCheckpoint(t, c))
	})

	t.Run("default_is_1000", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "default.db")))
		require.NoError(t, c.initErr)

		assert.Equal(t, 1000, readAutoCheckpoint(t, c))
	})

	t.Run("memory_mode_ignores_option", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory(), WithAutoCheckpoint(200))
		require.NoError(t, c.initErr)

		assert.Equal(t, 1000, readAutoCheckpoint(t, c))
	})
}

// readAutoCheckpoint reads back the effective wal_autocheckpoint page count
// from the provider's pinned connection.
func readAutoCheckpoint(t *testing.T, c *sqliteCache[string, string]) int {
	t.Helper()

	var pages int
	require.NoError(t, c.db.QueryRowContext(t.Context(), "PRAGMA wal_autocheckpoint").Scan(&pages))

	return pages
}

// TestJournalSizeLimitReadBack proves the D-08 64 MB bound rides the
// file-mode DSN and reaches the pinned connection: file mode reads back
// exactly journalSizeLimitBytes; memory mode keeps SQLite's unlimited default
// (-1); and with WithAutoCheckpoint(200) both pragmas coexist on the same
// connection (wal_autocheckpoint == 200, journal_size_limit == 67108864).
func TestJournalSizeLimitReadBack(t *testing.T) {
	t.Parallel()

	t.Run("file_mode_reads_back_64mb", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "journal-limit.db")))
		require.NoError(t, c.initErr)

		assert.Equal(t, int64(67108864), readJournalSizeLimit(t, c))
	})

	t.Run("memory_mode_keeps_unlimited_default", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		assert.Equal(t, int64(-1), readJournalSizeLimit(t, c))
	})

	t.Run("coexists_with_autocheckpoint", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "coexist.db")), WithAutoCheckpoint(200))
		require.NoError(t, c.initErr)

		assert.Equal(t, 200, readAutoCheckpoint(t, c))
		assert.Equal(t, int64(67108864), readJournalSizeLimit(t, c))
	})
}

// readJournalSizeLimit reads back the effective journal_size_limit from the
// provider's pinned connection.
func readJournalSizeLimit(t *testing.T, c *sqliteCache[string, string]) int64 {
	t.Helper()

	var limit int64
	require.NoError(t, c.db.QueryRowContext(t.Context(), "PRAGMA journal_size_limit").Scan(&limit))

	return limit
}

// TestBeginContentionBounded proves CONC-02/D-07: with _txlock=immediate a
// second connection holding the write lock makes a provider write surface a
// busy-classified error at BEGIN, bounded by busy_timeout — the provider
// performs no retry beyond the open window (elapsed below two budgets). After
// the raw transaction rolls back, the identical write succeeds (recovery).
func TestBeginContentionBounded(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "contention.db")

	c := newInternalProvider(t, WithPath(path))
	require.NoError(t, c.initErr)

	// A second raw handle with the production DSN; _txlock=immediate makes its
	// BeginTx issue BEGIN IMMEDIATE, seizing the write lock immediately.
	blocker, err := sql.Open("sqlite", buildDSN(path, false, 0, journalSizeLimitBytes))
	require.NoError(t, err)

	t.Cleanup(func() { _ = blocker.Close() })

	tx, err := blocker.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	start := time.Now()
	err = c.SetFunc(t.Context(), "blocked", "v")
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorContains(t, err, "cache/sqlite:")
	assert.True(t, isBusyError(err), "contention must surface as a busy-classified error, got: %v", err)
	assert.Less(t, elapsed, 2*busyTimeout, "provider-side retry of BEGIN would double the wait (D-07)")

	// Recovery: releasing the write lock lets the identical write succeed.
	require.NoError(t, tx.Rollback())
	require.NoError(t, c.SetFunc(t.Context(), "blocked", "v"))
}

// TestCheckpointColumns proves the PRAGMA wal_checkpoint(TRUNCATE) three-column
// scan (Pitfall 16): a healthy checkpoint succeeds — a one-value scan would
// fail with an argument-count error — and driver-level failures come back
// wrapped with the cache/sqlite: prefix.
func TestCheckpointColumns(t *testing.T) {
	t.Parallel()

	t.Run("healthy_file_checkpoint_scans_three_columns", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "columns.db")))
		require.NoError(t, c.initErr)

		require.NoError(t, c.checkpoint(t.Context()))
	})

	t.Run("driver_error_is_wrapped_with_prefix", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithPath(filepath.Join(t.TempDir(), "broken.db")))
		require.NoError(t, c.initErr)
		require.NoError(t, c.db.Close())

		err := c.checkpoint(t.Context())
		require.Error(t, err)
		require.ErrorContains(t, err, "cache/sqlite:")

		err = c.vacuum(t.Context())
		require.Error(t, err)
		require.ErrorContains(t, err, "cache/sqlite:")
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

// keyRowCount returns the raw row count for key, bypassing the provider's
// expiry filter. It takes no *testing.T so it is safe to call from
// require.Eventually condition goroutines.
func keyRowCount(db *sql.DB, key string) (int, error) {
	var count int

	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM cache_entries WHERE cache_key = ?", key).Scan(&count)

	return count, err
}

// countKeyRows is the test-facing wrapper over keyRowCount.
func countKeyRows(t *testing.T, db *sql.DB, key string) int {
	t.Helper()

	count, err := keyRowCount(db, key)
	require.NoError(t, err)

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

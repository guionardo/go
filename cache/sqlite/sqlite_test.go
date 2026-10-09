package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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

func TestBatchMGet(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "a", "1"))
	require.NoError(t, c.Set(t.Context(), "b", "2"))
	require.NoError(t, c.Set(t.Context(), "c", "3"))

	// A missing key is absent from the result; the found subset returns.
	got := c.MGet(t.Context(), "a", "missing", "c")
	assert.Equal(t, map[string]string{"a": "1", "c": "3"}, got)

	// No arguments: the empty no-op returns an empty (non-nil) map.
	empty := c.MGet(t.Context())
	require.NotNil(t, empty)
	assert.Empty(t, empty)

	// A short-TTL key disappears from MGet once expired (filter-only reads).
	require.NoError(t, c.Set(t.Context(), "ttl", "soon", 40*time.Millisecond))

	require.Eventually(t, func() bool {
		return len(c.MGet(t.Context(), "ttl")) == 0
	}, time.Second, 5*time.Millisecond)
}

// TestBatchMSetOneTTL pins the black-box D-04 contract: one TTL binds the
// whole batch, so every key of an MSet expires together (require.Eventually
// over MGet emptiness, mirroring the mem provider's resolveTTL-once).
func TestBatchMSetOneTTL(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	items := map[string]string{
		"k1": "1",
		"k2": "2",
		"k3": "3",
	}

	require.NoError(t, c.MSet(t.Context(), items, 40*time.Millisecond))

	require.Len(t, c.MGet(t.Context(), "k1", "k2", "k3"), 3)

	require.Eventually(t, func() bool {
		return len(c.MGet(t.Context(), "k1", "k2", "k3")) == 0
	}, time.Second, 5*time.Millisecond)
}

// TestBatchMDel pins the black-box BATCH-03 contract: chunked deletes at 150
// keys (two 100-key chunks) remove every row, a re-run is an idempotent nil,
// and MGet reflects the deletions.
func TestBatchMDel(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	keys := make([]string, 0, 150)
	for i := range 150 {
		keys = append(keys, fmt.Sprintf("k%03d", i))
	}

	require.NoError(t, c.MSet(t.Context(), map[string]string{"k000": "0", "k149": "149"}))
	require.Len(t, c.MGet(t.Context(), keys...), 2)

	require.NoError(t, c.MDel(t.Context(), keys...))
	assert.Empty(t, c.MGet(t.Context(), keys...))
	assert.Empty(t, c.MGet(t.Context(), "k000", "k001"))

	// Idempotent: deleting missing keys is a nil no-op.
	require.NoError(t, c.MDel(t.Context(), keys...))
}

// TestBatchSemantics owns the black-box batch contract (replaces the Phase 15
// placeholder smoke test): MSet -> MGet roundtrip with a missing key absent,
// MDel idempotency, MGet after MDel, empty-input no-ops, and one TTL expiring
// every key of the batch — semantics aligned with the mem provider it mirrors.
func TestBatchSemantics(t *testing.T) {
	t.Parallel()

	t.Run("mset_mget_roundtrip_with_missing_key_absent", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		require.NoError(t, c.MSet(t.Context(), map[string]string{"a": "1", "b": "2"}))

		got := c.MGet(t.Context(), "a", "b", "missing")
		assert.Equal(t, map[string]string{"a": "1", "b": "2"}, got)
	})

	t.Run("mdel_idempotent_and_mget_reflects_deletes", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		require.NoError(t, c.MSet(t.Context(), map[string]string{"a": "1", "b": "2"}))

		require.NoError(t, c.MDel(t.Context(), "a"))
		assert.Equal(t, map[string]string{"b": "2"}, c.MGet(t.Context(), "a", "b"))

		// Deleting already-deleted keys is a nil no-op (idempotency).
		require.NoError(t, c.MDel(t.Context(), "a"))
	})

	t.Run("empty_inputs_are_no_ops", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		// MSet with an empty map stores nothing and returns nil; MDel with no
		// keys is nil; MGet with no keys returns an empty (non-nil) map.
		require.NoError(t, c.MSet(t.Context(), map[string]string{}))
		require.NoError(t, c.MDel(t.Context()))

		got := c.MGet(t.Context())
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("one_ttl_expires_every_key_of_the_batch", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		t.Cleanup(func() { _ = c.Close() })

		keys := map[string]string{"k1": "1", "k2": "2", "k3": "3"}
		require.NoError(t, c.MSet(t.Context(), keys, 40*time.Millisecond))
		require.Len(t, c.MGet(t.Context(), "k1", "k2", "k3"), 3)

		require.Eventually(t, func() bool {
			return len(c.MGet(t.Context(), "k1", "k2", "k3")) == 0
		}, time.Second, 5*time.Millisecond)
	})
}

// TestConcurrentBatchOpsRace is the CONC-03 evidence: 8 goroutines mixing
// point and batch operations on one provider — file mode and :memory: —
// must complete without a data race on the pinned single-connection pool.
func TestConcurrentBatchOpsRace(t *testing.T) {
	t.Parallel()

	t.Run("file_mode", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](
			sqlite.WithPath(filepath.Join(t.TempDir(), "race.db")), sqlite.WithDefaultTTL(time.Hour))

		t.Cleanup(func() { _ = c.Close() })

		runBatchWorkers(t, c)
	})

	t.Run("memory_mode", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory(), sqlite.WithDefaultTTL(time.Hour))

		t.Cleanup(func() { _ = c.Close() })

		runBatchWorkers(t, c)
	})
}

// runBatchWorkers drives 8 goroutines through a bounded mixed-op loop: Set,
// Get, Delete, MGet (3 keys), MSet (5 keys), MDel, and GetOrSet, on
// per-worker prefixed keys plus one shared key. Each worker records its first
// error; after the start-channel release every worker error must be nil.
//
//nolint:funlen,gocognit,cyclop // CONC-03 harness: 8-op mixed loop with per-op guards
func runBatchWorkers(t *testing.T, c cache.BatchCache[string, string]) {
	t.Helper()

	const workers = 8

	const iterations = 50

	start := make(chan struct{})
	errs := make([]error, workers)

	var wg sync.WaitGroup

	for i := range workers {
		wg.Go(func() {
			<-start

			ctx := t.Context()
			prefix := "w" + strconv.Itoa(i)

			step := func(s int) bool {
				if s%3 == 0 {
					if err := c.Delete(ctx, fmt.Sprintf("%s:%02d", prefix, s)); err != nil {
						errs[i] = err

						return false
					}
				}

				c.MGet(ctx, fmt.Sprintf("%s:%02d", prefix, s), fmt.Sprintf("%s:%02d", prefix, s), "missing")

				if s%5 == 0 {
					if err := c.MDel(ctx, "missing"); err != nil {
						errs[i] = err

						return false
					}
				}

				if _, err := c.GetOrSet(ctx, prefix+":set", func(context.Context) (string, error) {
					return "computed", nil
				}); err != nil {
					errs[i] = err

					return false
				}

				return true
			}

			for s := range iterations {
				k := fmt.Sprintf("%s:%02d", prefix, s)

				if err := c.Set(ctx, k, "v"); err != nil {
					errs[i] = err

					return
				}

				if _, err := c.Get(ctx, k); err != nil {
					errs[i] = err

					return
				}

				batchKeys := map[string]string{
					prefix + ":batch:" + strconv.Itoa(s):            "b",
					prefix + ":batch:" + strconv.Itoa(iterations+s): "b2",
				}
				_ = c.MSet(ctx, batchKeys)

				if !step(s) {
					return
				}
			}
		})
	}

	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "worker %d", i)
	}
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

func TestSidecarRemoval(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "sidecar.db")

	c := sqlite.New[string, string](sqlite.WithPath(path))

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "k", "v"))
	require.NoError(t, c.Close())

	// The engine's last-connection close checkpoints the WAL and removes the
	// sidecars; the database file itself must remain. Windows may lag, so poll.
	require.Eventually(t, func() bool {
		_, walErr := os.Stat(path + "-wal")
		_, shmErr := os.Stat(path + "-shm")
		_, dbErr := os.Stat(path)

		return errors.Is(walErr, fs.ErrNotExist) && errors.Is(shmErr, fs.ErrNotExist) && dbErr == nil
	}, 2*time.Second, 20*time.Millisecond)
}

func TestReopenPersistence(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "reopen.db")

	c1 := sqlite.New[string, string](sqlite.WithPath(path))

	t.Cleanup(func() { _ = c1.Close() })

	require.NoError(t, c1.Set(t.Context(), "keep", "no-ttl"))
	require.NoError(t, c1.Set(t.Context(), "ttl", "hour", time.Hour))
	require.NoError(t, c1.Close())

	c2 := sqlite.New[string, string](sqlite.WithPath(path))

	t.Cleanup(func() { _ = c2.Close() })

	got, err := c2.Get(t.Context(), "keep")
	require.NoError(t, err)
	assert.Equal(t, "no-ttl", got)

	got, err = c2.Get(t.Context(), "ttl")
	require.NoError(t, err)
	assert.Equal(t, "hour", got)

	// A further write/read roundtrip after reopen is only possible when the
	// WAL-derived state is consistent.
	require.NoError(t, c2.Set(t.Context(), "more", "still-writable"))

	got, err = c2.Get(t.Context(), "more")
	require.NoError(t, err)
	assert.Equal(t, "still-writable", got)
}

func TestExpiryAcrossRestart(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "restart.db")

	c1 := sqlite.New[string, string](sqlite.WithPath(path))

	t.Cleanup(func() { _ = c1.Close() })

	require.NoError(t, c1.Set(t.Context(), "gone", "v", 40*time.Millisecond))
	require.NoError(t, c1.Close())

	c2 := sqlite.New[string, string](sqlite.WithPath(path))

	t.Cleanup(func() { _ = c2.Close() })

	// The open sweep may reclaim the row or the read filter may hide it —
	// either is correct per TTL-02/TTL-03; Eventually covers the TTL elapsing
	// after an immediate reopen.
	require.Eventually(t, func() bool {
		_, err := c2.Get(t.Context(), "gone")

		return errors.Is(err, cache.ErrMiss)
	}, time.Second, 5*time.Millisecond)
}

func TestCloseAfterDeferredError(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithPath("bad?path"))

	// Close is cleanup: it returns nil even when the handle carries a deferred
	// open error (the failing operation already surfaces that error).
	require.NoError(t, c.Close())
}

func TestDoubleClose(t *testing.T) {
	t.Parallel()

	t.Run("sequential_is_idempotent", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		require.NoError(t, c.Close())
		require.NoError(t, c.Close()) // idempotent
	})

	t.Run("concurrent_close_is_serialized_by_the_cas_guard", func(t *testing.T) {
		t.Parallel()

		c := sqlite.New[string, string](sqlite.WithMemory())

		const closers = 8

		errs := make([]error, closers)

		var wg sync.WaitGroup

		for i := range closers {
			wg.Go(func() {
				errs[i] = c.Close()
			})
		}

		wg.Wait()

		for i, err := range errs {
			require.NoErrorf(t, err, "closer %d", i)
		}
	})
}

func TestClosedErrorPrefix(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())
	require.NoError(t, c.Close())

	_, err := c.Get(t.Context(), "k")
	require.ErrorIs(t, err, sqlite.ErrClosed)
	require.ErrorContains(t, err, "cache/sqlite:")
}

func TestOptimizable(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithPath(filepath.Join(t.TempDir(), "optimize.db")))

	t.Cleanup(func() { _ = c.Close() })

	opt, ok := c.(sqlite.Optimizable)
	require.True(t, ok, "New must return a value implementing sqlite.Optimizable")

	require.NoError(t, c.Set(t.Context(), "k", "v"))

	require.NoError(t, opt.Checkpoint(t.Context()))
	require.NoError(t, opt.Vacuum(t.Context()))

	got, err := c.Get(t.Context(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}

func TestOptimizableMemory(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())

	t.Cleanup(func() { _ = c.Close() })

	opt, ok := c.(sqlite.Optimizable)
	require.True(t, ok)

	// Memory mode has no WAL: the checkpoint is a valid busy=0 no-op.
	require.NoError(t, opt.Checkpoint(t.Context()))
}

func TestCheckpointBlocked(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "blocked.db")

	c := sqlite.New[string, string](sqlite.WithPath(path))

	t.Cleanup(func() { _ = c.Close() })

	require.NoError(t, c.Set(t.Context(), "k", "v"))

	// A second connection holding an open read transaction blocks the
	// TRUNCATE checkpoint: after the busy timeout the PRAGMA reports busy != 0,
	// which Checkpoint must surface as an error — never a silent nil.
	db2, err := sql.Open("sqlite", path)
	require.NoError(t, err)

	t.Cleanup(func() { _ = db2.Close() })

	tx, err := db2.BeginTx(t.Context(), &sql.TxOptions{ReadOnly: true})
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	rows, err := tx.QueryContext(t.Context(), "SELECT COUNT(*) FROM cache_entries")
	require.NoError(t, err)

	t.Cleanup(func() { _ = rows.Close() })

	var count int

	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&count))

	opt, ok := c.(sqlite.Optimizable)
	require.True(t, ok)

	err = opt.Checkpoint(t.Context())
	require.Error(t, err, "blocked checkpoint must never be a silent nil")
	require.ErrorContains(t, err, "busy")
}

func TestOptimizableClosed(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithMemory())
	require.NoError(t, c.Close())

	opt, ok := c.(sqlite.Optimizable)
	require.True(t, ok)

	err := opt.Checkpoint(t.Context())
	require.ErrorIs(t, err, sqlite.ErrClosed)
	require.ErrorContains(t, err, "cache/sqlite:")

	err = opt.Vacuum(t.Context())
	require.ErrorIs(t, err, sqlite.ErrClosed)
	require.ErrorContains(t, err, "cache/sqlite:")
}

func TestOptimizableDeferredErr(t *testing.T) {
	t.Parallel()

	c := sqlite.New[string, string](sqlite.WithPath("bad?path"))

	t.Cleanup(func() { _ = c.Close() })

	opt, ok := c.(sqlite.Optimizable)
	require.True(t, ok)

	err := opt.Checkpoint(t.Context())
	require.ErrorIs(t, err, sqlite.ErrInvalidPath)
	require.ErrorContains(t, err, "cache/sqlite:")

	err = opt.Vacuum(t.Context())
	require.ErrorIs(t, err, sqlite.ErrInvalidPath)
	require.ErrorContains(t, err, "cache/sqlite:")
}

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUniqueKeys pins D-02 dedup semantics: duplicates collapse preserving
// first-seen order, and distinct key values whose fmt.Sprint forms collide
// collapse to the first occurrence (the SQL keyspace is the stringified form).
func TestUniqueKeys(t *testing.T) {
	t.Parallel()

	t.Run("duplicates_collapse_preserving_first_seen_order", func(t *testing.T) {
		t.Parallel()

		got := uniqueKeys([]string{"b", "a", "b", "c", "a"})
		assert.Equal(t, []string{"b", "a", "c"}, got)
	})

	t.Run("form_collision_collapses_to_first_occurrence", func(t *testing.T) {
		t.Parallel()

		// any(1) and any("1") share the fmt.Sprint form "1".
		got := uniqueKeys([]any{strconv.Itoa(1), any(1), any("1")})
		require.Len(t, got, 1, "equal stringified forms must collapse (Finding 5)")
		assert.Equal(t, "1", got[0])
	})

	t.Run("empty_input_returns_empty_slice", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, uniqueKeys([]string{}))
	})
}

// TestChunksOf pins the D-01 chunk boundaries: size-bounded slices preserving
// input order; an empty input yields no chunks (so no query is issued).
func TestChunksOf(t *testing.T) {
	t.Parallel()

	keys := make([]string, 250)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%03d", i)
	}

	cases := []struct {
		name string
		n    int
		size int
		want []int
	}{
		{name: "empty_input_no_chunks", n: 0, size: 100, want: nil},
		{name: "single_key_one_chunk", n: 1, size: 100, want: []int{1}},
		{name: "exactly_100_one_chunk", n: 100, size: 100, want: []int{100}},
		{name: "101_splits_100_plus_1", n: 101, size: 100, want: []int{100, 1}},
		{name: "250_splits_100_100_50", n: 250, size: 100, want: []int{100, 100, 50}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			chunks := chunksOf(keys[:tc.n], tc.size)
			require.Len(t, chunks, len(tc.want))

			for i, wantLen := range tc.want {
				require.Len(t, chunks[i], wantLen)
			}

			// Order is preserved across chunks.
			flat := 0
			for _, chunk := range chunks {
				assert.Equal(t, keys[flat:flat+len(chunk)], chunk)
				flat += len(chunk)
			}
		})
	}
}

// TestInPlaceholders pins the only generated SQL text: a list of ? markers.
func TestInPlaceholders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "?", inPlaceholders(1))
	assert.Equal(t, "?,?,?", inPlaceholders(3))
	assert.Equal(t, strings.Repeat("?,", 99)+"?", inPlaceholders(100))
}

// TestMGetFunc covers the BATCH-01 behavior matrix on the internal receiver:
// empty no-op, fresh decode, duplicate collapse, expired-absent-by-SQL (with
// the row retained — reads never delete), corrupt-row skip, chunk-failure
// skip, closed-provider noise-free empty map, and cross-chunk retrieval.
//
//nolint:funlen // behavior matrix, mirroring the mem_test.go precedent
func TestMGetFunc(t *testing.T) {
	t.Parallel()

	t.Run("empty_input_no_query_empty_map", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		got := c.MGetFunc(t.Context())
		require.NotNil(t, got, "D-02: an empty map, never nil")
		assert.Empty(t, got)
	})

	t.Run("seeded_fresh_keys_return_values", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.SetFunc(t.Context(), "a", "va"))
		require.NoError(t, c.SetFunc(t.Context(), "b", "vb"))

		got := c.MGetFunc(t.Context(), "a", "b", "missing")
		assert.Equal(t, map[string]string{"a": "va", "b": "vb"}, got)
	})

	t.Run("duplicate_input_keys_produce_one_entry", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.SetFunc(t.Context(), "dup", "vd"))

		got := c.MGetFunc(t.Context(), "dup", "dup", "dup")
		assert.Equal(t, map[string]string{"dup": "vd"}, got)
	})

	t.Run("expired_row_absent_but_retained", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		_, err := c.db.ExecContext(
			t.Context(), UpsertSQL, "stale", `"v"`, time.Now().Add(-time.Hour).UnixNano())
		require.NoError(t, err)

		require.NoError(t, c.SetFunc(t.Context(), "fresh", "vf"))

		got := c.MGetFunc(t.Context(), "stale", "fresh")
		assert.Equal(t, map[string]string{"fresh": "vf"}, got)
		assert.Equal(t, 1, countKeyRows(t, c.db, "stale"), "MGet must never delete rows (TTL-02)")
	})

	t.Run("corrupt_row_skipped_siblings_return", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		_, err := c.db.ExecContext(t.Context(), UpsertSQL, "corrupt", "{not-json", nil)
		require.NoError(t, err)

		require.NoError(t, c.SetFunc(t.Context(), "ok", "v"))

		got := c.MGetFunc(t.Context(), "corrupt", "ok")
		assert.Equal(t, map[string]string{"ok": "v"}, got)
	})

	t.Run("chunk_failure_skipped_warn_only", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// The frozen signature has no error channel: a chunk error is logged
		// and skipped, yielding an empty map — never a panic (D-03).
		got := c.MGetFunc(ctx, "a")
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("closed_provider_returns_empty_map", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)
		require.NoError(t, c.CloseFunc())

		got := c.MGetFunc(t.Context(), "a")
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("cross_chunk_150_keys_all_returned", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		want := make(map[string]string, 150)
		keys := make([]string, 0, 150)

		for i := range 150 {
			key := fmt.Sprintf("chunk-%03d", i)
			want[key] = fmt.Sprintf("v-%03d", i)
			keys = append(keys, key)
		}

		// Seed point-wise: this test owns MGet only; MSet seeding arrives with
		// its own implementation (Task 2).
		for key, value := range want {
			require.NoError(t, c.SetFunc(t.Context(), key, value))
		}

		got := c.MGetFunc(t.Context(), keys...)
		assert.Equal(t, want, got)
	})
}

// countTableRows returns the total raw row count of cache_entries, bypassing
// the provider's expiry filter. It takes no *testing.T so it is safe to call
// from require.Eventually condition goroutines.
func countTableRows(db *sql.DB) (int, error) {
	var count int

	err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM cache_entries").Scan(&count)

	return count, err
}

// TestMSetFunc covers the BATCH-02 behavior matrix on the internal receiver:
// empty no-op, pre-marshal atomicity, happy path with one TTL for the batch,
// upsert overwrite, and closed-provider error.
//
//nolint:funlen // behavior matrix, mirroring the mem_test.go precedent
func TestMSetFunc(t *testing.T) {
	t.Parallel()

	t.Run("empty_map_nil_no_transaction", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.MSetFunc(t.Context(), map[string]string{}))

		count, err := countTableRows(c.db)
		require.NoError(t, err)
		assert.Zero(t, count, "D-02: an empty batch writes nothing")
	})

	t.Run("marshal_failure_writes_nothing", func(t *testing.T) {
		t.Parallel()

		// A [string, any] provider so a chan value reaches the marshal step.
		cfg := defaultConfig()
		WithMemory()(cfg)

		anyC := &sqliteCache[string, any]{}
		anyC.initErr = anyC.open(t.Context(), cfg)

		t.Cleanup(func() { _ = anyC.CloseFunc() })

		require.NoError(t, anyC.initErr)

		// Pre-marshal ordering (D-04): the valid pair must not land when a
		// sibling value fails to marshal — nothing is written before BeginTx.
		err := anyC.MSetFunc(t.Context(), map[string]any{"ok": "v", "bad": make(chan int)})
		require.Error(t, err)
		require.ErrorContains(t, err, "cache/sqlite:")

		count, err := countTableRows(anyC.db)
		require.NoError(t, err)
		assert.Zero(t, count, "a marshal error must leave zero rows from the batch")
	})

	t.Run("happy_path_one_ttl_for_batch", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		items := map[string]string{"a": "1", "b": "2", "c": "3"}
		require.NoError(t, c.MSetFunc(t.Context(), items, time.Hour))

		got := c.MGetFunc(t.Context(), "a", "b", "c")
		assert.Equal(t, items, got)

		// One TTL resolved once for the whole batch: all rows share the exact
		// same absolute expiry (mem parity, D-04).
		expA := readRawExpiry(t, c.db, "a")
		expB := readRawExpiry(t, c.db, "b")
		expC := readRawExpiry(t, c.db, "c")

		require.NotNil(t, expA)
		assert.Equal(t, expA, expB)
		assert.Equal(t, expA, expC)
	})

	t.Run("overwrite_replaces_value_and_expiry", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.SetFunc(t.Context(), "k", "old", time.Hour))
		require.NoError(t, c.MSetFunc(t.Context(), map[string]string{"k": "new"}))

		got, err := c.GetFunc(t.Context(), "k")
		require.NoError(t, err)
		assert.Equal(t, "new", got)

		// Upsert semantics: the no-TTL MSet replaced the hour-long expiry with
		// SQL NULL.
		assert.Nil(t, readRawExpiry(t, c.db, "k"))
	})

	t.Run("closed_provider_wrapped_errclosed", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)
		require.NoError(t, c.CloseFunc())

		err := c.MSetFunc(t.Context(), map[string]string{"k": "v"})
		require.ErrorIs(t, err, ErrClosed)
		require.ErrorContains(t, err, "cache/sqlite:")
	})
}

// readRawExpiry returns the raw expires_at value for key (nil for SQL NULL).
// It fails the test on a query error.
func readRawExpiry(t *testing.T, db *sql.DB, key string) any {
	t.Helper()

	var exp sql.NullInt64

	err := db.QueryRowContext(context.Background(),
		"SELECT expires_at FROM cache_entries WHERE cache_key = ?", key).Scan(&exp)
	require.NoError(t, err)

	if !exp.Valid {
		return nil
	}

	return exp.Int64
}

// TestMSetNoPartialWrites proves the D-04 atomicity invariant: an MSet whose
// context cancels mid-batch leaves the table at 0 or all rows — never a
// partial write (Pitfall 4). The invariant is checked across repeated runs so
// the rollback branch executes in practice.
func TestMSetNoPartialWrites(t *testing.T) {
	t.Parallel()

	const batchSize = 2000

	items := make(map[string]string, batchSize)
	for i := range batchSize {
		items[fmt.Sprintf("bulk-%04d", i)] = fmt.Sprintf("v-%04d", i)
	}

	for iteration := range 5 {
		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		// Cancel roughly 1 ms after start: early enough that the exec loop
		// usually cannot complete, late enough that the transaction opened.
		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)

		err := c.MSetFunc(ctx, items)

		cancel()

		count, countErr := countTableRows(c.db)
		require.NoError(t, countErr)

		require.Truef(t, count == 0 || count == batchSize,
			"iteration %d: stored count must be 0 or %d, never partial (got %d, err %v)",
			iteration, batchSize, count, err)

		if err == nil {
			require.Equal(t, batchSize, count, "a successful MSet stores the whole batch")
		}
	}
}

// TestMDelFunc covers the BATCH-03 behavior matrix on the internal receiver:
// missing no-op, duplicate collapse, chunked deletes at 250 keys, idempotent
// re-run, and fail-fast wrapped errors.
//
//nolint:funlen // behavior matrix, mirroring the mem_test.go precedent
func TestMDelFunc(t *testing.T) {
	t.Parallel()

	t.Run("missing_keys_nil", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.MDelFunc(t.Context(), "no-such-a", "no-such-b"))
	})

	t.Run("empty_input_nil_no_query", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.MDelFunc(t.Context()))
	})

	t.Run("duplicate_input_keys_delete_once", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		require.NoError(t, c.SetFunc(t.Context(), "dup", "v"))

		require.NoError(t, c.MDelFunc(t.Context(), "dup", "dup", "dup"))
		assert.Equal(t, 0, countKeyRows(t, c.db, "dup"))
	})

	t.Run("chunked_250_keys_all_deleted", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)

		keys := make([]string, 0, 250)

		for i := range 250 {
			key := fmt.Sprintf("del-%03d", i)
			keys = append(keys, key)
			require.NoError(t, c.SetFunc(t.Context(), key, "v"))
		}

		require.NoError(t, c.MDelFunc(t.Context(), keys...))
		assert.Empty(t, c.MGetFunc(t.Context(), keys...))

		// Idempotent: the second identical call is a nil no-op.
		require.NoError(t, c.MDelFunc(t.Context(), keys...))
	})

	t.Run("closed_db_handle_fail_fast_wrapped", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)
		require.NoError(t, c.db.Close())

		err := c.MDelFunc(t.Context(), "a")
		require.Error(t, err)
		require.ErrorContains(t, err, "cache/sqlite:")
	})

	t.Run("closed_provider_wrapped_errclosed", func(t *testing.T) {
		t.Parallel()

		c := newInternalProvider(t, WithMemory())
		require.NoError(t, c.initErr)
		require.NoError(t, c.CloseFunc())

		err := c.MDelFunc(t.Context(), "a")
		require.ErrorIs(t, err, ErrClosed)
		require.ErrorContains(t, err, "cache/sqlite:")
	})
}

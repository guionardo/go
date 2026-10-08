package sqlite

import (
	"context"
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

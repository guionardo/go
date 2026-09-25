package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoPostgres(t *testing.T) string {
	t.Helper()

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://localhost:5432/cache_test?sslmode=disable"
	}

	c, err := postgres.New[string, string](postgres.WithConnString(connString))
	if err != nil {
		t.Skip("postgres not available:", err)
	}
	_ = c.Close()

	return connString
}

func newTestCache(t *testing.T, connString string) cache.BatchCache[string, string] {
	t.Helper()

	c, err := postgres.New[string, string](
		postgres.WithConnString(connString),
	)
	require.NoError(t, err)

	t.Cleanup(func() { _ = c.Close() })

	return c
}

func TestPostgresCache_SetGet(t *testing.T) {
	connString := skipIfNoPostgres(t)
	c := newTestCache(t, connString)

	t.Run("set_and_get_returns_value", func(t *testing.T) {
		err := c.Set(t.Context(), "postgres_test_set_get", "v")
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "postgres_test_set_get")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		_, err := c.Get(t.Context(), "nonexistent_key")
		require.Error(t, err)
		assert.ErrorContains(t, err, "key not found")
	})

	t.Run("get_expired_returns_error", func(t *testing.T) {
		err := c.Set(t.Context(), "postgres_test_expired", "v", 1*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		_, err = c.Get(t.Context(), "postgres_test_expired")
		require.Error(t, err)
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		_ = c.Set(t.Context(), "postgres_test_delete", "v")

		err := c.Delete(t.Context(), "postgres_test_delete")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "postgres_test_delete")
		require.Error(t, err)
	})

	t.Run("get_or_set_computes", func(t *testing.T) {
		got, err := c.GetOrSet(t.Context(), "postgres_test_gos", func(context.Context) (string, error) {
			return "computed", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "computed", got)
	})
}

func TestPostgresCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("close_does_not_error", func(t *testing.T) {
		t.Parallel()

		connString := os.Getenv("DATABASE_URL")
		if connString == "" {
			connString = "postgres://localhost:5432/cache_test?sslmode=disable"
		}
		c, err := postgres.New[string, string](postgres.WithConnString(connString))
		if err != nil {
			t.Skip("postgres not available:", err)
		}

		err = c.Close()
		require.NoError(t, err)
	})

	t.Run("close_is_idempotent", func(t *testing.T) {
		t.Parallel()

		connString := os.Getenv("DATABASE_URL")
		if connString == "" {
			connString = "postgres://localhost:5432/cache_test?sslmode=disable"
		}
		c, err := postgres.New[string, string](postgres.WithConnString(connString))
		if err != nil {
			t.Skip("postgres not available:", err)
		}

		_ = c.Close()
		err = c.Close()
		require.NoError(t, err)
	})
}

func TestPostgresCache_Batch(t *testing.T) {
	connString := skipIfNoPostgres(t)
	c := newTestCache(t, connString)

	t.Run("mget_returns_found", func(t *testing.T) {
		// Ensure keys exist
		require.NoError(t, c.Set(t.Context(), "pg_mget_a", "v1"))
		require.NoError(t, c.Set(t.Context(), "pg_mget_b", "v2"))

		result := c.MGet(t.Context(), "pg_mget_a", "pg_mget_b", "missing")
		assert.Equal(t, "v1", result["pg_mget_a"])
		assert.Equal(t, "v2", result["pg_mget_b"])
		assert.NotContains(t, result, "missing")
	})

	t.Run("mget_empty_keys", func(t *testing.T) {
		result := c.MGet(t.Context())
		assert.Empty(t, result)
	})

	t.Run("mset_stores_all", func(t *testing.T) {
		err := c.MSet(t.Context(), map[string]string{"pg_mset_a": "va", "pg_mset_b": "vb"})
		require.NoError(t, err)

		got, err := c.Get(t.Context(), "pg_mset_a")
		require.NoError(t, err)
		assert.Equal(t, "va", got)

		got, err = c.Get(t.Context(), "pg_mset_b")
		require.NoError(t, err)
		assert.Equal(t, "vb", got)
	})

	t.Run("mset_empty_map", func(t *testing.T) {
		err := c.MSet(t.Context(), map[string]string{})
		require.NoError(t, err)
	})

	t.Run("mdel_deletes", func(t *testing.T) {
		_ = c.Set(t.Context(), "pg_mdel_a", "val_a")
		_ = c.Set(t.Context(), "pg_mdel_b", "val_b")

		err := c.MDel(t.Context(), "pg_mdel_a", "pg_mdel_b")
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "pg_mdel_a")
		require.ErrorIs(t, err, cache.ErrMiss)

		_, err = c.Get(t.Context(), "pg_mdel_b")
		require.ErrorIs(t, err, cache.ErrMiss)
	})

	t.Run("mdel_empty_keys", func(t *testing.T) {
		err := c.MDel(t.Context())
		require.NoError(t, err)
	})
}

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

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

func newTestCache(t *testing.T, connString string) *postgres.Cache[string, string] {
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
		err := c.Set(context.Background(), "postgres_test_set_get", "v")
		require.NoError(t, err)

		got, err := c.Get(context.Background(), "postgres_test_set_get")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	})

	t.Run("get_miss_returns_error", func(t *testing.T) {
		_, err := c.Get(context.Background(), "nonexistent_key")
		require.Error(t, err)
		assert.ErrorContains(t, err, "key not found")
	})

	t.Run("get_expired_returns_error", func(t *testing.T) {
		err := c.Set(context.Background(), "postgres_test_expired", "v", 1*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		_, err = c.Get(context.Background(), "postgres_test_expired")
		require.Error(t, err)
	})

	t.Run("delete_removes_key", func(t *testing.T) {
		_ = c.Set(context.Background(), "postgres_test_delete", "v")

		err := c.Delete(context.Background(), "postgres_test_delete")
		require.NoError(t, err)

		_, err = c.Get(context.Background(), "postgres_test_delete")
		require.Error(t, err)
	})

	t.Run("get_or_set_computes", func(t *testing.T) {
		got, err := c.GetOrSet(context.Background(), "postgres_test_gos", func() (string, error) {
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

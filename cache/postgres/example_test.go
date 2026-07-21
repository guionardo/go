package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/guionardo/go/cache/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoExamplePostgres(t *testing.T) string {
	t.Helper()

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://localhost:5432/cache_test?sslmode=disable"
	}

	c, err := postgres.New[string, string](postgres.WithConnString(connString))
	if err != nil {
		t.Skip("postgres not available")
	}
	_ = c.Close()

	return connString
}

func TestPostgresExample_SetGet(t *testing.T) {
	connString := skipIfNoExamplePostgres(t)

	c, err := postgres.New[string, string](postgres.WithConnString(connString))
	require.NoError(t, err)

	err = c.Set(context.Background(), "example", "pg-value")
	require.NoError(t, err)

	value, err := c.Get(context.Background(), "example")
	require.NoError(t, err)
	assert.Equal(t, "pg-value", value)

	err = c.Close()
	require.NoError(t, err)
}

package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTableSQL_UsesConfiguredTableName(t *testing.T) {
	t.Parallel()

	sql := createTableSQL("app_cache")

	assert.Contains(t, sql, `CREATE UNLOGGED TABLE IF NOT EXISTS "app_cache"`)
	assert.Contains(t, sql, `CREATE INDEX IF NOT EXISTS "idx_app_cache_expires_at"`)
	assert.Contains(t, sql, `ON "app_cache" (expires_at)`)
	assert.NotContains(t, sql, "cache_entries", "configured table must not fall back to the default name")
}

func TestCreateTableSQL_DefaultTableName(t *testing.T) {
	t.Parallel()

	// The default name is still cache_entries, and the published CreateTableSQL
	// constant stays in sync with what New actually executes for that default.
	assert.Equal(t, "cache_entries", defaultTableName)
	assert.Contains(t, createTableSQL(defaultTableName), `"cache_entries"`)
	assert.Equal(t, CreateTableSQL, strings.ReplaceAll(createTableSQL(defaultTableName), `"`, ""))
}

func TestCreateTableSQL_QuotesIdentifiers(t *testing.T) {
	t.Parallel()

	sql := createTableSQL(`weird"; DROP TABLE users; --`)

	assert.Contains(t, sql, `"weird""; DROP TABLE users; --"`)
	assert.NotContains(t, sql, `; DROP TABLE users; -- (`)
}

func TestPrewarmSQL_UsesConfiguredTableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `SELECT pg_prewarm('app_cache')`, prewarmSQL("app_cache"))
	assert.Equal(t, "SELECT pg_prewarm('o''brien')", prewarmSQL("o'brien"))
	assert.Equal(t, PrewarmSQL, prewarmSQL(defaultTableName))
}

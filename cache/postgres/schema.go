package postgres

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// defaultTableName is the table name used when WithTableName is not supplied.
const defaultTableName = "cache_entries"

// CreateTableSQL is the SQL statement for creating the default cache entries
// table. The table stores JSON-serialized values with optional TTL expiration.
//
// It is the exact output of createTableSQL(defaultTableName); providers built
// with WithTableName create the same DDL for the configured table.
const CreateTableSQL = `
CREATE UNLOGGED TABLE IF NOT EXISTS cache_entries (
    cache_key   TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;
`

// PrewarmSQL preloads the default cache table into PostgreSQL shared buffers.
// Uses pg_prewarm (contrib extension) — best-effort, silently skipped if
// the extension is not installed.
const PrewarmSQL = `SELECT pg_prewarm('cache_entries')`

// createTableSQL builds the DDL for the cache table named tableName.
//
// The table identifier is sanitized with pgx.Identifier, exactly like the
// queries the provider runs, so the name is quoted as an identifier and a
// caller-supplied value cannot break out of the quoting. The index name is
// derived from the table name and quoted the same way.
func createTableSQL(tableName string) string {
	table := pgx.Identifier{tableName}.Sanitize()
	index := pgx.Identifier{"idx_" + tableName + "_expires_at"}.Sanitize()

	return fmt.Sprintf(`
CREATE UNLOGGED TABLE IF NOT EXISTS %[1]s (
    cache_key   TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS %[2]s
    ON %[1]s (expires_at)
    WHERE expires_at IS NOT NULL;
`, table, index)
}

// prewarmSQL builds the pg_prewarm statement for the cache table named
// tableName. pg_prewarm takes a regclass value, so the name is passed as a
// quoted string literal — embedded single quotes are doubled, the literal's
// counterpart to identifier quoting.
func prewarmSQL(tableName string) string {
	return fmt.Sprintf("SELECT pg_prewarm('%s')", strings.ReplaceAll(tableName, "'", "''"))
}

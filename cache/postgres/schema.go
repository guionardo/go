package postgres

// CreateTableSQL is the SQL statement for creating the cache entries table.
// The table stores JSON-serialized values with optional TTL expiration.
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

// PrewarmSQL preloads the cache table into PostgreSQL shared buffers.
// Uses pg_prewarm (contrib extension) — best-effort, silently skipped if
// the extension is not installed.
const PrewarmSQL = `SELECT pg_prewarm('cache_entries')`

package postgres

// CreateTableSQL is the SQL statement for creating the cache entries table.
// The table stores JSON-serialized values with optional TTL expiration.
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key   TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;
`

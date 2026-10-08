package sqlite

// Schema constants for the fixed three-column cache_entries table. There is
// no user_version and no migration scaffolding: bootstrap is idempotent DDL
// only (CREATE ... IF NOT EXISTS under BEGIN IMMEDIATE).

// CreateTableSQL creates the cache entries table: JSON values keyed by their
// fmt.Sprint form, with an optional absolute UnixNano expiry.
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key  TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    expires_at INTEGER
) WITHOUT ROWID;`

// CreateIndexSQL creates the partial index backing expiry sweeps. The index
// only covers rows that actually carry an expiry.
const CreateIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;`

// SelectSQL fetches a non-expired value by key. Expiration is filtered in SQL;
// reads never delete rows.
const SelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`

// UpsertSQL stores a value, replacing any existing entry for the key.
const UpsertSQL = `INSERT INTO cache_entries (cache_key, value, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(cache_key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`

// DeleteSQL removes a key. Deleting a missing key is a no-op.
const DeleteSQL = `DELETE FROM cache_entries WHERE cache_key = ?`

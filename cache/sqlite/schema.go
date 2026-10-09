package sqlite

// Schema constants for the fixed three-column cache_entries table. There is
// no user_version and no migration scaffolding: bootstrap is idempotent DDL
// only (CREATE ... IF NOT EXISTS under BEGIN IMMEDIATE).

const (
	// CreateTableSQL creates the cache entries table: JSON values keyed by their
	// fmt.Sprint form, with an optional absolute UnixNano expiry.
	CreateTableSQL = `
CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key  TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    expires_at INTEGER
) WITHOUT ROWID;`

	// CreateIndexSQL creates the partial index backing expiry sweeps. The index
	// only covers rows that actually carry an expiry.
	CreateIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;`

	// SelectSQL fetches a non-expired value by key. Expiration is filtered in SQL;
	// reads never delete rows.
	SelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`

	// MGetSelectSQL fetches one chunk of non-expired values by key list. The
	// expiry predicate is folded in, so expired rows are absent by SQL — no
	// code-level skip is needed. cache_key is selected because fmt.Sprint is
	// not invertible and the WITHOUT ROWID table has no rowid shortcut
	// (Finding 5); rows are re-keyed to K through the chunk's own input map.
	// %s is ONLY the "?,?,..." marker list produced by inPlaceholders — never
	// caller data; keys and the bound now stay parameters.
	MGetSelectSQL = `SELECT cache_key, value FROM cache_entries
WHERE cache_key IN (%s) AND (expires_at IS NULL OR expires_at > ?)`

	// MDelSQL deletes one chunk of keys. The %s marker-list contract is the
	// same as MGetSelectSQL; deleting a missing key is a no-op (BATCH-03).
	MDelSQL = `DELETE FROM cache_entries WHERE cache_key IN (%s)`

	// UpsertSQL stores a value, replacing any existing entry for the key.
	UpsertSQL = `INSERT INTO cache_entries (cache_key, value, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(cache_key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`

	// DeleteSQL removes a key. Deleting a missing key is a no-op.
	DeleteSQL = `DELETE FROM cache_entries WHERE cache_key = ?`

	// SweepSQL deletes expired rows in one pass. The predicate matches the
	// partial index on expires_at (IS NOT NULL) and compares against a bound
	// UnixNano — never SQL datetime('now').
	SweepSQL = `DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at <= ?`
)

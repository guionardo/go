// Package sqlite provides an embedded, durable cache backend for the cache
// package — import path github.com/guionardo/go/cache/sqlite — backed by
// pure-Go SQLite (modernc.org/sqlite) with no CGO.
//
// It is positioned as "mem but survives restarts; redis/postgres without the
// infrastructure": a local cache whose entries persist across process runs in
// a WAL-mode SQLite database, with no server to run.
//
// Usage:
//
//	c := sqlite.New[string, string](sqlite.WithName("app"))
//	defer func() { _ = c.Close() }()
//
//	if err := c.Set(ctx, "user:42", "value"); err != nil {
//		// handle error
//	}
//	value, err := c.Get(ctx, "user:42") // wraps cache.ErrMiss when absent or expired
//
// Location precedence is memory > path > name, with the zero-value
// configuration opening an in-memory cache: WithMemory() (or the literal
// ":memory:" path) opens an in-memory cache; WithPath opens a file-backed
// cache at the given path — paths containing '?' or '#' are rejected with
// ErrInvalidPath; WithName resolves to os.UserCacheDir()/<name>/cache.db,
// creating the directory with 0700 permissions. In every mode the pool is
// pinned to a single connection: that is what makes :memory: correct (each
// pooled connection would otherwise open a private database).
//
// TTLs resolve per call first, then the provider default (WithDefaultTTL),
// then no expiry. Expiry is stored as an absolute UnixNano timestamp, so it
// survives restarts; expired entries are never returned (reads filter on
// expires_at) and are reclaimed by a best-effort sweep when the cache opens
// and by the optional WithSweepInterval ticker.
//
// Data and state: file-backed entries persist across process runs. Close
// closes the database, which checkpoints the WAL and removes the -wal/-shm
// sidecars on the last connection. File mode verifies journal_mode (WAL) at
// open; a mismatch only logs a warning. File mode supports same-host
// multi-process sharing of a cache file: WAL serializes writers and the 5 s
// busy_timeout bounds write contention, which surfaces as an error at BEGIN
// rather than silent corruption. A fresh database opened concurrently with
// another process briefly retries connection establishment (busy-only,
// bounded by the same timeout). A canceled MGet returns the partial results
// gathered so far — the frozen MGet signature has no error channel. WAL
// requires local storage: network filesystems and cross-host sharing are
// unsupported.
//
// Long-running processes can tune WAL checkpointing with WithAutoCheckpoint
// and reclaim space explicitly through the Optimizable interface
// (Checkpoint for a cheap WAL truncate, Vacuum for a heavy full rewrite).
//
// New never fails: open and validation errors are recorded and deferred, and
// are returned by the first operation that needs the database.
//
// Cache contents are unencrypted at rest; the cache directory is created with
// 0700 permissions and the database file inherits the process umask.
package sqlite

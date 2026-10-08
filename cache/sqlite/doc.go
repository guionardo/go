// Package sqlite provides an embedded, durable cache backend for the cache
// package, backed by pure-Go SQLite (modernc.org/sqlite) with no CGO.
//
// It is positioned as "mem but survives restarts; redis/postgres without the
// infrastructure": a local, single-process-oriented cache whose entries persist
// across process runs in a WAL-mode SQLite database.
//
// Usage:
//
//	c := sqlite.New[string, string](sqlite.WithName("app"))
//	defer func() { _ = c.Close() }()
//
// Location precedence: WithMemory() (or an empty path, or the zero-value
// configuration) opens an in-memory cache; WithPath opens a file-backed cache
// at the given path — paths containing '?' or '#' are rejected via
// ErrInvalidPath; WithName resolves to
// os.UserCacheDir()/<name>/cache.db, creating the directory with 0700
// permissions.
//
// New never fails: open and validation errors are recorded and deferred, and
// are returned by the first operation that needs the database.
//
// Cache contents are unencrypted at rest; the cache directory is created with
// 0700 permissions and the database file inherits the process umask.
package sqlite

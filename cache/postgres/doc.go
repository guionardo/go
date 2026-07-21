// Package postgres provides a PostgreSQL backend for cache.Cache.
//
// Uses pgx/v5 with pgxpool for connection pooling. Creates an UNLOGGED
// table (blazing-fast writes, no WAL) and optionally calls pg_prewarm
// on startup. A background goroutine sweeps expired entries.
//
// Usage:
//
//	c, err := postgres.New[string, string](
//	    postgres.WithConnString("postgres://user:pass@localhost/cache"),
//	    postgres.WithTableName("app_cache"),
//	    postgres.WithPoolSize(10),
//	    postgres.WithSweepInterval(1*time.Minute),
//	    postgres.WithDefaultTTL(5*time.Minute),
//	)
package postgres

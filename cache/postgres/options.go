package postgres

import "time"

// Config holds configuration for the Postgres cache provider.
type Config struct {
	ConnString    string
	TableName     string
	PoolSize      int
	SweepInterval time.Duration
	DefaultTTL    time.Duration
}

// Option is a functional option for configuring the Postgres cache provider.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		TableName:     "cache_entries",
		PoolSize:      5,
		SweepInterval: 1 * time.Minute,
	}
}

// WithConnString sets the Postgres connection string.
// Example: postgres://user:pass@localhost:5432/dbname
func WithConnString(connString string) Option {
	return func(cfg *Config) {
		cfg.ConnString = connString
	}
}

// WithTableName sets the custom table name for cache entries.
// Defaults to "cache_entries".
func WithTableName(name string) Option {
	return func(cfg *Config) {
		cfg.TableName = name
	}
}

// WithPoolSize sets the maximum connection pool size.
func WithPoolSize(n int) Option {
	return func(cfg *Config) {
		cfg.PoolSize = n
	}
}

// WithSweepInterval sets the interval for the background sweep goroutine.
func WithSweepInterval(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.SweepInterval = d
	}
}

// WithDefaultTTL sets the provider-level default TTL for all keys.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.DefaultTTL = ttl
	}
}

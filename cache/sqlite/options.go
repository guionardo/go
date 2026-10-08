package sqlite

import "time"

type (
	// Config holds configuration for the SQLite cache provider.
	Config struct {
		Name          string
		Path          string
		Memory        bool
		DefaultTTL    time.Duration
		SweepInterval time.Duration
	}

	// Option is a functional option for configuring the SQLite cache provider.
	Option func(*Config)
)

// defaultConfig returns the zero-value configuration: an in-memory cache
// (zero value = memory mode).
func defaultConfig() *Config {
	return &Config{}
}

// WithName sets the cache name used to derive the default location,
// os.UserCacheDir()/<name>/cache.db. The name must match [A-Za-z0-9._-]+.
func WithName(name string) Option {
	return func(cfg *Config) {
		cfg.Name = name
	}
}

// WithPath sets an explicit database file path. The parent directory is
// created as needed. Paths containing '?' or '#' are rejected with
// ErrInvalidPath. The literal ":memory:" selects memory mode.
func WithPath(path string) Option {
	return func(cfg *Config) {
		cfg.Path = path
	}
}

// WithMemory forces an in-memory cache. It exists for discoverability; the
// zero-value configuration is already memory mode.
func WithMemory() Option {
	return func(cfg *Config) {
		cfg.Memory = true
	}
}

// WithDefaultTTL sets the provider-level default TTL for all keys.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.DefaultTTL = ttl
	}
}

// WithSweepInterval enables a periodic expired-row sweeper. The default is no
// sweeper at all — a deliberate divergence from mem/postgres' default-on
// sweeper (D-08): an unset or non-positive interval creates no goroutine, no
// ticker, and no cancellation channels.
func WithSweepInterval(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.SweepInterval = d
	}
}

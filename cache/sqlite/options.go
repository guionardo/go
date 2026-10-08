package sqlite

import "time"

// Config holds configuration for the SQLite cache provider.
type Config struct {
	Name       string
	Path       string
	Memory     bool
	DefaultTTL time.Duration
}

// Option is a functional option for configuring the SQLite cache provider.
type Option func(*Config)

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

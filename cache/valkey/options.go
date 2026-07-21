package valkey

import "time"

// Config holds Valkey-specific cache configuration.
type Config struct {
	Addr       string
	Password   string
	DB         int
	PoolSize   int
	DefaultTTL time.Duration
}

// Option is a functional option for configuring a Valkey cache provider.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		Addr:     "localhost:6379",
		PoolSize: 10,
	}
}

// WithAddr sets the Valkey server address.
func WithAddr(addr string) Option {
	return func(cfg *Config) {
		cfg.Addr = addr
	}
}

// WithPassword sets the Valkey server password.
func WithPassword(password string) Option {
	return func(cfg *Config) {
		cfg.Password = password
	}
}

// WithDB sets the Valkey database index.
func WithDB(db int) Option {
	return func(cfg *Config) {
		cfg.DB = db
	}
}

// WithPoolSize sets the Valkey connection pool size.
func WithPoolSize(n int) Option {
	return func(cfg *Config) {
		cfg.PoolSize = n
	}
}

// WithDefaultTTL sets the provider-level default TTL for all keys.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.DefaultTTL = ttl
	}
}

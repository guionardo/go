package redis

import "time"

// Config holds Redis-specific cache configuration.
type Config struct {
	Addr       string
	Password   string
	DB         int
	PoolSize   int
	DefaultTTL time.Duration
}

// Option is a functional option for configuring a Redis cache provider.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		Addr:     "localhost:6379",
		PoolSize: 10,
	}
}

// WithAddr sets the Redis server address.
func WithAddr(addr string) Option {
	return func(cfg *Config) {
		cfg.Addr = addr
	}
}

// WithPassword sets the Redis server password.
func WithPassword(password string) Option {
	return func(cfg *Config) {
		cfg.Password = password
	}
}

// WithDB sets the Redis database index.
func WithDB(db int) Option {
	return func(cfg *Config) {
		cfg.DB = db
	}
}

// WithPoolSize sets the Redis connection pool size.
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

package memcache

import "time"

// Config holds configuration for the Memcache cache provider.
type Config struct {
	Servers      []string
	Timeout      time.Duration
	DefaultTTL   time.Duration
	MaxIdleConns int
}

// Option is a functional option for configuring the Memcache cache provider.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		Servers:      []string{"localhost:11211"},
		Timeout:      100 * time.Millisecond,
		MaxIdleConns: 2,
	}
}

// WithServers sets the list of memcache servers.
func WithServers(servers ...string) Option {
	return func(cfg *Config) {
		cfg.Servers = servers
	}
}

// WithTimeout sets the client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithDefaultTTL sets the provider-level default TTL.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.DefaultTTL = ttl
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(n int) Option {
	return func(cfg *Config) {
		cfg.MaxIdleConns = n
	}
}

package cache

import (
	"time"
)

// Config holds shared cache configuration for providers.
type Config struct {
	DefaultTTL time.Duration
}

// Option is a functional option for configuring a cache provider.
type Option interface {
	// Apply applies the option to the configuration.
	Apply(*Config)
}

type optionFunc func(*Config)

// Apply applies the option to the configuration.
func (f optionFunc) Apply(cfg *Config) {
	f(cfg)
}

// WithDefaultTTL sets the provider-level default TTL for all keys.
func WithDefaultTTL(ttl time.Duration) Option {
	return optionFunc(func(cfg *Config) {
		cfg.DefaultTTL = ttl
	})
}

package cache

import (
	"time"
)

type config struct {
	defaultTTL time.Duration
}

// Option is a functional option for configuring a cache provider.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) {
	f(cfg)
}

// WithDefaultTTL sets the provider-level default TTL for all keys.
func WithDefaultTTL(ttl time.Duration) Option {
	return optionFunc(func(cfg *config) {
		cfg.defaultTTL = ttl
	})
}

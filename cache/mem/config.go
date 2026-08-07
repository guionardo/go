package mem

import "time"

type (
	Config struct {
		DefaultTTL    time.Duration
		MaxEntries    int
		SweepInterval time.Duration
	}
	ConfigFunc func(*Config)
)

func WithDefaultTTL(ttl time.Duration) ConfigFunc {
	return func(cfg *Config) {
		cfg.DefaultTTL = ttl
	}
}

func WithMaxEntries(n int) ConfigFunc {
	return func(cfg *Config) {
		cfg.MaxEntries = n
	}
}
func WithSweepInterval(interval time.Duration) ConfigFunc {
	return func(cfg *Config) {
		cfg.SweepInterval = interval
	}
}

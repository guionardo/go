package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptionFunc_Apply(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	opt := optionFunc(func(c *Config) {
		c.DefaultTTL = 10 * time.Second
	})
	opt.Apply(cfg)

	assert.Equal(t, 10*time.Second, cfg.DefaultTTL)
}

func TestWithDefaultTTL(t *testing.T) {
	t.Parallel()

	opt := WithDefaultTTL(5 * time.Minute)
	cfg := &Config{}
	opt.Apply(cfg)

	assert.Equal(t, 5*time.Minute, cfg.DefaultTTL)
}

func TestWithDefaultTTL_Zero(t *testing.T) {
	t.Parallel()

	opt := WithDefaultTTL(0)
	cfg := &Config{DefaultTTL: 1 * time.Minute}
	opt.Apply(cfg)

	assert.Equal(t, time.Duration(0), cfg.DefaultTTL)
}

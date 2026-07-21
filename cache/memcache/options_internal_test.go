package memcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions_WithServers(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithServers("s1:11211", "s2:11211")(cfg)

	assert.Equal(t, []string{"s1:11211", "s2:11211"}, cfg.Servers)
}

func TestOptions_WithTimeout(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithTimeout(5 * time.Second)(cfg)

	assert.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestOptions_WithDefaultTTL(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithDefaultTTL(30 * time.Second)(cfg)

	assert.Equal(t, 30*time.Second, cfg.DefaultTTL)
}

func TestOptions_WithMaxIdleConns(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithMaxIdleConns(10)(cfg)

	assert.Equal(t, 10, cfg.MaxIdleConns)
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	assert.Equal(t, []string{"localhost:11211"}, cfg.Servers)
	assert.Equal(t, 100*time.Millisecond, cfg.Timeout)
	assert.Equal(t, 2, cfg.MaxIdleConns)
	assert.Equal(t, time.Duration(0), cfg.DefaultTTL)
}

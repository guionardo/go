package valkey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions_WithPassword(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithPassword("secret")(cfg)

	assert.Equal(t, "secret", cfg.Password)
}

func TestOptions_WithDB(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithDB(3)(cfg)

	assert.Equal(t, 3, cfg.DB)
}

func TestOptions_WithPoolSize(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithPoolSize(20)(cfg)

	assert.Equal(t, 20, cfg.PoolSize)
}

func TestOptions_WithDefaultTTL(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithDefaultTTL(30 * time.Second)(cfg)

	assert.Equal(t, 30*time.Second, cfg.DefaultTTL)
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	assert.Equal(t, "localhost:6379", cfg.Addr)
	assert.Equal(t, 10, cfg.PoolSize)
	assert.Empty(t, cfg.Password)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, time.Duration(0), cfg.DefaultTTL)
}

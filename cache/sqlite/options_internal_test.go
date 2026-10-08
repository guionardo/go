package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions_WithName(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithName("app")(cfg)

	assert.Equal(t, "app", cfg.Name)
}

func TestOptions_WithPath(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithPath("/tmp/cache.db")(cfg)

	assert.Equal(t, "/tmp/cache.db", cfg.Path)
}

func TestOptions_WithMemory(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithMemory()(cfg)

	assert.True(t, cfg.Memory)
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

	assert.Empty(t, cfg.Name)
	assert.Empty(t, cfg.Path)
	assert.False(t, cfg.Memory)
	assert.Equal(t, time.Duration(0), cfg.DefaultTTL)
}

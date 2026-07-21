package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions_WithConnString(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithConnString("postgres://user:pass@localhost/mydb")(cfg)

	assert.Equal(t, "postgres://user:pass@localhost/mydb", cfg.ConnString)
}

func TestOptions_WithTableName(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithTableName("my_cache")(cfg)

	assert.Equal(t, "my_cache", cfg.TableName)
}

func TestOptions_WithPoolSize(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithPoolSize(10)(cfg)

	assert.Equal(t, 10, cfg.PoolSize)
}

func TestOptions_WithSweepInterval(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithSweepInterval(30 * time.Second)(cfg)

	assert.Equal(t, 30*time.Second, cfg.SweepInterval)
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

	assert.Equal(t, "cache_entries", cfg.TableName)
	assert.Equal(t, 5, cfg.PoolSize)
	assert.Equal(t, 1*time.Minute, cfg.SweepInterval)
	assert.Empty(t, cfg.ConnString)
	assert.Equal(t, time.Duration(0), cfg.DefaultTTL)
}

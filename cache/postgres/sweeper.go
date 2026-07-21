package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// sweepLoop runs periodically to delete expired cache entries.
func (c *Cache[K, V]) sweepLoop() {
	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.sweep()
		case <-c.stop:
			return
		}
	}
}

// sweep deletes all expired entries from the cache table.
// Sweep is best-effort maintenance — errors are logged but not returned.
func (c *Cache[K, V]) sweep() {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE expires_at IS NOT NULL AND expires_at < NOW()",
		pgx.Identifier{c.tableName}.Sanitize(),
	)

	if _, err := c.pool.Exec(context.Background(), query); err != nil {
		slog.Warn("cache/postgres: sweep failed", "error", err)
	}
}

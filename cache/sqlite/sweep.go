package sqlite

import (
	"context"
	"time"
)

// sweep deletes expired rows in one pass. It is best-effort maintenance
// (D-09): failures are logged and swallowed, cache operations are never
// coupled to sweep errors, and no caller receives a sweep error.
func (c *sqliteCache[K, V]) sweep(ctx context.Context) {
	if _, err := c.db.ExecContext(ctx, SweepSQL, time.Now().UnixNano()); err != nil {
		logger().Warn("cache/sqlite: sweep failed", "error", err)
	}
}

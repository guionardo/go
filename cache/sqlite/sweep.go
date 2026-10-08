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

// startSweeper creates the cancellation channels and launches the periodic
// sweep goroutine. Callers only invoke it when the interval is positive.
func (c *sqliteCache[K, V]) startSweeper() {
	c.stop = make(chan struct{})
	c.done = make(chan struct{})

	go c.sweepLoop()
}

// sweepLoop runs the opt-in periodic sweep until Close cancels it (D-08).
// Each tick's sweep completes on the pinned connection before the next select
// iteration, so ticks cannot overlap.
func (c *sqliteCache[K, V]) sweepLoop() {
	defer close(c.done)

	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.sweep(context.Background())
		case <-c.stop:
			return
		}
	}
}

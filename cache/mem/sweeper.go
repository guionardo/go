package mem

import (
	"context"
	"time"
)

func (c *memoryCache[K, V]) sweepLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.sweep()
		case <-ctx.Done():
			return
		case <-c.stop:
			return
		}
	}
}

func (c *memoryCache[K, V]) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.removeExpired()
}

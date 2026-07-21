package mem

import (
	"time"
)

// sweepLoop runs periodically to evict expired entries.
func (c *Cache[K, V]) sweepLoop() {
	ticker := time.NewTicker(1 * time.Minute)
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

// sweep removes all expired entries from the cache.
func (c *Cache[K, V]) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, e := range c.entries {
		if e.expiresAt != nil && now.After(*e.expiresAt) {
			delete(c.entries, k)
		}
	}
}

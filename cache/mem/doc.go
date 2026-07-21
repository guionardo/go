// Package mem provides an in-memory backend for cache.Cache.
//
// Thread-safe via sync.RWMutex. A background goroutine sweeps expired
// entries on a configurable interval. Passive TTL checking also occurs
// on every Get call for prompt invalidation.
//
// Zero external dependencies.
//
// Usage:
//
//	c := mem.New[string, string](cache.WithDefaultTTL(5*time.Minute))
//	c.Set(ctx, "key", "value")
package mem

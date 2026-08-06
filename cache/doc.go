// Package cache provides a generic key-value cache abstraction
// with pluggable backend providers.
//
// The Cache[K, V] interface exposes Get, Set, Delete, GetOrSet, and Close —
// all accepting context.Context for cancellation and timeout propagation.
// GetOrSet's setter receives a context.Context (the caller's ctx) and is
// invoked only on a miss; concurrent misses on the same key may be deduplicated.
//
// Usage:
//
//	import "github.com/guionardo/go/cache"
//
//	var c cache.Cache[string, string]
//	c = mem.New[string, string]()
//	c.Set(ctx, "key", "value")
//	v, err := c.Get(ctx, "key")
//
// Providers (importable sub-packages):
//
//	cache/mem       — in-memory (stdlib, zero deps, background TTL sweep)
//	cache/redis     — Redis (go-redis/v9, lazy connect)
//	cache/valkey    — Valkey (valkey-go, eager connect)
//	cache/memcache  — Memcache (gomemcache, lazy connect)
//	cache/postgres  — PostgreSQL (pgx/v5, pgxpool, eager connect)
//
// Configuration via functional options:
//
//	c := redis.New[string, string](
//	    redis.WithAddr("localhost:6379"),
//	    redis.WithDefaultTTL(5*time.Minute),
//	)
//
// Sentinel errors (wrapped with provider prefix):
//
//	var ErrMiss    = errors.New("cache: key not found")
//	var ErrClosed  = errors.New("cache: cache is closed")
//	var ErrCanceled = errors.New("cache: canceled") // wraps the waiter's ctx.Err()
//
// SingleflightGetOrSet[K, V] is an exported helper that dedups concurrent
// GetOrSet misses on the same key. Providers embed it to share one
// implementation of the miss-path setter wrap (see Phase 6).
//
// Consumer code imports providers at construction time only —
// the cache.Cache interface is the only type in business logic.
package cache

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
//	c = mem.New[string, string](ctx) // ctx drives the background TTL sweeper
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
//	cache/sqlite    — embedded SQLite (modernc.org/sqlite, pure Go, durable file or :memory:)
//
// The BatchCache[K, V] interface extends Cache with batch operations (all
// providers implement it via their New constructor — cast to BatchCache to
// access MGet/MSet/MDel):
//
//	bc := c.(cache.BatchCache[string, string])
//	results := bc.MGet(ctx, "a", "b", "c")
//
// Batch operations (supported by all providers):
//
//	MGet(ctx, keys ...K) map[K]V        — retrieve multiple keys (missing keys absent from result)
//	MSet(ctx, items map[K]V, ttl ...time.Duration) — store multiple values with optional single TTL
//	MDel(ctx, keys ...K)                 — delete multiple keys (idempotent)
//
// Each provider uses its client library's single-round-trip primitive: mem takes
// one lock acquisition, redis uses a go-redis Pipeline, valkey uses valkey-go
// DoMulti, memcache uses GetMulti, postgres uses pgx SendBatch, and sqlite uses
// chunked IN queries plus a single-transaction prepared upsert.
// BatchCache[K,V] embeds Cache[K,V] for full backward compatibility.
//
// Configuration via functional options:
//
//	c := redis.New[string, string](
//	    redis.WithAddr("localhost:6379"),
//	    redis.WithDefaultTTL(5*time.Minute),
//	)
//
// Sentinel errors:
//
//	var ErrMiss   = errors.New("cache: key not found")
//	var ErrClosed = errors.New("cache: cache is closed")
//
// cache.ErrMiss is honoured by every provider: the in-memory provider returns
// it bare, while redis, valkey, memcache, postgres and sqlite wrap it with
// their own prefix. cache.ErrClosed is the one with per-provider adoption —
// cache/sqlite is currently the only provider that returns it; the others do
// not signal close. Consumers branch with errors.Is, never with ==.
//
// SingleflightGetOrSet[K, V] is an exported helper that dedups concurrent
// GetOrSet misses on the same key. Providers embed it to share one
// implementation of the miss-path setter wrap (see Phase 6).
//
// Consumer code imports providers at construction time only —
// the cache.Cache interface is the only type in business logic.
package cache

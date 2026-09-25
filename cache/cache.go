package cache

import (
	"context"
	"time"
)

// Cache is a generic key-value cache interface.
// K must be comparable (for in-memory map keys).
// V can be any type; external providers serialize via encoding/json.
type Cache[K comparable, V any] interface {
	// Get retrieves a value by key. Returns ErrMiss if not found.
	Get(ctx context.Context, key K) (V, error)

	// Set stores a value with optional per-key TTL.
	// If ttl is empty, provider-level default is used.
	Set(ctx context.Context, key K, value V, ttl ...time.Duration) error

	// Delete removes a key from the cache.
	Delete(ctx context.Context, key K) error

	// GetOrSet returns the existing value or computes, stores, and returns it.
	// The setter receives a context.Context (the caller's ctx) and is invoked
	// only on a miss. Concurrent misses on the same key may be deduplicated.
	GetOrSet(ctx context.Context, key K, setter func(context.Context) (V, error), ttl ...time.Duration) (V, error)

	// Close cleans up provider resources (connection pools, goroutines).
	Close() error
}

// BatchCache extends Cache with batch operations for multiple keys.
// Implementations: concreteCache (returned by NewConcreteCache).
// Existing Cache[K,V] implementors are unaffected per D-02.
type BatchCache[K comparable, V any] interface {
	Cache[K, V]

	// MGet retrieves values for multiple keys. Only found keys are included
	// in the result map; missing keys are silently absent (D-03).
	MGet(ctx context.Context, keys ...K) map[K]V

	// MSet stores multiple key-value pairs with an optional single TTL (D-04).
	// If ttl is empty, each key uses the provider-level default TTL.
	MSet(ctx context.Context, items map[K]V, ttl ...time.Duration) error

	// MDel removes multiple keys. Idempotent — deleting already-missing
	// keys is not an error (D-05).
	MDel(ctx context.Context, keys ...K) error
}

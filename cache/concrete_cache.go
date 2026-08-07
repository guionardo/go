package cache

import (
	"context"
	"time"
)

type (
	concreteCache[K comparable, V any] struct {
		sf    SingleflightGetOrSet[K, V]
		cache cacher[K, V]
	}
	cacher[K comparable, V any] interface {
		GetFunc(ctx context.Context, key K) (V, error)
		SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error
		DeleteFunc(ctx context.Context, key K) error
		CloseFunc() error
	}
)

func NewConcreteCache[K comparable, V any](c cacher[K, V]) Cache[K, V] {
	return &concreteCache[K, V]{
		cache: c,
	}
}

// Get retrieves a value by key. Returns ErrMiss if not found.
func (c *concreteCache[K, V]) Get(ctx context.Context, key K) (value V, err error) {
	return c.cache.GetFunc(ctx, key)
}

// Set stores a value with optional per-key TTL.
// If ttl is empty, provider-level default is used.
func (c *concreteCache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	return c.cache.SetFunc(ctx, key, value, ttl...)
}

// Delete removes a key from the cache.
func (c *concreteCache[K, V]) Delete(ctx context.Context, key K) error {
	return c.cache.DeleteFunc(ctx, key)
}

// GetOrSet returns the existing value or computes, stores, and returns it.
// The setter receives a context.Context (the caller's ctx) and is invoked
// only on a miss. Concurrent misses on the same key are deduplicated: the
// setter runs once (in the leader's goroutine) and every caller receives the
// identical value or error. The fast-path Get stays OUTSIDE the singleflight
// group (SF-02); the group fn double-checks Get so a concurrent direct Set is
// not clobbered (SF-08). See SingleflightGetOrSet for the full contract.
func (c *concreteCache[K, V]) GetOrSet(
	ctx context.Context,
	key K,
	setter func(context.Context) (V, error),
	ttl ...time.Duration,
) (V, error) {
	return c.sf.Do(ctx, key, c.cache.GetFunc, c.cache.SetFunc, setter, ttl...)
}

// Close cleans up provider resources (connection pools, goroutines).
func (c *concreteCache[K, V]) Close() error {
	return c.cache.CloseFunc()
}

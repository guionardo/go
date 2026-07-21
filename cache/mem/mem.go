// Package mem provides an in-memory backend for cache.Cache[K, V].
//
// The in-memory cache is safe for concurrent use (sync.RWMutex), supports
// per-key TTL with a background sweep goroutine and a passive TTL check on
// Get. It requires no external dependencies and is the default choice for
// testing (swap to a remote provider in production without changing
// consumer code).
//
// Usage:
//
//	c := mem.New[string, string]()
//	c.Set(ctx, "key", "value")
//	v, err := c.Get(ctx, "key")
package mem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guionardo/go/cache"
)

// Cache is an in-memory cache provider implementing cache.Cache[K, V].
type Cache[K comparable, V any] struct {
	mu         sync.RWMutex
	entries    map[K]*entry[V]
	defaultTTL time.Duration
	stop       chan struct{}
}

// New creates a new in-memory cache provider with optional functional options.
func New[K comparable, V any](opts ...cache.Option) *Cache[K, V] {
	cfg := cache.Config{
		DefaultTTL: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt.Apply(&cfg)
	}

	c := &Cache[K, V]{
		entries:    make(map[K]*entry[V]),
		stop:       make(chan struct{}),
		defaultTTL: cfg.DefaultTTL,
	}

	go c.sweepLoop()

	return c
}

// Get retrieves a value by key. Returns cache.ErrMiss if not found or expired.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		var zero V
		return zero, fmt.Errorf("cache/mem: %w", cache.ErrMiss)
	}

	// Passive TTL check — backstop for sweep interval
	if e.expiresAt != nil && time.Now().After(*e.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		var zero V
		return zero, fmt.Errorf("cache/mem: %w", cache.ErrMiss)
	}

	return e.value, nil
}

// Set stores a value with optional per-key TTL.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	expiresAt := c.resolveTTL(ttl...)

	c.mu.Lock()
	c.entries[key] = &entry[V]{value: value, expiresAt: expiresAt}
	c.mu.Unlock()

	return nil
}

// Delete removes a key from the cache.
func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()

	return nil
}

// GetOrSet returns the existing value or computes, stores, and returns it.
func (c *Cache[K, V]) GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error) {
	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	computed, err := setter()
	if err != nil {
		var zero V
		return zero, err
	}

	if err := c.Set(ctx, key, computed, ttl...); err != nil {
		var zero V
		return zero, err
	}

	return computed, nil
}

// Close shuts down the background sweep goroutine.
// Safe to call multiple times (idempotent).
func (c *Cache[K, V]) Close() error {
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}
	return nil
}

func (c *Cache[K, V]) resolveTTL(ttl ...time.Duration) *time.Time {
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().Add(ttl[0])
		return &t
	}
	if c.defaultTTL > 0 {
		t := time.Now().Add(c.defaultTTL)
		return &t
	}
	return nil
}

// compile-time interface assertion
var _ cache.Cache[string, any] = (*Cache[string, any])(nil)

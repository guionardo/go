package memcache

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/guionardo/go/cache"
)

// Cache implements cache.Cache[K, V] using a Memcache backend.
type Cache[K comparable, V any] struct {
	client     *memcache.Client
	defaultTTL time.Duration
}

// memcacheResult carries the result of a gomemcache operation for context cancellation.
type memcacheResult struct {
	item *memcache.Item
	err  error
}

// New creates a new Memcache cache provider with optional functional options.
func New[K comparable, V any](opts ...Option) *Cache[K, V] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	mc := memcache.New(cfg.Servers...)
	mc.Timeout = cfg.Timeout
	mc.MaxIdleConns = cfg.MaxIdleConns

	return &Cache[K, V]{
		client:     mc,
		defaultTTL: cfg.DefaultTTL,
	}
}

// Get retrieves a value by key. Returns cache.ErrMiss if not found.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	keyStr := fmt.Sprint(key)
	ch := make(chan memcacheResult, 1)

	go func() {
		item, err := c.client.Get(keyStr)
		ch <- memcacheResult{item, err}
	}()

	select {
	case <-ctx.Done():
		var zero V
		return zero, fmt.Errorf("cache/memcache: %w", ctx.Err())
	case r := <-ch:
		if r.err == memcache.ErrCacheMiss {
			var zero V
			return zero, fmt.Errorf("cache/memcache: %w", cache.ErrMiss)
		}
		if r.err != nil {
			var zero V
			return zero, fmt.Errorf("cache/memcache: %w", r.err)
		}

		var value V
		if err := json.Unmarshal(r.item.Value, &value); err != nil {
			var zero V
			return zero, fmt.Errorf("cache/memcache: %w", err)
		}

		return value, nil
	}
}

// Set stores a value with optional per-key TTL.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache/memcache: %w", err)
	}

	expiration := c.resolveTTL(ttl...)
	item := &memcache.Item{
		Key:        fmt.Sprint(key),
		Value:      data,
		Expiration: expiration,
	}

	ch := make(chan error, 1)
	go func() {
		ch <- c.client.Set(item)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("cache/memcache: %w", ctx.Err())
	case err := <-ch:
		if err != nil {
			return fmt.Errorf("cache/memcache: %w", err)
		}
		return nil
	}
}

// Delete removes a key from the cache. Idempotent — deleting a missing key is not an error.
func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
	ch := make(chan error, 1)
	go func() {
		ch <- c.client.Delete(fmt.Sprint(key))
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("cache/memcache: %w", ctx.Err())
	case err := <-ch:
		if err == memcache.ErrCacheMiss {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cache/memcache: %w", err)
		}
		return nil
	}
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

// Close is a no-op for memcache — the client does not support Close.
func (c *Cache[K, V]) Close() error {
	return nil
}

// resolveTTL converts the optional TTL to a memcache expiration value.
// Returns 0 for no expiry (memcache protocol: 0 means no expiry).
func (c *Cache[K, V]) resolveTTL(ttl ...time.Duration) int32 {
	var d time.Duration
	switch {
	case len(ttl) > 0 && ttl[0] > 0:
		d = ttl[0]
	case c.defaultTTL > 0:
		d = c.defaultTTL
	default:
		return 0
	}

	seconds := d.Seconds()
	if seconds < 1 {
		return 1
	}
	if seconds > float64(math.MaxInt32) {
		return math.MaxInt32
	}
	return int32(seconds)
}

// compile-time interface assertion
var _ cache.Cache[string, any] = (*Cache[string, any])(nil)

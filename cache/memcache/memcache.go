package memcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/bradfitz/gomemcache/memcache"

	"github.com/guionardo/go/cache"
)

// memcacheCache is the Memcache-backed provider. It implements the
// cache.cacher primitive interface (GetFunc/SetFunc/DeleteFunc/CloseFunc);
// New wraps it in a cache.NewConcreteCache, which supplies the shared Cache
// surface (singleflight GetOrSet dedup, Cache interface).
type memcacheCache[K comparable, V any] struct {
	client     *memcache.Client
	defaultTTL time.Duration
}

// memcacheResult carries the result of a gomemcache operation for context cancellation.
type memcacheResult struct {
	item *memcache.Item
	err  error
}

// New creates a new Memcache cache provider with optional functional options.
// Returns a cache.BatchCache[K,V] (embeds Cache[K,V] for backward compatibility).
func New[K comparable, V any](opts ...Option) cache.BatchCache[K, V] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	mc := memcache.New(cfg.Servers...)
	mc.Timeout = cfg.Timeout
	mc.MaxIdleConns = cfg.MaxIdleConns

	return cache.NewConcreteCache(&memcacheCache[K, V]{
		client:     mc,
		defaultTTL: cfg.DefaultTTL,
	})
}

// GetFunc retrieves a value by key. Returns cache.ErrMiss if not found.
func (c *memcacheCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
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

// SetFunc stores a value with optional per-key TTL.
func (c *memcacheCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
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

// DeleteFunc removes a key from the cache. Idempotent — deleting a missing key is not an error.
func (c *memcacheCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
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

// CloseFunc is a no-op for memcache — the client does not support Close.
func (c *memcacheCache[K, V]) CloseFunc() error {
	return nil
}

// MGetFunc retrieves values for multiple keys using native GetMulti (D-08).
// Missing keys are silently absent from the result map.
// Uses original key ordering to avoid K-type ambiguity (Pitfall 1).
func (c *memcacheCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	keysStr := make([]string, len(keys))
	for i, k := range keys {
		keysStr[i] = fmt.Sprint(k)
	}

	ch := make(chan struct {
		items map[string]*memcache.Item
		err   error
	}, 1)

	go func() {
		items, err := c.client.GetMulti(keysStr)
		ch <- struct {
			items map[string]*memcache.Item
			err   error
		}{items, err}
	}()

	select {
	case <-ctx.Done():
		return nil
	case r := <-ch:
		result := make(map[K]V, len(keys))
		for i, key := range keys {
			item, ok := r.items[keysStr[i]]
			if !ok {
				continue // missing key — skip
			}
			var value V
			if err := json.Unmarshal(item.Value, &value); err != nil {
				continue // deserialization error — skip (D-06 best-effort)
			}
			result[key] = value
		}
		return result
	}
}

// MSetFunc stores multiple key-value pairs using per-key goroutines
// with error accumulation via errors.Join (D-06).
func (c *memcacheCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	type result struct {
		err error
	}

	ch := make(chan result, len(items))
	expiration := c.resolveTTL(ttl...)

	for key, value := range items {
		go func(k K, v V) {
			data, err := json.Marshal(v)
			if err != nil {
				ch <- result{fmt.Errorf("cache/memcache: %w", err)}
				return
			}
			item := &memcache.Item{
				Key:        fmt.Sprint(k),
				Value:      data,
				Expiration: expiration,
			}
			if err := c.client.Set(item); err != nil {
				ch <- result{fmt.Errorf("cache/memcache: %w", err)}
				return
			}
			ch <- result{}
		}(key, value)
	}

	var errs []error
	for range items {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cache/memcache: %w", ctx.Err())
		case r := <-ch:
			if r.err != nil {
				errs = append(errs, r.err)
			}
		}
	}

	return errors.Join(errs...)
}

// MDelFunc removes multiple keys using per-key goroutines
// with error accumulation via errors.Join (D-06).
// Treats ErrCacheMiss as success (idempotent).
func (c *memcacheCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	ch := make(chan error, len(keys))

	for _, key := range keys {
		go func(k K) {
			err := c.client.Delete(fmt.Sprint(k))
			if err == memcache.ErrCacheMiss {
				ch <- nil // idempotent — missing key is not an error
				return
			}
			if err != nil {
				ch <- fmt.Errorf("cache/memcache: %w", err)
				return
			}
			ch <- nil
		}(key)
	}

	var errs []error
	for range keys {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cache/memcache: %w", ctx.Err())
		case err := <-ch:
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

// resolveTTL converts the optional TTL to a memcache expiration value.
// Returns 0 for no expiry (memcache protocol: 0 means no expiry).
func (c *memcacheCache[K, V]) resolveTTL(ttl ...time.Duration) int32 {
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

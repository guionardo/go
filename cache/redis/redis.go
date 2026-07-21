package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/guionardo/go/cache"
)

// Cache is a Redis-backed generic cache implementation.
type Cache[K comparable, V any] struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// New creates a new Redis cache provider.
func New[K comparable, V any](opts ...Option) *Cache[K, V] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	return &Cache[K, V]{
		client:     client,
		defaultTTL: cfg.DefaultTTL,
	}
}

// Get retrieves a value by key. Returns cache.ErrMiss if not found.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	data, err := c.client.Get(ctx, fmt.Sprint(key)).Bytes()
	if err == redis.Nil {
		return zero, fmt.Errorf("cache/redis: %w", cache.ErrMiss)
	}
	if err != nil {
		return zero, fmt.Errorf("cache/redis: %w", err)
	}

	var value V
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf("cache/redis: %w", err)
	}

	return value, nil
}

// Set stores a value with optional per-key TTL.
// If ttl is empty, the provider-level default TTL is used.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache/redis: %w", err)
	}

	expiration := c.resolveTTL(ttl...)
	if err := c.client.Set(ctx, fmt.Sprint(key), data, expiration).Err(); err != nil {
		return fmt.Errorf("cache/redis: %w", err)
	}

	return nil
}

// Delete removes a key from the cache.
func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
	if err := c.client.Del(ctx, fmt.Sprint(key)).Err(); err != nil {
		return fmt.Errorf("cache/redis: %w", err)
	}

	return nil
}

// GetOrSet returns the existing value or computes, stores, and returns it.
func (c *Cache[K, V]) GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error) {
	var zero V

	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	value, err = setter()
	if err != nil {
		return zero, fmt.Errorf("cache/redis: %w", err)
	}

	if err := c.Set(ctx, key, value, ttl...); err != nil {
		return zero, err
	}

	return value, nil
}

// Close cleans up the Redis connection.
func (c *Cache[K, V]) Close() error {
	return c.client.Close()
}

// resolveTTL resolves the effective TTL for a Set operation.
// Precedence: per-call TTL > provider-level default > 0 (no expiry).
func (c *Cache[K, V]) resolveTTL(ttl ...time.Duration) time.Duration {
	if len(ttl) > 0 && ttl[0] > 0 {
		return ttl[0]
	}
	if c.defaultTTL > 0 {
		return c.defaultTTL
	}
	return 0
}

package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/guionardo/go/cache"
)

// redisCache is the Redis-backed provider. It implements the cache.cacher
// primitive interface (GetFunc/SetFunc/DeleteFunc/CloseFunc); New wraps it in
// a cache.NewConcreteCache, which supplies the shared Cache surface
// (singleflight GetOrSet dedup, Cache interface).
type redisCache[K comparable, V any] struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// New creates a new Redis cache provider with optional functional options.
// Returns a cache.Cache sharing the in-memory singleflight GetOrSet.
func New[K comparable, V any](opts ...Option) cache.Cache[K, V] {
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

	return cache.NewConcreteCache(&redisCache[K, V]{
		client:     client,
		defaultTTL: cfg.DefaultTTL,
	})
}

// GetFunc retrieves a value by key. Returns cache.ErrMiss if not found.
func (c *redisCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
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

// SetFunc stores a value with optional per-key TTL.
// If ttl is empty, the provider-level default TTL is used.
func (c *redisCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
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

// DeleteFunc removes a key from the cache.
func (c *redisCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
	if err := c.client.Del(ctx, fmt.Sprint(key)).Err(); err != nil {
		return fmt.Errorf("cache/redis: %w", err)
	}

	return nil
}

// CloseFunc cleans up the Redis connection.
func (c *redisCache[K, V]) CloseFunc() error {
	return c.client.Close()
}

// MGetFunc retrieves values for multiple keys via per-key fallback.
// TODO(07-03): Replace with go-redis Pipeline for single round-trip.
func (c *redisCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	result := make(map[K]V, len(keys))
	for _, key := range keys {
		v, err := c.GetFunc(ctx, key)
		if err == nil {
			result[key] = v
		}
	}
	return result
}

// MSetFunc stores multiple key-value pairs via per-key fallback.
// TODO(07-03): Replace with go-redis Pipeline for single round-trip.
func (c *redisCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	var errs []error
	for key, value := range items {
		if err := c.SetFunc(ctx, key, value, ttl...); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// MDelFunc removes multiple keys via per-key fallback.
// TODO(07-03): Replace with go-redis Pipeline for single round-trip.
func (c *redisCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	var errs []error
	for _, key := range keys {
		if err := c.DeleteFunc(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// resolveTTL resolves the effective TTL for a Set operation.
// Precedence: per-call TTL > provider-level default > 0 (no expiry).
func (c *redisCache[K, V]) resolveTTL(ttl ...time.Duration) time.Duration {
	if len(ttl) > 0 && ttl[0] > 0 {
		return ttl[0]
	}
	if c.defaultTTL > 0 {
		return c.defaultTTL
	}
	return 0
}

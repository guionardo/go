package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	valkey "github.com/valkey-io/valkey-go"

	"github.com/guionardo/go/cache"
)

// valkeyCache is the Valkey-backed provider. It implements the cache.cacher
// primitive interface (GetFunc/SetFunc/DeleteFunc/CloseFunc); New wraps it in
// a cache.NewConcreteCache, which supplies the shared Cache surface
// (singleflight GetOrSet dedup, Cache interface).
type valkeyCache[K comparable, V any] struct {
	client     valkey.Client
	initErr    error
	defaultTTL time.Duration
}

// New creates a new Valkey cache provider with optional functional options.
// Returns a cache.Cache sharing the in-memory singleflight GetOrSet.
// Operations will error if the connection failed at construction time.
func New[K comparable, V any](opts ...Option) cache.Cache[K, V] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Addr},
		Password:    cfg.Password,
		SelectDB:    cfg.DB,
	})

	return cache.NewConcreteCache(&valkeyCache[K, V]{
		client:     client,
		initErr:    err,
		defaultTTL: cfg.DefaultTTL,
	})
}

// GetFunc retrieves a value by key. Returns cache.ErrMiss if not found.
func (c *valkeyCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
	var zero V

	if c.initErr != nil {
		return zero, fmt.Errorf("cache/valkey: %w", c.initErr)
	}

	data, err := c.client.Do(ctx, c.client.B().Get().Key(fmt.Sprint(key)).Build()).ToString()
	if err != nil && errors.Is(err, valkey.Nil) {
		return zero, fmt.Errorf("cache/valkey: %w", cache.ErrMiss)
	}
	if err != nil {
		return zero, fmt.Errorf("cache/valkey: %w", err)
	}

	var value V
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return zero, fmt.Errorf("cache/valkey: %w", err)
	}

	return value, nil
}

// SetFunc stores a value with optional per-key TTL.
// If ttl is empty, the provider-level default TTL is used.
func (c *valkeyCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	if c.initErr != nil {
		return fmt.Errorf("cache/valkey: %w", c.initErr)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache/valkey: %w", err)
	}

	expiration := c.resolveTTL(ttl...)
	cmd := c.client.B().Set().Key(fmt.Sprint(key)).Value(string(data)).Build()
	if expiration > 0 {
		cmd = c.client.B().Set().Key(fmt.Sprint(key)).Value(string(data)).Px(expiration).Build()
	}
	if err := c.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("cache/valkey: %w", err)
	}

	return nil
}

// DeleteFunc removes a key from the cache.
func (c *valkeyCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
	if c.initErr != nil {
		return fmt.Errorf("cache/valkey: %w", c.initErr)
	}

	cmd := c.client.B().Del().Key(fmt.Sprint(key)).Build()
	if err := c.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("cache/valkey: %w", err)
	}

	return nil
}

// CloseFunc cleans up the Valkey connection.
func (c *valkeyCache[K, V]) CloseFunc() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

// resolveTTL resolves the effective TTL for a Set operation.
// Precedence: per-call TTL > provider-level default > 0 (no expiry).
func (c *valkeyCache[K, V]) resolveTTL(ttl ...time.Duration) time.Duration {
	if len(ttl) > 0 && ttl[0] > 0 {
		return ttl[0]
	}
	if c.defaultTTL > 0 {
		return c.defaultTTL
	}
	return 0
}

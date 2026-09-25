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
// Returns a cache.BatchCache sharing the in-memory singleflight GetOrSet.
// Operations will error if the connection failed at construction time.
func New[K comparable, V any](opts ...Option) cache.BatchCache[K, V] {
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

// MGetFunc retrieves values for multiple keys via valkey-go DoMulti.
// Each key is a separate GET command; missing keys are silently excluded
// from the result (D-06 best-effort). Checks initErr first.
func (c *valkeyCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	if c.initErr != nil {
		return nil
	}

	cmds := make([]valkey.Completed, len(keys))
	for i, key := range keys {
		cmds[i] = c.client.B().Get().Key(fmt.Sprint(key)).Build()
	}
	responses := c.client.DoMulti(ctx, cmds...)

	result := make(map[K]V, len(keys))
	for i, key := range keys {
		data, err := responses[i].ToString()
		if errors.Is(err, valkey.Nil) {
			continue // missing key — skip silently
		}
		if err != nil {
			continue // per-key error — skip (D-06 best-effort)
		}
		var value V
		if err := json.Unmarshal([]byte(data), &value); err != nil {
			continue // deserialization error — skip
		}
		result[key] = value
	}
	return result
}

// MSetFunc stores multiple key-value pairs via valkey-go DoMulti.
// A single TTL applies to all keys in the batch. Checks initErr first.
func (c *valkeyCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	if c.initErr != nil {
		return fmt.Errorf("cache/valkey: %w", c.initErr)
	}

	expiration := c.resolveTTL(ttl...)
	cmds := make([]valkey.Completed, 0, len(items))

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cache/valkey: %w", err) // hard failure on marshal
		}
		cmd := c.client.B().Set().Key(fmt.Sprint(key)).Value(string(data)).Build()
		if expiration > 0 {
			cmd = c.client.B().Set().Key(fmt.Sprint(key)).Value(string(data)).Px(expiration).Build()
		}
		cmds = append(cmds, cmd)
	}

	responses := c.client.DoMulti(ctx, cmds...)
	for _, resp := range responses {
		if err := resp.Error(); err != nil {
			return fmt.Errorf("cache/valkey: %w", err)
		}
	}
	return nil
}

// MDelFunc removes multiple keys via valkey-go DoMulti.
// Checks initErr first.
func (c *valkeyCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
	if c.initErr != nil {
		return fmt.Errorf("cache/valkey: %w", c.initErr)
	}

	cmds := make([]valkey.Completed, len(keys))
	for i, key := range keys {
		cmds[i] = c.client.B().Del().Key(fmt.Sprint(key)).Build()
	}

	responses := c.client.DoMulti(ctx, cmds...)
	for _, resp := range responses {
		if err := resp.Error(); err != nil {
			return fmt.Errorf("cache/valkey: %w", err)
		}
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

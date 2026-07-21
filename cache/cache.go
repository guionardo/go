// Package cache provides a generic cache interface with multiple backend providers.
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
	GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error)

	// Close cleans up provider resources (connection pools, goroutines).
	Close() error
}

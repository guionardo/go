// Package cache provides a generic key-value cache interface with pluggable
// backend providers. The core Cache[K, V] interface exposes Get, Set, Delete,
// GetOrSet, and Close — all accepting context.Context for cancellation and
// tracing.
//
// Providers are imported as independent sub-packages:
//
//   import "github.com/guionardo/go/cache/mem"        // in-memory (zero deps)
//   import "github.com/guionardo/go/cache/redis"       // go-redis/v9
//   import "github.com/guionardo/go/cache/valkey"      // valkey-go
//   import "github.com/guionardo/go/cache/memcache"    // gomemcache
//   import "github.com/guionardo/go/cache/postgres"    // pgx/v5
//
// Each provider uses functional options for configuration and serializes
// values via encoding/json. The interface is designed so consumer code
// never imports a provider directly — swap backends by changing the
// constructor call:
//
//	cache.New[string, string](cache.WithDefaultTTL(5*time.Minute))
//	  → mem.New[string, string]()
//	  → redis.New[string, string](redis.WithAddr("localhost:6379"))
//
// Sentinel errors (ErrMiss, ErrClosed) are returned wrapped with the
// provider prefix so callers can errors.Is() against them.
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

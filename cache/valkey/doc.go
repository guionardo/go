// Package valkey provides a Valkey backend for cache.Cache.
//
// Uses github.com/valkey-io/valkey-go. Wire-compatible with Redis.
// Connection is eager — validates connectivity at construction time.
// Values are JSON-serialized with per-key TTL.
//
// Usage:
//
//	c := valkey.New[string, string](
//	    valkey.WithAddr("localhost:6379"),
//	    valkey.WithDefaultTTL(5*time.Minute),
//	)
package valkey

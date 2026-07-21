// Package memcache provides a Memcache backend for cache.Cache.
//
// Uses github.com/bradfitz/gomemcache. Connection is lazy — dials on
// the first query. Operations are wrapped in goroutines for context
// cancellation support. Values are JSON-serialized.
//
// Usage:
//
//	c := memcache.New[string, string](
//	    memcache.WithServers("localhost:11211"),
//	    memcache.WithTimeout(5*time.Second),
//	    memcache.WithDefaultTTL(5*time.Minute),
//	)
package memcache

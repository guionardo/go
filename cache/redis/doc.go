// Package redis provides a Redis backend for cache.Cache.
//
// Uses go-redis/v9. Connection is lazy — dials on the first query.
// Values are JSON-serialized with per-key TTL.
//
// Usage:
//
//	c := redis.New[string, string](
//	    redis.WithAddr("localhost:6379"),
//	    redis.WithPassword("secret"),
//	    redis.WithDB(0),
//	    redis.WithDefaultTTL(5*time.Minute),
//	)
package redis

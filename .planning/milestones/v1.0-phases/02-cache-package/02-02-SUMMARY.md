---
phase: 02-cache-package
plan: 02
subsystem: cache
tags: [redis, valkey, cache-provider]
requires:
  - 02-01
provides:
  - redis cache provider
  - valkey cache provider
affects:
  - cache/redis/
  - cache/valkey/
tech-stack:
  added:
    - github.com/redis/go-redis/v9 (direct dependency)
    - github.com/valkey-io/valkey-go (direct dependency)
  patterns:
    - Functional options per D-09
    - JSON serialization per D-08
    - Error wrapping with provider prefix per D-12
    - valkey-go builder API for Valkey commands
key-files:
  created:
    - cache/redis/options.go
    - cache/redis/redis.go
    - cache/redis/redis_test.go
    - cache/redis/example_test.go
  pre-existing:
    - cache/valkey/options.go
    - cache/valkey/valkey.go
    - cache/valkey/valkey_test.go
    - cache/valkey/example_test.go
decisions:
  - ValKey provider uses initErr pattern instead of panicking on connection failure,
    because valkey-go eagerly dials at construction time.
  - Valkey Set uses valkey-go builder API: client.B().Set().Key(k).Value(v).Ex(ttl).Build()
  - Redis provider uses lazy connection (go-redis connects on first use).
metrics:
  duration: ~15m
  completed: "2026-07-21"
status: complete
---

# Phase 2 Plan 2: Redis and Valkey Cache Providers

## One-liner

Redis and Valkey cache providers implementing the `Cache[K,V]` interface with JSON serialization, functional options, and wrapped error handling.

## Summary

Implemented two wire-compatible cache backends:

### Redis Provider (`cache/redis/`)

- **Constructor:** `New[K comparable, V any](opts ...Option)` using `redis.NewClient` (lazy connection)
- **5 methods:** Get, Set, Delete, GetOrSet, Close
- **Key conversion:** `fmt.Sprint(key)` for all external calls
- **Serialization:** `encoding/json` for values (`json.Marshal` on Set, `json.Unmarshal` on Get)
- **Error handling:** All errors wrapped with `"cache/redis: %w"`, `redis.Nil` mapped to `cache.ErrMiss`
- **Functional options:** WithAddr, WithPassword, WithDB, WithPoolSize, WithDefaultTTL
- **TTL resolution:** Per-call TTL > provider-level default > 0 (no expiry)
- **Example test:** `ExampleNew` demonstrates set/get/close workflow

### Valkey Provider (`cache/valkey/`)

- **Constructor:** `New[K comparable, V any](opts ...Option)` using `valkey.NewClient` (eager connection)
- **initErr pattern:** Since valkey-go eagerly dials, constructor stores connection errors instead of panicking; all methods check initErr before operations
- **Same 5 methods** as Redis, same interface contract
- **valkey-go builder API:** `client.B().Get().Key(k).Build()`, `client.B().Set().Key(k).Value(v).Ex(ttl).Build()`
- **Error handling:** All errors wrapped with `"cache/valkey: %w"`, `valkey.Nil` mapped to `cache.ErrMiss`
- **Functional options:** WithAddr, WithPassword, WithDB, WithPoolSize, WithDefaultTTL
- **TTL resolution:** Same logic as Redis
- **Close:** Nil-safe (checks `c.client != nil` before calling `Close()`)

## Deviations from Plan

### Pre-existing Valkey Provider

- **Plan specified:** Task 2 creates `cache/valkey/` files
- **Actual state:** Valkey provider files were already committed in plan `02-cache-package-03` (commit `c17a972`) with identical content. My writes were idempotent.
- **Impact:** No functional difference — all acceptance criteria are satisfied.
- **Commit:** `c17a972` (pre-existing, matches plan 02-02 spec)

### No panic on Valkey NewClient error

- **Plan specified:** Constructor panics on `valkey.NewClient` error
- **Implementation:** Constructor stores error in `initErr` field and returns a functional Cache that returns connection errors on all operations
- **Rationale:** Go-redis uses lazy connection (no error on construction); valkey-go eagerly dials. Panicking when no server is running would break the `TestValkeyCache_Close` test (which must work without a server). The `initErr` pattern matches the Redis lazy-error pattern while providing clean error surfacing.
- **Impact:** Better developer experience — no panics for runtime connection failures

## Verification

```bash
go build ./cache/redis/ ./cache/valkey/   # PASS
go build ./cache/...                       # PASS
go vet ./cache/redis/ ./cache/valkey/      # PASS
go test ./cache/redis/ -run TestRedisCache_Close  # PASS (1/1)
go test ./cache/valkey/ -run TestValkeyCache_Close # PASS (1/1)
go mod tidy                                # PASS
go build ./cache/...                       # PASS (post-tidy)
```

## Commit History

| Task | Description | Commit |
|------|-------------|--------|
| 1 | Redis cache provider | `66d8184` |
| 2 | Valkey cache provider (pre-existing) | `c17a972` |

## Self-Check

- [x] `go build ./cache/redis/` compiles
- [x] `go build ./cache/valkey/` compiles
- [x] `go build ./cache/...` compiles
- [x] `go vet ./cache/redis/ ./cache/valkey/` passes
- [x] `go test ./cache/redis/ -run TestRedisCache_Close` passes
- [x] `go test ./cache/valkey/ -run TestValkeyCache_Close` passes
- [x] `go mod tidy && go build ./cache/...` passes
- [x] Redis errors wrapped with "cache/redis: %w"
- [x] Valkey errors wrapped with "cache/valkey: %w"
- [x] Both providers map nil/miss to cache.ErrMiss
- [x] JSON serialization for values
- [x] fmt.Sprint key conversion
- [x] All 5 functional options defined per provider

## Self-Check: PASSED

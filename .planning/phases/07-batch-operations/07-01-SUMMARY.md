---
phase: 07-batch-operations
plan: 01
subsystem: cache
tags:
  - batch-operations
  - infrastructure
  - interface
  - tests
  - coverage
requires:
  - Existing Cache[K,V] interface
  - Existing cacher interface
  - Existing concreteCache type
provides:
  - BatchCache[K,V] embedding interface
  - cacher.MGetFunc/MSetFunc/MDelFunc methods
  - concreteCache.MGet/MSet/MDel delegation
  - fakeCacher batch test double
  - Provider stub implementations
affects:
  - All 5 providers need optimized batch implementations (07-02 through 07-04)
tech-stack: {}
key-files:
  created: []
  modified:
    - cache/cache.go
    - cache/concrete_cache.go
    - cache/concrete_cache_test.go
    - cache/mem/mem.go
    - cache/mem/mem_test.go
    - cache/memcache/memcache.go
    - cache/redis/redis.go
    - cache/valkey/valkey.go
    - cache/postgres/postgres.go
    - cache/doc.go
decisions:
  - D-01: Batch ops on cacher interface
  - D-02: Cache[K,V] unchanged, BatchCache embeds it
  - D-03: MGet returns map[K]V with only found keys
  - D-04: MSet accepts variadic single TTL
  - D-05: MDel idempotent
  - D-06: errors.Join for error accumulation
metrics:
  duration_minutes: ~15
  completed_date: "2026-08-07"
status: complete
---

# Phase 7 Plan 1: Core Infrastructure Summary

**One-liner:** Added `BatchCache[K,V]` embedding interface, expanded `cacher` with 3 batch methods, added `concreteCache` delegation, changed `NewConcreteCache` return type, provided provider stubs, and added complete test coverage with 14 batch test subtests.

## Decisions Applied

| Decision | Implementation |
|----------|---------------|
| **D-01** (batch ops on cacher) | `cacher` interface gained `MGetFunc`/`MSetFunc`/`MDelFunc` in `concrete_cache.go` |
| **D-02** (Cache unchanged) | `Cache[K,V]` interface untouched; `BatchCache[K,V]` embeds it in `cache.go` |
| **D-03** (MGet return type) | `MGet(ctx, keys ...K) map[K]V` — only found keys in result |
| **D-04** (MSet TTL) | `MSet(ctx, items map[K]V, ttl ...time.Duration)` — optional single TTL |
| **D-05** (MDel idempotent) | `MDel(ctx, keys ...K)` — deleting missing keys is not an error |
| **D-06** (error aggregation) | `errors.Join` used in provider stubs for batch error accumulation |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Provider stubs required for compilation**
- **Found during:** Task 1 verification
- **Issue:** Adding `MGetFunc`/`MSetFunc`/`MDelFunc` to `cacher` interface broke compilation for all 5 provider types that implement it
- **Fix:** Added minimal per-key-loop stub implementations to `mem`, `redis`, `valkey`, `memcache`, and `postgres` providers. Optimized implementations deferred to 07-02 (memcache), 07-03 (redis/valkey), and 07-04 (postgres).
- **Files modified:** `cache/mem/mem.go`, `cache/redis/redis.go`, `cache/valkey/valkey.go`, `cache/memcache/memcache.go`, `cache/postgres/postgres.go`
- **Commit:** 61a8009

**2. [Rule 3 - Blocking] fakeCacher missing batch methods blocked `go vet`**
- **Found during:** Task 1 verification (`go vet ./cache/...`)
- **Issue:** `fakeCacher` in `concrete_cache_test.go` didn't have the new batch methods, so it no longer implemented `cacher`
- **Fix:** Added `MGetFunc`/`MSetFunc`/`MDelFunc` to `fakeCacher` with locking and `f.err` check pattern
- **Files modified:** `cache/concrete_cache_test.go`
- **Commit:** 61a8009

### Scope Adjustments

**3. [Coverage compliance] Added unit tests alongside interface changes**
- **Reason:** AGENTS.md requires `make coverage-quick` to pass before every commit. Adding 3 untested methods to `concrete_cache.go` and `mem.go` dropped file coverage below 70%.
- **What was done:** Wrote `TestConcreteCache_Batch` (7 subtests) and `TestMemCache_Batch` (6 subtests) covering found/missing/empty/error/TTL scenarios.
- **Note:** This effectively merged Task 2 (tests) into Task 1 to comply with project commit rules. Tests follow TDD behavior spec from the plan.
- **Commit:** 61a8009

## Known Stubs

| Stub | File | Lines | Reason |
|------|------|-------|--------|
| `redisCache` batch methods | `cache/redis/redis.go` | After CloseFunc | Per-key fallback — replace with Pipeline in 07-03 |
| `valkeyCache` batch methods | `cache/valkey/valkey.go` | After CloseFunc | Per-key fallback — replace with DoMulti in 07-03 |
| `memcacheCache` batch methods | `cache/memcache/memcache.go` | After CloseFunc | Per-key fallback — replace with GetMulti + goroutines in 07-02 |
| `postgresCache` batch methods | `cache/postgres/postgres.go` | Before resolveTTL | Per-key fallback — replace with SendBatch in 07-04 |

Each stub is marked with a `TODO(07-N):` comment referencing the plan that will optimize it.

## Threat Surface Scan

No new security surface introduced. Batch operations use the same serialization (`encoding/json`) and same context propagation as existing single-key operations. Per the plan's threat model (T-07-01, T-07-02), both threats are accepted at LOW severity — no mitigation code required.

## Self-Check: PASSED

- ✅ `go build ./cache/...` — compiles cleanly
- ✅ `go vet ./cache/...` — no issues
- ✅ `go test ./cache/ -run TestConcreteCache_Batch -v -count=1` — 8 passed
- ✅ `go test ./cache/ -race -run TestConcreteCache_Batch -count=1` — 8 passed
- ✅ `go test ./cache/ -count=1` — 44 passed
- ✅ `go test ./cache/mem/ -race -count=1` — 30 passed
- ✅ `make coverage-quick` — File 70%, Package 80%, Total 75% — all PASS

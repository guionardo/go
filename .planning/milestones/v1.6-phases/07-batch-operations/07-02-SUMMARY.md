---
phase: 07-batch-operations
plan: 02
subsystem: cache
tags:
  - batch-operations
  - mem
  - memcache
  - single-lock
  - getmulti
  - goroutines
requires:
  - phase: 07-batch-operations
    plan: 01
    provides: BatchCache interface, cacher batch methods, concreteCache delegation
provides:
  - mem.MGetFunc/MSetFunc/MDelFunc with single lock acquisition (D-07)
  - memcache.MGetFunc using native GetMulti (D-08)
  - memcache.MSetFunc/MDelFunc using per-key goroutines + errors.Join
  - mem.New and memcache.New return cache.BatchCache[K,V]
  - Provider batch tests (mem external + memcache internal + external)
affects:
  - 07-03: redis/valkey batch operations
  - 07-04: postgres batch operations
tech-stack:
  added: []
  patterns:
    - mem single-lock batch (D-07)
    - memcache GetMulti batch MGet (D-08)
    - memcache goroutine-per-key error accumulation
    - Internal test pattern for network-dependent providers
key-files:
  created: []
  modified:
    - cache/mem/mem.go
    - cache/mem/mem_test.go
    - cache/memcache/memcache.go
    - cache/memcache/memcache_test.go
    - cache/memcache/memcache_internal_test.go
key-decisions:
  - "mem batch ops: single lock acquisition, no error accumulation (D-07)"
  - "memcache batch ops: GetMulti for MGet, goroutine-per-key for MSet/MDel (D-08)"
  - "Internal tests for memcache batch cover error pathways without server"
patterns-established:
  - "mem batch ops: RLock for MGet, Lock for MSet/MDel, single acquisition per batch"
  - "memcache batch: goroutine+select for context cancellation, errors.Join for accumulation"
  - "Network provider internal tests: construct cache struct directly with library client with no servers"
requirements-completed:
  - BATCH-03
  - BATCH-04
coverage:
  - id: D1
    description: "mem.MGetFunc uses single RLock acquisition and returns only found keys"
    requirement: BATCH-03
    verification:
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch/mget_returns_only_found_keys"
        status: pass
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch/mget_empty_keys_returns_empty_map"
        status: pass
    human_judgment: false
  - id: D2
    description: "mem.MSetFunc uses single Lock acquisition and stores all keys"
    requirement: BATCH-03
    verification:
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch/mset_stores_all_keys"
        status: pass
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch/mset_empty_map_does_not_error"
        status: pass
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch/mset_with_ttl_stores_keys"
        status: pass
    human_judgment: false
  - id: D3
    description: "mem.MDelFunc uses single Lock acquisition and is idempotent"
    requirement: BATCH-03
    verification:
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch/mdel_deletes_all_keys"
        status: pass
    human_judgment: false
  - id: D4
    description: "memcache.MGetFunc uses native GetMulti, returns found keys with original ordering"
    requirement: BATCH-04
    verification:
      - kind: unit
        ref: "cache/memcache/memcache_internal_test.go#TestMemcacheBatch_NoServer/mget_returns_empty_on_error"
        status: pass
    human_judgment: true
    rationale: "Full MGet requires a live memcache server; only error pathway tested without server"
  - id: D5
    description: "memcache.MSetFunc uses per-key goroutines with errors.Join"
    requirement: BATCH-04
    verification:
      - kind: unit
        ref: "cache/memcache/memcache_internal_test.go#TestMemcacheBatch_NoServer/mset_returns_error"
        status: pass
      - kind: unit
        ref: "cache/memcache/memcache_internal_test.go#TestMemcacheBatch_NoServer/mset_empty_map_no_error"
        status: pass
    human_judgment: false
  - id: D6
    description: "memcache.MDelFunc uses per-key goroutines, idempotent with error accumulation"
    requirement: BATCH-04
    verification:
      - kind: unit
        ref: "cache/memcache/memcache_internal_test.go#TestMemcacheBatch_NoServer/mdel_returns_error"
        status: pass
      - kind: unit
        ref: "cache/memcache/memcache_internal_test.go#TestMemcacheBatch_NoServer/mdel_empty_keys_no_error"
        status: pass
    human_judgment: false
  - id: D7
    description: "mem.New and memcache.New return cache.BatchCache[K,V] (backward compatible)"
    verification:
      - kind: unit
        ref: "cache/mem/mem_test.go#TestMemCache_Batch"
        status: pass
    human_judgment: false
duration: ~20min
completed: 2026-08-07
status: complete
---

# Phase 7 Plan 2: mem and memcache Batch Operations Summary

**mem single-lock batch (D-07), memcache GetMulti + goroutine-per-key batch (D-08), New() return type changed to BatchCache for both providers**

## Performance

- **Duration:** ~20 min
- **Tasks:** 3 (2 implementation + 1 tests)
- **Files modified:** 5

## Accomplishments
- mem.MGetFunc: single RLock acquisition, iterate keys via store.get, returns only found keys (D-07)
- mem.MSetFunc: single Lock acquisition, resolveTTL once, store.set all items
- mem.MDelFunc: single Lock acquisition, store.delete all keys
- memcache.MGetFunc: native GetMulti for single round-trip, original key ordering to avoid K-type ambiguity (Pitfall 1)
- memcache.MSetFunc: per-key goroutines with error accumulation via errors.Join
- memcache.MDelFunc: per-key goroutines, ErrCacheMiss treated as success (idempotent)
- mem.New and memcache.New return `cache.BatchCache[K,V]` (backward compatible — embeds `Cache[K,V]`)
- Provider batch tests: mem (7 subtests, always runs), memcache (6 subtests with skipIfNoMemcache + 5 internal subtests without server)

## Task Commits

Each task was committed atomically:

1. **Task 1+2: Implement batch ops + New return type for mem and memcache** - `2f4c632` (feat)
2. **Task 3: Add provider-level batch tests** - `bcbf97d` (test)

## Files Created/Modified
- `cache/mem/mem.go` - New() return type changed to cache.BatchCache[K,V]
- `cache/mem/mem_test.go` - Removed type assertions (New() returns BatchCache directly)
- `cache/memcache/memcache.go` - Optimized MGetFunc (GetMulti), MSetFunc/MDelFunc (goroutines + errors.Join), New() returns BatchCache
- `cache/memcache/memcache_test.go` - Added TestMemcacheCache_Batch (6 subtests, skipIfNoMemcache)
- `cache/memcache/memcache_internal_test.go` - Added TestMemcacheBatch_NoServer (5 subtests, no server needed)

## Decisions Made
- mem batch ops acquire lock once per batch (D-07) — no error accumulation needed since map ops cannot fail
- memcache batch uses native GetMulti for MGet (D-08) and goroutine-per-key for MSet/MDel
- Context cancellation respected via goroutine+select pattern in memcache batch methods
- Internal tests added for memcache batch error pathways to maintain coverage thresholds

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added internal batch tests for memcache coverage**
- **Found during:** Task 3 verification
- **Issue:** Plan expected `skipIfNoMemcache` pattern for memcache batch tests, but replacing stub methods with complex goroutine-based implementations added ~67 uncovered statements, dropping total coverage below 75% threshold
- **Fix:** Added `TestMemcacheBatch_NoServer` in `memcache_internal_test.go` that constructs `memcacheCache` with no-server client — exercises goroutine creation, channel send/receive, error accumulation via errors.Join, and error wrapping without needing a live memcache
- **Files modified:** `cache/memcache/memcache_internal_test.go`
- **Verification:** 5 subtests pass; memcache package coverage jumped from 15.1% to 56.2%; total coverage from 74.6% to 77.1%
- **Committed in:** bcbf97d (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical — coverage compliance)
**Impact on plan:** Maintained required coverage thresholds. Internal tests exercise error pathways that would otherwise only be tested end-to-end.

## Issues Encountered
- `rtk go test -run TestMemcacheCache_Batch` returned "No tests found" via the rtk output wrapper, but the test binary actually has the test registered and runs correctly when executed directly. This is an rtk output filtering issue, not a test compilation issue.
- Coverage dropped below 75% threshold after replacing memcache stubs with complex batch implementations — resolved by adding internal tests.

## Self-Check: PASSED

- ✅ `go build ./cache/mem/...` — compiles
- ✅ `go build ./cache/memcache/...` — compiles
- ✅ `go test ./cache/mem/ -run TestMemCache_Batch -v -count=1` — 7 passed
- ✅ `go test ./cache/memcache/ -run TestMemcacheCache_Batch -v -count=1` — skipped (no memcache)
- ✅ `go test ./cache/mem/ -race -run TestMemCache_Batch -count=1` — 7 passed
- ✅ `go test ./cache/memcache/ -race -count=1` — 22 passed
- ✅ `make coverage-quick` — File 70%, Package 80%, Total 75% — all PASS (77.1%)

## Next Phase Readiness
- mem and memcache providers fully optimized for batch operations
- Redis, Valkey, and Postgres providers still have per-key stubs — 07-03 (redis/valkey) and 07-04 (postgres) will optimize them with pipelines/SendBatch

---
*Phase: 07-batch-operations*
*Completed: 2026-08-07*

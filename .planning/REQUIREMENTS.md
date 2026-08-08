# Requirements: go

**Defined:** 2026-08-06
**Core Value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.

## v1 Requirements

### Singleflight GetOrSet

- [ ] **SF-01**: Concurrent GetOrSet misses on the same key run the setter exactly once, and all callers receive the same value
- [x] **SF-02**: The cache package exposes a shared `singleflightGetOrSet[K, V]` helper that wraps only the setter (not the fast-path Get)
- [ ] **SF-03**: All 5 providers (mem, redis, valkey, memcache, postgres) delegate GetOrSet through the shared helper
- [ ] **SF-04**: redis and valkey GetOrSet preserve their historical error prefixes; mem, memcache, and postgres return raw setter errors; the valkey initErr guard stays provider-side
- [x] **SF-05**: A panicking setter is recovered by the wrapper and returned as an error to every waiter
- [x] **SF-06**: A caller whose context is canceled abandons its wait promptly (DoChan + select variant) instead of blocking for the setter duration
- [ ] **SF-07**: A Delete issued while a setter is in flight is not resurrected by the late Set (generation-tombstone re-check before Set)
- [x] **SF-08**: The helper double-checks Get inside the singleflight fn so a concurrent direct Set is not clobbered by a stale in-flight computation
- [ ] **SF-09**: Singleflight GetOrSet passes the race detector and keeps existing per-provider TTL semantics unchanged

### Batch Operations

- [ ] **BATCH-01**: Cache interface exposes `MGet(keys ...K) map[K]V` returning only found keys
- [ ] **BATCH-02**: Cache interface exposes `MSet(items map[K]V, ttl ...time.Duration)` and `MDel(keys ...K)`
- [ ] **BATCH-03**: mem provider implements batch ops by iterating the underlying store with a single lock acquisition
- [ ] **BATCH-04**: memcache provider uses native `GetMulti` for MGet
- [ ] **BATCH-05**: redis and valkey providers use pipelines for MGet/MSet/MDel
- [x] **BATCH-06**: postgres provider batches operations in a single transaction/query batch
- [x] **BATCH-07**: All batch operations handle empty and partial key sets without errors, and pass the race detector

### Benchmark Suite

- [x] **BENCH-01**: Thundering-herd GetOrSet benchmark demonstrates the setter runs N times (naive) vs 1 time (singleflight) under concurrent load
- [x] **BENCH-02**: Batch operations benchmark demonstrates pipeline/GetMulti batching against naive per-key loops
- [x] **BENCH-03**: A `make benchmark` target runs the suite with `-benchmem`

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Strings Package

- **STRNG-01**: String utilities package (truncation, padding, join/split)

### Retry Package

- **RETRY-01**: Retry package with backoff strategies and jitter support

## Out of Scope

| Feature | Reason |
|---------|--------|
| Cache eviction policies (LRU/LFU) | Requires per-provider storage changes; existing TTL-based sweep covers current needs |
| Cross-provider batch consistency guarantees | Providers have divergent semantics; batch ops are best-effort per provider |
| Persistent singleflight dedup across processes | singleflight dedups within one process only; cross-node dedup needs locks/queues out of scope |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SF-01 | Phase 6 | Pending |
| SF-02 | Phase 5 | Complete |
| SF-03 | Phase 6 | Pending |
| SF-04 | Phase 6 | Pending |
| SF-05 | Phase 5 | Complete |
| SF-06 | Phase 5 | Complete |
| SF-07 | Phase 6 | Pending |
| SF-08 | Phase 5 | Complete |
| SF-09 | Phase 6 | Pending |
| BATCH-01 | Phase 7 | Pending |
| BATCH-02 | Phase 7 | Pending |
| BATCH-03 | Phase 7 | Pending |
| BATCH-04 | Phase 7 | Pending |
| BATCH-05 | Phase 7 | Pending |
| BATCH-06 | Phase 7 | Complete |
| BATCH-07 | Phase 7 | Complete |
| BENCH-01 | Phase 8 | Complete |
| BENCH-02 | Phase 8 | Complete |
| BENCH-03 | Phase 8 | Complete |

**Coverage:**

- v1 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-06*
*Last updated: 2026-08-06 after roadmap creation (v1.6 Cache Dedup, Phases 5-8)*

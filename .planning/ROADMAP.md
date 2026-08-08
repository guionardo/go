# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- ✅ **v1.5 Self-Update** — Phase 4 (shipped 2026-07-21)
- 🚧 **v1.6 Cache Dedup** — Phases 5-8 (in progress)

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (5.1, 5.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>✅ v1.4 Core Packages (Phase 1) — SHIPPED 2026-07-21</summary>

- [x] **Phase 1: Cache Package** - Generic Cache over 5 backends (3/3 plans) — completed 2026-07-21

</details>

<details>
<summary>✅ v1.5 Self-Update (Phase 4) — SHIPPED 2026-07-21</summary>

- [x] **Phase 4: Release self-update with swapper binary** - Self-update mechanism (3/3 plans) — completed 2026-07-21

</details>

### 🚧 v1.6 Cache Dedup (In Progress)

**Milestone Goal:** Eliminate duplicate setter work in cache misses and reduce round trips — via singleflight GetOrSet across all 5 providers, batch operations, and a benchmark suite.

- [x] **Phase 5: Shared singleflight helper** - Shared `singleflightGetOrSet` helper in the cache package: setter-only wrap, DoChan cancel variant, panic recovery, Get double-check (completed 2026-08-06)
- [ ] **Phase 6: Provider integration** - All 5 providers delegate GetOrSet through the helper; error-prefix parity, delete-during-flight tombstone guard, race/TTL parity
- [x] **Phase 7: Batch operations** - MGet/MSet/MDel on all 5 providers with provider-appropriate batching (single-lock, GetMulti, pipelines, query batch) — completed 2026-08-08
- [ ] **Phase 8: Benchmark suite** - Thundering-herd dedup and batch batching benchmarks, runnable via `make benchmark`

## Phase Details

### Phase 5: Shared singleflight helper

**Goal**: The cache package exposes a shared `singleflightGetOrSet[K, V]` helper that wraps only the setter, so concurrent misses on the same key run the setter once, never panic callers, and abandon on cancel
**Depends on**: Nothing (first phase of v1.6)
**Requirements**: SF-02, SF-05, SF-06, SF-08
**Success Criteria** (what must be TRUE):

  1. The shared helper runs the setter exactly once for N concurrent misses on the same key and returns the same value to every caller, without holding the fast-path Get in the group
  2. A panicking setter is recovered inside the wrapper — every waiter receives an error, and no panic escapes to any caller
  3. A caller whose context is canceled abandons its wait promptly (cancel latency far below the setter duration) via DoChan + select, instead of blocking until the setter finishes
  4. The helper re-checks Get inside the singleflight fn, so a concurrent direct Set is never clobbered by a stale in-flight computation
  5. The helper passes the race detector under concurrent miss/Set/Delete patterns (core of SF-09; full per-provider verification completes in Phase 6)

**Plans**: TBD

### Phase 6: Provider integration

**Goal**: All 5 providers (mem, redis, valkey, memcache, postgres) delegate GetOrSet through the shared helper with historical error semantics, delete-during-flight protection, and unchanged TTL behavior
**Depends on**: Phase 5
**Requirements**: SF-01, SF-03, SF-04, SF-07, SF-09
**Success Criteria** (what must be TRUE):

  1. On every provider, N concurrent GetOrSet misses on the same key run the setter exactly once and all callers receive the same value; subsequent GetOrSet calls hit the cache without re-running the setter
  2. All 5 providers delegate through the shared helper — no provider keeps a hand-rolled GetOrSet body
  3. redis and valkey GetOrSet errors keep their historical `cache/redis:` / `cache/valkey:` prefixes; mem, memcache, and postgres return raw setter errors; valkey's initErr guard still short-circuits provider-side
  4. A Delete issued while a setter is in flight is not resurrected by the leader's late Set (generation-tombstone re-check before Set)
  5. GetOrSet passes the race detector across all providers and per-provider TTL semantics are unchanged (same expiry behavior as before singleflight)

**Plans**: TBD

### Phase 7: Batch operations

**Goal**: The cache package exposes MGet/MSet/MDel batch operations on all 5 providers with provider-appropriate batching and no errors on empty or partial key sets
**Depends on**: Phase 6
**Requirements**: BATCH-01, BATCH-02, BATCH-03, BATCH-04, BATCH-05, BATCH-06, BATCH-07
**Success Criteria** (what must be TRUE):

  1. Users can fetch multiple keys in one MGet call and receive a map containing only found keys, and can MSet a map of items (with optional TTL) and MDel multiple keys in single calls
  2. mem provider batch operations iterate the underlying store under a single lock acquisition (no per-key locking)
  3. memcache MGet uses native GetMulti; redis and valkey use pipelines for MGet/MSet/MDel; postgres batches operations in a single transaction/query batch
  4. Empty and partial key sets complete without errors, and all batch operations pass the race detector

**Plans**: 4 plans

Plans:
- [ ] 07-01-PLAN.md — Core infrastructure: Cache interface, cacher interface, concreteCache delegation, fakeCacher, unit tests, doc.go
- [ ] 07-02-PLAN.md — mem + memcache providers: single-lock batch ops, GetMulti, per-key goroutines, tests
- [ ] 07-03-PLAN.md — redis + valkey providers: pipeline/DoMulti batch ops, integration tests
- [ ] 07-04-PLAN.md — postgres provider + E2E tests + race detector verification

### Phase 8: Benchmark suite

**Goal**: A benchmark suite that quantifies the singleflight dedup win and the batch batching win, runnable via `make benchmark`
**Depends on**: Phase 7
**Requirements**: BENCH-01, BENCH-02, BENCH-03
**Success Criteria** (what must be TRUE):

  1. The thundering-herd benchmark shows the setter running N times with the naive path vs exactly 1 time with singleflight under concurrent load
   2. The batch benchmark shows pipeline/GetMulti batching outperforming naive per-key loops
   3. `make benchmark` runs the full suite with `-benchmem` and exits successfully

**Plans**: 3 plans

Plans:
- [ ] 08-01-PLAN.md — Thundering-herd singleflight benchmark (mem + Docker-gated all providers)
- [ ] 08-02-PLAN.md — Batch operations benchmark (mem, per-key comparison)
- [ ] 08-03-PLAN.md — `make benchmark` + `make benchmark-quick` + Docker-backed batch provider benchmarks

## Progress

**Execution Order:**
Phases execute in numeric order: 5 → 6 → 7 → 8

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Cache Package | v1.4 | 3/3 | Complete | 2026-07-21 |
| 4. Self-Update | v1.5 | 3/3 | Complete | 2026-07-21 |
| 5. Shared singleflight helper | v1.6 | 3/3 | Complete    | 2026-08-06 |
| 6. Provider integration | v1.6 | TBD | Not started | - |
| 7. Batch operations | v1.6 | 4/4 | Complete | 2026-08-08 |
| 8. Benchmark suite | v1.6 | 0/3 | Not started | - |

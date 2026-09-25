# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- ✅ **v1.5 Self-Update** — Phase 4 (shipped 2026-07-21)
- ✅ **v1.6 Cache Dedup** — Phases 5-9 (shipped 2026-08-08)

## Phases

<details>
<summary>✅ v1.4 Core Packages (Phase 1) — SHIPPED 2026-07-21</summary>

- [x] **Phase 1: Cache Package** - Generic Cache over 5 backends (3/3 plans) — completed 2026-07-21

</details>

<details>
<summary>✅ v1.5 Self-Update (Phase 4) — SHIPPED 2026-07-21</summary>

- [x] **Phase 4: Release self-update with swapper binary** - Self-update mechanism (3/3 plans) — completed 2026-07-21

</details>

<details>
<summary>✅ v1.6 Cache Dedup (Phases 5-9) — SHIPPED 2026-08-08</summary>

**Milestone Goal:** Eliminate duplicate setter work in cache misses and reduce round trips — via singleflight GetOrSet across all 5 providers, batch operations, and a benchmark suite.

- [x] **Phase 5: Shared singleflight helper** - Shared `singleflightGetOrSet` helper in the cache package: setter-only wrap, DoChan cancel variant, panic recovery, Get double-check (completed 2026-08-06)
- [x] **Phase 6: Provider integration** - All 5 providers delegate GetOrSet through the helper; error-prefix parity, delete-during-flight tombstone guard, race/TTL parity
- [x] **Phase 7: Batch operations** - MGet/MSet/MDel on all 5 providers with provider-appropriate batching (single-lock, GetMulti, pipelines, query batch) — completed 2026-08-08
- [x] **Phase 8: Benchmark suite** - Thundering-herd dedup and batch batching benchmarks, runnable via `make benchmark` — completed 2026-08-08
- [x] **Phase 9: Post-v1.6 cleanup** - Config HTTP endpoint, httptest mock ServeMux routing, Windows header matching fix — completed 2026-08-08

</details>

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Cache Package | v1.4 | 3/3 | Complete | 2026-07-21 |
| 4. Self-Update | v1.5 | 3/3 | Complete | 2026-07-21 |
| 5. Shared singleflight helper | v1.6 | 3/3 | Complete | 2026-08-06 |
| 6. Provider integration | v1.6 | — | Complete | 2026-08-08 |
| 7. Batch operations | v1.6 | 4/4 | Complete | 2026-08-08 |
| 8. Benchmark suite | v1.6 | 3/3 | Complete | 2026-08-08 |
| 9. Post-v1.6 cleanup | v1.6 | 1/1 | Complete | 2026-08-08 |

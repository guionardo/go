---
gsd_state_version: 1.0
milestone: v1.6
milestone_name: Cache Dedup
current_phase: 8
current_phase_name: Benchmark suite
status: active
stopped_at: Completed 08-03-PLAN.md
last_updated: "2026-08-08T01:18:13.738Z"
progress:
  total_phases: 4
  completed_phases: 3
  total_plans: 10
  completed_plans: 10
  percent: 75
---

# STATE

## Current

- **Milestone:** v1.6 Cache Dedup ✅
- **Phase:** 8 — Benchmark suite (complete)
- **Plan:** 3 plans completed (08-01 through 08-03)
- **Progress:** [██████████] 100% — milestone complete!

## Status

- Phase 5 (Shared singleflight helper) — complete
- Phase 6 (Provider integration) — retroactively closed (work already shipped)
- Phase 7 (Batch operations) — 07-01 complete, 07-02 complete, 07-03 complete, 07-04 complete
- Phase 8 (Benchmark suite) — 08-01 complete (thundering-herd benchmarks), 08-02 complete (batch benchmarks), 08-03 complete (make benchmark targets + Docker-gated batch providers)

## Phase 8 Plans

| Plan | Wave | Description |
|------|------|-------------|
| 08-01 | 1 | ✅ Thundering-herd singleflight benchmark with mem + 4 Docker providers |
| 08-02 | 1 | ✅ Batch benchmarks (MGet/MSet/MDel) |
| 08-03 | 2 | ✅ Make benchmark targets + Docker-gated batch subtests |

## Phase 7 Plans

| Plan | Wave | Description |
|------|------|-------------|
| 07-01 | 1 | ✅ Core infrastructure: Cache/cacher interfaces, concreteCache, fakeCacher, unit tests, doc.go |
| 07-02 | 2 | ✅ mem + memcache providers: single-lock batch ops, GetMulti, per-key goroutines |
| 07-03 | 2 | redis + valkey providers: Pipeline/DoMulti batch ops, integration tests (parallel w/ 07-02) |
| 07-04 | 3 | ✅ postgres SendBatch batch ops, E2E batch subtests, race detector |

## Last Activity

- **Date:** 2026-08-08
- **Desc:** Phase 8 Plan 01 complete: thundering-herd benchmark in cache/bench_test.go with mem provider (always runs) and Docker-gated subtests for redis, valkey, memcache, postgres. All concurrency levels (10, 50, 100, 500) verified: naive ≈ N setter_runs/op, singleflight ≈ 1 setter_runs/op.

## Session

**Last session:** 2026-08-08T01:18:13.732Z
**Stopped at:** Completed 08-03-PLAN.md
**Resume file:** None

## Performance Metrics

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 07-batch-operations P04 | 7min | 3 tasks | 3 files |
| Phase 08-benchmark-suite P01 | 15min | 2 tasks | 1 file |
| Phase 08-benchmark-suite P02 | 12min | 2 tasks | 1 file |
| Phase 08-benchmark-suite P03 | 8min | 2 tasks | 2 files |

## Decisions

- [Phase ?]: pgx SendBatch used for all postgres batch ops (MGet/MSet/MDel) per D-11
- [Phase ?]: providerCase.fn type changed to cache.BatchCache[string,string] for batch E2E access
- [Phase ?]: Error accumulation via errors.Join for MSetFunc/MDelFunc per D-06
- [Phase 8]: Naive herd benchmark uses TOCTOU pattern (check-outside/compute-outside-store) to demonstrate thundering-herd behavior
- [Phase 8]: skipIfNoDocker checks DOCKER_HOST + docker info for robust detection of testcontainers-ready Docker daemon
- [Phase ?]: Phase 8: Naive herd benchmark uses TOCTOU pattern (check-outside/compute-outside-store) to demonstrate thundering-herd behavior
- [Phase ?]: Phase 8: skipIfNoDocker checks DOCKER_HOST + docker info for robust detection of testcontainers-ready Docker daemon
- [Phase ?]: MSet native benchmarks show higher ns/op but lower allocs/op than per-key for mem provider — map iteration inside write lock adds overhead. Batching win more pronounced for network-backed providers where round-trip time dominates.
- [Phase 8 P03]: DOCKER_HOST passthrough added to benchmark targets (like test-e2e) so Docker-backed benchmarks actually run
- [Phase 8 P03]: runBatchBenchmarks helper extracted for provider-agnostic batch benchmark structure
- [Phase 8 P03]: Docker-gated batch subtests added for redis, valkey, memcache, postgres with pinned images

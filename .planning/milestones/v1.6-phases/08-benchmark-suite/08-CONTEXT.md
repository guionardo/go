# Phase 8: Benchmark suite - Context

**Gathered:** 2026-08-08
**Status:** Ready for planning

<domain>
## Phase Boundary

A benchmark suite for the `cache` package that quantifies the singleflight dedup win (thundering-herd) and the batch batching win (pipeline/GetMulti vs per-key loops). Runnable via `make benchmark`.
</domain>

<decisions>
## Implementation Decisions

### Benchmark File Organization
- **D-01:** All benchmarks live in a single file: `cache/bench_test.go`. Keep it focused on the two domains (singleflight, batch) with clear section separation.

### Thundering-Herd Benchmark
- **D-02:** Benchmark with multiple concurrency levels (N=10, 50, 100, 500 concurrent callers minimum) to show how the dedup win scales with contention.
- **D-03:** Use the `mem` provider for the primary benchmark (zero-dependency, network-free). The benchmark compares a naive path (no singleflight, setter runs N times) vs singleflight path (setter runs exactly once).
- **D-04:** Track: total elapsed time, setter invocation count (via atomic counter), allocs/op, bytes/op.

### Batch Benchmark
- **D-05:** Benchmark with multiple batch sizes (N=1, 10, 100, 1000 keys) for MGet, MSet, and MDel operations.
- **D-06:** Compare native batch (pipeline/GetMulti) against concreteCache per-key fallback loop. This quantifies the actual win factor of batching.
- **D-07:** Use the `mem` provider for the primary batch benchmark (zero-dependency). The fallback is the default per-key implementation in `concreteCache`.

### Provider Coverage
- **D-08:** All 5 providers benchmarked (mem, redis, valkey, memcache, postgres). Use `skipIfNoDocker` guard (same pattern as existing E2E tests) for non-mem providers.
- **D-09:** `make benchmark` runs the mem benchmark by default (always available). Docker-backed provider benchmarks run when Docker is available.
- **D-10:** At minimum, the `mem` singleflight herd benchmark runs as part of the default target.

### Make Benchmark Target
- **D-11:** `make benchmark` runs: `go test -bench=. -benchtime=1s -count=5 -benchmem ./cache/...` for statistically meaningful results.
- **D-12:** Add `make benchmark-quick` for a faster run (shorter benchtime, fewer iterations) during development.
- **D-13:** The `make benchmark` target is added to the existing Makefile alongside the existing `test`, `test-e2e`, `coverage`, and `coverage-quick` targets.

### the agent's Discretion
- Benchmark naming conventions (subtest names, benchmark function naming) left to planner/executor.
- Specific benchmark implementations (how to structure goroutine harness for herd test) left to planner.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Cache architecture (what to benchmark)
- `cache/cache.go` — `Cache[K,V]` and `BatchCache[K,V]` interfaces
- `cache/concrete_cache.go` — `cacher` interface and `concreteCache` implementation (per-key fallback comparison target)
- `cache/singleflight.go` — `SingleflightGetOrSet` helper (subject of thundering-herd benchmark)
- `cache/mem/mem.go` — mem provider (primary benchmark target, zero deps)

### Test patterns (benchmark structure)
- `cache/cache_e2e_test.go` — E2E test pattern with `skipIfNoDocker` guard (use same pattern for Docker-backed benchmarks)
- `cache/singleflight_test.go` — Existing concurrent test harness with `sync.WaitGroup` + `atomic.Int64` (adapt for benchmarking)
- `cache/concrete_cache_test.go` — Unit tests with `fakeCacher` (reference for per-key fallback)

### Requirements
- `.planning/REQUIREMENTS.md` — BENCH-01, BENCH-02, BENCH-03

### Build system
- `Makefile` — Add `benchmark` and `benchmark-quick` targets alongside existing targets

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `SingleflightGetOrSet[K,V]` — existing shared helper; benchmark calls `Do` with N concurrent goroutines
- `BatchCache[K,V]` — batch operations interface; benchmark calls `MGet`/`MSet`/`MDel` and compares against per-key fallback
- `concreteCache` — provides per-key fallback for batch ops when provider doesn't override; this IS the fallback to benchmark against
- `atomic.Int64` — used in existing tests for setter-run counting; reuse for benchmark metrics
- `skipIfNoDocker` — existing E2E test guard; reuse for Docker-backed provider benchmarks

### Established Patterns
- Go benchmark functions: `func BenchmarkXxx(b *testing.B)` with `b.N`, `b.ResetTimer()`, `b.RunParallel()`
- Test harness: `sync.WaitGroup` fan-out + `atomic.Int64` counter (from `cache/singleflight_test.go`)
- Provider creation: `mem.New()` returns `BatchCache[K,V]` — direct call, no Docker needed
- Docker-gated tests: `skipIfNoDocker` at function start (from `cache/cache_e2e_test.go`)

### Integration Points
- `cache/bench_test.go` — new file (all benchmarks)
- `Makefile` — add `benchmark` and `benchmark-quick` targets

</code_context>

<specifics>
## Specific Ideas

- Thundering-herd benchmark: N goroutines call `GetOrSet` concurrently with a slow setter (e.g., `time.Sleep(time.Millisecond)`). Naive path runs setter N times; singleflight runs it once.
- Batch benchmark: N keys inserted first, then benchmark `MGet`/`MSet`/`MDel` with varying N. Compare against per-key loop doing sequential `Get`/`Set`/`Delete`.
- Use `b.Run` sub-benchmarks for concurrency levels / batch sizes: `BenchmarkSingleflightGetOrSet/Concurrency50`, `BenchmarkBatch/MGet/Size100`, etc.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 8-Benchmark suite*
*Context gathered: 2026-08-08*

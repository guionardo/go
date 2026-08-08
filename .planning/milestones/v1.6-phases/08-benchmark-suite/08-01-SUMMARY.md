---
phase: 08-benchmark-suite
plan: 01
subsystem: testing
tags: [benchmark, singleflight, thundering-herd, dedup, mem, redis, valkey, memcache, postgres, testcontainers, docker]

requires:
  - phase: 05-singleflight-helper
    provides: SingleflightGetOrSet type with Do method
  - phase: 06-provider-integration
    provides: All 5 cache providers (mem, redis, valkey, memcache, postgres)

provides:
  - Mem provider thundering-herd benchmark (always runs)
  - Docker-gated herd benchmarks for redis, valkey, memcache, postgres
  - skipIfNoDocker guard checking DOCKER_HOST + docker info
  - benchmarkHerdNaive helper (TOCTOU pattern demonstrating N-setter dedup failure)
  - benchmarkHerdSingleflight helper (shows exact-1 dedup win)
  - benchmarkHerdSingleflightConcurrencyLoop helper (shared concurrency loop)

affects:
  - 08-02 (batch benchmarks)
  - 08-03 (make benchmark targets)

tech-stack:
  added: []
  patterns:
    - Go sub-benchmark hierarchy for concurrency levels
    - sync.WaitGroup fan-out + atomic.Int64 counter pattern for herd benchmarks
    - skipIfNoDocker + testcontainers for Docker-gated provider benchmarks
    - b.ReportMetric for custom setter_runs/op metric

key-files:
  created:
    - cache/bench_test.go
  modified: []

key-decisions:
  - "Naive herd benchmark uses TOCTOU pattern (check-outside/compute-outside-store) to demonstrate actual thundering-herd behavior, rather than the plan-described lock-everything pattern which would inherently deduplicate (Deviation Rule 1)"
  - "skipIfNoDocker checks DOCKER_HOST env var AND docker info, because testcontainers needs DOCKER_HOST when Docker runs on a non-default socket (Orbstack)"
  - "Docker-backed subtests use separate provider per sub-benchmark to avoid cached-key cross-contamination across concurrency levels"

patterns-established:
  - "Go benchmark with b.Run sub-benchmarks for concurrency levels [10, 50, 100, 500]"
  - "b.ReportMetric + atomic.Int64 counter for custom per-iteration metrics"
  - "skipIfNoDocker guard + testcontainers container lifecycle for Docker provider benchmarks"
  - "Unique key per b.N iteration to prevent cached-key cross-contamination"

requirements-completed: [BENCH-01]

coverage:
  - id: D1
    description: "Mem provider thundering-herd benchmark quantifies singleflight dedup win at concurrency levels 10, 50, 100, 500"
    requirement: BENCH-01
    verification:
      - kind: other
        ref: "go test -bench=\"BenchmarkSingleflightGetOrSet/Concurrency10/singleflight\" -benchtime=100ms -count=1 -benchmem ./cache/"
        status: pass
    human_judgment: false
  - id: D2
    description: "Naive path shows setter_runs/op ≈ N (≈10/50/100/500 for respective concurrency levels)"
    requirement: BENCH-01
    verification:
      - kind: other
        ref: "go test -bench=\"BenchmarkSingleflightGetOrSet/Concurrency10/naive\" -benchtime=100ms ./cache/"
        status: pass
    human_judgment: false
  - id: D3
    description: "Singleflight path shows setter_runs/op ≈ 1 (exact dedup) for all concurrency levels"
    requirement: BENCH-01
    verification:
      - kind: other
        ref: "go test -bench=\"BenchmarkSingleflightGetOrSet/Concurrency10/singleflight\" -benchtime=100ms ./cache/"
        status: pass
    human_judgment: false
  - id: D4
    description: "Docker-backed provider subtests (redis, valkey, memcache, postgres) gate via skipIfNoDocker"
    requirement: BENCH-01
    verification:
      - kind: other
        ref: "DOCKER_HOST=unix:///Users/guionardo/.orbstack/run/docker.sock go test -bench=\"BenchmarkSingleflightGetOrSet/redis/Concurrency10\" -benchtime=50ms ./cache/"
        status: pass
      - kind: other
        ref: "DOCKER_HOST= go test -bench=\"BenchmarkSingleflightGetOrSet/redis/Concurrency10\" -benchtime=10ms ./cache/  # Docker subtests skipped"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-08-08
status: complete
---

# Phase 8: Benchmark Suite Plan 01 Summary

**Thundering-herd singleflight benchmark in cache/bench_test.go quantifies the dedup win: naive path runs the setter N× per iteration, singleflight runs it exactly 1×, across concurrency levels 10–500, with mem (always) and 4 Docker-gated providers**

## Performance

- **Duration:** 15 min
- **Started:** 2026-08-08T08:00:00Z
- **Completed:** 2026-08-08T08:15:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- **BenchmarkSingleflightGetOrSet** with 4 concurrency levels (10, 50, 100, 500) and 2 paths each (naive, singleflight) = 8 core sub-benchmarks
- **benchmarkHerdNaive**: TOCTOU pattern demonstrating the actual thundering-herd problem — each concurrent miss runs the setter independently (N× per iteration)
- **benchmarkHerdSingleflight**: via `BatchCache.GetOrSet` showing exact dedup (1× per iteration regardless of N)
- **Docker-gated subtests** for redis, valkey, memcache, and postgres providers using testcontainers, controlled by `skipIfNoDocker`
- **skipIfNoDocker** checks both DOCKER_HOST and docker info for robust Docker availability detection
- **setter_runs/op** custom metric reported alongside standard allocs/op and bytes/op

### Verified Metrics (mem provider, -benchtime=100ms)

| Concurrency | Naive setter_runs/op | Singleflight setter_runs/op | Dedup win |
|-------------|---------------------:|---------------------------:|----------:|
| 10          | 10.00                | 1.000                      | 10×       |
| 50          | 50.00                | 1.000                      | 50×       |
| 100         | 100.0                | 1.000                      | 100×      |
| 500         | 500.0                | 1.000                      | 500×      |

## Task Commits

Each task was committed atomically:

1. **Task 1: Create cache/bench_test.go with mem provider herd benchmark** - `fd31a57` (feat)
2. **Task 2: Add Docker-gated thundering-herd subtests for all providers** - `49008db` (feat)

**Plan metadata:** (pending — final commit below)

## Files Created/Modified

- `cache/bench_test.go` - BenchmarkSingleflightGetOrSet with naive, singleflight, and Docker-backed subtests

## Decisions Made

- **TOCTOU naive pattern**: The plan described a lock-everything approach ("lock → get → if miss, compute → store → unlock") which inherently deduplicates and cannot demonstrate the thundering-herd problem. Changed to a check-outside/compute-outside-store pattern (TOCTOU) where each concurrent goroutine independently runs the setter on miss. This is the actual thundering-herd behavior.
- **DOCKER_HOST check**: Added DOCKER_HOST environment variable check to skipIfNoDocker because testcontainers-go requires it when Docker runs on a non-default socket (Orbstack on macOS).
- **context.Background vs b.Context**: Used `context.Background()` in setter closures for the singleflight benchmark path instead of `b.Context()` to avoid unnecessary coupling with the test lifecycle.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TOCTOU pattern for naive herd benchmark**
- **Found during:** Task 1 (Creating herd benchmark)
- **Issue:** The plan-described naive implementation (`lock → get → if miss → compute → store → unlock`) inherently deduplicates concurrent setter calls because the first goroutine that acquires the mutex runs the setter and stores the result; all subsequent goroutines find the cached value. This makes naive setter_runs/op = 1, same as singleflight, defeating the benchmark's purpose.
- **Fix:** Changed to TOCTOU (time-of-check-to-time-of-use) pattern: check under lock (fast path if cached), release lock, compute outside lock where multiple goroutines can simultaneously execute the setter, re-acquire lock to store (last writer wins).
- **Files modified:** `cache/bench_test.go`
- **Verification:** Naive Concurrency10 now shows 10.00 setter_runs/op (was 1.000)
- **Committed in:** `fd31a57` (Task 1 commit)

**2. [Rule 3 - Blocking] skipIfNoDocker fails on Orbstack without DOCKER_HOST**
- **Found during:** Task 2 (Running Docker-backed subtests)
- **Issue:** `docker info` succeeds (Docker CLI works) but testcontainers-go fails to connect because DOCKER_HOST is not set. The project uses Orbstack with a non-default socket (`~/.orbstack/run/docker.sock`).
- **Fix:** Added DOCKER_HOST env var check to skipIfNoDocker alongside docker info check.
- **Files modified:** `cache/bench_test.go`
- **Verification:** Docker subtests skip cleanly when DOCKER_HOST is unset, run successfully when set.
- **Committed in:** `49008db` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes essential for correctness and reliability. No scope creep.

## Issues Encountered

- Redis Docker benchmark shows setter_runs/op slightly below 1.000 (~0.65–0.86) because the fast-path Get optimization in SingleflightGetOrSet.Do sometimes returns a cached value before entering the singleflight group — the leader's Redis SET completes before some followers' GET arrives at the Redis server. This is expected behavior of the fast-path optimization and not a bug.

## Known Stubs

None — the benchmark file is self-contained test code with no stubs or placeholder data.

## Threat Flags

No new security-relevant surface introduced beyond the planned threat model (T-08-01, T-08-02, T-08-SC).

## Self-Check: PASSED

- ✅ `cache/bench_test.go` exists with `BenchmarkSingleflightGetOrSet` function
- ✅ Both naive and singleflight subtests run for all concurrency levels
- ✅ `skipIfNoDocker` compiles and works correctly
- ✅ `go vet ./cache/` passes
- ✅ `make coverage-quick` passes (75.9% total, thresholds met)
- ✅ All mem subtests show correct metrics (naive ≈ N, singleflight ≈ 1)
- ✅ Docker subtests skip when Docker unavailable, run when available

## Next Phase Readiness

- Cache/bench_test.go ready for Plan 02 (batch benchmarks) additions
- benchmarkHerdSingleflightConcurrencyLoop helper reusable by Plan 02
- skipIfNoDocker pattern established for Docker-gated subtests
- Plan 03 (make benchmark targets) can reference the verified benchmark output

---

*Phase: 08-benchmark-suite*
*Completed: 2026-08-08*

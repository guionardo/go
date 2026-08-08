---
phase: 08-benchmark-suite
plan: 03
subsystem: testing
tags: [benchmark, batch, mget, mset, mdel, makefile, make, docker, testcontainers, redis, valkey, memcache, postgres]

requires:
  - phase: 08-01-benchmark-suite
    provides: cache/bench_test.go with skipIfNoDocker and BenchmarkSingleflightGetOrSet
  - phase: 08-02-benchmark-suite
    provides: cache/bench_test.go with BenchmarkBatch (mem provider)

provides:
  - `make benchmark` target with -benchtime=1s -count=5 -benchmem
  - `make benchmark-quick` target for fast dev iteration
  - Docker-gated batch benchmark subtests for redis, valkey, memcache, postgres
  - runBatchBenchmarks helper function for shared MGet/MSet/MDel benchmark structure

affects: []

tech-stack:
  added: []
  patterns:
    - Makefile benchmark targets with DOCKER_HOST passthrough for Docker-backed providers
    - Provider loop pattern with setup functions for Docker-gated subtests
    - runBatchBenchmarks helper for provider-agnostic batch benchmark structure

key-files:
  created: []
  modified:
    - Makefile
    - cache/bench_test.go

key-decisions:
  - "Makefile benchmark targets pass DOCKER_HOST to go test, consistent with test-e2e target (Rule 2 deviation)"
  - "runBatchBenchmarks extracted as a shared helper to avoid duplicating the batch benchmark structure across all 4 Docker providers"
  - "Docker providers run via a setup-functions loop, keeping the benchmark function flat and maintainable"
  - "Batch sizes [1, 10, 100, 1000] used for Docker providers, matching mem provider sizes"

patterns-established:
  - "Makefile benchmark targets with DOCKER_HOST passthrough (needed for testcontainers Docker detection)"
  - "dockerProvider struct pattern for clean provider-permutations loop in benchmarks"
  - "runBatchBenchmarks provider-agnostic helper for shared MGet/MSet/MDel structure"

requirements-completed: [BENCH-03]

coverage:
  - id: D1
    description: "make benchmark target runs with -benchtime=1s -count=5 -benchmem and exits successfully"
    requirement: BENCH-03
    verification:
      - kind: other
        ref: "make benchmark 2>&1 (runs all benchmarks with -count=5)"
        status: pass
    human_judgment: false
  - id: D2
    description: "make benchmark-quick target runs with -benchtime=100ms -count=1 -benchmem and exits successfully"
    requirement: BENCH-03
    verification:
      - kind: other
        ref: "make benchmark-quick 2>&1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Docker-gated batch subtests exist for redis, valkey, memcache, postgres with skipIfNoDocker guard"
    requirement: BENCH-03
    verification:
      - kind: other
        ref: "DOCKER_HOST=unix:///Users/guionardo/.orbstack/run/docker.sock go test -bench=BenchmarkBatch/redis/MGet/Size10/native -benchtime=100ms ./cache/"
        status: pass
      - kind: other
        ref: "DOCKER_HOST= go test -bench=BenchmarkBatch/redis/MGet/Size10/native -benchtime=10ms ./cache/  # Docker subtests skipped"
        status: pass
    human_judgment: false
  - id: D4
    description: "Benchmark targets appear in make help output"
    requirement: BENCH-03
    verification:
      - kind: other
        ref: "make help | grep benchmark"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-08-07
status: complete
---

# Phase 8: Benchmark Suite Plan 03 Summary

**`make benchmark` and `make benchmark-quick` targets added to Makefile alongside existing test/coverage targets, with Docker-gated batch benchmark subtests for redis, valkey, memcache, and postgres providers using the shared runBatchBenchmarks helper**

## Performance

- **Duration:** 8 min
- **Started:** 2026-08-07T22:14:00Z
- **Completed:** 2026-08-07T22:22:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- **`make benchmark`** target runs `go test -bench=. -benchtime=1s -count=5 -benchmem ./cache/...` with DOCKER_HOST passthrough (per D-11)
- **`make benchmark-quick`** target runs `go test -bench=. -benchtime=100ms -count=1 -benchmem ./cache/...` for fast iteration (per D-12)
- **Docker-gated batch subtests** for redis, valkey, memcache, and postgres providers inside BenchmarkBatch, each creating a test container on demand
- **`runBatchBenchmarks` helper** extracts the shared MGet/MSet/MDel benchmark structure so all 4 providers reuse the same measurement code
- **DOCKER_HOST passthrough** added to both benchmark targets (like test-e2e) so Docker-backed benchmarks actually run when Docker is available
- **Pinned container images**: redis:7-alpine, valkey/valkey:8-alpine, memcached:1-alpine, postgres:16-alpine (per T-08-06 mitigation)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add make benchmark and make benchmark-quick targets to Makefile** — `80040be` (feat)
2. **Task 2: Add Docker-gated batch benchmark subtests for all providers** — `ca000a5` (feat)

## Files Created/Modified

- `Makefile` — Added `benchmark` and `benchmark-quick` targets with DOCKER_HOST passthrough; updated .PHONY
- `cache/bench_test.go` — Added `runBatchBenchmarks` helper; added Docker-gated provider subtests (redis, valkey, memcache, postgres) inside BenchmarkBatch

## Decisions Made

- **DOCKER_HOST passthrough**: The Makefile defines DOCKER_HOST at the top level but doesn't export it to subprocesses by default. Added `DOCKER_HOST=$(DOCKER_HOST)` prefix to both benchmark targets (matching test-e2e target convention) so Docker-backed benchmarks actually run when Docker is available.
- **Shared runBatchBenchmarks helper**: Extracted a `runBatchBenchmarks(b, c, name)` function that runs the full MGet/MSet/MDel suite for any provider. All 4 Docker-backed providers call this helper, avoiding 3× duplication of the benchmark structure.
- **Provider loop pattern**: Each Docker provider is defined as a `dockerProvider{name, setup}` struct with a setup closure that creates the test container and provider. The loop calls `b.Run(p.name, ...)` which gates via `skipIfNoDocker` inside the setup function.
- **Batch sizes**: Sizes [1, 10, 100, 1000] used for Docker providers, matching the existing mem provider sizes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] DOCKER_HOST not passed to benchmark targets**
- **Found during:** Task 1 (Verify benchmark target)
- **Issue:** The Makefile benchmark targets ran `go test` without passing DOCKER_HOST. The skipIfNoDocker helper checks `os.Getenv("DOCKER_HOST")` — without it being exported to the subprocess, Docker-backed benchmarks would always be skipped even when Docker is available.
- **Fix:** Added `DOCKER_HOST=$(DOCKER_HOST)` prefix to both `benchmark` and `benchmark-quick` targets, matching the existing `test-e2e` target convention.
- **Files modified:** `Makefile`
- **Verification:** `DOCKER_HOST=unix:///Users/guionardo/.orbstack/run/docker.sock go test -bench="BenchmarkBatch/redis/MGet/Size10/native" -benchtime=100ms ./cache/` returns valid benchmark results (73,845 ns/op).
- **Committed in:** `ca000a5` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Essential for Docker-backed benchmarks to work correctly. No scope creep.

## Issues Encountered

- None — plan executed smoothly with one Rule 2 deviation to ensure Docker-backed benchmarks actually run.

## Next Phase Readiness

- All Phase 8 plans complete — benchmark suite is ready for verification
- `make benchmark` runs all 5 provider benchmarks (mem always, 4 Docker-backed when Docker available)
- `make help` shows both new targets in the Testing section
- `go vet ./cache/` passes
- `make coverage-quick` passes (75.9%)

## Self-Check: PASSED

- ✅ `Makefile` — confirmed present
- ✅ `cache/bench_test.go` — confirmed present
- ✅ `08-03-SUMMARY.md` — confirmed present
- ✅ Commits `80040be`, `ca000a5` — confirmed in git log
- ✅ `make help` shows `benchmark` and `benchmark-quick` targets

- ✅ `Makefile` has `benchmark` and `benchmark-quick` targets with DOCKER_HOST passthrough
- ✅ `cache/bench_test.go` has `runBatchBenchmarks` helper and Docker-gated provider subtests
- ✅ `go vet ./cache/` passes
- ✅ `make coverage-quick` passes (75.9% total, thresholds met)
- ✅ `make benchmark-quick` runs and produces benchmark output
- ✅ `make help` shows both new targets
- ✅ Docker-backed subtests skip cleanly without Docker
- ✅ Docker-backed subtests run successfully with Docker (redis verified: 73,845 ns/op for MGet Size10)
- ✅ All existing targets (`test`, `test-e2e`, `coverage`, `coverage-quick`) still work

---

*Phase: 08-benchmark-suite*
*Completed: 2026-08-07*

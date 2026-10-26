---
phase: 17-delivery-hardening
plan: 02
subsystem: quality
tags: [benchmarks, sqlite, mem, qual-04, cache, batch-cache, testing]

# Dependency graph
requires:
  - phase: 08-benchmark-suite
    provides: the v1.6 harness in `cache/bench_test.go` — `skipIfNoDocker`, `prePopulateKeys`, `runBatchBenchmarks` (MGet/MSet/MDel native vs per-key at sizes 1/10/100/1000), `BenchmarkBatch`
  - phase: 16-batch-surface-multi-process-hardening
    provides: the shipped sqlite batch surface (MGet/MSet/MDel) that these benchmarks measure
  - phase: 15-provider-foundation-core-cache-semantics
    provides: "`cache/sqlite` with `WithMemory`/`WithPath` and a `Close` that checkpoints the WAL"
provides:
  - "`BenchmarkBatch/sqlite/Memory/...` and `BenchmarkBatch/sqlite/File/...` — 48 sub-benchmarks (2 families x MGet/MSet/MDel x sizes 1/10/100/1000 x native|per-key), directly name-comparable with the existing mem arms"
  - "`BenchmarkPrimitives` — 12 sub-benchmarks comparing `mem`, `sqlite/Memory` and `sqlite/File` across `Get`/`Set`/`Delete`/`GetOrSet` over a bounded 1024-key ring"
  - "`primitiveKeyRing` (untyped const 1024) and `runPrimitiveBenchmarks(b, c)` — the reusable primitive-op harness for any future provider"
affects: [17-03 (docs must not publish numbers), gsd-ship (QUAL-04 evidence), verify-work]

# Actuals (#2632) — pairs with the plan's estimate (45000 tokens / 2 tasks, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 898
  tasks: 2
  commits: 2
  plan_head_before: 1c549dd92938c769411be547bdd22687888e17e8
  plan_head_after: cfd09379076b4f6ebb82f9611e29bb0c720e7c3c

# Tech tracking
tech-stack:
  added: []            # no new dependency, go.mod untouched, no new GitHub Action
  patterns:
    - "Measure a new backend by registering it as an always-run family inside the existing benchmark, not by adding a top-level Benchmark function — names sort next to the mem baseline, so the comparison is the default `go test -bench` view"
    - "One cache instance per family, closed via `b.Cleanup` registered AFTER `b.TempDir()` so the close runs first (LIFO) — the Windows temp-removal guarantee"
    - "Bound the key space with a ring (`i % primitiveKeyRing`) inside a timed loop, so a file-backed provider measures the operation rather than b-tree growth"
    - "Distinct `runBatchBenchmarks` name per family — every generated key embeds it, which is what keeps two sqlite families from colliding on pre-populated keys"

key-files:
  created: []
  modified:
    - cache/bench_test.go

key-decisions:
  - "D-05 followed: sqlite reuses `runBatchBenchmarks`/`prePopulateKeys` unchanged, and the primitive-ops benchmark adds the missing surface (`Delete` included on top of the D-05 Get/Set/GetOrSet trio) against both sqlite modes"
  - "D-06 followed literally as two nested `b.Run` families under one `b.Run(\"sqlite\")`: `Memory` (`sqlite.WithMemory()`) and `File` (`filepath.Join(b.TempDir(), \"batch.db\")` computed inside the closure, so `-count=5` yields five fresh databases)"
  - "D-07 followed: both families are always-run — no `skipIfNoDocker`, no container, no `cache/sqlite/bench_test.go`; the unexported helpers stay in `package cache_test` and `./cache/...` discovers the new families with no Makefile or CI change"
  - "D-08 honoured: no ns/op, allocs or ratio appears in this summary, the plan, or the README — QUAL-04's evidence is existence, naming and shape"
  - "`BenchmarkPrimitives` registers providers as `mem` / `sqlite/Memory` / `sqlite/File` (fresh caches, `b.Cleanup(Close)`) so a single `go test -bench=BenchmarkPrimitives` run yields the full 3x4 comparison matrix"

patterns-established:
  - "Nested provider families: group second-tier variants (`sqlite/Memory`) under a provider family (`sqlite`) when a backend has more than one meaningful mode"
  - "Ring-bounded benchmark loops: any write-heavy benchmark on a durable store iterates a fixed key ring, never a unique key per iteration"

requirements-completed: [QUAL-04]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "sqlite is registered in `BenchmarkBatch` in two always-run, Docker-free families (`Memory` on `:memory:`, `File` on `b.TempDir()`), each delegating to `runBatchBenchmarks` with a distinct name, producing 48 native-vs-per-key sub-benchmarks at sizes 1/10/100/1000 while the mem arms still run"
    requirement: "QUAL-04"
    verification:
      - kind: unit
        ref: "go test ./cache/ -run '^$' -bench 'BenchmarkBatch/sqlite' -benchtime=1x -count=1 -> 48 `BenchmarkBatch/sqlite/...` lines (24 arms x 2 families)"
        status: pass
      - kind: unit
        ref: "name assertions: `BenchmarkBatch/sqlite/Memory/MGet/Size1000/native`, `.../per-key`, `BenchmarkBatch/sqlite/File/MSet/Size10/native`, `BenchmarkBatch/sqlite/File/MDel/Size100/per-key` all present"
        status: pass
      - kind: unit
        ref: "go test ./cache/ -run '^$' -bench 'BenchmarkBatch/MGet' -> `BenchmarkBatch/MGet/Size1000/native` still runs (mem baseline untouched)"
        status: pass
      - kind: other
        ref: "golangci-lint run ./cache/... -> `0 issues.`"
        status: pass
      - kind: other
        ref: "make coverage-quick -> File 70% / Package 80% / Total 75% all PASS (82.1%, 2629/3201)"
        status: pass
    human_judgment: false
  - id: D2
    description: "`BenchmarkPrimitives` measures `Get`/`Set`/`Delete`/`GetOrSet` for `mem`, `sqlite/Memory` and `sqlite/File` over the bounded `primitiveKeyRing` key space, and the whole `./cache/...` quick suite runs green"
    requirement: "QUAL-04"
    verification:
      - kind: unit
        ref: "go test ./cache/ -run '^$' -bench 'BenchmarkPrimitives' -benchtime=1x -count=1 -> 12 `BenchmarkPrimitives/...` lines (3 providers x 4 ops)"
        status: pass
      - kind: unit
        ref: "name assertions: `BenchmarkPrimitives/mem/Get`, `.../sqlite/Memory/Set`, `.../sqlite/File/GetOrSet`, `.../sqlite/File/Delete` all present"
        status: pass
      - kind: other
        ref: "golangci-lint run ./cache/... -> `0 issues.`; make benchmark-quick -> exit 0, 7 PASS, 0 FAIL"
        status: pass
      - kind: other
        ref: "make coverage-quick -> all three thresholds PASS (82.1%, 2628/3201) — the new benchmark file did not lower coverage"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-10-26
status: complete
---

# Phase 17 Plan 02: SQLite Benchmark Coverage Summary

**sqlite now runs in the v1.6 benchmark suite twice — `BenchmarkBatch/sqlite/Memory` and `/File`, 48 native-vs-per-key arms directly name-comparable with mem — plus a new `BenchmarkPrimitives` that measures Get/Set/Delete/GetOrSet for mem against both sqlite modes over a bounded key ring.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-10-26T22:44Z
- **Completed:** 2026-10-26T22:50Z
- **Tasks:** 2
- **Files modified:** 1 (`cache/bench_test.go`, +86 / -0 — additive only)

## Accomplishments

- `BenchmarkBatch` gained a `b.Run("sqlite", ...)` section with two nested families, placed immediately before the Docker-backed section: `Memory` (`sqlite.WithMemory()`) and `File` (`sqlite.WithPath(filepath.Join(b.TempDir(), "batch.db"))`, the path computed **inside** the closure so `-count=5` gets five fresh databases). Each delegates to the untouched `runBatchBenchmarks` with its own name (`sqlite_memory`, `sqlite_file`), which is what keeps their pre-populated keys from colliding.
- 48 new sub-benchmarks measured at `-benchtime=1x`: 2 families x MGet/MSet/MDel x sizes 1/10/100/1000 x native|per-key. The mem arms (`BenchmarkBatch/MGet/Size1000/native`, ...) still run unchanged, so the two families land next to the baseline in default bench output.
- New `BenchmarkPrimitives` with 12 sub-benchmarks: `mem`, `sqlite/Memory`, `sqlite/File` x `Get`/`Set`/`Delete`/`GetOrSet`. `GetOrSet` measures the warm fast-path (deliberate, per D-05) and `Delete` was included on top of the D-05 trio, which the CONTEXT discretion item left open.
- `primitiveKeyRing = 1024` bounds every key the primitive arms mint, so file-mode `Set` measures overwrite cost rather than unbounded b-tree growth — the T-17-07 mitigation, and the reason `make benchmark-quick` stays in its ~20 s envelope.
- Cleanup ordering is load-bearing and correct in all four sqlite constructions: `b.TempDir()` first, cache second, `b.Cleanup(func() { _ = c.Close() })` last with a blank line above it — LIFO means the handle closes before temp removal, the Windows-safe order (T-17-06, PITFALLS Pitfall 6).
- Zero collateral change: `prePopulateKeys`/`runBatchBenchmarks` signatures, the mem blocks, `batchSizes`, the Docker `providers` slice and its loop, the Makefile, the CI workflow and `go.mod` are all byte-identical. Both families are always-run — no `skipIfNoDocker`, no container.

## Task Commits

Each task was committed atomically:

1. **Task 1: sqlite batch families — Memory and File registered as always-run providers** - `53b3ee5` (feat)
2. **Task 2: Primitive-ops benchmark — mem vs both sqlite modes across Get/Set/Delete/GetOrSet** - `cfd0937` (feat)

**Plan metadata:** this SUMMARY (`docs: complete plan`) — committed separately.

## Files Created/Modified

- `cache/bench_test.go` — new imports (`path/filepath`, `github.com/guionardo/go/cache/sqlite`); `BenchmarkBatch` sqlite `Memory`/`File` families ahead of the Docker section; package-level `const primitiveKeyRing = 1024` above the first func (`decorder`); `runPrimitiveBenchmarks` and `BenchmarkPrimitives` appended after `BenchmarkBatch`.

## Decisions Made

Followed D-05..D-08 without deviation. The two discretion items the plan left open were resolved as follows:

- **`Delete` joined the primitive arms.** D-05 names Get/Set/GetOrSet and the CONTEXT discretion allowed more; `Delete` is the fourth operation every provider implements, costs one extra line per iteration, and makes the matrix a complete 3x4. It is measured the only way that is comparable across providers — set then delete in the same iteration.
- **Doc comments on both new functions.** The file's convention is a leading `// Name ...` sentence on every helper (`prePopulateKeys`, `runBatchBenchmarks`), so the new const, helper and benchmark each carry one; this also keeps `godoclint`/`revive` quiet on the new lines.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Both counts (48 and 12), the four name assertions, `golangci-lint` (0 issues on the first run of each task) and `make coverage-quick` passed without a fix-up cycle.

`make benchmark-quick` printed the expected `redis: ... connection refused` lines: the Docker-gated providers self-skip via `skipIfNoDocker`, and the target still exits 0 with 7 `PASS` package lines and zero `FAIL`. That is pre-existing behaviour, not a regression from this plan.

## Out-of-Scope Follow-ups (recorded, deliberately NOT fixed here)

- **Probe-fallback assumption (QUAL-04, unclassified — review manually):** "following the v1.6 benchmark suite conventions" is satisfied structurally — helper reuse, the always-run vs `skipIfNoDocker` split, and `b.N` loops — and is verified by benchmark existence, naming and shape, never by a ratio (D-08). A reviewer who wants a numeric regression gate would need a separate requirement; no threshold is asserted anywhere and none should be inferred from this summary.
- **Full `make benchmark` stays ~5 min** (RESEARCH Pitfall 8) because the Docker-backed providers actually spin containers; `benchmark-quick` remains the fast gate this plan validated.

## Next Phase Readiness

- QUAL-04's first half is shipped: sqlite is measurable against mem and against per-key loops at the same four batch sizes, in both modes.
- 17-03 must not publish any number from these benchmarks (D-08) — the README benchmark table stays mem/Docker-only.
- `make coverage-quick` stays green on the committed tree: file 70% / package 80% / total 75% all PASS at 82.1%, unchanged by the additive benchmark file.

---

## Self-Check: PASSED

- FOUND: `cache/bench_test.go`
- FOUND: `.planning/phases/17-delivery-hardening/17-02-SUMMARY.md`
- FOUND: commit `53b3ee5` (Task 1)
- FOUND: commit `cfd0937` (Task 2)

*Phase: 17-delivery-hardening*
*Completed: 2026-10-26*

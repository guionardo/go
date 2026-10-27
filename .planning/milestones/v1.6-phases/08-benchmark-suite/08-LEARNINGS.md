---
phase: 08
phase_name: "Benchmark suite"
project: "github.com/guionardo/dev/go"
generated: "2026-10-08"
counts:
  decisions: 8
  lessons: 6
  patterns: 7
  surprises: 5
missing_artifacts:
  - "08-UAT.md"
---

# Phase 08 Learnings: Benchmark suite

## Decisions

### Naive herd path uses TOCTOU — not lock-everything
The plan-specified naive implementation (lock → get → compute → store under one mutex) was replaced with a check-outside/compute-outside-store (TOCTOU) pattern in `benchmarkHerdNaive`: each concurrent goroutine independently runs the setter on miss and re-acquires the lock only to store (last writer wins).

**Rationale:** The lock-everything version inherently deduplicates — the first mutex holder stores the value and all followers hit the cache — making naive setter_runs/op = 1 and defeating the benchmark's purpose. After the change naive Concurrency10 reports 10.00 setter_runs/op.
**Source:** 08-01-SUMMARY.md

### skipIfNoDocker checks DOCKER_HOST AND docker info
`skipIfNoDocker` verifies both `exec.Command("docker", "info").Run()` and the `DOCKER_HOST` environment variable before declaring Docker usable.

**Rationale:** With Orbstack on macOS the CLI works over a non-default socket (`~/.orbstack/run/docker.sock`), but testcontainers-go fails to connect unless DOCKER_HOST is set — docker info alone would let subtests start and fail instead of skipping.
**Source:** 08-01-SUMMARY.md

### context.Background() in benchmark setter closures, not b.Context()
Singleflight benchmark setters use `context.Background()` rather than `b.Context()`.

**Rationale:** Avoids unnecessary coupling of the benchmark setter work with the test lifecycle.
**Source:** 08-01-SUMMARY.md

### Read benchmarks pre-populate once; write/delete benchmarks use fresh keys per iteration
`BenchmarkBatch/MGet` pre-populates keys once via `prePopulateKeys` and reuses them across b.N iterations; MSet and MDel generate unique keys per iteration.

**Rationale:** Reads are idempotent, so reuse keeps setup out of the measurement; write/delete operations would otherwise see stale state (cached values, already-deleted keys) across b.N cycles.
**Source:** 08-02-SUMMARY.md

### Provider-agnostic runBatchBenchmarks helper + dockerProvider setup loop
The MGet/MSet/MDel structure was extracted into `runBatchBenchmarks(b, c, name)`, and each Docker provider is a `dockerProvider{name, setup}` entry whose setup closure creates the container/provider before calling the shared helper; sizes [1, 10, 100, 1000] match the mem provider.

**Rationale:** Avoids triplicating the benchmark structure for redis, valkey, memcache, and postgres, keeping `BenchmarkBatch` flat and maintainable.
**Source:** 08-03-SUMMARY.md

### Makefile benchmark targets pass DOCKER_HOST through
Both `benchmark` and `benchmark-quick` invoke go test with a `DOCKER_HOST=$(DOCKER_HOST)` prefix, matching the existing `test-e2e` convention.

**Rationale:** A Makefile variable is not exported to subprocess environments by default, so without the prefix `skipIfNoDocker` would see an empty DOCKER_HOST and silently skip every Docker-backed benchmark even when Docker is available.
**Source:** 08-03-SUMMARY.md

### Single bench_test.go with mem as primary and Docker-gated providers
All benchmarks live in `cache/bench_test.go` (D-01); mem is the always-run primary provider for both herd and batch domains (D-03/D-07) while redis, valkey, memcache, and postgres require Docker (D-08/D-09).

**Rationale:** Keeps the suite zero-dependency and runnable everywhere while still covering all 5 providers locally; Docker availability never gates the default target.
**Source:** 08-CONTEXT.md

### Two Makefile targets: full statistical run vs quick dev run
`make benchmark` runs `-bench=. -benchtime=1s -count=5 -benchmem ./cache/...`; `make benchmark-quick` runs `-benchtime=100ms -count=1`.

**Rationale:** D-11/D-12 — `-count=5` gives statistically meaningful full-run results, while the quick variant enables fast iteration during development.
**Source:** 08-CONTEXT.md

---

## Lessons

### A lock-everything "naive" path cannot exhibit thundering herd
The plan's naive implementation produced setter_runs/op = 1.000 — identical to singleflight — because the mutex serialized goroutines and the first one filled the cache for everyone.

**Context:** Found while building Task 1 of Plan 01; fixed under Rule 1 (verification: naive Concurrency10 went from 1.000 to 10.00). A comparison harness must be checked to actually exhibit the phenomenon it is measuring.
**Source:** 08-01-SUMMARY.md

### `docker info` success does not mean testcontainers can connect
Docker CLI checks pass while testcontainers-go fails on Orbstack because the daemon listens on a non-default socket.

**Context:** Found during Plan 01 Task 2 — the skip guard had to add a DOCKER_HOST env check; the same gap then required the Makefile passthrough in Plan 03.
**Source:** 08-01-SUMMARY.md

### For the mem provider, native MSet can lose on ns/op while winning on allocs/op
Native MSet iterates the items map inside the write lock, so per-key MSet appears slightly faster in ns/op; only allocations (resolveTTL once — ~1,000 fewer allocs/op at Size1000) consistently favor native.

**Context:** Documented in 08-02 issues — a mem-provider artifact; for network-backed providers round-trip time dominates and single-lock batching would show a dramatic win.
**Source:** 08-02-SUMMARY.md

### Prep overhead dominates MDel at Size1000
Creating 1000 unique keys per iteration before each MDel makes the native vs per-key difference vanish at Size1000 even though Size100 shows a ~49% native win.

**Context:** The benchmark measures the full operation including prep, which costs the same regardless of batching — the win is only visible where prep does not dominate.
**Source:** 08-02-SUMMARY.md

### Redis setter_runs/op lands below the ideal 1.000
Docker-backed redis herd benchmarks show ~0.65–0.86 setter_runs/op because the fast-path Get in `SingleflightGetOrSet.Do` sometimes returns a cached value before entering the group — the leader's SET reaches Redis before some followers' GET arrives.

**Context:** Expected behavior of the fast-path optimization, not a bug; mem shows the exact 1.000 dedup.
**Source:** 08-01-SUMMARY.md

### Makefile variables are not visible to subprocess environments
Defining DOCKER_HOST in the Makefile did not export it; the benchmark targets would always skip Docker subtests until `DOCKER_HOST=$(DOCKER_HOST)` was prepended.

**Context:** Rule 2 deviation in Plan 03 caught while verifying the target — match the `test-e2e` convention whenever a guard reads an env var.
**Source:** 08-03-SUMMARY.md

---

## Patterns

### b.Run sub-benchmark hierarchy for scale points
`BenchmarkSingleflightGetOrSet/Concurrency{N}/{naive,singleflight}` and `BenchmarkBatch/{MGet,MSet,MDel}/Size{N}/{native,per-key}`.

**When to use:** Quantifying how a win scales across contention or batch-size levels — one level set per benchmark, both comparison paths under each.
**Source:** 08-01-SUMMARY.md

### Custom metric via b.ReportMetric + atomic.Int64
`b.ReportMetric(float64(setterRuns.Load())/float64(b.N), "setter_runs/op")` reports setter invocations per op alongside the standard metrics.

**When to use:** Dedup/caching benchmarks where the key signal is an invocation count, not just time — appears in standard benchmem output.
**Source:** 08-01-SUMMARY.md

### Unique key per b.N iteration prevents cross-contamination
Keys embed the iteration index (e.g. `herd_sf_{n}_{i}`), so every iteration starts from a cold state even as b.N grows.

**When to use:** GetOrSet or write benchmarks where a cached key from a prior b.N cycle would short-circuit the measured path; Plan 01 pins it and Plan 02 reuses it for MSet/MDel.
**Source:** 08-01-SUMMARY.md

### Docker-gated benchmark via skipIfNoDocker + per-subtest container
Each provider sub-benchmark calls `skipIfNoDocker(b, reason)` first, then starts its testcontainer and creates the provider with `defer c.Close()`.

**When to use:** Provider-parameterized benchmarks that must stay green without Docker — mem always runs, the rest skip cleanly.
**Source:** 08-01-SUMMARY.md

### Read-prepopulate vs write-unique-keys state setup
Read benchmarks pre-populate once then `b.ResetTimer()` to exclude setup; write/delete benchmarks generate fresh keys inside the b.N loop.

**When to use:** Batch benchmark comparisons — keeps setup out of read measurements while preventing stale state on mutation paths.
**Source:** 08-02-SUMMARY.md

### Result map accumulation to prevent DCE
The per-key MGet fallback stores results into a `map[string]string` instead of discarding them.

**When to use:** Benchmarks whose fallback path returns values the compiler could eliminate as dead code when unused; it also mirrors MGet's behavior.
**Source:** 08-02-SUMMARY.md

### dockerProvider{name, setup} loop + runBatchBenchmarks helper
A slice of `dockerProvider` structs pairs a name with a setup closure that starts the container and returns a `cache.BatchCache`; the loop dispatches `b.Run(p.name, ...)` into the shared `runBatchBenchmarks` helper.

**When to use:** The same benchmark suite must run over multiple providers — one structure, N providers, no duplication.
**Source:** 08-03-SUMMARY.md

---

## Surprises

### The plan's naive benchmark would have proven nothing
The lock-everything naive path produced setter_runs/op = 1.000 — the exact singleflight result — silently making the comparison meaningless.

**Impact:** Forced the TOCTOU rewrite (setter computed outside the lock); naive now reports 10/50/100/500 across the four concurrency levels, restoring the demonstrated N× vs 1× contrast.
**Source:** 08-01-SUMMARY.md

### Docker works but testcontainers does not (Orbstack)
`docker info` succeeds through the CLI while testcontainers-go cannot connect because Orbstack uses a non-default socket.

**Impact:** Drove the DOCKER_HOST check inside skipIfNoDocker and, later, the `DOCKER_HOST=$(DOCKER_HOST)` prefix on both Makefile benchmark targets — without either, Docker-backed benchmarks silently never run.
**Source:** 08-01-SUMMARY.md

### Redis dedup is not exactly 1.0
Redis shows setter_runs/op ~0.65–0.86 instead of 1.000 because the fast-path Get can return a cached value before a follower enters the singleflight group.

**Impact:** Documented as expected fast-path behavior rather than a defect; only the mem provider shows the exact 1.000 dedup the success criteria expected.
**Source:** 08-01-SUMMARY.md

### Native MSet is slower than per-key on mem
The batching win inverts for mem MSet ns/op: iterating the map inside the write lock makes native appear slower, while allocs/op (resolveTTL once) still favors native.

**Impact:** The "native outperforms per-key" success criterion holds only via allocations for MSet; the ns/op win is expected to appear on network-backed providers instead.
**Source:** 08-02-SUMMARY.md

### Memcache batch benchmarks time out under benchmark load
During verification, Docker-backed memcache MSet/MDel at Size100/1000 hit I/O timeouts under heavy benchmark load.

**Impact:** Classified as environmental container networking, not a code bug — the guard works and the zero-dependency mem benchmarks (primary per D-07/D-09/D-10) pass; verification still scored 3/3.
**Source:** 08-VERIFICATION.md

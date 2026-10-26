# Phase 17: Delivery Hardening - Context

**Gathered:** 2026-10-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Make the functionally complete `cache/sqlite` provider (Phases 15–16) **shippable per repo standards**, with no change to provider behavior:

- **QUAL-01** — a visible, enforced CGO-free proof on the existing three-OS CI matrix.
- **QUAL-02** — evidence that the five named test scenario classes are covered and that coverage gates pass with no `.testcoverage-quick.yml` override.
- **QUAL-03** — `doc.go` audited and extended, a runnable example that demonstrates durability, and the README index/paragraph verified.
- **QUAL-04** — sqlite registered in the v1.6 benchmark suite against mem and per-key loops.

In scope: `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/sqlite/doc.go`, `cache/sqlite/example_test.go`, `README.md`, and a verification record of the coverage audit.

Out of scope: any provider logic, schema, DSN, or option change; new tests beyond filling a genuinely missing named class; coverage-config edits; Windows `-race` broadening; benchmark numbers published in docs; README restructuring (when-to-use tables, new sections).

Most of the doc surface already exists from Phase 15 Plan 15-03 — this phase audits and extends rather than creates.

</domain>

<decisions>
## Implementation Decisions

### CGO-Free CI Proof (QUAL-01)
- **D-01:** `CGO_ENABLED=0` is set as **job-level `env`** on the existing three-OS `test` job, so the build, the unit tests, AND the two-process E2E all run CGO-free — the E2E is the milestone's headline promise, so proving it under `CGO_ENABLED=0` is the strongest available evidence.
- **D-02:** The existing `SQLite race detector` step sets **`CGO_ENABLED=1` in its own step env**. `-race` requires cgo and a C toolchain (which the ubuntu/macos runners provide); it cannot run under the job-level `CGO_ENABLED=0`. The Windows gate on that step is untouched.
- **D-03:** CI evidence is an explicit step named **`Build (CGO-free)`** that runs `go env CGO_ENABLED` (so the `0` is visible in the log) followed by `go build ./...`, placed before the existing `go test -v ./...`. The job env is the enforcement; the build step is the visible proof. Matches the `CGO_ENABLED=0 go build ./...` mode recorded in milestone research.
- **D-04:** The Windows `-race` broadening stays **deferred** — it is a Phase 16 reviewer-flagged follow-up with no requirement behind it, and QUAL-01 asks only for CGO-free build+test on three OSes. — **Reversibility:** reversible.

### Benchmarks (QUAL-04)
- **D-05:** Reuse the v1.6 batch helpers (`runBatchBenchmarks`, `prePopulateKeys`; native vs per-key at sizes 1/10/100/1000) for sqlite, **plus** a new primitive-ops benchmark comparing **mem and sqlite** across `Get`/`Set`/`GetOrSet`. No singleflight herd arm for sqlite: it measures the shared dedup helper, not the backend.
- **D-06:** sqlite is measured in **two named sub-benchmark families** — `Memory` (`:memory:`) and `File` (`WithPath` on `b.TempDir()`). Memory keeps the mem comparison apples-to-apples; file measures the durable path users actually ship.
- **D-07:** Benchmark code lives in **`cache/bench_test.go`** (`package cache_test`) and registers sqlite as an **always-run provider like mem — no Docker gate**. Placement follows from helper reuse: `runBatchBenchmarks` and `prePopulateKeys` are unexported in `cache_test`, so a `cache/sqlite/bench_test.go` file would duplicate or force export. `./cache/...` in `make benchmark`/`benchmark-quick` picks it up automatically; no CI benchmark step is added.
- **D-08:** Ship **benchmark entries only** — no measured numbers in the README or in phase artifacts, so nothing goes stale when a dependency or machine changes. — **Reversibility:** reversible.

### Docs (QUAL-03)
- **D-09:** `cache/sqlite/doc.go` keeps its current structure and gains one **batch paragraph**: `MGet` silently skips missing, expired, and undecodable entries (already partly stated via the canceled-MGet sentence), `MSet` writes one transaction with one TTL and rolls back the whole batch on any error, `MDel` is chunked and idempotent. The four QUAL-03 topics (TTL, reopen persistence, `:memory:`, multi-process constraints) are **re-verified against shipped code** as part of the same pass.
- **D-10:** A **second runnable example** is added to `cache/sqlite/example_test.go` demonstrating durability: `WithPath` on `t.TempDir()` → `Set` → `Close` → reopen → `Get`. The existing in-memory `Example()` stays as the entry-point example.
- **D-11:** README changes are **verification + one clause**: confirm the package-index row, provider-table row, and sqlite paragraph match shipped behavior, and add a single batch-surface clause mirroring doc.go. No new sections, no duplicated examples, no performance numbers.

### Test Completeness (QUAL-02)
- **D-12:** QUAL-02 closes by **audit, not churn**: map each named class — unit, `:memory:`, reopen/persistence, concurrent, sweep — to the tests that already cover it, run `make coverage-quick`, and record the mapping as evidence. A test is added only if a named class genuinely has none. No coverage-margin hardening.
- **D-13:** **Coverage gate inputs stay unchanged.** `make coverage`'s E2E merge remains `./cache/`-root-only, so the `e2e`-tagged sqlite two-process tests never enter the profile. Rationale to be recorded so a later agent does not "fix" it: the unit gate already passes at ~92.5% with no override, and E2E correctness is proven by its dedicated CI step.
- **D-14:** **No new guard machinery** for the "no override" promise. A passing gate is the evidence, and any future `cache/sqlite` override appears as a reviewable config diff.

### the agent's Discretion
- Exact CI step name/ordering beyond D-03, and whether `go vet` joins the build step.
- Whether `runBatchBenchmarks` needs a signature change (label/cleanup hook) to host a second provider cleanly; how the file-mode cache is closed (`b.Cleanup`) and whether file-mode uses a fresh temp path per sub-benchmark.
- Which primitive operations the comparison benchmark includes beyond `Get`/`Set`/`GetOrSet` (e.g. `Delete`), and the exact sub-benchmark naming.
- Wording of the doc.go batch paragraph and README clause; the new example's name (`ExampleNew_withPath` vs package-level `Example_newWithPath`) and whether it asserts `errors.Is(err, cache.ErrMiss)`.
- The format of the five-class coverage mapping evidence in the plan/SUMMARY.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase & requirements
- `.planning/ROADMAP.md` — § `### Phase 17: Delivery Hardening`: goal, dependencies, the four success criteria, execution order
- `.planning/REQUIREMENTS.md` — QUAL-01..04 (the only open v1 requirements) + the Out-of-Scope table; do not re-open closed requirements
- `.planning/PROJECT.md` — constraints (Go 1.26 only, minimal deps, three-OS CI, quality gates) and the v1.8 Current State paragraph

### Prior phase decisions (carried forward, do not re-litigate)
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-CONTEXT.md` — D-01..D-12: `:memory:` selectability, location precedence, pool pinning, pragma read-back, filter-only reads, never-failing `New`, sentinels; also assigned "doc.go structure, runnable example shape, README index row wording" to agent discretion
- `.planning/phases/16-batch-surface-multi-process-hardening/16-CONTEXT.md` — D-01..D-11: batch semantics (chunk 100, skip-on-error MGet, atomic MSet, idempotent MDel), bounded busy-only open retry, `journal_size_limit` 64 MB, 3-OS two-process E2E, `-race` gate rationale

### Milestone research
- `.planning/research/STACK.md` § "CGO / CI Implications" — why `modernc.org/sqlite` is the CGO-free pick, the `CGO_ENABLED=0 go build ./...` verification mode, cross-compile and race notes; also the v1.60.1 / libc v1.77.1 pin rationale
- `.planning/research/PITFALLS.md` — Windows handle/sidecar pitfalls relevant to file-mode benchmarks and CI

### Code under change
- `.github/workflows/go.yml` — the three-OS `test` job (job-level env, build step, E2E step, race step) and the separate `coverage` job
- `cache/bench_test.go` — `package cache_test`; `skipIfNoDocker`, `prePopulateKeys`, `runBatchBenchmarks` (MGet/MSet/MDel native vs per-key, sizes 1/10/100/1000), `BenchmarkBatch`, `BenchmarkSingleflightGetOrSet` (the conventions to follow)
- `cache/sqlite/doc.go` — current package documentation to audit and extend
- `cache/sqlite/example_test.go` — the existing runnable in-memory `Example()`
- `README.md` — package index table row, provider table row, `### Package cache` sqlite paragraph
- `Makefile` — `benchmark` / `benchmark-quick` (`go test -bench=. ./cache/...`), `coverage` (E2E merge from `./cache/` only), `coverage-quick` (unit-only + `.testcoverage-quick.yml`)
- `.testcoverage.yml` and `.testcoverage-quick.yml` — thresholds (file 70 / package 80 / total 75) and the existing override list; **`cache/sqlite` must not appear in either**
- `cache/sqlite/twoprocess_e2e_test.go` — `//go:build e2e`; the excluded-from-coverage lane referenced by D-13

### Conventions
- `.opencode/skills/spike-findings-go/SKILL.md` — benchmark invocation conventions (`-benchmem`, benchtime) and cache-integration constraints
- `.planning/codebase/TESTING.md` — testify/`t.Parallel`/`t.TempDir` patterns and coverage tooling (`go-test-coverage` v2)
- `.planning/codebase/CONVENTIONS.md` — doc comment rules (`godoclint`, `// Package <name> provides ...`), 120-char lines, decorder ordering

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cache/bench_test.go` — the whole v1.6 benchmark harness is reusable as-is: `runBatchBenchmarks(b, c, name)` already loops `batchSizes := []int{1, 10, 100, 1000}` and runs `native` vs `per-key`; adding a provider is a `b.Run("sqlite", ...)` block like the existing `mem` one.
- `cache/sqlite/doc.go` — 56 lines of already-correct package docs; the batch paragraph is an insertion, not a rewrite.
- `cache/sqlite/example_test.go` — the existing `Example()` establishes the house style (early-return on error, `// Output:` assertion) for the new durability example.
- `cache/sqlite` tests — every QUAL-02 class already has a named test: `TestMemoryRoundTrip`/`TestMemoryIsolation`, `TestReopenPersistence`/`TestExpiryAcrossRestart`, `TestConcurrentBatchOpsRace`, `TestSweepOnOpen`/`TestPeriodicSweep`/`TestSweeperDisabled`/`TestSweepFailureSwallowed`, plus the `e2e`-tagged `TestTwoProcessContention`/`TestTwoProcessCrashRecovery`.

### Established Patterns
- Benchmarks are `package cache_test`, Docker-backed providers gated by `skipIfNoDocker`, mem always-run — sqlite joins as the second always-run provider.
- `make coverage-quick` before every commit (repo rule); `make quality-report` before tagging.
- CI coverage uses `vladopajic/go-test-coverage@v2` with `./.testcoverage.yml`; the local quick gate uses `.testcoverage-quick.yml`.
- Package docs lead with `// Package <name> provides ...` and carry a positioning line; godoclint enforces the prefix.

### Integration Points
- `.github/workflows/go.yml` `test` job — job-level `env`, one new build step, one step-level env override on the race step.
- `cache/bench_test.go` — new sqlite provider block + a new primitive-ops benchmark function; no changes to existing mem/Docker blocks.
- `cache/sqlite/doc.go` / `example_test.go` / `README.md` — additive edits only.
- `go.mod` already pins `modernc.org/sqlite v1.60.1` and `modernc.org/libc v1.77.1` — QUAL-01's pin half is already satisfied and needs verification, not work.

</code_context>

<specifics>
## Specific Ideas

- The CGO-free proof is most valuable where it is hardest: the two-process E2E running under `CGO_ENABLED=0` on Windows — that combination is the real claim (no C toolchain, no dynamic SQLite library).
- D-13's rationale must be written down explicitly, because widening `./cache/` to `./cache/...` looks like an obvious improvement to any agent who has not seen why it was declined.
- Declined-and-recorded alternatives (do not reopen): Windows `-race` via mingw; a dedicated per-OS cgo-free job; publishing benchmark numbers; a when-to-use backend table; a coverage-margin hardening pass; an explicit no-override guard test.

</specifics>

<deferred>
## Deferred Ideas

- **Windows `-race` broadening** — needs a mingw toolchain setup step and has no requirement behind it. Stays a documented follow-up in the workflow comment (Phase 16 reviewer-flagged).

### Reviewed Todos (not folded)
None — `todo.match-phase 17` returned no matches.

</deferred>

---

*Phase: 17-Delivery Hardening*
*Context gathered: 2026-10-26*

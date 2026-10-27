# Phase 17: Delivery Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-26
**Phase:** 17-Delivery Hardening
**Areas discussed:** CGO-free CI proof, Benchmark scope & placement, Docs delta, Test-completeness bar

---

## CGO-free CI proof

| Option | Description | Selected |
|--------|-------------|----------|
| Job-level env + race override | `CGO_ENABLED=0` at job level so build, tests and the two-process E2E all run CGO-free; race step sets `CGO_ENABLED=1` | ✓ |
| Dedicated cgo-free job per OS | New matrix job for build+test with CGO_ENABLED=0, test job untouched | |
| Ubuntu-only cgo-free job | Cheapest, proves only one of the three OSes | |
| Per-step env, leave job default | Env on individual steps only; guarantee implicit | |

**User's choice:** Job-level env + race override
**Notes:** One job keeps the matrix runtime unchanged while proving the headline promise (two-process E2E) under the exact build tag QUAL-01 names.

### Follow-up: Windows `-race`

| Option | Description | Selected |
|--------|-------------|----------|
| Stay deferred | QUAL-01 asks for CGO-free build+test on three OSes; race needs cgo, no requirement behind it | ✓ |
| Attempt mingw + enable race on Windows | New CI machinery and slower Windows job for a guarantee not required | |
| Keep gate, sharpen the comment | Rewrite the workflow comment to state why, with a date | |

**User's choice:** Stay deferred

### Follow-up: evidence shape

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit build step + env echo | Named `Build (CGO-free)` step: `go env CGO_ENABLED` then `go build ./...` | ✓ |
| Tests only | Job env is the proof; tests compile the package anyway | |
| Build + runtime link assertion | ldd/otool/objdump grep for libsqlite3 per OS | |

**User's choice:** Explicit build step + env echo
**Notes:** Matches QUAL-01's literal "builds and tests" wording and the `CGO_ENABLED=0 go build ./...` mode recorded in milestone research.

---

## Benchmark scope & placement

| Option | Description | Selected |
|--------|-------------|----------|
| Batch suite + primitive vs mem | Reuse `runBatchBenchmarks` + new primitive Get/Set/GetOrSet comparison mem vs sqlite | ✓ |
| Full v1.6 parity (batch + herd) | Also run the singleflight herd benchmark for sqlite | |
| Batch suite only | Native vs per-key at four sizes, no per-op comparison | |
| Primitive ops only | No batch story | |

**User's choice:** Batch suite + primitive vs mem
**Notes:** The herd arm measures the shared singleflight helper more than the backend, so it was dropped.

| Option | Description | Selected |
|--------|-------------|----------|
| Both, as named sub-benchmarks | `Memory` (`:memory:`) and `File` (`WithPath` on `b.TempDir()`) | ✓ |
| `:memory:` only | Cleanest mem comparison, never measures the durable path | |
| File-backed only | Real path, misleading mem comparison and machine variance | |

**User's choice:** Both, as named sub-benchmarks
**Notes:** Benchmarks run only via `make benchmark`, so two arms cost nothing in CI.

| Option | Description | Selected |
|--------|-------------|----------|
| In cache/bench_test.go, always-run | `package cache_test`, sqlite like mem with no Docker gate, helpers reused | ✓ |
| New cache/sqlite/bench_test.go | Provider-local, requires exporting or duplicating helpers | |
| Split: comparisons shared, extras local | Comparison plus sqlite-only extras | |

**User's choice:** In cache/bench_test.go, always-run

| Option | Description | Selected |
|--------|-------------|----------|
| Ship entries only | Code is the deliverable; numbers never enter docs | ✓ |
| Entries + dated phase artifact | One local run recorded in 17-BENCHMARKS.md | |
| Entries + README numbers | Highest visibility, highest staleness risk | |

**User's choice:** Ship entries only

---

## Docs delta

| Option | Description | Selected |
|--------|-------------|----------|
| Audit + add batch paragraph | Keep structure, add MGet/MSet/MDel semantics, re-verify the four QUAL-03 topics | ✓ |
| Verify and fix inaccuracies only | Smallest diff, batch surface stays undocumented | |
| Full rewrite as a guide | Section per mode plus batch API; duplicates the example | |

**User's choice:** Audit + add batch paragraph

| Option | Description | Selected |
|--------|-------------|----------|
| Add file-backed persistence example | Keep in-memory `Example()`, add Set → Close → reopen → Get on `t.TempDir()` | ✓ |
| Keep the single in-memory example | Already satisfies "runnable example" | |
| Convert existing example to file-backed | One example, but doc example writes to disk | |

**User's choice:** Add file-backed persistence example

| Option | Description | Selected |
|--------|-------------|----------|
| Verify + batch clause | Confirm index row, provider row, paragraph; add one batch clause | ✓ |
| Verify only | Correct only inaccurate statements | |
| Verify + when-to-use table | New README structure beyond the requirement | |

**User's choice:** Verify + batch clause

---

## Test-completeness bar

| Option | Description | Selected |
|--------|-------------|----------|
| Audit-only gate | Map the five named classes to existing tests, run `make coverage-quick`, record evidence | ✓ |
| Audit + gap-fill weak classes | Add one targeted test where a class is only implicit | |
| Coverage margin hardening | Push package ≥90% with file margin so future edits cannot trip the gate | |

**User's choice:** Audit-only gate
**Notes:** All five classes already have named tests; the gate already passes with no override.

| Option | Description | Selected |
|--------|-------------|----------|
| Leave gate inputs as-is | `make coverage` E2E merge stays `./cache/` only; record the reasoning | ✓ |
| Widen merge to ./cache/... | sqlite E2E contributes coverage; slower, inherits E2E flakiness | |
| Widen merge + CI | Also run the e2e lane in the CI coverage job | |

**User's choice:** Leave gate inputs as-is

| Option | Description | Selected |
|--------|-------------|----------|
| Trust the gate + review | Passing gate is the evidence; future override is a reviewable diff | ✓ |
| Add an explicit no-override guard | CI grep or unit test asserting no `cache/sqlite` override | |
| Record baseline numbers in artifacts | Document the package number for future drift detection | |

**User's choice:** Trust the gate + review

---

## the agent's Discretion

- CI step naming/ordering beyond the `Build (CGO-free)` step; whether `go vet` joins it.
- Whether `runBatchBenchmarks` needs a signature change (label/cleanup hook); file-mode cache closing and per-sub-benchmark temp paths.
- Which primitive operations beyond Get/Set/GetOrSet the comparison benchmark includes, and exact sub-benchmark naming.
- Wording of the doc.go batch paragraph and README clause; name of the new durability example and whether it asserts `errors.Is(err, cache.ErrMiss)`.
- Format of the five-class coverage mapping evidence.

## Deferred Ideas

- **Windows `-race` broadening** — remains a Phase 16 reviewer-flagged follow-up; needs mingw and no requirement asks for it.

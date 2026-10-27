---
phase: 17-delivery-hardening
verified: 2026-10-27T12:19:00Z
status: passed
score: 16/16 must-haves verified
covered_files: [".github/workflows/go.yml", ".planning/phases/17-delivery-hardening/17-01-PLAN.md", ".planning/phases/17-delivery-hardening/17-01-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-02-PLAN.md", ".planning/phases/17-delivery-hardening/17-02-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-03-PLAN.md", ".planning/phases/17-delivery-hardening/17-03-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md", ".planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md", "README.md", "cache/README.md", "cache/bench_test.go", "cache/cache_e2e_test.go", "cache/errors.go", "cache/sqlite/doc.go", "cache/sqlite/example_test.go", "go.mod"]
covered_digest: "v3:sha256:7ab65d152aa8732035911007ac98e59b6867e043798c4d1aa86a91d8302cbc77"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 16/16
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 17: Delivery Hardening Verification Report

**Phase Goal:** The package ships per repo standards — CGO-free builds and tests green across the three-OS CI matrix, coverage gates pass with no override, docs and a runnable example exist, and benchmarks register the backend against mem and per-key loops.
**Verified:** 2026-10-27T12:19:00Z
**Status:** passed
**Re-verification:** Yes — triggered by one working-tree change in `.github/workflows/go.yml`; every QUAL-01..04 verdict re-derived from the tree and from fresh command output, nothing carried forward on trust

## What changed since the previous report, and how it was handled

`git status --short` over the covered inputs shows exactly one modification: `.github/workflows/go.yml` (mtime 09:14 local, after the prior `verified: 12:10:28Z`). Nothing else moved — `cache/errors.go` and `17-REVIEW-DISPOSITION.md` are still at 09:07, `go.mod`, `Makefile` and both coverage configs have zero diff against the phase merge, and every other tracked covered file is clean.

| Change | Covered file | Effect on the phase |
|---|---|---|
| `Cache conformance E2E (local providers)`: `-run 'TestCacheE2E_(Mem|SQLite)'` → `-run 'TestCacheE2E_(Mem|SQLite)$'` (`go.yml:84`) | `.github/workflows/go.yml` | **One line, one anchor.** The unanchored pattern also selected `TestCacheE2E_Memcache` (`go test -tags=e2e -list` returns 3 names with it, 2 without), and that test spins testcontainers with no `-short` guard (`cache/cache_e2e_test.go:346-358`), so it fails on a macOS runner with no Docker. This is exactly what run `40501916223` shows (below). Nothing QUAL-01 depends on moved: job-level `CGO_ENABLED: "0"` (`:20`), `Build swapper binaries` (`:32`) → `Build (CGO-free)` (`:38-65`), `Run tests` (`:67`), the single Windows-gated race override (`:94-98`) and the three-OS matrix (`:17`) are byte-identical to the prior pass |

The fix is **uncommitted** in the working tree; it is not yet exercised by CI. It is reported as W-1 below, not as a phase-17 gap.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The three-OS `test` job declares job-level `env: CGO_ENABLED: "0"`, so every step in it runs without cgo | ✓ VERIFIED | `.github/workflows/go.yml:19-20` sits between `runs-on` and `steps:`; `grep -c 'CGO_ENABLED: "0"'` = **1**; steps `:21-98` all inherit it; parsed YAML confirms `jobs.test.env == {"CGO_ENABLED" => "0"}` |
| 2 | `Build (CGO-free)` runs under `shell: bash`, asserts `go env CGO_ENABLED` is `0` (non-zero exit otherwise), runs `go build ./...` + `go vet ./...`, asserts both driver modules resolve to a pin, and is ordered after `Build swapper binaries` | ✓ VERIFIED | `go.yml`: swapper `:32`, build gate `:38`, `shell: bash` `:39`, `set -euo pipefail` `:41`, `cgo=$(go env CGO_ENABLED)` `:46`, `exit 1` `:48-51`, `go build ./...` `:53`, `go vet ./...` `:54`, `pins=$(go list -m …)` `:58`, per-module grep + `exit 1` `:60-65`; `Run tests` follows at `:67`; parsed YAML reproduces the whole block |
| 3 | The `SQLite race detector` step alone sets `CGO_ENABLED: "1"` and keeps `if: runner.os != 'Windows'` | ✓ VERIFIED | `go.yml:94-98`; `grep -c 'CGO_ENABLED: "1"'` = **1**; `grep -c "runner.os != 'Windows'"` = **1** |
| 4 | Locally the job reproduces CGO-free: build, vet, the full unit suite, the two-process E2E and the local conformance E2E pass, and `-race` without cgo still refuses | ✓ VERIFIED | Fresh this pass: `CGO_ENABLED=0 go build ./...` → `BUILD_OK`, `CGO_ENABLED=0 go vet ./...` → `VET_OK`; `make coverage-quick` → every package `ok`, zero `FAIL`; `-tags=e2e -run 'TestTwoProcess' ./cache/sqlite/` → `ok` 0.135 s; `-tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)$' ./cache/` → `ok` 9.165 s, exit 0, 2 top-level tests, 54 subtest `--- PASS` lines, zero Memcache lines; `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c` → `go: -race requires cgo` |
| 5 | `go.mod` is untouched by the phase and pins `modernc.org/sqlite v1.60.1` with `modernc.org/libc v1.77.1` | ✓ VERIFIED | `git diff 1f7e0cf HEAD -- go.mod` → 0 lines (re-checked this pass); `go list -m modernc.org/sqlite modernc.org/libc` → `v1.60.1` / `v1.77.1` |
| 6 | The three-OS CI matrix builds and tests green CGO-free on Linux, macOS and Windows with this job shape | ✓ VERIFIED | Run `40493064377` (`main` @ `2a3a57cd`, created 11:02:03Z): `conclusion: success`, 4/4 jobs. And in the branch run `40501916223` the macOS leg shows `Build swapper binaries` `success`, `Build (CGO-free)` `success`, `Run tests` `success` under the current shape — the only failure is the closeout-appended conformance step (W-1), which asserts nothing about cgo. See W-1 for the pending three-OS validation of the two appended steps |
| 7 | `BenchmarkBatch` registers sqlite as an always-run provider in two named families (`Memory`, `File`) delegating to `runBatchBenchmarks` with distinct names, yielding 48 sub-benchmarks while the mem arms still run | ✓ VERIFIED | `cache/bench_test.go:495-518` (`b.Run("sqlite")` `:503` → `Memory` `:508` with `sqlite.WithMemory()`, `File` `:516` with `WithPath(...tempdir...)`); measured now: `grep -c '^BenchmarkBatch/sqlite/'` = **48**; `-bench 'BenchmarkBatch/MGet/Size1000'` emits `native` **and** `per-key` |
| 8 | `BenchmarkPrimitives` compares `mem`, `sqlite/Memory` and `sqlite/File` across Get/Set/Delete/GetOrSet over the bounded 1024-key ring | ✓ VERIFIED | `cache/bench_test.go:768`; `primitiveKeyRing = 1024` `:30`; measured **12** arms and their names enumerate exactly `mem`, `sqlite/Memory`, `sqlite/File` × `Set/Get/Delete/GetOrSet`; `requireUsable`/`requireNoErr` guard intact |
| 9 | Benchmark arms time what their names claim: MDel arms time only the delete, `GetOrSet` times the miss path, and both key spaces are bounded | ✓ VERIFIED | `cache/bench_test.go:442-481` (`b.StopTimer()` around seeding + `Set`, `b.StartTimer()` immediately before `MDel`/`Delete`); `:749-759` (`StopTimer` → `requireNoErr(Delete)` → `StartTimer` → `GetOrSet`); `batchKeyRing(...)` per `MDel` arm `:445`; `i % primitiveKeyRing` `:752`; `cache/README.md:211-221` documents each arm |
| 10 | Benchmarks are reachable via plain `go test -bench=. ./cache/...` (no `e2e` tag), with no Docker gate on sqlite, no `cache/sqlite/bench_test.go`, no exported helper and no new dependency | ✓ VERIFIED | `cache/bench_test.go` is `package cache_test` with **no** `//go:build` line; `b.Run("sqlite")` `:503` sits before the Docker-gated block `:520`; `ls cache/sqlite/*bench*` → none; only `BenchmarkSingleflightGetOrSet`, `BenchmarkBatch`, `BenchmarkPrimitives` match `^func [A-Z]`; `git diff 1f7e0cf HEAD -- go.mod` → 0 lines; `Makefile:95-103` still globs `./cache/...` |
| 11 | `cache/sqlite/doc.go` carries one batch paragraph inside the package comment, between "Data and state" and "Long-running processes", with no chunk-size number | ✓ VERIFIED | `go doc ./cache/sqlite` renders "Batch operations follow the same durability model…" immediately after the "Data and state:" paragraph and before "Long-running processes…"; `grep -c 100 cache/sqlite/doc.go` = **0** |
| 12 | The four pre-existing QUAL-03 doc topics — TTL resolution/absolute expiry, reopen persistence + `-wal`/`-shm` lifecycle, `:memory:` + location precedence, same-host multi-process constraints — remain present and factually correct | ✓ VERIFIED | Rendered doc text this pass: TTL order + absolute UnixNano + filter-only reads + open sweep/ticker; reopen persistence + WAL checkpoint + sidecar removal + 5 s `busy_timeout` + local-storage-only; `:memory:` + path/name precedence + pinned single connection; "unencrypted at rest" retained |
| 13 | A runnable example proves durability through `WithPath` across Close/reopen with an asserted output, and uses neither `t.TempDir` nor `WithName` | ✓ VERIFIED | `cache/sqlite/example_test.go:39-81`, `// Output: value: hello` at `:80`; `os.MkdirTemp` `:40`; fresh `go test -run Example -v` → `--- PASS: Example`, `--- PASS: ExampleNew_withPath`; `grep -n 'WithName\|TempDir'` matches only the doc-comment mention at `:14` |
| 14 | The README package index names the provider and the sqlite batch clause exists, with no published performance numbers | ✓ VERIFIED | Root index row `README.md:53` names `SQLite` among six backends and links `./cache/README.md`; `cache/README.md:182-189` carries the batch clause (MGet skips, MSet one TTL/one transaction/all-or-nothing, MDel chunked idempotent, unencrypted at rest); the only `ns/op` match is prose at `cache/README.md:217` — no measurement published |
| 15 | `17-COVERAGE-AUDIT.md` records the five QUAL-02 classes with live test citations, the verbatim gate tail, the no-override proof for both configs, the D-13 rationale, and a dated-snapshot caveat | ✓ VERIFIED | Headings `## Five named classes -> existing tests` `:20`, `## Gate output` `:33`, `## No override` `:55`, `## Why the coverage gate inputs stay unchanged (D-13)` `:73`, `## Not added in this phase` `:88`; snapshot caveat `:5-7`; **33/33** distinct `Test*` names it cites resolve to a `func` in `cache/` (0 missing); both override greps reproduce: `grep -v '^#' .testcoverage-quick.yml \| grep -c 'cache/sqlite'` → 0, same for `.testcoverage.yml` → 0 |
| 16 | The coverage gate passes today with no override (package ≥80 %, file ≥70 %, total ≥75 %) | ✓ VERIFIED | Fresh `make coverage-quick`: `File coverage threshold (70%) satisfied: PASS`, `Package coverage threshold (80%) satisfied: PASS`, `Total coverage threshold (75%) satisfied: PASS`, `Total test coverage: 82.4% (2678/3250)`; `cache/sqlite` **92.7 %** this run (see I-1 for the run-to-run 92.4–92.7 % variance); neither coverage config, the Makefile targets, nor `cache/sqlite`'s row in either file changed (`git diff 1f7e0cf HEAD -- …` → 0 lines) |

**Score:** 16/16 truths verified (0 present, behavior-unverified)

### Deferred Items

None — every must-have is satisfied on this tree. (The one deferred idea in the phase, a Windows
`-race` lane, is documented in the workflow comment and remains a future-phase follow-up under WR-04;
it is not a success criterion.)

### Advisory (New Scope, Unevidenced)

None. Every finding in this pass is evidenced by fresh command output or by a live CI run; the
remaining items (W-1, I-1, I-2) contradict no acceptance criterion.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/go.yml` | job-level `CGO_ENABLED: "0"`, asserting `Build (CGO-free)` proof step, one step-level race override, swapper built first, three-OS matrix | ✓ VERIFIED | Structural checks pass (1 job-level `0` `:20`, 1 step-level `1` `:97`, one Windows gate `:95`, swapper `:32` before build gate `:38`, matrix `:17`); YAML parses; run `40493064377` exercises the shape green on all three legs |
| `go.mod` | `modernc.org/sqlite v1.60.1`, `modernc.org/libc v1.77.1`, unchanged | ✓ VERIFIED | `go list -m` matches; no diff vs the phase merge |
| `cache/bench_test.go` | sqlite `Memory`/`File` batch families, `primitiveKeyRing`, `BenchmarkPrimitives` | ✓ VERIFIED | 48 + 12 sub-benchmarks executed this pass |
| `cache/sqlite/doc.go` | one batch paragraph inside the package comment | ✓ VERIFIED | `go doc` renders it between its neighbours |
| `cache/sqlite/example_test.go` | `ExampleNew_withPath` with an asserted output | ✓ VERIFIED | Both examples pass |
| `README.md` + `cache/README.md` | accurate package index; sqlite paragraph gains the batch clause | ✓ VERIFIED | Index row `README.md:53`; clause `cache/README.md:182-189`; no measured numbers |
| `.planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md` | QUAL-02 evidence record with a snapshot caveat | ✓ VERIFIED | 33/33 cited tests exist; gate reproduces PASS; no-override proof reproduces (0/0) |
| `.planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md` | ledger whose YAML and table agree | ✓ VERIFIED | 11 × `disposition: fixed`, `open: 0`, `total: 11` |
| `cache/errors.go` | `ErrClosed` doc comment states provider behaviour accurately | ✓ VERIFIED | Comment-only diff vs prior state; unchanged since the prior pass (mtime predates it) |

`verify.artifacts` totals re-run this pass: 17-01 → 2/2, 17-02 → 1/1, 17-03 → 4/4 (`all_passed: true`, exit 0).

### Key Link Verification

`verify.key-links` reports `total: 0` for all three plans (exit 1) because their `must_haves.key_links`
are plain strings, not the structured `{from,to,via}` form. Wiring was re-checked manually:

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `Build swapper binaries` | `Build (CGO-free)` | `//go:embed` dependency | ✓ WIRED | `release/embed_swapper.go` embeds the generated `swapper/swapper_*` binaries; a clean `go build ./...` fails without them, so `:32` → `:38` is load-bearing |
| job-level `env:` | every `run:` in the `test` job | GitHub job-env inheritance | ✓ WIRED | `CGO_ENABLED: "0"` exists once at `:20`; the build step also asserts the value (`:46-51`), so a lost env fails it; run `40501916223` shows that step `success` on macOS |
| race step `env:` | `-race` build | documented step-over-job precedence | ✓ WIRED | `go.yml:94-98`; race `success` on ubuntu/macOS and `skipped` on windows in run `40493064377` |
| `shell: bash` on `Build (CGO-free)` | exit codes of every command in the block | bash `-e -o pipefail` | ✓ WIRED | `set -euo pipefail` `:41`; the Windows leg runs the step `success` |
| conformance step `-run` pattern | `TestCacheE2E_Mem` + `TestCacheE2E_SQLite` only | anchored alternation `$` | ✓ WIRED | `go test -tags=e2e -list 'TestCacheE2E_(Mem\|SQLite)$'` → exactly 2 names; without `$` → 3, adding `TestCacheE2E_Memcache` |
| `BenchmarkBatch` → `b.Run("sqlite")` | `runBatchBenchmarks` per family | distinct names `sqlite_memory` / `sqlite_file` | ✓ WIRED | `cache/bench_test.go:503-518`; 48 arms observed |
| `b.TempDir()` | `b.Cleanup(Close)` | LIFO close before temp removal | ✓ WIRED | Path computed before construction, cleanup registered after |
| `primitiveKeyRing` / `batchKeyRing` | timed loops | `i % primitiveKeyRing`, `batchKeyRing(...)` | ✓ WIRED | `:752`, `:445`, `:412`, `:472` |
| `make benchmark` / `benchmark-quick` | new families | `go test -bench=. ./cache/...` | ✓ WIRED | `Makefile:95-103`; families discovered with no Makefile edit |
| doc.go batch paragraph | `cache/sqlite/batch.go` + `dsn.go` | claims traced to shipped code | ✓ WIRED | chunking, single transaction + rollback, fail-fast MDel all map to code |
| `cache/README.md` clause | `doc.go` batch paragraph | same facts, no numbers | ✓ WIRED | `cache/README.md:182-189` mirrors the package comment |
| `ExampleNew_withPath` | `WithPath` + Close + reopen | second cache reads the first cache's value | ✓ WIRED | Example passes with `// Output: value: hello` |
| coverage audit | gate + both configs | quoted gate tail + two grep counts | ✓ WIRED | Fresh run reproduces three PASS lines and 0 overrides |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `cache/sqlite/example_test.go` | `value` | reopen of the on-disk DB written by the first cache | Yes — printed line matches `// Output: value: hello` under `go test` | ✓ FLOWING |
| `cache/bench_test.go` primitive arms | provider responses | live `mem` / `sqlite` instances; `requireUsable` probes before timing | Yes — non-zero `ns/op` emitted only after a successful probe | ✓ FLOWING |
| `cache/bench_test.go` batch families | cache instances | `sqlite.WithMemory()` / `WithPath(b.TempDir())` feeding `runBatchBenchmarks` | Yes — 48 sub-benchmarks executed, none skipped | ✓ FLOWING |
| `.github/workflows/go.yml` | `cgo=$(go env CGO_ENABLED)` / `pins=$(go list -m …)` | the live Go toolchain inside the job | Yes — the step's exit status is a function of the observed value, not a constant | ✓ FLOWING |

### Behavioral Spot-Checks

All commands run this pass on `b2ef96a` with the one-line workflow edit in the working tree.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CGO-free build + vet | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...` | `BUILD_OK` / `VET_OK` | ✓ PASS |
| Coverage gate + full unit suite | `make coverage-quick` | every package `ok`, zero `FAIL`; three threshold `PASS`; total 82.4 % (2678/3250); sqlite 92.7 % | ✓ PASS |
| CGO-free two-process E2E | `CGO_ENABLED=0 go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -timeout=180s` | `ok` 0.135 s | ✓ PASS |
| CI conformance step, as now written | `CGO_ENABLED=0 go test -tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)$' ./cache/ -count=1 -timeout=120s -v` | exit 0, `ok` 9.165 s; 2 top-level + 54 subtest `--- PASS`; 0 Memcache mentions | ✓ PASS |
| Same step's pattern BEFORE the fix (selection only) | `go test -tags=e2e -list 'TestCacheE2E_(Mem\|SQLite)' ./cache/` | 3 names — includes `TestCacheE2E_Memcache` (testcontainers, no `-short` guard) | ✓ PASS (bug reproduced, fix targeted) |
| `-race` needs cgo (negative probe) | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c -o /dev/null ./cache/sqlite/` | `go: -race requires cgo` | ✓ PASS |
| Benchmark smoke step, locally | `CGO_ENABLED=0 go test -run=^$ -bench=. -benchtime=1x ./cache/...` | exit 0; 92 `Benchmark` lines; all packages `ok` | ✓ PASS |
| sqlite batch benchmarks | `-bench 'BenchmarkBatch/sqlite' -benchtime=1x` counted | 48 | ✓ PASS |
| mem arms still run | `-bench 'BenchmarkBatch/MGet/Size1000'` | `native` and `per-key` lines emitted | ✓ PASS |
| primitive benchmarks | `-bench 'BenchmarkPrimitives' -benchtime=1x` | 12 lines (`mem`, `sqlite/Memory`, `sqlite/File` × 4 ops) | ✓ PASS |
| examples | `CGO_ENABLED=0 go test ./cache/sqlite/ -run Example -v -count=1` | 2 `--- PASS` lines | ✓ PASS |
| docs render | `go doc ./cache/sqlite` | batch paragraph renders between its neighbours | ✓ PASS |
| three-OS matrix, phase-17 shape | `gh run view 40493064377` | 4/4 jobs `success` (`main` @ `2a3a57cd`) | ✓ PASS |
| three-OS matrix, closeout steps | `gh run view 40501916223` | macOS: steps 4-7 `success`, step 8 `failure`, 9-10 `skipped`; ubuntu + windows `cancelled`; coverage job `success` | ⚠️ W-1 (pending after the one-line fix) |

### Probe Execution

No project or phase-declared `probe-*.sh` exists (`find scripts -path '*/tests/probe-*.sh'` returns
nothing; neither the plans nor the summaries reference a probe). The plan `<verify>` blocks use
direct `go`/`make` commands, all executed above.

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| — | — | No probes declared | N/A |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| QUAL-01 | 17-01 | `CGO_ENABLED=0` on Linux/macOS/Windows; pinned `modernc.org/sqlite` + exact libc pin | ✓ SATISFIED | Workflow structural checks (1 job-level `0`, 1 step-level `1`, swapper first, asserting build step), YAML parses, local CGO-free build/vet/test/E2E, negative race probe, `go list -m`, run `40493064377` green on all three legs, and `Build (CGO-free)` + `Run tests` green on macOS in run `40501916223` |
| QUAL-02 | 17-03 | Tests cover unit, `:memory:`, reopen/persistence, concurrent, sweep; gates pass with no override | ✓ SATISFIED | 33/33 cited tests exist and cover the five classes; fresh `make coverage-quick` passes all three thresholds, `cache/sqlite` 92.7 %, both coverage configs hold 0 `cache/sqlite` entries and are byte-identical to the phase merge |
| QUAL-03 | 17-03 | `doc.go` documents the provider; a runnable example exists; README index updated | ✓ SATISFIED | `go doc` renders the batch paragraph plus the four pre-existing topics; both examples pass with `// Output:` assertions; `README.md:53` index row + `cache/README.md:182-189` clause |
| QUAL-04 | 17-02 | Benchmarks measure sqlite against mem and per-key loops per v1.6 conventions | ✓ SATISFIED | 48 batch + 12 primitive sub-benchmarks executed; shared helpers reused; always-run (no Docker gate); no new dependency; arm semantics documented and honest |

**Orphaned requirements:** none — all four IDs appear in plan frontmatter (`17-01`: QUAL-01;
`17-02`: QUAL-04; `17-03`: QUAL-02, QUAL-03) and match `.planning/REQUIREMENTS.md`, whose checkboxes
are `[x]` and whose traceability table lists all four as `Complete`.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `cache/sqlite/*_test.go` | QUAL-02 | Yes — `grep -rn 't.Skip' cache/sqlite/*_test.go` → **none** (the E2E lane is `//go:build e2e`, run by its own CI step, not skipped) | 0 | No | Behavioral (CRUD, reopen, races, two-process contention, sweep) | PASS |
| `cache/sqlite/example_test.go` | QUAL-03 | Yes (2 examples) | 0 | No | Value (`// Output: value: hello` on both) | PASS |
| `cache/bench_test.go` (sqlite families) | QUAL-04 | Yes — sqlite families never call `skipIfNoDocker` | 0 for sqlite | No | Behavioral — errors fatal via `requireUsable`/`requireNoErr` | PASS |
| `cache/cache_e2e_test.go` | QUAL-02 (supporting) | Yes under `-tags=e2e`; the only `t.Skip`s are the four Docker-provider `-short` guards (`:269`, `:305`, `:348`, `:388`), not mem/sqlite | 0 for mem/sqlite | No | Behavioral (set/get, miss, delete, GetOrSet, batch, close) | PASS |

**Disabled tests on requirements:** 0 → no BLOCKER. **Circular patterns detected:** 0.
**Insufficient assertions:** 0.

### Prohibition Check

The plan prohibitions carry `status: unresolved` with no wired check, so each was judged against the
code on this tree (LLM-judge, non-authoritative — flagged for human review, never silently passed):

| Plan | Prohibition | Verdict | Evidence |
|------|-------------|---------|----------|
| 17-01 | No new GitHub Action, no `permissions:` widening | ✓ HOLDS | `test` job still `checkout@v4` + `setup-go@v5` with `contents: read` / `pull-requests: read` (`:12-14`) |
| 17-01 | No Windows `-race` step, no mingw setup | ✓ HOLDS | One race step only, still gated `if: runner.os != 'Windows'` (`:95`) |
| 17-01 | No per-OS cgo-free job, no per-step toggling beyond one documented exception | ✓ HOLDS | `CGO_ENABLED: "0"` count 1 (job level), `CGO_ENABLED: "1"` count 1 (race step) |
| 17-01 | No change to the separate `coverage` job | ⚠️ DEVIATED (documented, non-blocking) | `55340f4` moved it to `checkout@v4` / `setup-go@v5` + `go-version-file: go.mod` to close IN-04. `threshold-total: 75` and `.testcoverage.yml` are unchanged |
| 17-02 | No `cache/sqlite/bench_test.go`, no exported helpers | ✓ HOLDS | `ls cache/sqlite/*bench*` → none; only the three `Benchmark*` entry points are exported |
| 17-02 | No Docker gate / container / new dependency for sqlite | ✓ HOLDS | sqlite block is before the `skipIfNoDocker` section (`:503` vs `:520`); `go.mod` diff empty |
| 17-02 | No published measured numbers | ✓ HOLDS | No `ns/op`/`allocs/op`/ratio figure in `README.md` or `cache/README.md` other than the prose at `:217` |
| 17-02 | No singleflight herd arm for sqlite | ✓ HOLDS | `BenchmarkSingleflightGetOrSet` covers `mem` only (`:162`) |
| 17-02 | No change to the existing mem blocks or `batchSizes` 1/10/100/1000 | ⚠️ DEVIATED (documented, non-blocking) | `40a7936`/`1e6a9b0`/`a06e207` changed the shared `runBatchBenchmarks` the mem family uses (timer scope, miss path, bounded ring). `batchSizes` is still `{1,10,100,1000}` at `:394`, arm names unchanged, and `cache/README.md:211-221` documents the new semantics |
| 17-03 | No confidentiality/encryption implication | ✓ HOLDS | `doc.go` keeps "unencrypted at rest" in the rendered `go doc` output |
| 17-03 | No new README section, no duplicated example, no performance numbers | ✓ HOLDS | One paragraph inside the existing sqlite behaviour section; no number published |
| 17-03 | No `t.TempDir()` in the example, never `sqlite.WithName` | ✓ HOLDS | `example_test.go:40` uses `os.MkdirTemp`; `WithName` appears only in the doc comment at `:14` |
| 17-03 | No chunk-size number in doc.go or README | ✓ HOLDS | `grep -c 100 cache/sqlite/doc.go` = 0; `cache/README.md`'s only `100`s are the benchmark batch sizes |

The two deviations post-date phase-17 execution, arrived with the v1.8 closeout, sit next to the review
finding they close, and are covered by tests/benchmarks that pass. Reported for review, not treated as
gaps.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (`14/14`, `not_honored: []`,
`blocking: false`) — fresh gate result from this pass.

### Anti-Patterns Found

| ID | File | Line | Pattern | Severity | Impact |
|----|------|------|---------|----------|--------|
| — | `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/sqlite/doc.go`, `cache/sqlite/example_test.go`, `cache/cache_e2e_test.go`, `cache/errors.go`, `README.md`, `cache/README.md` | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`; no stub returns; no hardcoded empty data; changed line is substantive | — | None — the debt-marker gate is clean |
| **W-1** | `.github/workflows/go.yml` | 75-76, 83-84 | The two closeout-appended steps (`Benchmark smoke`, `Cache conformance E2E (local providers)`) still have no green three-OS run. The branch is now pushed: run `40501916223` (`b2ef96a`) failed on macOS **at the conformance step**, which selected `TestCacheE2E_Memcache` (testcontainers, no `-short` guard) through the unanchored `-run` pattern; ubuntu/windows were cancelled behind it. The one-line `$` anchor is fixed in the working tree only | ⚠️ Warning | No phase-17 guarantee is affected. In that same run `Build (CGO-free)`, `Run tests` and `Benchmark smoke` were `success` on macOS under the current shape; locally the fixed step selects exactly 2 top-level tests (54 subtests) and passes CGO-free in 9.17 s, and the benchmark-smoke command exits 0 with 92 arms. The next push of this branch should confirm both steps on all three legs |
| **I-1** | `.planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md` | 7, 39, 46 | The audit's "current" figures (total 82.4 %, `cache/sqlite` 92.4 %) and its quoted gate tail (92.6 %) differ from this run's 92.7 %. Three consecutive measurements reproduce the spread: **92.7 %, 92.4 %, 92.4 %** — the package's busy-retry path makes statement coverage timing-dependent | ℹ️ Info | Run-to-run variance, not drift: the audit is explicitly a dated snapshot (`:5-7`), and 92.4-92.7 % clears the 80 % package threshold by >12 points either way. This also retires the prior pass's I-2 as a real figure mismatch rather than a stale record |
| **I-2** | `.github/workflows/go.yml` | 83-84 | The anchored pattern drops the incidental `TestCacheE2E_Memcache` run that the ubuntu leg (which has Docker) previously got through the over-broad pattern; no CI lane runs the Docker-backed provider E2E | ℹ️ Info | This makes the step match its own documented scope ("Restricted to the local providers (mem, sqlite) on purpose", `:78-82`) instead of contradicting it. Docker-backed provider E2E coverage was never a phase-17 criterion and remains absent by design, unchanged |
| — | `cache/bench_test.go` | ~341-358 | The pre-existing `benchmarkMGetNative`/`benchmarkMGetPerKey` mem arms ignore `Get` errors (`if v, err := c.Get(...); err == nil`) | ℹ️ Info | Inherited from the v1.6 suite and structurally inapplicable to `MGet`; the sqlite families are protected by `prePopulateKeys`' `b.Fatal` and by `requireUsable`/`requireNoErr`, so CR-01's scope is closed |

### Human Verification

N/A — infrastructure/foundation phase (CI/CD, docs, benchmarks) with no user-facing elements, and no
truth asserts a state transition or cancellation/cleanup invariant that lacks a test. All acceptance
criteria are verifiable programmatically and were re-verified above (`behavior_unverified: 0`).

### Flags for Human Review (non-blocking)

These are review flags, not UAT items, and do not affect the status:

1. **Commit and watch the one-line CI fix** — W-1. `go.yml:84` is modified but uncommitted; push
   `gsd/v1.8-closeout` and confirm `Benchmark smoke` and `Cache conformance E2E (local providers)`
   go green on ubuntu, macOS and Windows (macOS already passes the first two CGO-free steps).
2. **Two prohibition deviations** (Prohibition Check) — the coverage-job modernization and the mem-arm
   timing/ring changes arrived with the closeout, are documented in `cache/README.md`, and contradict
   no success criterion; a reviewer may simply acknowledge them.

### Gaps Summary

No blocking gaps and no human verification items. This re-verification re-derived every QUAL-01..04
verdict from the tree and from fresh output after the single input change: the workflow still carries
its job-level `CGO_ENABLED: "0"` (`:20`), its asserting `Build (CGO-free)` step (`:38-65`) after
`make swapper` (`:32`), exactly one step-level race override (`:97`) behind the Windows gate (`:95`),
and the three-OS matrix (`:17`) — all untouched by the one-line `$` anchor on the conformance step.
YAML parses; run `40493064377` is green on all three legs for this shape, and run `40501916223` shows
`Build (CGO-free)`, `Run tests` and `Benchmark smoke` green on macOS with only the closeout conformance
step failing on its over-broad pattern, which the working-tree fix narrows to exactly the two local
providers. Locally: CGO-free build and vet pass, `make coverage-quick` passes all three thresholds with
no override at 82.4 % (2678/3250) with `cache/sqlite` at 92.7 %, the fixed conformance step passes
(2 top-level, 54 subtests, 0 Memcache), the two-process E2E passes, the benchmark-smoke command exits 0
with 92 arms, 48 sqlite batch + 12 primitive arms emit, both examples pass with asserted output,
`go doc` renders the batch paragraph between its neighbours, and 33/33 audit citations name tests that
exist. Remaining findings are a warning that the two closeout steps await a green three-OS run, plus
three informational notes (coverage-run variance, the memcache selection now matching the step's
documented scope, and the inherited mem-arm error suppression) — none contradicts QUAL-01..04.
Phase goal achieved.

---

_Verified: 2026-10-27T12:19:00Z_
_Verifier: the agent (gsd-verifier)_

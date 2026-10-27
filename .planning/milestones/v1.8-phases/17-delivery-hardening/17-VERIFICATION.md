---
phase: 17-delivery-hardening
verified: 2026-10-27T12:27:19Z
status: passed
score: 16/16 must-haves verified
covered_files: [".github/workflows/go.yml", ".planning/phases/17-delivery-hardening/17-01-PLAN.md", ".planning/phases/17-delivery-hardening/17-01-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-02-PLAN.md", ".planning/phases/17-delivery-hardening/17-02-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-03-PLAN.md", ".planning/phases/17-delivery-hardening/17-03-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md", ".planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md", "README.md", "cache/README.md", "cache/bench_test.go", "cache/cache_e2e_test.go", "cache/errors.go", "cache/sqlite/doc.go", "cache/sqlite/example_test.go", "go.mod"]
covered_digest: "v3:sha256:3a6d19d7a17aef1383a4b1ac5a520c6453b76d8bcc779329e183aef416e6aff5"
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
**Verified:** 2026-10-27T12:27:19Z
**Status:** passed
**Re-verification:** Yes — triggered by one commit (`bcd94ac`) touching `.github/workflows/go.yml`; every QUAL-01..04 verdict re-derived from the tree and from fresh command output, nothing carried forward on trust

## Change under re-verification

The report went stale because `.github/workflows/go.yml` changed after the previous `verified:` timestamp. `git show bcd94ac` is the whole delta: **5 insertions, 0 deletions** — three comment lines plus `shell: bash` on two closeout-added steps.

| Change | Effect on the phase |
|---|---|
| `Benchmark smoke` gains `shell: bash` (`go.yml:79`) | pwsh treats `$` as a variable sigil, so `-run=^$` was mangled on windows-latest and `go test` fell back to package `.`. Bash keeps the quoting literal on every runner. Run `40502965929` (parent commit `2ae004c`) shows exactly this: Windows failed **only** at `Benchmark smoke`; `Build (CGO-free)` and `Run tests` were `success` |
| `Cache conformance E2E (local providers)` gains `shell: bash` (`go.yml:88`) | Same `$` in the anchored pattern `'TestCacheE2E_(Mem\|SQLite)$'`; bash is the consistent shell for both commands. Local run under bash selects exactly 2 top-level tests (54 subtests, 0 Memcache) and exits 0 |

Nothing QUAL-01 depends on moved. Re-parsed from YAML this pass: job-level `env == {"CGO_ENABLED" => "0"}`; the three-OS matrix `[ubuntu-latest, macos-latest, windows-latest]`; `Build swapper binaries` (index 2) before `Build (CGO-free)` (index 3, `shell: bash`); `Run tests` (index 4); exactly one step-level `CGO_ENABLED: "1"` (index 8, `if: runner.os != 'Windows'`); `coverage` job unchanged at `threshold-total: 75`.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The three-OS `test` job declares job-level `env: CGO_ENABLED: "0"`, so every step in it runs without cgo | ✓ VERIFIED | `go.yml:19-20`; parsed YAML → `job_env={"CGO_ENABLED"=>"0"}`; `grep -c 'CGO_ENABLED: "0"'` = **1**; steps `:21-103` inherit it |
| 2 | `Build (CGO-free)` runs under `shell: bash`, asserts `go env CGO_ENABLED` is `0` (exit 1 otherwise), runs `go build ./...` + `go vet ./...`, asserts both driver modules resolve to a pin, and is ordered after `Build swapper binaries` | ✓ VERIFIED | `go.yml`: swapper `:32`, build gate `:38`, `shell: bash` `:39`, `set -euo pipefail` `:41`, assert `:46-51`, build `:53`, vet `:54`, pin loop `:58-65`; parsed step order confirms swapper(2) → build(3) |
| 3 | The `SQLite race detector` step alone sets `CGO_ENABLED: "1"` and keeps `if: runner.os != 'Windows'` | ✓ VERIFIED | `go.yml:99-103`; parsed: only step 8 has `env={"CGO_ENABLED"=>"1"}` and `if="runner.os != 'Windows'"`; `grep -c 'CGO_ENABLED: "1"'` = **1** |
| 4 | Locally the job reproduces CGO-free: build, vet, the full unit suite, the two-process E2E and the local conformance E2E pass, and `-race` without cgo still refuses | ✓ VERIFIED | Fresh this pass: `CGO_ENABLED=0 go build ./...` → `BUILD_OK`; `CGO_ENABLED=0 go vet ./...` → `VET_OK`; `make coverage-quick` exit 0, every package `ok`, zero `FAIL`; two-process E2E `ok` 0.333 s; conformance `ok` 9.324 s; `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c` → `go: -race requires cgo` |
| 5 | `go.mod` is untouched by the phase and pins `modernc.org/sqlite v1.60.1` with `modernc.org/libc v1.77.1` | ✓ VERIFIED | `git diff 1f7e0cf HEAD -- go.mod` → 0 lines; `go.mod:19` / `:120`; `go list -m` → `v1.60.1` / `v1.77.1` |
| 6 | The three-OS CI matrix builds and tests green CGO-free on Linux, macOS and Windows with this job shape | ✓ VERIFIED | Run `40493064377` (4/4 jobs `success`, headSha `2a3a57cd`) exercises this CGO shape on all three legs; it predates the two closeout steps (`55340f4` not an ancestor), so it certifies the QUAL-01 shape only. Run `40502965929` shows Windows `Build (CGO-free)` + `Run tests` `success` with the closeout steps present. The in-flight run for `bcd94ac` is not asserted — see W-1 |
| 7 | `BenchmarkBatch` registers sqlite as an always-run provider in two named families (`Memory`, `File`) delegating to `runBatchBenchmarks` with distinct names, yielding 48 sub-benchmarks while the mem arms still run | ✓ VERIFIED | `cache/bench_test.go:495-518`; measured now: `grep -c '^BenchmarkBatch/sqlite/'` = **48**; mem arms present (`BenchmarkBatch/MGet/Size1000/{native,per-key}` emitted) |
| 8 | `BenchmarkPrimitives` compares `mem`, `sqlite/Memory` and `sqlite/File` across Get/Set/Delete/GetOrSet over the bounded 1024-key ring | ✓ VERIFIED | `cache/bench_test.go:768`; `primitiveKeyRing = 1024` `:30`; measured **12** arms, names enumerate exactly the three providers × four ops |
| 9 | Benchmark arms time what their names claim: MDel arms time only the delete, `GetOrSet` times the miss path, and both key spaces are bounded | ✓ VERIFIED | `:442-481` (`StopTimer` around seeding + `Set`, `StartTimer` before `MDel`/`Delete`); `:749-759` (`StopTimer` → `Delete` → `StartTimer` → `GetOrSet`); `batchKeyRing(...)` `:445`; `i % primitiveKeyRing` `:752`; documented `cache/README.md:211-221` |
| 10 | Benchmarks are reachable via plain `go test -bench=. ./cache/...` (no `e2e` tag), with no Docker gate on sqlite, no `cache/sqlite/bench_test.go`, no exported helper and no new dependency | ✓ VERIFIED | `cache/bench_test.go` is `package cache_test`, no `//go:build` line; `b.Run("sqlite")` `:503` precedes the Docker-gated block `:520`; `ls cache/sqlite/*bench*` → none; only `BenchmarkSingleflightGetOrSet`/`BenchmarkBatch`/`BenchmarkPrimitives` match `^func [A-Z]`; `go.mod` diff empty |
| 11 | `cache/sqlite/doc.go` carries one batch paragraph inside the package comment, between "Data and state" and "Long-running processes", with no chunk-size number | ✓ VERIFIED | `go doc ./cache/sqlite` → `Data and state` line 36, `Batch operations…` line 46, `Long-running processes` line 57; `grep -c 100 cache/sqlite/doc.go` = 0 |
| 12 | The four pre-existing QUAL-03 doc topics — TTL resolution/absolute expiry, reopen persistence + `-wal`/`-shm` lifecycle, `:memory:` + location precedence, same-host multi-process constraints — remain present and factually correct | ✓ VERIFIED | Rendered doc text this pass: TTL order + absolute UnixNano + filter-only reads + open sweep/ticker; reopen persistence + WAL checkpoint + sidecar removal + 5 s `busy_timeout` + local-storage-only; `:memory:` + path/name precedence + pinned single connection; "unencrypted at rest" retained |
| 13 | A runnable example proves durability through `WithPath` across Close/reopen with an asserted output, and uses neither `t.TempDir` nor `WithName` | ✓ VERIFIED | `cache/sqlite/example_test.go:39-81`, `// Output: value: hello` `:80`; `os.MkdirTemp` `:40`; fresh run → `--- PASS: Example`, `--- PASS: ExampleNew_withPath` |
| 14 | The README package index names the provider and the sqlite batch clause exists, with no published performance numbers | ✓ VERIFIED | `README.md:53` names `SQLite` among six backends and links `./cache/README.md`; clause `cache/README.md:182-189`; only `ns/op` occurrence is prose at `:217` |
| 15 | `17-COVERAGE-AUDIT.md` records the five QUAL-02 classes with live test citations, the verbatim gate tail, the no-override proof for both configs, the D-13 rationale, and a dated-snapshot caveat | ✓ VERIFIED | Headings `:20`, `:33`, `:55`, `:73`, `:88`; snapshot caveat `:5-7`; **33/33** distinct `Test*` names it cites resolve to a `func` in `cache/` (0 missing); both override greps reproduce: 0 / 0 |
| 16 | The coverage gate passes today with no override (package ≥80 %, file ≥70 %, total ≥75 %) | ✓ VERIFIED | Fresh `make coverage-quick` (exit 0): `File coverage threshold (70%) satisfied: PASS`, `Package coverage threshold (80%) satisfied: PASS`, `Total coverage threshold (75%) satisfied: PASS`, `Total test coverage: 82.4% (2679/3250)`; `cache/sqlite` **92.7 %**; neither coverage config nor the Makefile targets changed (`git diff 1f7e0cf HEAD -- … Makefile` → 0 lines); sqlite appears in neither override list |

**Score:** 16/16 truths verified (0 present, behavior-unverified)

### Deferred Items

None. The one deferred idea in the phase — a Windows `-race` lane — remains a documented future-phase follow-up under WR-04 and is not a success criterion.

### Advisory (New Scope, Unevidenced)

None. Every finding in this pass is evidenced by fresh command output, by parsed workflow structure, or by a read-only lookup of a completed CI run.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/go.yml` | job-level `CGO_ENABLED: "0"`, asserting `Build (CGO-free)` proof step, one step-level race override, swapper built first, three-OS matrix, two closeout steps under bash | ✓ VERIFIED | Parsed YAML confirms all of it; `shell: bash` on steps 3, 5, 6; `verify.artifacts` 2/2 |
| `go.mod` | `modernc.org/sqlite v1.60.1`, `modernc.org/libc v1.77.1`, unchanged | ✓ VERIFIED | `go list -m` matches; no diff vs the phase merge |
| `cache/bench_test.go` | sqlite `Memory`/`File` batch families, `primitiveKeyRing`, `BenchmarkPrimitives` | ✓ VERIFIED | 48 + 12 sub-benchmarks executed this pass; `verify.artifacts` 1/1 |
| `cache/sqlite/doc.go` | one batch paragraph inside the package comment | ✓ VERIFIED | `go doc` renders it between its neighbours |
| `cache/sqlite/example_test.go` | `ExampleNew_withPath` with an asserted output | ✓ VERIFIED | Both examples pass |
| `README.md` + `cache/README.md` | accurate package index; sqlite paragraph gains the batch clause | ✓ VERIFIED | Index row `README.md:53`; clause `cache/README.md:182-189`; no measured numbers |
| `.planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md` | QUAL-02 evidence record with a snapshot caveat | ✓ VERIFIED | 33/33 cited tests exist; gate reproduces PASS; no-override proof reproduces (0/0) |
| `.planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md` | ledger whose YAML and table agree | ✓ VERIFIED | 11 × `disposition: fixed`, `open: 0`, `total: 11` |
| `cache/errors.go` | `ErrClosed` doc comment states provider behaviour accurately | ✓ VERIFIED | Unchanged this round; comment-only diff vs prior state |

`verify.artifacts` re-run this pass: 17-01 → 2/2, 17-02 → 1/1, 17-03 → 4/4 (`all_passed: true`, exit 0).

### Key Link Verification

`verify.key-links` reports `total: 0` for all three plans (exit 1) because their `must_haves.key_links` are plain strings, not the structured `{from,to,via}` form. Wiring was re-checked manually:

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `Build swapper binaries` | `Build (CGO-free)` | `//go:embed` dependency | ✓ WIRED | `release/embed_swapper.go` embeds `swapper/swapper_*`; clean build fails without them |
| job-level `env:` | every `run:` in the `test` job | GitHub job-env inheritance | ✓ WIRED | Parsed `job_env={"CGO_ENABLED"=>"0"}`; build step asserts the value (`:46-51`), so a lost env fails it |
| race step `env:` | `-race` build | step-over-job precedence | ✓ WIRED | Parsed: only step 8 carries `"1"`, gated by `runner.os != 'Windows'` |
| `shell: bash` on `Build (CGO-free)` | exit codes of every command in the block | bash `-e -o pipefail` | ✓ WIRED | `set -euo pipefail` `:41` |
| `shell: bash` on `Benchmark smoke` | literal `-run=^$` reaching `go test` | bash does not expand a trailing `$`; pwsh did | ✓ WIRED | `go.yml:79`; the pre-fix Windows failure in run `40502965929` is the counter-example; local bash run exits 0 with 92 arms |
| `shell: bash` on `Cache conformance E2E` | anchored alternation `$` | single-quoted pattern under bash | ✓ WIRED | `go.yml:88-89`; `-list` → exactly 2 names |
| conformance step `-run` pattern | `TestCacheE2E_Mem` + `TestCacheE2E_SQLite` only | anchored alternation `$` | ✓ WIRED | `-list` returns exactly 2; Memcache absent (0 mentions in `-v` output) |
| `BenchmarkBatch` → `b.Run("sqlite")` | `runBatchBenchmarks` per family | distinct names `sqlite_memory` / `sqlite_file` | ✓ WIRED | `cache/bench_test.go:503-518`; 48 arms observed |
| `b.TempDir()` | `b.Cleanup(Close)` | LIFO close before temp removal | ✓ WIRED | Path computed before construction, cleanup registered after |
| `primitiveKeyRing` / `batchKeyRing` | timed loops | `i % primitiveKeyRing`, `batchKeyRing(...)` | ✓ WIRED | `:752`, `:445`, `:412`, `:472` |
| `make benchmark` / `benchmark-quick` | new families | `go test -bench=. ./cache/...` | ✓ WIRED | `Makefile:95-103` unchanged; families discovered |
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

All commands run this pass on `bcd94ac` (clean tree).

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CGO-free build + vet | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...` | `BUILD_OK` / `VET_OK` | ✓ PASS |
| Coverage gate + full unit suite | `make coverage-quick` | exit 0; every package `ok`, zero `FAIL`; three threshold `PASS`; total 82.4 % (2679/3250); sqlite 92.7 % | ✓ PASS |
| CI benchmark-smoke step, as now written | `bash -c 'go test -run=^$ -bench=. -benchtime=1x ./cache/...'` | exit 0; 92 `Benchmark` lines; 0 `FAIL` | ✓ PASS |
| CI conformance step, as now written | `bash -c "go test -tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)$' ./cache/ -count=1 -timeout=120s"` | exit 0; `ok` 9.324 s | ✓ PASS |
| Conformance selection + contents | same with `-v`, plus `-list` | `-list` → exactly 2; `-v` → 2 top-level `--- PASS` + 54 subtests; 0 Memcache mentions | ✓ PASS |
| CGO-free two-process E2E | `CGO_ENABLED=0 go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1` | `ok` 0.333 s | ✓ PASS |
| `-race` needs cgo (negative probe) | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c -o /dev/null ./cache/sqlite/` | `go: -race requires cgo` | ✓ PASS |
| sqlite batch benchmarks | grep `^BenchmarkBatch/sqlite/` in the smoke output | 48 | ✓ PASS |
| mem arms still run | grep `^BenchmarkBatch/(MGet\|MSet\|MDel)/Size1000/(native\|per-key)` | 6 lines emitted | ✓ PASS |
| primitive benchmarks | grep `^BenchmarkPrimitives/` | 12 arms (3 providers × 4 ops) | ✓ PASS |
| examples | `CGO_ENABLED=0 go test ./cache/sqlite/ -run Example -v -count=1` | 2 `--- PASS` lines | ✓ PASS |
| docs render | `go doc ./cache/sqlite` | batch paragraph at line 46, between 36 and 57 | ✓ PASS |
| three-OS matrix, CGO shape | `gh run view 40493064377` | 4/4 jobs `success`; ubuntu/macos/windows legs green | ✓ PASS |
| closeout steps, pre-fix run | `gh run view 40502965929` (Windows leg) | `Build (CGO-free)` + `Run tests` `success`; `Benchmark smoke` `failure` — the pwsh defect `shell: bash` fixes | ✓ PASS (defect confirmed) |

### Probe Execution

No project or phase-declared `probe-*.sh` exists. The plan `<verify>` blocks use direct `go`/`make` commands, all executed above.

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| — | — | No probes declared | N/A |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| QUAL-01 | 17-01 | `CGO_ENABLED=0` on Linux/macOS/Windows; pinned `modernc.org/sqlite` + exact libc pin | ✓ SATISFIED | Parsed job env `0`, single step-level `1` behind the Windows gate, swapper-before-build ordering, asserting build step; local CGO-free build/vet/test/E2E; negative race probe; `go list -m` pins; run `40493064377` green on three legs for this shape |
| QUAL-02 | 17-03 | Tests cover unit, `:memory:`, reopen/persistence, concurrent, sweep; gates pass with no override | ✓ SATISFIED | 33/33 cited tests exist and cover the five classes; fresh `make coverage-quick` passes all three thresholds, sqlite 92.7 %, 0 overrides in both configs; configs and Makefile unchanged |
| QUAL-03 | 17-03 | `doc.go` documents the provider; a runnable example exists; README index updated | ✓ SATISFIED | `go doc` renders the batch paragraph plus four pre-existing topics; both examples pass with `// Output:`; `README.md:53` index row + `cache/README.md:182-189` clause |
| QUAL-04 | 17-02 | Benchmarks measure sqlite against mem and per-key loops per v1.6 conventions | ✓ SATISFIED | 48 batch + 12 primitive sub-benchmarks executed; shared helpers reused; always-run (no Docker gate); no new dependency; arm semantics documented |

**Orphaned requirements:** none — all four IDs appear in plan frontmatter and match `.planning/REQUIREMENTS.md`, whose checkboxes are `[x]` and whose traceability table lists all four as `Complete`.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `cache/sqlite/*_test.go` | QUAL-02 | Yes — no `t.Skip` in the package's unit tests | 0 | No | Behavioral (CRUD, reopen, races, two-process contention, sweep) | PASS |
| `cache/sqlite/example_test.go` | QUAL-03 | Yes (2 examples) | 0 | No | Value (`// Output: value: hello`) | PASS |
| `cache/bench_test.go` (sqlite families) | QUAL-04 | Yes — sqlite families never call `skipIfNoDocker` | 0 for sqlite | No | Behavioral — errors fatal via `requireUsable`/`requireNoErr` | PASS |
| `cache/cache_e2e_test.go` | QUAL-02 (supporting) | Yes under `-tags=e2e`; the only `t.Skip`s are the four Docker-provider `-short` guards (`:269`, `:305`, `:348`, `:388`), not mem/sqlite | 0 for mem/sqlite | No | Behavioral (set/get, miss, delete, GetOrSet, batch, close) | PASS |

**Disabled tests on requirements:** 0 → no BLOCKER. **Circular patterns detected:** 0. **Insufficient assertions:** 0.

### Prohibition Check

The plan prohibitions carry `status: unresolved` with no wired check, so each was judged against the code on this tree (LLM-judge, non-authoritative — flagged for human review, never silently passed):

| Plan | Prohibition | Verdict | Evidence |
|------|-------------|---------|----------|
| 17-01 | No new GitHub Action, no `permissions:` widening | ✓ HOLDS | `test` job still `checkout@v4` + `setup-go@v5` with `contents: read` / `pull-requests: read` (`:12-14`) |
| 17-01 | No Windows `-race` step, no mingw setup | ✓ HOLDS | One race step only, still gated `if: runner.os != 'Windows'` (`:100`) |
| 17-01 | No per-OS cgo-free job, no per-step toggling beyond one documented exception | ✓ HOLDS | Parsed: one job-level `0`, one step-level `1` |
| 17-01 | No change to the separate `coverage` job | ⚠️ DEVIATED (documented, non-blocking) | `55340f4` moved it to `checkout@v4` / `setup-go@v5` + `go-version-file: go.mod` to close IN-04. `threshold-total: 75` and `.testcoverage.yml` unchanged |
| 17-02 | No `cache/sqlite/bench_test.go`, no exported helpers | ✓ HOLDS | `ls cache/sqlite/*bench*` → none; only the three `Benchmark*` entry points are exported |
| 17-02 | No Docker gate / container / new dependency for sqlite | ✓ HOLDS | sqlite block before `skipIfNoDocker` (`:503` vs `:520`); `go.mod` diff empty |
| 17-02 | No published measured numbers | ✓ HOLDS | Only `ns/op` occurrence is prose at `cache/README.md:217` |
| 17-02 | No singleflight herd arm for sqlite | ✓ HOLDS | `BenchmarkSingleflightGetOrSet` covers `mem` only (`:162`) |
| 17-02 | No change to the existing mem blocks or `batchSizes` 1/10/100/1000 | ⚠️ DEVIATED (documented, non-blocking) | `40a7936`/`1e6a9b0`/`a06e207` changed the shared `runBatchBenchmarks` the mem family uses (timer scope, miss path, bounded ring). `batchSizes := []int{1, 10, 100, 1000}` `:394`, arm names unchanged, semantics documented `cache/README.md:211-221` |
| 17-03 | No confidentiality/encryption implication | ✓ HOLDS | "unencrypted at rest" retained in rendered `go doc` |
| 17-03 | No new README section, no duplicated example, no performance numbers | ✓ HOLDS | One paragraph inside the existing sqlite behaviour section; no number published |
| 17-03 | No `t.TempDir()` in the example, never `sqlite.WithName` | ✓ HOLDS | `example_test.go:40` uses `os.MkdirTemp`; `WithName` only in the doc comment `:14` |
| 17-03 | No chunk-size number in doc.go or README | ✓ HOLDS | `grep -c 100 cache/sqlite/doc.go` = 0 |

The two deviations post-date phase-17 execution, arrived with the v1.8 closeout, sit next to the review finding they close, and are covered by tests/benchmarks that pass. Reported for review, not treated as gaps.

### Decision Coverage

The automated gate skips this phase (`check decision-coverage-verify` expects `CONTEXT.md`; the phase stores `17-CONTEXT.md`), so D-01..D-14 were re-checked by hand against the shipped artifacts: **all 14 honored in substance**. D-01..D-09, D-11..D-14 match verbatim. D-10's mechanism differs from its wording — it named `t.TempDir()` for the durability example, while `17-03`'s prohibition (and its must-have) forbid `t.TempDir` inside the example, so the implementation uses `os.MkdirTemp`; both keep the database out of the real cache directory, so the decision's intent (a temp-backed durability example, existing `Example()` untouched) holds.

### Anti-Patterns Found

| ID | File | Line | Pattern | Severity | Impact |
|----|------|------|---------|----------|--------|
| — | `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/sqlite/doc.go`, `cache/sqlite/example_test.go`, `cache/errors.go` | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`; no stub returns; no hardcoded empty data; changed lines are substantive (`shell: bash` is load-bearing) | — | None — the debt-marker gate is clean |
| **W-1** | `.github/workflows/go.yml` | 78-89 | The two closeout-appended steps have no green three-OS run yet for the current commit. Run `40502965929` (`2ae004c`) failed on Windows **only** at `Benchmark smoke` (pwsh `$` mangling — exactly the defect `bcd94ac` fixes); the run for `bcd94ac` is in flight and its outcome is not asserted here | ⚠️ Warning | No phase-17 guarantee is affected: the CGO shape is untouched, and locally both commands now run under bash with exit 0 (92 arms; 2 top-level / 54 subtests / 0 Memcache). The next completed run should confirm both steps on all three legs |
| **I-1** | `.planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md` | 7, 39, 43 | The audit's dated figures (total 82.1 % (2628/3201), sqlite 92.6 %) differ from this run's 82.4 % (2679/3250) / 92.7 %. The package's busy-retry path makes statement coverage timing-dependent, and the tree grew 49 statements after the closeout | ℹ️ Info | The audit is explicitly a dated snapshot (`:5-7`); 92.7 % clears the 80 % package threshold by >12 points, and the gate passes on all three thresholds |
| **I-2** | `.github/workflows/go.yml` | 87-89 | The anchored pattern intentionally drops `TestCacheE2E_Memcache`; no CI lane runs the Docker-backed provider E2E | ℹ️ Info | Matches the step's own comment (`:82-86`) and the phase's scope; Docker-backed E2E coverage was never a phase-17 criterion |
| **I-3** | `cache/bench_test.go` | ~341-358 | The pre-existing mem `benchmarkMGetNative`/`benchmarkMGetPerKey` arms ignore `Get` errors (`if v, err := c.Get(...); err == nil`) | ℹ️ Info | Inherited from v1.6; the sqlite families are protected by `prePopulateKeys`' `b.Fatal` and `requireUsable`/`requireNoErr` |

### Human Verification

N/A — infrastructure/foundation phase (CI/CD, docs, benchmarks) with no user-facing elements, and no truth asserts a state transition or cancellation/cleanup invariant that lacks a test. All acceptance criteria are verifiable programmatically and were re-verified above (`behavior_unverified: 0`).

### Flags for Human Review (non-blocking)

These are review flags, not UAT items, and do not affect the status:

1. **Watch the in-flight run for `bcd94ac`** — W-1. Confirm `Benchmark smoke` and `Cache conformance E2E (local providers)` go green on ubuntu, macOS and Windows now that both declare `shell: bash`. No local evidence contradicts that expectation, but CI is the only Windows runtime.
2. **Two prohibition deviations** (Prohibition Check) — the coverage-job modernization and the mem-arm timing/ring changes arrived with the closeout, are documented in `cache/README.md`, and contradict no success criterion; a reviewer may simply acknowledge them.

### Gaps Summary

No blocking gaps and no human verification items. The single input change since the prior pass — commit `bcd94ac`, 5 insertions adding `shell: bash` to `Benchmark smoke` and `Cache conformance E2E (local providers)` — touches neither the CGO shape nor any phase-17 deliverable. Re-derived from the tree: job-level `CGO_ENABLED: "0"` (`:20`), the asserting `Build (CGO-free)` step (`:38-65`) after `make swapper` (`:32`), exactly one step-level race override (`:102`) behind the Windows gate (`:100`), and the three-OS matrix (`:17`) are all intact, and the YAML parses to exactly that structure. Locally: CGO-free build and vet pass; `make coverage-quick` passes all three thresholds with no override at 82.4 % (2679/3250) with `cache/sqlite` at 92.7 %; the benchmark-smoke command under bash exits 0 with 92 arms; the conformance step under bash selects exactly 2 top-level tests (54 subtests, 0 Memcache) and exits 0; the two-process E2E passes; 48 sqlite batch + 12 primitive arms emit; both examples pass with asserted output; `go doc` renders the batch paragraph between its neighbours; and 33/33 audit citations name tests that exist. The pre-fix run `40502965929` shows the defect was confined to `Benchmark smoke` on Windows, with `Build (CGO-free)` and `Run tests` already green there. Remaining findings are one warning (the in-flight run is the first three-OS exercise of the two closeout steps under bash) and three informational notes — none contradicts QUAL-01..04. Phase goal achieved.

---

_Verified: 2026-10-27T12:27:19Z_
_Verifier: the agent (gsd-verifier)_

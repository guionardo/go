---
phase: 17-delivery-hardening
verified: 2026-10-27T12:10:28Z
status: passed
score: 16/16 must-haves verified
covered_files: [".github/workflows/go.yml", ".planning/phases/17-delivery-hardening/17-01-PLAN.md", ".planning/phases/17-delivery-hardening/17-01-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-02-PLAN.md", ".planning/phases/17-delivery-hardening/17-02-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-03-PLAN.md", ".planning/phases/17-delivery-hardening/17-03-SUMMARY.md", ".planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md", ".planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md", "README.md", "cache/README.md", "cache/bench_test.go", "cache/cache_e2e_test.go", "cache/errors.go", "cache/sqlite/doc.go", "cache/sqlite/example_test.go", "go.mod"]
covered_digest: "v3:sha256:fdd504f8563f672a0636eb44f429845d8a53c559082f353dbcddd05af55a1e22"
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
**Verified:** 2026-10-27T12:09:21Z
**Status:** passed
**Re-verification:** Yes — final pass on `gsd/v1.8-closeout` (`cd348c3`), re-derived end to end from the working tree; nothing carried forward from the prior report

## What changed since the previous report, and how it was handled

`git log --since='2026-10-27T12:03:37Z'` over the workflow, package, docs and phase path set returns
nothing: the only movement is two uncommitted record/comment edits, both newer than the prior
`verified:` timestamp by file mtime (09:07 local vs 09:03). Every claim below was re-executed.

| Change | Covered file | Effect on the phase |
|---|---|---|
| `17-REVIEW-DISPOSITION.md` — the eleven per-finding YAML `disposition:` fields now read `fixed`, matching the table and `open: 0` | `.planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md` | Bookkeeping only, and it **closes the prior I-3**: the frontmatter block, the table and the aggregate now agree (`fixed` × 11, `open: 0`, `total: 11`). No acceptance criterion reads this ledger |
| `cache/errors.go` — the `ErrClosed` doc comment now states each provider's real post-`Close` behaviour | `cache/errors.go` | **Comment-only.** `git diff` shows exactly one hunk inside the `ErrClosed` doc comment and no code line. Claims spot-checked against the tree: `cache/sqlite` wraps the sentinel (`sqlite.go:48`), memcache's `CloseFunc` is a documented no-op (`memcache.go:134-136`), mem reports `ErrMiss` (`mem.go:49`); redis/valkey/postgres reference no shared sentinel and surface their client library's own closed error. No behaviour, build or coverage change |

No other covered input moved. The prior pass's I-2 (audit snapshot said `cache/sqlite` 92.4 % while
that pass measured 92.7 % module-wide) is also closed by this run: `make coverage-quick` reports
**92.4 %**, exactly the snapshot's "current" figure.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The three-OS `test` job declares job-level `env: CGO_ENABLED: "0"`, so every step in it runs without cgo | ✓ VERIFIED | `.github/workflows/go.yml:19-20` sits between `runs-on` and `steps:`; `grep -c 'CGO_ENABLED: "0"'` = **1**; steps `:21-98` all inherit it |
| 2 | `Build (CGO-free)` runs under `shell: bash`, asserts `go env CGO_ENABLED` is `0` (non-zero exit otherwise), runs `go build ./...` + `go vet ./...`, asserts both driver modules resolve to a pin, and is ordered after `Build swapper binaries` | ✓ VERIFIED | `.github/workflows/go.yml`: swapper `:32`, build gate `:38`, `shell: bash` `:39`, `set -euo pipefail` `:41`, `cgo=$(go env CGO_ENABLED)` `:46`, `exit 1` `:48-51`, `go build ./...` `:53`, `go vet ./...` `:54`, `pins=$(go list -m …)` `:58`, per-module grep + `exit 1` `:60-65`; `Run tests` follows at `:67` |
| 3 | The `SQLite race detector` step alone sets `CGO_ENABLED: "1"` and keeps `if: runner.os != 'Windows'` | ✓ VERIFIED | `.github/workflows/go.yml:95-98`; `grep -c 'CGO_ENABLED: "1"'` = **1**; `grep -c "runner.os != 'Windows'"` = **1**. Run `40493064377`: the step is `success` on ubuntu + macOS and `skipped` on windows |
| 4 | Locally the job reproduces CGO-free: build, vet, the full unit suite, the two-process E2E and the local conformance E2E pass, and `-race` without cgo still refuses | ✓ VERIFIED | Fresh this pass: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...` → both exit 0; `make coverage-quick` ran `go test ./...` → every package `ok` (one `[no test files]`), zero `FAIL`; `-tags=e2e -run 'TestTwoProcess' ./cache/sqlite/` → `ok` (0.35 s); `-tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)' ./cache/` → `ok` (12.96 s); `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c` → `go: -race requires cgo` |
| 5 | `go.mod` is untouched by the phase and pins `modernc.org/sqlite v1.60.1` with `modernc.org/libc v1.77.1` | ✓ VERIFIED | `git diff 1f7e0cf HEAD -- go.mod` → 0 lines; `go list -m modernc.org/sqlite modernc.org/libc` → `v1.60.1` / `v1.77.1`; `go.mod:19` and `:120` |
| 6 | The three-OS CI matrix actually builds and tests green CGO-free on Linux, macOS and Windows with this job shape | ✓ VERIFIED | `gh run view 40493064377` (current `main` at `2a3a57cd`, post-phase-17): `conclusion: success`, 4/4 jobs. On all three legs `Build swapper binaries` → `Build (CGO-free)` → `Run tests` → `SQLite two-process E2E` are `success`; `SQLite race detector` is `success` on ubuntu + macOS, `skipped` on windows. This run predates the two closeout-appended steps — see W-1 |
| 7 | `BenchmarkBatch` registers sqlite as an always-run provider in two named families (`Memory`, `File`) delegating to `runBatchBenchmarks` with distinct names, yielding 48 sub-benchmarks while the mem arms still run | ✓ VERIFIED | `cache/bench_test.go:495-518` (`b.Run("sqlite")` → `Memory` `:508` with `sqlite.WithMemory()`, `File` `:516` with `WithPath(filepath.Join(b.TempDir(),"batch.db"))`; mem family `:499`). Measured now: `grep -c '^BenchmarkBatch/sqlite/'` = **48**; `-bench 'BenchmarkBatch/MGet/Size1000'` emits `native` **and** `per-key` |
| 8 | `BenchmarkPrimitives` compares `mem`, `sqlite/Memory` and `sqlite/File` across Get/Set/Delete/GetOrSet over the bounded 1024-key ring | ✓ VERIFIED | `cache/bench_test.go:768` (three providers) and `:716-763` (keyed arms); `primitiveKeyRing = 1024` `:30`; `i % primitiveKeyRing` at `:731`, `:738`, `:752`; measured **12** `BenchmarkPrimitives/` arms; CR-01 guard intact — `requireUsable` fails on an unusable provider, `requireNoErr` after every operation |
| 9 | Benchmark arms time what their names claim: MDel arms time only the delete, `GetOrSet` times the miss path, and both key spaces are bounded | ✓ VERIFIED | `cache/bench_test.go:445-482` (`b.StopTimer()` around seeding, `StartTimer` before the delete); `:750-758` (delete-before-`GetOrSet` with the timer stopped); `batchKeyRing` `:365-372` reused by every batch arm (`:412`, `:445`, `:472`); `cache/README.md:211-221` states each arm's measurement |
| 10 | Benchmarks are reachable via plain `go test -bench=. ./cache/...` (no `e2e` tag), with no Docker gate on sqlite, no `cache/sqlite/bench_test.go`, no exported helper and no new dependency | ✓ VERIFIED | `cache/bench_test.go` is `package cache_test` with **no** `//go:build` line; the sqlite block is outside the `skipIfNoDocker` section; `ls cache/sqlite/*bench*` → no match; only `BenchmarkSingleflightGetOrSet`, `BenchmarkBatch`, `BenchmarkPrimitives` match `^func [A-Z]`; `go.mod` diff empty; `Makefile:95-103` still globs `./cache/...` |
| 11 | `cache/sqlite/doc.go` carries one batch paragraph inside the package comment, between "Data and state" and "Long-running processes", with no chunk-size number | ✓ VERIFIED | `cache/sqlite/doc.go:45-55`, neighbours at `:34` and `:56`; `go doc ./cache/sqlite` renders it verbatim between them; claims traced to shipped code — `chunkSize = 100` (`batch.go:20`), single `BeginTx` `:192` + deferred `Rollback` `:197`, and `_txlock=immediate` in both DSNs (`dsn.go:15-16`); `grep -c 100 doc.go` = **0** |
| 12 | The four pre-existing QUAL-03 doc topics — TTL resolution/absolute expiry, reopen persistence + `-wal`/`-shm` lifecycle, `:memory:` + location precedence, same-host multi-process constraints — remain present and factually correct | ✓ VERIFIED | `doc.go:19-26` (`:memory:`, path/name precedence, pinned single connection), `:28-32` (TTL order, absolute UnixNano, filter-only reads, open sweep + ticker), `:34-43` (reopen persistence, WAL checkpoint + sidecar removal, 5 s `busy_timeout`, local-storage only), `:66-68` (`unencrypted at rest` retained) — each claim matched against `dsn.go`, `batch.go` and the tested behaviours |
| 13 | A runnable example proves durability through `WithPath` across Close/reopen with an asserted output, and uses neither `t.TempDir` nor `WithName` | ✓ VERIFIED | `cache/sqlite/example_test.go:39-81`, `// Output: value: hello` at `:80`, `os.MkdirTemp` `:40`, `defer first.Close()` `:55`, explicit `first.Close()` `:63` before `second := sqlite.New(WithPath(path))` `:69`; fresh `go test -run Example -v` → `--- PASS: Example`, `--- PASS: ExampleNew_withPath`. `grep -n 'WithName\|TempDir'` matches only the doc-comment mention at `:14` |
| 14 | The README package index names the provider and the sqlite batch clause exists, with no published performance numbers | ✓ VERIFIED | Root index row `README.md:53` lists `SQLite` among the six backends and links `./cache/README.md`; the sqlite paragraph is `cache/README.md:182-189` (MGet skips, MSet one TTL/one transaction/all-or-nothing, MDel chunked idempotent, unencrypted at rest), mirroring `doc.go:45-55`; the only `ns/op` match is prose `cache/README.md:217` describing what an arm reports — no measurement is published |
| 15 | `17-COVERAGE-AUDIT.md` records the five QUAL-02 classes with live test citations, the verbatim gate tail, the no-override proof for both configs, the D-13 rationale, and a dated-snapshot caveat | ✓ VERIFIED | File read end-to-end: five class rows, snapshot note `:5-9`; `## Gate output`, `## No override`, `## Why the coverage gate inputs stay unchanged (D-13)`. All **33** distinct `Test*` names it cites resolve to a `func` in `cache/` (0 missing). Both override greps reproduce: `grep -v '^#' .testcoverage-quick.yml \| grep -c 'cache/sqlite'` → 0, same for `.testcoverage.yml` → 0 |
| 16 | The coverage gate passes today with no override (package ≥80 %, file ≥70 %, total ≥75 %) | ✓ VERIFIED | Fresh `make coverage-quick`: `File coverage threshold (70%) satisfied: PASS`, `Package coverage threshold (80%) satisfied: PASS`, `Total coverage threshold (75%) satisfied: PASS`, `Total test coverage: 82.4% (2678/3250)`; `cache/sqlite` **92.4 %** — identical to the audit's "current" snapshot figure. Neither coverage config, the Makefile targets, nor `cache/sqlite`'s row in either file changed (`git diff 1f7e0cf HEAD -- .testcoverage.yml .testcoverage-quick.yml Makefile` → 0 lines) |

**Score:** 16/16 truths verified (0 present, behavior-unverified)

### Deferred Items

None — every must-have is satisfied on this tree. (The one deferred idea in the phase, a Windows
`-race` lane, is documented in the workflow comment and remains a future-phase follow-up under WR-04;
it is not a success criterion.)

### Advisory (New Scope, Unevidenced)

None. Every finding in this pass is evidenced; the two prior record-accuracy notes (I-2, I-3) are
closed by this tree, and the remaining items (W-1, the inherited mem-arm error suppression) contradict
no acceptance criterion.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/go.yml` | job-level `CGO_ENABLED: "0"`, asserting `Build (CGO-free)` proof step, one step-level race override, swapper built first, three-OS matrix | ✓ VERIFIED | Structural checks pass (1 job-level `0`, 1 step-level `1`, one Windows gate, swapper `:32` before build gate `:38`); the three-OS run `40493064377` exercises the shape green |
| `go.mod` | `modernc.org/sqlite v1.60.1`, `modernc.org/libc v1.77.1`, unchanged | ✓ VERIFIED | `go list -m` matches; no diff vs the phase merge |
| `cache/bench_test.go` | sqlite `Memory`/`File` batch families, `primitiveKeyRing`, `runPrimitiveBenchmarks`, `BenchmarkPrimitives` | ✓ VERIFIED | 48 + 12 sub-benchmarks executed; additive to the mem suite |
| `cache/sqlite/doc.go` | one batch paragraph inside the package comment | ✓ VERIFIED | `go doc` renders it between its neighbours; claims traced to `batch.go`/`dsn.go` |
| `cache/sqlite/example_test.go` | `ExampleNew_withPath` with an asserted output | ✓ VERIFIED | Both examples pass |
| `README.md` + `cache/README.md` | accurate package index; sqlite paragraph gains the batch clause | ✓ VERIFIED | Index row `README.md:53`; clause `cache/README.md:182-189`; no measured numbers |
| `.planning/phases/17-delivery-hardening/17-COVERAGE-AUDIT.md` | QUAL-02 evidence record with a snapshot caveat | ✓ VERIFIED | 33/33 cited tests exist; gate reproduces PASS; no-override proof reproduces (0/0); snapshot "current" figures match this run |
| `.planning/phases/17-delivery-hardening/17-REVIEW-DISPOSITION.md` | ledger whose YAML and table agree | ✓ VERIFIED | 11 × `disposition: fixed`, `open: 0`, `total: 11`, table matching |
| `cache/errors.go` | `ErrClosed` doc comment states provider behaviour accurately | ✓ VERIFIED | Comment-only diff; sqlite wraps, redis/valkey/postgres surface client errors, memcache Close is a no-op (`memcache.go:134-136`), mem reports misses (`mem.go:49`); no code or coverage change |

`verify.artifacts` totals: 17-01 → 2/2, 17-02 → 1/1, 17-03 → 4/4 (`all_passed: true`, exit 0).

### Key Link Verification

`verify.key-links` reports `total: 0` for all three plans (exit 1) because their `must_haves.key_links`
are plain strings, not the structured `{from,to,via}` form. Wiring was verified manually:

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `Build swapper binaries` | `Build (CGO-free)` | `//go:embed` dependency | ✓ WIRED | `release/embed_swapper.go:14-17` embeds `swapper/swapper_{linux_amd64,darwin_amd64,darwin_arm64,windows_amd64.exe}`, gitignored (`.gitignore:36-37`) and produced by `make swapper` (`Makefile:159-169`); a clean `go build ./...` fails without them, so the order `go.yml:32` → `:38` is load-bearing |
| job-level `env:` | every `run:` in the `test` job | GitHub job-env inheritance | ✓ WIRED | `CGO_ENABLED: "0"` exists once at `:20`; run `40493064377` shows `Build (CGO-free)` success on all three legs (the step also asserts the value, so a lost env fails it) |
| race step `env:` | `-race` build | documented step-over-job precedence | ✓ WIRED | `go.yml:95-98`; race `success` on ubuntu/macOS, `skipped` on windows in run `40493064377` |
| `shell: bash` on `Build (CGO-free)` | exit codes of every command in the block | bash `-e -o pipefail` | ✓ WIRED | `set -euo pipefail` `:41`; the Windows leg runs the step `success` with both pins asserted |
| `go list -m` in the build step | CI log pins | `pins=$(…)` + per-module grep | ✓ WIRED | `:58-65` inside the same shell block, so a missing pin fails the step |
| `BenchmarkBatch` → `b.Run("sqlite")` | `runBatchBenchmarks` per family | distinct names `sqlite_memory` / `sqlite_file` | ✓ WIRED | `cache/bench_test.go:503-518`; every generated key embeds the name, so families cannot collide (48 arms observed) |
| `b.TempDir()` | `b.Cleanup(Close)` | LIFO close before temp removal | ✓ WIRED | Path computed before construction, cleanup registered after (`:511-517`, `:784-789`) |
| `primitiveKeyRing` / `batchKeyRing` | timed loops | `i % primitiveKeyRing`, `batchKeyRing` | ✓ WIRED | `:731`, `:738`, `:752`; `:412`, `:445`, `:472` |
| `make benchmark` / `benchmark-quick` | new families | `go test -bench=. ./cache/...` | ✓ WIRED | `Makefile:95-103`; families discovered with no Makefile edit |
| doc.go batch paragraph | `cache/sqlite/batch.go` + `dsn.go` | claims traced to shipped code | ✓ WIRED | `chunkSize=100` (`batch.go:20`), `BeginTx` + deferred `Rollback` `:192`/`:197`, `_txlock=immediate` (`dsn.go:15-16`) |
| `cache/README.md` clause | `doc.go:45-55` | same facts, no numbers | ✓ WIRED | `cache/README.md:182-189` mirrors the package comment |
| `ExampleNew_withPath` | `WithPath` + Close + reopen | second cache reads the first cache's value | ✓ WIRED | Example passes with `// Output: value: hello` |
| coverage audit | gate + both configs | quoted gate tail + two grep counts | ✓ WIRED | Fresh run reproduces three PASS lines and 0 overrides |
| `TestCacheE2E_SQLite` | sqlite file mode | `sqlite.WithPath(filepath.Join(t.TempDir(),"e2e.db"))` | ✓ WIRED | `cache/cache_e2e_test.go:257-262`; suite `ok` locally CGO-free (12.96 s) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `cache/sqlite/example_test.go` | `value` | reopen of the on-disk DB written by the first cache | Yes — printed line matches `// Output: value: hello` under `go test` | ✓ FLOWING |
| `cache/bench_test.go` primitive arms | provider responses | live `mem` / `sqlite` instances; `requireUsable` proves the provider works before timing | Yes — non-zero `ns/op` emitted only after a successful Set/Get probe | ✓ FLOWING |
| `cache/bench_test.go` batch families | cache instances | `sqlite.WithMemory()` / `WithPath(b.TempDir())`, each feeding `runBatchBenchmarks` | Yes — 48 sub-benchmarks executed, none skipped | ✓ FLOWING |
| `cache/sqlite/doc.go`, `cache/README.md` | prose claims | shipped `batch.go` / `dsn.go` behaviour | Yes — chunking, single transaction, rollback and fail-fast MDel all map to code | ✓ FLOWING |

### Behavioral Spot-Checks

All commands were run this pass on `cd348c3` with the comment-only `cache/errors.go` edit in place.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CGO-free build + vet | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...` | both exit 0 | ✓ PASS |
| Coverage gate + full unit suite | `make coverage-quick` | every package `ok`, zero `FAIL`; three threshold `PASS`; total 82.4 % (2678/3250); sqlite 92.4 % | ✓ PASS |
| CGO-free two-process E2E | `CGO_ENABLED=0 go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -timeout=180s` | `ok` 0.35 s | ✓ PASS |
| Local conformance E2E (CI step equivalent) | `CGO_ENABLED=0 go test -tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)' ./cache/ -count=1 -timeout=120s` | `ok` 12.96 s | ✓ PASS |
| `-race` needs cgo (negative probe) | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -race -c -o /dev/null ./cache/sqlite/` | `go: -race requires cgo` | ✓ PASS |
| sqlite batch benchmarks | `go test ./cache/ -run '^$' -bench 'BenchmarkBatch/sqlite' -benchtime=1x -count=1 \| grep -c '^BenchmarkBatch/sqlite/'` | 48 | ✓ PASS |
| mem arms still run | `-bench 'BenchmarkBatch/MGet/Size1000'` | `native` and `per-key` lines emitted | ✓ PASS |
| primitive benchmarks | `-bench 'BenchmarkPrimitives' -benchtime=1x` | 12 lines (`mem`, `sqlite/Memory`, `sqlite/File` × 4 ops) | ✓ PASS |
| examples | `CGO_ENABLED=0 go test ./cache/sqlite/ -run Example -v -count=1` | 2 `--- PASS` lines | ✓ PASS |
| docs render | `go doc ./cache/sqlite` | batch paragraph renders between its neighbours | ✓ PASS |
| three-OS matrix (current `main`) | `gh run view 40493064377` | 4/4 jobs `success`; `Build (CGO-free)` success on all three legs; race green on ubuntu/macOS, skipped on windows | ✓ PASS |

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
| QUAL-01 | 17-01 | `CGO_ENABLED=0` on Linux/macOS/Windows; pinned `modernc.org/sqlite` + exact libc pin | ✓ SATISFIED | Workflow structural checks (1 job-level `0`, 1 step-level `1`, swapper first, asserting build step), local CGO-free build/vet/test/E2E, negative race probe, `go list -m`, and run `40493064377` green on all three legs |
| QUAL-02 | 17-03 | Tests cover unit, `:memory:`, reopen/persistence, concurrent, sweep; gates pass with no override | ✓ SATISFIED | 33/33 cited tests exist and cover the five classes; fresh `make coverage-quick` passes all three thresholds, `cache/sqlite` 92.4 %, and both coverage configs hold 0 `cache/sqlite` entries (and are byte-identical to the phase merge) |
| QUAL-03 | 17-03 | `doc.go` documents the provider; a runnable example exists; README index updated | ✓ SATISFIED | `go doc` renders the batch paragraph; both examples pass with `// Output:` assertions; `README.md:53` index row + `cache/README.md:182-189` clause |
| QUAL-04 | 17-02 | Benchmarks measure sqlite against mem and per-key loops per v1.6 conventions | ✓ SATISFIED | 48 batch + 12 primitive sub-benchmarks executed; shared helpers reused; always-run (no Docker gate); no new dependency; arm semantics documented and honest |

**Orphaned requirements:** none — all four IDs appear in plan frontmatter (`17-01`: QUAL-01;
`17-02`: QUAL-04; `17-03`: QUAL-02, QUAL-03) and match `.planning/REQUIREMENTS.md`, whose checkboxes
are `[x]` and whose traceability table lists all four as `Complete`.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `cache/sqlite/*_test.go` | QUAL-02 | Yes — `grep -rn 't.Skip\|t.Skipf' cache/sqlite/*_test.go` → **none** (the E2E lane is `//go:build e2e`, run by its own CI step, not skipped) | 0 | No | Behavioral (CRUD, reopen, races, two-process contention, sweep) | PASS |
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
| 17-02 | No Docker gate / container / new dependency for sqlite | ✓ HOLDS | sqlite block is outside the `skipIfNoDocker` section; `go.mod` diff empty |
| 17-02 | No published measured numbers | ✓ HOLDS | No `ns/op`/`allocs/op`/ratio figure in `README.md` or `cache/README.md` |
| 17-02 | No singleflight herd arm for sqlite | ✓ HOLDS | `BenchmarkSingleflightGetOrSet` covers `mem` only (`:162`) |
| 17-02 | No change to the existing mem blocks or `batchSizes` 1/10/100/1000 | ⚠️ DEVIATED (documented, non-blocking) | `40a7936`/`1e6a9b0`/`a06e207` changed the shared `runBatchBenchmarks` the mem family uses (timer scope, miss path, bounded ring). `batchSizes` is still `{1,10,100,1000}`, arm names are unchanged, and `cache/README.md:211-221` documents the new semantics |
| 17-03 | No confidentiality/encryption implication | ✓ HOLDS | `doc.go:66-67` keeps "Cache contents are unencrypted at rest" |
| 17-03 | No new README section, no duplicated example, no performance numbers | ✓ HOLDS | One paragraph inside the existing sqlite behaviour section; no number published |
| 17-03 | No `t.TempDir()` in the example, never `sqlite.WithName` | ✓ HOLDS | `example_test.go:40` uses `os.MkdirTemp`; `WithName` appears only in the doc comment at `:14` |
| 17-03 | No chunk-size number in doc.go or README | ✓ HOLDS | `grep 100 cache/sqlite/doc.go` → none; `cache/README.md`'s only `100`s are the benchmark batch sizes |

The two deviations post-date phase-17 execution, arrived with the v1.8 closeout, sit next to the review
finding they close, and are covered by tests/benchmarks that pass. Reported for review, not treated as
gaps.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (`14/14`, `not_honored: []`,
`blocking: false`) — fresh gate result from this pass.

### Anti-Patterns Found

| ID | File | Line | Pattern | Severity | Impact |
|----|------|------|---------|----------|--------|
| — | `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/sqlite/doc.go`, `cache/sqlite/example_test.go`, `cache/cache_e2e_test.go`, `cache/errors.go`, `README.md`, `cache/README.md` | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`; no stub returns; no hardcoded empty data | — | None — the debt-marker gate is clean |
| **W-1** | `.github/workflows/go.yml` | 75-84, 107-111 | The two closeout-appended steps (`Benchmark smoke`, `Cache conformance E2E (local providers)`) and the coverage-job action bump have no three-OS run yet: the branch is unpushed (`gh run list --branch gsd/v1.8-closeout` returns nothing) and `main` does not contain them. The latest green matrices (`40493064377`) predate them | ⚠️ Warning | No phase-17 guarantee is affected: both steps inherit the job-level `CGO_ENABLED: "0"`, and their local CGO-free equivalents pass (48 + 12 benchmark arms; conformance E2E `ok` in 12.96 s, including the `sqlite_file` row). The next push of this branch should confirm the two steps on all three legs |
| — | `cache/bench_test.go` | 341-358 | The pre-existing `benchmarkMGetNative`/`benchmarkMGetPerKey` mem arms ignore `Get` errors (`if v, err := c.Get(...); err == nil`) | ℹ️ Info | Inherited from the v1.6 suite and structurally inapplicable to `MGet` (no error channel); the sqlite families are protected by `prePopulateKeys`' `b.Fatal` on `Set` and by `requireUsable`/`requireNoErr` on the primitive path, so CR-01's scope is closed |

**Resolved since the previous report:** I-3 (ledger YAML now `fixed` × 11, `open: 0`, matching the
table) and I-2 (this run's module-wide `cache/sqlite` figure is 92.4 %, exactly the audit snapshot's
"current" value). Both were record-accuracy notes; closing them changes no truth, artifact or link.

### Human Verification

N/A — infrastructure/foundation phase (CI/CD, docs, benchmarks) with no user-facing elements. All
acceptance criteria are verifiable programmatically and were re-verified above.

### Flags for Human Review (non-blocking)

These are review flags, not UAT items, and do not affect the status:

1. **Watch the first `gsd/v1.8-closeout` CI run** — W-1. Confirm the two appended steps go green on
   ubuntu, macOS and Windows.
2. **Two prohibition deviations** (Prohibition Check) — the coverage-job modernization and the mem-arm
   timing/ring changes arrived with the closeout, are documented in `cache/README.md`, and contradict
   no success criterion; a reviewer may simply acknowledge them.

### Gaps Summary

No blocking gaps and no human verification items. This final pass re-derived every phase claim from the
code and from fresh command output: the workflow still carries its job-level `CGO_ENABLED=0`, its
asserting `Build (CGO-free)` step after `make swapper`, exactly one step-level race override behind the
Windows gate, and the three-OS matrix is green in run `40493064377` (all legs `success` on the build
gate and two-process E2E; race success on ubuntu/macOS, skipped on windows). Local CGO-free build, vet,
the full unit suite, the two-process E2E and the local conformance E2E pass; the `-race requires cgo`
negative probe still fires. Benchmarks emit 48 sqlite batch arms and 12 primitive arms with the mem arms
still running; both examples pass with asserted output; `go doc` renders the batch paragraph between its
neighbours; 33/33 audit citations name tests that exist. `make coverage-quick` passes all three
thresholds with no override at 82.4 % (2678/3250) and `cache/sqlite` at 92.4 %, matching the audit
snapshot, with both coverage configs byte-identical to the phase merge. The review ledger's YAML now
agrees with its table. The only remaining findings are a warning that two closeout-appended CI steps are
not yet exercised on the matrix, an inherited mem-arm info note, and two documented prohibition
deviations — none of which contradicts QUAL-01..04. Phase goal achieved.

---

_Verified: 2026-10-27T12:10:28Z_
_Verifier: the agent (gsd-verifier)_

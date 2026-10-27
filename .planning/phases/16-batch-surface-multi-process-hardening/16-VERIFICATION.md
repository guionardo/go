---
phase: 16-batch-surface-multi-process-hardening
verified: 2026-10-27T11:48:53Z
status: passed
score: 15/15 must-haves verified
covered_files:
  - .github/workflows/go.yml
  - .planning/phases/16-batch-surface-multi-process-hardening/16-01-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-01-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-02-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-02-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-03-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-03-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-04-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-04-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-05-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-05-SUMMARY.md
  - cache/sqlite/batch.go
  - cache/sqlite/batch_internal_test.go
  - cache/sqlite/doc.go
  - cache/sqlite/dsn.go
  - cache/sqlite/dsn_internal_test.go
  - cache/sqlite/retry.go
  - cache/sqlite/retry_internal_test.go
  - cache/sqlite/schema.go
  - cache/sqlite/sqlite.go
  - cache/sqlite/sqlite_internal_test.go
  - cache/sqlite/sqlite_test.go
  - cache/sqlite/twoprocess_e2e_test.go
  - release/release.go
  - release/self_update.go
  - release/update.go
  - release/update_test.go
covered_digest: "v3:sha256:cf2406e754ce57df1339765a0ec5f01b5bf9bfeb609281c7ffafb93a71eb7452"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 14/14
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 16: Batch Surface + Multi-Process Hardening Verification Report

**Phase Goal:** The `BatchCache` surface is complete with v1.6 semantics, and the milestone's headline promise is proven — two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`.
**Verified:** 2026-10-27T11:48:53Z
**Status:** passed
**Re-verification:** Yes — re-derived from the codebase after Phase 17, Phase 17.1, and the v1.8 closeout changed files the prior report covered. No must-have was carried forward on trust: every truth below was re-checked against the current tree, and the formerly deferred remote-CI item is now closed by observable GitHub Actions run evidence (see "External CI Evidence").

## Goal Achievement

### Observable Truths

Truths merged from the 5 ROADMAP.md success criteria (non-negotiable) and the PLAN frontmatter must-haves of 16-01..16-05, deduplicated. Every truth was re-verified against the shipped code and its tests in this session; no SUMMARY.md claim was used as evidence.

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1 | `MGet` retrieves via chunked `IN` queries (fixed `chunkSize = 100`, dedup before chunking); missing and expired rows are absent by the SQL predicate, undecodable rows are skipped at debug, a failed chunk is warn-and-skipped, and an empty input issues no query (BATCH-01) | ✓ VERIFIED | `cache/sqlite/batch.go:22` (`chunkSize = 100`), `:78-116` (MGetFunc), `:124-152` (mgetChunk closes `rows` per chunk); `schema.go:36` (`MGetSelectSQL` with expiry predicate, `%s` = markers only). `TestMGetFunc` 8/8 subtests PASS (empty-no-query, expired, corrupt-row skip, chunk-failure warn, cross-chunk 150 keys, closed provider). |
| 2 | `MSet` is atomic: all values marshal before the transaction opens; one `BeginTx` (BEGIN IMMEDIATE via `_txlock`) with one prepared `UpsertSQL`, one `ExecContext` per pair, `Commit`, deferred Rollback backstop — any error leaves zero rows from the batch (BATCH-02) | ✓ VERIFIED | `batch.go:157-186` (pre-marshal, no tx yet), `:188-222` (msetTx: BeginTx → PrepareContext → Exec per pair → Commit; `defer Rollback`). `TestMSetFunc` + `TestMSetNoPartialWrites` (5 iterations, 2000 pairs, `count == 0 \|\| count == 2000`) PASS. |
| 3 | `MSet` applies exactly one TTL per batch: `resolveTTL` is called once and the same bound `expires_at` is used for every row (BATCH-02) | ✓ VERIFIED | `batch.go:174` single `resolveTTL(ttl...)`, `:176-184` one `exp` bound to every Exec. `TestBatchMSetOneTTL` + `TestBatchSemantics/one_ttl_expires_every_key_of_the_batch` PASS. |
| 4 | `MDel` deletes via chunked `IN` lists; duplicate and missing keys are no-ops; the operation is idempotent; the first failing chunk returns immediately, wrapped with the `cache/sqlite:` prefix (BATCH-03) | ✓ VERIFIED | `batch.go:224-244` (uniqueKeys → chunksOf → ExecContext per chunk, first error wrapped); `schema.go:41` (`MDelSQL`). `TestMDelFunc` 6/6 subtests PASS (missing nil, empty no-query, duplicate collapse, 250-key chunking, closed-handle fail-fast prefix, `ErrClosed` identity) + `TestBatchMDel`. |
| 5 | Two OS processes drive the SHIPPED provider against one fresh cache file: both children exit 0, and a raw reopen reports `journal_mode = wal`, `integrity_check = ok`, exactly 287 rows per role, both final markers, and the deleted keys absent (CONC-01, CONC-05, D-10A) | ✓ VERIFIED | `twoprocess_e2e_test.go:413-473` (TestTwoProcessContention; `perRoleRows = 287` with the derivation at `:52-58`); children construct `sqlite.New(WithPath(...))` (`:110-127`). Re-ran this session: PASS (0.02s); remote CI PASS on ubuntu, macOS **and Windows** (log excerpts below). |
| 6 | Crash arm: a crasher holds an open BEGIN IMMEDIATE with an uncommitted row (marker-synchronized), the parent kills it, the survivor exits 0, and the reopen shows the uncommitted row rolled back, the survivor marker present, and `integrity_check = ok` (CONC-05, D-10B) | ✓ VERIFIED | `twoprocess_e2e_test.go:135-192` (crashMain), `:475-520` (TestTwoProcessCrashRecovery; `assertKeyAbsent("crash:uncommitted")`, survivor `:final` decoded). Re-ran: PASS (0.03s); remote CI PASS on all three OSes. |
| 7 | Contention surfaces at BEGIN within `busy_timeout` as a bounded, busy-classified, `cache/sqlite:`-wrapped error; the provider performs no retry beyond the open window; the identical write succeeds after the blocker rolls back (CONC-02, D-07) | ✓ VERIFIED | `sqlite_internal_test.go:386-419` (TestBeginContentionBounded: `assert.True(isBusyError(err))`, `elapsed < 2*busyTimeout`, rollback → retry succeeds) PASS (5.11s ≈ one 5 s budget, not two). Exactly one production `retryBusy(` call site (`sqlite.go:118`, at PingContext) — grep over non-test files confirms no other caller. |
| 8 | The provider passes `go test -race` under concurrent goroutines on the pinned single-connection pool (CONC-03) | ✓ VERIFIED | `TestConcurrentBatchOpsRace` (file + memory subtests, 8 goroutines mixing point/batch/GetOrSet) PASS locally under `-race` (2.27s, no `DATA RACE`); remote race step green on ubuntu (6.16s) and macOS (6.42s). |
| 9 | Package docs state WAL's constraints: same-host file sharing supported, contention bounded by `busy_timeout` and surfacing at BEGIN, bounded busy-only open retry, network/cross-host unsupported, canceled MGet returns partial results (CONC-04, A5) | ✓ VERIFIED | `cache/sqlite/doc.go` "Data and state:" paragraph (same-host, 5 s `busy_timeout`, surfaces at BEGIN, bounded open retry, "network filesystems and cross-host sharing are unsupported") and "Batch operations" paragraph (canceled `MGet` → partial results). Phase 17.1's `MGet` deferred-error exception is documented in the same block and matches `batch.go:82-92`. |
| 10 | CI runs the two-process E2E on the three-OS matrix and the race detector on non-Windows runners; both steps execute the intended tests (CONC-05, D-09) | ✓ VERIFIED | `.github/workflows/go.yml:86-87` — exactly one `SQLite two-process E2E` step, no OS gate, `-run 'TestTwoProcess'`; `:94-98` — one `SQLite race detector` step gated `runner.os != 'Windows'` with step-level `CGO_ENABLED: "1"`. **Remote proof** (run 40493064377): windows job log shows `CGO_ENABLED: 0`, `=== RUN TestTwoProcessContention` → PASS, `=== RUN TestTwoProcessCrashRecovery` → PASS; ubuntu/macOS race steps show `ok github.com/guionardo/go/cache/sqlite`; windows race step `skipped`. |
| 11 | DI-15-01 closed: a bounded, busy-only open retry (5 ms backoff / 5 s absolute deadline / `errors.As` classifier on `Code()&0xff == 5`) wraps exactly one `PingContext` between pool pinning and bootstrap; non-busy errors are never retried; concurrent first-open is deterministic (D-06, CONC-01) | ✓ VERIFIED | `retry.go:15-73` (constants, `isBusyError`, `retryBusy`, `retryBusyWithin` deadline check); `sqlite.go:100-130` (pin → `retryBusy(ctx, c.db.PingContext)` → `bootstrap`). `TestIsBusyError`, `TestRetryBusyWithin`, `TestBusyTimeoutDSNSync`, `TestBootstrapIdempotence/concurrent_first_open_all_usable` (4 workers, all usable) PASS. |
| 12 | `journal_size_limit 64MB` rides the file-mode DSN; append order is autocheckpoint → journal_size_limit; memory mode and non-positive values append nothing; read-backs are 67108864 / -1 / coexistence with 200 (D-08) | ✓ VERIFIED | `dsn.go:15-27` (constants incl. `dsnJournalSizeLimitPrefix`), `:94-114` (4-arg `buildDSN`); `TestDSNBuildDSN`, `TestJournalSizeLimitReadBack`, `TestAutoCheckpointReadBack`, `TestPragmaReadBack` PASS. |
| 13 | The retry budget is mechanically pinned to the DSN; the e2e harness never enters the unit lane; no coverage override and no go.mod change (D-07, prohibitions) | ✓ VERIFIED | `TestBusyTimeoutDSNSync` parses `_busy_timeout=` out of `dsnSuffixFile` and asserts 5000 ms. Untagged `go test -list 'TestTwoProcess' ./cache/sqlite/` lists **zero**; `-tags=e2e` lists both; untagged package suite `ok`. No `cache/sqlite` entry in `.testcoverage-quick.yml`; `git diff --name-only ea54bd1..HEAD -- go.mod go.sum` is empty. |
| 14 | G-16-1: a failed `DownloadUpdate` leaves no partial artifact on ANY OS — the handle is closed before `removeFile(filePath)` — and a removal failure is surfaced via `errors.Join`, pinned by a non-parallel seam test (CONC-05) | ✓ VERIFIED | `release/update.go:27-33` grouped `var (githubAPIBase; removeFile = os.Remove)` seam; `:146-160` `_ = f.Close()` → `removeFile(filePath)` → `errors.Join(download, removal)`. `TestDownloadUpdate_CleanupFailureSurfaced` (`nolint:paralleltest`, `require.ErrorIs(stubErr)`) + `TestDownloadUpdate_DigestMismatch` (`require.NoFileExists`) + all 6 DownloadUpdate tests PASS locally; remote windows "Run tests" step green (reconciliation observed, not assumed). |
| 15 | The CONC-01 evidence artifact exists and records the spike harness, the raw-arm failure rate, the policy-arm result, the confirmed retry triple, and the disposition (D-11) | ✓ VERIFIED | `16-SPIKE-FINDINGS.md`: 8/10 raw-arm failures with `SQLITE_BUSY` at first use, 30/30 clean policy runs, 301-row/`integrity_check=ok`/`wal` assertions, 5 ms / 5 s / busy-only / PingContext-seam triple, DI-15-01 disposition. It documents the 16-01 harness shape (superseded by the 16-04 provider-driven arms, as its §5 states). |

**Score:** 15/15 truths verified (0 present, behavior-unverified).

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `cache/sqlite/batch.go` | MGetFunc/MSetFunc/MDelFunc real implementations + dedup/chunk/placeholder helpers, `chunkSize = 100` | ✓ VERIFIED | Read in full; single-tx seam intact, no stub returns, no per-pair autocommit path. |
| `cache/sqlite/batch_internal_test.go` | Dedup, chunk boundaries, placeholder text, decode skip, empty no-ops, atomicity, MDel fail-fast/idempotency, closed-branch log level | ✓ VERIFIED | 9 tests; all PASS. |
| `cache/sqlite/twoprocess_e2e_test.go` | `//go:build e2e` helper-process harness: provider-driven children, contention + crash arms, bounded waits | ✓ VERIFIED | First line `//go:build e2e`; both arms re-ran PASS; every child interaction timed (`childTimeout`, `markerTimeout`); cleanup kills+waits. |
| `cache/sqlite/sqlite.go` | `open()` retry wiring between pool pinning and bootstrap; `check()` single-prefix error contract | ✓ VERIFIED | Read; one `retryBusy` call site; `check()` returns `ErrClosed`/`ErrInvalidPath` as-is, wraps only bare errors (17.1 contract). |
| `cache/sqlite/retry.go` | busy classifier, constants, bounded loop | ✓ VERIFIED | Present; deadline checked before each attempt. |
| `cache/sqlite/dsn.go` | 4-arg `buildDSN`, journal-size-limit prefix + constant | ✓ VERIFIED | Present; compile-time constants only — no caller strings. |
| `cache/sqlite/schema.go` | `MGetSelectSQL` (expiry folded) + `MDelSQL` + `UpsertSQL` | ✓ VERIFIED | Present; `%s` is a marker list only. |
| `cache/sqlite/doc.go` | CONC-04 multi-process contract + canceled-MGet note (+ 17.1 MGet error exception) | ✓ VERIFIED | Present and matches shipped behavior. |
| `.github/workflows/go.yml` | one 'SQLite two-process E2E' (no OS gate) + one 'SQLite race detector' (non-Windows) | ✓ VERIFIED | Confirmed by name- and command-level read; remote steps executed as intended. |
| `release/update.go` | close-before-remove, `removeFile` seam, `errors.Join` | ✓ VERIFIED | Read in full; success path unchanged. |
| `release/update_test.go` | seam regression test + untouched DigestMismatch guard | ✓ VERIFIED | Read; `DigestMismatch` still asserts `require.NoFileExists`. |
| `.planning/.../16-SPIKE-FINDINGS.md` | D-11 evidence record | ✓ VERIFIED | Present; records raw/policy counts and the policy triple. |

No artifact is MISSING, STUB, ORPHANED or HOLLOW.

### Key Link Verification

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `MGetFunc` | `MGetSelectSQL` | uniqueKeys → chunksOf(100) → per-chunk `QueryContext` with marker list + bound `now` | WIRED | `rows` closed before the next chunk (single-conn pool would otherwise deadlock). |
| `MSetFunc` | `UpsertSQL` | pre-marshal → `resolveTTL` once → `msetTx` (BeginTx → Prepare → Exec per pair → Commit) | WIRED | Zero `SetFunc` references in the batch path. |
| `MDelFunc` | `MDelSQL` | uniqueKeys → chunksOf → per-chunk `ExecContext` | WIRED | Deleting a missing key is a SQL no-op. |
| `open()` | `retryBusy` → `isBusyError` → `retryBusyWithin` | `c.db.PingContext` between pool pinning and bootstrap | WIRED | Single call site (`sqlite.go:118`). |
| `DownloadUpdate` error branch | `f.Close()` → `removeFile(filePath)` → `errors.Join` | `release/update.go:146-160` ordering | WIRED | Mirrors `self_update.go`'s correct idiom. |
| `.github/workflows/go.yml` step | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/` | matrix job on ubuntu/macos/windows | WIRED | Remote logs show both tests executing and passing on all three OSes. |
| `TestMain` env intercept | `helperMain()` → provider children / `crashMain()` | `exec.Command(os.Args[0])` re-exec | WIRED | Untagged lane lists zero `TestTwoProcess` tests. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `MGetFunc` | result map | real chunked `QueryContext` against `cache_entries`; no static fallback | Yes | ✓ FLOWING |
| `MSetFunc` | rows | real prepared `ExecContext` per pair inside one transaction | Yes | ✓ FLOWING |
| `MDelFunc` | deletions | real chunked `ExecContext` deletes | Yes | ✓ FLOWING |
| `TestTwoProcessContention` | per-role row counts / markers | rows written by child OS processes through the shipped provider, read back by a raw handle | Yes (287/role observed) | ✓ FLOWING |
| `DownloadUpdate` cleanup | `removeFile(filePath)` | production default `os.Remove`; stub only in the seam test | Yes | ✓ FLOWING |

No static returns, hardcoded literals, or mocks in the batch, e2e, or cleanup paths.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Both two-process arms (CONC-01/CONC-05) | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -v` | `TestTwoProcessContention` PASS (0.02s), `TestTwoProcessCrashRecovery` PASS (0.03s) | ✓ PASS |
| Batch + contention matrix (BATCH-01..03, CONC-02) | `go test ./cache/sqlite/ -run 'TestMGetFunc\|TestMSetFunc\|TestMSetNoPartialWrites\|TestMDelFunc\|TestUniqueKeys\|TestChunksOf\|TestBatchMGet\|TestBatchMSetOneTTL\|TestBatchMDel\|TestBatchSemantics\|TestBeginContentionBounded' -count=1 -v` | all PASS; `TestBeginContentionBounded` 5.11s (one budget), 0-or-2000 atomicity holds | ✓ PASS |
| Race cleanliness (CONC-03) | `go test -race ./cache/sqlite/ -run 'TestConcurrentBatchOpsRace' -count=1` | ok 2.27s, file + memory subtests PASS, no `DATA RACE` | ✓ PASS |
| Untagged package suite (no regressions) | `go test ./cache/sqlite/ -count=1` | ok 5.23s | ✓ PASS |
| E2E lane isolation (prohibition) | `go test ./cache/sqlite/ -list 'TestTwoProcess'` vs `-tags=e2e -list ...` | untagged: none; tagged: both names | ✓ PASS |
| Release / G-16-1 | `go test ./release/ -run 'TestDownloadUpdate' -count=1` (+ full `./release/` earlier) | ok 3.20s; 6/6 DownloadUpdate tests PASS | ✓ PASS |
| Coverage gate (no override) | `make coverage-quick` | PASS — total 82.4%, sqlite 92.7%, release 81.3%; thresholds 70/80/75 satisfied | ✓ PASS |
| Benchmark smoke (closeout-added CI step) | `go test -run='^$' -bench=. -benchtime=1x ./cache/...` | all packages ok | ✓ PASS |
| Conformance E2E local providers (closeout-added CI step) | `go test -tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)' ./cache/ -count=1 -timeout=120s` | ok 12.97s | ✓ PASS |
| Windows cross-build (CGO-free) | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./release/ ./cache/sqlite/` | OK | ✓ PASS |

*Tooling note (informational):* the first `make coverage-quick` invocation in this session failed inside `go-test-coverage` with `could not find file [elease/release.go]` while every package line reported `ok`. A rerun of `make coverage-quick` and a direct `go-test-coverage --profile=<fresh profile>` run both passed with the same config and totals (82.4%), and the captured profile contained no malformed paths — treated as a transient tool read, not a code or gate failure. Recorded for auditability.

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| Conventional probes | `find scripts -path '*/tests/probe-*.sh'` | none in this repo's convention | SKIPPED (not applicable) |
| 16-05 RED evidence | `16-05-RED-EVIDENCE-task2.json` | present; exitCode 1 against the pre-fix "seam never consulted" shape | PASS (evidence artifact) |
| 16-03 RED evidence | `16-03-RED-EVIDENCE-task{1,2}.json` | present (evidence artifacts, not runnable probes) | PASS (evidence artifact) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| BATCH-01 | 16-03 | Chunked `MGet`, silent skips for missing/expired/undecodable, bounded chunk size | ✓ SATISFIED | `batch.go:78-152`; `TestMGetFunc` (8 subtests) + `TestBatchMGet` + `TestBatchSemantics` PASS |
| BATCH-02 | 16-03 | Single-transaction prepared-upsert `MSet`, atomic, one TTL per batch | ✓ SATISFIED | `batch.go:157-222`; `TestMSetNoPartialWrites` (0-or-2000) + `TestMSetFunc` + `TestBatchMSetOneTTL` PASS |
| BATCH-03 | 16-03 | Chunked `MDel`, idempotent for missing/duplicate keys | ✓ SATISFIED | `batch.go:224-244`; `TestMDelFunc` (6 subtests, 250-key chunking) + `TestBatchMDel` PASS |
| CONC-01 | 16-01/16-02/16-04 | Two processes share one file without corruption or hard failure under contention | ✓ SATISFIED | Provider-driven E2E PASS locally and on 3-OS CI; spike record; bounded open retry closes DI-15-01 |
| CONC-02 | 16-02 | Immediate lock mode so contention surfaces at BEGIN; bounded by `busy_timeout`; no unbounded retry | ✓ SATISFIED | `_txlock=immediate` in both DSNs; `TestBeginContentionBounded` (busy-classified, < 2 budgets, recovery); single `retryBusy` call site |
| CONC-03 | 16-03/16-04 | `-race` clean on the pinned single-connection pool | ✓ SATISFIED | Local `-race` PASS; CI race step green on ubuntu/macOS with `CGO_ENABLED=1`, skipped on Windows |
| CONC-04 | 16-04 | Docs state WAL local-storage/same-host constraints | ✓ SATISFIED | `doc.go` "Data and state" paragraph |
| CONC-05 | 16-04/16-05 | CI E2E spawns two OS processes on one temp DB and asserts no corruption + correct behavior | ✓ SATISFIED | Workflow step wired without OS gate; remote windows log shows both tests running and passing; G-16-1 Windows reconciliation observed via green "Run tests" leg |

**Orphaned requirements:** none. All eight IDs appear in at least one plan's `requirements:` frontmatter (16-01: CONC-01; 16-02: CONC-01, CONC-02; 16-03: BATCH-01..03, CONC-03; 16-04: CONC-01, CONC-03, CONC-04, CONC-05; 16-05: CONC-05) and all are marked Complete in `REQUIREMENTS.md` (lines 113-120). QUAL-01..04 map to Phase 17 and are out of scope.

### External CI Evidence (the formerly deferred human item, now observed)

The prior report deferred "remote 3-OS matrix + G-16-1 Windows reconciliation" to a future push. That push happened; the evidence is directly observable, not inferred:

- **Run 40493064377** — `Go tests and checking`, push to `main` (17.1 merge), 2026-10-27T11:02Z, conclusion **success**.
  - `Run tests (ubuntu-latest)`, `(macos-latest)`, `(windows-latest)` — all success; steps `Build (CGO-free)`, `Run tests`, `SQLite two-process E2E`, `SQLite race detector` present in all three jobs.
  - **windows** `SQLite two-process E2E` log: `env: CGO_ENABLED: 0`, then `=== RUN TestTwoProcessContention` → `--- PASS (0.11s)`, `=== RUN TestTwoProcessCrashRecovery` → `--- PASS (0.12s)`, `ok github.com/guionardo/go/cache/sqlite 0.258s`.
  - **windows** `Run tests` step: **success** — this is the G-16-1 reconciliation (the pre-fix `require.NoFileExists` failure signature is gone).
  - **ubuntu / macOS** `SQLite race detector`: `go test -race ./cache/sqlite/ -count=1` with `CGO_ENABLED: 1` → `ok ... 6.161s` / `6.419s`. **windows** race step: `skipped` (documented gate).
  - `Go test coverage check` job: success.
- The Phase-16 steps in that run are byte-identical to the current tree's (`git diff` between the phase merge `ea54bd1` and HEAD shows no change to the two-process E2E or race step commands/gates; the later edits only add a job-level `CGO_ENABLED=0`, a `Build (CGO-free)` assertion step, `Benchmark smoke`, and the conformance E2E step, and pin `checkout`/`setup-go`).
- The closeout branch (`gsd/v1.8-closeout`, HEAD `cd348c3`) is not yet pushed, so its two *added* CI steps have no remote run; both were executed locally in this session and pass (see Behavioral Spot-Checks). They are not Phase 16 must-haves.

### Downstream Change Notes (why this report was stale — and what it did not weaken)

| Change | Files | Effect on Phase 16 must-haves |
| ------ | ----- | ----------------------------- |
| Phase 17 (CGO-free CI, benchmarks, docs) | `.github/workflows/go.yml`, `cache/bench_test.go`, docs | Job-level `CGO_ENABLED=0` now also covers the E2E step; the remote windows log confirms the two-process tests still run and pass under it. The race step re-enables `CGO_ENABLED=1` at step level — without that override `-race` would fail on Linux, so this is a strengthening, not a weakening. No step removed, no pattern narrowed. |
| Phase 17.1 (error contract) | `cache/sqlite/{sqlite.go,batch.go,doc.go}`, `cache/errors.go`, tests | `sqlite.ErrClosed` wraps `cache.ErrClosed`; `check()` no longer double-wraps; `MGet`'s deferred-error log splits Error (real failure) / Warn (closed). Phase 16 assertions still hold: prefix contracts (`TestMDelFunc/closed_db_handle_fail_fast_wrapped`), sentinel identity (`closed_provider_wrapped_errclosed`), and the warn-and-skip chunk policy. `doc.go` was updated consistently (MGet named as the "first operation" exception). |
| v1.8 closeout (CI + benchmark arms) | `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/cache_e2e_test.go` | Adds `Benchmark smoke` and local-provider conformance E2E steps, pins `checkout@v4`/`setup-go@v5`, fixes benchmark arms, adds a `sqlite_file` conformance row. Both new steps pass locally. The Phase-16 E2E/race steps are untouched. |

**CI note (informational, not a Phase 16 gap):** the conformance E2E step is deliberately restricted to `TestCacheE2E_(Mem|SQLite)` because the Redis/Valkey/Memcache/Postgres cases need testcontainers/Docker, which the macOS and Windows runners lack — so those providers' conformance paths are not exercised in CI. Review finding IN-05 is partially addressed: `checkout`/`setup-go` are pinned, while `vladopajic/go-test-coverage@v2` remains a floating major tag. Neither affects a Phase 16 must-have.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any phase-modified file (batch, retry, dsn, schema, sqlite, doc, e2e, release, workflow) | — | None |
| — | — | No stub returns, empty handlers, or hardcoded-empty data in the batch/cleanup paths; `removeFile = os.Remove` is a real production default swapped only in the seam test | — | None |

Debt-marker gate: clean. No unreferenced markers.

### Code-Review Findings (recorded, not verification gaps)

Open findings from `16-REVIEW.md` / `16-REVIEW-DISPOSITION.md` (pre-existing or orthogonal to this phase's must-haves):

| Finding | Severity | Disposition | Note |
| -------- | -------- | ----------- | ---- |
| WR-01: unsanitized `asset.Name` → path traversal in `DownloadUpdate` | warning | open | Pre-existing, defense-in-depth; not introduced by 16-05 |
| IN-01: redundant double-close on the error path | info | open | Intentional (close-before-remove); deferred close covers the success path |
| IN-02: `ctx` unused in `DownloadUpdate` | info | open | Pre-existing |
| IN-03: `removeFile` seam lacks the shared mutex | info | open | Race-free because the test is non-parallel (`nolint:paralleltest`) |
| IN-04: `MSet` nondeterminism on colliding stringified keys | info | open | Documented provider parity, out of scope |
| IN-05: unpinned third-party coverage action | info | open (partially addressed) | `checkout`/`setup-go` pinned; `go-test-coverage@v2` still floating |

### Advisory (New Scope, Unevidenced)

None. No new-scope blocker was raised that lacks deterministic evidence.

### Human Verification Required

None. The single item the prior report deferred — remote 3-OS CI execution including the Windows G-16-1 reconciliation — is now closed by observable GitHub Actions run evidence (run 40493064377: all three jobs success, windows E2E log shows both two-process tests executing and passing, windows `Run tests` green, race step green on ubuntu/macOS and skipped on Windows). No visual, real-time, or external-service behavior remains unobserved for this phase.

### Gaps Summary

No gaps. Every roadmap success criterion and every plan-level must-have truth is verified against the shipped code in this checkout, and the batch surface, the two-process proof, bounded contention, race cleanliness, the documentation contract, and the G-16-1 cleanup fix are all confirmed by tests that were executed in this session — not by SUMMARY claims.

- **Batch surface (BATCH-01..03):** real chunked/atomic/idempotent implementations in `batch.go`; the unit matrix and black-box semantics tests are green; no placeholders remain.
- **Two-process proof (CONC-01/CONC-05):** both provider-driven arms pass locally and on ubuntu, macOS, and Windows CI; 287-row/integrity/WAL/deleted-key assertions intact.
- **Bounded contention (CONC-02):** behavioral test proves BEGIN contention surfaces as a classified busy error within one budget with no provider-side retry; a single `retryBusy` call site.
- **Race cleanliness (CONC-03):** local `-race` clean; CI race step green on both non-Windows runners.
- **Documentation (CONC-04):** the full multi-process contract, including the canceled-MGet partial-result note and the 17.1 MGet exception, is present.
- **G-16-1 (16-05):** close-before-remove, the seam, and the joined cleanup error are in place and pinned; the Windows CI leg is green.
- **Prohibitions honored:** e2e harness invisible to the unit lane, no coverage override, no `go.mod` change, no new exported option, no caller data in SQL/DSN, no unbounded waits, no `t.Parallel` in the seam-mutating test.

---

_Verified: 2026-10-27T11:48:53Z_
_Verifier: the agent (gsd-verifier)_

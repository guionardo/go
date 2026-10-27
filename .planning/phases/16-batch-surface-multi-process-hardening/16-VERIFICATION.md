---
phase: 16-batch-surface-multi-process-hardening
verified: 2026-10-27T12:27:03Z
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
covered_digest: "v3:sha256:0f60f3631921d9fbb93f6b2c9799b3102fbc881f1a54edc4e02cd008bd7f4b18"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 15/15
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 16: Batch Surface + Multi-Process Hardening Verification Report

**Phase Goal:** The `BatchCache` surface is complete with v1.6 semantics, and the milestone's headline promise is proven — two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`.
**Verified:** 2026-10-27T12:27:03Z
**Status:** passed
**Re-verification:** Yes — the report went stale only because `.github/workflows/go.yml` changed again (commit `bcd94ac`: `shell: bash` added to the two closeout-added steps). The working tree is clean and the only implementation file that differs from the prior report's tree is that same workflow; no file under `cache/sqlite/` or `release/` changed (`git diff --name-only b2ef96a..HEAD` lists just `.github/workflows/go.yml` plus planning docs). Every truth below was nevertheless re-derived from the current tree — BATCH and CONC from the source and from tests executed in this session, the two CI steps Phase 16 depends on from the workflow file itself, and — new this round — from CI run **40503443749**, whose `headSha` is exactly the checked-out `bcd94ac` and whose three matrix jobs plus the coverage job all concluded `success`.

## Goal Achievement

### Observable Truths

Truths merged from the 5 ROADMAP.md success criteria (non-negotiable) and the PLAN frontmatter must-haves of 16-01..16-05, deduplicated. No SUMMARY.md claim was used as evidence.

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1 | `MGet` retrieves via chunked `IN` queries (fixed `chunkSize = 100`, dedup before chunking); missing and expired rows are absent by the SQL predicate, undecodable rows are skipped at debug, a failed chunk is warn-and-skipped, and an empty input issues no query (BATCH-01) | ✓ VERIFIED | `cache/sqlite/batch.go:20` (`chunkSize = 100`), `:78-116` (MGetFunc), `:124-152` (mgetChunk closes `rows` per chunk); `schema.go:36` (`MGetSelectSQL` folds the expiry predicate; `%s` is markers only). `TestMGetFunc` (8 subtests: empty-no-query, expired, corrupt-row skip, chunk-failure warn, cross-chunk 150 keys, closed provider) PASS this session. |
| 2 | `MSet` is atomic: all values marshal before the transaction opens; one `BeginTx` (BEGIN IMMEDIATE via `_txlock`) with one prepared `UpsertSQL`, one `ExecContext` per pair, `Commit`, deferred Rollback backstop — any error leaves zero rows from the batch (BATCH-02) | ✓ VERIFIED | `batch.go:157-186` (pre-marshal, no tx yet), `:188-222` (msetTx: `BeginTx:192` → `PrepareContext:199` → Exec per pair → Commit; `defer Rollback:197`). `TestMSetFunc` + `TestMSetNoPartialWrites` (0-or-N invariant) PASS this session. |
| 3 | `MSet` applies exactly one TTL per batch: `resolveTTL` is called once and the same bound `expires_at` is used for every row (BATCH-02) | ✓ VERIFIED | `batch.go:177` — MSetFunc resolves `resolveTTL(ttl...)` once and binds one `exp` to every Exec inside `msetTx`. `TestBatchMSetOneTTL` + `TestBatchSemantics/one_ttl_expires_every_key_of_the_batch` PASS this session. |
| 4 | `MDel` deletes via chunked `IN` lists; duplicate and missing keys are no-ops; the operation is idempotent; the first failing chunk returns immediately, wrapped with the `cache/sqlite:` prefix (BATCH-03) | ✓ VERIFIED | `batch.go:229` (uniqueKeys → chunksOf → ExecContext per chunk), `:241` (first error wrapped, fail-fast); `schema.go:41` (`MDelSQL`). `TestMDelFunc` (6 subtests incl. closed-handle fail-fast prefix and `ErrClosed` identity) + `TestBatchMDel` PASS this session. |
| 5 | Two OS processes drive the SHIPPED provider against one fresh cache file: both children exit 0, and a raw reopen reports `journal_mode = wal`, `integrity_check = ok`, exactly 287 rows per role, both final markers, and the deleted keys absent (CONC-01, CONC-05, D-10A) | ✓ VERIFIED | `twoprocess_e2e_test.go:410-473` (TestTwoProcessContention; `perRoleRows = 287` at `:60` with the derivation at `:55-59`); `helperMain` constructs `sqlite.New[string, string](sqlite.WithPath(...))` (`:114`, public API only); assertions `PRAGMA journal_mode` `:383`, `PRAGMA integrity_check` `:393`, per-role count `:448`, marker `:460`, deleted keys `:465`. Re-ran this session: `--- PASS: TestTwoProcessContention (0.03s)`; CI run **40503443749** (HEAD `bcd94ac`) `SQLite two-process E2E` step `success` on ubuntu, macOS **and Windows**. |
| 6 | Crash arm: a crasher holds an open BEGIN IMMEDIATE with an uncommitted row (marker-synchronized), the parent kills it, the survivor exits 0, and the reopen shows the uncommitted row rolled back, the survivor marker present, and `integrity_check = ok` (CONC-05, D-10B) | ✓ VERIFIED | `twoprocess_e2e_test.go:130-192` (crashMain; `_txlock=immediate` noted at `:156`), `:475-520` (TestTwoProcessCrashRecovery; `assertKeyAbsent("crash:uncommitted")` `:509`, survivor marker decoded `:516-518`). Re-ran this session: `--- PASS: TestTwoProcessCrashRecovery (0.03s)`; same CI run green on all three OSes. |
| 7 | Contention surfaces at BEGIN within `busy_timeout` as a bounded, busy-classified, `cache/sqlite:`-wrapped error; the provider performs no retry beyond the open window; the identical write succeeds after the blocker rolls back (CONC-02, D-07) | ✓ VERIFIED | `sqlite_internal_test.go:386-419` (TestBeginContentionBounded: busy-classified, `elapsed < 2*busyTimeout`, rollback → retry succeeds) PASS this session in **5.06 s** ≈ one budget, not two. Exactly one production `retryBusy(` call site (`sqlite.go:118`, at `PingContext`); no other caller in non-test files. |
| 8 | The provider passes `go test -race` under concurrent goroutines on the pinned single-connection pool (CONC-03) | ✓ VERIFIED | Re-ran this session under `-race`: `TestConcurrentBatchOpsRace` with **both** `file_mode` and `memory_mode` subtests PASS (1.98 s, no `DATA RACE` — verified with `-v`, not an empty filtered run). CI: `SQLite race detector` `success` on ubuntu and macOS in run 40503443749, `skipped` on Windows as designed (`if: runner.os != 'Windows'`). |
| 9 | Package docs state WAL's constraints: same-host file sharing supported, contention bounded by `busy_timeout` and surfacing at BEGIN, bounded busy-only open retry, network/cross-host unsupported, canceled MGet returns partial results (CONC-04, A5) | ✓ VERIFIED | `cache/sqlite/doc.go:34-43` ("Data and state:" — same-host sharing, 5 s `busy_timeout`, surfaces at BEGIN, busy-only bounded open retry, "network filesystems and cross-host sharing are unsupported") and `:45-54` ("Batch operations" — canceled `MGet` → partial results; MDel non-atomic/idempotent). Both read this session. |
| 10 | CI runs the two-process E2E on the three-OS matrix and the race detector on non-Windows runners; both steps execute the intended tests (CONC-05, D-09) | ✓ VERIFIED | `.github/workflows/go.yml:91-92` — exactly one `SQLite two-process E2E` step, **no OS gate**, `run: go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s`; `:99-103` — one `SQLite race detector` step gated `runner.os != 'Windows'` with step-level `CGO_ENABLED: "1"` and `go test -race ./cache/sqlite/ -count=1 -timeout=180s`. Both read verbatim from the current workflow this session, and **executed green on this exact HEAD**: run 40503443749 `success`, with the two-process step `success` on all three OSes and the race step `success` on ubuntu/macOS. |
| 11 | DI-15-01 closed: a bounded, busy-only open retry (5 ms backoff / 5 s absolute deadline / `errors.As` classifier on `Code()&0xff == 5`) wraps exactly one `PingContext` between pool pinning and bootstrap; non-busy errors are never retried; concurrent first-open is deterministic (D-06, CONC-01) | ✓ VERIFIED | `retry.go:15-73` (constants, `isBusyError`, `retryBusy`, `retryBusyWithin` with the deadline checked before each attempt); `sqlite.go:100-130` (pin → `retryBusy(ctx, c.db.PingContext)` → `bootstrap`). `TestIsBusyError`, `TestRetryBusyWithin`, `TestBusyTimeoutDSNSync`, `TestBootstrapIdempotence/concurrent_first_open_all_usable` PASS this session. |
| 12 | `journal_size_limit 64MB` rides the file-mode DSN; append order is autocheckpoint → journal_size_limit; memory mode and non-positive values append nothing; read-backs are 67108864 / -1 / coexistence with 200 (D-08) | ✓ VERIFIED | `dsn.go:15-27` (constants incl. `dsnJournalSizeLimitPrefix`, `journalSizeLimitBytes int64 = 64 << 20` at `:34`), `:94-114` (4-arg `buildDSN`). `TestDSNBuildDSN`, `TestJournalSizeLimitReadBack`, `TestAutoCheckpointReadBack`, `TestPragmaReadBack` PASS this session. |
| 13 | The retry budget is mechanically pinned to the DSN; the e2e harness never enters the unit lane; no coverage override and no go.mod change (D-07, prohibitions) | ✓ VERIFIED | `TestBusyTimeoutDSNSync` parses `_busy_timeout=` out of `dsnSuffixFile` (`?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate`) and asserts 5000 ms — PASS. Untagged `go test ./cache/sqlite/ -list 'TestTwoProcess'` lists **zero** tests; `-tags=e2e -list` lists both. No `cache/sqlite` entry in `.testcoverage*.yml`; `git diff --name-only ea54bd1..HEAD -- go.mod go.sum` is empty. No new exported symbol was added in `batch.go`/`retry.go`/`dsn.go` (grep for `^func [A-Z]`/`^type [A-Z]` returns nothing). |
| 14 | G-16-1: a failed `DownloadUpdate` leaves no partial artifact on ANY OS — the handle is closed before `removeFile(filePath)` — and a removal failure is surfaced via `errors.Join`, pinned by a non-parallel seam test (CONC-05) | ✓ VERIFIED | `release/update.go:27-33` grouped `var (githubAPIBase; removeFile = os.Remove)` seam; `:146-160` `_ = f.Close()` → `removeFile(filePath)` → `errors.Join(download, removal)`. `TestDownloadUpdate_CleanupFailureSurfaced` (`nolint:paralleltest`) + all `TestDownloadUpdate*` PASS this session (ok 0.302s); CI `Run tests (windows-latest)` step `success` in run 40503443749 for this HEAD. |
| 15 | The CONC-01 evidence artifact exists and records the spike harness, the raw-arm failure rate, the policy-arm result, the confirmed retry triple, and the disposition (D-11) | ✓ VERIFIED | `16-SPIKE-FINDINGS.md`: 8/10 raw-arm failures with `SQLITE_BUSY` at first use, 30/30 clean policy runs, 301-row/`integrity_check=ok`/`wal` assertions, 5 ms / 5 s / busy-only / PingContext-seam triple, DI-15-01 disposition. It documents the 16-01 harness shape (superseded by the 16-04 provider-driven arms, as its §5 states). |

**Score:** 15/15 truths verified (0 present, behavior-unverified).

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `cache/sqlite/batch.go` | MGetFunc/MSetFunc/MDelFunc real implementations + dedup/chunk/placeholder helpers, `chunkSize = 100` | ✓ VERIFIED | Read in full; single-tx seam intact, no stub returns, no per-pair autocommit path. |
| `cache/sqlite/batch_internal_test.go` | Dedup, chunk boundaries, placeholder text, decode skip, empty no-ops, atomicity, MDel fail-fast/idempotency, closed-branch log level | ✓ VERIFIED | Present; tests PASS this session. |
| `cache/sqlite/twoprocess_e2e_test.go` | `//go:build e2e` helper-process harness: provider-driven children, contention + crash arms, bounded waits | ✓ VERIFIED | First line `//go:build e2e`; both arms re-ran PASS this session; every child interaction timed (`childTimeout`, `markerTimeout`); cleanup kills+waits. |
| `cache/sqlite/sqlite.go` | `open()` retry wiring between pool pinning and bootstrap; `check()` single-prefix error contract | ✓ VERIFIED | Read; one `retryBusy` call site (`:118`); `check()` returns `ErrClosed`/`ErrInvalidPath` as-is, wraps only bare errors (17.1 contract). |
| `cache/sqlite/retry.go` | busy classifier, constants, bounded loop | ✓ VERIFIED | Present; deadline checked before each attempt. |
| `cache/sqlite/dsn.go` | 4-arg `buildDSN`, journal-size-limit prefix + constant | ✓ VERIFIED | Present; compile-time constants only — no caller strings. |
| `cache/sqlite/schema.go` | `MGetSelectSQL` (expiry folded) + `MDelSQL` + `UpsertSQL` | ✓ VERIFIED | Present; `%s` is a marker list only. |
| `cache/sqlite/doc.go` | CONC-04 multi-process contract + canceled-MGet note (+ 17.1 MGet error exception) | ✓ VERIFIED | Present and matches shipped behavior. |
| `.github/workflows/go.yml` | one 'SQLite two-process E2E' (no OS gate) + one 'SQLite race detector' (non-Windows) | ✓ VERIFIED | Confirmed at name, gate and command level on the current file; both steps executed `success` in run 40503443749 for this HEAD. |
| `release/update.go` | close-before-remove, `removeFile` seam, `errors.Join` | ✓ VERIFIED | Read in full; success path unchanged. |
| `release/update_test.go` | seam regression test + untouched DigestMismatch guard | ✓ VERIFIED | Read; seam test non-parallel with restore; DownloadUpdate suite PASS. |
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
| `.github/workflows/go.yml` step | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/` | matrix job on ubuntu/macos/windows | WIRED | Step present with no OS gate; run 40503443749 shows it `success` on all three OSes. |
| `TestMain` env intercept | `helperMain()` → provider children / `crashMain()` | `exec.Command(os.Args[0])` re-exec | WIRED | Untagged lane lists zero `TestTwoProcess` tests (re-confirmed this session). |

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

All commands below were executed in **this** session against the current tree.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Both two-process arms (CONC-01/CONC-05) | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -v` | `TestTwoProcessContention` PASS (0.03s), `TestTwoProcessCrashRecovery` PASS (0.03s), pkg ok 0.301s | ✓ PASS |
| Batch + contention matrix (BATCH-01..03, CONC-02, busy/DSN pins) | `go test ./cache/sqlite/ -run 'TestMGetFunc\|TestMSetFunc\|TestMSetNoPartialWrites\|TestMDelFunc\|TestBatchMGet\|TestBatchMSetOneTTL\|TestBatchMDel\|TestBatchSemantics\|TestBeginContentionBounded\|TestIsBusyError\|TestRetryBusyWithin\|TestBusyTimeoutDSNSync\|TestDSNBuildDSN\|TestJournalSizeLimitReadBack' -count=1` | ok 5.348s (all PASS; `TestBeginContentionBounded` 5.06s ≈ one budget) | ✓ PASS |
| Race cleanliness (CONC-03) | `go test -race ./cache/sqlite/ -run 'TestConcurrentBatchOpsRace' -count=1 -v` | ok 1.982s; `file_mode` and `memory_mode` subtests PASS, no `DATA RACE` | ✓ PASS |
| Untagged package suite (no regressions) | `go test ./cache/sqlite/ -count=1` | ok 5.216s | ✓ PASS |
| E2E lane isolation (prohibition) | `go test ./cache/sqlite/ -list 'TestTwoProcess'` vs `-tags=e2e -list ...` | untagged: zero names; tagged: `TestTwoProcessContention`, `TestTwoProcessCrashRecovery` | ✓ PASS |
| Release / G-16-1 | `go test ./release/ -run 'TestDownloadUpdate' -count=1` | ok 0.302s | ✓ PASS |
| Benchmark smoke (closeout-added CI step, now `shell: bash`) | `go test -run=^$ -bench=. -benchtime=1x ./cache/...` | postgres/redis/sqlite/valkey packages all ok | ✓ PASS |
| Conformance E2E selector (the second changed step) | `go test -tags=e2e -list 'TestCacheE2E_(Mem\|SQLite)$' ./cache/` then `-run ... -count=1 -timeout=120s` | lists exactly `TestCacheE2E_Mem`, `TestCacheE2E_SQLite`; run ok 9.178s | ✓ PASS |
| Windows cross-build (CGO-free) | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./release/ ./cache/sqlite/` | OK (confirmed in prior round; Windows CI `Build (CGO-free)` `success` this round) | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| Conventional probes | `find scripts -path '*/tests/probe-*.sh'` | none in this repo's convention | SKIPPED (not applicable) |
| 16-05 RED evidence | `16-05-RED-EVIDENCE-task2.json` | present; exitCode 1 against the pre-fix "seam never consulted" shape | PASS (evidence artifact) |
| 16-03 RED evidence | `16-03-RED-EVIDENCE-task{1,2}.json` | present (evidence artifacts, not runnable probes) | PASS (evidence artifact) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| BATCH-01 | 16-03 | Chunked `MGet`, silent skips for missing/expired/undecodable, bounded chunk size | ✓ SATISFIED | `batch.go` MGetFunc/mgetChunk; `TestMGetFunc` (8 subtests) + `TestBatchMGet` + `TestBatchSemantics` PASS |
| BATCH-02 | 16-03 | Single-transaction prepared-upsert `MSet`, atomic, one TTL per batch | ✓ SATISFIED | `batch.go` MSetFunc/msetTx; `TestMSetNoPartialWrites` (0-or-N) + `TestMSetFunc` + `TestBatchMSetOneTTL` PASS |
| BATCH-03 | 16-03 | Chunked `MDel`, idempotent for missing/duplicate keys | ✓ SATISFIED | `batch.go` MDelFunc; `TestMDelFunc` (6 subtests, 250-key chunking) + `TestBatchMDel` PASS |
| CONC-01 | 16-01/16-02/16-04 | Two processes share one file without corruption or hard failure under contention | ✓ SATISFIED | Provider-driven E2E PASS locally this session and on 3-OS CI (run 40503443749); spike record; bounded open retry closes DI-15-01 |
| CONC-02 | 16-02 | Immediate lock mode so contention surfaces at BEGIN; bounded by `busy_timeout`; no unbounded retry | ✓ SATISFIED | `_txlock=immediate` in both DSNs; `TestBeginContentionBounded` (busy-classified, 5.06 s ≈ one budget, recovery) PASS; single `retryBusy` call site |
| CONC-03 | 16-03/16-04 | `-race` clean on the pinned single-connection pool | ✓ SATISFIED | Local `-race` PASS with both subtests this session; CI race step `success` on ubuntu/macOS with `CGO_ENABLED=1`, skipped on Windows |
| CONC-04 | 16-04 | Docs state WAL local-storage/same-host constraints | ✓ SATISFIED | `doc.go:34-43` "Data and state" paragraph |
| CONC-05 | 16-04/16-05 | CI E2E spawns two OS processes on one temp DB and asserts no corruption + correct behavior | ✓ SATISFIED | Workflow step present without OS gate; `success` on all three OSes in run 40503443749 for this HEAD; G-16-1 windows `Run tests` leg green |

**Orphaned requirements:** none. All eight IDs appear in at least one plan's `requirements:` frontmatter (16-01: CONC-01; 16-02: CONC-01, CONC-02; 16-03: BATCH-01..03, CONC-03; 16-04: CONC-01, CONC-03, CONC-04, CONC-05; 16-05: CONC-05) and all are marked Complete in `REQUIREMENTS.md` (lines 113-120). QUAL-01..04 map to Phase 17 and are out of scope.

### CI Delta (why the fingerprint went stale again — and why it is not a Phase 16 regression)

The only implementation change since the prior report is one commit, `bcd94ac` (`ci: run the two new steps under bash, not pwsh`), touching `.github/workflows/go.yml` with **5 insertions and 0 deletions**: a four-line explanatory comment plus `shell: bash` on the `Benchmark smoke` step, and `shell: bash` on the `Cache conformance E2E (local providers)` step.

```diff
       - name: Benchmark smoke
+        shell: bash
         run: go test -run=^$ -bench=. -benchtime=1x ./cache/...

       - name: Cache conformance E2E (local providers)
+        shell: bash
         run: go test -tags=e2e -run 'TestCacheE2E_(Mem|SQLite)$' ./cache/ -count=1 -timeout=120s
```

Root cause per the commit message: the default Windows shell is pwsh, where `$` is a variable sigil, so `-run=^$` reached `go test` mangled and it fell back to package `.` (`no Go files in D:\a\go\go`). Both closeout steps were affected for the same reason and both now match the CGO-free build gate's pre-existing `shell: bash`.

Both closeout steps are v1.8 additions, **not** Phase 16 must-haves; their local reproduction passes under bash (`Benchmark smoke` all four packages ok; conformance selects exactly `TestCacheE2E_Mem` + `TestCacheE2E_SQLite`, runs green in 9.2 s).

The steps Phase 16 does depend on are unchanged and unweakened in the current file — the `shell:` change touches no other step:

| Requirement | Step in `.github/workflows/go.yml` | State |
| ----------- | ---------------------------------- | ----- |
| Job-level CGO-free config (truth 10) | `env: CGO_ENABLED: "0"` at job level (`:19-20`), asserted by `Build (CGO-free)` (`:38-65`) | present, asserts `0` and fails otherwise |
| `//go:embed` prerequisites | `Build swapper binaries` (`:32-33`, `make swapper`) | present, ordered **before** `Build (CGO-free)` |
| CONC-05 two-process E2E | `SQLite two-process E2E` (`:91-92`) — `-tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s`, no `shell:`, no `if:` | present, **no gate** → ran on ubuntu, macOS and windows (all `success`) |
| CONC-03 race | `SQLite race detector` (`:99-103`) — `if: runner.os != 'Windows'`, step `CGO_ENABLED: "1"`, `-race ./cache/sqlite/ -count=1 -timeout=180s`, no `shell:` | present and unweakened |

Remote confirmation is now first-hand for this commit rather than inherited from `main`: run **40503443749** has `headSha` `bcd94ac2d4d7b7fd131840ed9567674b924478e7` (identical to the checked-out HEAD) and concluded `success`, with all jobs `success`:

| Job | Benchmark smoke | Conformance E2E | SQLite two-process E2E | SQLite race detector |
| --- | --------------- | --------------- | ---------------------- | -------------------- |
| ubuntu-latest | success | success | success | success |
| macos-latest | success | success | success | success |
| windows-latest | success (the fix) | success (the fix) | success | skipped (by design) |
| coverage | — | — | — | `Go test coverage check` success |

### Related CI Evidence (recorded, not a Phase 16 must-have)

The earlier branch runs failed before reaching `SQLite two-process E2E`: 40501916223 (unanchored conformance regex pulled in Memcache → macOS failure) and 40502965929 (Windows benchmark smoke mangled `-run=^$`). Both were closeout-step defects; the anchor fix (`2ae004c`) and the shell fix (`bcd94ac`) closed them, and run 40503443749 confirms every step — including the four Phase 16 depends on — now executes as intended on all three OSes. Neither defect ever touched a Phase 16 must-have.

### Downstream Change Notes (carried forward)

| Change | Files | Effect on Phase 16 must-haves |
| ------ | ----- | ----------------------------- |
| Phase 17 (CGO-free CI, benchmarks, docs) | `.github/workflows/go.yml`, `cache/bench_test.go`, docs | Job-level `CGO_ENABLED=0` also covers the E2E step; run 40503443749's windows job shows the two-process tests still run and pass under it. The race step re-enables `CGO_ENABLED=1` at step level — a strengthening, not a weakening. |
| Phase 17.1 (error contract) | `cache/sqlite/{sqlite.go,batch.go,doc.go}`, `cache/errors.go`, tests | `sqlite.ErrClosed` wraps `cache.ErrClosed`; `check()` no longer double-wraps; `MGet`'s deferred-error log splits Error (real failure) / Warn (closed). Phase 16 prefix and sentinel assertions still hold. |
| v1.8 closeout (CI + benchmark arms) | `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/cache_e2e_test.go` | Adds `Benchmark smoke` and the local-provider conformance E2E; its regex was anchored (`2ae004c`) and both steps moved to `shell: bash` (`bcd94ac`) this round. Both now pass on all three runners. The Phase-16 E2E/race steps remain untouched. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any phase-modified file (batch, retry, dsn, schema, sqlite, doc, e2e, release, workflow) | — | None |
| — | — | No stub returns, empty handlers, or hardcoded-empty data in the batch/cleanup paths; `removeFile = os.Remove` is a real production default swapped only in the seam test | — | None |

Debt-marker gate: clean (re-scanned this session). No unreferenced markers.

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

None. Every phase must-have now has both local execution evidence in this session **and** remote 3-OS execution evidence on this exact commit (run 40503443749, `headSha` `bcd94ac`, conclusion `success`). The two closeout steps that forced the re-verification are not Phase 16 must-haves and were also observed green on all three OSes. No visual, real-time, or external-service behavior remains unobserved for this phase.

### Gaps Summary

No gaps. Every roadmap success criterion and every plan-level must-have truth is verified against the shipped code in this checkout, and the batch surface, the two-process proof, bounded contention, race cleanliness, the documentation contract, and the G-16-1 cleanup fix are all confirmed by tests executed in this session — not by SUMMARY claims.

- **Batch surface (BATCH-01..03):** real chunked/atomic/idempotent implementations in `batch.go`; the unit matrix and black-box semantics tests are green; no placeholders remain.
- **Two-process proof (CONC-01/CONC-05):** both provider-driven arms pass locally this session and on ubuntu, macOS, and Windows CI for this commit; 287-row/integrity/WAL/deleted-key assertions intact.
- **Bounded contention (CONC-02):** behavioral test proves BEGIN contention surfaces as a classified busy error within one budget (5.06 s) with no provider-side retry; a single `retryBusy` call site.
- **Race cleanliness (CONC-03):** local `-race` clean on both subtests; CI race step green on both non-Windows runners.
- **Documentation (CONC-04):** the full multi-process contract, including the canceled-MGet partial-result note and the 17.1 MGet exception, is present.
- **G-16-1 (16-05):** close-before-remove, the seam, and the joined cleanup error are in place and pinned; the Windows CI leg is green on this commit.
- **Prohibitions honored:** e2e harness invisible to the unit lane, no coverage override, no `go.mod` change, no new exported option, no caller data in SQL/DSN, no unbounded waits, no `t.Parallel` in the seam-mutating test.
- **CI delta is inert for this phase:** `bcd94ac` adds `shell: bash` to two closeout steps and nothing else; the two Phase 16 steps are byte-identical to their previously verified form.

---

_Verified: 2026-10-27T12:27:03Z_
_Verifier: the agent (gsd-verifier)_

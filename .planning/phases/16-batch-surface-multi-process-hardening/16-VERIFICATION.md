---
phase: 16-batch-surface-multi-process-hardening
verified: 2026-10-27T12:16:28Z
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
covered_digest: "v3:sha256:c40313b1465b05861abf7de1e49bdd2c71158ecabaff8cbbfa71075861a584c2"
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
**Verified:** 2026-10-27T12:16:28Z
**Status:** passed
**Re-verification:** Yes — the report went stale only because `.github/workflows/go.yml` changed (a one-line `-run` anchor on the closeout-added conformance step). Nothing in the Phase 16 implementation surface changed since the prior report (`git status` shows the workflow as the only modified file; HEAD is `b2ef96a`). Every truth below was nevertheless re-derived from the current tree — the BATCH and CONC verdicts from the source and from tests executed in this session, the two CI steps Phase 16 depends on from the workflow file itself.

## Goal Achievement

### Observable Truths

Truths merged from the 5 ROADMAP.md success criteria (non-negotiable) and the PLAN frontmatter must-haves of 16-01..16-05, deduplicated. No SUMMARY.md claim was used as evidence.

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1 | `MGet` retrieves via chunked `IN` queries (fixed `chunkSize = 100`, dedup before chunking); missing and expired rows are absent by the SQL predicate, undecodable rows are skipped at debug, a failed chunk is warn-and-skipped, and an empty input issues no query (BATCH-01) | ✓ VERIFIED | `cache/sqlite/batch.go:22` (`chunkSize = 100`), `:78-116` (MGetFunc), `:124-152` (mgetChunk closes `rows` per chunk); `schema.go:36` (`MGetSelectSQL` folds the expiry predicate; `%s` is markers only). `TestMGetFunc` (8 subtests: empty-no-query, expired, corrupt-row skip, chunk-failure warn, cross-chunk 150 keys, closed provider) PASS this session. |
| 2 | `MSet` is atomic: all values marshal before the transaction opens; one `BeginTx` (BEGIN IMMEDIATE via `_txlock`) with one prepared `UpsertSQL`, one `ExecContext` per pair, `Commit`, deferred Rollback backstop — any error leaves zero rows from the batch (BATCH-02) | ✓ VERIFIED | `batch.go:157-186` (pre-marshal, no tx yet), `:188-222` (msetTx: BeginTx → PrepareContext → Exec per pair → Commit; `defer Rollback`). `TestMSetFunc` + `TestMSetNoPartialWrites` (0-or-N invariant) PASS this session. |
| 3 | `MSet` applies exactly one TTL per batch: `resolveTTL` is called once and the same bound `expires_at` is used for every row (BATCH-02) | ✓ VERIFIED | `batch.go` MSetFunc resolves `resolveTTL(ttl...)` once and binds one `exp` to every Exec inside `msetTx`. `TestBatchMSetOneTTL` + `TestBatchSemantics/one_ttl_expires_every_key_of_the_batch` PASS this session. |
| 4 | `MDel` deletes via chunked `IN` lists; duplicate and missing keys are no-ops; the operation is idempotent; the first failing chunk returns immediately, wrapped with the `cache/sqlite:` prefix (BATCH-03) | ✓ VERIFIED | `batch.go:224-244` (uniqueKeys → chunksOf → ExecContext per chunk, first error wrapped); `schema.go:41` (`MDelSQL`). `TestMDelFunc` (6 subtests incl. closed-handle fail-fast prefix and `ErrClosed` identity) + `TestBatchMDel` PASS this session. |
| 5 | Two OS processes drive the SHIPPED provider against one fresh cache file: both children exit 0, and a raw reopen reports `journal_mode = wal`, `integrity_check = ok`, exactly 287 rows per role, both final markers, and the deleted keys absent (CONC-01, CONC-05, D-10A) | ✓ VERIFIED | `twoprocess_e2e_test.go:413-473` (TestTwoProcessContention; `perRoleRows = 287` with the derivation at `:52-58`); `helperMain` constructs `sqlite.New[string, string](sqlite.WithPath(...))` (public API only). Re-ran this session: `--- PASS: TestTwoProcessContention (0.04s)`; remote CI PASS on ubuntu, macOS **and Windows** (run 40493064377, conclusion `success`). |
| 6 | Crash arm: a crasher holds an open BEGIN IMMEDIATE with an uncommitted row (marker-synchronized), the parent kills it, the survivor exits 0, and the reopen shows the uncommitted row rolled back, the survivor marker present, and `integrity_check = ok` (CONC-05, D-10B) | ✓ VERIFIED | `twoprocess_e2e_test.go:135-192` (crashMain), `:475-520` (TestTwoProcessCrashRecovery; `assertKeyAbsent("crash:uncommitted")`, survivor `:final` decoded). Re-ran this session: `--- PASS: TestTwoProcessCrashRecovery (0.04s)`; remote CI PASS on all three OSes. |
| 7 | Contention surfaces at BEGIN within `busy_timeout` as a bounded, busy-classified, `cache/sqlite:`-wrapped error; the provider performs no retry beyond the open window; the identical write succeeds after the blocker rolls back (CONC-02, D-07) | ✓ VERIFIED | `sqlite_internal_test.go:386-419` (TestBeginContentionBounded: `assert.True(isBusyError(err))`, `elapsed < 2*busyTimeout`, rollback → retry succeeds) PASS this session (~5 s ≈ one budget, not two). Exactly one production `retryBusy(` call site (`sqlite.go:118`, at PingContext); no other caller in non-test files. |
| 8 | The provider passes `go test -race` under concurrent goroutines on the pinned single-connection pool (CONC-03) | ✓ VERIFIED | `TestConcurrentBatchOpsRace` (file + memory subtests) PASS this session under `-race` (1.99s, no `DATA RACE`); remote race step green on ubuntu and macOS. |
| 9 | Package docs state WAL's constraints: same-host file sharing supported, contention bounded by `busy_timeout` and surfacing at BEGIN, bounded busy-only open retry, network/cross-host unsupported, canceled MGet returns partial results (CONC-04, A5) | ✓ VERIFIED | `cache/sqlite/doc.go:34-43` ("Data and state:" — same-host sharing, 5 s `busy_timeout`, surfaces at BEGIN, busy-only bounded open retry, "network filesystems and cross-host sharing are unsupported") and `:45-54` ("Batch operations" — canceled `MGet` → partial results; MDel non-atomic/idempotent). Both read this session. |
| 10 | CI runs the two-process E2E on the three-OS matrix and the race detector on non-Windows runners; both steps execute the intended tests (CONC-05, D-09) | ✓ VERIFIED | `.github/workflows/go.yml:86-87` — exactly one `SQLite two-process E2E` step, **no OS gate**, `run: go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s`; `:94-98` — one `SQLite race detector` step gated `runner.os != 'Windows'` with step-level `CGO_ENABLED: "1"` and `go test -race ./cache/sqlite/ -count=1 -timeout=180s`. Both read verbatim from the current (modified) workflow this session; remote proof: run 40493064377 conclusion `success`. |
| 11 | DI-15-01 closed: a bounded, busy-only open retry (5 ms backoff / 5 s absolute deadline / `errors.As` classifier on `Code()&0xff == 5`) wraps exactly one `PingContext` between pool pinning and bootstrap; non-busy errors are never retried; concurrent first-open is deterministic (D-06, CONC-01) | ✓ VERIFIED | `retry.go:15-73` (constants, `isBusyError`, `retryBusy`, `retryBusyWithin` deadline checked before each attempt); `sqlite.go:100-130` (pin → `retryBusy(ctx, c.db.PingContext)` → `bootstrap`). `TestIsBusyError`, `TestRetryBusyWithin`, `TestBusyTimeoutDSNSync`, `TestBootstrapIdempotence/concurrent_first_open_all_usable` PASS this session. |
| 12 | `journal_size_limit 64MB` rides the file-mode DSN; append order is autocheckpoint → journal_size_limit; memory mode and non-positive values append nothing; read-backs are 67108864 / -1 / coexistence with 200 (D-08) | ✓ VERIFIED | `dsn.go:15-27` (constants incl. `dsnJournalSizeLimitPrefix`, `journalSizeLimitBytes = 64 << 20`), `:94-114` (4-arg `buildDSN`). `TestDSNBuildDSN`, `TestJournalSizeLimitReadBack`, `TestAutoCheckpointReadBack`, `TestPragmaReadBack` PASS this session. |
| 13 | The retry budget is mechanically pinned to the DSN; the e2e harness never enters the unit lane; no coverage override and no go.mod change (D-07, prohibitions) | ✓ VERIFIED | `TestBusyTimeoutDSNSync` parses `_busy_timeout=` out of `dsnSuffixFile` (`?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate`) and asserts 5000 ms — PASS. Untagged `go test ./cache/sqlite/ -list 'TestTwoProcess'` lists **zero** tests; `-tags=e2e -list` lists both. No `cache/sqlite` entry in `.testcoverage*.yml`; `git diff --name-only ea54bd1..HEAD -- go.mod go.sum` is empty. |
| 14 | G-16-1: a failed `DownloadUpdate` leaves no partial artifact on ANY OS — the handle is closed before `removeFile(filePath)` — and a removal failure is surfaced via `errors.Join`, pinned by a non-parallel seam test (CONC-05) | ✓ VERIFIED | `release/update.go:27-33` grouped `var (githubAPIBase; removeFile = os.Remove)` seam; `:146-160` `_ = f.Close()` → `removeFile(filePath)` → `errors.Join(download, removal)`. `TestDownloadUpdate_CleanupFailureSurfaced` (`nolint:paralleltest`) + all `TestDownloadUpdate*` PASS this session (ok 0.149s); remote windows "Run tests" step green. |
| 15 | The CONC-01 evidence artifact exists and records the spike harness, the raw-arm failure rate, the policy-arm result, the confirmed retry triple, and the disposition (D-11) | ✓ VERIFIED | `16-SPIKE-FINDINGS.md`: 8/10 raw-arm failures with `SQLITE_BUSY` at first use, 30/30 clean policy runs, 301-row/`integrity_check=ok`/`wal` assertions, 5 ms / 5 s / busy-only / PingContext-seam triple, DI-15-01 disposition. It documents the 16-01 harness shape (superseded by the 16-04 provider-driven arms, as its §5 states). |

**Score:** 15/15 truths verified (0 present, behavior-unverified).

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `cache/sqlite/batch.go` | MGetFunc/MSetFunc/MDelFunc real implementations + dedup/chunk/placeholder helpers, `chunkSize = 100` | ✓ VERIFIED | Read in full; single-tx seam intact, no stub returns, no per-pair autocommit path. |
| `cache/sqlite/batch_internal_test.go` | Dedup, chunk boundaries, placeholder text, decode skip, empty no-ops, atomicity, MDel fail-fast/idempotency, closed-branch log level | ✓ VERIFIED | Present; tests PASS this session. |
| `cache/sqlite/twoprocess_e2e_test.go` | `//go:build e2e` helper-process harness: provider-driven children, contention + crash arms, bounded waits | ✓ VERIFIED | First line `//go:build e2e`; both arms re-ran PASS this session; every child interaction timed (`childTimeout`, `markerTimeout`); cleanup kills+waits. |
| `cache/sqlite/sqlite.go` | `open()` retry wiring between pool pinning and bootstrap; `check()` single-prefix error contract | ✓ VERIFIED | Read; one `retryBusy` call site; `check()` returns `ErrClosed`/`ErrInvalidPath` as-is, wraps only bare errors (17.1 contract). |
| `cache/sqlite/retry.go` | busy classifier, constants, bounded loop | ✓ VERIFIED | Present; deadline checked before each attempt. |
| `cache/sqlite/dsn.go` | 4-arg `buildDSN`, journal-size-limit prefix + constant | ✓ VERIFIED | Present; compile-time constants only — no caller strings. |
| `cache/sqlite/schema.go` | `MGetSelectSQL` (expiry folded) + `MDelSQL` + `UpsertSQL` | ✓ VERIFIED | Present; `%s` is a marker list only. |
| `cache/sqlite/doc.go` | CONC-04 multi-process contract + canceled-MGet note (+ 17.1 MGet error exception) | ✓ VERIFIED | Present and matches shipped behavior. |
| `.github/workflows/go.yml` | one 'SQLite two-process E2E' (no OS gate) + one 'SQLite race detector' (non-Windows) | ✓ VERIFIED | Confirmed at name, gate and command level on the current (modified) file; remote steps executed as intended in run 40493064377. |
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
| `.github/workflows/go.yml` step | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/` | matrix job on ubuntu/macos/windows | WIRED | Step present with no OS gate; remote logs (run 40493064377) show both tests executing and passing on all three OSes. |
| `TestMain` env intercept | `helperMain()` → provider children / `crashMain()` | `exec.Command(os.Args[0])` re-exec | WIRED | Untagged lane lists zero `TestTwoProcess` tests (re-confirmed). |

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
| Both two-process arms (CONC-01/CONC-05) | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -count=1 -v` | `TestTwoProcessContention` PASS (0.04s), `TestTwoProcessCrashRecovery` PASS (0.04s), pkg ok 0.387s | ✓ PASS |
| Batch + contention matrix (BATCH-01..03, CONC-02, busy/DSN pins) | `go test ./cache/sqlite/ -run 'TestMGetFunc\|TestMSetFunc\|TestMSetNoPartialWrites\|TestMDelFunc\|TestBatchMGet\|TestBatchMSetOneTTL\|TestBatchMDel\|TestBatchSemantics\|TestBeginContentionBounded\|TestIsBusyError\|TestRetryBusyWithin\|TestBusyTimeoutDSNSync\|TestDSNBuildDSN\|TestJournalSizeLimitReadBack' -count=1` | ok 8.232s (all PASS) | ✓ PASS |
| Race cleanliness (CONC-03) | `go test -race ./cache/sqlite/ -run 'TestConcurrentBatchOpsRace' -count=1` | ok 1.99s, file + memory subtests PASS, no `DATA RACE` | ✓ PASS |
| Untagged package suite (no regressions) | `go test ./cache/sqlite/ -count=1` | ok 5.215s | ✓ PASS |
| E2E lane isolation (prohibition) | `go test ./cache/sqlite/ -list 'TestTwoProcess'` vs `-tags=e2e -list ...` | untagged: zero names; tagged: `TestTwoProcessContention`, `TestTwoProcessCrashRecovery` | ✓ PASS |
| Release / G-16-1 | `go test ./release/ -run 'TestDownloadUpdate' -count=1` | ok 0.149s | ✓ PASS |
| Benchmark smoke (closeout-added CI step) | `go test -run='^$' -bench=. -benchtime=1x ./cache/...` | all four packages ok | ✓ PASS |
| Conformance E2E anchor (the changed line) | `go test -tags=e2e -list 'TestCacheE2E_(Mem\|SQLite)$' ./cache/` then `go test -tags=e2e -run 'TestCacheE2E_(Mem\|SQLite)$' ./cache/ -count=1 -timeout=120s` | lists exactly `TestCacheE2E_Mem`, `TestCacheE2E_SQLite`; run ok 9.204s | ✓ PASS |
| Windows cross-build (CGO-free) | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./release/ ./cache/sqlite/` | OK | ✓ PASS |

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
| CONC-01 | 16-01/16-02/16-04 | Two processes share one file without corruption or hard failure under contention | ✓ SATISFIED | Provider-driven E2E PASS locally this session and on 3-OS CI (run 40493064377); spike record; bounded open retry closes DI-15-01 |
| CONC-02 | 16-02 | Immediate lock mode so contention surfaces at BEGIN; bounded by `busy_timeout`; no unbounded retry | ✓ SATISFIED | `_txlock=immediate` in both DSNs; `TestBeginContentionBounded` (busy-classified, < 2 budgets, recovery) PASS; single `retryBusy` call site |
| CONC-03 | 16-03/16-04 | `-race` clean on the pinned single-connection pool | ✓ SATISFIED | Local `-race` PASS this session; CI race step green on ubuntu/macOS with `CGO_ENABLED=1`, skipped on Windows |
| CONC-04 | 16-04 | Docs state WAL local-storage/same-host constraints | ✓ SATISFIED | `doc.go:34-43` "Data and state" paragraph |
| CONC-05 | 16-04/16-05 | CI E2E spawns two OS processes on one temp DB and asserts no corruption + correct behavior | ✓ SATISFIED | Workflow step present without OS gate; remote windows log shows both tests running and passing; G-16-1 Windows reconciliation via green "Run tests" leg |

**Orphaned requirements:** none. All eight IDs appear in at least one plan's `requirements:` frontmatter (16-01: CONC-01; 16-02: CONC-01, CONC-02; 16-03: BATCH-01..03, CONC-03; 16-04: CONC-01, CONC-03, CONC-04, CONC-05; 16-05: CONC-05) and all are marked Complete in `REQUIREMENTS.md` (lines 113-120). QUAL-01..04 map to Phase 17 and are out of scope.

### CI Delta (why the prior fingerprint went stale — and why it is not a Phase 16 regression)

The only change in the tree since the prior report is one line in `.github/workflows/go.yml`:

```diff
-        run: go test -tags=e2e -run 'TestCacheE2E_(Mem|SQLite)' ./cache/ -count=1 -timeout=120s
+        run: go test -tags=e2e -run 'TestCacheE2E_(Mem|SQLite)$' ./cache/ -count=1 -timeout=120s
```

The unanchored alternation also matched `TestCacheE2E_Memcache`, which needs Docker and failed on runners without it. The anchor restricts the step to the two local providers. Verified this session: the anchored pattern lists exactly `TestCacheE2E_Mem` and `TestCacheE2E_SQLite` and runs green (9.2 s). This step is a v1.8 closeout addition, **not** a Phase 16 must-have.

The steps Phase 16 does depend on are unchanged and unweakened in the current file:

| Requirement | Step in `.github/workflows/go.yml` | State |
| ----------- | ---------------------------------- | ----- |
| Job-level CGO-free config (13 / truth 10) | `env: CGO_ENABLED: "0"` at job level, asserted by `Build (CGO-free)` | present, asserts `0` and fails otherwise |
| `//go:embed` prerequisites | `Build swapper binaries` (`make swapper`) | present, ordered **before** `Build (CGO-free)` |
| CONC-05 two-process E2E | `SQLite two-process E2E` — `-tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s` | present, **no `if:` gate** → runs on ubuntu, macOS and windows |
| CONC-03 race | `SQLite race detector` — `if: runner.os != 'Windows'`, step `CGO_ENABLED: "1"`, `-race ./cache/sqlite/ -count=1 -timeout=180s` | present and unweakened |

Remote history: run **40493064377** (`Go tests and checking` on `main`) has conclusion `success` and, per the prior report's log excerpts, ran both two-process arms on Windows and the race step on ubuntu/macOS. This branch's earlier run 40501916223 failed only at the now-fixed conformance step on macOS (ubuntu/windows cancelled), so the two-process E2E had not yet executed on this branch before this fix; it is judged here on the unchanged step definition plus its green `main` history, and validated locally (both arms PASS).

### Related CI Evidence (recorded, not a Phase 16 must-have)

An earlier run of this branch (40501916223) failed at the closeout-added conformance E2E on macOS before reaching `SQLite two-process E2E`; ubuntu and windows were cancelled as a result. After the anchor fix the local reproduction of that step passes, and the conformance provider restriction (Redis/Valkey/Memcache/Postgres need testcontainers) remains deliberate. Neither affects a Phase 16 must-have.

### Downstream Change Notes (carried forward — unchanged this session)

| Change | Files | Effect on Phase 16 must-haves |
| ------ | ----- | ----------------------------- |
| Phase 17 (CGO-free CI, benchmarks, docs) | `.github/workflows/go.yml`, `cache/bench_test.go`, docs | Job-level `CGO_ENABLED=0` also covers the E2E step; the remote windows log confirms the two-process tests still run and pass under it. The race step re-enables `CGO_ENABLED=1` at step level — a strengthening, not a weakening. |
| Phase 17.1 (error contract) | `cache/sqlite/{sqlite.go,batch.go,doc.go}`, `cache/errors.go`, tests | `sqlite.ErrClosed` wraps `cache.ErrClosed`; `check()` no longer double-wraps; `MGet`'s deferred-error log splits Error (real failure) / Warn (closed). Phase 16 prefix and sentinel assertions still hold. |
| v1.8 closeout (CI + benchmark arms) | `.github/workflows/go.yml`, `cache/bench_test.go`, `cache/cache_e2e_test.go` | Adds `Benchmark smoke` and the local-provider conformance E2E on top of a pinned `checkout@v4`/`setup-go@v5`; its `-run` pattern was anchored this round. Both steps pass locally. The Phase-16 E2E/race steps are untouched. |

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

None. The two items a strict reading would want observed externally are both closed: remote 3-OS CI execution (run 40493064377 conclusion `success`, with the Windows two-process log and green `Run tests` leg) and local execution of both two-process arms in this session. The branch's own CI run for the anchored conformance step remains unobserved on a runner, but that step is a closeout addition, not a Phase 16 must-have — its regex behavior was reproduced locally and it cannot mask a Phase 16 regression. No visual, real-time, or external-service behavior remains unobserved for this phase.

### Gaps Summary

No gaps. Every roadmap success criterion and every plan-level must-have truth is verified against the shipped code in this checkout, and the batch surface, the two-process proof, bounded contention, race cleanliness, the documentation contract, and the G-16-1 cleanup fix are all confirmed by tests executed in this session — not by SUMMARY claims.

- **Batch surface (BATCH-01..03):** real chunked/atomic/idempotent implementations in `batch.go`; the unit matrix and black-box semantics tests are green; no placeholders remain.
- **Two-process proof (CONC-01/CONC-05):** both provider-driven arms pass locally this session and on ubuntu, macOS, and Windows CI; 287-row/integrity/WAL/deleted-key assertions intact.
- **Bounded contention (CONC-02):** behavioral test proves BEGIN contention surfaces as a classified busy error within one budget with no provider-side retry; a single `retryBusy` call site.
- **Race cleanliness (CONC-03):** local `-race` clean; CI race step green on both non-Windows runners.
- **Documentation (CONC-04):** the full multi-process contract, including the canceled-MGet partial-result note and the 17.1 MGet exception, is present.
- **G-16-1 (16-05):** close-before-remove, the seam, and the joined cleanup error are in place and pinned; the Windows CI leg is green.
- **Prohibitions honored:** e2e harness invisible to the unit lane, no coverage override, no `go.mod` change, no new exported option, no caller data in SQL/DSN, no unbounded waits, no `t.Parallel` in the seam-mutating test.

---

_Verified: 2026-10-27T12:16:28Z_
_Verifier: the agent (gsd-verifier)_

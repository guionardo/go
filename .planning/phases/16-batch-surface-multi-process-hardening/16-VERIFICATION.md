---
phase: 16-batch-surface-multi-process-hardening
verified: 2026-10-08T23:27:32Z
status: human_needed
score: 13/13 must-haves verified
covered_files:
  - .planning/phases/16-batch-surface-multi-process-hardening/16-01-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-02-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-03-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-04-PLAN.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-01-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-02-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-03-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-04-SUMMARY.md
  - .planning/phases/16-batch-surface-multi-process-hardening/16-SPIKE-FINDINGS.md
  - cache/sqlite/batch.go
  - cache/sqlite/batch_internal_test.go
  - cache/sqlite/retry.go
  - cache/sqlite/retry_internal_test.go
  - cache/sqlite/dsn.go
  - cache/sqlite/dsn_internal_test.go
  - cache/sqlite/sqlite.go
  - cache/sqlite/sqlite_internal_test.go
  - cache/sqlite/schema.go
  - cache/sqlite/sqlite_test.go
  - cache/sqlite/twoprocess_e2e_test.go
  - cache/sqlite/doc.go
  - .github/workflows/go.yml

# not exist in this gsd-tools build (`query verification --help` lists only
# `status`), so the #4155 digest could not be generated. covered_files above
# are listed manually (ROOT-relative) rather than fabricating a digest.
behavior_unverified: 0
overrides_applied: 0
gaps: []
human_verification:
  - test: "Push the milestone branch to GitHub and open the workflow run for the 'SQLite two-process E2E' step (go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s) on ubuntu, macos, and windows, and the 'SQLite race detector' step (go test -race ./cache/sqlite/) on the non-Windows runners."
    expected: "Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows."
    why_human: "GitHub Actions is an external service — the workflow wiring and the exact commands are verified locally (both arms PASS 3/3 here), but the remote 3-OS matrix execution cannot be observed from the local repo. 16-04-SUMMARY explicitly defers CI-side confirmation to the first push."
covered_digest: "v2:sha256:1276d8ffd521ee0695666391df2b4e93cf0d427187e7d7ff4a6e389121ed8cc3"

---

# Phase 16: Batch Surface + Multi-Process Hardening Verification Report

**Phase Goal:** The `BatchCache` surface is complete with v1.6 semantics, and the milestone's headline promise is proven — two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`.
**Verified:** 2026-10-08T23:27:32Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Truths merged from ROADMAP.md success criteria (5, non-negotiable) and PLAN frontmatter must_haves (8 plan-level specifics), deduplicated.

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1   | MGet retrieves via chunked IN queries (chunkSize=100, dedup before chunking); missing/expired/undecodable silently skipped (expiry predicate in SQL, decode skip at debug, chunk failures warn-and-skip); empty input issues no query (BATCH-01) | ✓ VERIFIED | `cache/sqlite/batch.go` MGetFunc+mgetChunk; `schema.go` MGetSelectSQL (expiry folded, `%s` = markers only); unit group ran green — 21 PASS incl. TestMGetFunc (9 subtests), TestUniqueKeys, TestChunksOf, TestInPlaceholders |
| 2   | MSet is atomic: pre-marshal before BeginTx, one BEGIN IMMEDIATE transaction, one prepared UpsertSQL, one ExecContext per pair, Commit, defer Rollback backstop; any error rolls back the whole batch (0-or-all under cancellation) (BATCH-02) | ✓ VERIFIED | `batch.go` MSetFunc+msetTx; TestMSetFunc (5 subtests) + TestMSetNoPartialWrites (5 iterations, 0-or-2000 invariant) ran green in the TestMSet group |
| 3   | MSet applies exactly one TTL per batch (resolveTTL once, identical raw expires_at across rows) (BATCH-02) | ✓ VERIFIED | `batch.go` MSetFunc resolveTTL-once; TestBatchMSetOneTTL black-box (all keys expire together) in the TestMSet PASS group |
| 4   | MDel deletes via chunked IN lists; duplicate/missing keys no-ops; idempotent; first-chunk error fail-fast with `cache/sqlite:` prefix (BATCH-03) | ✓ VERIFIED | `batch.go` MDelFunc; TestMDelFunc (6 subtests) + TestBatchMDel black-box (150 keys) green |
| 5   | Two OS processes share one cache file under WAL without corruption or hard failure under contention — the phase exit criterion; children drive the SHIPPED provider (sqlite.New+WithPath), both exit 0, reopen asserts journal_mode=wal, integrity_check=ok, deleted keys absent, exactly 287 rows/role (CONC-01) | ✓ VERIFIED | `twoprocess_e2e_test.go` TestTwoProcessContention — ran 3/3 PASS with -count=3; 287 derivation independently re-verified (100 − 14 distinct deletions + 200 batch + 1 marker); spike evidence in 16-SPIKE-FINDINGS.md (raw 8/10 → policy 30/30) |
| 6   | Crash arm: kill-mid-write recovery — crasher holds open BEGIN IMMEDIATE with uncommitted row (marker-synchronized), parent kills it, survivor exits 0, reopen shows uncommitted row rolled back, survivor marker present, integrity_check ok (D-10B, CONC-05) | ✓ VERIFIED | `twoprocess_e2e_test.go` TestTwoProcessCrashRecovery — 3/3 PASS with -count=3 (behavioral run of the kill/wait/reopen sequence) |
| 7   | Contention surfaces at BEGIN within busy_timeout as a bounded, busy-classified, `cache/sqlite:`-wrapped error — no provider retry beyond the open window; recovery after the blocker rolls back (CONC-02) | ✓ VERIFIED | TestBeginContentionBounded ran PASS (5.11s — under 2×busyTimeout); comment-stripped `retryBusy(` count in sqlite.go = 1; retry.go classifier (Code()&0xff==5) + absolute-deadline loop |
| 8   | Provider passes `go test -race` under concurrent goroutines on the pinned single-connection pool, mixed batch + point ops (CONC-03) | ✓ VERIFIED | TestConcurrentBatchOpsRace ran PASS for file_mode (0.74s) + memory_mode (0.67s), no DATA RACE output |
| 9   | Package docs state WAL constraints — same-host sharing supported, contention bounded at BEGIN, bounded open retry, network/cross-host unsupported, canceled MGet returns partial results (CONC-04, A5) | ✓ VERIFIED | `cache/sqlite/doc.go` Data-and-state paragraph — 'same-host', 'partial results', bounded-open-retry and network-filesystem wording all present |
| 10  | CI E2E spawns two OS processes sharing one temp database and asserts no corruption + correct behavior; steps exact-named, race step gated `runner.os != 'Windows'` (CONC-05, D-09) | ✓ VERIFIED (wiring + local behavior) | `.github/workflows/go.yml` has exactly one 'SQLite two-process E2E' step (no OS gate) and one 'SQLite race detector' step with the non-Windows gate + rationale comment; the exact e2e command passes locally. Remote 3-OS execution is external — see Human Verification |
| 11  | DI-15-01 closed: bounded busy-only open retry (5 ms backoff / 5 s absolute deadline / busy-only classifier at the PingContext seam) wraps exactly one call between pool pinning and bootstrap; concurrent first-open deterministic (D-06) | ✓ VERIFIED | `sqlite.go` open() lines 107-118 (single retryBusy between pool pinning at :105 and bootstrap at :120); retry.go identifiers present, no modernc import; TestBootstrapIdempotence/concurrent_first_open_all_usable ran PASS ×20 (behavioral) |
| 12  | journal_size_limit 64MB rides the file-mode DSN; append order autocheckpoint→journal_size_limit; memory mode and non-positive values append nothing; read-backs 67108864 / -1 / coexistence (D-08) | ✓ VERIFIED | `dsn.go` 4-arg buildDSN + constants; TestDSNBuildDSN + TestJournalSizeLimitReadBack + TestAutoCheckpointReadBack all in the 23×PASS group; options.go untouched (no new exported option) |
| 13  | Retry budget mechanically pinned to the DSN; non-busy errors never retried; no go.mod changes; no always-on harness in the unit lane (busy-timeout sync, D-07, prohibitions) | ✓ VERIFIED | TestBusyTimeoutDSNSync PASS (busy_timeout=5000 == busyTimeout.Milliseconds()); isBusyError matrix incl. non-busy fail-fast; `git log 3ec60ce..a717aac -- go.mod go.sum` empty; untagged `go test -list 'TestTwoProcess'` lists zero tests, tagged lists both arms |

**Score:** 13/13 truths verified (0 present, behavior-unverified).

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `cache/sqlite/twoprocess_e2e_test.go` | e2e-tagged helper-process harness: provider-driven children, contention + crash arms | ✓ VERIFIED | `//go:build e2e` first line, package sqlite_test; TestMain intercept; TestTwoProcessContention + TestTwoProcessCrashRecovery; raw DSN mirror confined to parent verification + crash child; no spike retry toggle (grep for SQLITE_E2E_RETRY/retryPing/isBusyError = 0 hits) |
| `.planning/.../16-SPIKE-FINDINGS.md` | D-11 evidence record (harness shape, raw-vs-policy counts, integrity/row-count evidence, confirmed policy, disposition) | ✓ VERIFIED | All five sections present; raw arm 8/10 with 16/30 probe prior; policy arm 30/30; policy triple (5 ms / 5 s / busy-only / PingContext); cites D-11 and DI-15-01 |
| `cache/sqlite/retry.go` | busyErr, isBusyError, busyTimeout/busyBackoff, retryBusy, retryBusyWithin | ✓ VERIFIED | Read in full — matches plan; no modernc import; named constants sqliteBusyCode/Mask |
| `cache/sqlite/retry_internal_test.go` | classifier matrix, loop/deadline tests, DSN-sync test | ✓ VERIFIED | fakeError with Code(); TestIsBusyError/TestRetryBusyWithin/TestBusyTimeoutDSNSync all PASS |
| `cache/sqlite/dsn.go` | 4-arg buildDSN, journal-size-limit prefix + constant | ✓ VERIFIED | Read in full; strconv-formatted ints only; memory omits both pragmas |
| `cache/sqlite/batch.go` | MGetFunc/MSetFunc/MDelFunc real impls + helpers, chunkSize=100 | ✓ VERIFIED | Read in full; uniqueKeys/chunksOf/inPlaceholders; msetTx keeps single-tx seam (BeginTx/Prepare/Commit 1/1/1) |
| `cache/sqlite/batch_internal_test.go` | unit matrix (dedup, chunks, placeholders, decode skip, no-partial-writes, fail-fast) | ✓ VERIFIED | Ran green within the batch test groups (40 PASS lines) |
| `cache/sqlite/schema.go` | MGetSelectSQL + MDelSQL constants | ✓ VERIFIED | Read in full; expiry predicate folded in; marker-list contract documented |
| `cache/sqlite/sqlite.go` | open() retry wiring; batch placeholders removed | ✓ VERIFIED | Exactly one retryBusy call (comment-stripped count = 1); 0 declarations of MGetFunc/MSetFunc/MDelFunc (1 each in batch.go) |
| `cache/sqlite/sqlite_internal_test.go` | TestJournalSizeLimitReadBack, TestBeginContentionBounded, deterministic first-open | ✓ VERIFIED | Both tests ran PASS; concurrent first-open PASS ×20 |
| `cache/sqlite/sqlite_test.go` | black-box batch semantics + TestConcurrentBatchOpsRace | ✓ VERIFIED | TestBatchSemantics + race test both ran PASS |
| `.github/workflows/go.yml` | 'SQLite two-process E2E' (3-OS) + 'SQLite race detector' (non-Windows gate) | ✓ VERIFIED | Exact names, exact commands, gate + rationale comment present |
| `cache/sqlite/doc.go` | CONC-04 multi-process contract + canceled-MGet note | ✓ VERIFIED | Data-and-state paragraph extended; 'same-host' + 'partial results' present |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `open()` | `retryBusy` → `isBusyError` → `retryBusyWithin` | `c.db.PingContext` between pool pinning (:105) and bootstrap (:120) | WIRED | Read in sqlite.go:113; exhaustion closes handle, nils db, rides deferred initErr |
| `buildDSN` | pinned connection read-back | DSN `_pragma` keys | WIRED | TestJournalSizeLimitReadBack + TestAutoCheckpointReadBack green (67108864 / -1 / 200+67108864 coexistence) |
| `MGetFunc` | `MGetSelectSQL` | uniqueKeys → chunksOf(100) → inPlaceholders → bound args | WIRED | batch.go:84-109; rows closed per chunk (defer in mgetChunk); re-key map per chunk |
| `MSetFunc` | `UpsertSQL` | msetTx: BeginTx → PrepareContext → Exec per pair → Commit | WIRED | batch.go:182-208; zero SetFunc references (no per-pair autocommit) |
| `concreteCache.MGet/MSet/MDel` | `sqliteCache.MGetFunc/MSetFunc/MDelFunc` | frozen cacher interface (concrete_cache.go:20-22; delegation at :71-82) | WIRED | Compiled and executed by e2e children (c.MSet/c.MGet/c.MDel) |
| CI matrix job | E2E children on shared temp DB | `go test -tags=e2e -run 'TestTwoProcess'` step | WIRED | Workflow grep confirmed; exact command passes locally |
| Crash child | parent Kill+Wait → reopen assertions | marker file + `crash:uncommitted` row | WIRED | TestTwoProcessCrashRecovery behavioral run 3/3 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| MGetFunc | result map | real `QueryContext` against cache_entries (chunked IN + bound now) — no static fallback | Yes | ✓ FLOWING |
| MSetFunc | rows | real `ExecContext` prepared upsert per pair in one tx | Yes | ✓ FLOWING |
| MDelFunc | deletions | real `ExecContext` chunked deletes | Yes | ✓ FLOWING |
| E2E contention | per-role 287 rows | provider-driven children writing the shared file; parent raw reopen verified | Yes | ✓ FLOWING |

No static returns, hardcoded literals, or mocks found in the batch paths.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Two-process contention arm | `go test -tags=e2e -run 'TestTwoProcessContention' ./cache/sqlite/ -v -count=3` | 3/3 `--- PASS: TestTwoProcessContention` | ✓ PASS |
| Both e2e arms | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1` | Both arms PASS (0.05s each) | ✓ PASS |
| Unit-lane isolation | `/opt/homebrew/bin/go test -list 'TestTwoProcess' ./cache/sqlite/` (untagged vs `-tags=e2e`) | Untagged: zero tests; tagged: both arms listed | ✓ PASS |
| CONC-03 race | `go test -race ./cache/sqlite/ -run 'TestConcurrentBatchOpsRace' -v -count=1` | PASS file_mode 0.74s + memory_mode 0.67s, no DATA RACE | ✓ PASS |
| DI-15-01 deterministic first-open | `go test ./cache/sqlite/ -run 'TestBootstrapIdempotence/concurrent_first_open_all_usable' -count=20` | PASS ×20 | ✓ PASS |
| Bounded BEGIN contention | `go test ./cache/sqlite/ -run 'TestBeginContentionBounded' -v -timeout=60s` | PASS (5.11s — under 2×busy_timeout, no retry) | ✓ PASS |
| Retry/DSN/batch unit group | `go test ./cache/sqlite/ -run 'TestIsBusyError\|TestRetryBusy\|TestBusyTimeoutDSNSync\|TestDSNBuildDSN\|TestJournalSizeLimitReadBack\|TestAutoCheckpointReadBack'` | 23 `--- PASS` lines | ✓ PASS |
| Batch + semantics group | `go test ./cache/sqlite/ -run 'TestUniqueKeys\|TestChunksOf\|TestInPlaceholders\|TestMGetFunc\|TestBatchMGet\|TestMSet\|TestMDel\|TestBatchSemantics'` | 40 `--- PASS` lines | ✓ PASS |
| Full unit lane | `go test ./cache/sqlite/ -count=1` | ok (5.3s) | ✓ PASS |
| Full repo | `go test ./... -count=1` | all packages ok | ✓ PASS |
| Coverage gate | `make coverage-quick` | PASS — total 81.9%, all thresholds satisfied, no override | ✓ PASS |
| Lint/vet | `go vet ./cache/...` | clean | ✓ PASS |
| Windows cross-build | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/` | OK | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| 16-03 RED-evidence records | `16-03-RED-EVIDENCE-task1.json` / `task2.json` present in phase dir (evidence artifacts, not runnable probes) | Files exist | PASS |
| Conventional probes | `find scripts -path '*/tests/probe-*.sh'` | No probe scripts exist in this repo's convention | SKIPPED (not applicable) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| BATCH-01 | 16-03 | MGet chunked IN, skip missing/expired/undecodable | ✓ SATISFIED | batch.go + TestMGetFunc/TestBatchMGet green |
| BATCH-02 | 16-03 | MSet single tx, prepared upsert, one TTL, atomic | ✓ SATISFIED | batch.go + TestMSetFunc/TestMSetNoPartialWrites/TestBatchMSetOneTTL green |
| BATCH-03 | 16-03 | MDel chunked IN, idempotent | ✓ SATISFIED | batch.go + TestMDelFunc/TestBatchMDel green |
| CONC-01 | 16-01/16-02/16-04 | Two-process WAL sharing proven (spike + permanent E2E + in-process regression) | ✓ SATISFIED | E2E contention 3/3, first-open regression ×20, SPIKE-FINDINGS |
| CONC-02 | 16-02 | BEGIN contention bounded at busy_timeout, no unbounded auto-retry | ✓ SATISFIED | TestBeginContentionBounded PASS; single retry call site |
| CONC-03 | 16-03/16-04 | `go test -race` clean on pinned pool | ✓ SATISFIED | TestConcurrentBatchOpsRace PASS both modes; CI race step wired |
| CONC-04 | 16-04 | Docs state WAL constraints (local storage, same-host) | ✓ SATISFIED | doc.go paragraph |
| CONC-05 | 16-04 | CI E2E two OS processes on one temp DB | ✓ SATISFIED (local); remote CI pending | Workflow steps wired; exact commands pass locally; remote 3-OS run → Human Verification |

**Orphaned requirements:** none. All 8 phase requirement IDs appear in at least one plan's `requirements:` frontmatter (16-01: CONC-01; 16-02: CONC-01, CONC-02; 16-03: BATCH-01..03, CONC-03; 16-04: CONC-01, CONC-03, CONC-04, CONC-05) and all are marked Complete in REQUIREMENTS.md traceability. QUAL-01..04 map to Phase 17 — correctly out of scope.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any phase-modified file | — | None found |
| — | — | No stub returns, empty handlers, or hardcoded-empty props in batch/retry/e2e code | — | None found |

Two documented, pre-existing, out-of-scope notes (both untracked scratch, unchanged by this phase, confirmed absent from the e2e file): `internal_probe/two_process_test.go` referenced a removed crashHelper symbol (16-02/16-03/16-04 summaries); not shipped, not tracked.

### Human Verification Required

### 1. CI matrix run (external service)

**Test:** Push the milestone branch and watch the workflow run — 'SQLite two-process E2E' step on ubuntu, macos, and windows; 'SQLite race detector' on the non-Windows runners.
**Expected:** Both `TestTwoProcessContention` and `TestTwoProcessCrashRecovery` pass on all three OSes (no OS gate, D-09); the race step passes on ubuntu + macos and is skipped on windows.
**Why human:** GitHub Actions is an external service. The workflow wiring is verified in-repo and the exact commands pass locally (3/3 each arm, race clean), but remote 3-OS execution cannot be observed from the local checkout — the phase's own summary defers this to the first push.

### Gaps Summary

No gaps found. Every roadmap success criterion and every plan-level must-have truth is verified against the actual codebase with direct behavioral evidence (real test runs, not just symbol presence):

- Batch surface (BATCH-01..03): real chunked/atomic/idempotent implementations in `batch.go`, placeholders gone from `sqlite.go`, black-box + unit matrices green.
- Two-process proof (CONC-01/CONC-05): provider-driven contention arm with exact 287-row/integrity/WAL assertions and kill-mid-write crash arm both PASS repeatedly; CI steps wired exactly as planned.
- Bounded contention (CONC-02): behavioral test proves BEGIN contention surfaces as a typed busy error under 2×busy_timeout with no provider-side retry; exactly one `retryBusy` call site.
- Race cleanliness (CONC-03): `-race` PASS on both pool modes; CI race step gated correctly.
- Documentation (CONC-04): full multi-process contract in `doc.go`.
- Prohibitions all honored: e2e harness invisible to the unit lane, no coverage override, no go.mod changes, no new exported option, no caller strings in DSN/SQL, no unbounded waits.

The single human item is the remote CI 3-OS execution, which cannot be observed locally.

---

_Verified: 2026-10-08T23:27:32Z_
_Verifier: the agent (gsd-verifier)_
---
phase: 16-batch-surface-multi-process-hardening
verified: 2026-10-26T14:24:04Z
status: human_needed
score: 14/14 must-haves verified
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
covered_digest: "v3:sha256:3aca42b1ec671488584a165d48c64bee50b7719bb17fc5491908a0fbd2def265"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 13/13
  gaps_closed:
    - "G-16-1 code-side: DownloadUpdate error path closes the os.Create handle BEFORE removeFile(filePath), routes removal through a package-level removeFile = os.Remove seam in a single grouped var block, and surfaces a removal failure via errors.Join — the Windows-only partial-file leak (root cause: os.Remove on an open handle, ERROR_SHARING_VIOLATION) is eliminated in code"
  gaps_remaining: []
  regressions: []
gaps: []
human_verification:
  - test: "Push the milestone branch to GitHub and open the workflow run. Confirm (a) the windows-latest 'Run tests' leg is green — TestDownloadUpdate_DigestMismatch's require.NoFileExists (update_test.go:490) now passes because close-before-remove makes DeleteFile succeed, closing G-16-1; (b) the 'SQLite two-process E2E' step passes on ubuntu, macos, and windows (no OS gate); (c) the 'SQLite race detector' step passes on ubuntu + macos and is skipped on windows (runner.os != 'Windows')."
    expected: "windows 'Run tests' green (G-16-1 reconciled, no cascade-skip); both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; race step green on ubuntu/macos, skipped on windows."
    why_human: "GitHub Actions is an external service. The workflow wiring, the exact commands, and the close-before-remove fix are all verified in-repo (both e2e arms PASS, race clean, all 6 DownloadUpdate tests PASS locally, coverage gate green), but the remote 3-OS matrix execution — including the Windows-only assertion that reconciles G-16-1 — cannot be observed from the local checkout. 16-05-SUMMARY explicitly defers this to the next push."
---

# Phase 16: Batch Surface + Multi-Process Hardening Verification Report

**Phase Goal:** The `BatchCache` surface is complete with v1.6 semantics, and the milestone's headline promise is proven — two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`.
**Verified:** 2026-10-26T14:24:04Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (prior report 2026-10-08T23:27:32Z, status human_needed; G-16-1 code-side now closed)

## Goal Achievement

### Observable Truths

Truths merged from ROADMAP.md success criteria (5, non-negotiable) and PLAN frontmatter must_haves (8 plan-level specifics), deduplicated — plus one gap-closure truth (14) added by plan 16-05. Truths 1–13 re-verified via full-suite regression (`make coverage-quick`, 82.1% total, all packages ok) + tagged e2e arms + race spot-check.

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1   | MGet retrieves via chunked IN queries (chunkSize=100, dedup before chunking); missing/expired/undecodable silently skipped (expiry predicate in SQL, decode skip at debug, chunk failures warn-and-skip); empty input issues no query (BATCH-01) | ✓ VERIFIED | `cache/sqlite/batch.go` MGetFunc+mgetChunk; `schema.go` MGetSelectSQL (expiry folded, `%s` = markers only); regression green (full unit lane ok, coverage 92.3% sqlite) |
| 2   | MSet is atomic: pre-marshal before BeginTx, one BEGIN IMMEDIATE transaction, one prepared UpsertSQL, one ExecContext per pair, Commit, defer Rollback backstop; any error rolls back the whole batch (0-or-all under cancellation) (BATCH-02) | ✓ VERIFIED | `batch.go` MSetFunc+msetTx; TestMSetFunc + TestMSetNoPartialWrites (0-or-2000 invariant) in the green full suite |
| 3   | MSet applies exactly one TTL per batch (resolveTTL once, identical raw expires_at across rows) (BATCH-02) | ✓ VERIFIED | `batch.go` MSetFunc resolveTTL-once; TestBatchMSetOneTTL black-box green |
| 4   | MDel deletes via chunked IN lists; duplicate/missing keys no-ops; idempotent; first-chunk error fail-fast with `cache/sqlite:` prefix (BATCH-03) | ✓ VERIFIED | `batch.go` MDelFunc; TestMDelFunc + TestBatchMDel green |
| 5   | Two OS processes share one cache file under WAL without corruption or hard failure under contention — the phase exit criterion; children drive the SHIPPED provider, both exit 0, reopen asserts journal_mode=wal, integrity_check=ok, deleted keys absent, exactly 287 rows/role (CONC-01) | ✓ VERIFIED | `twoprocess_e2e_test.go` TestTwoProcessContention — re-ran PASS (0.05s) this session; 287 derivation independently re-verified in prior report |
| 6   | Crash arm: kill-mid-write recovery — crasher holds open BEGIN IMMEDIATE with uncommitted row (marker-synchronized), parent kills it, survivor exits 0, reopen shows uncommitted row rolled back, survivor marker present, integrity_check ok (D-10B, CONC-05) | ✓ VERIFIED | `twoprocess_e2e_test.go` TestTwoProcessCrashRecovery — re-ran PASS (0.03s) this session |
| 7   | Contention surfaces at BEGIN within busy_timeout as a bounded, busy-classified, `cache/sqlite:`-wrapped error — no provider retry beyond the open window; recovery after the blocker rolls back (CONC-02) | ✓ VERIFIED | TestBeginContentionBounded in the green suite; single `retryBusy(` call site in sqlite.go (prior report confirmed comment-stripped count = 1) |
| 8   | Provider passes `go test -race` under concurrent goroutines on the pinned single-connection pool, mixed batch + point ops (CONC-03) | ✓ VERIFIED | TestConcurrentBatchOpsRace — re-ran `go test -race` PASS (2.13s), no DATA RACE |
| 9   | Package docs state WAL constraints — same-host sharing supported, contention bounded at BEGIN, bounded open retry, network/cross-host unsupported, canceled MGet returns partial results (CONC-04, A5) | ✓ VERIFIED | `cache/sqlite/doc.go` Data-and-state paragraph ('same-host', 'partial results', bounded-open-retry, network-filesystem wording present) |
| 10  | CI E2E spawns two OS processes sharing one temp database and asserts no corruption + correct behavior; steps exact-named, race step gated `runner.os != 'Windows'` (CONC-05, D-09) | ✓ VERIFIED (wiring + local behavior) | `.github/workflows/go.yml` has exactly one 'SQLite two-process E2E' step (no OS gate) and one 'SQLite race detector' step with the non-Windows gate; exact e2e command passes locally. Remote 3-OS execution is external — see Human Verification |
| 11  | DI-15-01 closed: bounded busy-only open retry (5 ms backoff / 5 s absolute deadline / busy-only classifier at the PingContext seam) wraps exactly one call between pool pinning and bootstrap; concurrent first-open deterministic (D-06) | ✓ VERIFIED | `sqlite.go` open() single retryBusy between pool pinning and bootstrap; retry.go identifiers present; TestBootstrapIdempotence/concurrent_first_open_all_usable in green suite |
| 12  | journal_size_limit 64MB rides the file-mode DSN; append order autocheckpoint→journal_size_limit; memory mode and non-positive values append nothing; read-backs 67108864 / -1 / coexistence (D-08) | ✓ VERIFIED | `dsn.go` 4-arg buildDSN + constants; TestDSNBuildDSN/TestJournalSizeLimitReadBack/TestAutoCheckpointReadBack green |
| 13  | Retry budget mechanically pinned to the DSN; non-busy errors never retried; no go.mod changes; no always-on harness in the unit lane (busy-timeout sync, D-07, prohibitions) | ✓ VERIFIED | TestBusyTimeoutDSNSync + isBusyError matrix green; untagged `go test -list 'TestTwoProcess'` lists zero, tagged lists both; `git diff --name-only 6fa542d..cb2da46` contains no go.mod/go.sum |
| 14  | G-16-1 closed (code-side): a failed DownloadUpdate leaves no partial artifact on ANY OS — the os.Create handle is closed BEFORE removeFile(filePath) (Windows DeleteFile no longer hits ERROR_SHARING_VIOLATION), removal routes through the package-level `removeFile = os.Remove` seam in a single grouped var block, and a removal failure surfaces via errors.Join instead of a silent discard (CONC-05) | ✓ VERIFIED (local code + behavioral) | `release/update.go:28-33` grouped `var (githubAPIBase; removeFile = os.Remove)`; `:146-160` error branch closes `f` (`:150`) before `removeFile(filePath)` (`:152`) and joins errors (`:153-157`); `release/update_test.go:495-530` TestDownloadUpdate_CleanupFailureSurfaced (non-parallel, `nolint:paralleltest`, `require.ErrorIs(stubErr)`); all 6 DownloadUpdate tests PASS locally; TestDownloadUpdate_DigestMismatch require.NoFileExists passes (POSIX). Windows leg reconciliation is remote-only — see Human Verification |

**Score:** 14/14 truths verified (0 present, behavior-unverified).

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `cache/sqlite/twoprocess_e2e_test.go` | e2e-tagged helper-process harness: provider-driven children, contention + crash arms | ✓ VERIFIED | `//go:build e2e` first line; both arms re-ran PASS this session |
| `cache/sqlite/batch.go` | MGetFunc/MSetFunc/MDelFunc real impls + helpers, chunkSize=100 | ✓ VERIFIED | Present; single-tx seam (BeginTx/Prepare/Commit) intact |
| `cache/sqlite/retry.go` | busyErr, isBusyError, busyTimeout/busyBackoff, retryBusy, retryBusyWithin | ✓ VERIFIED | Present; named constants sqliteBusyCode/Mask |
| `cache/sqlite/dsn.go` | 4-arg buildDSN, journal-size-limit prefix + constant | ✓ VERIFIED | Present |
| `cache/sqlite/schema.go` | MGetSelectSQL + MDelSQL constants | ✓ VERIFIED | Present |
| `cache/sqlite/doc.go` | CONC-04 multi-process contract + canceled-MGet note | ✓ VERIFIED | Present |
| `.github/workflows/go.yml` | 'SQLite two-process E2E' (3-OS) + 'SQLite race detector' (non-Windows gate) | ✓ VERIFIED | Exact names confirmed; gate `runner.os != 'Windows'` at line 38 |
| `release/update.go` | DownloadUpdate error path: explicit f.Close() BEFORE removeFile; grouped-var `removeFile = os.Remove` seam; errors.Join of download + cleanup errors | ✓ VERIFIED | Read in full — matches plan; seam in single grouped var block; close-then-remove mirrors self_update.go:187-190 |
| `release/update_test.go` | TestDownloadUpdate_CleanupFailureSurfaced (non-parallel seam stub); TestDownloadUpdate_DigestMismatch untouched as Windows regression guard | ✓ VERIFIED | Read in full — new test at :495-530 (stub → require.ErrorIs(stubErr) → restore); DigestMismatch at :460-492 unchanged with require.NoFileExists at :490 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `DownloadUpdate` error branch | `f.Close()` → `removeFile(filePath)` → `errors.Join` | `release/update.go:150-157` close-then-remove ordering | WIRED | Explicit `_ = f.Close()` before `removeFile`; mirrors self_update.go:187-190 (repo's correct idiom) |
| `removeFile` seam | `os.Remove` (production default) / test stub | grouped `var` block (update.go:28-33) | WIRED | Single top-level var declaration (decorder dec-num safe); swapped only in the regression test |
| `TestDownloadUpdate_CleanupFailureSurfaced` | `removeFile` seam → `require.ErrorIs(stubErr)` | stub returns sentinel; error chain asserted | WIRED | RED evidence valid (16-05-RED-EVIDENCE-task2.json, exitCode 1 against pre-fix shape — seam never consulted) |
| `TestDownloadUpdate_DigestMismatch` | `require.NoFileExists` → Windows CI leg | close-before-remove makes DeleteFile succeed | WIRED | Windows-only reconciliation deferred to next push (external) |
| `open()` | `retryBusy` → `isBusyError` → `retryBusyWithin` | `c.db.PingContext` between pool pinning and bootstrap | WIRED | Single retry call site; prior report read-confirmed |
| `MSetFunc` | `UpsertSQL` | msetTx: BeginTx → PrepareContext → Exec per pair → Commit | WIRED | Zero SetFunc references (no per-pair autocommit) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| MGetFunc | result map | real `QueryContext` against cache_entries (chunked IN + bound now) — no static fallback | Yes | ✓ FLOWING |
| MSetFunc | rows | real `ExecContext` prepared upsert per pair in one tx | Yes | ✓ FLOWING |
| MDelFunc | deletions | real `ExecContext` chunked deletes | Yes | ✓ FLOWING |
| DownloadUpdate error path | removeFile(filePath) | real `os.Remove` (production default; stub only in test) | Yes | ✓ FLOWING |

No static returns, hardcoded literals, or mocks found in the batch paths or the download cleanup path.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| G-16-1 regression + existing tests | `go test ./release/ -run 'TestDownloadUpdate' -count=1 -v` | 6/6 PASS (incl. CleanupFailureSurfaced + DigestMismatch) | ✓ PASS |
| Full release package | `go test ./release/ -count=1` | ok (1.49s) | ✓ PASS |
| Two-process contention arm | `go test -tags=e2e -run 'TestTwoProcessContention' ./cache/sqlite/ -v -count=1` | PASS (0.05s) | ✓ PASS |
| Both e2e arms | `go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1` | Both PASS | ✓ PASS |
| CONC-03 race | `go test -race ./cache/sqlite/ -run 'TestConcurrentBatchOpsRace' -count=1` | ok, no DATA RACE (2.13s) | ✓ PASS |
| Coverage gate | `make coverage-quick` | PASS — total 82.1%, release 81.3%, all thresholds (≥70 file / ≥80 package / ≥75 total) satisfied, no override | ✓ PASS |
| Lint/vet | `go vet ./release/` + `gofmt -l release/` | vet clean, gofmt empty | ✓ PASS |
| Windows cross-build | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./release/` | OK | ✓ PASS |
| Scope discipline | `git diff --name-only 6fa542d..cb2da46` | only release/update.go + release/update_test.go (48 insertions, 5 deletions); go.mod/go.sum untouched | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| 16-05 RED evidence | `16-05-RED-EVIDENCE-task2.json` present; content shows `exitCode: 1`, target `TestDownloadUpdate_CleanupFailureSurfaced`, assertion `require.ErrorIs` failing on the pre-fix "seam never consulted" shape | RED evidence valid | PASS |
| 16-03 RED evidence | `16-03-RED-EVIDENCE-task1.json` / `task2.json` present (evidence artifacts, not runnable probes) | Files exist | PASS |
| Conventional probes | `find scripts -path '*/tests/probe-*.sh'` | No probe scripts in this repo's convention | SKIPPED (not applicable) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| BATCH-01 | 16-03 | MGet chunked IN, skip missing/expired/undecodable | ✓ SATISFIED | batch.go + TestMGetFunc/TestBatchMGet green |
| BATCH-02 | 16-03 | MSet single tx, prepared upsert, one TTL, atomic | ✓ SATISFIED | batch.go + TestMSetFunc/TestMSetNoPartialWrites/TestBatchMSetOneTTL green |
| BATCH-03 | 16-03 | MDel chunked IN, idempotent | ✓ SATISFIED | batch.go + TestMDelFunc/TestBatchMDel green |
| CONC-01 | 16-01/16-02/16-04 | Two-process WAL sharing proven (spike + permanent E2E + in-process regression) | ✓ SATISFIED | E2E contention PASS this session; SPIKE-FINDINGS |
| CONC-02 | 16-02 | BEGIN contention bounded at busy_timeout, no unbounded auto-retry | ✓ SATISFIED | TestBeginContentionBounded green; single retry call site |
| CONC-03 | 16-03/16-04 | `go test -race` clean on pinned pool | ✓ SATISFIED | TestConcurrentBatchOpsRace PASS; CI race step wired |
| CONC-04 | 16-04 | Docs state WAL constraints (local storage, same-host) | ✓ SATISFIED | doc.go paragraph |
| CONC-05 | 16-04/16-05 | CI E2E two OS processes on one temp DB **+ G-16-1 Windows leak closed** | ✓ SATISFIED (local); remote CI pending | Workflow steps wired; e2e commands pass locally; G-16-1 code fix + regression test green locally; remote 3-OS + Windows reconciliation → Human Verification |

**Orphaned requirements:** none. All 8 phase requirement IDs appear in at least one plan's `requirements:` frontmatter (16-01: CONC-01; 16-02: CONC-01, CONC-02; 16-03: BATCH-01..03, CONC-03; 16-04: CONC-01, CONC-03, CONC-04, CONC-05; 16-05: CONC-05) and all are marked Complete in REQUIREMENTS.md traceability. QUAL-01..04 map to Phase 17 — correctly out of scope.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any phase-modified file | — | None found |
| — | — | No stub returns, empty handlers, or hardcoded-empty props in batch/retry/e2e/download code | — | None found |

The `removeFile = os.Remove` seam is a real production default, not a stub — it is swapped only inside the regression test. No debt markers, no static fallbacks.

### Code-Review Findings (recorded, not verification gaps)

Open review findings from `16-REVIEW.md` / `16-REVIEW-DISPOSITION.md` (out of scope for this phase's must-haves, pre-existing or orthogonal):

| Finding | Severity | Disposition | Note |
| -------- | -------- | ----------- | ---- |
| WR-01: unsanitized `asset.Name` → path traversal in DownloadUpdate | warning | open | Pre-existing, defense-in-depth (self-update already trusts release binaries); not introduced by 16-05 |
| IN-01: redundant double-close on error path | info | open | Intentional (close-before-remove); deferred close still covers success path |
| IN-02: `ctx` unused in DownloadUpdate | info | open | Pre-existing |
| IN-03: `removeFile` seam lacks the shared `mu` | info | open | Race-free only because the test is non-parallel (documented via `nolint:paralleltest`) |
| IN-04: MSet non-determinism on colliding stringified keys | info | open | Out of scope |
| IN-05: `coverage` job pins unpinned third-party actions | info | open | Pre-existing |

None of these block the phase goal; they are advisory hardening notes, not must-have failures.

### Human Verification Required

### 1. Remote CI matrix run + G-16-1 Windows reconciliation (external service)

**Test:** Push the milestone branch and open the workflow run. Confirm three things on one push:
1. The windows-latest "Run tests" leg is **green** — `TestDownloadUpdate_DigestMismatch`'s `require.NoFileExists` (update_test.go:490) now passes because close-before-remove makes `DeleteFile` succeed (G-16-1 reconciled), and the E2E/race steps are **no longer cascade-skipped**.
2. The "SQLite two-process E2E" step passes on ubuntu, macos, and windows (both `TestTwoProcessContention` and `TestTwoProcessCrashRecovery`).
3. The "SQLite race detector" step passes on ubuntu + macos and is skipped on windows.

**Expected:** windows "Run tests" green (no leftover partial file); both e2e arms green on all three OSes; race step green on non-Windows runners.

**Why human:** GitHub Actions is an external service. The close-before-remove fix, the seam, the errors.Join surfacing, and the regression test are all verified in-repo (all 6 DownloadUpdate tests PASS locally, coverage gate green, e2e arms PASS, race clean), but the remote 3-OS matrix execution — and specifically the Windows-only assertion that reconciles G-16-1 — cannot be observed from the local checkout. 16-05-SUMMARY explicitly defers the Windows CI leg to the next push.

### Gaps Summary

No code gaps remain. Every roadmap success criterion and every plan-level must-have truth is verified against the actual codebase, and the G-16-1 gap (the one blocker surfaced by the prior verification's human CI item) is now closed in code:

- **Batch surface (BATCH-01..03):** real chunked/atomic/idempotent implementations in `batch.go`, placeholders gone from `sqlite.go`, black-box + unit matrices green.
- **Two-process proof (CONC-01/CONC-05):** provider-driven contention arm with exact 287-row/integrity/WAL assertions and kill-mid-write crash arm both PASS; CI steps wired exactly as planned.
- **Bounded contention (CONC-02):** behavioral test proves BEGIN contention surfaces as a typed busy error with no provider-side retry; single `retryBusy` call site.
- **Race cleanliness (CONC-03):** `-race` PASS; CI race step gated correctly.
- **Documentation (CONC-04):** full multi-process contract in `doc.go`.
- **G-16-1 closure (CONC-05, plan 16-05):** the Windows-only partial-download leak in `release/update.go` is fixed — explicit close-before-remove (mirroring `self_update.go:187-190`), the `removeFile` seam in a single grouped var block (decorder-clean), and `errors.Join` surfacing the removal failure — pinned by `TestDownloadUpdate_CleanupFailureSurfaced` (RED-evidence-verified against the pre-fix shape) and the untouched `TestDownloadUpdate_DigestMismatch` Windows regression guard. Scope discipline intact: only the two planned files changed, `go.mod` untouched, no new exported API, no success-path behavior change.
- **Prohibitions all honored:** e2e harness invisible to the unit lane, no coverage override, no go.mod changes, no new exported option, no caller strings in DSN/SQL, no unbounded waits, no `t.Parallel` in the seam-mutating test, no permission-based cross-platform tricks.

The single remaining item is the remote CI 3-OS execution, which cannot be observed locally — the same external-service human item as the prior report, now extended to confirm the G-16-1 Windows reconciliation on the next push.

---

_Verified: 2026-10-26T14:24:04Z_
_Verifier: the agent (gsd-verifier)_

---
phase: 16-batch-surface-multi-process-hardening
plan: 05
subsystem: release
tags: [release, download, windows, cleanup, error-join, seam, regression-test, gap-closure]

# Dependency graph
requires:
  - phase: 16-batch-surface-multi-process-hardening
    provides: the G-16-1 root-cause diagnosis (.planning/debug/ci-two-process-e2e-windows-failure.md) — os.Remove-before-close on the download error path + discarded remove error
  - phase: 16-batch-surface-multi-process-hardening
    provides: 16-04's CI two-process E2E + race steps whose windows-latest leg G-16-1 had cascade-skipped
provides:
  - DownloadUpdate error path: explicit f.Close() before removeFile(filePath), package-level removeFile = os.Remove test seam in a single grouped var block, errors.Join surfacing of the removal failure
  - TestDownloadUpdate_CleanupFailureSurfaced — seam-backed regression pinning the surfaced cleanup error (require.ErrorIs on the sentinel)
affects: [verify-work (G-16-1 reconciliation on the next push), gsd-ship, release package consumers on Windows]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens / 2 tasks, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 705
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []            # no new dependencies (go.mod untouched)
  patterns:
  - "Package-level test seam via a grouped var block: removeFile = os.Remove sits beside githubAPIBase in one var ( ... ) so decorder's dec-num check counts a single top-level var declaration (mirrors self_update.go:42-45)"
  - "Close-then-remove ordering on the download error path mirrors self_update.go:187-190 — the deferred f.Close() stays for the success path; the explicit _ = f.Close() before removal is intentional (double close's ErrClosed is ignored)"

key-files:
  created: []
  modified:
  - release/update.go
  - release/update_test.go

key-decisions:
  - "The close-before-remove + errors.Join portion was already present on the branch (landed via a prior PR); this plan completed it by adding the removeFile seam, routing the removal through the seam, and normalizing the error message to 'removing partial file %s: %w' per the plan spec — the fs.ErrNotExist guard and io/fs import were dropped to match the plan's exact error-branch shape"
  - "The removeFile seam lives in the grouped var block placed AFTER the three type declarations but BEFORE func (f optionFunc) apply — the plan's 'keep the block where it is' position sat after the apply method and tripped decorder's dec-order check once the line became 'new', so the block was moved up to satisfy type→var→func ordering"
  - "RED evidence was captured with go test -json (the installed gsd-core tdd-red-evidence classifier parses the go-test-json event stream), not the plain -v text the reference tdd.md described — machine-verified RED_EVIDENCE_OK against the pre-fix shape where removeFile is declared but never consulted"

patterns-established:
  - "Seam-consultation RED: to prove a regression test is red against 'the seam is never consulted', the committed fix is temporarily routed back through os.Remove (seam declared but unused) so the target test fails on its own require.ErrorIs assertion — compile-clean, assertion-level RED, then restored for GREEN"

requirements-completed: [CONC-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "DownloadUpdate error-path fix: explicit f.Close() before removeFile(filePath) so Windows DeleteFile no longer fails with ERROR_SHARING_VIOLATION, the package-level removeFile = os.Remove seam in a single grouped var block, and errors.Join surfacing of the removal failure instead of a silent discard"
    requirement: "CONC-05"
    verification:
      - kind: unit
        ref: "release/update_test.go#TestDownloadUpdate_DigestMismatch (require.NoFileExists on the partial path — Windows CI guard) + release/update_test.go#TestDownloadUpdate_CleanupFailureSurfaced"
        status: pass
      - kind: unit
        ref: "go test ./release/ -count=1 (whole package green); make coverage-quick PASS (total 82.1%, release 80.9%)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Regression test TestDownloadUpdate_CleanupFailureSurfaced — stubs removeFile to return a sentinel error and asserts the download-failure + sentinel chain (require.ErrorIs), non-parallel (nolint:paralleltest) because it mutates the global removeFile seam"
    requirement: "CONC-05"
    verification:
      - kind: unit
        ref: "release/update_test.go#TestDownloadUpdate_CleanupFailureSurfaced (go test -run TestDownloadUpdate_CleanupFailureSurfaced -count=1 -v — PASS; RED evidence against the pre-fix shape verified RED_EVIDENCE_OK)"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-10-26
status: complete
# Commit ledger (#3968) — measured, not narrated.
plan_head_before: 6fa542dc71749314fd40a23ad24722a213fc7e55
plan_head_after: cb2da46cc474b8629d2538ae0c0c2080b80f4bdb
commits: 2
---

# Phase 16 Plan 05: G-16-1 Close-Then-Remove + Surfaced Cleanup Error Summary

**The Windows-only partial-download leak in `DownloadUpdate` is closed: the `os.Create` handle is closed before `removeFile(filePath)` (mirroring `self_update.go`'s close-then-remove idiom, so `syscall.DeleteFile` no longer hits `ERROR_SHARING_VIOLATION`), the removal error is surfaced via `errors.Join` instead of discarded, and a seam-backed regression test pins the surfaced error chain.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-10-26T13:55:00Z (approx)
- **Completed:** 2026-10-26T14:12:14Z
- **Tasks:** 2/2
- **Files:** 0 created, 2 modified (48 insertions, 5 deletions)
- **Estimate vs actuals:** plan estimated 30000 tokens (low confidence); realized diff measures 705 estimate-tokens (chars/4) — the close-before-remove + `errors.Join` shape was already present on the branch, so the residual work (seam + message normalization + test) stayed tiny.

## Accomplishments

- **Close-then-remove (G-16-1 missing item 1):** the `asset.Download(f)` error branch closes `f` explicitly (`_ = f.Close()`) before removal, and the success-path `defer func() { _ = f.Close() }()` is unchanged — the double close on the error path is intentional and its `ErrClosed` is ignored.
- **`removeFile` test seam:** the standalone `var githubAPIBase` became a single grouped `var ( ... )` block containing both `githubAPIBase` and `removeFile = os.Remove` (one-line seam comment). The grouped shape keeps decorder's `dec-num` check green (one top-level var declaration).
- **Surfaced cleanup error (G-16-1 missing item 2):** the removal now routes through `removeFile(filePath)` and, on failure, returns `errors.Join(fmt.Errorf("download failed: %w", err), fmt.Errorf("removing partial file %s: %w", filePath, removeErr))`; on success it returns the download-error wrap exactly as before (preserving the `"download failed"` message for `TestDownloadUpdate_DigestMismatch`).
- **Regression pin:** `TestDownloadUpdate_CleanupFailureSurfaced` stubs `removeFile` to return a sentinel error and asserts `require.ErrorIs(t, err, stubErr)` alongside `require.ErrorContains(t, err, "download failed")` and `require.Empty(t, filePath)` — red against the pre-fix "seam never consulted" shape, green with the fix.
- **Scope discipline:** only `release/update.go` and `release/update_test.go` changed; `go.mod` untouched; success-path and API behavior unchanged; all five prior `DownloadUpdate` tests still pass.

## Task Commits

Each task was committed atomically:

1. **Task 1: Close the partial file before removal and surface the cleanup error** - `d81bed2` (fix)
2. **Task 2: Pin the surfaced cleanup error with a seam-backed regression test** - `cb2da46` (test)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update — docs commit)

## Files Created/Modified

- `release/update.go` (MOD) — grouped `var ( githubAPIBase; removeFile = os.Remove )` block (placed after the type declarations, before `func (f optionFunc) apply` for decorder type→var→func order); error branch closes `f` before `removeFile(filePath)` and joins the removal error; `io/fs` import dropped (the `fs.ErrNotExist` guard removed per the plan's exact shape).
- `release/update_test.go` (MOD) — `errors` import; `TestDownloadUpdate_CleanupFailureSurfaced` appended after `TestDownloadUpdate_DigestMismatch` (non-parallel, `//nolint:paralleltest` for the global `removeFile` mutation).

## Decisions Made

- The close-before-remove + `errors.Join` half of the plan was already on the branch (a prior PR had shipped it without the seam); this plan completed the seam and normalized the error message to `"removing partial file %s: %w"` exactly as the plan's Task-1 spec dictates.
- The `removeFile` seam block was moved above `func (f optionFunc) apply` because the plan's "keep it where it is" position (after the `apply` method) tripped decorder's `dec-order` check the moment the line became `new` (golangci-lint `new: true` only flags new lines, so the pre-existing position had been silently suppressed).
- RED evidence used `go test -json` (the installed classifier's `go-test-json` adapter) rather than the plain `-v` text the reference tdd.md described.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] decorder `dec-order` violation after moving the seam into a grouped var block**
- **Found during:** Task 1 (golangci-lint run before commit)
- **Issue:** The plan said to keep the grouped var block "where it is" (after the type declarations, before the funcs), but the existing `var githubAPIBase` already sat after `func (f optionFunc) apply`, so the newly-grouped block triggered `var must not be placed after func` (decorder) once the line became "new".
- **Fix:** Moved the grouped var block above `func (f optionFunc) apply` — type, type, type, var, func order — behavior-neutral.
- **Files modified:** release/update.go
- **Verification:** `golangci-lint run ./release/...` → 0 issues.
- **Committed in:** `d81bed2`

**2. [Rule 3 - Blocking] wsl_v5 whitespace in the new test**
- **Found during:** Task 2 (golangci-lint run before commit)
- **Issue:** `removeFile = func(...) { ... }` immediately followed by `defer func() { removeFile = original }()` without a blank line triggered `wsl_v5` ("too many statements above defer").
- **Fix:** Added a blank line between the assignment block and the defer, matching the `self_update_test.go` idiom.
- **Files modified:** release/update_test.go
- **Verification:** `golangci-lint run ./release/...` → 0 issues; test green.
- **Committed in:** `cb2da46`

---

**Total deviations:** 2 auto-fixed (both Rule 3 — lint accommodations, zero behavioral effect).
**Impact on plan:** None — all acceptance criteria met as written; scope discipline intact (only the two planned files changed).

## TDD Gate Compliance

Repo constraints (no empty commits; per-commit `make coverage-quick` green) override the standalone-RED-commit pattern, as in 15-01..16-03. Task 2's `tdd="true"` regression test produced in-session assertion-level RED evidence, machine-verified:

- **RED:** `release/update.go`'s error branch was temporarily routed through `os.Remove(filePath)` (seam declared but never consulted), so `TestDownloadUpdate_CleanupFailureSurfaced` failed on its own `require.ErrorIs(t, err, stubErr)` assertion — the chain contained only the download failure. Verified `RED_EVIDENCE_OK` via `gsd_run check tdd-red-evidence` on `16-05-RED-EVIDENCE-task2.json` (record: exit 1, target `TestDownloadUpdate_CleanupFailureSurfaced`, format `go-test-json`, 1 fail, 0 pass).
- **GREEN:** restored the `removeFile(filePath)` seam; the test passes and the whole `release` package is green.
- **Gate-scope check:** commits match `^(fix|test)\((0*16)-(0*5)\):` — Task 1 is the fix (the GREEN implementation), Task 2 is the test-only commit (`test(16-05)`), the repo's documented pattern.

## Issues Encountered

- None blocking. The plan's "current defect" premise was slightly stale — the close-before-remove + `errors.Join` code was already on the branch (commit `addc359`, a prior PR); the residual work was the `removeFile` seam, the message normalization, and the regression test, all completed per spec.
- The Windows CI leg that finally reconciles G-16-1 (`TestDownloadUpdate_DigestMismatch` `require.NoFileExists` on `windows-latest`) runs only on the next remote push — documented as deferred to `/gsd-verify-work` (the plan's `<verification>` "post-push, remote" clause).

## Known Stubs

None — no placeholder values, TODOs, or unwired data sources. The `removeFile = os.Remove` seam is a real production default, not a stub; it is swapped only inside the regression test.

## Threat Flags

None beyond the plan's `<threat_model>`. Enforced: T-16-12 (close-before-remove so `DeleteFile` succeeds; removal failures surfaced via `errors.Join`, never silent; `TestDownloadUpdate_DigestMismatch` + `TestDownloadUpdate_CleanupFailureSurfaced` pin both behaviors; no permission-based fallbacks), T-16-SC (no package installs — `go.mod` untouched).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- G-16-1's code-side closure is complete: close-then-remove (mirror of `self_update.go`), the surfaced cleanup error, and the seam-backed regression test are all in place and locally green.
- The remaining reconciliation is remote: the next push runs the windows-latest "Run tests" leg, which should now pass `TestDownloadUpdate_DigestMismatch`'s `require.NoFileExists` and stop the cascade-skip of the "SQLite two-process E2E" and "SQLite race detector" steps. `/gsd-verify-work` drives this on resume.
- No blockers.

---

*Phase: 16-batch-surface-multi-process-hardening*
*Completed: 2026-10-26*

## Self-Check: PASSED

- Files: `release/update.go`, `release/update_test.go`, `16-05-SUMMARY.md` all exist on disk.
- Commits: `d81bed2` (Task 1, fix), `cb2da46` (Task 2, test) both present in git history; plan range `6fa542d..cb2da46` lists exactly the two planned paths.
- Gates: `go test ./release/ -count=1` green, `go vet` clean, `gofmt -l release/` empty, `golangci-lint run ./release/...` 0 issues, `make coverage-quick` PASS (total 82.1%, release 80.9% — no override), `go.mod` untouched.
- TDD gate: RED evidence recorded and verified `RED_EVIDENCE_OK` (`16-05-RED-EVIDENCE-task2.json`, phase dir; untracked evidence artifact).

---
phase: 14-semantics-hardening-release-polish
plan: 03
subsystem: testing
tags: [release, coverage-gate, self-update, httptest, error-paths]

# Dependency graph
requires:
  - phase: 04-release-self-update-with-swapper-binary
    provides: release/update.go CheckForUpdate/DownloadUpdate (the coverage-gate target, D-09)
provides:
  - TestCheckForUpdate_NoOptionsDerivesOwnerRepo — mandatory no-options row pinning the module-derivation branch (getCurrentModule → url.Parse → words[1]/words[2])
  - seven error-path rows covering request creation, network, invalid JSON, invalid version, MkdirAll/Create ENOTDIR, digest mismatch + os.Remove cleanup
  - make coverage-quick GREEN repo-wide — release/update.go 68.9% → 95.9% (file:70 satisfied), the pre-existing known-red CLOSED test-only (Pitfall 4)
affects: [14-04-docs-audit-gate]

# Actuals (#2632) — pairs with the plan's estimate (26000 tokens)
actuals:
  tokens: 1666       # chars/4 over the realized 1-file diff (6667 chars)
  tasks: 2
  commits: 2         # MEASURED: git rev-list --count ${PLAN_HEAD_BEFORE}..HEAD (#3968)
  plan_head_before: a225f3e226584e82016e122db872bd17ada3fae0
  plan_head_after: a9e8fea3d13c52f9e9b5d7c3b89b9cdbee255a6c

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "mu.Lock() + githubAPIBase save/restore + httptest.NewServer discipline for every row mutating the package-level base URL (t.Parallel() first, then mu.Lock(); //nolint:paralleltest marker)"
    - "cross-platform error triggers only: ENOTDIR (file-as-parent, slash-in-matched-name) and closed-server URLs — never chmod (AGENTS.md Windows CI rule)"
    - "test-only coverage closure: release/update.go byte-identical to phase start; the gate closes by covering dead statements, not by editing production code (Pitfall 4)"

key-files:
  created: []
  modified:
    - release/update_test.go

key-decisions:
  - "The mandatory no-options row derives owner/repo from the module path: CheckForUpdate(ctx, \"v1.0.0\") with NO options enters getCurrentModule → debug.ReadBuildInfo Main.Path \"github.com/guionardo/go\" → url.Parse (scheme-less → Path) → words[1]=guionardo, words[2]=go; the mock handler's require.Equal on /repos/guionardo/go/releases/latest pins the derivation (T-14-12 mitigation)"
  - "Seven error-path rows pin error-not-panic on every failure branch of CheckForUpdate (91-92 request creation, 102-103 network, 112-113 decode, 117-118 version) and DownloadUpdate (130-131 MkdirAll, 136-137 Create, 141-144 digest mismatch with os.Remove cleanup)"
  - "ENOTDIR triggers + closed-server URLs replace permission-based triggers — cross-platform on all three CI OSes (T-14-11 mitigation)"
  - "Coverage arithmetic adjusted from research: 71/74 = 95.9% not 74/74 = 100% — the three remaining blocks (62-63, 67-68, 72-73) are module-derivation error sub-branches unreachable from a test binary (debug.ReadBuildInfo always returns the module's Main.Path); gate (file:70) green regardless"

patterns-established:
  - "Coverage-gate closure rows must follow the existing mu + githubAPIBase save/restore shape (update_test.go:87-124 precedent) with t.Parallel() after the lock — verified race-clean via go test -race"

requirements-completed: [ROBT-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "TestCheckForUpdate_NoOptionsDerivesOwnerRepo — CheckForUpdate with no options derives owner/repo from the module path and requests /repos/guionardo/go/releases/latest (covers update.go:59-81, closes the 70% file gate: 68.9% → 82.4% after Task 1)"
    requirement: ROBT-05
    verification:
      - kind: unit
        ref: "release/update_test.go#TestCheckForUpdate_NoOptionsDerivesOwnerRepo"
        status: pass
      - kind: other
        ref: "make coverage-quick (file threshold 70% satisfied: PASS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Seven error-path rows — request creation (space-in-host), network (closed server), invalid release JSON, invalid release version, MkdirAll ENOTDIR (file-as-parent), Create ENOTDIR (slash-in-matched-name), digest mismatch with os.Remove cleanup — every failure returns an error, never a panic (covers 91-92, 102-103, 112-113, 117-118, 130-131, 136-137, 141-144; final file coverage 71/74 = 95.9%)"
    requirement: ROBT-05
    verification:
      - kind: unit
        ref: "release/update_test.go#TestCheckForUpdate_RequestCreationError, TestCheckForUpdate_NetworkError, TestCheckForUpdate_InvalidReleaseJSON, TestCheckForUpdate_InvalidReleaseVersion, TestDownloadUpdate_MkdirAllError, TestDownloadUpdate_CreateError, TestDownloadUpdate_DigestMismatch"
        status: pass
      - kind: other
        ref: "go test -race ./release/... (69 passed, no DATA RACE)"
        status: pass
    human_judgment: false

# Metrics
duration: 8 min
completed: 2026-09-29
status: complete
---

# Phase 14 Plan 3: Release Coverage Gate Closure Summary

**No-options + seven error-path release tests close the pre-existing coverage-gate known-red (D-09): release/update.go 68.9% → 95.9%, `make coverage-quick` green repo-wide, all test-only (Pitfall 4)**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-29T10:29:54Z
- **Completed:** 2026-09-29T10:38:07Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- **TestCheckForUpdate_NoOptionsDerivesOwnerRepo** — the mandatory D-09 row: `CheckForUpdate(ctx, "v1.0.0")` with NO options enters the module-derivation branch (update.go:59-81); the mock handler asserts the exact derived path `/repos/guionardo/go/releases/latest`, pinning `getCurrentModule` → `url.Parse` → `words[1]/words[2]` (T-14-12). After this row alone the file gate was green: 61/74 = 82.4%.
- **Seven error-path rows** — request creation (space-in-host URL), network (closed httptest server), invalid release JSON, invalid release version, MkdirAll ENOTDIR (file-as-parent), Create ENOTDIR (slash-in-matched name), and digest mismatch with `os.Remove` cleanup asserted via `require.NoFileExists`. Every failure path returns an error — never a panic (T-14-10).
- **`make coverage-quick` GREEN** — the milestone's only repo-wide gate failure is CLOSED (SC4): file:70 satisfied (update.go 95.9%), pkg:80 satisfied, total 80.7%. `go test ./release/...` 69 passed, `go test -race ./release/...` clean, `go test ./...` 817 passed across 25 packages.
- **release/update.go byte-identical to phase start** — `git status --porcelain release/update.go` empty at every commit; test-only closure per Pitfall 4.
- **No chmod/permission triggers** — ENOTDIR triggers and closed-server URLs only; `grep -nE 'Chmod|os\.Chmod' release/update_test.go` empty (AGENTS.md Windows CI rule, T-14-11).

## Task Commits

Each task was committed atomically:

1. **Task 1: TestCheckForUpdate_NoOptions — the mandatory row that closes the coverage gate (D-09)** - `34e03a9` (test)
2. **Task 2: Seven error-path rows — request creation, network, invalid JSON, invalid version, MkdirAll/Create ENOTDIR, digest mismatch (100% file coverage)** - `a9e8fea` (test)

**Plan metadata:** pending docs commit

## Files Created/Modified

- `release/update_test.go` - 8 new test functions (+190 lines): `TestCheckForUpdate_NoOptionsDerivesOwnerRepo`, `TestCheckForUpdate_RequestCreationError`, `TestCheckForUpdate_NetworkError`, `TestCheckForUpdate_InvalidReleaseJSON`, `TestCheckForUpdate_InvalidReleaseVersion`, `TestDownloadUpdate_MkdirAllError`, `TestDownloadUpdate_CreateError`, `TestDownloadUpdate_DigestMismatch`; added `path/filepath` import for the ENOTDIR rows.

## Decisions Made

- **Module-derivation pinned by handler assertion** — the no-options row asserts `/repos/guionardo/go/releases/latest` inside the mock handler, so any derivation change fails the test visibly (T-14-12 mitigation).
- **ENOTDIR over chmod** — file-as-parent and slash-in-matched-name triggers are cross-platform on Linux/macOS/Windows CI; chmod semantics differ on Windows (AGENTS.md), so permission-based triggers are banned.
- **Digest mismatch asserts cleanup** — the row verifies both the error AND that the partial file was removed (`require.NoFileExists`), pinning the `os.Remove` cleanup path (141-144).
- **Coverage arithmetic adjusted from research** — see deviations.

## Deviations from Plan

### Plan-prediction adjustment (no rule — measured outcome differs from research arithmetic)

**1. Coverage reached 95.9% (71/74), not the predicted 74/74 = 100%**
- **Found during:** Task 2 verification (`go tool cover -func` on the post-Task-2 profile)
- **Issue:** The plan's research predicted the seven error rows would reach 74/74 = 100%. The measured profile shows 71/74 = 95.9%: three blocks remain uncovered — update.go:62-63 (getCurrentModule error return), 67-68 (url.Parse error return), 72-73 (short module path error return). These are error sub-branches of the module-derivation path that only fire when the module name is malformed; in a test binary `debug.ReadBuildInfo()` always returns `Main.Path = "github.com/guionardo/go"` (research-verified), so no test input can reach them without a production seam — and production changes are explicitly prohibited (Pitfall 4). The plan's statement-level arithmetic counted the derivation branch as 13 statements including these sub-branches; cover instrumentation counts them as separately uncovered.
- **Fix:** None needed — the gate contract (file:70, satisfied at 95.9%) is fully met; the plan's mandatory row and all seven error rows exist exactly as specified. The three unreachable blocks are inherent to the test-only constraint.
- **Files modified:** none
- **Verification:** `go test ./release/...` 69 passed; `make coverage-quick` PASS (all three thresholds); coverage profile audited
- **Committed in:** `a9e8fea` (part of Task 2 commit)

---

**Total deviations:** 1 plan-arithmetic adjustment (0 auto-fixes needed)
**Impact on plan:** The plan's hard contract — `make coverage-quick` green with test-only additions — is fully delivered. The 100% figure was a research prediction, not a gate threshold; the three unreachable blocks do not affect the milestone gate (SC4).

## Issues Encountered

- `go-test-coverage --config` run directly (without the Makefile target) failed with `open cover.out: no such file or directory` — the Makefile generates the profile first. Resolved by using `make coverage-quick` as the plan specifies.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 14-04 (docs + audit + gate):** the pre-existing coverage-gate known-red is CLOSED — 14-04's `make coverage-quick` gate step starts from green.
- ROBT-05 robustness contract extended to the release package's update/download error paths: every failure mode returns an error, never a panic, on all three CI OSes.
- No blockers; release/update.go untouched and ready for the final phase's verification report.

---
*Phase: 14-semantics-hardening-release-polish*
*Completed: 2026-09-29*

## Self-Check: PASSED

- [x] 14-03-SUMMARY.md exists on disk
- [x] Task 1 commit 34e03a9 exists (git log)
- [x] Task 2 commit a9e8fea exists (git log)
- [x] All acceptance criteria re-run post-commit: 1 no-options row, 7 error-path rows, 0 chmod matches, release/update.go unmodified
- [x] make coverage-quick: File 70% PASS, Package 80% PASS, Total 75% PASS (80.7%)
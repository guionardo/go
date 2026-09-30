---
phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
plan: 02
subsystem: project-probe
tags: [go, detector, gomod, registry, nil-slot, tdd]

# Dependency graph
requires:
  - phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
    provides: readmeDescription (plan 11-01) — README first-paragraph fallback consumed by detectGo (D-03)
  - phase: 10-package-foundation-api-contract-repo-cleanup
    provides: readManifest (1 MB cap + BOM strip + WR-01 gate), (ProjectData, bool) detectorFunc contract, LanguageGo constant, runDetectors/callDetector dispatch
provides:
  - detectGo(folder) (ProjectData, bool) — presence-based go.mod match (D-disc-1) wired at registry index 0
  - parseGoMod(content) (modulePath, goVersion) — strings.Fields line scanner with exact first-token match
  - 7-position nil-slot registry literal (D-11) with nil-skip in runDetectors
  - TestDetectorPositions — position-pinning contract test for phases 11-03/12/13
  - Refreshed doc.go closing paragraph naming the live Go detector
affects: [11-03 (JS/TS slot 3 + PHP slot 6), phase-12 (Python 1, Rust 4), phase-13 (C#/.NET 2, Java 5), phase-14 (ROBT-05 audit)]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens, confidence low)
actuals:
  tokens: 3978     # chars/4 over the realized diff (cd97b8a..HEAD)
  tasks: 2         # tasks completed
  commits: 2       # MEASURED: git rev-list --count cd97b8a..HEAD (#3968)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Nil-slot registry literal: 7 permanent positions with nil placeholders; runDetectors skips nil before callDetector (D-11, Pattern 1)"
    - "Hand-rolled line parser: TrimSpace + trailing-// strip + strings.Fields + exact first-token equality (Pattern 2, Pitfall 6)"
    - "Presence-based match rule: go.mod existence matches even with zero parseable directives (D-disc-1); Name/Version degrade independently (DATA-02/D-02)"

key-files:
  created:
    - project_probe/detect_go.go
    - project_probe/detect_go_test.go
  modified:
    - project_probe/registry.go
    - project_probe/registry_test.go
    - project_probe/doc.go

key-decisions:
  - "Go detector matches on go.mod presence, not parse success (D-disc-1): a garbage go.mod still yields Language=Go with folder-base Name and empty Version — the go tool itself assumes go 1.16 when the directive is absent"
  - "Registry becomes a 7-position nil-slot literal (D-11): detectGo at index 0, nil at 1/2/3/4/5/6; nil entries skipped in runDetectors before the panic-recover path"
  - "Version = raw go directive string, never normalized (D-02/DATA-03): 1.21rc1 stays 1.21rc1; toolchain lines never match the go directive (Pitfall 6)"
  - "filepath.Base fallback for Name when the module line is absent or unparseable (DATA-02/A7); never the path package (ROBT-04)"

patterns-established:
  - "Detector shape: readManifest → parse → ProjectData{Language} → DATA-02 Name chain → D-02 raw Version → DATA-04 readmeDescription chain → (data, true); every failure degrades to (ProjectData{}, false)"
  - "TDD adaptation carried: RED verified-but-uncommitted (pre-commit go test hook), RED evidence in feat commit body (Phase 10 precedent)"

requirements-completed: [DETC-02, DATA-02, DATA-04]

coverage:
  - id: D1
    description: "Go detector end-to-end through Probe — go.mod presence match, module line → Name verbatim (outer quotes stripped), go directive → Version raw, README first paragraph → Description (D-01/D-02/D-03/D-disc-1/DATA-02/DATA-04)"
    requirement: DETC-02
    verification:
      - kind: unit
        ref: "project_probe/detect_go_test.go#TestProbe_GoEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_go_test.go#TestProbe_GoBOM"
        status: pass
      - kind: unit
        ref: "project_probe/detect_go_test.go#TestProbe_GoPresence"
        status: pass
      - kind: unit
        ref: "project_probe/detect_go_test.go#TestProbe_GoNameFallback"
        status: pass
    human_judgment: false
  - id: D2
    description: "parseGoMod grammar edges — quoted/backtick modules, trailing comments, go 1.21/1.21rc1/1.21.4 verbatim, toolchain/gopher/modulex isolation, first-occurrence-wins, folder-base fallback"
    requirement: DETC-02
    verification:
      - kind: unit
        ref: "project_probe/detect_go_test.go#TestDetectGo (13 rows)"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-11 registry contract — 7-position nil-slot literal, detectGo at index 0, future slots nil, nil-skip in runDetectors"
    requirement: DETC-02
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestDetectorPositions"
        status: pass
    human_judgment: false

# Metrics
duration: 8min
completed: 2026-09-29
status: complete
commits: 2
plan_head_before: cd97b8a545f5d7f334c6a37297946b0e1a86e0e5
plan_head_after: 661d85020475c9736b2c68995b2359da475658ee
---

# Phase 11 Plan 02: Go Detector End-to-End + Registry 7-Slot Literal Summary

**Go detector (detectGo + parseGoMod) wired into the production registry at index 0 with a 7-position nil-slot literal (D-11), nil-skip dispatch, refreshed doc.go, and position-pinning test — a probe of any folder with a go.mod now returns Language=Go with module-name, raw toolchain-floor version, and README-derived description**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-29T04:14:17Z
- **Completed:** 2026-09-29T04:22:19Z
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `detectGo(folder) (ProjectData, bool)` — presence-based go.mod match (D-disc-1) via `readManifest` (D-12): Name = module line verbatim with outer quotes stripped (D-01), folder-base fallback when the module line is absent (DATA-02), Version = raw `go` directive never fabricated/normalized (D-02, toolchain floor), Description via the plan 11-01 `readmeDescription` fallback (D-03). `filepath.Base` only, never the `path` package (ROBT-04)
- `parseGoMod(content) (modulePath, goVersion)` — hand-rolled line scanner (Pattern 2): trailing `//` comment strip (A2), `strings.Fields` tokenization, exact first-token equality so `gopher 1.2`, `modulex y`, and `toolchain go1.26.4` never match (Pitfall 6), outer-quote strip on the module value only (D-01, backtick leniency A4), first occurrence wins, block form `module ( path )` intentionally unparsed (A3)
- Registry: `var detectors = []detectorFunc{detectGo, nil, nil, nil, nil, nil, nil}` — 7 permanent positions (D-11); nil entries skipped in `runDetectors` before `callDetector` so empty slots never reach the panic-recover path (Pattern 1)
- `TestDetectorPositions` pins len==7, Go non-nil at 0, slots 1/2/4/5 nil (phases 12/13), slots 3/6 nil with "plan 11-03" comments (T-11-07 tampering mitigation)
- doc.go's stale "registry is empty in this phase" paragraph refreshed to name the live Go detector and the README fallback
- 74 package tests green (60 prior + 14 new), package coverage 96.2% (gate ≥80%), `GOOS=windows` build+vet clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — Go detector end-to-end through Probe, RED→GREEN** - `fcb499d` (feat(11): implement Go detector) — RED evidence in commit body
2. **Task 2: Expansion — pin go.mod parser edges and DATA-02 folder-base fallback** - `661d850` (test(11): pin go.mod parser edges)

**Plan metadata:** pending docs commit (this SUMMARY + STATE/ROADMAP).

_Note: TDD RED is verified-but-uncommitted per the Phase 10 adaptation (pre-commit `go test ./...` hook); RED evidence recorded in the feat commit message body._

## Files Created/Modified

- `project_probe/detect_go.go` - detectGo + parseGoMod (stdlib-only: path/filepath, strings; no direct file I/O, no os/exec, no net/http)
- `project_probe/detect_go_test.go` - 4 e2e Probe rows (EndToEnd/BOM/Presence/NameFallback, t.Parallel-safe) + 13-row TestDetectGo parser-edge table
- `project_probe/registry.go` - 7-position nil-slot literal (D-11) + 2-line nil-skip in runDetectors; comment block refreshed
- `project_probe/registry_test.go` - TestDetectorPositions added (no t.Parallel); TestRunDetectors_EmptyRegistry stale comment refreshed (Pitfall 4)
- `project_probe/doc.go` - closing paragraph now names the live Go detector + readmeDescription fallback

## Decisions Made

- **Presence-based match (D-disc-1) executed as planned:** go.mod existence alone matches — `TestProbe_GoPresence` pins a garbage go.mod yielding Language=Go, folder-base Name, empty Version. This is the ecosystem-correct reading (the go tool assumes go 1.16 when the directive is absent)
- **D-11 registry as nil-slot literal:** positions are the contract; phases 11-03/12/13 edit slots in place, never reorder — `TestDetectorPositions` guards cascade precedence (T-11-07)
- **Commit scope `(11)`** — per plan acceptance criteria and Phase 10 precedent (`git log --oneline -1` checks), not `(11-02)`

## Deviations from Plan

None - plan executed exactly as written.

## TDD Gate Compliance

Plan type is `tdd`; the Phase 10 user-approved TDD adaptation governs (STATE.md decision: RED verified-but-uncommitted because the pre-commit hook runs `go test ./...` on every Go-file commit; RED evidence recorded in the feat commit message body).

| Gate | Status | Evidence |
|------|--------|----------|
| RED | ✓ (adapted) | `go test ./project_probe/... -run 'TestProbe_Go'` exited 1 before wiring: TestProbe_GoEndToEnd/GoBOM/GoPresence/GoNameFallback all failed with `expected: "Go", actual: "unknown"` (LanguageUnknown from the empty registry — a behavioral end-to-end failure through the production stack, not a compile error). Record in feat commit fcb499d body |
| GREEN | ✓ | `feat(11): implement Go detector` (fcb499d) — all 4 e2e rows + full suite pass (60 tests); `GOOS=windows go vet` clean |
| REFACTOR | — | No refactor commit needed; Task 2's `test(11): pin go.mod parser edges` (661d850) added the 13-row edge table, 74 tests green |

Note: the canonical gate regex (`^test(11-02):` / `^feat(11-02):`) does not match this plan's commit messages because the plan mandates the `(11)` scope (acceptance criteria assert `git log --oneline -1` shows `feat(11): …`); Phase 10 shipped the same scope convention.

## Issues Encountered

- The `gsd_run check tdd-red-evidence` subcommand is not available in this gsd_run build (same as plan 11-01) — RED evidence recorded per the Phase 10 adaptation (commit body + this summary), which is the plan-mandated process.
- `gsd_run query git.base-branch --is-protected` returned the base branch name ("main") rather than a boolean in this build — the executor's #3819 protected-branch assertion fell back to the five-name check; branch `gsd/v1.7-project-probe` is not protected.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `detectGo` is live at registry index 0 — plan 11-03 fills slots 3 (JS/TS) and 6 (PHP) in the same nil-slot literal without reordering; `TestDetectorPositions` asserts the intermediate state
- Go wins the cascade over JS/PHP by position 0 (DETC-01) — mixed-repo precedence is now live
- Known-red carried forward: `make coverage-quick` file threshold fails on `release/update.go` 68.9% vs 70% (pre-existing, deferred — 11-RESEARCH Pitfall 2, do NOT fix in this phase)

## Self-Check: PASSED

- [x] project_probe/detect_go.go exists (detectGo + parseGoMod, stdlib-only imports)
- [x] project_probe/detect_go_test.go exists (4 e2e Probe rows + 13-row TestDetectGo table)
- [x] project_probe/registry.go carries the 7-slot nil literal + nil-skip; registry_test.go has TestDetectorPositions (no t.Parallel)
- [x] project_probe/doc.go closing paragraph names the live Go detector
- [x] Commits exist: fcb499d (feat), 661d850 (test) — git log verified
- [x] `go test ./project_probe/...` exits 0 (74 tests); `GOOS=windows go build` + `go vet` clean
- [x] Coverage: package 96.2% (gate ≥80%), files all ≥85.7%; `make coverage-quick` total 78.4% (≥75%) with only the documented release/update.go known-red
- [x] Anti-feature greps (grep -E): `os/exec|net/http|EvalSymlinks|WalkDir`, `os\.Open|os\.ReadFile|io\.ReadAll`, bare `"path"` — all print nothing over detect_go.go
- [x] Acceptance criteria: TestProbe_GoEndToEnd asserts example.com/acme/1.26.4/"A CLI for acme."; TestProbe_GoPresence asserts folder-base Name + empty Version; TestDetectorPositions asserts len==7 with [0] non-nil and [1/2/4/5] nil

---
*Phase: 11-text-json-detectors-go-js-ts-php-readme-fallback*
*Completed: 2026-09-29*
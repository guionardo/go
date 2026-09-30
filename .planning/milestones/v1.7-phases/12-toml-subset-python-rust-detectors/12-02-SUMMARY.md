---
phase: 12-toml-subset-python-rust-detectors
plan: 02
subsystem: project-probe
tags: [python, pyproject.toml, pep621, poetry, detector, registry, tdd]

# Dependency graph
requires:
  - phase: 12-toml-subset-python-rust-detectors (12-01)
    provides: readTOMLSection — section-aware TOML-subset reader with strict degrade-to-empty and global skip states
  - phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
    provides: readManifest (1 MB cap + BOM strip + WR-01 FIFO gate), readmeDescription (DATA-04), 7-slot registry literal with nil-skip/callDetector, detectGo presence-match shape
provides:
  - detectPython(folder) (ProjectData, bool) — pyproject.toml presence-match (D-09), [project] PEP 621 fields with [tool.poetry] whole-section fallback (D-04/D-disc-3), DATA-02/DATA-04 chains (D-05)
  - Registry slot 1 live: detectPython at index 1 in place (D-08) — 4 of 7 slots live
  - TestDetectorPositions interim pin: detectors[1] NotNil, detectors[4] Nil (Pitfall 9)
  - 16-test Python fixture matrix (8 e2e probe rows + 8 edge rows incl. TestDetectPython table)
affects: [12-03 (detectRust — consumes readTOMLSection for [package]; final TestDetectorPositions flip detectors[4] NotNil; cascade precedence rows), phase 13 (C#/.NET slot 2, Java slot 5)]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens, 2 tasks, low confidence)
actuals:
  tokens: 4210        # chars/4 over the realized diff (16840 chars: detect_python.go 1386 + detect_python_test.go 13966 + registry.go/registry_test.go 1488)
  tasks: 2            # tasks completed
  commits: 2          # MEASURED: git rev-list --count 0be90b5..HEAD (#3968)
  plan_head_before: 0be90b552c36b0cd5e59b3d59c9ef21f36cf4d31
  plan_head_after: ee74217e5d14b91059660b8dd01e2c86e53b7bb2

# Tech tracking
tech-stack:
  added: [none — stdlib path/filepath only; testify existing test dep]
  patterns: [presence-match detector with whole-section fallback (detect_go.go exact analog), len==0 map guard for section-absent-or-unsupported, probe-level t.TempDir fixture rows with t.Parallel]

key-files:
  created: [project_probe/detect_python.go, project_probe/detect_python_test.go]
  modified: [project_probe/registry.go, project_probe/registry_test.go]

key-decisions:
  - "Whole-section precedence executed per D-disc-3 (flagged assumption): [project] non-empty map wins; [tool.poetry] read ONLY when len(fields)==0 — no per-field mixing across sections; pinned by TestProbe_PythonProjectWins + TestProbe_PythonEmptyProjectFallsToPoetry"
  - "Presence-match asymmetry (D-09) shipped: a garbage pyproject.toml claims Python at index 1 while JS needs parse success at index 3; cascade consequence pinned in plan 12-03's TestProbe_CascadePrecedence"
  - "dynamic = [\"version\"] has NO code branch in the detector — the reader degrades the array to an absent key, Version reads \"\" (DATA-03 never-fabricated); pinned by TestProbe_PythonDynamicVersion"
  - "Interim registry flip only (Pitfall 9): TestDetectorPositions changes exactly one assertion (detectors[1] NotNil) — detectors[4] stays Nil for plan 12-03's RED step; the flip is the ONLY registry_test.go change per the plan prohibition (doc comment left per plan)"

patterns-established:
  - "Detector shape (detect_go.go exact analog): readManifest → readTOMLSection → ProjectData{Language}; len==0 whole-section fallback; DATA-02 folder-base then DATA-04 README chains"
  - "Probe-level fixture rows with t.TempDir + os.WriteFile, t.Parallel safe (no detectors-slice mutation)"

requirements-completed: [DETC-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "detectPython — pyproject.toml presence-match with [project] PEP 621 name/version/description, [tool.poetry] legacy whole-section fallback, DATA-02/DATA-04 fallback chains (DETC-05, D-04/D-05/D-09)"
    requirement: DETC-05
    verification:
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonPoetryLegacy"
        status: pass
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonProjectWins"
        status: pass
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonPresence"
        status: pass
    human_judgment: false
  - id: D2
    description: "Registry slot 1 filled in place with the interim position pin — detectPython at index 1, detectors[4] still Nil (D-08, Pitfall 9); Python now outranks JS/TS and PHP in the cascade"
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestDetectorPositions"
        status: pass
    human_judgment: false
  - id: D3
    description: "Python edge matrix — sub-table isolation (project + poetry arms), empty-[project] fallback, BOM/CRLF tolerance, poetry inline-table degrade, poetry name-absent folder-base, empty-file presence, nested-folder base edge, poetry README description fallback"
    requirement: DETC-05
    verification:
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonOptionalDepsIsolation"
        status: pass
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonBOM"
        status: pass
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestProbe_PythonCRLF"
        status: pass
      - kind: unit
        ref: "project_probe/detect_python_test.go#TestDetectPython"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-09-29
status: complete
---

# Phase 12 Plan 2: Python Detector Summary

**detectPython wired into the production registry at index 1 — pyproject.toml presence-match with PEP 621 `[project]` fields, whole-section `[tool.poetry]` legacy fallback (D-disc-3), DATA-02 folder-base + DATA-04 README chains, and the interim TestDetectorPositions pin (Python live, Rust still nil)**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-29T05:48:41Z
- **Completed:** 2026-09-29T05:54:54Z
- **Tasks:** 2 (1 tracer RED→GREEN, 1 expansion)
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `detectPython(folder) (ProjectData, bool)` — presence-based match (D-09): `readManifest(folder, "pyproject.toml")` exclusively (1 MB cap + BOM strip + WR-01 FIFO gate inherited); `[project]` PEP 621 name/version/description primary; `len(fields)==0` guard falls back to `[tool.poetry]` (D-04) — the guard covers both "section absent" and "section present but everything unsupported" (D-disc-3, no per-field mixing); Name → `filepath.Base(folder)` when absent (DATA-02); Version raw or `""` (DATA-03 — `dynamic = ["version"]` degrades in the reader, never fabricated); Description → `readmeDescription(folder)` when empty (DATA-04). Never-fail: read failure → `(ProjectData{}, false)`.
- **Registry slot 1 filled in place** (D-08): `detectPython` replaces the nil at index 1 — Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP order preserved; `runDetectors` nil-skip and `callDetector` recover unchanged. Python now outranks JS/TS (3) and PHP (6) in the cascade.
- **TestDetectorPositions interim flip** (Pitfall 9): `detectors[1]` NotNil (Python — Phase 12), `detectors[4]` stays Nil (Rust fills in plan 12-03) — the exact mid-fill state, one assertion changed, no parallel marker added.
- **16-test Python matrix**: 8 e2e probe rows (EndToEnd, PoetryLegacy, ProjectWins, DynamicVersion, Presence, NameFallback, DescriptionFallback, MissingManifest) + 8 edge rows (OptionalDepsIsolation, PoetryDepsIsolation, EmptyProjectFallsToPoetry, BOM, CRLF, PoetryPackagesTable, PoetryNameAbsent, TestDetectPython 3-row table).
- Anti-features verified absent from detect_python.go (`grep -E`: `os/exec|net/http|EvalSymlinks|WalkDir` — nothing), no direct file I/O (`os.Open|os.ReadFile|io.ReadAll` — nothing), `path/filepath` only (bare `"path"` import — nothing).

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — Python detector end-to-end through Probe, RED→GREEN (detectPython + registry slot 1 + interim position flip)** - `4587026` (feat) — RED evidence recorded in the commit body
2. **Task 2: Expansion — pin Python detector edges (sub-table isolation, BOM/CRLF, empty-section fallback, poetry edges)** - `ee74217` (test)

**Plan metadata:** `docs(12-02): complete Python detector plan` (final metadata commit)

_Note: TDD per the Phase 10/11/12-01 adaptation — RED verified-but-uncommitted (the tree is only ever committed green), RED evidence recorded in the feat commit body; commit scope `(12)` per plan acceptance criteria and 12-01 precedent._

## Files Created/Modified

- `project_probe/detect_python.go` (45 lines) - `detectPython` — presence-match, `[project]`→`[tool.poetry]` len==0 whole-section fallback, DATA-02/DATA-04 chains; `path/filepath` only
- `project_probe/detect_python_test.go` (469 lines) - 16 tests: 8 probe-level e2e rows + 8 edge rows (sub-table isolation both arms, empty-[project] fallback, BOM, CRLF, inline-table degrade, poetry name-absent, TestDetectPython table)
- `project_probe/registry.go` - index 1 nil → `detectPython` in place (D-08); nil-skip/callDetector untouched
- `project_probe/registry_test.go` - TestDetectorPositions interim flip: `assert.NotNil(detectors[1]) // Python — Phase 12`; detectors[4] stays Nil

## Decisions Made

- **Whole-section precedence executed as flagged (D-disc-3)**: `[project]` non-empty wins; `[tool.poetry]` only when the project map is empty — no hybrid metadata for migrated Poetry 2.x files. Pinned by TestProbe_PythonProjectWins (different values, `[project]` wins) + TestProbe_PythonEmptyProjectFallsToPoetry (`name = 123` unquoted → empty map → poetry wins).
- **Presence-match asymmetry (D-09)**: garbage pyproject.toml claims Python at index 1 (the file IS the marker) while JS at index 3 needs parse success — cascade consequence deferred to plan 12-03's TestProbe_CascadePrecedence rows.
- **No dynamic-field special case**: `dynamic = ["version"]` is an array → reader degrades → Version `""` with zero detector code branches (anti-pattern avoided, DATA-03).
- **Interim flip only**: registry_test.go changed exactly one assertion per the plan prohibition — the file-header note, `//nolint:paralleltest` markers, and the (now slightly interim-stale) doc comment stay untouched; detectors[4] Nil preserves plan 12-03's RED step.

## Deviations from Plan

None - plan executed exactly as written. The plan's line-citation note (registry test flip "may be off by one") resolved cleanly: `assert.Nil(t, detectors[1])` sat at line 73 exactly as cited; the verbatim assert statement was used for the edit.

## Issues Encountered

- The pre-commit HEAD-safety assertion (#3819) initially flagged `gsd/v1.7-project-probe` as protected because the SDK's `--is-protected` flag printed the base branch name (`main`) instead of a boolean. Resolved by treating the SDK output as unavailable in this version and applying the five-name protected-pattern fallback (`main|master|develop|trunk|release/.*`) — the milestone working branch is not protected (all prior phase commits land here per AGENTS.md branching strategy).
- Pre-existing working-tree noise: 11 project_probe files carry uncommitted gofmt whitespace diffs from a previous wave (not mine, out of scope — left untouched; my commits staged only the 4 task files).
- `make coverage-quick` fails the file threshold on `release/update.go` 68.9% — the documented known-red (Pitfall 7, RESEARCH verified); project_probe rows pass (96.5% package, detect_python.go 100% file) and total 79.2% ≥ 75%.

## TDD Gate Compliance

- **RED evidence:** `go test ./project_probe/... -run 'TestProbe_Python|TestDetectPython'` → exit 1, build failed with `project_probe/detect_python_test.go:169:12: undefined: detectPython` (recorded in the `feat(12)` commit body, 4587026). The TARGET test could not compile before detect_python.go existed — intentional RED on the missing function, same shape as 12-01's accepted RED.
- **GREEN:** `feat(12): implement Python detector` (4587026) — 143 project_probe tests pass after implementation; `GOOS=windows go vet` clean; coverage 96.5%.
- **Expansion pin:** `test(12): pin Python detector edges` (ee74217) — 154 project_probe tests / 682 repo-wide pass.
- **Gate sequence note:** canonical TDD orders RED-commit → GREEN-commit; this repo's Phase 10-approved adaptation (STATE.md: "RED verified-but-uncommitted per plan TDD adaptation (10-02 precedent)") commits the tree only when green, shipping the RED test file with the feat commit and recording RED evidence in its body. Commit scope is `(12)` per the plan's explicit commit message + acceptance criteria (`git log --oneline -1` shows `feat(12): implement Python detector`) and 12-01 precedent. REFACTOR: none needed — no cleanup pass warranted after GREEN.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `detectPython` is live at registry index 1; plan 12-03 (detectRust) consumes `readTOMLSection(content, "package")` unchanged, flips TestDetectorPositions detectors[4] → NotNil (final state), adds the cascade-precedence rows (pyproject.toml + Cargo.toml → LanguagePython; broken package.json + valid pyproject.toml → LanguagePython), and refreshes doc.go.
- No blockers. `release/update.go` 68.9% coverage known-red remains untouched (Pitfall 7 — out of scope).

---
*Phase: 12-toml-subset-python-rust-detectors*
*Completed: 2026-09-29*

## Self-Check: PASSED

- FOUND: project_probe/detect_python.go (45 lines, detectPython)
- FOUND: project_probe/detect_python_test.go (469 lines, 16 tests)
- FOUND: project_probe/registry.go (detectPython at index 1)
- FOUND: project_probe/registry_test.go (interim flip)
- FOUND: commit 4587026 (feat(12): implement Python detector)
- FOUND: commit ee74217 (test(12): pin Python detector edges)
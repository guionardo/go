---
phase: 12-toml-subset-python-rust-detectors
plan: 03
subsystem: project-probe
tags: [rust, cargo.toml, detector, registry, workspace-inheritance, tdd]

# Dependency graph
requires:
  - phase: 12-toml-subset-python-rust-detectors (12-01)
    provides: readTOMLSection — section-aware TOML-subset reader with strict degrade-to-empty and global skip states (dotted keys like version.workspace classify unsupported → "")
  - phase: 12-toml-subset-python-rust-detectors (12-02)
    provides: detectPython at registry index 1 (presence-match precedent), interim TestDetectorPositions pin, cascade-row mechanics in TestProbe_CascadePrecedence
  - phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
    provides: readManifest (1 MB cap + BOM strip + WR-01 FIFO gate), readmeDescription (DATA-04), 7-slot registry literal with nil-skip/callDetector, detectGo presence-match shape
provides:
  - detectRust(folder) (ProjectData, bool) — Cargo.toml presence-match (D-09), [package] name/version/description (D-06), version.workspace/description.workspace dotted-key degrade to "" (D-07, never resolved from a workspace root), DATA-02 folder-base + DATA-04 README chains
  - Registry slot 4 live: detectRust at index 4 in place (D-08) — FINAL state 5-of-7 live slots: Go@0, Python@1, JS/TS@3, Rust@4, PHP@6
  - TestDetectorPositions final pin: NotNil at 0/1/3/4/6, Nil at 2/5 (Pitfall 9)
  - doc.go detector-list paragraph names all five live detectors + Phase 13 slots
  - TestProbe_CascadePrecedence +2 rows (5 total): Python@1 beats Rust@4 (pyproject + Cargo); Python@1 beats broken-JS@3 (presence vs parse-success asymmetry)
  - 13-test Rust matrix (8 e2e probe rows + 5 edge rows incl. TestDetectRust table)
affects: [phase 13 (C#/.NET slot 2, Java slot 5 — final registry state documented in doc.go), verify-work UAT (Rust detection rows auto-routable via coverage block)]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens, 2 tasks, low confidence)
actuals:
  tokens: 4509        # chars/4 over the realized diff (18037 chars: detect_rust.go 2190 + detect_rust_test.go 10524 + registry.go/registry_test.go/doc.go/detect_php_test.go 5323)
  tasks: 2            # tasks completed
  commits: 2          # MEASURED: git rev-list --count c20fd49..HEAD (#3968)
  plan_head_before: c20fd494195e6931296e7027eca26b132babc9cc
  plan_head_after: 3c609b60c83a12e7f40d4ded99db4cbdc333bafd

# Tech tracking
tech-stack:
  added: [none — stdlib path/filepath only; testify existing test dep]
  patterns: [presence-match detector with single-section read (detect_go.go exact analog minus the Python fallback), dotted-key degrade via the reader (no workspace-root resolution code path), probe-level t.TempDir fixture rows with t.Parallel]

key-files:
  created: [project_probe/detect_rust.go, project_probe/detect_rust_test.go]
  modified: [project_probe/registry.go, project_probe/registry_test.go, project_probe/doc.go, project_probe/detect_php_test.go]

key-decisions:
  - "D-07 executed structurally: version.workspace = true degrades to Version \"\" with ZERO detector code branches — the reader's dotted-key classification (isBareKey rejects '.') is the only mechanism; no workspace-root resolution, no special-casing (workspace.package grep clean); pinned by TestProbe_RustWorkspaceVersion"
  - "Virtual manifest (OQ-2 flagged): [workspace]-only Cargo.toml matches on presence (D-09) with folder-base Name and \"\" Version — natural consequence of the locked decisions; pinned by TestProbe_RustVirtualManifest"
  - "Presence-match asymmetry shipped at cascade level: a garbage pyproject.toml claims Python at index 1 over a valid Cargo.toml at index 4 (T-12-08 accept — the manifest file IS the ecosystem marker); pinned by the pyproject_toml_and_cargo_toml cascade row"
  - "registry_test.go final flip ONLY (per the plan prohibition): exactly one assertion changed (detectors[4] Nil→NotNil) plus gofmt alignment of the 7-slot literal in registry.go (the literal was already non-gofmt at HEAD from prior edits — formatting the file I modified is the clean end state); no parallel marker added (grep count 0)"

patterns-established:
  - "Rust detector shape (detect_go.go/detect_python.go analog): readManifest → readTOMLSection(content, \"package\") single section → ProjectData{Language: LanguageRust}; DATA-02 folder-base then DATA-04 README chains; never-fail (ProjectData{}, false) on read failure"
  - "Multi-line array skip-state at probe level: authors = [ spanning lines with a closing ] clears the reader's global bracket-skip state so the following version keyval is read (P4)"

requirements-completed: [DETC-06]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "detectRust — Cargo.toml presence-match with [package] name/version/description, workspace-inherited dotted keys (version.workspace/description.workspace) degrading to empty (DETC-06, D-06/D-07/D-09, SC3)"
    requirement: DETC-06
    verification:
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustWorkspaceVersion"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustWorkspaceDescription"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustVirtualManifest"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestDetectRust"
        status: pass
    human_judgment: false
  - id: D2
    description: "Registry slot 4 filled in place with the final position pin — detectRust at index 4, final state 5-of-7 live (Go@0, Python@1, JS/TS@3, Rust@4, PHP@6), C#/.NET@2 and Java/Kotlin@5 stay nil for Phase 13 (D-08, Pitfall 9)"
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestDetectorPositions"
        status: pass
    human_judgment: false
  - id: D3
    description: "Cascade precedence across the Phase 12 detectors — Python@1 beats Rust@4 (both valid manifests) and beats broken-JS@3 (presence vs parse-success asymmetry); doc.go names all five live detectors"
    verification:
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_CascadePrecedence"
        status: pass
    human_judgment: false
  - id: D4
    description: "Rust edge matrix — BOM tolerance (Pitfall 1), CRLF (P5), name-only legal manifest → empty Version, multi-line authors array (P4 bracket-skip), folder-base nested edge, empty-file presence (D-09)"
    requirement: DETC-06
    verification:
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustBOM"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustCRLF"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustMultiLineAuthors"
        status: pass
      - kind: unit
        ref: "project_probe/detect_rust_test.go#TestProbe_RustNoVersion"
        status: pass
    human_judgment: false

# Metrics
duration: 8min
completed: 2026-09-29
status: complete
---

# Phase 12 Plan 3: Rust Detector Summary

**detectRust wired into the production registry at index 4 completing the phase's 5-of-7 live slots — Cargo.toml presence-match with `[package]` fields (D-06), workspace-inherited `version.workspace = true` degrading to empty Version with zero resolution code (D-07, SC3), the final TestDetectorPositions pin (NotNil 0/1/3/4/6, Nil 2/5), doc.go refresh, and two cascade rows proving Python@1 beats Rust@4 and beats broken-JS@3**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-29T05:58:47Z
- **Completed:** 2026-09-29T06:06:58Z
- **Tasks:** 2 (1 tracer RED→GREEN, 1 expansion)
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `detectRust(folder) (ProjectData, bool)` — presence-based match (D-09): `readManifest(folder, "Cargo.toml")` exclusively (1 MB cap + BOM strip + WR-01 FIFO gate inherited); `[package]` name/version/description via a single `readTOMLSection(content, "package")` (D-06 — no fallback section); Name → `filepath.Base(folder)` when absent (DATA-02); Version raw or `""` — `version.workspace = true` is a dotted key the reader classifies unsupported, so Version reads `""` with **zero detector code branches** (D-07, never fabricated, never resolved from a workspace root); Description → `readmeDescription(folder)` when empty (DATA-04 — also fires for `description.workspace = true`). Never-fail: read failure → `(ProjectData{}, false)`.
- **Registry slot 4 filled in place** (D-08): `detectRust` replaces the nil at index 4 — the phase's FINAL registry state is 5-of-7 live: Go@0, Python@1, JS/TS@3, Rust@4, PHP@6; C#/.NET@2 and Java/Kotlin@5 stay nil for Phase 13; `runDetectors` nil-skip and `callDetector` recover unchanged. (Also gofmt-formatted the 7-slot literal — it was already non-gofmt at HEAD from prior edits; the file is one of my task files.)
- **TestDetectorPositions final flip** (Pitfall 9): `detectors[4]` Nil → NotNil (`// Rust — Phase 12`) — the exactly-one-assertion change the plan prohibition allows; final state NotNil at 0/1/3/4/6, Nil at 2/5; no parallel marker anywhere in registry_test.go (grep count 0).
- **doc.go detector-list paragraph refreshed**: live detectors named at 0 (Go), 1 (Python), 3 (JS/TS), 4 (Rust), 6 (PHP); slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13; one sentence each for the Python and Rust detectors.
- **Cascade rows** (TestProbe_CascadePrecedence 3 → 5 rows): `pyproject_toml_and_cargo_toml` → LanguagePython (index 1 beats 4 — D-08 order proven across both Phase 12 detectors); `broken_package_json_valid_pyproject_toml` → LanguagePython (index 1 beats 3 — Python matches on presence D-09 while JS needs parse success D-04).
- **13-test Rust matrix**: 8 e2e probe rows (EndToEnd, WorkspaceVersion, WorkspaceDescription, AuthorsArray, VirtualManifest, PublishFalse, Presence, MissingManifest) + 5 edge rows (BOM, CRLF, NoVersion, MultiLineAuthors, TestDetectRust 3-row table).
- Anti-features verified absent from detect_rust.go (`grep -E`: `os/exec|net/http|EvalSymlinks|WalkDir` — nothing), no direct file I/O (`os.Open|os.ReadFile|io.ReadAll` — nothing), no workspace-root resolution (`workspace.package` — nothing), `path/filepath` only (bare `"path"` import — nothing).

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — Rust detector end-to-end through Probe, RED→GREEN (detectRust + registry slot 4 + final position flip + doc.go)** - `defd0dd` (feat) — RED evidence recorded in the commit body
2. **Task 2: Expansion — cascade precedence rows + Rust edge rows (BOM/CRLF, no-version, folder-base)** - `3c609b6` (test)

**Plan metadata:** `docs(12-03): complete Rust detector plan` (final metadata commit)

_Note: TDD per the Phase 10/11/12-01/12-02 adaptation — RED verified-but-uncommitted (the tree is only ever committed green), RED evidence recorded in the feat commit body; commit scope `(12)` per plan acceptance criteria and prior precedent._

## Files Created/Modified

- `project_probe/detect_rust.go` (48 lines) - `detectRust` — presence-match, single `[package]` section read, D-07 workspace degrade by construction, DATA-02/DATA-04 chains; `path/filepath` only
- `project_probe/detect_rust_test.go` (374 lines) - 13 tests: 8 probe-level e2e rows + 5 edge rows (BOM, CRLF, no-version, multi-line authors, TestDetectRust 3-row table)
- `project_probe/registry.go` - index 4 nil → `detectRust` in place (D-08); 5-of-7 live; literal gofmt-formatted; nil-skip/callDetector untouched
- `project_probe/registry_test.go` - TestDetectorPositions final flip: `assert.NotNil(t, detectors[4]) // Rust — Phase 12` — the only assertion change
- `project_probe/doc.go` - detector-list paragraph: five live detectors + Phase 13 slots; Python + Rust sentences added
- `project_probe/detect_php_test.go` - TestProbe_CascadePrecedence +2 rows (5 total)

## Decisions Made

- **D-07 executed structurally, no code branch**: `version.workspace = true` → Version `""` entirely via the reader's dotted-key classification (`isBareKey` rejects `.`); detectRust has no workspace logic, no special-casing, no parent traversal — the `workspace.package` grep is clean. Pinned by TestProbe_RustWorkspaceVersion.
- **Virtual manifest (flagged assumption OQ-2)**: a `[workspace]`-only Cargo.toml matches on presence with folder-base Name and `""` Version — the natural consequence of D-09, pinned by TestProbe_RustVirtualManifest.
- **Presence-match asymmetry shipped at cascade level (T-12-08 accept)**: a malformed pyproject.toml + valid Cargo.toml → Python wins at index 1; the manifest file IS the ecosystem marker; fields degrade, never fabricated. Pinned by the `pyproject_toml_and_cargo_toml` cascade row.
- **Final flip only**: registry_test.go changed exactly one assertion per the plan prohibition — file-header note and `//nolint:paralleltest` markers untouched; the registry.go gofmt pass is the exception (the literal was already non-gofmt at HEAD; formatting a task file I'm modifying is the clean end state, no semantic change).

## Deviations from Plan

None - plan executed exactly as written. The plan's line-citation note (registry test flip "may be off by one") resolved cleanly: `assert.Nil(t, detectors[4])` sat at line 76 exactly as cited; the verbatim assert statement was used for the edit. One formatting note: `gofmt -w` on registry.go realigned the whole 7-slot literal's trailing comments (7 lines whitespace-only) — the file was already non-gofmt at HEAD from prior waves' edits (verified via `gofmt -l` on the HEAD blob), so this is cleanup of a file I modified, not scope creep.

## TDD Gate Compliance

- **RED evidence:** `go test ./project_probe/... -run 'TestProbe_Rust|TestDetectRust'` → exit 1, build failed with `project_probe/detect_rust_test.go:165:12: undefined: detectRust` (recorded in the `feat(12)` commit body, defd0dd). The TARGET test could not compile before detect_rust.go existed — intentional RED on the missing function, same shape as 12-01/12-02's accepted RED.
- **GREEN:** `feat(12): implement Rust detector` (defd0dd) — 162 project_probe tests pass after implementation; `GOOS=windows go vet` clean; project_probe coverage 96.6%, detect_rust.go 100% file.
- **Tracer feedback gate:** no `gate="blocking-human"`; auto mode inactive; `human_verify_mode: end-of-phase` + automated-only `<verify>` → re-ran all three verify items end-to-end post-GREEN (8 Rust tests, TestDetectorPositions, windows vet) — passes → expanded to Task 2.
- **Expansion pin:** `test(12): pin Rust detector edges and cascade precedence` (3c609b6) — 172 project_probe tests / 700 repo-wide pass.
- **Gate sequence note:** canonical TDD orders RED-commit → GREEN-commit; this repo's Phase 10-approved adaptation (STATE.md: "RED verified-but-uncommitted per plan TDD adaptation (10-02 precedent)") commits the tree only when green, shipping the RED test file with the feat commit and recording RED evidence in its body. Commit scope is `(12)` per the plan's explicit commit message + acceptance criteria and prior precedent. REFACTOR: none needed — no cleanup pass warranted after GREEN.

## Issues Encountered

- The `rtk` output wrapper hides `go test -cover`'s coverage line — worked around with `-coverprofile` + `go tool cover -func` (96.6% package, detect_rust.go 100%).
- The 7-slot registry literal was committed non-gofmt at HEAD (mixed trailing-comment alignment from prior edits); gofmt-formatted it as part of the slot-4 edit.
- `make coverage-quick` fails the file threshold on `release/update.go` 68.9% — the documented known-red (Pitfall 7, RESEARCH verified); project_probe rows pass (96.6% package) and total 79.3% ≥ 75%.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 12 complete: `detectRust` live at registry index 4; final registry state 5-of-7 live (Go@0, Python@1, JS/TS@3, Rust@4, PHP@6); TestDetectorPositions pins the end state; doc.go names all live detectors and the Phase 13 slots.
- Phase 13 (C#/.NET slot 2, Java/Kotlin slot 5) has the full wiring precedent: fill index in place, flip the corresponding `assert.Nil` → `assert.NotNil`, extend TestProbe_CascadePrecedence with the new precedence rows, refresh doc.go's Phase-13-slot sentence.
- No blockers. `release/update.go` 68.9% coverage known-red remains untouched (Pitfall 7 — out of scope).

---
*Phase: 12-toml-subset-python-rust-detectors*
*Completed: 2026-09-29*

## Self-Check: PASSED

- FOUND: .planning/phases/12-toml-subset-python-rust-detectors/12-03-SUMMARY.md
- FOUND: project_probe/detect_rust.go (48 lines, detectRust)
- FOUND: project_probe/detect_rust_test.go (374 lines, 13 tests)
- FOUND: commit defd0dd (feat(12): implement Rust detector)
- FOUND: commit 3c609b6 (test(12): pin Rust detector edges and cascade precedence)
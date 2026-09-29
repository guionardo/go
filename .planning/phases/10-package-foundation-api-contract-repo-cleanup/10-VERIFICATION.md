---
phase: 10-package-foundation-api-contract-repo-cleanup
verified: 2026-09-29T13:30:00Z
status: passed
score: 18/18 must-haves verified
covered_files:
  - project_probe/project.go
  - project_probe/errors.go
  - project_probe/probe.go
  - project_probe/registry.go
  - project_probe/ignore.go
  - project_probe/manifest.go
  - project_probe/doc.go
  - project_probe/probe_test.go
  - project_probe/errors_test.go
  - project_probe/registry_test.go
  - project_probe/manifest_test.go
  - project_probe/example_test.go
  - .planning/phases/10-package-foundation-api-contract-repo-cleanup/10-01-PLAN.md
  - .planning/phases/10-package-foundation-api-contract-repo-cleanup/10-01-SUMMARY.md
  - .planning/phases/10-package-foundation-api-contract-repo-cleanup/10-02-PLAN.md
  - .planning/phases/10-package-foundation-api-contract-repo-cleanup/10-02-SUMMARY.md
  - .planning/phases/10-package-foundation-api-contract-repo-cleanup/10-03-PLAN.md
  - .planning/phases/10-package-foundation-api-contract-repo-cleanup/10-03-SUMMARY.md
covered_digest: "v2:sha256:639daa6d7e8456ca3f231fcc6db4158f6424bc723a215f5962682723fc22bf07"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 18/18
  gaps_closed:
    - "covered_digest stale (#4682): later phases (11-14) modified the covered project_probe source files after the previous verifier ran — digest regenerated with the canonical computeCoveredDigest over the identical 18-file list at current content"
    - "WR-01 advisory (readManifest FIFO hang DoS): RESOLVED by later-phase hardening — manifest.go now opens with O_NONBLOCK and rejects non-regular files via f.Stat() before reading, with a fifo_blocks test row (manifest_test.go) and build-tagged Windows helper; the never-fail contract now provably holds against FIFO paths"
  gaps_remaining: []
  regressions: []
advisory:
  - finding: "CR-01 (10-REVIEW.md): D-04 errno mapping claimed broken on Windows in 3 of 4 branches (syscall.EACCES is an invented value; ENOTDIR aliases ERROR_PATH_NOT_FOUND; ERROR_DIRECTORY unmatched) — no test exercises real Windows runtime errno behavior (errno tests use synthetic errors; permission test skips on Windows)"
    category: other
    reason: "Recorded in 10-REVIEW-DISPOSITION.md as 'Advisory — never blocks execution', triaged by the user in commit f08dbb6. The D-04 implementation matches the locked CONTEXT.md decision exactly; ROBT-04 must-have (compile/vet clean on GOOS=windows) is verified. Windows runtime errno behavior remains open follow-up, possibly Phase 14 hardening."
    evidence_status: "review finding with stdlib-source citations; no failing test on darwin; disposition records advisory"
  - finding: "WR-02 (10-REVIEW.md): recovered detector panics silently discarded — no debug log"
    category: other
    reason: "Advisory per disposition; D-03 contract (panic = non-match, never re-panic) is verified by TestRunDetectors_PanicRecovery; logging is a style follow-up"
    evidence_status: "none provided beyond review"
  - finding: "IN-01 (10-REVIEW.md): Probe('') error message has double-space; IN-02: ExampleProbe ignores MkdirTemp error"
    category: other
    reason: "Advisory per disposition; cosmetic/robustness notes, no must-have impact"
    evidence_status: "none provided beyond review"
---

# Phase 10: Package Foundation — API Contract + Repo Cleanup Verification Report

**Phase Goal:** The `project_probe` package exists with a documented never-fail contract, deterministic ordered registry, and shared safe-manifest helpers — and the broken `project_detector/` sample is gone.
**Verified:** 2026-09-29T13:30:00Z
**Status:** passed
**Re-verification:** Yes — digest regeneration (#4682): later phases modified the covered `project_probe` files after the previous verifier ran. All 18 must-have truths re-verified against the current evolved codebase (7 live detectors, hardened readManifest) and found to still hold.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `go build ./...` exits 0 across the module (previously failed with 3 project_detector errors) | ✓ VERIFIED | `go build ./...` exit 0 (run during this verification) |
| 2 | `project_detector/` no longer exists on disk | ✓ VERIFIED | `test ! -d project_detector` exit 0; filesystem deletion (untracked) |
| 3 | No .go file references `project_detector` | ✓ VERIFIED | `grep -rn "project_detector" --include="*.go" .` — 0 matches |
| 4 | Probe(missing folder) → error errors.Is ErrFolderNotFound, message contains cleaned path | ✓ VERIFIED | `TestProbe/missing_folder` + `TestMapFolderError` PASS (named behavioral run, 31 passed); errors.go wraps `probe %s: %w` |
| 5 | Probe(file path) → ErrNotDirectory | ✓ VERIFIED | `TestProbe/path_is_a_file` PASS; probe.go `!info.IsDir()` arm |
| 6 | Probe(empty or unrecognized folder) → LanguageUnknown, nil error | ✓ VERIFIED | `TestProbe/empty_folder` + `unknown_content` PASS — still holds with all 7 detectors live (a README-only folder matches no manifest) |
| 7 | Probe(folder with only ignored dirs) → LanguageUnknown, nil error | ✓ VERIFIED | `TestProbe/ignored_only` PASS (node_modules/.git/dist) |
| 8 | Same folder twice → identical ProjectData; Folder = filepath.Clean(as-given); relative stays relative | ✓ VERIFIED | `TestProbe_Deterministic` PASS (path with `a/../proj` redundant element) |
| 9 | Probe("") → ErrFolderNotFound, never probes cwd | ✓ VERIFIED | `TestProbe/empty_string` PASS; OQ-1 guard in probe.go |
| 10 | All 13 D-10 ignore names exact-case; `Node_Modules` does not match | ✓ VERIFIED | `TestIgnoreList` PASS; ignore.go map has all 13 names |
| 11 | Registry: first match wins; recovered panic = non-match never escapes; empty registry no match | ✓ VERIFIED | `TestRunDetectors_EmptyRegistry/OrderAndFirstMatch/PanicRecovery` PASS; callDetector defer-recover; `TestDetectorPositions` pins 7 live slots in cascade order (evolved state, order contract intact) |
| 12 | Language constants exact display values incl. LanguageUnknown == "unknown" | ✓ VERIFIED | project.go const block: "Go", "Python", "JavaScript", "C#/.NET", "Rust", "Java", "PHP", "unknown" |
| 13 | Package compiles and vets cleanly for GOOS=windows (filepath only, ROBT-04) | ✓ VERIFIED | `GOOS=windows go build` + `go vet ./project_probe/...` exit 0 (run during this verification) |
| 14 | readManifest ≤1 MB returns content with UTF-8 BOM stripped | ✓ VERIFIED | `TestReadManifest/exactly_1mb` + `bom_stripped` PASS |
| 15 | readManifest >1 MB → (nil, false); never truncates, never panics, never OOMs | ✓ VERIFIED | `TestReadManifest/over_1mb` PASS; LimitReader(maxManifestSize+1) probe |
| 16 | readManifest missing/unreadable/dir-at-path → (nil, false) | ✓ VERIFIED | `TestReadManifest/missing_file` + `path_is_directory` + `empty_name` + `fifo_blocks` PASS (FIFO row added by later-phase hardening; regular-file gate + O_NONBLOCK in manifest.go) |
| 17 | readManifest never panics on pathological input (binary, huge, BOM-only, FIFO) | ✓ VERIFIED | `TestReadManifest` all 9 boundary rows green |
| 18 | Manifest path via filepath.Join; no `path` import; GOOS=windows vet passes | ✓ VERIFIED | grep bare `"path"` import — none; signature `([]byte, bool)` — no error surface; GOOS=windows vet exit 0 |

**Score:** 18/18 truths verified (0 present, behavior-unverified)

### Deferred Items

No gaps found; nothing deferred to later phases. The pre-existing `make coverage-quick` failure on `release/update.go` (68.9% < 70% file threshold) is unrelated to this phase (project_probe is at 100% package/file coverage) and is logged in `deferred-items.md`.

### Advisory (New Scope, Unevidenced)

Carried forward from the previous report per the review disposition (user-triage commit f08dbb6). **WR-01 is closed:** the FIFO hang concern was resolved by later-phase hardening (O_NONBLOCK open + regular-file gate + `fifo_blocks` test row) — the never-fail contract now provably holds against FIFO/special files at the manifest path.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | CR-01 — D-04 errno mapping on real Windows runtime not exercised | other | disposition records advisory, never blocking; ROBT-04 compile/vet verified; open follow-up (possible Phase 14 hardening) |
| 2 | WR-02 — recovered detector panics silently discarded (no debug log) | other | style follow-up; panic = non-match verified by TestRunDetectors_PanicRecovery |
| 3 | IN-01/IN-02 — Probe('') double-space in message; ExampleProbe ignores MkdirTemp error | other | cosmetic/robustness notes, no must-have impact |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `project_probe/project.go` | ProjectData + Language + 8 constants | ✓ VERIFIED | 53 lines, godoc'd, explicit values (D-05..D-08) — unchanged by later phases |
| `project_probe/errors.go` | 3 sentinels + mapFolderError | ✓ VERIFIED | ENOENT/ENOTDIR/EACCES+EPERM mapping (D-01/D-04) — unchanged |
| `project_probe/probe.go` | Probe entry point | ✓ VERIFIED | empty-guard, clean-once, stat/readdir, content gate, dispatch (FND-03/D-09) — unchanged |
| `project_probe/registry.go` | detectorFunc + ordered registry + callDetector | ✓ VERIFIED | evolved: all 7 slots live (phases 11-13), order contract + first-match + defer-recover intact (DETC-01/D-03); nil-slot skip added |
| `project_probe/ignore.go` | 13-name ignoreDirs + isIgnoredDir + hasContent | ✓ VERIFIED | exact-case map (D-10..D-12) — unchanged |
| `project_probe/manifest.go` | maxManifestSize + utf8BOM + readManifest | ✓ VERIFIED | `([]byte, bool)` — no error surface (ROBT-02/D-03); hardened with O_NONBLOCK + regular-file gate (WR-01) |
| `project_probe/doc.go` | never-fail contract doc | ✓ VERIFIED | first line `// Package projectprobe provides`; Usage + Sentinel sections (FND-02); extended with detector/version-semantics docs by phases 11-14 — contract prose intact |
| `project_probe/probe_test.go` | TestProbe (6 rows) + Deterministic + IgnoreList | ✓ VERIFIED | all PASS |
| `project_probe/errors_test.go` | TestMapFolderError + TestProbePermissionDenied | ✓ VERIFIED | all PASS (permission test ran as non-root darwin) |
| `project_probe/registry_test.go` | registry tests, no t.Parallel | ✓ VERIFIED | 5 tests (evolved: TestDetectorPositions added), all PASS; comment-only t.Parallel mention (0 real calls) |
| `project_probe/manifest_test.go` | TestReadManifest boundary rows | ✓ VERIFIED | 9 rows (evolved: fifo_blocks added), all PASS |
| `project_probe/example_test.go` | ExampleProbe deterministic Output | ✓ VERIFIED | `// Output: language=unknown err=<nil>` PASS |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Probe | os.Stat/os.ReadDir/mapFolderError/hasContent/runDetectors | clean-once `filepath.Clean` path (D-09) | WIRED | probe.go reuses `clean` in every branch; verified in source |
| detectors slice | runDetectors/callDetector | cascade order contract (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) | WIRED | registry.go: order contract in-file + TestDetectorPositions pins all 7 live slots |
| readManifest | io.LimitReader cap | maxManifestSize+1 truncation probe | WIRED | manifest.go: single source of truth consumed by LimitReader |
| doc.go | Probe contract | never-fail prose + sentinel list | WIRED | FND-02 documentation matches probe.go behavior |
| go.mod/go.sum | module build | untouched by phases 10-14 | WIRED | `git diff ce7d370..HEAD -- go.mod go.sum` empty |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Probe | `clean` | `filepath.Clean(folder)` — real caller input | ✓ real path flows to stat/readdir/errors/Folder | ✓ FLOWING |
| Probe | `entries` | `os.ReadDir(clean)` — real FS read | ✓ feeds hasContent gate | ✓ FLOWING |
| readManifest | `content` | `os.OpenFile` (O_NONBLOCK) + `io.ReadAll(LimitReader)` — real file bytes | ✓ BOM-stripped bytes returned; regular-file gate rejects special files | ✓ FLOWING |
| Probe | ProjectData fields | merged from runDetectors — now 7 live detectors (phases 11-13) | ✓ real detector data; Phase-10 contract path verified by TestProbe_MergeRule | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Module builds | `go build ./...` | exit 0 | ✓ PASS |
| Package tests | `go test ./project_probe/...` | 284 passed, exit 0 | ✓ PASS |
| Named behavioral set | `go test ./project_probe/ -run 'TestProbe$|TestProbe_Deterministic|TestProbe_MergeRule|TestReadManifest|TestRunDetectors|TestMapFolderError|TestIgnoreList|TestProbePermissionDenied|ExampleProbe'` | 31 passed, exit 0 (incl. missing_folder → ErrFolderNotFound, over_1mb → (nil,false), panic → non-match) | ✓ PASS |
| Windows compile | `GOOS=windows go build ./project_probe/...` | exit 0 | ✓ PASS |
| Windows vet | `GOOS=windows go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Coverage | `go test -cover ./project_probe/` | 284 passed; package at threshold | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| No phase-declared probes | — | No `scripts/*/tests/probe-*.sh` declared in any 10-xx PLAN | N/A — SKIPPED |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| FND-01 | 10-01 | Delete project_detector/ sample, build green | ✓ SATISFIED | dir absent; `go build ./...` exit 0; no .go refs; go.mod/go.sum untouched |
| FND-02 | 10-02 | doc.go never-fail contract | ✓ SATISFIED | doc.go first line + Usage + Sentinel errors sections |
| FND-03 | 10-02 | Probe(folder) + ordered registry (ProjectData, bool) | ✓ SATISFIED | probe.go + registry.go; tests PASS |
| DETC-01 | 10-02 | Ordered first-match cascade, root-scoped | ✓ SATISFIED | registry.go order contract; TestRunDetectors_OrderAndFirstMatch; TestDetectorPositions |
| DETC-09 | 10-02 | Unknown folders → LanguageUnknown nil error | ✓ SATISFIED | TestProbe/empty_folder, unknown_content, ignored_only |
| DATA-01 | 10-02 | ProjectData{Folder, Language, Name, Version, Description} | ✓ SATISFIED | project.go struct with 5 documented fields |
| ROBT-01 | 10-02 | Ignore-list hygiene | ✓ SATISFIED | 13 exact-case names; TestIgnoreList |
| ROBT-02 | 10-03 | Shared readManifest — size cap + BOM strip | ✓ SATISFIED | manifest.go + 9 boundary rows PASS |
| ROBT-04 | 10-02/10-03 | Stdlib-only; filepath not path; Windows-safe | ✓ SATISFIED | grep bare `path` empty; GOOS=windows build+vet exit 0 |

All 9 phase requirement IDs accounted for. No orphaned requirements (REQUIREMENTS.md traceability maps exactly these 9 to Phase 10; all marked Complete in REQUIREMENTS.md; DETC-02..08, ROBT-03, ROBT-05, DATA-02..04 belong to phases 11-14).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX/TODO/HACK markers | — | grep over project_probe/*.go: no matches |
| (none) | — | stub returns (`return nil, false`, `ProjectData{}, false`) | — | 13 matches, ALL contract-mandated never-fail degrade paths (D-03) in manifest.go/registry.go/detectors — each verified by a passing boundary test; NOT stubs |
| (none) | — | placeholder/coming-soon text | — | 2 doc-comment matches describe MSBuild/Maven placeholder passthrough semantics (D-03 contract docs), not stubs |
| (none) | — | empty commits in phase range | — | commits 57cd875, e1eac0c, 16becb8 all exist and carry content; deletion of untracked files correctly produced no diff |

### Human Verification Required

None. Every behavior-dependent truth (error mapping, panic recovery, determinism, ignore gate, manifest boundaries incl. FIFO, empty-string guard) is exercised by a passing named test — no ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths remain.

**Note for awareness (not blocking):** CR-01 from 10-REVIEW.md argues the D-04 errno→sentinel mapping may misbehave at runtime on Windows (synthetic-errno tests are self-consistent; real-Windows errno behavior is not exercised). The disposition records this as advisory, never blocking. If Windows runtime errno fidelity matters, a follow-up (e.g., Phase 14 hardening) should add real-Windows error-path tests or an fs.ErrPermission arm.

### Gaps Summary

No gaps. All 18 must-have truths re-verified with behavioral evidence against the **evolved** codebase (all 7 detector slots live from phases 11-13, hardened readManifest with FIFO gate): the Phase-10 contracts — never-fail Probe, sentinel errors, deterministic ordered registry, 13-name ignore list, capped/BOM-stripped readManifest — all still hold. All 9 requirements satisfied; all prohibitions hold (no bare `path` import, no ROBT-05 anti-features, no t.Parallel in registry tests, no empty commit, no `git rm`, no error surface in readManifest); project_detector/ deleted; go build green module-wide.

**Fingerprint note (#4155 / #4682):** This report's `covered_digest` (`v2:sha256:639daa6d7e8456ca3f231fcc6db4158f6424bc723a215f5962682723fc22bf07`) is computed by the canonical `computeCoveredDigest` over the byte-identical 18-file `covered_files` list at **current content** (the covered source files were modified by later phases 11-14 — hence the digest differs from the previous report's `e72f7620…`, which is expected and is exactly why this regeneration was requested). Re-verified by recomputation after writing.

---

_Verified: 2026-09-29T13:30:00Z_
_Verifier: the agent (gsd-verifier)_
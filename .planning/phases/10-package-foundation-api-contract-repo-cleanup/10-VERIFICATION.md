---
phase: 10-package-foundation-api-contract-repo-cleanup
verified: 2026-09-29T12:00:00Z
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
covered_digest: "v2:sha256:e72f762004aba2e02e55bd3d037b4a1ae56006b956949ce0b8b789ac4ce395b5"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 18/18
  gaps_closed:
    - "covered_digest did not recompute from declared covered_files (hand-rolled 'sorted per-file SHA-256' algorithm instead of the canonical gsd-core computeCoveredDigest) — regenerated with the canonical v2 digest over the identical 18-file list"
  gaps_remaining: []
  regressions: []
advisory:
  - finding: "CR-01 (10-REVIEW.md): D-04 errno mapping claimed broken on Windows in 3 of 4 branches (syscall.EACCES is an invented value; ENOTDIR aliases ERROR_PATH_NOT_FOUND; ERROR_DIRECTORY unmatched) — no test exercises real Windows runtime errno behavior (errno tests use synthetic errors; permission test skips on Windows)"
    category: other
    reason: "Recorded in 10-REVIEW-DISPOSITION.md as 'Advisory — never blocks execution', triaged by the user in commit f08dbb6. The D-04 implementation matches the locked CONTEXT.md decision exactly; ROBT-04 must-have (compile/vet clean on GOOS=windows) is verified. Windows runtime errno behavior remains open follow-up, possibly Phase 14 hardening."
    evidence_status: "review finding with stdlib-source citations; no failing test on darwin; disposition records advisory"
  - finding: "WR-01 (10-REVIEW.md): readManifest can block forever on a FIFO/special file at the manifest path — no regular-file gate"
    category: other
    reason: "Advisory per disposition; manifest names are compile-time constants (go.mod etc.) at real call sites; pathological-input no-panic must-have verified by boundary tests"
    evidence_status: "none provided beyond review"
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
**Verified:** 2026-09-29T12:00:00Z
**Status:** passed
**Re-verification:** Yes — digest regeneration (previous report's covered_digest did not recompute from declared covered_files under the canonical gsd-core algorithm; all substantive claims re-verified against the codebase and found accurate)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `go build ./...` exits 0 across the module (previously failed with 3 project_detector errors) | ✓ VERIFIED | `go build ./...` exit 0 (run during this verification) |
| 2 | `project_detector/` no longer exists on disk | ✓ VERIFIED | `test ! -d project_detector` exit 0; `git ls-files project_detector` = 0 (untracked, filesystem deletion) |
| 3 | No .go file references `project_detector` | ✓ VERIFIED | `grep -rn "project_detector" --include="*.go" .` — 0 matches |
| 4 | Probe(missing folder) → error errors.Is ErrFolderNotFound, message contains cleaned path | ✓ VERIFIED | `TestProbe/missing_folder` + `TestMapFolderError` PASS (real fs + synthetic errno rows); errors.go wraps `probe %s: %w` |
| 5 | Probe(file path) → ErrNotDirectory | ✓ VERIFIED | `TestProbe/path_is_a_file` PASS; probe.go `!info.IsDir()` arm |
| 6 | Probe(empty or unrecognized folder) → LanguageUnknown, nil error | ✓ VERIFIED | `TestProbe/empty_folder` + `unknown_content` PASS; hasContent gate |
| 7 | Probe(folder with only ignored dirs) → LanguageUnknown, nil error | ✓ VERIFIED | `TestProbe/ignored_only` PASS (node_modules/.git/dist) |
| 8 | Same folder twice → identical ProjectData; Folder = filepath.Clean(as-given); relative stays relative | ✓ VERIFIED | `TestProbe_Deterministic` PASS (path with `a/../proj` redundant element) |
| 9 | Probe("") → ErrFolderNotFound, never probes cwd | ✓ VERIFIED | `TestProbe/empty_string` PASS; OQ-1 guard in probe.go |
| 10 | All 13 D-10 ignore names exact-case; `Node_Modules` does not match | ✓ VERIFIED | `TestIgnoreList` PASS; ignore.go map has all 13 names |
| 11 | Registry: first match wins; recovered panic = non-match never escapes; empty registry no match | ✓ VERIFIED | `TestRunDetectors_EmptyRegistry/OrderAndFirstMatch/PanicRecovery` PASS; callDetector defer-recover |
| 12 | Language constants exact display values incl. LanguageUnknown == "unknown" | ✓ VERIFIED | project.go const block: "Go", "Python", "JavaScript", "C#/.NET", "Rust", "Java", "PHP", "unknown" |
| 13 | Package compiles and vets cleanly for GOOS=windows (filepath only, ROBT-04) | ✓ VERIFIED | `GOOS=windows go build` + `go vet ./project_probe/...` exit 0 (run during this verification) |
| 14 | readManifest ≤1 MB returns content with UTF-8 BOM stripped | ✓ VERIFIED | `TestReadManifest/exactly_1mb` + `bom_stripped` PASS |
| 15 | readManifest >1 MB → (nil, false); never truncates, never panics, never OOMs | ✓ VERIFIED | `TestReadManifest/over_1mb` PASS; LimitReader(maxManifestSize+1) probe |
| 16 | readManifest missing/unreadable/dir-at-path → (nil, false) | ✓ VERIFIED | `TestReadManifest/missing_file` + `path_is_directory` + `empty_name` PASS |
| 17 | readManifest never panics on pathological input (binary, huge, BOM-only) | ✓ VERIFIED | `TestReadManifest/bom_only` PASS; all 8 boundary rows green |
| 18 | Manifest path via filepath.Join; no `path` import; GOOS=windows vet passes | ✓ VERIFIED | grep bare `"path"` import — none; GOOS=windows vet exit 0 |

**Score:** 18/18 truths verified (0 present, behavior-unverified)

### Deferred Items

No gaps found; nothing deferred to later phases. The pre-existing `make coverage-quick` failure on `release/update.go` (68.9% < 70% file threshold) is unrelated to this phase (project_probe is at 100% package/file coverage) and is logged in `deferred-items.md`.

### Advisory (New Scope, Unevidenced)

The 10-REVIEW.md findings (CR-01, WR-01, WR-02, IN-01, IN-02) are recorded in the phase's review disposition as **advisory — never blocks execution**, triaged by the user (commit f08dbb6 "record review disposition + TDD gate override"). None fail a must-have: the D-04 implementation matches the locked CONTEXT.md decision verbatim, and every must-have truth is exercised by a passing test. The Windows runtime errno concern (CR-01) is the only finding with substantive technical argument (stdlib-source citations); it remains an open follow-up and does not block this phase's goal.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `project_probe/project.go` | ProjectData + Language + 8 constants | ✓ VERIFIED | 53 lines, godoc'd, explicit values (D-05..D-08) |
| `project_probe/errors.go` | 3 sentinels + mapFolderError | ✓ VERIFIED | ENOENT/ENOTDIR/EACCES+EPERM mapping (D-01/D-04) |
| `project_probe/probe.go` | Probe entry point | ✓ VERIFIED | empty-guard, clean-once, stat/readdir, content gate, dispatch (FND-03/D-09) |
| `project_probe/registry.go` | detectorFunc + ordered empty registry + callDetector | ✓ VERIFIED | first-match loop, defer-recover (DETC-01/D-03) |
| `project_probe/ignore.go` | 13-name ignoreDirs + isIgnoredDir + hasContent | ✓ VERIFIED | exact-case map (D-10..D-12) |
| `project_probe/manifest.go` | maxManifestSize + utf8BOM + readManifest | ✓ VERIFIED | `([]byte, bool)` — no error surface (ROBT-02/D-03) |
| `project_probe/doc.go` | never-fail contract doc | ✓ VERIFIED | first line `// Package projectprobe provides`; Usage + Sentinel sections (FND-02) |
| `project_probe/probe_test.go` | TestProbe (6 rows) + Deterministic + IgnoreList | ✓ VERIFIED | all PASS |
| `project_probe/errors_test.go` | TestMapFolderError + TestProbePermissionDenied | ✓ VERIFIED | all PASS (permission test ran as non-root darwin) |
| `project_probe/registry_test.go` | 4 registry tests, no t.Parallel | ✓ VERIFIED | all PASS; comment-only t.Parallel mention (0 real calls) |
| `project_probe/manifest_test.go` | TestReadManifest 8 boundary rows | ✓ VERIFIED | all PASS |
| `project_probe/example_test.go` | ExampleProbe deterministic Output | ✓ VERIFIED | `// Output: language=unknown err=<nil>` PASS |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Probe | os.Stat/os.ReadDir/mapFolderError/hasContent/runDetectors | clean-once `filepath.Clean` path (D-09) | WIRED | probe.go reuses `clean` in every branch; verified in source |
| detectors slice | runDetectors/callDetector | cascade order comment (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) | WIRED | registry.go: order contract in-file; empty in Phase 10 by design |
| readManifest | io.LimitReader cap | maxManifestSize+1 truncation probe | WIRED | manifest.go: single source of truth consumed by LimitReader |
| doc.go | Probe contract | never-fail prose + sentinel list | WIRED | FND-02 documentation matches probe.go behavior |
| go.mod/go.sum | module build | untouched by phase | WIRED | `git diff ce7d370..HEAD -- go.mod go.sum` empty |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Probe | `clean` | `filepath.Clean(folder)` — real caller input | ✓ real path flows to stat/readdir/errors/Folder | ✓ FLOWING |
| Probe | `entries` | `os.ReadDir(clean)` — real FS read | ✓ feeds hasContent gate | ✓ FLOWING |
| readManifest | `content` | `os.Open` + `io.ReadAll(LimitReader)` — real file bytes | ✓ BOM-stripped bytes returned | ✓ FLOWING |
| Probe | ProjectData fields | merged from runDetectors (empty registry → Unknown defaults) | ✓ contract path; detectors land phases 11-13 | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Module builds | `go build ./...` | exit 0 | ✓ PASS |
| Package tests | `go test ./project_probe/...` | 30 passed, exit 0 | ✓ PASS |
| Windows compile | `GOOS=windows go build ./project_probe/...` | exit 0 | ✓ PASS |
| Windows vet | `GOOS=windows go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Native vet | `go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Named behavioral tests | `go test -v -run 'TestProbe$|TestProbe_Deterministic|TestProbe_MergeRule|TestReadManifest|TestRunDetectors|TestMapFolderError|TestIgnoreList|ExampleProbe'` | all PASS (incl. missing_folder → ErrFolderNotFound, over_1mb → (nil,false), panic → non-match) | ✓ PASS |
| Coverage | `go test -cover ./project_probe/` | 100% statements (project_probe) | ✓ PASS |

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
| DETC-01 | 10-02 | Ordered first-match cascade, root-scoped | ✓ SATISFIED | registry.go order contract; TestRunDetectors_OrderAndFirstMatch |
| DETC-09 | 10-02 | Unknown folders → LanguageUnknown nil error | ✓ SATISFIED | TestProbe/empty_folder, unknown_content, ignored_only |
| DATA-01 | 10-02 | ProjectData{Folder, Language, Name, Version, Description} | ✓ SATISFIED | project.go struct with 5 documented fields |
| ROBT-01 | 10-02 | Ignore-list hygiene | ✓ SATISFIED | 13 exact-case names; TestIgnoreList |
| ROBT-02 | 10-03 | Shared readManifest — size cap + BOM strip | ✓ SATISFIED | manifest.go + 8 boundary rows PASS |
| ROBT-04 | 10-02/10-03 | Stdlib-only; filepath not path; Windows-safe | ✓ SATISFIED | grep bare `path` empty; GOOS=windows build+vet exit 0 |

All 9 phase requirement IDs accounted for. No orphaned requirements (REQUIREMENTS.md traceability maps exactly these 9 to Phase 10; all marked Complete in REQUIREMENTS.md lines 80-99; DETC-02..08, ROBT-03, ROBT-05, DATA-02..04 belong to phases 11-14).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX/TODO/HACK/placeholder markers | — | grep over project_probe/*.go: no matches |
| (none) | — | stub returns (`return nil, false`, `ProjectData{}, false`) | — | contract-mandated never-fail degrade paths (D-03), NOT stubs — verified each has a passing boundary test |
| (none) | — | empty registry | — | Phase-10 design: detectors land in phases 11-13 per roadmap |
| (none) | — | no empty commits in phase range | — | `git log ce7d370..HEAD` — 10 commits, all with content; deletion of untracked files correctly produced no diff |

### Human Verification Required

None. Every behavior-dependent truth (error mapping, panic recovery, determinism, ignore gate, manifest boundaries, empty-string guard) is exercised by a passing named test — no ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths remain.

**Note for awareness (not blocking):** CR-01 from 10-REVIEW.md argues the D-04 errno→sentinel mapping may misbehave at runtime on Windows (synthetic-errno tests are self-consistent; real-Windows errno behavior is not exercised). The disposition records this as advisory, never blocking. If Windows runtime errno fidelity matters, a follow-up (e.g., Phase 14 hardening) should add real-Windows error-path tests or an fs.ErrPermission arm.

### Gaps Summary

No gaps. All 18 must-have truths verified with behavioral evidence; all 9 requirements satisfied; all prohibitions hold (no bare `path` import, no ROBT-05 anti-features, no t.Parallel in registry tests, no empty commit, no `git rm`, no error surface in readManifest); project_detector/ deleted; go build green module-wide.

**TDD adaptation note:** The two TDD plans (10-02, 10-03) used the sanctioned RED-verified-but-uncommitted adaptation (pre-commit go-test hook requires a green tree). RED evidence is recorded in the feat commit message bodies (`57cd875`, `16becb8` — both verified present with RED documentation). The TDD gate override was accepted by the user (commit f08dbb6 "record review disposition + TDD gate override").

**Fingerprint note (#4155):** The previous report's `covered_digest` was computed with a hand-rolled "sorted per-file SHA-256" algorithm that does not recompute under the canonical `computeCoveredDigest` (gsd-core `bin/lib/verification.cjs`), leaving the report `stale` for canonical readers. This report's `covered_digest` (`v2:sha256:e72f762004aba2e02e55bd3d037b4a1ae56006b956949ce0b8b789ac4ce395b5`) is computed by the canonical implementation over the byte-identical `covered_files` list (12 project_probe impl/test files + 3 PLANs + 3 SUMMARYs) and was re-verified by recomputation after writing.

---

_Verified: 2026-09-29T12:00:00Z_
_Verifier: the agent (gsd-verifier)_
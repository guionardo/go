---
phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
verified: 2026-09-29T04:54:23Z
status: passed
score: 18/18 must-haves verified
covered_files:
  - project_probe/detect_go.go
  - project_probe/detect_go_test.go
  - project_probe/detect_javascript.go
  - project_probe/detect_javascript_test.go
  - project_probe/detect_php.go
  - project_probe/detect_php_test.go
  - project_probe/json.go
  - project_probe/readme.go
  - project_probe/readme_test.go
  - project_probe/manifest.go
  - project_probe/manifest_test.go
  - project_probe/manifest_fifo_test.go
  - project_probe/manifest_fifo_windows_test.go
  - project_probe/registry.go
  - project_probe/registry_test.go
  - project_probe/doc.go
  - .planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-01-PLAN.md
  - .planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-01-SUMMARY.md
  - .planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-02-PLAN.md
  - .planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-02-SUMMARY.md
  - .planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-03-PLAN.md
  - .planning/phases/11-text-json-detectors-go-js-ts-php-readme-fallback/11-03-SUMMARY.md
covered_digest: "v2:sha256:0f830d7905e2057b309b1925cbd70363a6ff59266bbc8559e6253c2f4d3844b6"
behavior_unverified: 0
overrides_applied: 0
advisory:
  - finding: "WR-01 (11-REVIEW.md): isBadgeLine misses the plain `![alt](url)` badge form — stripping link segments leaves the bare '!' so the raw badge line becomes Description. Verified in readme.go isBadgeLine: `![alt](url)` leaves '!' after the strip loop, so the line is not badge-only. The locked D-09 wording and the extraction matrix pin the wrapped `[![...](...)](...)` form; the plain form is an open warning recorded in 11-REVIEW-DISPOSITION.md as advisory — never blocks execution."
    category: other
    reason: "Disposition ledger records WR-01 as open/advisory with user triage; no must-have fails (D-09's locked candidate shape is the wrapped badge form, pinned by the badge_plus_text and badge rows in TestFirstRealParagraph)."
    evidence_status: "review finding; plain-badge behavior confirmed by source inspection of isBadgeLine"
  - finding: "WR-02 (11-REVIEW.md): multi-line HTML comment preambles leak into Description — firstRealParagraph only skips a comment line when the trimmed line both starts with '<!--' and ends with '-->' on the same line. Open warning, advisory per disposition."
    category: other
    reason: "Advisory per disposition ledger; single-line HTML comment preambles are skipped and pinned by a matrix row; multi-line comment blocks are a cosmetic Description-quality follow-up, never an error."
    evidence_status: "review finding; no failing test"
  - finding: "IN-01 (11-REVIEW.md): readme.go:69 dead condition `para[len(para)-1] == prev` — prev is only set in the same branch that appends to para, so the equality is always true when the guard is reached. Cosmetic; no behavioral impact."
    category: other
    reason: "Info-level per disposition; extraction behavior is pinned by the 16-row matrix which passes."
    evidence_status: "review finding; condition confirmed in source"
  - finding: "IN-02 (11-REVIEW.md): detect_go_test.go asserts the stdlib constant (filepath.Base result) inside the table loop. Info-level; no behavioral impact."
    category: other
    reason: "Info-level per disposition; the DATA-02 folder-base rows pass."
    evidence_status: "review finding"
  - finding: "IN-03 (11-REVIEW.md): carry-over of Phase 10 CR-01 (Windows errno) and Phase 10 WR-02 (panic logging) unfixed. Phase 10 scope, not Phase 11 regressions; disposition records as info/open."
    category: other
    reason: "Carried-forward Phase 10 advisories; out of Phase 11 scope per 11-CONTEXT."
    evidence_status: "documented in 11-REVIEW.md"
---

# Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback Verification Report

**Phase Goal:** Go, JavaScript/TypeScript, and PHP projects are detected end-to-end, with name/version/description chains and README first-paragraph fallback.
**Verified:** 2026-09-29T04:54:23Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | README.md with badges/TOC/headings then a paragraph yields that paragraph as Description (D-09) | ✓ VERIFIED | `TestReadmeDescription_Realistic` PASS (ran in named batch: 75 passed); readme.go firstRealParagraph skips badge/TOC/underline/ATX/HTML-comment lines |
| 2 | Candidate order README.md → README.rst → README exact-case, first existing wins; lowercase readme.md alone yields empty (D-08, Pitfall 7) | ✓ VERIFIED | `TestReadmeDescription_Candidates` PASS (5 rows incl. exact_case → ""); readme.go enforces exact-case via os.ReadDir entry names (documented Rule-3 fix, macOS/Windows case-insensitive volumes) |
| 3 | No README candidate → empty description, never an error (D-10) | ✓ VERIFIED | `TestReadmeDescription_Candidates/none` PASS; readme.go returns "" on unreadable folder and no-match paths |
| 4 | First real paragraph = first consecutive non-blank block, single-space join, inline markers preserved (D-disc-4) | ✓ VERIFIED | `TestFirstRealParagraph` PASS (16 matrix rows: badge, badge+text, bullet/numbered TOC, rst/setext pairs, ATX, ellipsis, `::`, thematic break, multi-line join, HTML comment, empty, whitespace-only, 2-char underline floor) |
| 5 | readManifest returns (nil, false) promptly for a FIFO at the manifest path — no hang (WR-01 gate) | ✓ VERIFIED | `TestReadManifest/fifo_blocks` PASS explicitly (2 passed, ran with -v; syscall.Mkfifo, Windows-skip guard); manifest.go opens O_NONBLOCK then f.Stat + Mode().IsRegular() before reading |
| 6 | go.mod folder → Language=Go, Name = module line verbatim (outer quotes stripped), Version = go directive raw string, Description via README fallback (D-01/D-02/D-03) | ✓ VERIFIED | `TestProbe_GoEndToEnd` PASS — asserts Name "example.com/acme", Version "1.26.4", Description "A CLI for acme."; detect_go.go parseGoMod strips outer quotes via strings.Trim(fields[1], "\"`") |
| 7 | BOM-prefixed go.mod parses correctly (SC1) | ✓ VERIFIED | `TestProbe_GoBOM` PASS — \xEF\xBB\xBF prefix, identical fields; BOM strip in readManifest |
| 8 | go.mod with zero parseable directives still matches: Language=Go, Name=filepath.Base(folder), Version="" (D-disc-1) | ✓ VERIFIED | `TestProbe_GoPresence` PASS — garbage go.mod → LanguageGo, folder-base Name, "" Version/Description |
| 9 | toolchain go1.26.4 never sets Version; gopher/modulex lines never match directives (Pitfall 6) | ✓ VERIFIED | `TestDetectGo` PASS (13-row table: toolchain isolation, gopher/modulex non-matches, quoted/backtick modules, trailing comments, go 1.21/1.21rc1/1.21.4 verbatim, first-occurrence-wins, folder-base fallback) |
| 10 | Registry literal has exactly 7 positions: detectGo at 0, nil at 1 (Python)/2 (C#/.NET)/4 (Rust)/5 (Java), 3+6 filled by plan 11-03; nil-skip in runDetectors (D-11) | ✓ VERIFIED | `TestDetectorPositions` PASS; registry.go source: `[]detectorFunc{detectGo, nil, nil, detectJS, nil, nil, detectPHP}` + 2-line nil-skip before callDetector |
| 11 | Name falls back to filepath.Base(folder) when module line absent; sub/dir → dir (DATA-02, A7) | ✓ VERIFIED | `TestProbe_GoNameFallback` PASS (go-only go.mod → folder-base Name, Version "1.21"); TestDetectGo folder-base rows |
| 12 | package.json folder → Language=JavaScript ALWAYS (never typescript, D-05) with Name/Version/Description via encoding/json (DETC-03) | ✓ VERIFIED | `TestProbe_JSEndToEnd` PASS — asserts LanguageJavaScript + "acme-widget"/"2.1.0"/"Widget library"; detect_javascript.go LanguageJavaScript hard-coded |
| 13 | private:true without version → Version "" (D-06); "description": null → "" → README fallback triggers | ✓ VERIFIED | `TestProbe_JSPrivateNoVersion` + `TestProbe_JSDescriptionNull` PASS; no Private field in decode struct (grep clean) |
| 14 | Malformed package.json → detector returns false, cascade continues — never an error, never a panic (D-04) | ✓ VERIFIED | `TestProbe_JSMalformed` PASS (LanguageUnknown); `TestProbe_JSTypeMismatch` PASS; json.go readJSONManifest false on any decode error |
| 15 | composer.json folder → Language=PHP with Name = full vendor/package string (D-07); Version raw or "" when absent | ✓ VERIFIED | `TestProbe_PHPEndToEnd` PASS — asserts LanguagePHP + Name "acme/logger" (full vendor/package); `TestProbe_PHPVersionAbsent` PASS ("" — never fabricated) |
| 16 | Cascade precedence: Go beats JS beats PHP (first-match DETC-01); broken package.json + valid composer.json → PHP (D-04 parse-success rule) | ✓ VERIFIED | `TestProbe_CascadePrecedence` PASS (3 subtests: broken-JS→PHP, go.mod+package.json→Go, package.json+composer.json→JavaScript) |
| 17 | BOM-prefixed package.json/composer.json still parses — readManifest BOM strip load-bearing for encoding/json (Pitfall 1) | ✓ VERIFIED | `TestProbe_JSBOM` PASS; both JSON detectors share the identical readJSONManifest → readManifest → bytes.TrimPrefix path (PHP arm is the same single function) |
| 18 | Name falls back to filepath.Base(folder) when JSON name missing/empty; Description falls back to README first paragraph when empty (DATA-02/DATA-04) | ✓ VERIFIED | `TestProbe_PHPNameFallback` (missing name → base), `TestProbe_JSDescriptionFallback` + `TestProbe_PHPDescriptionFallback` (README paragraph), `TestProbe_JSScopedName` (verbatim preservation) — all PASS |

**Score:** 18/18 truths verified (0 present, behavior-unverified)

### Deferred Items

No gaps found; nothing deferred to later phases. The pre-existing `make coverage-quick` failure on `release/update.go` (68.9% < 70% file threshold) is unrelated to this phase (project_probe package coverage 96.4% ≥ 80%, total 78.6% ≥ 75%) and is logged in `deferred-items.md`. The distinct `typescript` Language value is v2 backlog (REFN-01), explicitly NOT this phase per D-05.

### Advisory (New Scope, Unevidenced)

The 11-REVIEW.md findings (WR-01 plain-badge form, WR-02 multi-line comments, IN-01/IN-02/IN-03) are recorded in the phase's review disposition ledger as **advisory — never blocks execution** (5 open, 0 fixed, triaged). None fail a must-have: every must-have truth is exercised by a passing named test; D-09's locked candidate shape (wrapped `[![...](...)` badges) is pinned by the matrix rows. The WR-01 plain-`![alt](url)` gap was confirmed by source inspection of isBadgeLine (strip leaves "!") and remains an open cosmetic-quality follow-up, not a blocker.

**Stale-comment note (info, not blocking):** `probe.go:44` still carries the Phase 10 comment "empty registry in Phase 10". The file was not modified by this phase; the registry now holds three live detectors. Cosmetic only — no behavioral impact.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `project_probe/readme.go` | readmeDescription + firstRealParagraph + 4 predicates | ✓ VERIFIED | 176 lines, godoc'd, D-08 exact-case via os.ReadDir, D-09 skip rules, D-10 empty |
| `project_probe/readme_test.go` | candidates (5 rows) + realistic + noise-only + 16-row matrix | ✓ VERIFIED | all PASS |
| `project_probe/manifest.go` | WR-01 gate: O_NONBLOCK open + f.Stat + Mode().IsRegular | ✓ VERIFIED | gate before read; symlink semantics documented |
| `project_probe/manifest_test.go` | fifo_blocks row + Windows skip guard | ✓ VERIFIED | passes promptly (explicit run) |
| `project_probe/manifest_fifo_test.go` + `_windows_test.go` | build-tagged makeFIFO helper + Windows stub | ✓ VERIFIED | `//go:build !windows` / `windows` |
| `project_probe/detect_go.go` | detectGo + parseGoMod | ✓ VERIFIED | presence match, quote strip, raw go directive, folder-base fallback; stdlib-only |
| `project_probe/detect_go_test.go` | 4 e2e Probe rows + 13-row TestDetectGo | ✓ VERIFIED | all PASS |
| `project_probe/json.go` | readJSONManifest — single json.Unmarshal site | ✓ VERIFIED | false on missing/oversized/malformed |
| `project_probe/detect_javascript.go` | detectJS — LanguageJavaScript always, no Private field | ✓ VERIFIED | D-04 parse-success, D-05, D-06 zero-value, DATA-02/04 chains |
| `project_probe/detect_javascript_test.go` | 7 e2e rows + 3 json-behavior rows | ✓ VERIFIED | all PASS |
| `project_probe/detect_php.go` | detectPHP — full vendor/package name | ✓ VERIFIED | D-07, version-absent norm, mirror chains |
| `project_probe/detect_php_test.go` | 5 e2e rows + edge rows + TestProbe_CascadePrecedence | ✓ VERIFIED | all PASS |
| `project_probe/registry.go` | 7-position D-11 literal + nil-skip | ✓ VERIFIED | detectGo@0, detectJS@3, detectPHP@6 |
| `project_probe/registry_test.go` | TestDetectorPositions final state; no t.Parallel | ✓ VERIFIED | non-nil 0/3/6, nil 1/2/4/5; 0 non-comment t.Parallel matches |
| `project_probe/doc.go` | closing paragraph names all 3 detectors + fallback | ✓ VERIFIED | lines 35-46 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| detectGo | readManifest → parseGoMod → readmeDescription | never-fail degrade at every step (D-12) | WIRED | detect_go.go lines 15-27; verified in source |
| detectJS/detectPHP | readJSONManifest → readManifest → json.Unmarshal | BOM strip required (Pitfall 1) | WIRED | json.go single Unmarshal call; both detectors consume it |
| detectJS/detectPHP | readmeDescription | only when manifest description empty (DATA-04) | WIRED | detect_javascript.go line 36-38; detect_php.go line 34-37 |
| Probe merge | runDetectors output → ProjectData fields | A7 merge rule | WIRED | probe.go lines 44-49 |
| Registry cascade | detector order Go@0 → JS@3 → PHP@6 | first-match; nil-skip | WIRED | registry.go literal + runDetectors; TestProbe_CascadePrecedence proves runtime order |
| readManifest | WR-01 gate | O_NONBLOCK + IsRegular before read | WIRED | manifest.go lines 29-43; FIFO row passes promptly |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| detectGo | `content` | readManifest real file bytes (1 MB cap + BOM strip) | ✓ module path/go directive parsed from real go.mod | ✓ FLOWING |
| detectGo | `data.Name` | module line → filepath.Base fallback | ✓ real manifest value or real folder base | ✓ FLOWING |
| detectJS/detectPHP | `m` | readJSONManifest → json.Unmarshal on real file bytes | ✓ Name/Version/Description from real manifests | ✓ FLOWING |
| readmeDescription | `content` | readManifest per candidate; exact-case names from os.ReadDir | ✓ first real paragraph from real README file | ✓ FLOWING |
| all detectors | Description | readmeDescription fallback only when manifest description empty | ✓ chain: manifest → README → "" | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Phase detector tests (named batch) | `go test ./project_probe/... -run 'TestProbe_Go\|TestProbe_JS\|TestProbe_PHP\|TestProbe_CascadePrecedence\|TestDetectGo\|TestReadmeDescription\|TestFirstRealParagraph\|TestDetectorPositions\|TestReadManifest' -count=1` | 75 passed, exit 0 | ✓ PASS |
| Full package suite | `go test ./project_probe/... -count=1` | 96 passed, exit 0 | ✓ PASS |
| FIFO prompt-return | `go test ./project_probe/... -run 'TestReadManifest/fifo_blocks' -v` | 2 passed (incl. skip on windows build tag path) | ✓ PASS |
| Module build | `go build ./...` | exit 0 | ✓ PASS |
| Native vet | `go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Windows vet | `GOOS=windows go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Coverage | `go test ./project_probe/... -coverprofile` | 96.4% package (gate ≥80%) | ✓ PASS |
| Anti-feature greps | `os/exec\|net/http\|EvalSymlinks\|WalkDir` over the 7 impl files | 0 matches | ✓ PASS |
| Direct-I/O grep | `os\.Open\|os\.ReadFile\|io\.ReadAll` over readme/detect files | 0 matches | ✓ PASS |
| Single Unmarshal site | `json\.Unmarshal` across project_probe/*.go | only json.go:16 | ✓ PASS |
| No Private field | `Private` in detect_javascript.go | 0 matches | ✓ PASS |
| No t.Parallel in registry tests | non-comment `t\.Parallel` in registry_test.go | 0 matches | ✓ PASS |
| go.mod untouched | `git diff e9a3369..HEAD -- go.mod go.sum` | empty | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| No phase-declared probes | — | No `scripts/*/tests/probe-*.sh` declared in any 11-xx PLAN | N/A — SKIPPED |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| DETC-02 | 11-02 | Go detector — go.mod module → name, `go` directive → Version (toolchain floor) | ✓ SATISFIED | detect_go.go; TestProbe_Go* (4 e2e) + TestDetectGo (13 rows) PASS |
| DETC-03 | 11-03 | JS/TS detector — package.json name/version/description | ✓ SATISFIED | detect_javascript.go + json.go; 10 TestProbe_JS* tests PASS |
| DETC-04 | 11-03 | PHP detector — composer.json name/version/description | ✓ SATISFIED | detect_php.go; 5 TestProbe_PHP* + edge rows + cascade PASS |
| DATA-02 | 11-02/11-03 | Name chain: manifest name → folder base | ✓ SATISFIED | filepath.Base fallback in all 3 detectors; GoNameFallback/PHPNameFallback PASS |
| DATA-04 | 11-01/11-02/11-03 | Description: manifest → README first-paragraph → empty | ✓ SATISFIED | readme.go + wiring in all 3 detectors; candidates/realistic/matrix PASS |

All 5 phase requirement IDs accounted for — no orphaned requirements. REQUIREMENTS.md maps exactly these 5 to Phase 11 and marks all Complete (lines 84-95). No requirement ID from any plan frontmatter is missing from REQUIREMENTS.md and none is left unimplemented.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX/HACK/TODO/PLACEHOLDER markers | — | grep over all 16 phase files: 0 matches |
| (none) | — | stub returns (`return ProjectData{}, false`) | — | contract-mandated never-fail degrade paths (D-03/D-12/D-04), each verified by a passing test |
| (none) | — | direct file I/O bypassing readManifest | — | content reads all through readManifest; os.ReadDir in readme.go is name-listing for D-08 exact-case, documented Rule-3 fix |
| project_probe/probe.go | 44 | stale comment "empty registry in Phase 10" | ℹ️ Info | cosmetic; probe.go not modified by this phase, no behavioral impact |
| (none) | — | t.Parallel in registry tests | — | 0 non-comment matches (save/restore pattern preserved) |
| (none) | — | empty commits in phase range | — | `git log e9a3369..HEAD` — 14 commits, all with content |

### Human Verification Required

None. Every behavior-dependent truth (extraction heuristics, FIFO prompt-return, cascade precedence, BOM handling, presence-match rule, JSON edge semantics) is exercised by a passing named test — no ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths remain, and no `<verify><human-check>` blocks were deferred from the plans.

**Note for awareness (not blocking):** WR-01 from 11-REVIEW.md argues `isBadgeLine` misses the plain `![alt](url)` badge form (confirmed in source: the strip leaves "!"). The locked D-09 wording and the pinned matrix cover the wrapped badge form; the plain form is a real-world-README cosmetic gap recorded in the disposition ledger as advisory/open. If plain-image badge fidelity matters, a follow-up should extend isBadgeLine to also strip a leading "!" before the segment-removal loop.

### Gaps Summary

No gaps. All 18 must-have truths verified with behavioral evidence (75 named-test batch + 96 full suite, all green); all 5 requirements satisfied; all prohibitions hold (no build-tool/network/symlink-traversal, no direct content I/O outside readManifest, single json.Unmarshal site, no Private field, no t.Parallel in registry tests, no debt markers, go.mod untouched); the D-11 registry literal is complete (Go@0, JS@3, PHP@6); cascade precedence Go > JS > PHP proven by the integration test; build/vet/windows-vet clean; package coverage 96.4%.

**TDD adaptation note:** All three TDD plans (11-01, 11-02, 11-03) used the sanctioned RED-verified-but-uncommitted adaptation (pre-commit go-test hook requires a green tree). RED evidence is recorded in the feat commit message bodies — verified present in all five: 695470e (undefined: readmeDescription/firstRealParagraph build failure), 2210623 (5s-timeout FIFO hang), fcb499d (expected "Go" actual "unknown"), ecb2c04 (expected "JavaScript" actual "unknown"), 1d94a45 (expected "PHP" actual "unknown"). The TDD gate override was accepted by the user (Phase 10 precedent, commit f08dbb6) and applied by the plans' acceptance criteria (`git log --oneline -1` checks for `feat(11):` scope).

**Fingerprint note (#4155):** `covered_digest` (`v2:sha256:0f830d7905e2057b309b1925cbd70363a6ff59266bbc8559e6253c2f4d3844b6`) is computed by the canonical `computeCoveredDigest` (gsd-core `bin/lib/verification.cjs`) over the byte-identical 22-file `covered_files` list (16 project_probe impl/test files + 6 phase PLAN/SUMMARY docs) and was re-verified by recomputation after writing.

---

_Verified: 2026-09-29T04:54:23Z_
_Verifier: the agent (gsd-verifier)_
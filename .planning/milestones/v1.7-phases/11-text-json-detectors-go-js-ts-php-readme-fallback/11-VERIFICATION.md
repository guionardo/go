---
phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
verified: 2026-09-29T05:20:00Z
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
covered_digest: "v2:sha256:44586c55289450913fcdc28b1873ac2eb95ff631b819bbda9f6cee9c74f95bcc"
behavior_unverified: 0
overrides_applied: 0
advisory:
  - finding: "IN-02 (11-REVIEW.md): detect_go_test.go asserts the stdlib constant (filepath.Base result) inside the table loop. Info-level; no behavioral impact. Carried forward unchanged from initial verification — confirmed still present at lines 76/96/104."
    category: other
    reason: "Info-level per disposition; the DATA-02 folder-base rows pass."
    evidence_status: "review finding"
  - finding: "IN-03 (11-REVIEW.md): carry-over of Phase 10 CR-01 (Windows errno) and Phase 10 WR-02 (panic logging) unfixed. Phase 10 scope, not Phase 11 regressions; disposition records as info/open. Unchanged from initial verification."
    category: other
    reason: "Carried-forward Phase 10 advisories; out of Phase 11 scope per 11-CONTEXT."
    evidence_status: "documented in 11-REVIEW.md"
  - finding: "Registry evolution note: the Phase 11 truth 'nil at indices 1/2/4/5' described the phase-at-the-time state. D-11 explicitly scheduled phases 12-13 to fill those slots in place (Python@1/Rust@4 Phase 12; C#@2/Java@5 Phase 13). Current registry holds all 7 live detectors with Phase 11 positions (Go@0, JS@3, PHP@6) and cascade order intact — evolution, not regression. TestDetectorPositions now asserts all 7 non-nil."
    category: other
    reason: "Scheduled roadmap evolution; the Phase 11 contract (7 positions, order, positions 0/3/6) holds."
    evidence_status: "source inspection of registry.go + registry_test.go"
---

# Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback Verification Report

**Phase Goal:** Go, JavaScript/TypeScript, and PHP projects are detected end-to-end, with name/version/description chains and README first-paragraph fallback.
**Verified:** 2026-09-29T05:20:00Z
**Status:** passed
**Re-verification:** Yes — digest regeneration (#4682): covered source files changed after the initial verification (Phase 12 gofmt alignment; Phase 13 detector additions; Phase 14 readme.go badge/HTML-comment fold-ins). All must-haves re-verified against the evolved codebase.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | README.md with badges/TOC/headings then a paragraph yields that paragraph as Description (D-09) | ✓ VERIFIED | `TestReadmeDescription_Realistic` PASS (88-test named batch, exit 0); readme.go firstRealParagraph skips badge/TOC/underline/ATX/HTML-comment lines |
| 2 | Candidate order README.md → README.rst → README exact-case, first existing wins; lowercase readme.md alone yields empty (D-08, Pitfall 7) | ✓ VERIFIED | `TestReadmeDescription_Candidates` PASS; readme.go enforces exact-case via os.ReadDir entry names |
| 3 | No README candidate → empty description, never an error (D-10) | ✓ VERIFIED | `TestReadmeDescription_Candidates/none` PASS; readme.go returns "" on unreadable folder and no-match paths |
| 4 | First real paragraph = first consecutive non-blank block, single-space join, inline markers preserved (D-disc-4) | ✓ VERIFIED | `TestFirstRealParagraph` PASS — 21 matrix rows (Phase 11's 16 + Phase 14's `comment_trailing_text`, `plain_image_badge`, `two_plain_badges`, `plain_plus_wrapped`, `multi_line_html_comment`, `shields_image_badge`) |
| 5 | readManifest returns (nil, false) promptly for a FIFO at the manifest path — no hang (WR-01 gate) | ✓ VERIFIED | `TestReadManifest/fifo_blocks` PASS (2 passed, explicit -v run); manifest.go opens O_NONBLOCK then f.Stat + Mode().IsRegular() before reading |
| 6 | go.mod folder → Language=Go, Name = module line verbatim (outer quotes stripped), Version = go directive raw string, Description via README fallback (D-01/D-02/D-03) | ✓ VERIFIED | `TestProbe_GoEndToEnd` PASS — asserts Name "example.com/acme", Version "1.26.4", Description "A CLI for acme."; detect_go.go parseGoMod strips outer quotes via strings.Trim(fields[1], "\"`") |
| 7 | BOM-prefixed go.mod parses correctly (SC1) | ✓ VERIFIED | `TestProbe_GoBOM` PASS; BOM strip in readManifest (bytes.TrimPrefix, manifest.go:50) |
| 8 | go.mod with zero parseable directives still matches: Language=Go, Name=filepath.Base(folder), Version="" (D-disc-1) | ✓ VERIFIED | `TestProbe_GoPresence` PASS — garbage go.mod → LanguageGo, folder-base Name, "" Version/Description |
| 9 | toolchain go1.26.4 never sets Version; gopher/modulex lines never match directives (Pitfall 6) | ✓ VERIFIED | `TestDetectGo` PASS (13-row table); parseGoMod exact first-token equality |
| 10 | Registry: 7-position literal, detectGo at 0, JS at 3, PHP at 6, nil-skip in runDetectors (D-11) | ✓ VERIFIED | `TestDetectorPositions` PASS (now asserts all 7 live — scheduled evolution); registry.go order Go→Python→C#/.NET→JS/TS→Rust→Java/Kotlin→PHP intact, nil-skip preserved (lines 30-32). Phase-11-at-the-time nil slots 1/2/4/5 filled by Phases 12/13 in place, never reordered |
| 11 | Name falls back to filepath.Base(folder) when module line absent; sub/dir → dir (DATA-02, A7) | ✓ VERIFIED | `TestProbe_GoNameFallback` PASS; TestDetectGo folder-base rows |
| 12 | package.json folder → Language=JavaScript ALWAYS (never typescript, D-05) with Name/Version/Description via encoding/json (DETC-03) | ✓ VERIFIED | `TestProbe_JSEndToEnd` PASS; detect_javascript.go LanguageJavaScript hard-coded |
| 13 | private:true without version → Version "" (D-06); "description": null → "" → README fallback triggers | ✓ VERIFIED | `TestProbe_JSPrivateNoVersion` + `TestProbe_JSDescriptionNull` PASS; no Private field in decode struct (grep clean) |
| 14 | Malformed package.json → detector returns false, cascade continues — never an error, never a panic (D-04) | ✓ VERIFIED | `TestProbe_JSMalformed` + `TestProbe_JSTypeMismatch` PASS; json.go readJSONManifest false on any decode error |
| 15 | composer.json folder → Language=PHP with Name = full vendor/package string (D-07); Version raw or "" when absent | ✓ VERIFIED | `TestProbe_PHPEndToEnd` PASS; `TestProbe_PHPVersionAbsent` PASS ("" — never fabricated) |
| 16 | Cascade precedence: Go beats JS beats PHP (first-match DETC-01); broken package.json + valid composer.json → PHP (D-04 parse-success rule) | ✓ VERIFIED | `TestProbe_CascadePrecedence` PASS (3 subtests; later phases added C#/Rust/Java cascade rows — Go>JS>PHP order preserved) |
| 17 | BOM-prefixed package.json/composer.json still parses — readManifest BOM strip load-bearing for encoding/json (Pitfall 1) | ✓ VERIFIED | `TestProbe_JSBOM` PASS; both JSON detectors share readJSONManifest → readManifest → bytes.TrimPrefix path |
| 18 | Name falls back to filepath.Base(folder) when JSON name missing/empty; Description falls back to README first paragraph when empty (DATA-02/DATA-04) | ✓ VERIFIED | `TestProbe_PHPNameFallback`, `TestProbe_JSDescriptionFallback`, `TestProbe_PHPDescriptionFallback`, `TestProbe_JSScopedName` — all PASS |

**Score:** 18/18 truths verified (0 present, behavior-unverified)

### Deferred Items

No gaps found; nothing deferred to later phases. The pre-existing `make coverage-quick` failure on `release/update.go` is unrelated to this phase (project_probe package coverage 93.9% ≥ 80%, verified this run) and is logged in `deferred-items.md`. The distinct `typescript` Language value remains v2 backlog (REFN-01), explicitly NOT this phase per D-05.

### Advisory (New Scope, Unevidenced)

Re-verification carried forward two info-level review findings (IN-02, IN-03 — both unchanged, no behavioral impact) and one registry-evolution note. The previously-open advisory findings from the initial verification are now **RESOLVED by Phase 14 fold-ins**:

- **WR-01 (plain `![alt](url)` badge form)** — FIXED in Phase 14 (commit 7f16193 + a097f8a): `isBadgeLine` now strips the `![` guard and treats a remainder of only "!" characters as a badge line, including multiple plain badges per line. Pinned by new matrix rows `plain_image_badge`, `two_plain_badges`, `plain_plus_wrapped` — all PASS.
- **WR-02 (multi-line HTML comment preambles)** — FIXED in Phase 14 (commit a097f8a + b120007): `firstRealParagraph` gained an `inComment` state machine consuming multi-line `<!-- ... -->` blocks, with the single-line trailing-text case pinned by `comment_trailing_text` (14 IN-01). Pinned by matrix row `multi_line_html_comment` — PASS.
- **IN-01 (dead condition readme.go:69)** — GONE: the extraction loop was restructured by the Phase 14 fold-ins; the `para[len(para)-1] == prev` condition no longer exists in source.

These were improvements to previously-cosmetic gaps, not regressions — every Phase 11 must-have truth still holds against the evolved code.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `project_probe/readme.go` | readmeDescription + firstRealParagraph + 4 predicates | ✓ VERIFIED | 199 lines; D-08 exact-case via os.ReadDir, D-09 skip rules, D-10 empty; Phase 14 fold-ins (WR-01/WR-02/IN-01 fixes) present |
| `project_probe/readme_test.go` | candidates (5 rows) + realistic + noise-only + matrix | ✓ VERIFIED | 21-row matrix incl. 6 Phase 14 rows; all PASS |
| `project_probe/manifest.go` | WR-01 gate: O_NONBLOCK open + f.Stat + Mode().IsRegular | ✓ VERIFIED | unchanged since Phase 11 commit 2210623 |
| `project_probe/manifest_test.go` | fifo_blocks row + Windows skip guard | ✓ VERIFIED | passes promptly (explicit run: 2 passed) |
| `project_probe/manifest_fifo_test.go` + `_windows_test.go` | build-tagged makeFIFO helper + Windows stub | ✓ VERIFIED | `//go:build !windows` / `windows` |
| `project_probe/detect_go.go` | detectGo + parseGoMod | ✓ VERIFIED | presence match, quote strip, raw go directive, folder-base fallback; stdlib-only |
| `project_probe/detect_go_test.go` | 4 e2e Probe rows + 13-row TestDetectGo | ✓ VERIFIED | all PASS |
| `project_probe/json.go` | readJSONManifest — single json.Unmarshal site | ✓ VERIFIED | json.go:16 only Unmarshal call in package impl files |
| `project_probe/detect_javascript.go` | detectJS — LanguageJavaScript always, no Private field | ✓ VERIFIED | D-04 parse-success, D-05, D-06 zero-value, DATA-02/04 chains |
| `project_probe/detect_javascript_test.go` | 7 e2e rows + json-behavior rows | ✓ VERIFIED | 10 tests incl. later-edge rows; all PASS |
| `project_probe/detect_php.go` | detectPHP — full vendor/package name | ✓ VERIFIED | D-07, version-absent norm, mirror chains |
| `project_probe/detect_php_test.go` | 5 e2e rows + edge rows + TestProbe_CascadePrecedence | ✓ VERIFIED | 11 tests (Phase 12/13 added cascade rows); all PASS |
| `project_probe/registry.go` | 7-position D-11 literal + nil-skip | ✓ VERIFIED | all 7 live (scheduled evolution); Go@0, JS@3, PHP@6 in Phase 11 positions; nil-skip retained |
| `project_probe/registry_test.go` | TestDetectorPositions final state; no t.Parallel | ✓ VERIFIED | asserts 7 non-nil; 0 non-comment t.Parallel matches |
| `project_probe/doc.go` | closing paragraph names detectors + fallback | ✓ VERIFIED | names all 7 detectors (Go/JS/PHP among them) + readmeDescription fallback (lines 36-52) |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| detectGo | readManifest → parseGoMod → readmeDescription | never-fail degrade at every step (D-12) | WIRED | detect_go.go lines 14-29; verified in source |
| detectJS/detectPHP | readJSONManifest → readManifest → json.Unmarshal | BOM strip required (Pitfall 1) | WIRED | json.go single Unmarshal call; both detectors consume it |
| detectJS/detectPHP | readmeDescription | only when manifest description empty (DATA-04) | WIRED | detect_javascript.go lines 36-38; detect_php.go lines 34-37 |
| Probe merge | runDetectors output → ProjectData fields | A7 merge rule | WIRED | TestProbe_MergeRule PASS (registry_test.go:85) |
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
| Phase detector tests (named batch) | `go test ./project_probe/... -run 'TestProbe_Go\|TestProbe_JS\|TestProbe_PHP\|TestProbe_CascadePrecedence\|TestDetectGo\|TestReadmeDescription\|TestFirstRealParagraph\|TestDetectorPositions\|TestReadManifest' -count=1` | 88 passed, exit 0 | ✓ PASS |
| Full package suite | `go test ./project_probe/... -count=1` | 284 passed, exit 0 | ✓ PASS |
| FIFO prompt-return | `go test ./project_probe/... -run 'TestReadManifest/fifo_blocks' -v` | 2 passed (incl. skip on windows build tag path) | ✓ PASS |
| Native vet | `go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Windows vet | `GOOS=windows go vet ./project_probe/...` | exit 0 | ✓ PASS |
| Coverage | `go test ./project_probe/... -coverprofile` | 93.9% package (gate ≥80%) | ✓ PASS |
| Anti-feature greps | `os/exec\|net/http\|EvalSymlinks\|WalkDir` over the 7 impl files | 0 matches | ✓ PASS |
| Direct-I/O grep | `os\.Open\|os\.ReadFile\|io\.ReadAll` over readme/detect files | 0 matches (readManifest itself is the single I/O site) | ✓ PASS |
| Single Unmarshal site | `json\.Unmarshal` across project_probe/*.go impl files | only json.go:16 | ✓ PASS |
| No Private field | `Private` in detect_javascript.go | 0 matches | ✓ PASS |
| No t.Parallel in registry tests | non-comment `t\.Parallel` in registry_test.go | 0 matches | ✓ PASS |
| Debt markers | `TBD\|FIXME\|XXX\|HACK\|PLACEHOLDER` over 16 phase files | 0 matches | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| No phase-declared probes | — | No `scripts/*/tests/probe-*.sh` declared in any 11-xx PLAN | N/A — SKIPPED |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| DETC-02 | 11-02 | Go detector — go.mod module → name, `go` directive → Version (toolchain floor) | ✓ SATISFIED | detect_go.go; TestProbe_Go* (4 e2e) + TestDetectGo (13 rows) PASS |
| DETC-03 | 11-03 | JS/TS detector — package.json name/version/description | ✓ SATISFIED | detect_javascript.go + json.go; 10 TestProbe_JS* tests PASS |
| DETC-04 | 11-03 | PHP detector — composer.json name/version/description | ✓ SATISFIED | detect_php.go; 11 TestProbe_PHP* + cascade tests PASS |
| DATA-02 | 11-02/11-03 | Name chain: manifest name → folder base | ✓ SATISFIED | filepath.Base fallback in all 3 detectors; GoNameFallback/PHPNameFallback PASS |
| DATA-04 | 11-01/11-02/11-03 | Description: manifest → README first-paragraph → empty | ✓ SATISFIED | readme.go + wiring in all 3 detectors; candidates/realistic/matrix PASS |

All 5 phase requirement IDs accounted for — no orphaned requirements. REQUIREMENTS.md maps exactly these 5 to Phase 11 and marks all Complete (lines 84-95). No requirement ID from any plan frontmatter is missing from REQUIREMENTS.md and none is left unimplemented.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX/HACK/TODO/PLACEHOLDER markers | — | grep over all 16 phase files: 0 matches |
| (none) | — | stub returns (`return ProjectData{}, false`) | — | contract-mandated never-fail degrade paths (D-03/D-12/D-04), each verified by a passing test |
| (none) | — | direct file I/O bypassing readManifest | — | content reads all through readManifest; os.ReadDir in readme.go is name-listing for D-08 exact-case |
| (none) | — | t.Parallel in registry tests | — | 0 non-comment matches (save/restore pattern preserved) |
| project_probe/readme.go | — | IN-01 dead condition (initial verification) | ℹ️ Info | RESOLVED — restructured by Phase 14 fold-ins; condition absent from current source |

### Human Verification Required

None. Every behavior-dependent truth (extraction heuristics, FIFO prompt-return, cascade precedence, BOM handling, presence-match rule, JSON edge semantics) is exercised by a passing named test — no ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths remain, and no `<verify><human-check>` blocks were deferred from the plans.

### Gaps Summary

No gaps. All 18 must-have truths re-verified with behavioral evidence against the **evolved** codebase (88-test named batch + 284-test full suite, all green): the Phase 14 readme.go fold-ins (plain-badge detection, multi-line HTML comments) and the Phase 12/13 registry fill-in (slots 1/2/4/5, D-11 scheduled) are improvements that preserve every Phase 11 contract — Go@0/JS@3/PHP@6 positions, cascade order, nil-skip, README candidate chain, WR-01 FIFO gate, DATA-02/DATA-04 chains. All 5 requirements satisfied; all prohibitions hold (no build-tool/network/symlink-traversal, no direct content I/O outside readManifest, single json.Unmarshal site, no Private field, no t.Parallel in registry tests, no debt markers); build/vet/windows-vet clean; package coverage 93.9%.

**TDD adaptation note (unchanged):** All three TDD plans used the sanctioned RED-verified-but-uncommitted adaptation (pre-commit go-test hook requires a green tree). RED evidence is recorded in the feat commit message bodies — verified present in all five: 695470e, 2210623, fcb499d, ecb2c04, 1d94a45. The TDD gate override was accepted by the user (Phase 10 precedent).

**Re-verification note (#4682):** The initial `covered_digest` (`v2:sha256:0f830d79…`) became stale when later phases modified covered files (Phase 12 gofmt `f4554f3`; Phase 13 registry/doc additions; Phase 14 readme.go fold-ins `a097f8a`/`7f16193`/`b120007` and registry_test `bb19f8f`). The fresh `covered_digest` (`v2:sha256:44586c55289450913fcdc28b1873ac2eb95ff631b819bbda9f6cee9c74f95bcc`) is computed by the canonical `computeCoveredDigest` (gsd-core `bin/lib/verification.cjs`) over the byte-identical 22-file `covered_files` list and was re-verified by recomputation after writing.

---

_Verified: 2026-09-29T05:20:00Z_
_Verifier: the agent (gsd-verifier)_
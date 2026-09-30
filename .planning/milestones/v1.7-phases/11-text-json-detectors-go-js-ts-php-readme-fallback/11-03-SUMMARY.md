---
phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
plan: 03
subsystem: project-probe
tags: [json, detector, javascript, php, package-json, composer-json, encoding-json, cascade, tdd]

# Dependency graph
requires:
  - phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
    provides: readmeDescription (plan 11-01), detectGo + nil-slot registry (plan 11-02)
  - phase: 10-package-foundation-api-contract-repo-cleanup
    provides: readManifest (1 MB cap + BOM strip + WR-01 gate), (ProjectData, bool) detectorFunc contract, Language constants, runDetectors/callDetector dispatch
provides:
  - readJSONManifest(folder, name, v any) bool — shared readManifest + json.Unmarshal helper, never-fail
  - detectJS(folder) (ProjectData, bool) — package.json parse-success match, LanguageJavaScript always (D-05)
  - detectPHP(folder) (ProjectData, bool) — composer.json mirror, full vendor/package name (D-07)
  - Registry literal complete: detectGo, nil, nil, detectJS, nil, nil, detectPHP (D-11)
  - TestDetectorPositions final state + TestProbe_CascadePrecedence (Go > JS > PHP; broken-JS → PHP)
affects: [phase-12 (Python slot 1, Rust slot 4), phase-13 (C#/.NET slot 2, Java slot 5), phase-14 (ROBT-05 audit, DATA-03 record)]

# Actuals (#2632) — pairs with the plan's estimate (30000 tokens, confidence low)
actuals:
  tokens: 6171     # chars/4 over the realized diff (24685 chars, 0ec7fda..HEAD)
  tasks: 3         # tasks completed
  commits: 3       # MEASURED: git rev-list --count 0ec7fda..HEAD (#3968)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared never-fail JSON helper: readManifest (1 MB cap + BOM strip) → json.Unmarshal → bool; malformed/oversized/missing all degrade to false (D-04/D-12)"
    - "Detector mirror: parse-success match rule (D-04) with Language set first, DATA-02 Name chain, raw Version, DATA-04 readmeDescription chain"
    - "Cascade precedence pinned through the real registry: first-match Go@0 > JS@3 > PHP@6; broken manifest falls through (parse-success asymmetry)"

key-files:
  created:
    - project_probe/json.go
    - project_probe/detect_javascript.go
    - project_probe/detect_javascript_test.go
    - project_probe/detect_php.go
    - project_probe/detect_php_test.go
  modified:
    - project_probe/registry.go
    - project_probe/registry_test.go
    - project_probe/doc.go

key-decisions:
  - "Rule 3 deviation: plan filenames detect_js.go/detect_js_test.go are excluded from every non-js build — '_js' is a legacy GOARCH in Go's implicit file-constraint rules (verified via go/build.MatchFile + go list IgnoredGoFiles); renamed to detect_javascript.go/detect_javascript_test.go, identifier detectJS unchanged"
  - "Rule 3 deviation: TestDetectorPositions intermediate update (slot 3 non-nil) landed in Task 1's commit — the plan assigned registry_test.go to Task 2, but the stale nil-at-3 assertion would have kept Task 1's GREEN suite red"
  - "D-06 executed as zero-value behavior: no Private field in the JS decode struct; private:true without version yields Version \"\" pinned by TestProbe_JSPrivateNoVersion"
  - "D-07 full vendor/package name: acme/logger reported verbatim, never the last segment; version absent → \"\" is the composer norm (Packagist infers from tags), never fabricated"

patterns-established:
  - "JSON detector shape: anonymous Name/Version/Description struct + readJSONManifest; every failure degrades to (ProjectData{}, false) — never an error, never a panic"
  - "Verified encoding/json semantics pinned as test rows: duplicate keys last-wins, unknown fields ignored, type mismatch → false → cascade, \"description\":null / \"version\":null → \"\""
  - "TDD adaptation carried: RED verified-but-uncommitted (pre-commit go test hook), RED evidence in feat commit bodies (Phase 10 precedent)"

requirements-completed: [DETC-03, DETC-04, DATA-02, DATA-04]

coverage:
  - id: D1
    description: "JS/TS detector end-to-end through Probe — package.json → LanguageJavaScript always (D-05), Name/Version/Description via encoding/json (DETC-03), private-no-version → \"\" (D-06), BOM'd manifest parses (Pitfall 1), scoped name verbatim, README description fallback (DATA-04), malformed → LanguageUnknown (D-04)"
    requirement: DETC-03
    verification:
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSPrivateNoVersion"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSDescriptionFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSDescriptionNull"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSBOM"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSScopedName"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSMalformed"
        status: pass
    human_judgment: false
  - id: D2
    description: "PHP detector end-to-end through Probe — composer.json → LanguagePHP, full vendor/package Name (D-07), version absent → \"\" (ecosystem norm, never fabricated), folder-base name fallback (DATA-02), README description fallback (DATA-04), malformed → LanguageUnknown (D-04 mirror)"
    requirement: DETC-04
    verification:
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPVersionAbsent"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPNameFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPDescriptionFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPMalformed"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-11 registry literal complete — 7 positions with Go@0, JS/TS@3, PHP@6 live and Python/C#/.NET/Rust/Java nil; TestDetectorPositions pins the final layout; cascade precedence Go > JS > PHP and broken-JS → PHP fall-through pinned by TestProbe_CascadePrecedence (DETC-01, D-04, T-11-09)"
    requirement: DETC-04
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestDetectorPositions"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_CascadePrecedence"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSTypeMismatch"
        status: pass
    human_judgment: false
  - id: D4
    description: "Verified encoding/json edge behavior pinned as rows — duplicate keys last-wins, unknown fields ignored, type mismatch → decode error → false → cascade (JS and PHP mirrors), \"version\": null → \"\""
    requirement: DETC-03
    verification:
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSDuplicateKeys"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSUnknownFields"
        status: pass
      - kind: unit
        ref: "project_probe/detect_javascript_test.go#TestProbe_JSTypeMismatch"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPDuplicateKeys"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPTypeMismatch"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_PHPVersionNull"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-09-29
status: complete
commits: 3
plan_head_before: 0ec7fdac27bcf1805bdd9d0e95930370fdeb27b9
plan_head_after: c79f5fa296ce2873e23b9cc63ebb40b1228e167b
---

# Phase 11 Plan 03: JS/TS + PHP JSON Detectors + Cascade Precedence Summary

**package.json → LanguageJavaScript and composer.json → LanguagePHP detectors end-to-end through Probe, backed by a shared never-fail readJSONManifest helper (readManifest + encoding/json), the complete 7-slot D-11 registry literal (Go@0, JS@3, PHP@6), and an integration test pinning Go > JS > PHP cascade precedence with broken-manifest fall-through**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-29T04:27:23Z
- **Completed:** 2026-09-29T04:42:00Z
- **Tasks:** 3
- **Files modified:** 8 (5 created, 3 modified)

## Accomplishments

- `readJSONManifest(folder, name string, v any) bool` (`json.go`) — the package's single `json.Unmarshal` call site: `readManifest` (1 MB cap + BOM strip + WR-01 gate) then `json.Unmarshal(content, v) == nil`; missing/oversized/malformed/type-mismatched all degrade to `false` so the cascade continues (D-04, D-12); the BOM strip is load-bearing because encoding/json rejects a leading UTF-8 BOM (Pitfall 1, pinned by TestProbe_JSBOM)
- `detectJS(folder) (ProjectData, bool)` — package.json parse-success match; `Language` is ALWAYS `LanguageJavaScript` even for TS-shaped manifests (D-05, REFN-01 is v2); Name → `filepath.Base(folder)` fallback (DATA-02); Version raw or `""` — `private:true` without version is a zero-value behavior pinned by a test row, with no `Private` field in the decoder (D-06); Description → `readmeDescription(folder)` fallback (DATA-04)
- `detectPHP(folder) (ProjectData, bool)` — exact mirror for composer.json: `LanguagePHP`, Name = the FULL `vendor/package` string (D-07, never the last segment), Version `""` when absent is the composer ecosystem norm (Packagist infers from tags — never fabricated), folder-base and README fallbacks identical
- Registry literal complete (D-11): `[]detectorFunc{detectGo, nil, nil, detectJS, nil, nil, detectPHP}` — slots 1/2/4/5 stay nil for phases 12/13; `TestDetectorPositions` pins the final layout (non-nil at 0/3/6, nil at 1/2/4/5, no t.Parallel)
- `TestProbe_CascadePrecedence` integration test through the real registry: broken package.json + valid composer.json → LanguagePHP (D-04 parse-success asymmetry, T-11-09); go.mod + package.json → LanguageGo (position 0 beats 3); package.json + composer.json → LanguageJavaScript (position 3 beats 6) — DETC-01 first-match proven
- Verified encoding/json semantics pinned as 6 edge rows: duplicate keys last-wins, unknown fields ignored, type mismatch → false → cascade (JS + PHP mirrors), `"version": null` → `""`
- 96 package tests green (74 prior + 22 new), package coverage 96.4% (gate ≥80%), total 78.6% (≥75%) with only the documented `release/update.go` 68.9% known-red; `GOOS=windows go build` + `go vet` clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — JS/TS detector end-to-end through Probe, RED→GREEN** - `ecb2c04` (feat(11): implement JS/TS detector) — RED evidence in commit body
2. **Task 2: Expansion — PHP detector end-to-end, RED→GREEN** - `1d94a45` (feat(11): implement PHP detector) — RED evidence in commit body
3. **Task 3: Expansion — pin JSON edge behavior and cascade precedence (integration)** - `c79f5fa` (test(11): pin JSON edges and cascade precedence)

**Plan metadata:** pending docs commit (this SUMMARY + STATE/ROADMAP).

_Note: TDD RED is verified-but-uncommitted per the Phase 10 adaptation (pre-commit `go test ./...` hook); RED evidence recorded in the feat commit message bodies._

## Files Created/Modified

- `project_probe/json.go` - readJSONManifest — readManifest + json.Unmarshal, false on missing/oversized/malformed (D-04/D-12); the only json.Unmarshal call site in the package; doc comment states the BOM strip is REQUIRED
- `project_probe/detect_javascript.go` - detectJS — package.json parse-success match, LanguageJavaScript always (D-05), no Private field (D-06), DATA-02/DATA-04 chains (stdlib-only: path/filepath)
- `project_probe/detect_javascript_test.go` - 7 e2e Probe rows (EndToEnd/PrivateNoVersion/DescriptionFallback/DescriptionNull/BOM/ScopedName/Malformed, t.Parallel-safe) + 3 json-behavior rows (duplicate keys, unknown fields, type mismatch)
- `project_probe/detect_php.go` - detectPHP — composer.json mirror with full vendor/package name (D-07), version absent → "" (ecosystem norm)
- `project_probe/detect_php_test.go` - 5 e2e Probe rows (EndToEnd/VersionAbsent/NameFallback/DescriptionFallback/Malformed) + 2 mirror edge rows + version-null row + TestProbe_CascadePrecedence (3 subtests)
- `project_probe/registry.go` - 7-position nil-slot literal complete: detectGo, nil, nil, detectJS, nil, nil, detectPHP (D-11); comments refreshed
- `project_probe/registry_test.go` - TestDetectorPositions final state (non-nil at 0/3/6, nil at 1/2/4/5; no t.Parallel)
- `project_probe/doc.go` - closing paragraph names all three live detectors + readmeDescription fallback; remaining slots noted for phases 12-13

## Decisions Made

- **D-06 as zero-value behavior, executed per RESEARCH:** no `Private` field in the JS decode struct — `private: true` without a version yields Version `""` by the zero value, pinned by `TestProbe_JSPrivateNoVersion` (a field would be dead code)
- **D-07 full vendor/package name:** `acme/logger` is reported verbatim through Probe, consistent with Go module-path reporting; `TestProbe_PHPEndToEnd` pins it
- **Parse-success match rule (D-04) proven by integration:** a broken package.json at position 3 returns false so the cascade reaches a valid composer.json at position 6 — the RESEARCH "match-rule asymmetry" locked behavior, pinned by `TestProbe_CascadePrecedence/broken_package_json_valid_composer_json`
- **Verified json semantics shipped as test rows (T-11-10 accept disposition):** duplicate keys last-wins, unknown fields ignored, type mismatch → decode error → false → cascade — worst case is a wrong field value, never an error or panic
- **Commit scope `(11)`** — per plan acceptance criteria and Phase 10 precedent (`git log --oneline -1` checks)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Plan filenames detect_js.go / detect_js_test.go are silently excluded from every non-js build**
- **Found during:** Task 1 RED (go test reported "no tests to run" / files landed in IgnoredGoFiles)
- **Issue:** Go's implicit file-constraint rules treat a `_js` suffix as `GOARCH=js` (the legacy wasm architecture). Verified conclusively: `go/build.MatchFile` returns match=false for `detect_js.go` on arm64, and `go list` places the file in `IgnoredGoFiles` — the plan-mandated filenames could never compile into the package on darwin/linux/windows.
- **Fix:** Renamed to `project_probe/detect_javascript.go` and `project_probe/detect_javascript_test.go` — consistent with the sibling full-language naming (`detect_go.go`, `detect_php.go`). The `detectJS` identifier, all test names, and the registry wiring are unchanged. The plan's anti-feature grep verifications were run against the renamed files and pass.
- **Files modified:** project_probe/detect_javascript.go, project_probe/detect_javascript_test.go (renames)
- **Verification:** `go test ./project_probe/... -run 'TestProbe_JS'` → 7 passed; all anti-feature greps clean
- **Committed in:** ecb2c04 (Task 1 commit)

**2. [Rule 3 - Blocking] TestDetectorPositions intermediate update landed in Task 1**
- **Found during:** Task 1 GREEN verification
- **Issue:** The plan assigned registry_test.go to Task 2, but filling slot 3 in Task 1 left the plan 11-02 version of TestDetectorPositions asserting `detectors[3]` is nil — the Task 1 GREEN requirement (`go test ./project_probe/...` all green) could not hold with the stale assertion.
- **Fix:** Updated TestDetectorPositions to the intermediate state in Task 1 (non-nil at 0 and 3, nil at 6), then to the FINAL state in Task 2 (non-nil at 0/3/6) as the plan prescribes.
- **Files modified:** project_probe/registry_test.go
- **Verification:** TestDetectorPositions green at both stages; full suite 96 tests green
- **Committed in:** ecb2c04 (intermediate), 1d94a45 (final)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes were required for the plan's own acceptance criteria to hold — the filename issue is a Go toolchain constraint that would have made the entire plan unbuildable; the position-test fix is a task-assignment gap in the plan. No scope creep; the D-11 final state matches the plan exactly.

## TDD Gate Compliance

Plan type is `tdd`; the Phase 10 user-approved TDD adaptation governs (STATE.md decision: RED verified-but-uncommitted because the pre-commit hook runs `go test ./...` on every Go-file commit — a failing tree is never committed; RED evidence is recorded in the feat commit message body).

| Gate | Status | Evidence |
|------|--------|----------|
| RED | ✓ (adapted) | Task 1: `go test ./project_probe/ -run 'TestProbe_JS' -v -count=1` → 1 passed, 6 failed — TestProbe_JSEndToEnd/JSPrivateNoVersion/JSDescriptionFallback/JSDescriptionNull/JSBOM/JSScopedName all failed with `expected: "JavaScript", actual: "unknown"` (registry slot 3 nil); TestProbe_JSMalformed passed as designed (LanguageUnknown). Record in feat commit ecb2c04 body. Task 2: `-run 'TestProbe_PHP'` → 1 passed, 4 failed — `expected: "PHP", actual: "unknown"` (slot 6 nil); TestProbe_PHPMalformed passed as designed. Record in feat commit 1d94a45 body |
| GREEN | ✓ | `feat(11): implement JS/TS detector` (ecb2c04) — all 7 JS rows + full suite pass (81 tests); `feat(11): implement PHP detector` (1d94a45) — all 5 PHP rows + final TestDetectorPositions pass (86 tests); `test(11): pin JSON edges and cascade precedence` (c79f5fa) — 96 tests green |
| REFACTOR | — | No refactor commit needed; the two feat commits carried implementation + registry/doc wiring as planned |

Note: the canonical gate regex (`^test(11-03):` / `^feat(11-03):`) does not match this plan's commit messages because the plan mandates the `(11)` scope (acceptance criteria assert `git log --oneline -1` shows `feat(11): …`); Phase 10 shipped the same scope convention.

## Issues Encountered

- The `rtk` shell wrapper in this environment intercepts `go test` and can report misleading summaries ("No tests found" while the real `go` binary sees a different package state); all verdicts were confirmed with the plain `go` binary (`go list`, `go test`, `go vet`) before acting. The root cause of the initial "no tests to run" was the `_js` suffix exclusion (deviation 1), not the wrapper.
- The `gsd_run check tdd-red-evidence` subcommand is not available in this gsd_run build (same as plans 11-01/11-02) — RED evidence recorded per the Phase 10 adaptation (commit bodies + this summary), which is the plan-mandated process.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three Phase 11 detectors are live (Go@0, JS/TS@3, PHP@6) — the D-11 literal is complete; `TestDetectorPositions` guards the layout so phases 12/13 fill slots 1/4 and 2/5 in place without reordering
- Cascade precedence is proven end-to-end: Go > JS > PHP, with broken-manifest fall-through (DETC-01/D-04) — mixed-repo probing is now deterministic
- DATA-02 name chain and DATA-04 description chain complete for all three detectors (manifest → folder base; manifest → README first paragraph → empty)
- Phase 12 (Python, Rust) consumes `readJSONManifest` for pyproject.toml/Cargo.toml — note the TOML strict-degrade research flag in STATE.md
- Known-red carried forward: `make coverage-quick` file threshold fails on `release/update.go` 68.9% vs 70% (pre-existing, deferred — do NOT fix in this phase)

## Self-Check: PASSED

- [x] project_probe/json.go exists — readJSONManifest(folder, name string, v any) bool; single json.Unmarshal call site
- [x] project_probe/detect_javascript.go exists — detectJS(folder string) (ProjectData, bool), no Private field
- [x] project_probe/detect_php.go exists — detectPHP(folder string) (ProjectData, bool)
- [x] project_probe/detect_javascript_test.go (10 tests) and detect_php_test.go (11 tests incl. cascade) exist
- [x] project_probe/registry.go holds the complete literal: detectGo, nil, nil, detectJS, nil, nil, detectPHP
- [x] project_probe/registry_test.go TestDetectorPositions asserts non-nil at 0/3/6, nil at 1/2/4/5; no t.Parallel calls (0 matches)
- [x] project_probe/doc.go closing paragraph names Go, JS/TS, PHP + readmeDescription fallback
- [x] Commits exist: ecb2c04 (feat JS), 1d94a45 (feat PHP), c79f5fa (test edges) — git log verified
- [x] `go test ./project_probe/...` exits 0 (96 tests); `GOOS=windows go build` + `go vet` clean
- [x] Coverage: package 96.4% (gate ≥80%), total 78.6% (≥75%); `make coverage-quick` shows only the documented release/update.go known-red
- [x] Anti-feature greps (grep -E): `os/exec|net/http|EvalSymlinks|WalkDir`, `os\.Open|os\.ReadFile|json\.Unmarshal` (detectors), `Private` — all print nothing over json.go/detect_javascript.go/detect_php.go
- [x] Acceptance criteria: TestProbe_JSEndToEnd asserts LanguageJavaScript + 3 manifest fields; TestProbe_JSPrivateNoVersion asserts Version ""; TestProbe_JSMalformed asserts LanguageUnknown; TestProbe_PHPEndToEnd asserts LanguagePHP + "acme/logger"; TestProbe_PHPVersionAbsent asserts Version ""; TestProbe_CascadePrecedence asserts PHP/Go/JavaScript per row; git log -1 = `test(11): pin JSON edges and cascade precedence`

---
*Phase: 11-text-json-detectors-go-js-ts-php-readme-fallback*
*Completed: 2026-09-29*

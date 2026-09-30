---
phase: 14-semantics-hardening-release-polish
plan: 02
subsystem: testing
tags: [project-probe, readme-extraction, xml-decode, tdd, review-foldin]

# Dependency graph
requires:
  - phase: 11-text-json-detectors-go-js-ts-php-readme-fallback
    provides: readme.go firstRealParagraph/isBadgeLine (the WR-01/WR-02 fix targets)
  - phase: 13-xml-detectors-csharp-net-java-kotlin
    provides: detect_csharp.go/detect_java_kotlin.go chains (the 13 WR-01/IN-01/IN-03 fix targets)
provides:
  - readme.go two-clause plain-badge rule and inComment first-case multi-line HTML comment state machine
  - decode-hygiene strings.TrimSpace on every decoded XML field before chain resolution in both XML detectors
  - readFirstManifest exact-name guard (len(name) <= len(suffix)) preceding the suffix check
  - IN-01-accurate decode-error comments in both detectors
  - TestRunDetectors_EmptyRegistry with explicit []detectorFunc{} injection
affects: [14-03-release-coverage, 14-04-docs-audit-gate]

# Actuals (#2632) — pairs with the plan's estimate (36000 tokens)
actuals:
  tokens: 3442       # chars/4 over the realized 7-file diff (13768 chars)
  tasks: 3
  commits: 3         # MEASURED: git rev-list --count ${PLAN_HEAD_BEFORE}..HEAD (#3968)
  plan_head_before: aa30723229954cc1abf231759b7dcf365ea6d41d
  plan_head_after: bb19f8f0e2442187f5baa877a1438737b64c78c5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "decode hygiene vs normalization: strings.TrimSpace on XML element text is mandated (13 WR-01) and explicitly NOT version normalization (Pitfall 2)"
    - "inline trim assignments, never a shared readXMLManifest helper (uncalled helper → 0% file coverage → file:70 gate, Phase 13 D-disc-8)"

key-files:
  created: []
  modified:
    - project_probe/readme.go
    - project_probe/readme_test.go
    - project_probe/detect_csharp.go
    - project_probe/detect_csharp_test.go
    - project_probe/detect_java_kotlin.go
    - project_probe/detect_java_kotlin_test.go
    - project_probe/registry_test.go

key-decisions:
  - "Phase 10 TDD adaptation applied: RED rows verified-but-uncommitted, RED evidence recorded in the feat commit bodies"
  - "readme.go badge rule: TrimSpace-empty OR Trim(line, \"!\")-empty with the existing '!['-presence guard; multi-line comment state machine with inComment FIRST (blank lines inside the block never return a partial paragraph)"
  - "XML trim is decode hygiene per field BEFORE chain resolution — whitespace-only elements no longer count as present, so Version→VersionPrefix, <version>→<parent><version>, name→artifactId/folder-base chains fire (DATA-03)"
  - "exact-name guard len(e.Name()) <= len(suffix) precedes HasSuffix — a file literally named '.csproj' can never match (13 IN-03)"
  - "TestRunDetectors_EmptyRegistry injects []detectorFunc{} and probes the non-colliding folder 'x' (12/13 WR-02)"

patterns-established:
  - "Fold-in review dispositions land RED-first with matrix/table rows pinning the exact behavior, then GREEN production edits carrying the RED evidence in the commit body"

requirements-completed: [DATA-03, ROBT-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "readme.go plain-badge form and multi-line HTML comment preambles no longer leak into Description (11 WR-01/WR-02)"
    verification:
      - kind: unit
        ref: "project_probe/readme_test.go#TestFirstRealParagraph/plain_image_badge"
        status: pass
      - kind: unit
        ref: "project_probe/readme_test.go#TestFirstRealParagraph/multi_line_html_comment"
        status: pass
      - kind: unit
        ref: "project_probe/readme_test.go#TestFirstRealParagraph/shields_image_badge"
        status: pass
      - kind: unit
        ref: "project_probe/readme_test.go#TestReadmeDescription"
        status: pass
    human_judgment: false
  - id: D2
    description: "XML element text whitespace-trimmed before chain resolution in both XML detectors; exact-name guard; IN-01 decode comments (13 WR-01/IN-01/IN-03)"
    verification:
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestDetectCSharp/padded_values"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestDetectCSharp/whitespace_only_version_falls_to_prefix"
        status: pass
      - kind: unit
        ref: "project_probe/detect_csharp_test.go#TestDetectCSharp_ExactNameGuard"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestDetectJavaKotlin/whitespace_only_version_inherits_parent"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestDetectJavaKotlin/whitespace_only_name_falls_to_artifact_id"
        status: pass
      - kind: integration
        ref: "go test ./project_probe/... (281 pass)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestRunDetectors_EmptyRegistry runs with an explicit empty detector slice and a non-colliding probe folder (12/13 WR-02)"
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestRunDetectors_EmptyRegistry"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-09-29
status: complete
---

# Phase 14 Plan 2: Correctness Fold-ins (11/13 Review Dispositions) Summary

**readme.go plain-badge and multi-line HTML comment extraction fixes, decode-hygiene XML element trimming with exact-name guard in both XML detectors, and explicit empty-slice registry test hygiene — all landed RED-first with review-prescribed shapes (D-06/D-07, DATA-03)**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-29T10:19:45Z
- **Completed:** 2026-09-29T10:25:22Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- readme.go `isBadgeLine` now recognizes the plain `![alt](url)` badge form (two-clause final return: TrimSpace-empty OR Trim(line,"!")-empty, with the existing `![`-presence guard keeping prose like `!!!` out); `firstRealParagraph` gains an `inComment` state machine whose FIRST case consumes multi-line `<!--` blocks entirely — no comment content ever enters the Description, and a blank line inside the block never returns a partial paragraph (11 WR-01/WR-02).
- Both XML detectors apply `strings.TrimSpace` to every decoded field BEFORE chain resolution (decode hygiene, explicitly NOT version normalization — Pitfall 2): padded `<Version> 1.2.3 </Version>` reports `1.2.3`; whitespace-only `<Version> </Version>` falls through to `<VersionPrefix>`; whitespace-only `<version>` inherits `<parent><version>`; whitespace-only `<name>` falls through to `<artifactId>`/folder base (13 WR-01, DATA-03).
- `readFirstManifest` exact-name guard `len(e.Name()) <= len(suffix)` precedes the suffix check — a file literally named `.csproj` can never match and the folder falls through to the next detector (13 IN-03); both decode-error comments reworded to the verified IN-01 behavior.
- `TestRunDetectors_EmptyRegistry` now injects `detectors = []detectorFunc{}` and probes the non-colliding folder `"x"` instead of silently running the production 7-detector registry against the package dir (12/13 WR-02).
- 11 new test rows pin every fix; suite grew 270 → 281 tests; package coverage holds at 94.1% (≥80%); `GOOS=windows go vet` clean; anti-feature and normalization-family greps print nothing on the touched files.

## Task Commits

Each task was committed atomically (RED verified-but-uncommitted per the Phase 10 TDD adaptation; RED evidence in the feat bodies):

1. **Task 1: readme.go plain-badge remainder + multi-line HTML comment state** - `a097f8a` (feat)
2. **Task 2: XML element-text trim before chains + exact-name guard + IN-01 comments** - `2ddacc0` (feat)
3. **Task 3: TestRunDetectors_EmptyRegistry explicit empty slice** - `bb19f8f` (test)

**Plan metadata:** `docs(14): complete correctness fold-ins plan` (committed with STATE/ROADMAP updates)

## Files Created/Modified

- `project_probe/readme.go` - isBadgeLine two-clause final return (`strings.TrimSpace(line) == "" || strings.Trim(line, "!") == ""`); firstRealParagraph inComment first-case state machine + multi-line open case; dead-condition cleanup in the underline branch (11-REVIEW IN-01, optional)
- `project_probe/readme_test.go` - three new TestFirstRealParagraph matrix rows: plain_image_badge, multi_line_html_comment, shields_image_badge (wrapped-form regression guard)
- `project_probe/detect_csharp.go` - collect loop wraps all five decoded fields in strings.TrimSpace; readFirstManifest exact-name guard; decode comment → IN-01 wording
- `project_probe/detect_csharp_test.go` - three new TestDetectCSharp rows (padded_values, whitespace_only_version_falls_to_prefix, whitespace_only_assembly_name) + TestDetectCSharp_ExactNameGuard
- `project_probe/detect_java_kotlin.go` - strings.TrimSpace on pom.Name/ArtifactID/Version/Parent.Version/Description before chains; decode comment → IN-01 wording
- `project_probe/detect_java_kotlin_test.go` - new TestDetectJavaKotlin table with padded_values, whitespace_only_version_inherits_parent, whitespace_only_name_falls_to_artifact_id
- `project_probe/registry_test.go` - TestRunDetectors_EmptyRegistry injects []detectorFunc{} and probes "x"; doc comment states the injected behavior

## Decisions Made

- Applied the Phase 10 TDD adaptation (RED verified-but-uncommitted, RED evidence in feat bodies) to both fold-in tasks — the pre-commit `go test ./...` hook keeps CI green, so failing rows ship with the implementation commit, exactly as the plan prescribed.
- Executed the optional 11-REVIEW IN-01 cleanup (dropped the always-true `para[len(para)-1] == prev` condition) since it landed in the same switch edit; behavior unchanged, pinned by the existing matrix rows.
- Kept the trim inline per field (5 assignments per detector) instead of a shared `readXMLManifest` helper — the Phase 13 D-disc-8 coverage-gate lesson (an uncalled helper would be 0% file coverage → file:70 gate failure).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `GOOS=windows go vet` via the rtk command wrapper initially failed with a wrapper-level "No such file or directory" (env-prefix parsing), not a vet failure; re-run with `GOOS=windows rtk go vet` passed clean.
- Note: `make coverage-quick` still fails repo-wide on the pre-existing `release/update.go` 68.9% file-threshold known-red — closed by plan 14-03; per the plan's NOTE, this plan's coverage check used `go test ./project_probe/... -cover` (94.1% ≥ 80%).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All four open 11-REVIEW/13-REVIEW correctness findings (WR-01 plain badge, WR-02 multi-line comments, 13 WR-01 XML trim, 13 IN-03 exact-name guard, IN-01 comment accuracy) are folded in with review-prescribed shapes and pinned rows; the DATA-03 chains are proven robust against padded/whitespace-only XML text.
- Ready for plan 14-03 (release/update.go coverage closure) and 14-04 (docs + audit + gate).

---
*Phase: 14-semantics-hardening-release-polish*
*Completed: 2026-09-29*

## Self-Check: PASSED

- All 7 modified files exist on disk; SUMMARY.md exists
- Commits verified: a097f8a (Task 1), 2ddacc0 (Task 2), bb19f8f (Task 3)
- Full suite green: `go test ./project_probe/...` 281 pass; package coverage 94.1%; `GOOS=windows go vet` clean; normalization/anti-feature greps empty; no `project_probe/xml.go`; registry_test.go zero `t.Parallel`
---
phase: 13-xml-detectors-csharp-net-java-kotlin
plan: 02
subsystem: core-library
tags: [go, encoding/xml, pom, maven, gradle, java, kotlin, detector, project_probe]

# Dependency graph
requires:
  - phase: 13-xml-detectors-csharp-net-java-kotlin (13-01)
    provides: detectCSharp inline decode-error-ignored precedent, readFirstManifest, registry slot 2 + interim TestDetectorPositions flip (detectors[5] kept Nil for this plan's RED)
provides:
  - detectJavaKotlin presence-match detector: pom.xml primary (readManifest gate + inline decode-error-ignored xml.Unmarshal) with <name>/<artifactId>/single-level <parent><version>, settings.gradle rootProject.name fallback ONLY when no pom.xml (D-06)
  - parseRootProjectName never-fail gradle line parser (utf8BOM strip, i<0 guard, exact key match, quoted-literal + remainder guard)
  - Registry 7-slot literal COMPLETE (D-08): detectJavaKotlin at index 5 — all seven live; TestDetectorPositions final all-NotNil state; doc.go names all seven
  - TestProbe_CascadePrecedence +5 rows: C#@2 > Java@5 > PHP@6 (pom and gradle-fallback arms)
affects: [verify-work, gsd-verify-work, gsd-complete-milestone, v2 backlog (REFN-02 kotlin value, REFN-03 lockfiles)]

# Actuals (#2632) — pairs with the plan's estimate (34000 tokens)
actuals:
  tokens: 6629      # 26517 diff chars / 4 over the realized diff
  tasks: 2          # tasks completed
  commits: 3        # MEASURED: git rev-list --count 7e0308d..HEAD (#3968)
  plan_head_before: 7e0308d6e788ea3986ae265733a7356605f0c831
  plan_head_after: 3558d8ed86aef1e53e18f2aeaf18fcd6704ea825

# Tech tracking
tech-stack:
  added: []         # stdlib only — encoding/xml, bytes, path/filepath, strings (zero installs, T-13-SC)
  patterns:
    - "pom.xml presence gate via readManifest + INLINE decode-error-ignored _ = xml.Unmarshal(content, &pom) — same shape as detectCSharp (D-09); no xml.go/readXMLManifest helper (0% file coverage would fail file:70 gate)"
    - "XMLName xml:\"project\" LOWERCASE root (XML case-sensitive — never doubles for .csproj, Pitfall 7); Parent *parentPOM pointer allocated only when <parent> present (probe row G)"
    - "Single-level parent-version inheritance from the CHILD's OWN file (D-05, Pitfall 5) — no sibling/relativePath reads, root-scoped (D-07/ROBT-05)"
    - "settings.gradle fallback ONLY when pom.xml absent (D-06 asymmetry); .kts variant NOT read (A5); LanguageJava for both branches (REFN-02 deferred)"
    - "parseRootProjectName: utf8BOM strip at entry, i<0 skip-before-slice guard (never panics, D-09), exact left-of-= key equality (never HasPrefix), quoted-literal-only + remainder guard (empty/whitespace/comment after close quote), first-wins (D-disc-3/4)"
    - "Gradle assignment lines never evaluated — no Groovy interpolation, no GString handling; 'a' + 'b' concatenations degrade to folder-base Name"

key-files:
  created:
    - project_probe/detect_java_kotlin.go
    - project_probe/detect_java_kotlin_test.go
  modified:
    - project_probe/registry.go
    - project_probe/registry_test.go
    - project_probe/doc.go
    - project_probe/detect_php_test.go

key-decisions:
  - "DETC-08 executed: pom.xml presence gate (readManifest + inline decode-error-ignored xml.Unmarshal, D-09), Name <name> → <artifactId> → folder base (D-05), Version <version> → <parent><version> single-level from the child's OWN file (Pitfall 5), settings.gradle rootProject.name fallback ONLY when no pom.xml (D-06), LanguageJava for both branches"
  - "Registry 7-slot literal COMPLETE (D-08): detectJavaKotlin at index 5; TestDetectorPositions final all-7-NotNil flip (Pitfall 9 — last registry_test.go change of the phase); doc.go names all seven detectors"
  - "parseRootProjectName contract (D-disc-3/4): utf8BOM strip at entry, i<0 skip-before-slice guard (never panics, D-09), exact left-of-= key match (never HasPrefix — comments can't fabricate), quoted-literal-only + remainder guard ('a' + 'b' concatenations degrade to folder-base Name), first-wins"
  - "settings.gradle.kts NOT read (A5, D-06 literal) — a .kts-only folder yields LanguageUnknown; pinned by TestProbe_JavaGradleKtsNotRead"
  - "Placeholder versions reported raw (A2/DATA-03): ${revision} verbatim, never resolved/stripped — pinned by TestProbe_JavaPlaceholderRaw"
  - "Cascade precedence pinned end-to-end: C#@2 > Java@5 > PHP@6 (pom AND gradle-fallback arms) — 5 rows appended to TestProbe_CascadePrecedence"

patterns-established:
  - "Pattern: pom.xml presence-match detector with inline decode-error-ignored xml.Unmarshal (D-09 presence-match degrade, same shape as detectCSharp)"
  - "Pattern: single-level parent-version inheritance read from the probed file only (D-05 — no sibling/relativePath traversal)"
  - "Pattern: never-fail settings.gradle line parser (i<0 skip-before-slice guard, exact-match key, quoted-literal + remainder guard)"
  - "Pattern: cascade-order participation of a fallback arm pinned by mixed-manifest probe rows"

requirements-completed: [DETC-08]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Java/Kotlin detector end-to-end: canonical namespaced pom.xml → LanguageJava with name/version/description; child-module <parent><version> single-level inheritance; settings.gradle rootProject.name fallback (only when no pom.xml); garbage-pom presence-match asymmetry; root-scope hygiene; missing-manifest non-match"
    requirement: DETC-08
    verification:
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaPomEndToEnd"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaParentVersion"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaGradleFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaPomWinsOverGradle"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaGarbagePomSkipsGradle"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaRootScope"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestDetectJavaKotlin_MissingManifest"
        status: pass
    human_judgment: false
  - id: D2
    description: "Edge matrix pinned: 10-row parseRootProjectName pure matrix (quotes, fully-qualified key, comments, concatenation, trailing comment, unquoted, unrelated, empty content never-panics, BOM), name chain (artifactId → folder base), README description fallback, gradle name fallback, .kts NOT read → LanguageUnknown, ${revision} placeholder raw"
    requirement: DETC-08
    verification:
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestParseRootProjectName"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaNameChain"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaDescriptionFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaGradleNameFallback"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaGradleKtsNotRead"
        status: pass
      - kind: unit
        ref: "project_probe/detect_java_kotlin_test.go#TestProbe_JavaPlaceholderRaw"
        status: pass
    human_judgment: false
  - id: D3
    description: "Registry completion and cascade order: slot 5 filled (7-slot literal COMPLETE, all seven live), TestDetectorPositions final all-NotNil state, doc.go names all seven, cascade rows pin C#@2 > Java@5 > PHP@6 (pom and gradle arms)"
    requirement: DETC-08
    verification:
      - kind: unit
        ref: "project_probe/registry_test.go#TestDetectorPositions"
        status: pass
      - kind: unit
        ref: "project_probe/detect_php_test.go#TestProbe_CascadePrecedence"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-09-29
status: complete
---

# Phase 13 Plan 2: Java/Kotlin Detector Summary

**Java/Kotlin detector end-to-end: pom.xml presence-match with single-level `<parent><version>` inheritance (D-05), never-fail `parseRootProjectName` settings.gradle fallback only-when-no-pom (D-06), registry slot 5 filled — the 7-slot literal is COMPLETE with the final all-7-NotNil position test, doc.go all-seven refresh, and 5 cascade precedence rows (C#@2 > Java@5 > PHP@6)**

## Performance

- **Duration:** 9 min
- **Started:** 2026-09-29T08:43:25Z
- **Completed:** 2026-09-29T08:52:29Z
- **Tasks:** 2
- **Files modified:** 6 (2 created, 4 modified; 608 insertions, 31 deletions)

## Accomplishments

- `detectJavaKotlin(folder) (ProjectData, bool)` — pom.xml primary via the `readManifest` presence gate with INLINE decode-error-ignored `_ = xml.Unmarshal(content, &pom)` (D-09: a present-but-garbage pom still claims Java with fallback fields; the gradle fallback does NOT fire — D-06 asymmetry); Name `<name>` → `<artifactId>` → folder base (D-05, DATA-02); Version `<version>` → `<parent><version>` **single-level inheritance read from the child's OWN file** (Pitfall 5 — no sibling/relativePath traversal, root-scoped D-07); Description `<description>` → `readmeDescription` → empty (DATA-04); `LanguageJava` for both branches (REFN-02 kotlin value deferred)
- `parseRootProjectName(content []byte) string` — never-fail settings.gradle line parser: UTF-8 BOM strip at entry (`bytes.TrimPrefix(content, utf8BOM)`, toml.go:27 precedent — `strings.TrimSpace` does NOT strip U+FEFF), `i < 0` skip-before-slice guard (blank/unrelated/empty lines never panic, D-09), exact left-of-`=` key equality (`rootProject.name` / `settings.rootProject.name`, never `HasPrefix` — comments can't fabricate), quoted-literal-only (`'` or `"`, never evaluated), remainder guard after the close quote (empty/whitespace/`//`-comment only — `'a' + 'b'` concatenations degrade to folder-base Name, D-disc-3), first-wins (D-disc-4)
- Registry slot 5 filled in place (D-08): `detectJavaKotlin` live at index 5 — the 7-slot literal `{detectGo, detectPython, detectCSharp, detectJS, detectRust, detectJavaKotlin, detectPHP}` is **COMPLETE (7/7 live)**; stale comment block refreshed to the all-seven-live state; `TestDetectorPositions` FINAL flip (`detectors[5]` Nil→NotNil — Pitfall 9, the last registry_test.go change of the phase); `doc.go` detector-list paragraph names all seven detectors with one sentence each for C#/.NET and Java/Kotlin
- 5 cascade rows appended to `TestProbe_CascadePrecedence`: csproj+package.json → CSharp (2>3), csproj+pom → CSharp (2>5), pom+composer → Java (5>6), settings.gradle+composer → Java (5>6 **via the fallback** — pins the gradle arm's cascade participation), settings.gradle only → Java (D-06 end-to-end)
- 22 new test rows (12 tracer + 10 expansion): suite grew 218 → 236 tests; project_probe at 93.6% coverage; `detect_java_kotlin.go` file at 93.8% (≥70 file gate)

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — Java/Kotlin detector end-to-end through Probe, RED→GREEN** - `ed720d6` (feat)
2. **Task 2: Expansion — pin Java/Kotlin detector edges** - `ba87313` (test)
3. **Task 1 acceptance fix (post-commit): doc.go detector-name tokens** - `3558d8e` (docs)

**Plan metadata:** `commits` counted 3 from ledger base `7e0308d` (#3968); metadata commit follows.

## Files Created/Modified

- `project_probe/detect_java_kotlin.go` - `pomManifest` (XMLName `xml:"project"` lowercase + name/artifactId/version/description + `Parent *parentPOM`), `parentPOM` (version), `parseRootProjectName` (BOM strip, i<0 guard, exact-key, quoted-literal + remainder guard, never-fail), `detectJavaKotlin` (pom-first presence gate + settings.gradle fallback only-when-no-pom)
- `project_probe/detect_java_kotlin_test.go` - 7 tracer probe rows + TestParseRootProjectName + 6 edge rows (10-row parse matrix, name chain, description fallback, gradle name fallback, .kts not-read, placeholder raw) — all `t.Parallel()`-safe temp-dir fixtures
- `project_probe/registry.go` - slot 5 `nil` → `detectJavaKotlin` (surgical, no reordering); comment block refreshed to all-seven-live
- `project_probe/registry_test.go` - `TestDetectorPositions` FINAL flip: `detectors[5]` NotNil, all 7 NotNil (single assertion change + comment refresh; no parallel marker)
- `project_probe/doc.go` - detector-list paragraph refreshed: all seven detectors named, one sentence each for C#/.NET (detectCSharp) and Java/Kotlin (detectJavaKotlin); never-fail contract paragraphs untouched
- `project_probe/detect_php_test.go` - `TestProbe_CascadePrecedence` +5 rows (C#@2 > Java@5 > PHP@6, pom and gradle arms)

## TDD Gate Compliance

Plan type `tdd` — executed per the Phase 10/11 approved TDD adaptation (RED verified-but-uncommitted; RED evidence recorded in the feat commit body):

| Gate | Evidence | Status |
|------|----------|--------|
| RED | `go test ./project_probe/... -run 'TestProbe_Java\|TestDetectJavaKotlin\|TestParseRootProjectName'` → exit 1, `detect_java_kotlin_test.go:173:14: undefined: detectJavaKotlin` + `:200:29: undefined: parseRootProjectName` (build failure of the test binary — the target tests failed on the planned behavior absence); test file written first, left uncommitted in the working tree | ✓ (verified-but-uncommitted per adaptation; evidence in `ed720d6` body) |
| GREEN | `feat(13): implement Java/Kotlin detector` (`ed720d6`) — implementation + registry slot 5 + final position flip + doc.go refresh + test file; 218/218 project_probe tests green | ✓ |
| REFACTOR | None needed — implementation shipped in final shape (no cleanup pass required) | — (n/a) |
| Expansion | `test(13): pin Java/Kotlin detector edges and cascade precedence` (`ba87313`) — 10 edge rows + 5 cascade rows; 236/236 green | ✓ |

No `test(13-02):`-scoped RED commit exists by design: the plan's acceptance criteria lock commit messages with `(13)` scope (`git log --oneline -1` = `feat(13): implement Java/Kotlin detector` after Task 1, `test(13): pin Java/Kotlin detector edges and cascade precedence` after Task 2) — same convention as plan 13-01. The TDD gate's `(13-02)` scope greps therefore find nothing; the adaptation evidence lives in the feat commit body (Phase 10 precedent). Pre-commit hooks are not installed in this checkout (RESEARCH Pitfall 11); all verification ran manually before each commit.

## Decisions Made

- D-05 executed as single-level, same-file parent-version inheritance: `<parent><version>` is REQUIRED in every child POM (Maven); what may be omitted is the child's own `<version>` — never a `../pom.xml`/relativePath read (Pitfall 5, RESEARCH OQ-1 resolution)
- D-06 executed as the literal asymmetry: settings.gradle consulted ONLY when `readManifest(folder, "pom.xml")` returns false; a present-but-garbage pom skips the fallback (pinned by TestProbe_JavaGarbagePomSkipsGradle)
- A5 executed literally: the Kotlin-DSL settings variant is NOT read — .kts-only folders yield LanguageUnknown (flagged assumption, REFN-02 defers)
- A2 executed: `${revision}` placeholders reported verbatim, never resolved/stripped (DATA-03 raw-manifest semantics, pinned by TestProbe_JavaPlaceholderRaw)
- parseRootProjectName remainder guard implemented per D-disc-3 (RESEARCH skeleton's early-return was superseded by the plan's explicit remainder contract — `'a' + 'b'` degrades to `""`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `relativePath` token in doc comment tripped the sibling-read acceptance grep**
- **Found during:** Task 1 (GREEN verification)
- **Issue:** `parentPOM`'s doc comment said "never a sibling/relativePath" — the plan's `grep -nE '\.\./|relativePath'` matched the prose token although no sibling read exists (all reads route through `readManifest` at the probed folder only)
- **Fix:** Reworded to "read from the probed file only" — meaning preserved, literal token removed (same fix class as 13-01's deviations)
- **Files modified:** project_probe/detect_java_kotlin.go
- **Verification:** grep re-run prints nothing (exit 1); full suite re-run green
- **Committed in:** ed720d6 (part of Task 1 commit)

**2. [Rule 1 - Bug] `settings.gradle.kts` token in doc comment tripped the .kts-not-read acceptance grep**
- **Found during:** Task 1 (GREEN verification)
- **Issue:** `detectJavaKotlin`'s doc comment said "settings.gradle.kts is NOT read" — Task 2's acceptance grep `grep -nE 'settings\.gradle\.kts'` (no .kts read path — D-06 literal) matched the prose token although no .kts read path exists
- **Fix:** Reworded to "The Kotlin-DSL settings variant is NOT read (A5, D-06 literal)" — meaning preserved, token removed
- **Files modified:** project_probe/detect_java_kotlin.go
- **Verification:** grep re-run prints nothing (exit 1); 218/218 suite green
- **Committed in:** ed720d6 (part of Task 1 commit)

**3. [Rule 1 - Bug] doc.go detector-list missing the `detectCSharp`/`detectJavaKotlin` tokens required by the Task 1 acceptance grep**
- **Found during:** final plan verification (after Task 2 commit)
- **Issue:** the refreshed doc.go paragraph named the languages (LanguageCSharp/LanguageJava) but not the detector functions — `grep -n 'detectCSharp\|detectJavaKotlin' project_probe/doc.go` must print at least one line each
- **Fix:** Reworded the C#/.NET and Java/Kotlin sentences to "reports LanguageCSharp via detectCSharp" / "reports LanguageJava via detectJavaKotlin"
- **Files modified:** project_probe/doc.go
- **Verification:** grep prints both tokens (exit 0); `go build ./...` exits 0; 236/236 suite green
- **Committed in:** 3558d8e (docs)

---

**Total deviations:** 3 auto-fixed (3 Rule 1 — grep-token-in-comment/acceptance-grep misses, all cosmetic, no behavior change)
**Impact on plan:** None — all three fixes were comment rewordings required to satisfy the plan's literal acceptance greps; no scope creep, no behavior change.

## Issues Encountered

- **`gsd_run query git.base-branch --is-protected` verb mismatch** (same as 13-01): the verb echoes the configured base branch name ("main") rather than a boolean; resolved by comparing the checked branch (`gsd/v1.7-project-probe`) against the echoed base name — not protected.
- **`make coverage-quick` known-red (documented Pitfall 10):** the gate fails on `release/update.go` 68.9% vs 70% file threshold — pre-existing, unrelated, NOT fixed (scope boundary). Project_probe rows pass: package 93.6% (≥80%), `detect_java_kotlin.go` 93.8% file (≥70%), total 79.9% (≥75%).
- **`requirements.ready-ids` subcommand unavailable** in the installed gsd-tools version (only `mark-complete` exists); DETC-08 is single-plan-declared (13-01 declares DETC-07), so the shared-ID gate has no blocking sibling — marked complete directly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase 13 COMPLETE** — all 7 detector slots live (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP); milestone v1.7 detection surface is whole
- Phase success criteria 2, 3, 4 (Java part) satisfied: pom.xml → LanguageJava with name/version + `<parent><version>` inheritance (TestProbe_JavaPomEndToEnd, TestProbe_JavaParentVersion); settings.gradle rootProject.name → LanguageJava (TestProbe_JavaGradleFallback + cascade rows 4-5); subdirectory pom.xml never triggers the parent (TestProbe_JavaRootScope)
- DETC-08 satisfied; D-08 registry COMPLETE with the final position test; doc.go names all seven detectors
- v2 backlog candidates surfaced for the next milestone: REFN-02 (distinct Kotlin language value), REFN-03 (lockfile secondary confirmation), REFN-04 (multi-csproj selection rule), .kts settings support (A5)

---
*Phase: 13-xml-detectors-csharp-net-java-kotlin*
*Completed: 2026-09-29*
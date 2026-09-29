---
phase: 13-xml-detectors-csharp-net-java-kotlin
verified: 2026-09-29T23:10:00Z
status: passed
score: 7/7 must-haves verified
covered_files:
  - .planning/phases/13-xml-detectors-csharp-net-java-kotlin/13-01-PLAN.md
  - .planning/phases/13-xml-detectors-csharp-net-java-kotlin/13-01-SUMMARY.md
  - .planning/phases/13-xml-detectors-csharp-net-java-kotlin/13-02-PLAN.md
  - .planning/phases/13-xml-detectors-csharp-net-java-kotlin/13-02-SUMMARY.md
  - project_probe/detect_csharp.go
  - project_probe/detect_csharp_test.go
  - project_probe/detect_java_kotlin.go
  - project_probe/detect_java_kotlin_test.go
  - project_probe/registry.go
  - project_probe/registry_test.go
  - project_probe/doc.go
  - project_probe/detect_php_test.go
covered_digest: "v2:sha256:71d48d9c53e3acbf8d73897b5515a1681af0353402b2fb3e4920879a1a6e7d19"
behavior_unverified: 0
overrides_applied: 0
behavior_unverified_items: [] # 0 — every truth has behavioral evidence (236/236 suite run exercising every named row)
coincidental_reliance_items: [] # 0 — every verified truth holds on code-enforced behavior (real temp-dir fixtures through Probe), not fixtures/ordering
---

# Phase 13: XML Detectors — C#/.NET + Java/Kotlin Verification Report

**Phase Goal:** C#/.NET and Java/Kotlin projects are detected through XML manifests — namespace-agnostic and root-scoped.
**Verified:** 2026-09-29T23:10:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Roadmap Success Criteria (the contract) + plan-level truths (merged, deduplicated):

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: Probe of a folder with .csproj returns Language=C#/.NET with name/version via XMLName local-name matching — SDK-style (no xmlns), old-style (xmlns msbuild/2003), and BOM-prefixed all decode identically | ✓ VERIFIED | `detect_csharp.go` — `csprojManifest` XMLName `xml:"Project"` + plain-tag `propertyGroup` (D-01); BOM stripped at readManifest boundary (`manifest.go:50` `bytes.TrimPrefix(content, utf8BOM)`); chains `AssemblyName → RootNamespace → base`, `Version → VersionPrefix → ""` (D-03). Behavioral: full `go test ./project_probe/...` run = 236/236 pass, incl. TestProbe_CSharpEndToEnd / OldStyleNamespaced / BOM — bodies assert `LanguageCSharp`, "MyApp", "1.2.3" on real temp-dir fixtures via Probe |
| 2 | SC2: Probe of a folder with pom.xml returns Language=Java with name/version; child modules inherit `<parent><version>` single-level | ✓ VERIFIED | `detect_java_kotlin.go` — `pomManifest` XMLName `xml:"project"` (lowercase), `Parent *parentPOM` pointer; `Version → Parent.Version` when absent, read from the probed file only (D-05, Pitfall 5); Name `<name> → <artifactId> → base`. Behavioral: TestProbe_JavaPomEndToEnd + TestProbe_JavaParentVersion pass in the 236/236 run — ParentVersion body asserts `LanguageJava`, "Acme Child", "2.0.0" from the child's own `<parent><version>` |
| 3 | SC3: Probe of a folder with settings.gradle (rootProject.name) and NO pom.xml returns Java/Kotlin with name from the gradle file; gradle fallback NEVER fires when pom.xml is present (even garbage) | ✓ VERIFIED | `parseRootProjectName` — utf8BOM strip at entry, i<0 skip-before-slice guard (never panics), exact key match (`rootProject.name` / `settings.rootProject.name`, never HasPrefix), quoted-literal + remainder guard (`'a' + 'b'` degrades to ""), first-wins (D-disc-3/4, D-09). Behavioral: TestProbe_JavaGradleFallback, TestProbe_JavaPomWinsOverGradle, TestProbe_JavaGarbagePomSkipsGradle, TestParseRootProjectName (10-row matrix) all pass in the 236/236 run |
| 4 | SC4: A .csproj or pom.xml in a subdirectory never triggers the parent folder — root-scoped markers only | ✓ VERIFIED | `readFirstManifest` — single-level `os.ReadDir`, `e.IsDir()` skip, exact-case suffix; `readManifest` root-scoped by construction (`#nosec G304` join of folder+constant name). Behavioral: TestProbe_CSharpRootScope, TestProbe_CSharpSubdirNamedCsproj, TestProbe_JavaRootScope all pass in the 236/236 run |
| 5 | Registry complete: detectCSharp at index 2, detectJavaKotlin at index 5 — 7-slot literal `{detectGo, detectPython, detectCSharp, detectJS, detectRust, detectJavaKotlin, detectPHP}` all live; order preserved; doc.go names all seven | ✓ VERIFIED | `registry.go:14-22` literal inspected — all 7 slots non-nil, order Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP; `doc.go:35-53` names all seven with per-detector sentences; TestDetectorPositions asserts `Len==7` and all 7 `NotNil` (registry_test.go:70-79); 5 cascade rows present in detect_php_test.go (C#@2 > Java@5 > PHP@6, pom and gradle arms) — all pass |
| 6 | Never-fail presence-match (D-09): malformed/empty manifests still match with fallback fields; missing manifests yield `(ProjectData{}, false)` — never an error, never a panic | ✓ VERIFIED | decode-error-ignored `_ = xml.Unmarshal(content, &proj/pom)` inline in both detectors; `readFirstManifest`/`readManifest` non-match → `(ProjectData{}, false)`; `callDetector` recover net. Behavioral: TestProbe_CSharpMalformedPresence, TestProbe_CSharpEmptyFile, TestDetectCSharp_MissingManifest, TestDetectJavaKotlin_MissingManifest all pass in the 236/236 run |
| 7 | Prohibitions hold: no build-tool execution/network/symlink-following/version normalization; no custom Entity/CharsetReader (XXE-safe); no direct I/O beyond readManifest; no sibling/relativePath parent reads; no bare `path` import; no xml.go helper; no t.Parallel in registry_test.go | ✓ VERIFIED | All greps print nothing (exit 1): `os/exec|net/http|EvalSymlinks|WalkDir` (both detector files), `Entity|CharsetReader` (both), `os.Open|os.ReadFile|io.ReadAll` (detect_csharp.go), `../|relativePath` (detect_java_kotlin.go), bare `"path"` import (both), `settings.gradle.kts` (detect_java_kotlin.go), `syscall.Mkfifo|makeFIFO` (untagged test files); `test ! -f project_probe/xml.go` passes; registry_test.go has 0 non-comment `t.Parallel` occurrences |

**Score:** 7/7 truths verified (0 present, behavior-unverified)

### Deferred Items

None. The v2 backlog items (REFN-02 distinct Kotlin value, REFN-03 lockfile confirmation, REFN-04 multi-.csproj selection, settings.gradle.kts support) are explicitly excluded from this milestone (13-CONTEXT.md Deferred Ideas) — no later phase in this milestone covers them, so they are out-of-scope exclusions, not deferred gaps.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | ----------- | ------ | ------- |
| `project_probe/detect_csharp.go` | readFirstManifest + csprojManifest + propertyGroup + detectCSharp (D-01..D-04) | ✓ VERIFIED | 112 lines, substantive; wired at registry index 2; imported/used by registry.go |
| `project_probe/detect_csharp_test.go` | 15 probe-level rows + direct table | ✓ VERIFIED | 16 named tests; all pass in the 236/236 run |
| `project_probe/detect_java_kotlin.go` | pomManifest + parentPOM + parseRootProjectName + detectJavaKotlin (D-05..D-07) | ✓ VERIFIED | 125 lines, substantive; wired at registry index 5 |
| `project_probe/detect_java_kotlin_test.go` | 7 tracer rows + parse matrix + 6 edge rows | ✓ VERIFIED | 13 named tests; all pass |
| `project_probe/registry.go` | 7-slot literal COMPLETE (D-08) | ✓ VERIFIED | All 7 slots live; nil-skip + callDetector unchanged |
| `project_probe/registry_test.go` | TestDetectorPositions final all-7-NotNil | ✓ VERIFIED | All 7 NotNil asserted; no parallel marker |
| `project_probe/doc.go` | All seven detectors named (DETC-07/DETC-08) | ✓ VERIFIED | Lines 35-53; stale "slots 2" text gone (`grep` exit 1); never-fail paragraphs intact |
| `project_probe/detect_php_test.go` | TestProbe_CascadePrecedence +5 rows | ✓ VERIFIED | All 5 named rows present; pass in suite run |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `detectCSharp` | `readFirstManifest(folder, ".csproj")` | os.ReadDir (sorted, e.IsDir skip, exact-case suffix) → readManifest per candidate (1MB cap + BOM strip + WR-01 FIFO gate) | WIRED | Verified in source (detect_csharp.go:19-33, 66) + behavioral pass |
| `detectCSharp` | xml decode | inline `_ = xml.Unmarshal(content, &proj)` decode-error-ignored → collect-first-then-chain across PropertyGroups | WIRED | detect_csharp.go:71-93; multi-group collection pinned by TestProbe_CSharpMultiGroup |
| `detectJavaKotlin` | pom branch | `readManifest(folder, "pom.xml")` presence gate → inline decode-error-ignored unmarshal → Name/Version/Description chains with Parent.Version | WIRED | detect_java_kotlin.go:93-113; parent inheritance pinned by TestProbe_JavaParentVersion |
| `detectJavaKotlin` | gradle branch | `readManifest(folder, "settings.gradle")` ONLY when pom absent → parseRootProjectName → base fallback | WIRED | detect_java_kotlin.go:115-122; asymmetry pinned by TestProbe_JavaGarbagePomSkipsGradle |
| Probe → runDetectors → registry | detectors[2]=detectCSharp, detectors[5]=detectJavaKotlin | 7-slot ordered cascade, first match wins, panic recovery | WIRED | registry.go:14-22 + TestDetectorPositions + 5 cascade rows (mixed-manifest folders resolve C#@2 > Java@5 > PHP@6) |
| doc.go | detector list | all seven named with DETC-07/DETC-08 sentences | WIRED | doc.go:35-53; `grep detectCSharp|detectJavaKotlin` prints both |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| detectCSharp | Name/Version/Description | real .csproj file via readManifest (os.OpenFile + io.ReadAll, 1MB cap) | Yes — fixture files written to temp dirs, read through the production path | ✓ FLOWING |
| detectJavaKotlin (pom) | Name/Version/Description | real pom.xml via readManifest | Yes | ✓ FLOWING |
| detectJavaKotlin (gradle) | Name | real settings.gradle via readManifest | Yes | ✓ FLOWING |
| Description fallbacks | readmeDescription(folder) | real README.md read | Yes (existing Phase 10-12 mechanism, re-pinned by DescriptionFromManifest/ReadmeFallback/JavaDescriptionFallback rows) | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full project_probe suite (exercises every named Phase 13 row through real Probe calls on temp-dir fixtures) | `go test ./project_probe/...` | 236 passed in 1 packages | ✓ PASS |
| Module-wide build | `go build ./...` | exit 0 | ✓ PASS |
| Windows compilation (filepath portability, ROBT-04) | `GOOS=windows go vet ./project_probe/...` | exit 0, clean | ✓ PASS |
| Registry positions — all 7 live | `TestDetectorPositions` (in suite run) | pass — Len 7, all NotNil | ✓ PASS |
| Cascade precedence incl. gradle fallback arm | `TestProbe_CascadePrecedence` (in suite run) | pass — 5 appended rows present and green | ✓ PASS |
| TDD RED evidence (Phase 10 adaptation) | `git show ed720d6 --format=%B -s`, `git log 8b02ab2` | feat commits exist with RED evidence (undefined detectCSharp / detectJavaKotlin compile failures) in bodies | ✓ PASS |

### Probe Execution

N/A — no probes declared in any Phase 13 plan (probe-based verification was not part of this phase's contract).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DETC-07 | 13-01 (`requirements: [DETC-07]`) | C#/.NET detector — .csproj XML (namespace-agnostic, XMLName pattern) | ✓ SATISFIED | detect_csharp.go + 16 test rows; SC1/SC4 (C# part) green; REQUIREMENTS.md marks `[x] Complete` |
| DETC-08 | 13-02 (`requirements: [DETC-08]`) | Java/Kotlin detector — pom.xml (parent version inheritance), settings.gradle `rootProject.name` fallback | ✓ SATISFIED | detect_java_kotlin.go + 13 test rows + 5 cascade rows; SC2/SC3/SC4 (Java part) green; REQUIREMENTS.md marks `[x] Complete` |

**Orphaned requirements:** none — REQUIREMENTS.md maps only DETC-07 and DETC-08 to Phase 13, both declared and satisfied.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| project_probe/detect_csharp.go | 62 | "placeholders" token in doc comment | ℹ️ Info | False positive — documents raw-placeholder semantics (A2), not a stub; no placeholder code path exists |
| project_probe/registry_test.go | 16-20 | WR-02: TestRunDetectors_EmptyRegistry comment claims "injected slice" but runs the production registry | ⚠️ Warning (advisory) | Recorded in 13-REVIEW-DISPOSITION.md as open/advisory — test passes (runDetectors("") matches nothing: os.ReadDir("") errors); comment-only inaccuracy, no behavior impact, never blocks |
| project_probe/detect_csharp.go / detect_java_kotlin.go | (all) | WR-01: encoding/xml does not trim element text — padded values (e.g. `<Version> 1.2.3 </Version>`) would report with whitespace | ⚠️ Warning (advisory) | Recorded in 13-REVIEW-DISPOSITION.md as open/advisory — no SC requires trimming; not a blocker, no gap created |
| project_probe/detect_csharp.go / detect_java_kotlin.go | (all) | IN-01/IN-02/IN-03: doc-comment accuracy, gradle comment/expression degrade, `.csproj`-named-file edge | ℹ️ Info | All recorded open/advisory in 13-REVIEW-DISPOSITION.md; no truth affected |

No 🛑 blockers: no TBD/FIXME/XXX markers in any phase-modified file; no stubs, no empty implementations, no hardcoded-empty props; all data flows to real file reads.

### Human Verification Required

None. Every roadmap success criterion is behaviorally pinned by passing named unit tests that exercise the full production path (Probe → registry → detector → readManifest → decode → merge) on real temp-dir fixtures. No visual, real-time, external-service, or performance aspects apply to this Go library phase.

### Gaps Summary

No gaps. All 4 roadmap success criteria, all plan-frontmatter truths, the D-01..D-10 locked decisions, and every prohibition are verified against the actual codebase:

- SC1: XMLName local-name matching (`xml:"Project"` + plain child tags) — SDK-style, old-style xmlns, and BOM variants all decode identically (behavioral: 3 named tests pass).
- SC2: pom.xml → LanguageJava with `<name>`/`<artifactId>` and single-level `<parent><version>` inheritance from the child's own file (behavioral: TestProbe_JavaParentVersion passes).
- SC3: settings.gradle rootProject.name fallback fires only when no pom.xml; never-fail parser with exact-match/remainder guards (behavioral: matrix + probe rows pass).
- SC4: root-scoped only — single-level ReadDir with e.IsDir skip; subdirectory markers never trigger the parent (behavioral: 3 root-scope tests pass).
- Registry: 7-slot literal complete, order preserved, TestDetectorPositions all-7-NotNil, doc.go names all seven, cascade precedence pinned end-to-end.
- TDD gate: sanctioned RED-verified-but-uncommitted adaptation per Phase 10 precedent — RED evidence (compile failures) verified in feat commit bodies (`8b02ab2`, `ed720d6`).
- Known-red coverage (`release/update.go` 68.9%) is pre-existing, unrelated, and documented — not part of this phase's scope.

---

_Verified: 2026-09-29T23:10:00Z_
_Verifier: the agent (gsd-verifier)_
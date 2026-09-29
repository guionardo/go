---
phase: 13-xml-detectors-csharp-net-java-kotlin
verified: 2026-09-29T13:50:28Z
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
covered_digest: "v2:sha256:3ceebefc99870b3c12626c30b78e9e9bb3744abd3f2dd17dab4c197d69edf0a5"
behavior_unverified: 0
overrides_applied: 0
behavior_unverified_items: [] # 0 — every truth re-verified with behavioral evidence (284/284 suite run + direct named-test runs)
coincidental_reliance_items: [] # 0 — every verified truth holds on code-enforced behavior (real temp-dir fixtures through Probe), not fixtures/ordering
re_verification:
  previous_status: passed
  previous_score: 7/7
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 13: XML Detectors — C#/.NET + Java/Kotlin Verification Report

**Phase Goal:** C#/.NET and Java/Kotlin projects are detected through XML manifests — namespace-agnostic and root-scoped.
**Verified:** 2026-09-29T13:50:28Z
**Status:** passed
**Re-verification:** Yes — digest regeneration (#4682). Covered source files changed after the prior verifier ran: Phase 14 (`2ddacc0`, `bb19f8f`, `f23fe27`, `566544d`) modified `detect_csharp.go` / `detect_java_kotlin.go` (per-field TrimSpace fold-in — 13 WR-01 fix; IN-03 exact-name guard), `registry_test.go` (WR-02 explicit empty-registry injection), and `doc.go` (DATA-03 version-semantics section). All Phase 13 must-haves re-verified against the current codebase state — no regression found; the Phase 14 changes are improvements fixing advisories recorded in the prior report (WR-01, WR-02).

## Goal Achievement

### Observable Truths

Roadmap Success Criteria (the contract) + plan-level truths (merged, deduplicated) — all re-verified on the CURRENT codebase state:

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: Probe of a folder with .csproj returns Language=C#/.NET with name/version via XMLName local-name matching — SDK-style (no xmlns), old-style (xmlns msbuild/2003), and BOM-prefixed all decode identically | ✓ VERIFIED | `detect_csharp.go:44` — `csprojManifest` XMLName `xml:"Project"` + plain-tag `propertyGroup` (D-01); BOM stripped at readManifest boundary; chains `AssemblyName → RootNamespace → base`, `Version → VersionPrefix → ""` (D-03). Phase 14 fold-in (WR-01) added per-field `strings.TrimSpace` at decode time (lines 76-100) — decode hygiene only: the SDK/old-style/BOM rows use unpadded fixtures and decode identically. Behavioral: full `go test ./project_probe/...` = 284/284 pass incl. TestProbe_CSharpEndToEnd / OldStyleNamespaced / BOM; TestProbe_CSharpEndToEnd re-run directly: PASS |
| 2 | SC2: Probe of a folder with pom.xml returns Language=Java with name/version; child modules inherit `<parent><version>` single-level | ✓ VERIFIED | `detect_java_kotlin.go:18` — `pomManifest` XMLName `xml:"project"` (lowercase), `Parent *parentPOM` pointer; `Version → Parent.Version` when absent (lines 108-111, read from the probed file only — D-05, Pitfall 5); Name `<name> → <artifactId> → base`. Phase 14 TrimSpace fold-in trims each pom field pre-chain (lines 101-115) — parent inheritance preserved. Behavioral: TestProbe_JavaPomEndToEnd + TestProbe_JavaParentVersion pass in the 284/284 run; TestProbe_JavaParentVersion re-run directly: PASS (asserts LanguageJava, "Acme Child", "2.0.0") |
| 3 | SC3: Probe of a folder with settings.gradle (rootProject.name) and NO pom.xml returns Java/Kotlin with name from the gradle file; gradle fallback NEVER fires when pom.xml is present (even garbage) | ✓ VERIFIED | `parseRootProjectName` (lines 50-76) — utf8BOM strip at entry, i<0 skip-before-slice guard (never panics), exact key match (`rootProject.name` / `settings.rootProject.name`, never HasPrefix), quoted-literal + remainder guard ('a' + 'b' degrades to ""), first-wins (D-disc-3/4, D-09); pom-branch presence gate precedes the gradle branch (line 93 vs 119 — D-06 asymmetry). Behavioral: TestProbe_JavaGradleFallback, TestProbe_JavaPomWinsOverGradle, TestProbe_JavaGarbagePomSkipsGradle, TestParseRootProjectName (10-row matrix) all pass in the 284/284 run |
| 4 | SC4: A .csproj or pom.xml in a subdirectory never triggers the parent folder — root-scoped markers only | ✓ VERIFIED | `readFirstManifest` — single-level `os.ReadDir`, `e.IsDir()` skip, exact-case suffix (detect_csharp.go:19-36); Phase 14 IN-03 guard (`len(e.Name()) <= len(suffix)`, line 28) additionally rejects a file literally named ".csproj" — strengthens root-scope hygiene; `readManifest` root-scoped by construction. Behavioral: TestProbe_CSharpRootScope, TestProbe_CSharpSubdirNamedCsproj, TestProbe_JavaRootScope, TestDetectCSharp_ExactNameGuard (Phase 14 row) all pass in the 284/284 run |
| 5 | Registry complete: detectCSharp at index 2, detectJavaKotlin at index 5 — 7-slot literal `{detectGo, detectPython, detectCSharp, detectJS, detectRust, detectJavaKotlin, detectPHP}` all live; order preserved; doc.go names all seven | ✓ VERIFIED | `registry.go:14-22` literal inspected — all 7 slots non-nil, order Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP, all-seven-live comment intact; `doc.go:35-53` names all seven with per-detector sentences incl. both `detectCSharp` and `detectJavaKotlin` tokens (Phase 14 only ADDED the DATA-03 version-semantics section, lines 55-76, which documents the WR-01 trim as decode hygiene); TestDetectorPositions asserts `Len==7` and all 7 `NotNil` (registry_test.go:71-80 — unchanged by Phase 14's WR-02 fix to TestRunDetectors_EmptyRegistry); 5 cascade rows present in detect_php_test.go (C#@2 > Java@5 > PHP@6, pom and gradle arms) — all pass |
| 6 | Never-fail presence-match (D-09): malformed/empty manifests still match with fallback fields; missing manifests yield `(ProjectData{}, false)` — never an error, never a panic | ✓ VERIFIED | decode-error-ignored `_ = xml.Unmarshal(content, &proj/pom)` inline in both detectors (detect_csharp.go:74, detect_java_kotlin.go:95); `readFirstManifest`/`readManifest` non-match → `(ProjectData{}, false)`; `callDetector` recover net. Behavioral: TestProbe_CSharpMalformedPresence, TestProbe_CSharpEmptyFile, TestDetectCSharp_MissingManifest, TestDetectJavaKotlin_MissingManifest all pass in the 284/284 run |
| 7 | Prohibitions hold: no build-tool execution/network/symlink-following/version normalization; no custom Entity/CharsetReader (XXE-safe); no direct I/O beyond readManifest; no sibling/relativePath parent reads; no bare `path` import; no xml.go helper; no t.Parallel in registry_test.go | ✓ VERIFIED | Re-run against CURRENT state — all greps print nothing: `os/exec|net/http|EvalSymlinks|WalkDir` (both detector files), `Entity|CharsetReader` (both), `os\.Open|os\.ReadFile|io\.ReadAll` (detect_csharp.go — only pre-existing matches in manifest.go/manifest_test.go, outside this phase's files), `\.\./|relativePath` (detect_java_kotlin.go — only a comment in the test file), bare `"path"` import (both), `settings.gradle.kts` (detect_java_kotlin.go — only the test fixture); `test ! -f project_probe/xml.go` passes; registry_test.go has 0 non-comment `t.Parallel` occurrences (only the file-header NOTE mentions it) |

**Score:** 7/7 truths verified (0 present, behavior-unverified)

### Deferred Items

None. The v2 backlog items (REFN-02 distinct Kotlin value, REFN-03 lockfile confirmation, REFN-04 multi-.csproj selection, settings.gradle.kts support) are explicitly excluded from this milestone (13-CONTEXT.md Deferred Ideas) — no later phase in this milestone covers them, so they are out-of-scope exclusions, not deferred gaps.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | ----------- | ------ | ------- |
| `project_probe/detect_csharp.go` | readFirstManifest + csprojManifest + propertyGroup + detectCSharp (D-01..D-04) | ✓ VERIFIED | 119 lines, substantive; wired at registry index 2; Phase 14 added per-field TrimSpace (WR-01) + exact-name guard (IN-03) — both strengthen, none regress |
| `project_probe/detect_csharp_test.go` | probe-level rows + direct table | ✓ VERIFIED | 16 named tests (15 Phase 13 + Phase 14 ExactNameGuard); all pass in the 284/284 run |
| `project_probe/detect_java_kotlin.go` | pomManifest + parentPOM + parseRootProjectName + detectJavaKotlin (D-05..D-07) | ✓ VERIFIED | 129 lines, substantive; wired at registry index 5; Phase 14 TrimSpace fold-in preserves chains |
| `project_probe/detect_java_kotlin_test.go` | tracer rows + parse matrix + edge rows | ✓ VERIFIED | 13 named tests; all pass (Phase 14 appended its own rows) |
| `project_probe/registry.go` | 7-slot literal COMPLETE (D-08) | ✓ VERIFIED | All 7 slots live; nil-skip + callDetector unchanged |
| `project_probe/registry_test.go` | TestDetectorPositions final all-7-NotNil | ✓ VERIFIED | All 7 NotNil asserted (lines 71-80); no parallel marker; Phase 14 WR-02 fix (explicit empty registry) touches only TestRunDetectors_EmptyRegistry |
| `project_probe/doc.go` | All seven detectors named (DETC-07/DETC-08) | ✓ VERIFIED | Lines 35-53; stale "slots 2" text gone (`grep` exit 1); `detectCSharp`/`detectJavaKotlin` tokens present; never-fail paragraphs intact; Phase 14 added DATA-03 semantics section (documents the trim as decode hygiene) |
| `project_probe/detect_php_test.go` | TestProbe_CascadePrecedence +5 rows | ✓ VERIFIED | All 5 named rows present; pass in suite run |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `detectCSharp` | `readFirstManifest(folder, ".csproj")` | os.ReadDir (sorted, e.IsDir skip, exact-case suffix + IN-03 length guard) → readManifest per candidate (1MB cap + BOM strip + WR-01 FIFO gate) | WIRED | Verified in source (detect_csharp.go:19-36, 69) + behavioral pass |
| `detectCSharp` | xml decode | inline `_ = xml.Unmarshal(content, &proj)` decode-error-ignored → collect-first-then-trim-then-chain across PropertyGroups | WIRED | detect_csharp.go:74-100; multi-group collection pinned by TestProbe_CSharpMultiGroup |
| `detectJavaKotlin` | pom branch | `readManifest(folder, "pom.xml")` presence gate → inline decode-error-ignored unmarshal → trim → Name/Version/Description chains with Parent.Version | WIRED | detect_java_kotlin.go:93-117; parent inheritance pinned by TestProbe_JavaParentVersion |
| `detectJavaKotlin` | gradle branch | `readManifest(folder, "settings.gradle")` ONLY when pom absent → parseRootProjectName → base fallback | WIRED | detect_java_kotlin.go:119-126; asymmetry pinned by TestProbe_JavaGarbagePomSkipsGradle |
| Probe → runDetectors → registry | detectors[2]=detectCSharp, detectors[5]=detectJavaKotlin | 7-slot ordered cascade, first match wins, panic recovery | WIRED | registry.go:14-22 + TestDetectorPositions + 5 cascade rows (mixed-manifest folders resolve C#@2 > Java@5 > PHP@6) |
| doc.go | detector list | all seven named with DETC-07/DETC-08 sentences | WIRED | doc.go:35-53; `grep detectCSharp|detectJavaKotlin` prints both |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| detectCSharp | Name/Version/Description | real .csproj file via readFirstManifest/readManifest (os.OpenFile + io.ReadAll, 1MB cap) | Yes — fixture files written to temp dirs, read through the production path | ✓ FLOWING |
| detectJavaKotlin (pom) | Name/Version/Description | real pom.xml via readManifest | Yes | ✓ FLOWING |
| detectJavaKotlin (gradle) | Name | real settings.gradle via readManifest | Yes | ✓ FLOWING |
| Description fallbacks | readmeDescription(folder) | real README.md read | Yes (existing Phase 10-12 mechanism, re-pinned by DescriptionFromManifest/ReadmeFallback/JavaDescriptionFallback rows) | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full project_probe suite (exercises every named Phase 13 row through real Probe calls on temp-dir fixtures — CURRENT state incl. Phase 14 fold-in) | `go test ./project_probe/...` | 284 passed in 1 packages | ✓ PASS |
| Direct named-test re-run — SC1 row | `go test ./project_probe/... -run TestProbe_CSharpEndToEnd -v` | PASS | ✓ PASS |
| Direct named-test re-run — SC2 parent-inheritance invariant | `go test ./project_probe/... -run TestProbe_JavaParentVersion -v` | PASS | ✓ PASS |
| Module-wide build | `go build ./...` | exit 0 | ✓ PASS |
| Vet (native) | `go vet ./project_probe/...` | exit 0, clean | ✓ PASS |
| Windows compilation (filepath portability, ROBT-04) | `GOOS=windows go vet ./project_probe/...` | exit 0, clean | ✓ PASS |
| Registry positions — all 7 live | `TestDetectorPositions` (in suite run) | pass — Len 7, all NotNil | ✓ PASS |
| Cascade precedence incl. gradle fallback arm | `TestProbe_CascadePrecedence` (in suite run) | pass — 5 appended rows present and green | ✓ PASS |

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
| project_probe/detect_csharp.go / detect_java_kotlin.go | (all) | WR-01: padded XML element text (e.g. `<Version> 1.2.3 </Version>`) | ✅ RESOLVED (Phase 14) | The prior advisory is closed: per-field `strings.TrimSpace` folds in at decode time (2ddacc0) — padded values now report trimmed; whitespace-only elements no longer count as present and resolve to fallbacks (doc.go documents the trim as decode hygiene, never version normalization). No Phase 13 truth asserted padded-verbatim reporting |
| project_probe/registry_test.go | 16-20 | WR-02: comment claimed "injected slice" but ran the production registry | ✅ RESOLVED (Phase 14) | The prior advisory is closed: TestRunDetectors_EmptyRegistry now injects an explicit empty slice with restore (bb19f8f) — runDetectors("x") sees zero live detectors, no production-registry side effect |
| project_probe/detect_csharp.go | 28 | IN-03 exact-name guard (`len(e.Name()) <= len(suffix)`) | ℹ️ Info | Phase 14 strengthening: a file literally named ".csproj" can never match — root-scope hygiene improvement, pinned by TestDetectCSharp_ExactNameGuard; no Phase 13 truth affected (fixtures are always `MyApp.csproj`-style) |

No 🛑 blockers: no TBD/FIXME/XXX markers in any phase-modified file; no stubs, no empty implementations, no hardcoded-empty props; all data flows to real file reads.

### Human Verification Required

None. Every roadmap success criterion is behaviorally pinned by passing named unit tests that exercise the full production path (Probe → registry → detector → readManifest → decode → merge) on real temp-dir fixtures, re-run against the CURRENT codebase state after Phase 14's modifications. No visual, real-time, external-service, or performance aspects apply to this Go library phase.

### Gaps Summary

No gaps. Re-verification (#4682) confirms all 4 roadmap success criteria, all plan-frontmatter truths, the D-01..D-10 locked decisions, and every prohibition still hold after Phase 14 modified 4 covered files:

- **SC1**: XMLName local-name matching (`xml:"Project"` + plain child tags) — SDK-style, old-style xmlns, and BOM variants decode identically (behavioral: TestProbe_CSharpEndToEnd re-run directly PASS; suite 284/284).
- **SC2**: pom.xml → LanguageJava with `<name>`/`<artifactId>` and single-level `<parent><version>` inheritance from the child's own file (behavioral: TestProbe_JavaParentVersion re-run directly PASS).
- **SC3**: settings.gradle rootProject.name fallback fires only when no pom.xml; never-fail parser with exact-match/remainder guards (behavioral: matrix + probe rows pass in suite).
- **SC4**: root-scoped only — single-level ReadDir with e.IsDir skip; subdirectory markers never trigger the parent (behavioral: 3 root-scope tests pass; IN-03 guard strengthens).
- **Registry**: 7-slot literal complete, order preserved, TestDetectorPositions all-7-NotNil, doc.go names all seven, cascade precedence pinned end-to-end.
- **Phase 14 deltas are improvements, not regressions**: WR-01 TrimSpace fold-in (advisory closed), WR-02 explicit empty-registry injection (advisory closed), IN-03 exact-name guard (strengthening), doc.go DATA-03 section (documentation only).
- **Digest regenerated**: `covered_digest` recomputed over the unchanged 12-file `covered_files` list — old `71d48d9c...` → new `3ceebefc...` (files changed, list unchanged).

---

_Verified: 2026-09-29T13:50:28Z_
_Verifier: the agent (gsd-verifier)_
---
phase: 13
phase_name: "XML Detectors — C#/.NET + Java/Kotlin"
project: "github.com/guionardo/go"
generated: "2026-09-30"
counts:
  decisions: 8
  lessons: 4
  patterns: 5
  surprises: 4
missing_artifacts:
  - "13-UAT.md"
---

# Phase 13 Learnings: XML Detectors — C#/.NET + Java/Kotlin

## Decisions

### XML decode is inline — no xml.go/readXMLManifest helper
`_ = xml.Unmarshal(content, &proj)` lives inside detectCSharp/detectJavaKotlin; a shared helper was deliberately NOT created.

**Rationale:** An uncalled unexported helper sits at 0% file coverage and fails the `.testcoverage-quick.yml` file:70 gate (no project_probe override exists). Coverage-gate rationale dropped from the plan in revision.
**Source:** 13-01-SUMMARY.md

### readFirstManifest discovery for variable-named manifests
`.csproj` is a variable filename, so discovery precedes the read: single-level sorted `os.ReadDir`, exact-case suffix, `e.IsDir()` skip, per-candidate readManifest WR-01 gate — first readable candidate wins.

**Rationale:** D-04/A3/A6/A9 — root-scoped by construction; the subdirectory-marker rule (SC4) falls out of the single-level read.
**Source:** 13-01-SUMMARY.md

### Single-level parent-version inheritance from the child's OWN file
`<parent><version>` is REQUIRED in every child POM (Maven); what may be omitted is the child's own `<version>` — never a `../pom.xml`/relativePath read.

**Rationale:** D-05/Pitfall 5 — root-scoped (D-07/ROBT-05); sibling traversal would violate the probe's folder boundary.
**Source:** 13-02-SUMMARY.md

### settings.gradle fallback ONLY when pom.xml absent
A present-but-garbage pom skips the gradle fallback (D-06 asymmetry); `.kts` variant NOT read (A5 literal) — a .kts-only folder yields LanguageUnknown; LanguageJava for both branches (REFN-02 deferred).

**Rationale:** The pom file IS the ecosystem marker; the fallback is for pom-less folders only. Pinned by TestProbe_JavaGarbagePomSkipsGradle + TestProbe_JavaGradleKtsNotRead.
**Source:** 13-02-SUMMARY.md

### XMLName lowercase for pom.xml — XML is case-sensitive
`pomManifest` uses `xml:"project"` LOWERCASE root — never doubles for .csproj's `xml:"Project"` (Pitfall 7); `Parent *parentPOM` pointer allocated only when `<parent>` present.

**Rationale:** One struct decodes both SDK-style and old-style xmlns files (namespace-agnostic, D-01); case sensitivity is a hard XML rule.
**Source:** 13-02-SUMMARY.md

### parseRootProjectName never-fail contract
UTF-8 BOM strip at entry, `i < 0` skip-before-slice guard (never panics), exact left-of-`=` key equality (never HasPrefix — comments can't fabricate), quoted-literal-only + remainder guard (`'a' + 'b'` concatenations degrade to folder-base Name), first-wins.

**Rationale:** D-disc-3/4, D-09 — Groovy assignment lines are never evaluated (no interpolation, no GString handling); degrade is the contract.
**Source:** 13-02-SUMMARY.md

### Collect-first-then-chain across ALL PropertyGroups
AssemblyName priority is independent of group position — all PropertyGroups collected in document order, then chained (Version → VersionPrefix → empty, never the MSBuild-implicit 1.0.0).

**Rationale:** D-disc-6, D-03/DATA-03 — `$(...)` placeholders reported raw (A2), never resolved.
**Source:** 13-01-SUMMARY.md

### Placeholder versions reported raw
`${revision}` verbatim, never resolved/stripped (pinned by TestProbe_JavaPlaceholderRaw).

**Rationale:** A2/DATA-03 raw-manifest semantics — resolution would fabricate data.
**Source:** 13-02-SUMMARY.md

---

## Lessons

### Grep-token-in-comment traps are systemic
Literal tokens in doc comments trip the plan's literal acceptance greps: `os.Open` (13-01), `makeFIFO` (13-01), `relativePath` (13-02), `settings.gradle.kts` (13-02) — 4 occurrences across 2 plans, all cosmetic rewordings, no behavior change.

**Context:** Rule 1 deviations — a doc comment describing a prohibition matches the grep that verifies the prohibition. Reword with synonyms ("a plain open", "FIFO-creating helper", "read from the probed file only", "The Kotlin-DSL settings variant is NOT read").
**Source:** 13-01-SUMMARY.md

### Acceptance greps can require tokens that don't exist yet
Task 1's grep required `detectCSharp|detectJavaKotlin` tokens in doc.go — the refreshed paragraph named the languages (LanguageCSharp/LanguageJava) but not the functions, failing the grep.

**Context:** Post-commit fix `3558d8e` — reworded to "reports LanguageCSharp via detectCSharp". Greps must be validated against the acceptance criteria BEFORE commit.
**Source:** 13-02-SUMMARY.md

### XML element text is not whitespace-trimmed
Padded values (`<Version> 1.2.3 </Version>`) flow verbatim and block the fallback chains; whitespace-only elements count as present (WR-01 from 13-REVIEW).

**Context:** Resolved in Phase 14 (per-field `strings.TrimSpace` at decode time, documented as decode hygiene — never version normalization). Padded-verbatim was never a Phase 13 truth.
**Source:** 13-VERIFICATION.md

### The SDK's --is-protected verb echoes a name, not a boolean
`gsd_run query git.base-branch --is-protected <branch>` prints the configured base branch name ("main") rather than true/false — the strict non-"false" reading would false-FATAL every branch.

**Context:** Both 13-01 and 13-02 — resolved by comparing the checked branch against the echoed base name (semantic intent: never commit onto the base).
**Source:** 13-02-SUMMARY.md

---

## Patterns

### Variable-named manifest discovery via readFirstManifest
Sorted os.ReadDir → exact-case suffix → per-candidate readManifest gate — the discovery step for files whose names vary (MyApp.csproj, Tool.csproj).

**When to use:** Detectors for variable-named manifests; single-level keeps root-scope; the IN-03 length guard (`len(e.Name()) <= len(suffix)`) rejects files literally named ".csproj".
**Source:** 13-01-SUMMARY.md

### Inline decode-error-ignored xml.Unmarshal with XMLName local-name structs
`_ = xml.Unmarshal(content, &proj)` + `xml:"Project"`/`xml:"project"` — namespace-agnostic: one struct decodes SDK-style, old-style xmlns, and BOM-prefixed files; decode errors → zero struct → presence-match fallback fields.

**When to use:** XML manifests with varying namespaces; decode-error-ignored is the never-fail degrade (D-09), and inline keeps coverage-gate compliance.
**Source:** 13-01-SUMMARY.md

### Never-fail line parser with i<0 skip-before-slice guard
parseRootProjectName: BOM strip → i<0 guard → exact-key → quoted-literal + remainder guard → first-wins — blank/unrelated/empty lines never panic.

**When to use:** Any settings-file key-value parsing where malformed input must degrade silently; the i<0 guard before slicing is the never-panic invariant.
**Source:** 13-02-SUMMARY.md

### Cascade-order participation of a fallback arm pinned by mixed-manifest rows
The settings.gradle arm's cascade position is proven by `settings.gradle + composer.json → Java` — a fallback branch must be pinned in cascade tests, not just unit-tested alone.

**When to use:** Detectors with fallback branches — mixed-manifest probe rows prove the branch participates in precedence (C#@2 > Java@5 > PHP@6, 5 rows).
**Source:** 13-02-SUMMARY.md

### Interim registry flip — exactly one assertion per plan
TestDetectorPositions changes one assertion per plan (detectors[2] NotNil in 13-01; detectors[5] Nil→NotNil in 13-02); no parallel marker, no comment edits.

**When to use:** Multi-plan registry fills — the interim pin preserves the next plan's RED step and documents mid-fill state (Pitfall 9).
**Source:** 13-01-SUMMARY.md

---

## Surprises

### A shared helper would fail the coverage gate
The "obvious" refactor — a shared readXMLManifest helper — would sit at 0% file coverage (uncalled in tests) and fail the file:70 gate, forcing the inline decode instead.

**Impact:** Shaped the detector architecture: inline decode-error-ignored in each detector, one small duplication traded for gate compliance.
**Source:** 13-01-SUMMARY.md

### XML text extraction does not trim whitespace
`<Version> 1.2.3 </Version>` reported `" 1.2.3 "` verbatim — padded values blocked the VersionPrefix/README fallback chains (WR-01).

**Impact:** Required the Phase 14 TrimSpace fold-in across both XML detectors; whitespace-only elements now resolve to fallbacks instead of counting as present.
**Source:** 13-VERIFICATION.md

### A .kts-only folder yields LanguageUnknown
settings.gradle.kts is deliberately NOT read (A5 literal) — a Kotlin-DSL-only project is invisible to the probe.

**Impact:** Documented as a v2 backlog item (A5 support) alongside REFN-02 (distinct Kotlin value); the .kts-not-read asymmetry is pinned by TestProbe_JavaGradleKtsNotRead.
**Source:** 13-02-SUMMARY.md

### Garbage pom beats valid gradle settings
`TestProbe_JavaGarbagePomSkipsGradle` — a malformed pom.xml still claims Java at index 5 over a valid settings.gradle; the gradle fallback only fires when the pom is absent entirely.

**Impact:** The D-06 asymmetry is the most surprising cascade rule in the registry; pinned by the garbage-pom row and documented in doc.go.
**Source:** 13-02-SUMMARY.md
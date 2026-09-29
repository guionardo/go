# Phase 13: XML Detectors — C#/.NET + Java/Kotlin - Research

**Researched:** 2026-09-29
**Domain:** `encoding/xml` namespace-agnostic decoding, .csproj (MSBuild) metadata, Maven POM metadata + parent version inheritance, settings.gradle `rootProject.name`, registry-slot completion (7/7)
**Confidence:** HIGH

## Summary

Phase 13 fills the last two registry slots — `detectCSharp` at index 2 and `detectJavaKotlin` at index 5 — completing the 7-slot literal `{detectGo, detectPython, detectCSharp, detectJS, detectRust, detectJavaKotlin, detectPHP}` (D-08). The parsing core is `encoding/xml` with XMLName **local-name matching** (D-01): a struct tag of the form `xml:"Project"` (name only, no namespace) requires only the element's LOCAL name to match, making the same decode struct work for SDK-style `<Project Sdk="...">`, old-style `<Project ToolsVersion="..." xmlns="http://schemas.microsoft.com/developer/msbuild/2003">`, and namespaced `<project xmlns="http://maven.apache.org/POM/4.0.0">`. This behavior is **verified live on Go 1.27.0** (12-case probe run this session: root match with/without xmlns, child elements in foreign namespaces matching plain tags, BOM-prefixed XML parsing cleanly, root-name mismatch returning `UnmarshalError "expected element type <Project> but have <Projects>"`, multiple `<PropertyGroup>` elements collecting in document order, CDATA/comments/processing instructions handled, case-sensitive matching, unknown elements discarded) and **corroborated by the official docs** [CITED: pkg.go.dev/encoding/xml]: *"If the struct has a field named XMLName of type Name, Unmarshal records the element name in that field. If the XMLName field has an associated tag of the form `"name"` or `"namespace-URL name"`, the XML element must have the given name (and, optionally, name space) or else Unmarshal returns an error."*

Three ecosystem facts drive the detector shapes. (1) **MSBuild/.csproj**: the root element is *always* `Project` [CITED: learn.microsoft.com MSBuild project file schema]; properties live in `<PropertyGroup>` elements (multiple groups with `Condition` attributes are common — Debug/Release); `Version` is the most-used version property and defaults to `VersionPrefix[-VersionSuffix]` while an explicit `Version` overrides both [CITED: andrewlock.net/version-vs-versionsuffix-vs-packageversion]; Visual Studio writes UTF-8-with-BOM files (hence SC1's "with BOM" requirement, satisfied at the readManifest boundary per D-02). The C# detector needs one genuinely new mechanism vs every prior detector: `.csproj` is NOT a fixed filename, so `readManifest(folder, name)` cannot be called directly — discovery via `os.ReadDir(folder)` (entries sorted by filename, single level, root-scoped) + exact-case `.csproj` suffix filter (Phase 11 D-08 exact-case precedent), first **readable** candidate wins (deterministic stopgap; REFN-04's full multi-csproj selection rule is v2). (2) **Maven POM**: the research question "is `<parent><version>` in the child's own pom.xml or a sibling file?" is settled — **the child's own pom.xml**; Maven requires the parent block's version in every child POM ("You always have to specify parent's version"; what may be omitted is the *child's own* `<version>`, which then inherits the parent's) [CITED: maven.apache.org/pom.html, introduction-to-the-pom.html, stackoverflow 10582054]. So D-05's "inherit `<parent><version>`" is a single-file read — no sibling traversal, consistent with ROBT-05 root-scoped. `<name>` is the optional display name defaulting to `<artifactId>` [CITED: maven.apache.org/pom.html] — matching D-05's chain exactly. (3) **Gradle**: `rootProject.name = 'root-project'` (single quotes conventional in Groovy; double quotes valid and interpolate — never evaluate), fully-qualified `settings.rootProject.name = ...` also documented; the settings file lives in the project root [CITED: docs.gradle.org settings_file_basics + writing_settings_files]. D-06 locks `settings.gradle` only (the `.kts` variant is out of scope — flagged A5).

**Primary recommendation:** Two plans mirroring Phase 12's P02/P03 shape — **P01 = `detect_csharp.go` (`readFirstManifest` discovery + `csprojManifest` decode + chains, XML decode INLINE) + registry slot 2 + interim `TestDetectorPositions` flip (`detectors[2]` NotNil, `detectors[5]` stays Nil)**; **P02 = `detect_java_kotlin.go` (pom.xml + `<parent><version>` single-level inheritance + settings.gradle `rootProject.name` fallback ONLY when no pom.xml, same inline decode shape) + registry slot 5 + FINAL `TestDetectorPositions` (all 7 NotNil) + `doc.go` refresh + cascade rows**. There is **NO `xml.go` / `readXMLManifest` helper** — the shared-helper design was dropped in revision: an unexported helper with no production caller sits at 0% file coverage and fails the `.testcoverage-quick.yml` file:70 gate (no project_probe override); both detectors inline the 3-line decode instead (`_ = xml.Unmarshal(content, &v)` decode-error-ignored, the D-09 presence-match degrade). Both detectors match on manifest **presence** (D-09): a garbage .csproj/pom.xml still yields Language=C#/.NET or Java with empty fields and fallbacks; `readManifest` failure → `(ProjectData{}, false)`; malformed XML → empty fields, never a panic (xml.Unmarshal returns errors, never panics on malformed input; `callDetector` recover stays the outer net). Tests stay inline `t.TempDir()` + `os.WriteFile` with Go raw-string XML fixtures (repo convention — D-disc-7 precedent; XML contains no backticks so raw literals are safe). Inherited operational facts re-verified this session: `make coverage-quick` remains known-red on `release/update.go` 68.9% only (assert project_probe rows + total); 185 project_probe tests green; `_csharp`/`_java`/`_kotlin`/`_gradle` filename suffixes verified safe against `go tool dist list` (none in GOOS or GOARCH — the Phase 11 `_js` lesson); golangci-lint stays advisory (broken under default toolchain, CI runs no lint job).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### XML Parsing (DETC-07)
- **D-01:** `encoding/xml` with XMLName local-name matching (namespace-agnostic by LOCAL name, not URI) — stdlib-only, no dependencies. Works with or without `xmlns` on the root (SC1).
- **D-02:** BOM handled by readManifest (already stripped) — SC1's "with BOM" requirement is satisfied at the read boundary.

#### .csproj Detector (DETC-07)
- **D-03:** `detectCSharp(folder) (ProjectData, bool)` — `.csproj` via readManifest; Name chain: `AssemblyName` → `RootNamespace` → folder base (DATA-02); Version: `Version` → `VersionPrefix` → empty (never fabricated); Description → readmeDescription (DATA-04).
- **D-04:** Root-scoped only: `.csproj` in a subdirectory never triggers the parent folder (SC4).

#### Java/Kotlin Detector (DETC-08)
- **D-05:** `detectJavaKotlin(folder) (ProjectData, bool)` — pom.xml primary; Name: `<name>` → `<artifactId>` → folder base; Version: `<version>` → when absent inherit `<parent><version>` from parent POM (SC2, single level — no transitive chain).
- **D-06:** settings.gradle `rootProject.name` fallback ONLY when no pom.xml (SC3); Language = LanguageJava for both pom.xml and settings.gradle (Kotlin-distinct value is v2 REFN-02, NOT this phase).
- **D-07:** Root-scoped only: pom.xml in a subdirectory never triggers the parent folder (SC4).

#### Registry Wiring
- **D-08:** `detectCSharp` at index 2, `detectJavaKotlin` at index 5 — completes the 7-slot literal `{detectGo, detectPython, detectCSharp, detectJS, detectRust, detectJavaKotlin, detectPHP}`; TestDetectorPositions final state (all 7 non-nil); doc.go names all seven.
- **D-09:** Both detectors follow the never-fail presence-match contract: manifest presence = match, parse failure → empty/fallback fields, `readManifest` failure → `(ProjectData{}, false)`.
- **D-10:** Detector ORDER confirmed (STATE.md research decision lands here): Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP.

### the agent's Discretion
- Exact XML struct shapes for decoding, settings.gradle line parsing, fixture layout, test row organization. Follow repo conventions (CONVENTIONS.md) and prior-phase package patterns (PATTERNS.md).

### Deferred Ideas (OUT OF SCOPE)
- Distinct Kotlin Language value — v2 backlog (REFN-02)
- Lockfile secondary confirmation (package-lock.json, Cargo.lock, go.sum) — v2 (REFN-03)
- Multiple `.csproj`/`.sln` selection rule for .NET monorepos — v2 (REFN-04)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DETC-07 | C#/.NET detector — .csproj XML (namespace-agnostic, XMLName pattern) | XMLName local-name matching verified live on Go 1.27.0 (probe rows A/B/I: `xml:"Project"` matches `<Project xmlns=...>`, `<Project Sdk=...>`, and children in foreign namespaces) + [CITED: pkg.go.dev/encoding/xml] tag-form rule; `.csproj` discovery requirement (non-fixed filename) resolved via `os.ReadDir` + exact-case `.csproj` suffix (Pattern 2); `LanguageCSharp Language = "C#/.NET"` [VERIFIED: project_probe/project.go:39]; MSBuild property semantics [CITED: learn.microsoft.com + andrewlock.net] |
| DETC-08 | Java/Kotlin detector — pom.xml (parent version inheritance), settings.gradle `rootProject.name` fallback | `<parent><version>` lives in the CHILD's own pom.xml — Maven requires it; child's own `<version>` may be omitted and inherits [CITED: maven.apache.org/pom.html, introduction-to-the-pom.html, SO 10582054]; `<name>` optional, defaults to `<artifactId>` [CITED: maven.apache.org/pom.html]; `rootProject.name = 'x'`/`"x"` forms + root-level placement [CITED: docs.gradle.org settings_file_basics + writing_settings_files]; `LanguageJava Language = "Java"` [VERIFIED: project_probe/project.go:45]; single-level parent read verified by probe row G (Parent pointer decodes from the child file) |
</phase_requirements>

## Architectural Responsibility Map

Single-package stdlib library — no browser/SSR/CDN/database tiers exist. Every capability belongs to the package core; the map prevents the planner from pushing parsing into tests, docs, or other packages.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| XML decode (inline) | API/Backend (package core) | — | Inline `_ = xml.Unmarshal(content, &v)` decode-error-ignored inside `detect_csharp.go` and `detect_java_kotlin.go` (D-09 presence-match degrade); NO shared `xml.go`/`readXMLManifest` helper — an uncalled unexported helper would fail the file:70 coverage gate (dropped in revision; the `readJSONManifest` analog [VERIFIED: project_probe/json.go:11-17] informed the shape but is not reproduced) |
| .csproj discovery | API/Backend (package core) | — | `detect_csharp.go` — `readFirstManifest(folder, ".csproj")` via `os.ReadDir` (single level, root-scoped, sorted); the ONE new mechanism vs prior fixed-name detectors |
| C# metadata extraction | API/Backend (package core) | — | `detect_csharp.go` — D-03 chains (AssemblyName → RootNamespace → folder base; Version → VersionPrefix → empty; DATA-04 description) |
| Java/Kotlin metadata extraction | API/Backend (package core) | — | `detect_java_kotlin.go` — pom.xml decode + `<parent><version>` single-level (D-05) + settings.gradle fallback only-when-no-pom (D-06) |
| Registry position ownership | API/Backend (package core) | — | `registry.go` literal — fill indices 2 and 5 in place (D-08); no reordering |
| Manifest I/O safety | API/Backend (package core) | — | Existing `readManifest` (1 MB cap + BOM strip + WR-01 FIFO gate) [VERIFIED: project_probe/manifest.go:13,15,29-43,50] — consumed unchanged for pom.xml/settings.gradle and per discovered .csproj candidate |
| Description fallback | API/Backend (package core) | — | Existing `readmeDescription` [VERIFIED: project_probe/readme.go:29-48] — consumed unchanged |
| Docs (detector list) | Docs | — | `doc.go` paragraph refresh — all seven detectors named (D-08) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `encoding/xml` | 1.26.4 (go.mod) | Namespace-agnostic XML decoding of .csproj/pom.xml via XMLName local-name matching | D-01 locked; stdlib-only (ROBT-04); behavior verified live on the actual toolchain this session (12-case probe) |
| Go stdlib `os` | 1.26.4 | `os.ReadDir` for `.csproj` discovery | Entries sorted by filename (deterministic first-candidate rule); Phase 10 `Probe` precedent [VERIFIED: project_probe/probe.go:34] |
| Go stdlib `strings` | 1.26.4 | `.csproj` suffix filter (`HasSuffix`), settings.gradle line parsing (`Split`, `TrimSpace`, `HasPrefix`) | Phase 11 parseGoMod precedent [VERIFIED: project_probe/detect_go.go:43-68] |
| Go stdlib `path/filepath` | 1.26.4 | `Base` (folder-name fallback, DATA-02) | ROBT-04 — never `path`; Phase 11/12 precedent |
| `github.com/stretchr/testify` | v1.11.1 (test-only, existing) | `assert`/`require` | AGENTS.md mandate; already in go.mod [VERIFIED: go.mod:12] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| none | — | — | No supporting libraries needed — BOM, cap, FIFO gate, README extraction, panic recover all exist |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `xml.Unmarshal` (whole content) | `xml.Decoder` streaming | Locked by content shape: readManifest caps at 1 MB, so whole-content Unmarshal is the simple, correct tool; Decoder adds token-loop complexity for zero benefit at this size. Decoder matters only for incremental/large documents or custom CharsetReader needs — none apply [CITED: pkg.go.dev/encoding/xml] |
| Namespace-URI matching (`xml:"http://... Project"`) | Local-name matching (`xml:"Project"`) | Locked by D-01. The URI form requires the element's Space to equal the URI — an SDK-style csproj (no xmlns) would FAIL to match old-style structs and vice versa. Local-name form matches both (probe rows A/B) |
| Per-namespace decode structs | One struct per manifest, tags without namespace | Unnecessary: child tags without a namespace match elements in ANY namespace (probe row I) — a single `csprojManifest`/`pomManifest` covers namespaced and non-namespaced files |
| Fixed-name `readManifest(folder, "*.csproj")` | `os.ReadDir` discovery | `readManifest` opens a concrete path — a glob string fails at open. Discovery is the only correct mechanism for a variable-named manifest |
| Regex/generic XML tree walk (`etree`-style) | Struct-based `xml.Unmarshal` | Struct tags ARE the extraction contract (D-01 XMLName pattern); a tree walk would re-implement matching by hand (Don't Hand-Roll) |
| settings.gradle evaluation (Groovy GString interpolation) | Literal quote extraction, never evaluate | ROBT-05 anti-feature (no build-tool execution); interpolation is an execution semantics — extract the literal only (Pattern 4, D-disc-3) |

**Installation:** none — stdlib only; testify already present.
**Version verification:** `go.mod` declares `go 1.26.4` [VERIFIED: go.mod:3]; `github.com/stretchr/testify v1.11.1` [VERIFIED: go.mod:12]. Local toolchain go1.27.0 (darwin/arm64) — the same toolchain the encoding/xml probe ran on; CI pins via `go-version-file: go.mod`.

## Package Legitimacy Audit

> Gate outcome: **N/A — no external packages installed by this phase.** All runtime code is Go stdlib (`encoding/xml`, `os`, `strings`, `path/filepath`); `testify` is an established module dependency used test-only. The planner must not add any `go get` task — same disposition as Phases 11/12 (go ecosystem not covered by the package-legitimacy seam; gate satisfied by zero installs).

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|----------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/stretchr/testify` v1.11.1 | Go modules | ~10 yrs | widely used | stretchr/testify | OK (existing dep) | Approved — already in go.mod, no install action |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                        ┌──────────────────────────────────────────────┐
                        │          projectprobe.Probe(folder)           │
                        └───────────────────────┬──────────────────────┘
                                                │ clean-once → stat → readdir → hasContent (Phase 10, unchanged)
                                                ▼
                                   ┌──────────────────────────────────────────┐
                                   │   runDetectors(clean)  [registry.go]     │
                                   │   detectors = []detectorFunc{             │
                                   │       detectGo,         // idx 0 (P11)    │
                                   │       detectPython,     // idx 1 (P12)    │
                                   │       detectCSharp,     // idx 2 (THIS)   │
                                   │       detectJS,         // idx 3 (P11)    │
                                   │       detectRust,       // idx 4 (P12)    │
                                   │       detectJavaKotlin, // idx 5 (THIS)   │
                                   │       detectPHP,        // idx 6 (P11)    │
                                   │   }   ← 7/7 live — final state (D-08)    │
                                   │   nil-skip + callDetector recover (both   │
                                   │   already in place — unchanged)           │
                                   └──────┬───────────────────────┬───────────┘
                                          ▼                       ▼
                                    detectCSharp           detectJavaKotlin
                                          │                       │
                    os.ReadDir(folder) → first readable    readManifest(folder,"pom.xml")
                      ".csproj" candidate (sorted,         → pomManifest decode
                      root-scoped, exact-case)             → version absent? parent.version
                    → readManifest(folder, name)             (single level, D-05)
                    → csprojManifest decode                 └─ no pom.xml? (D-06)
                    → chain AssemblyName → RootNamespace      readManifest(folder,"settings.gradle")
                      → folder base (D-03)                    → parseRootProjectName literal
                    → Version → VersionPrefix → ""            → folder-base fallback
                    → Description → readmeDescription       → readmeDescription fallback
                                          │                       │
                                          └───────► ProjectData{Language, Name, Version, Description}
                                                       merged into Probe's defaults — first match wins
```

Entry point: `Probe` (unchanged). Processing: existing pipeline → registry (final 7/7) → two new detectors → `readManifest`/`readFirstManifest` reads with INLINE `xml.Unmarshal` decode (no shared XML helper — dropped in revision, see map row) → shared fallbacks. Branching: both detectors match on manifest presence (D-09) or fall through; the Java detector branches pom.xml-first then settings.gradle only when pom.xml is absent (D-06); XML decode failure degrades to empty fields, never errors/panics. External dependencies: filesystem only (ROBT-05 anti-features preserved — single-level reads, no symlink following, no recursion).

### Recommended Project Structure

```
project_probe/
├── registry.go              # MODIFIED: detectCSharp at idx 2, detectJavaKotlin at idx 5 (D-08); comment refresh
├── detect_csharp.go         # NEW: readFirstManifest (.csproj discovery) + csprojManifest + detectCSharp with INLINE xml.Unmarshal decode (D-03/D-04)
├── detect_java_kotlin.go    # NEW: pomManifest + parseRootProjectName + detectJavaKotlin with INLINE xml.Unmarshal decode (D-05/D-06/D-07)
├── doc.go                   # MODIFIED: detector-list paragraph names all seven (D-08)
├── detect_csharp_test.go    # NEW: probe-level rows (inline t.TempDir + os.WriteFile; BOM/xmlns/multi-group/malformed/root-scope)
├── detect_java_kotlin_test.go # NEW: probe-level rows (pom canonical/parent version/gradle fallback/malformed/root-scope)
├── registry_test.go         # MODIFIED: TestDetectorPositions — interim flip (idx 2) in P01, final (idx 5) in P02
├── detect_php_test.go       # MODIFIED: TestProbe_CascadePrecedence — add C#/Java precedence rows (P02)
└── (everything else)        # UNCHANGED — readManifest, readme, json, toml, probe, ignore, errors, example
```

Conventions honored: snake_case files; internal `package projectprobe` tests (unexported `readFirstManifest`, `parseRootProjectName` reachable — the decode is inline, so there is no `readXMLManifest` to test); `t.Parallel()` in fixture-only tests but NEVER in registry-mutating tests [VERIFIED: project_probe/registry_test.go:12-14]; decorder (type → const → var → func); composite literals with field names; `//nolint:paralleltest` on global-state tests. Filename suffixes verified build-safe this session via `go tool dist list`: `csharp`, `java`, `kotlin`, `gradle` appear in neither the GOOS list (`aix android darwin dragonfly freebsd illumos ios js linux netbsd openbsd plan9 solaris wasip1 windows`) nor the GOARCH list (`386 amd64 arm arm64 loong64 mips mips64 mips64le mipsle ppc64 ppc64le riscv64 s390x wasm`) — unlike Phase 11's `_js` GOOS collision.

### Pattern 1: XMLName local-name matching — the verified decode contract (D-01)

**What:** The single load-bearing behavior of the phase, verified live on Go 1.27.0 (probe run this session, 12 rows) and corroborated by the official docs [CITED: pkg.go.dev/encoding/xml]: *"If the XMLName field has an associated tag of the form `"name"` or `"namespace-URL name"`, the XML element must have the given name (and, optionally, name space) or else Unmarshal returns an error."* Concretely:

| Probe | Input | Result (Go 1.27.0) |
|-------|-------|--------------------|
| A | `<Project Sdk="Microsoft.NET.Sdk">` no xmlns | matches `xml:"Project"`; AssemblyName/Version extracted |
| B | `<Project ToolsVersion="15.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">` | matches the SAME struct; Space recorded in XMLName; children extracted |
| C | `<Projects>` root | `UnmarshalError: expected element type <Project> but have <Projects>` — detector degrades (never-fail) |
| D | BOM-prefixed (`EF BB BF`) content | parses cleanly — xml.Unmarshal tolerates a leading UTF-8 BOM; readManifest strip (D-02) is belt-and-suspenders |
| E | malformed XML (`<Project><PropertyGroup><AssemblyName>X`) | `XML syntax error: unexpected EOF` — error, never panic |
| F | `<project xmlns="http://maven.apache.org/POM/4.0.0">` canonical pom | Name/ArtifactID/Version/Description all extracted (lowercase root, namespaced) |
| G | child pom with `<parent><version>2.0.0</version></parent>`, no own version | Parent pointer allocated from the CHILD's own file — single-level inheritance read is a same-file read |
| H | XMLName field WITHOUT tag | records ANY root name (Space+Local) — the no-constraint form |
| I | child `<AssemblyName xmlns="urn:other">` | matches plain `xml:"AssemblyName"` tag — child tags without namespace are namespace-agnostic too |
| J | multiple `<PropertyGroup>` (Debug/Release) | slice collects ALL groups in document order — chain must scan across groups |
| K | CDATA / `<!-- -->` comment / `<?target ?>` PI | all handled; CDATA content lands in the string field |
| L | `<NAME>` vs field `xml:"Name"` | NOT matched — XML matching is case-sensitive (per docs: "Unmarshal uses a case-sensitive comparison to match XML element names to tag values and struct field names") |

Additional doc-verified rules that keep the detectors small: unknown elements and attributes are discarded ("Well-formed data that does not fit into v is discarded"); a missing element unmarshals as the zero value; pointer fields are allocated only when the element is present; input is assumed UTF-8 ("The parser assumes that its input is encoded in UTF-8" — a UTF-16 manifest errors → degrade, never fabricate) [CITED: pkg.go.dev/encoding/xml].

**When to use:** D-01 — both detectors. Two distinct XMLName tags are REQUIRED: `xml:"Project"` (capital P — csproj) and `xml:"project"` (lowercase — Maven); XML is case-sensitive so one struct can never serve both.

### Pattern 2: `readFirstManifest` discovery + inline XML decode (D-03/D-04)

**What:** The one genuinely new mechanism of the phase: `.csproj` is a **variable** filename, so discovery precedes the read. `os.ReadDir` returns entries sorted by filename [CITED: pkg.go.dev/os#ReadDir], which makes "first candidate" deterministic. The decode itself is INLINE in `detectCSharp` (`_ = xml.Unmarshal(content, &proj)` decode-error-ignored — Pattern 3): there is NO shared `readXMLManifest` helper. The `readJSONManifest` shape [VERIFIED: project_probe/json.go:11-17] informed the approach, but reproducing it as `xml.go` was dropped in revision — an unexported helper with no production caller sits at 0% file coverage and fails the `.testcoverage-quick.yml` file:70 gate (no project_probe override).

```go
// readFirstManifest returns the content of the first READABLE *.csproj entry
// in folder (single level, root-scoped — D-04/ROBT-05). os.ReadDir sorts by
// filename, so the choice is deterministic. Exact-case suffix matching per
// the Phase 11 D-08 precedent (os.Open alone resolves case-insensitively on
// macOS/Windows volumes — the directory entries are the source of truth).
// Multiple .csproj files (monorepo layouts) degrade to the first readable
// one; the full selection rule is deferred to v2 (REFN-04).
func readFirstManifest(folder, suffix string) (content []byte, ok bool) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, false
	}
	for _, e := range entries { // sorted by filename
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		if content, ok := readManifest(folder, e.Name()); ok { // FIFO/regular gate applies per candidate
			return content, true
		}
	}
	return nil, false
}
```

**When to use:** `detectCSharp` only. The ignore list is intentionally NOT consulted here — same visibility as every other detector (they match exact root-level filenames); `hasContent` already gates whether `Probe` runs detectors at all [VERIFIED: project_probe/probe.go:39-42]. A subdirectory named `foo.csproj` is skipped by `e.IsDir()`.

### Pattern 3: `detectCSharp` — chains across all PropertyGroups (DETC-07, D-03/D-04/D-09)

**What:** Presence-match on the first readable `.csproj` (D-09). Decode into a struct whose tags carry NO namespace (Pattern 1) — one struct covers SDK-style and old-style files. D-03's chains resolve across ALL `<PropertyGroup>` elements in document order (probe row J: multiple groups are the norm): collect-first-then-chain implements "AssemblyName → RootNamespace → folder base" with AssemblyName priority regardless of which group holds it.

```go
// csprojManifest is the decode shape for a .csproj file. Tags carry no
// namespace: XMLName local-name matching makes the same struct work for
// SDK-style <Project Sdk="..."> and old-style <Project xmlns="...msbuild/2003">
// (D-01, verified). Unknown elements (TargetFramework, ItemGroup, ...) are
// discarded by encoding/xml; multiple PropertyGroups collect in order.
type csprojManifest struct {
	XMLName        xml.Name        `xml:"Project"`
	PropertyGroups []propertyGroup `xml:"PropertyGroup"`
}

type propertyGroup struct {
	AssemblyName  string `xml:"AssemblyName"`
	RootNamespace string `xml:"RootNamespace"`
	Version       string `xml:"Version"`
	VersionPrefix string `xml:"VersionPrefix"`
	Description   string `xml:"Description"`
}

// detectCSharp reports a C#/.NET project when folder holds a readable
// .csproj (D-09: presence-match — malformed XML still matches, fields
// degrade to empty). Name: AssemblyName → RootNamespace → folder base
// (D-03, DATA-02); Version: Version → VersionPrefix → empty, never
// fabricated (D-03, DATA-03 — MSBuild would default VersionPrefix to 1.0.0,
// we report ""); Description: manifest → README first paragraph → empty
// (DATA-04). Root-scoped only: discovery is a single-level ReadDir (D-04).
func detectCSharp(folder string) (ProjectData, bool) {
	content, ok := readFirstManifest(folder, ".csproj")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	var proj csprojManifest
	_ = xml.Unmarshal(content, &proj) // decode error → zero struct → presence-match degrade

	var assemblyName, rootNamespace, version, versionPrefix, description string
	for _, pg := range proj.PropertyGroups { // document order; first non-empty per field
		if assemblyName == "" {
			assemblyName = pg.AssemblyName
		}
		if rootNamespace == "" {
			rootNamespace = pg.RootNamespace
		}
		if version == "" {
			version = pg.Version
		}
		if versionPrefix == "" {
			versionPrefix = pg.VersionPrefix
		}
		if description == "" {
			description = pg.Description
		}
	}

	data := ProjectData{Language: LanguageCSharp} // "C#/.NET" [VERIFIED: project_probe/project.go:39]
	data.Name = assemblyName
	if data.Name == "" {
		data.Name = rootNamespace
	}
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02
	}
	data.Version = version
	if data.Version == "" {
		data.Version = versionPrefix // D-03 chain; "" when both absent (never 1.0.0 fabrication)
	}
	data.Description = description
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04
	}
	return data, true
}
```

Verified ecosystem facts this shape relies on:
- Root element is ALWAYS `Project` — schema: "The root element of an MSBuild project file is the Project element", attributes include `Sdk` (SDK-style), `ToolsVersion` and `xmlns` (old-style) [CITED: learn.microsoft.com MSBuild project file schema reference].
- Properties live in `<PropertyGroup>` elements; multiple groups with `Condition` attributes are common (Debug/Release) — the collect-across-groups loop is mandatory, not optional [CITED: learn.microsoft.com MSBuild schema + dansiegel.net].
- `Version` is the property "most commonly set... It controls the default values of all the version numbers embedded in the build output"; default = `VersionPrefix[-VersionSuffix]`; "setting Version explicitly will override any VersionPrefix or VersionSuffix settings"; `VersionPrefix` defaults to `1.0.0` when unset — hence D-03's chain and the never-fabricate `""` [CITED: andrewlock.net + gist MSBuild Version Properties Cheatsheet].
- Visual Studio writes .csproj files as UTF-8 **with BOM** — SC1's "with BOM" is a real-world shape, satisfied by readManifest strip (D-02) [ASSUMED for the VS-writing-BOM claim — behavior pinned regardless by the BOM test row].

**When to use:** DETC-07. The `Description` read is the **A1 interpretation** (see Assumptions Log): D-03's arrow "Description → readmeDescription (DATA-04)" is read as the full DATA-04 chain (manifest → README → empty), matching every other manifest-bearing detector; the literal readme-only reading (Go-detector style) is the alternative.

### Pattern 4: `detectJavaKotlin` — pom.xml primary, settings.gradle fallback (DETC-08, D-05/D-06/D-07/D-09)

**What:** pom.xml via the `readManifest` presence gate with INLINE decode-error-ignored `xml.Unmarshal` (same shape as detectCSharp, Pattern 3 — no `readXMLManifest` helper); `<parent><version>` read from the CHILD'S OWN file (the research question is settled: Maven requires the parent block — including version — in every child POM; the child's own `<version>` is what may be omitted and then inherits [CITED: maven.apache.org/pom.html + introduction-to-the-pom.html]). settings.gradle is consulted ONLY when pom.xml is absent (D-06) — a malformed pom.xml still matches on presence (D-09), so the gradle fallback does NOT fire for a present-but-garbage pom.

```go
// pomManifest is the decode shape for a pom.xml. The root is the lowercase
// <project> element (XML is case-sensitive — this struct never doubles for
// .csproj); the Maven 4.0.0 namespace is tolerated via local-name matching
// (D-01). <parent><version> lives in THIS file — Maven requires the parent
// block's version in every child POM; the child's own <version> may be
// omitted and inherits it (single level per D-05 — no transitive chain).
type pomManifest struct {
	XMLName     xml.Name   `xml:"project"`
	Name        string     `xml:"name"`
	ArtifactID  string     `xml:"artifactId"`
	Version     string     `xml:"version"`
	Description string     `xml:"description"`
	Parent      *parentPOM `xml:"parent"` // allocated only when <parent> present
}

type parentPOM struct {
	Version string `xml:"version"`
}

// parseRootProjectName extracts the rootProject.name literal from
// settings.gradle content (D-06). The assignment must be an exact
// left-of-= match ("rootProject.name" or the documented fully-qualified
// "settings.rootProject.name") — a commented line like
// "// rootProject.name = 'x'" fails the exact match and can never
// fabricate a name. The value is the first quoted literal (' or " — both
// are legal Groovy; double quotes interpolate, which we never evaluate);
// a non-empty remainder after the closing quote (e.g. 'a' + 'b') is not
// stored — never partial data (Phase 12 D-disc-6 precedent). First match
// wins (parseGoMod precedent).
func parseRootProjectName(content []byte) string {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		if key != "rootProject.name" && key != "settings.rootProject.name" {
			continue
		}
		rest := strings.TrimSpace(line[i+1:])
		if len(rest) < 2 || (rest[0] != '\'' && rest[0] != '"') {
			continue
		}
		if end := strings.IndexByte(rest[1:], rest[0]); end >= 0 {
			return rest[1 : 1+end]
		}
	}
	return ""
}

// detectJavaKotlin reports a Java project when folder contains a pom.xml
// (primary) or — ONLY when no pom.xml exists — a settings.gradle with a
// rootProject.name (D-06). Language is LanguageJava for both (Kotlin-distinct
// value is v2 REFN-02). Name: <name> → <artifactId> → folder base (D-05,
// DATA-02); Version: <version> → <parent><version> when absent (D-05,
// single level); Description: <description> → README → empty (DATA-04).
// Root-scoped only (D-07); no sibling or parent POM files are ever read.
func detectJavaKotlin(folder string) (ProjectData, bool) {
	data := ProjectData{Language: LanguageJava} // "Java" [VERIFIED: project_probe/project.go:45]

	if content, ok := readManifest(folder, "pom.xml"); ok {
		var pom pomManifest
		_ = xml.Unmarshal(content, &pom) // decode error → zero struct → presence-match degrade (D-09)

		data.Name = pom.Name // <name> — Maven's optional display name
		if data.Name == "" {
			data.Name = pom.ArtifactID // <artifactId> — the required coordinate
		}
		if data.Name == "" {
			data.Name = filepath.Base(folder) // DATA-02
		}
		data.Version = pom.Version
		if data.Version == "" && pom.Parent != nil {
			data.Version = pom.Parent.Version // D-05: single-level parent inheritance
		}
		data.Description = pom.Description
		if data.Description == "" {
			data.Description = readmeDescription(folder) // DATA-04
		}
		return data, true
	}

	if content, ok := readManifest(folder, "settings.gradle"); ok { // D-06: ONLY when no pom.xml
		data.Name = parseRootProjectName(content)
		if data.Name == "" {
			data.Name = filepath.Base(folder) // DATA-02
		}
		data.Description = readmeDescription(folder) // DATA-04 (settings.gradle has no description)
		return data, true
	}

	return ProjectData{}, false
}
```

Verified ecosystem facts:
- Maven inheritance: "groupId:artifactId:version are all required fields (although, groupId and version do not need to be explicitly defined if they are inherited from a parent)" and "if you want the groupId or the version of your modules to be the same as their parents, you can remove the groupId or the version identity of your module in its POM" [CITED: maven.apache.org/pom.html + introduction-to-the-pom.html]. The parent block itself — with its version — is REQUIRED in the child ("You always have to specify parent's version") [CITED: stackoverflow.com/questions/10582054].
- `<name>`: "The full name of the project" — optional; the POM model defaults the display name to the artifactId — D-05's chain mirrors Maven's own default [CITED: maven.apache.org/pom.html].
- Multi-module layouts: the parent declares `<packaging>pom</packaging>` + `<modules><module>dir</module></modules>`; each module directory holds its own pom.xml [CITED: maven.apache.org/guides/introduction/introduction-to-the-pom.html]. Root-scoped probing reads ONLY the probed folder's own pom.xml (SC4) — a child module's `<modules>` list is never consulted.
- settings.gradle: "rootProject.name = 'root-project'" (Groovy single quotes conventional; double quotes valid — GStrings interpolate, which we never evaluate); fully-qualified "settings.rootProject.name = ..." also documented; the settings file lives in the project ROOT [CITED: docs.gradle.org/current/userguide/settings_file_basics.html + writing_settings_files.html]. The `.kts` variant (Kotlin DSL, double quotes only) is NOT read — D-06 names `settings.gradle` (A5).

**When to use:** DETC-08. Note the deliberate asymmetry: pom.xml present-but-garbage → Java with folder-base Name (presence-match, D-09); settings.gradle is skipped because pom.xml EXISTS. Only pom.xml absent → gradle fallback.

### Pattern 5: Registry completion + test/doc refresh (D-08/D-10)

**What:** Surgical edits to three existing files, final 7/7 state.

```go
// registry.go — final state (D-08): fill indices 2 and 5 in place
var detectors = []detectorFunc{
	detectGo,         // index 0 — Go (Phase 11)
	detectPython,     // index 1 — Python (Phase 12)
	detectCSharp,     // index 2 — C#/.NET (Phase 13)
	detectJS,         // index 3 — JavaScript/TypeScript (Phase 11)
	detectRust,       // index 4 — Rust (Phase 12)
	detectJavaKotlin, // index 5 — Java/Kotlin (Phase 13)
	detectPHP,        // index 6 — PHP (Phase 11)
}
```

`TestDetectorPositions` [VERIFIED: project_probe/registry_test.go:70-79] flips ONE assertion per plan (interim after P01: `detectors[2]` Nil→NotNil, `detectors[5]` stays Nil; final after P02: `detectors[5]` Nil→NotNil — Phase 12 Pitfall 9: flipping both early breaks P02's RED step). `doc.go` detector paragraph [VERIFIED: project_probe/doc.go:35-49] refresh: replace "currently holds live detectors at positions 0 (Go), 1 (Python), 3 (JS/TS), 4 (Rust), and 6 (PHP); slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13" with the all-seven state plus one sentence per new detector (csproj presence → LanguageCSharp with AssemblyName→RootNamespace→folder-base Name, Version→VersionPrefix→empty; pom.xml presence → LanguageJava with name/artifactId and single-level parent-version inheritance; settings.gradle rootProject.name fallback only when no pom.xml). Cascade rows appended to the `tests` table in `TestProbe_CascadePrecedence` [VERIFIED: project_probe/detect_php_test.go:172-292] (final plan only, mirroring Phase 12's P03):

1. `.csproj + package.json` → `LanguageCSharp` (index 2 beats 3)
2. `.csproj + pom.xml` → `LanguageCSharp` (index 2 beats 5)
3. `pom.xml + composer.json` → `LanguageJava` (index 5 beats 6)
4. `settings.gradle + composer.json` → `LanguageJava` (index 5 beats 6 via the fallback — pins that the gradle fallback participates in cascade order)
5. `settings.gradle only` → `LanguageJava` (D-06 fallback end-to-end)

**When to use:** D-08 wiring. No `runDetectors` change needed — nil-skip and `callDetector` recover are already in place [VERIFIED: project_probe/registry.go:28-53].

### Discretion Decisions Taken (CONTEXT "the agent's Discretion" — locked recommendations)

| # | Decision | Rationale |
|---|----------|-----------|
| D-disc-1 | **`xml.Unmarshal` whole-content inline** (decode-error-ignored in each detector) over `xml.Decoder` streaming | readManifest caps content at 1 MB; Unmarshal is the simple contract; Decoder adds a token loop for zero benefit |
| D-disc-2 | **One decode struct per manifest, tags without namespace** | Pattern 1 verified: plain tags match any namespace for root AND children (probe rows B/I) — no per-namespace structs, no `,any` |
| D-disc-3 | **settings.gradle: exact left-of-= match** (`rootProject.name` / `settings.rootProject.name`), first quoted literal, remainder-after-close-quote must be empty or a `//` comment | `// rootProject.name = 'x'` can never fabricate; `'a' + 'b'` concatenation expressions degrade (never partial — Phase 12 D-disc-6 precedent) |
| D-disc-4 | **First match wins** in `parseRootProjectName` | parseGoMod precedent [VERIFIED: detect_go.go:43-68]; double assignments are pathological (Gradle would last-wins — one test row if ever needed) |
| D-disc-5 | **`.csproj` discovery: first READABLE candidate in sorted ReadDir order**; exact-case `.csproj` suffix | os.ReadDir sorts (deterministic); Phase 11 D-08 exact-case precedent; FIFO candidate → readManifest gate rejects → next candidate (A9); REFN-04 defers the real multi-csproj rule |
| D-disc-6 | **Collect-first-then-chain across PropertyGroups** (first non-empty per field, then the D-03 chain) | Multiple groups are the norm (probe J); "AssemblyName → RootNamespace" priority must not depend on group order |
| D-disc-7 | **Inline fixtures** — `t.TempDir()` + `os.WriteFile` with Go raw-string XML literals | Repo convention (Phase 12 D-disc-7); XML never contains backticks so raw strings are safe; BOM rows via `"\xEF\xBB\xBF"` prefix (Phase 11 BOM row pattern) |
| D-disc-8 | **No `xml_test.go` AND no `xml.go`** — the decode is a 3-line inline in each detector; a standalone helper file would be uncalled (0% file coverage → file:70 gate failure) and a helper test file would test nothing of its own; all behavior tests live at detector level | The Phase 12 `toml_test.go` existed because the reader was 349 lines of real logic; here the interesting cases are probe-level (BOM/xmlns/multi-group/cascade) |

### Anti-Patterns to Avoid

- **Namespace-URI tags** (`xml:"http://schemas.microsoft.com/developer/msbuild/2003 Project"`): SDK-style projects (no xmlns) would fail the Space check — D-01 locks the plain `xml:"Project"` form (probe A vs the doc rule).
- **One shared XML struct for csproj and pom**: `<Project>` vs `<project>` — XML matching is case-sensitive; two structs with their own XMLName tags are mandatory.
- **`readManifest(folder, "*.csproj")` or `.csproj` literal**: the filename is project-specific; only discovery via ReadDir works. A glob string fails at `os.OpenFile`.
- **Resolving MSBuild `$(...)` / Maven `${...}` placeholders**: never evaluate, never strip, never resolve from other files — report the raw string (DATA-03 raw-manifest semantics; Phase 12's "no code branch" lesson; A2). Maven CI-friendly `${revision}` versions are common — raw output is truthful.
- **Reading a sibling/relativePath parent POM**: `<parent><version>` is in the CHILD's own file (verified); relativePath resolution would violate root-scoped ROBT-05 and D-05's single-level lock.
- **settings.gradle prefix matching**: `strings.HasPrefix(line, "rootProject.name")` lets `// rootProject.name = 'x'` and `foo // rootProject.name = 'x'` fabricate names — exact left-of-= equality only (D-disc-3).
- **settings.gradle evaluation**: `'a' + 'b'` or GString interpolation is execution — extract the literal; a non-empty remainder after the close quote is not stored (D-disc-3, Phase 12 D-disc-6).
- **`t.Parallel()` in registry-mutating tests**: AGENTS.md global-state rule; the interim/final TestDetectorPositions flips stay serial [VERIFIED: registry_test.go:12-14]. Detector fixture tests (TempDir-only) MAY parallelize.
- **Recursive/subdir probing**: no `WalkDir`, no descending into `sub/` looking for .csproj/pom.xml — root-scoped only (SC4, D-04/D-07, ROBT-05); pin with test rows.
- **Fabricating versions**: MSBuild's implicit `VersionPrefix = 1.0.0` default and Maven's Super POM `version = 4.0.0` default are both never reported — absent → `""` (D-03/DATA-03).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| XML parsing / namespace matching | A hand-rolled tokenizer or `regexp`-based extraction | `encoding/xml` with XMLName local-name tags (D-01) | Namespaces, CDATA, entities, comments, case rules, error reporting are all spec-hard; verified behavior on the actual toolchain this session |
| BOM removal | A leading-byte state machine per detector | `readManifest`'s `bytes.TrimPrefix(content, utf8BOM)` [VERIFIED: project_probe/manifest.go:15,50] | Phase 10 built it; SC1's "with BOM" is satisfied at the read boundary (D-02) — and xml.Unmarshal tolerates BOM anyway (verified) |
| File-size caps / FIFO hang protection | Per-detector read loops / plain `os.Open` | `readManifest` (1 MB LimitReader + O_NONBLOCK + `Mode().IsRegular()` gate) [VERIFIED: project_probe/manifest.go:29-43] | The WR-01 FIFO DoS is already closed; `readFirstManifest` routes every .csproj candidate through it |
| README description fallback | Re-extracting first paragraphs | `readmeDescription(folder)` [VERIFIED: project_probe/readme.go:29-48] | Phase 11 built it; both detectors call it when description is empty |
| Folder-name fallback | Hand-rolled last-segment extraction | `filepath.Base` | Stdlib; Windows-safe (ROBT-04) |
| JSON-helper shape | A bespoke `readXMLManifest` wrapper (uncalled → 0% file coverage) | Inline `_ = xml.Unmarshal(content, &v)` decode-error-ignored in each detector | The `readJSONManifest` analog [VERIFIED: project_probe/json.go:11-17] informed the shape, but a shared helper has no production caller and fails the file:70 coverage gate — the inline decode is 3 lines |
| Panic containment | Trusting the parser not to panic | `callDetector` recover → non-match [VERIFIED: project_probe/registry.go:44-53] | encoding/xml returns errors, never panics, on malformed input; the net stays behind it regardless |
| Test assertions | Hand-rolled compare helpers | `testify` `assert`/`require` | Repo mandate (AGENTS.md); `testifylint` enforced |

**Key insight:** everything "hard" in this phase — BOM, caps, FIFO DoS, README heuristics, panic safety, cross-platform paths — was solved in Phases 10-12. The genuinely new code is one ~15-line discovery helper, two ~10-line XML structs, two ~35-line detectors (each with a 3-line inline decode), and a ~20-line settings.gradle line parser. Every risk lives in (a) the XMLName local-name matching contract — now verified live — and (b) the .csproj discovery mechanism, the first time a detector reads a variable-named manifest.

## Common Pitfalls

### Pitfall 1: Namespace-URI tags break SDK-style .csproj
**What goes wrong:** `xml:"http://schemas.microsoft.com/developer/msbuild/2003 Project"` on XMLName rejects `<Project Sdk="Microsoft.NET.Sdk">` (no xmlns) with `UnmarshalError` — SC1 fails on the most common modern shape.
**Why it happens:** The `"namespace-URL name"` tag form requires the element's Space to equal the URI [CITED: pkg.go.dev/encoding/xml]; SDK-style files carry no xmlns.
**How to avoid:** Plain `xml:"Project"` (D-01); verified live — probe rows A (no xmlns) and B (xmlns) both match. Pin with two test rows: SDK-style and old-style.
**Warning signs:** A probe of a namespaced old-style csproj reports Unknown, or a code-review diff shows a URI in an xml tag.

### Pitfall 2: `.csproj` treated as a fixed filename
**What goes wrong:** `readManifest(folder, ".csproj")` or `readManifest(folder, "*.csproj")` never opens — `.csproj` is `MyApp.csproj`, a project-specific name.
**Why it happens:** Every prior detector reads a fixed manifest name (go.mod, package.json, ...); the C# manifest is the first variable-named one.
**How to avoid:** `readFirstManifest` (Pattern 2): `os.ReadDir` + exact-case `.csproj` suffix + first readable candidate through the readManifest gate. Pin with a fixture named `MyApp.csproj` (not `.csproj`).
**Warning signs:** The C# detector never matches in tests until the fixture file is named after the project.

### Pitfall 3: Version read from ONE PropertyGroup only
**What goes wrong:** `<Version>` sits in a `Condition`-gated second group (Debug/Release pattern) — a struct with a single `PropertyGroup` field misses it and reports `""`.
**Why it happens:** MSBuild files legitimately carry several PropertyGroups [CITED: learn.microsoft.com schema]; probe row J collected three.
**How to avoid:** `[]propertyGroup` slice + collect-first-then-chain (Pattern 3). Pin with a three-group fixture where Version lives in the LAST group.
**Warning signs:** A real-world csproj yields empty Version despite a visible `<Version>` element.

### Pitfall 4: `readManifest` on a UTF-16 .csproj
**What goes wrong:** A UTF-16-encoded file errors ("input is assumed to be encoded in UTF-8") → presence-match still fires with empty fields.
**Why it happens:** encoding/xml only handles UTF-8 (and declared charsets via `CharsetReader`, which Unmarshal does not use) [CITED: pkg.go.dev/encoding/xml Decoder docs].
**How to avoid:** Accept the degrade (never fabricate); Visual Studio writes UTF-8+BOM, so UTF-16 is rare [ASSUMED]. Optionally document the limitation in the detector comment. Pin with a UTF-8 BOM row (SC1) — that is the supported shape.
**Warning signs:** A project whose csproj was saved as UTF-16 reports Language=C#/.NET with folder-base Name.

### Pitfall 5: Maven parent version resolved from a sibling file
**What goes wrong:** The detector reads `../pom.xml` (the `relativePath` default) to find the parent's version — a root-scoped violation and wrong source.
**Why it happens:** Maven's own resolution does consult the relativePath; but the child's `<parent><version>` IS present in the child file and IS the value to report [CITED: maven.apache.org/pom.html + SO 10582054].
**How to avoid:** Read `pom.Parent.Version` from the probed file only (D-05 single level). Pin with the SC2 child-module fixture: child pom with `<parent><version>2.0.0</version></parent>` and no own `<version>` → Version "2.0.0".
**Warning signs:** A plan task mentions reading `../pom.xml` or a `relativePath`.

### Pitfall 6: settings.gradle comment/expression fabrication
**What goes wrong:** `// rootProject.name = 'evil'` (commented out) or `rootProject.name = 'a' + 'b'` yields a fabricated name.
**Why it happens:** Prefix matching on the line start, or extracting the first quoted literal without checking the remainder.
**How to avoid:** Exact left-of-= equality (D-disc-3); remainder after the close quote must be empty/whitespace/`//`-comment. Pin rows: commented line → folder-base Name; `'a' + 'b'` → folder-base Name; `'x' // comment` → "x".
**Warning signs:** A probe reports a name that appears only inside a comment.

### Pitfall 7: Case-sensitivity mistakes between the two XML roots
**What goes wrong:** Reusing one struct for `.csproj` and pom.xml, or tagging pom as `xml:"Project"` — Maven's root is lowercase `<project>`; matching is case-sensitive [CITED: pkg.go.dev/encoding/xml].
**How to avoid:** Two structs: `xml:"Project"` (csproj) and `xml:"project"` (pom). Probe rows A/F pin both.
**Warning signs:** pom.xml never matches; csproj with a lowercase root (invalid anyway) never matches.

### Pitfall 8: settings.gradle.kts silently expected
**What goes wrong:** A Kotlin-DSL-only project (`settings.gradle.kts`, no pom.xml) returns Unknown — the user expected Java/Kotlin.
**Why it happens:** D-06 names `settings.gradle` only; `.kts` support is not locked anywhere (REFN-02 touches build.gradle.kts for the kotlin VALUE, not the settings fallback).
**How to avoid:** Honor D-06 literally (A5 flagged for user confirmation); if confirmed as a gap, it is one extra `readManifest(folder, "settings.gradle.kts")` call plus a double-quote-only parse — but that is scope creep unless the user opts in.
**Warning signs:** A plan task adds settings.gradle.kts without a discuss-phase confirmation.

### Pitfall 9: Registry position assertions drift mid-phase
**What goes wrong:** `TestDetectorPositions` [VERIFIED: registry_test.go:70-79] asserts `detectors[2] == nil` and `detectors[5] == nil`; filling slot 2 (P01) without updating the test fails CI; flipping BOTH in P01 breaks P02's RED step.
**Why it happens:** The test pins the Phase 12 interim state; the flips belong to their plans (Phase 12 Pitfall 9 precedent).
**How to avoid:** Interim in P01 (`detectors[2]` NotNil, `[5]` stays Nil); final in P02 (both NotNil). Each plan's RED test is the flip itself.
**Warning signs:** A plan verification says "all tests pass" while the registry is mid-fill.

### Pitfall 10: `make coverage-quick` red before the first commit (pre-existing)
**What goes wrong:** The mandatory pre-commit gate fails on `release/update.go 68.9%` vs 70% — re-verified live this session.
**Why it happens:** Pre-existing, unrelated to project_probe (deferred-items); CI uses the same thresholds.
**How to avoid:** Run `make coverage-quick`; assert the **project_probe rows pass** (pkg ≥80% — new files `detect_csharp.go`/`detect_java_kotlin.go` must each clear the 70% file threshold; there is NO `xml.go` — the decode is inline in both detectors, and a dropped helper would sit at 0% file coverage) and total ≥75%; treat the `release/update.go` row as the documented known-red. Do NOT fix release/update.go (scope boundary).
**Warning signs:** A verification step fails with no remediation path — encode the known-red exception explicitly (Phase 12 Pitfall 7 pattern).

### Pitfall 11: Lint is not a gate (unchanged from Phase 12)
**What goes wrong:** `golangci-lint run` typecheck-panics on the go1.27.0 stdlib under the default toolchain, and flags Phase 11/12 files under `GOTOOLCHAIN=go1.26.4` — either way the "gate" fails and blocks commits.
**Why it happens:** golangci-lint v2.12.2 is built with an older Go; CI (`go.yml`) runs no lint job; pre-commit hooks are not installed.
**How to avoid:** Write lint-clean code per CONVENTIONS (cyclop ≤10, wsl_v5, testifylint, 120-col) but never treat lint output as a commit blocker. Verification gates are `go test ./...` + `make coverage-quick` project_probe rows.
**Warning signs:** A plan demands a green `golangci-lint run` — unachievable in this repo today.

## Code Examples

Verified patterns from official sources plus this session's probes:

### encoding/xml namespace behavior — the decision table (Pattern 1 mechanics)
```
Struct tag                              Input root                          Result (Go 1.27.0, live probe)
XMLName xml.Name `xml:"Project"`        <Project Sdk="...">                 match — Space "", Local "Project"
XMLName xml.Name `xml:"Project"`        <Project xmlns="...msbuild/2003">   match — Space = URI, Local "Project"
XMLName xml.Name `xml:"Project"`        <Projects>                          UnmarshalError "expected element type <Project> but have <Projects>"
XMLName xml.Name (no tag)               <Whatever xmlns="urn:x">            match — any root recorded
field string `xml:"AssemblyName"`       <AssemblyName xmlns="urn:other">    match — child tags w/o ns are namespace-agnostic
field string `xml:"name"`               <NAME>                              NO match — case-sensitive
[]propertyGroup `xml:"PropertyGroup"`   three <PropertyGroup> blocks        all three collected, document order
string `xml:"Version"`                  <Version><![CDATA[1.0]]></Version>  "1.0" — CDATA handled
```

### Fixture pattern — inline raw-string XML (detector tests)
```go
func TestProbe_CSharpEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "MyApp.csproj"),
		[]byte(`<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <AssemblyName>MyApp</AssemblyName>
    <Version>1.2.3</Version>
    <Description>A CLI for acme.</Description>
  </PropertyGroup>
</Project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageCSharp, data.Language)
	assert.Equal(t, "MyApp", data.Name)
	assert.Equal(t, "1.2.3", data.Version)
}
```
BOM row: prefix the same content with `"\xEF\xBB\xBF"` (SC1, mirrors Phase 11 BOM rows). Old-style row: `<Project ToolsVersion="15.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">` — same expectations (D-01). Malformed row: `<Project><PropertyGroup><AssemblyName>X` — still `LanguageCSharp`, folder-base Name (D-09 presence-match).

### SC2 child-module fixture (pom + parent inheritance)
```go
// child folder fixture — <parent><version> lives HERE (verified)
[]byte(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <parent>
    <groupId>com.acme</groupId>
    <artifactId>acme-parent</artifactId>
    <version>2.0.0</version>
  </parent>
  <artifactId>acme-child</artifactId>
  <name>Acme Child</name>
</project>
`)
// → LanguageJava, Name "Acme Child", Version "2.0.0" (parent inheritance, single level)
```

### SC3 settings.gradle fixture + parse rows
```go
// fixture
[]byte("rootProject.name = 'acme-tool'\ninclude 'sub-a'\n")
// → LanguageJava, Name "acme-tool"
// parse rows (pure content — parseRootProjectName):
// "rootProject.name = 'acme-tool'\n"                     → "acme-tool"
// "rootProject.name = \"acme-tool\"\n"                   → "acme-tool"  (double quotes legal)
// "settings.rootProject.name = 'acme-tool'\n"            → "acme-tool"  (fully-qualified form)
// "// rootProject.name = 'evil'\n"                       → ""           (comment — exact-match guard)
// "rootProject.name = 'a' + 'b'\n"                       → ""           (remainder — never partial)
// "rootProject.name = 'x' // trailing comment\n"         → "x"          (comment after close quote)
// "rootProject.name = noquotes\n"                        → ""           (unquoted — degrade)
// "include 'sub-a'\n"                                    → ""           (unrelated line)
```

### Registry position test — final state (after P02)
```go
func TestDetectorPositions(t *testing.T) { //nolint:paralleltest // reads global detectors
	assert.Len(t, detectors, 7)
	assert.NotNil(t, detectors[0]) // Go
	assert.NotNil(t, detectors[1]) // Python
	assert.NotNil(t, detectors[2]) // C#/.NET — Phase 13 (was Nil)
	assert.NotNil(t, detectors[3]) // JS/TS
	assert.NotNil(t, detectors[4]) // Rust
	assert.NotNil(t, detectors[5]) // Java/Kotlin — Phase 13 (was Nil)
	assert.NotNil(t, detectors[6]) // PHP
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Old-style .csproj (`<Project ToolsVersion xmlns="...msbuild/2003">`) | SDK-style `<Project Sdk="Microsoft.NET.Sdk">` — no xmlns, minimal PropertyGroups | 2017 (.NET Core SDK) | Namespace-agnostic matching is NOT optional: SDK-style files have no xmlns, old-style files do — the same struct must decode both (probe A/B); Version/VersionPrefix semantics unchanged [CITED: learn.microsoft.com .NET SDK overview] |
| Maven multi-module duplicate versions | CI-friendly `${revision}` placeholders (single version in parent properties) | Maven 3.5.0 (2017) | Child POMs carry `<version>${revision}</version>` or omit version entirely — raw reporting of placeholders (A2) and parent-version inheritance (D-05) are the two modern shapes |
| Gradle Groovy settings.gradle (single quotes) | Kotlin DSL settings.gradle.kts (double quotes) | Gradle 5.0 (2018) | D-06 locks settings.gradle; .kts-only projects fall through to Unknown (A5 — flagged for user confirmation; REFN-02 defers Kotlin distinction) |
| Detectors 5/7 live (Phase 12) | 7/7 live (this phase) | Phase 13 | Registry literal complete; C#@2 beats JS@3, Java@5 beats PHP@6 in mixed cascades; TestDetectorPositions loses its last Nil assertions |

**Deprecated/outdated:**
- `.csproj` `xmlns="http://schemas.microsoft.com/developer/msbuild/2003"` — old-style only; still millions of legacy repos, hence namespace-agnostic (D-01).
- Maven `<version>` duplication across every child POM — replaced by `${revision}`/parent inheritance; our single-level read handles both.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | D-03's "Description → readmeDescription (DATA-04)" is the full DATA-04 chain: csproj `<Description>` (when present) → readmeDescription → empty. The literal alternative (readme-only, Go-detector style) was not confirmable — the discuss phase ran in auto mode | Pattern 3 | If the user meant readme-only, C# Description additionally includes csproj `<Description>` values — a 3-line change (drop the field read); milestone DATA-04 ("manifest → README → empty") supports the chain reading |
| A2 | MSBuild `$(...)` and Maven `${...}` placeholder versions are reported RAW, never resolved, never stripped, never degraded | Patterns 3/4 | `${revision}`/`$(MyVersion)` leak verbatim into Version; Phase 12's "no code branch for dynamic forms" lesson supports raw reporting (DATA-03 "raw manifest string"); if the user prefers degrade-to-empty, it is one small check + test rows |
| A3 | Multi-.csproj folder: first READABLE .csproj in sorted ReadDir order wins (deterministic stopgap) | Pattern 2 | Arbitrary for genuine monorepos, but deterministic and documented; REFN-04 (v2) defines the real selection rule |
| A4 | settings.gradle parse contract: exact left-of-= key, first quoted literal, remainder must be empty/whitespace/comment — never partial, never evaluated | Pattern 4 | Comment/expression fabrication is prevented; pathological concatenations degrade to folder-base Name |
| A5 | `settings.gradle.kts` is NOT read (D-06 names settings.gradle only) — Kotlin-DSL-only projects with no pom.xml → LanguageUnknown | Pattern 4 | If the user intended .kts support, a Kotlin-DSL project reports Unknown today; one extra readManifest call + double-quote parse would close it — flagged because REFN-02 (v2) mentions build.gradle.kts, not settings.gradle.kts |
| A6 | Exact-case `.csproj` suffix matching (Phase 11 D-08 exact-case precedent) | Pattern 2 | `.CsProj`-style variants → non-match; rare, and consistent with the repo's exact-case manifest matching |
| A7 | Filename suffixes `_csharp`/`_java`/`_kotlin`/`_gradle` are build-safe | Project Structure | Verified live this session via `go tool dist list` — none in GOOS or GOARCH (unlike `_js`) |
| A8 | xml.Unmarshal tolerates a leading UTF-8 BOM (verified live) — D-02's readManifest strip is belt-and-suspenders, not load-bearing | Pattern 1 | No risk either way; the BOM test row pins SC1 |
| A9 | A FIFO named `a.csproj` + valid `b.csproj` → b wins (readManifest gate rejects a, loop continues) | Pattern 2 | Benign; a FIFO-only folder → non-match (no hang — WR-01 gate) |
| A10 | Visual Studio writes .csproj as UTF-8 with BOM | Pitfall 4 | SC1's "with BOM" is real-world; behavior pinned by the BOM test row regardless; if VS wrote UTF-16, SC1 would fail — but the D-02 boundary + verified BOM tolerance cover UTF-8+BOM |

## Open Questions (RESOLVED)

1. **Where does `<parent><version>` live — child pom or sibling parent file?**
   - What we know: D-05 says "inherit `<parent><version>` from parent POM"; SC2 says "child modules inherit `<parent><version>`".
   - What's unclear: whether a sibling/relativePath parent file must be read.
   - Recommendation: **RESOLVED — the child's OWN pom.xml.** Maven requires the parent block's version in every child POM ("You always have to specify parent's version" — SO 10582054; POM Reference: groupId/version "do not need to be explicitly defined if they are inherited from a parent", but the parent DECLARATION itself carries the version) [CITED: maven.apache.org + stackoverflow]. Reading the child file only also satisfies root-scoped ROBT-05. Pin with the SC2 fixture (Pattern "SC2 child-module fixture").

2. **C# Description chain (A1)** — auto-mode discuss left D-03's "Description → readmeDescription (DATA-04)" ambiguous; recommendation: full DATA-04 chain. The planner should note this interpretation in the plan; a discuss-phase confirmation is cheap if the user cares.

3. **Placeholder versions (A2)** — raw reporting recommended; degrade-to-empty is the alternative. No action unless the user opts in.

4. **settings.gradle.kts (A5)** — out of scope per D-06 literal reading; flagged for the planner to surface at verification if a Kotlin-DSL project shows Unknown.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All code/tests | ✓ | go1.27.0 local (module declares 1.26.4 [VERIFIED: go.mod:3]; CI pins via go-version-file) | Toolchain auto-download per go.mod |
| `make` | `make coverage-quick` gate | ✓ | GNU Make 3.81 | — |
| `go-test-coverage` | Coverage gate | ✓ | installed (~/go/bin) | `make check-go-test-coverage` auto-installs |
| `golangci-lint` | (advisory only) | ⚠️ broken under default toolchain | v2.12.2 (built go1.26.3) | Not a gate — CI runs no lint job; pre-commit hooks not installed (Pitfall 11) |
| `pre-commit` | Commit hooks | ✗ not installed | — | Hooks don't run locally; CI enforces tests + coverage only |
| Coverage gate health | Every commit | ⚠️ **known-red** | `make coverage-quick` fails on `release/update.go` 68.9% vs 70% (pre-existing, re-verified this session) | Assert project_probe rows (pkg ≥80%, files ≥70% incl. the 3 new files) + total ≥75% only; document the known-red (Pitfall 10) |
| 3-OS CI matrix | Cross-platform guarantees | ✓ (GitHub Actions) | — | Local darwin; no new platform-specific code (stdlib only; `os.ReadDir` + encoding/xml are portable) |

**Missing dependencies with no fallback:** none — stdlib-only phase; lint is advisory (Pitfall 11).
**Missing dependencies with fallback:** pre-commit hooks (not installed) — CI covers tests/coverage; lint is advisory.

## Validation Architecture

> **SKIPPED** — `.planning/config.json` sets `workflow.nyquist_validation: false` explicitly; per the research contract this section is omitted. The repo's standard gates (`go test ./...`, `make coverage-quick`, `testify` assertions) apply as documented in Project Constraints.

## Security Domain

> Required — `workflow.security_enforcement: true` in .planning/config.json. ASVS level 1; block on HIGH.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no users/sessions) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (OS permissions surface via existing sentinels) |
| V5 Input Validation | yes | All manifest content is untrusted input: `readManifest` 1 MB cap + BOM strip + FIFO gate [VERIFIED: project_probe/manifest.go:13,29-43,50]; `encoding/xml` returns errors (never panics) on malformed XML — decode failure degrades to empty fields (D-09); `readFirstManifest` is single-level (no traversal); `callDetector` recover remains the outer net [VERIFIED: project_probe/registry.go:44-53] |
| V6 Cryptography | no | — |

### Known Threat Patterns for {stdlib XML decoding + manifest discovery}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| FIFO/special file named `*.csproj` hangs the probe | DoS | Already closed — `readManifest` O_NONBLOCK + `Mode().IsRegular()` gate (WR-01) applies per candidate; a rejected candidate falls through to the next, a FIFO-only folder degrades to non-match [VERIFIED: project_probe/manifest.go:29-43] |
| Pathological XML (1 MB of malformed markup, deep nesting, unknown entities) | DoS | encoding/xml is a bounded, error-returning parser (no panics on malformed input — probe row E); 1 MB cap inherited from readManifest; `callDetector` recover net |
| XXE / entity-expansion ("billion laughs") | DoS / Tampering | encoding/xml does NOT process DTDs or resolve external entities — only the five predefined entities (`lt gt amp apos quot`) exist in strict mode; no custom `Entity`/`CharsetReader` wiring in this package; content capped at 1 MB [CITED: pkg.go.dev/encoding/xml Decoder.Entity] |
| Malformed .csproj/pom.xml spoofing a different language | Spoofing | Presence-match (D-09) — the manifest file IS the ecosystem marker; a garbage csproj claims C# at index 2 before JS@3; cascade order pins the outcome (mirrors Phase 12's pyproject asymmetry) |
| Root-scope traversal via `relativePath` / subdirectory discovery | Tampering | Forbidden: `readFirstManifest` is a single-level `os.ReadDir`; pom parent version is read from the probed file only (D-05); no `EvalSymlinks`, no `WalkDir`, no `os/exec`, no `net/http` (ROBT-05 anti-features — Phase 10 prohibitions stand) |
| settings.gradle comment/expression fabrication | Tampering | Exact left-of-= key match + quoted-literal-only extraction + remainder guard (D-disc-3) — a commented-out assignment can never produce a name |

## Project Constraints (from AGENTS.md)

Directives the planner must honor (verbatim intent from `./AGENTS.md`):

- **Milestone branches:** every milestone gets its own git branch for the PR flow; work on the milestone branch, then open a PR to `main`. Branch name: `gsd/v1.7-{slug}` (current branch: `gsd/v1.7-project-probe` — verified this session).
- **Before every commit:** run `make coverage-quick` — enforces thresholds in `.testcoverage-quick.yml` (packages ≥80%, files ≥70%, total ≥75%). **Known-red exception:** the gate fails on `release/update.go` (68.9% vs 70%, pre-existing, re-verified this session) — assert project_probe rows + total instead and document the known-red (Pitfall 10). Do not commit if *project_probe* fails.
- **Before pushing a tag:** regenerate and stage the quality report (`make quality-report`, `git add quality-report.md`).
- **Cross-platform testing (CI runs Linux/macOS/Windows):** tests modifying global state (the detector registry slice) must NOT use `t.Parallel()`; the new detector tests use only `t.TempDir()` folders and MAY parallelize; no OS-specific file-mode assertions; `os.ReadDir` + encoding/xml are portable (no syscalls beyond the existing readManifest gate).
- **Code style:** standard Go idioms; focused single-purpose functions; `testify` for assertions; unit + edge-case tests; Go doc comments for all exported symbols; Conventional Commits; minimal external dependencies — prefer stdlib.
- **Spike findings:** consult `Skill("spike-findings-go")` before committing (auto-loaded during implementation; its panic-recovery precedent is the `callDetector` net this phase's XML decode sits behind — encoding/xml returns errors, never panics, so the net should never be needed).

## Sources

### Primary (HIGH confidence)
- [VERIFIED (live probe, Go 1.27.0, this session)] — 12-case encoding/xml probe (temp dir, deleted after run; repo clean): namespace-agnostic root matching with/without xmlns, root-name mismatch error text, BOM tolerance, child foreign-namespace matching, multiple PropertyGroups, CDATA/comments/PI, case sensitivity, pom parent pointer from the child file, XMLName-without-tag any-root behavior
- [VERIFIED (live, this session)] — `go tool dist list`: `csharp`/`java`/`kotlin`/`gradle` absent from GOOS and GOARCH (filename safety); `make coverage-quick` red on `release/update.go` 68.9% only; 185 project_probe tests pass; branch `gsd/v1.7-project-probe`
- [VERIFIED (in-repo): project_probe/project.go:39] — `LanguageCSharp Language = "C#/.NET"`
- [VERIFIED (in-repo): project_probe/project.go:45] — `LanguageJava Language = "Java"`
- [VERIFIED (in-repo): project_probe/registry.go:14-22] — 7-slot literal with `nil` at indices 2 and 5 (to fill); nil-skip at 28-39; `callDetector` recover at 44-53
- [VERIFIED (in-repo): project_probe/manifest.go:13,15,29-43,50] — `maxManifestSize = 1 << 20`, `utf8BOM`, O_NONBLOCK + `IsRegular()` WR-01 gate, `bytes.TrimPrefix` BOM strip
- [VERIFIED (in-repo): project_probe/json.go:11-17] — `readJSONManifest` shape (the analog considered for XML; NOT reproduced — the decode is inline in each detector: an uncalled `readXMLManifest` helper would fail the file:70 coverage gate)
- [VERIFIED (in-repo): project_probe/detect_go.go:43-68] — `parseGoMod` line-parser precedent (first-wins, TrimSpace, comment handling)
- [VERIFIED (in-repo): project_probe/readme.go:29-48] — `readmeDescription` fallback
- [VERIFIED (in-repo): project_probe/probe.go:34,39-42] — `os.ReadDir` + `hasContent` gate
- [VERIFIED (in-repo): project_probe/registry_test.go:12-14,70-79] — no-parallel convention + `TestDetectorPositions` current asserts (indices 2/5 Nil — must flip)
- [VERIFIED (in-repo): project_probe/detect_php_test.go:172-292] — `TestProbe_CascadePrecedence` table (new rows appended)
- [VERIFIED (in-repo): project_probe/doc.go:35-49] — stale detector-list paragraph to refresh
- [VERIFIED (in-repo): go.mod:3,12] — `go 1.26.4`, `github.com/stretchr/testify v1.11.1`
- [VERIFIED (in-repo): .github/workflows/go.yml] — CI runs `go test -v ./...` + coverage only; no lint job (Phase 12 verification, same session)
- [VERIFIED (in-repo): .planning/config.json] — `workflow.nyquist_validation: false` (Validation Architecture skipped), `workflow.security_enforcement: true` (Security Domain required), all search providers disabled

### Secondary (MEDIUM confidence)
- [CITED: pkg.go.dev/encoding/xml (fetched this session)] — Unmarshal rules: XMLName tag forms ("name" vs "namespace-URL name"), case-sensitive matching, unknown-data discard, zero values for missing elements, pointer allocation, UTF-8 input assumption, Decoder Entity/Strict (no DTD/entity expansion)
- [CITED: learn.microsoft.com/en-us/visualstudio/msbuild/msbuild-project-file-schema-reference] — Project root element; PropertyGroup/Property; attributes (Sdk, ToolsVersion, xmlns)
- [CITED: learn.microsoft.com/en-us/dotnet/core/project-sdk/overview] — SDK-style projects, `Sdk` attribute, implicit imports
- [CITED: learn.microsoft.com/en-us/dotnet/core/project-sdk/msbuild-props] — SDK property reference (Version family context)
- [CITED: andrewlock.net/version-vs-versionsuffix-vs-packageversion] — Version = VersionPrefix[-VersionSuffix]; explicit Version overrides; VersionPrefix default 1.0.0
- [CITED: maven.apache.org/pom.html] — POM Reference: required coordinates, parent declaration, `<name>`/`<description>` semantics, inheritance
- [CITED: maven.apache.org/guides/introduction/introduction-to-the-pom.html] — parent element with version in the child; version/groupId omission inherits; multi-module aggregation (`packaging=pom`, `<modules>`)
- [CITED: stackoverflow.com/questions/10582054] — "You always have to specify parent's version"; what may be omitted is the child's own version
- [CITED: docs.gradle.org/current/userguide/settings_file_basics.html + writing_settings_files.html] — `rootProject.name = '...'`/`"..."` forms; `settings.rootProject.name` fully-qualified form; settings file in project root
- [CITED: pkg.go.dev/os#ReadDir] — entries returned sorted by filename (deterministic first-candidate rule)

### Tertiary (LOW confidence)
- [ASSUMED] — Visual Studio writes .csproj as UTF-8 with BOM (A10; behavior pinned by the BOM test row regardless)
- [ASSUMED] — UTF-16 .csproj files are rare in the wild (Pitfall 4; degrade is safe either way)
- [ASSUMED] — MSBuild `<Description>` property appears in real csproj files often enough for the DATA-04 chain to matter (A1; harmless when absent)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib-only mandate; zero new dependencies; `encoding/xml` behavior verified live on the actual toolchain plus official docs; every consumed helper verified in-repo with line citations
- Architecture: HIGH — namespace matching grounded in a 12-case live probe + the doc rule; .csproj discovery grounded in the os.ReadDir contract + MSBuild schema; Maven parent-version location settled by primary docs; two interpretation assumptions (A1, A5) flagged honestly
- Pitfalls: HIGH — coverage known-red and lint breakage re-verified this session; filename safety verified via `go tool dist list`; every parser trap (namespaces, case, multi-group, comments, placeholders) has a probe or doc citation

**Research date:** 2026-09-29
**Valid until:** 2026-10-29 (encoding/xml semantics and the MSBuild/Maven/Gradle specs are stable; the lint/coverage environment facts are ours to re-verify)
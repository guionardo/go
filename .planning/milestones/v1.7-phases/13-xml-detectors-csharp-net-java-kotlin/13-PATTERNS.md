# Phase 13: XML Detectors — C#/.NET + Java/Kotlin - Pattern Map

**Mapped:** 2026-09-29
**Files analyzed:** 8 (2 new runtime, 2 new test files, 4 modified)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `project_probe/detect_csharp.go` (new) | detector (cascade entry, pure func) | file-I/O → discovery (`os.ReadDir` + suffix) → decode (INLINE `_ = xml.Unmarshal` decode-error-ignored — no shared helper) → transform | `project_probe/detect_rust.go` (presence-match, chains, folder-base) + `project_probe/readme.go:29-48` + `probe.go:34` (`os.ReadDir` precedent) | exact (detector shape) — discovery loop is the one new mechanism; the XML decode is inline, NOT a `readXMLManifest` helper (dropped in revision: uncalled helper → 0% file coverage → file:70 gate failure) |
| `project_probe/detect_java_kotlin.go` (new) | detector (cascade entry, pure func) | file-I/O → decode (pom.xml, INLINE `_ = xml.Unmarshal` decode-error-ignored) + line-parse fallback (settings.gradle) → transform | `project_probe/detect_php.go` (parse-success branch via helper) + `project_probe/detect_go.go:43-68` (`parseGoMod` line parser precedent) | exact |
| `project_probe/detect_csharp_test.go` (new) | test | file-I/O fixtures (inline raw-string XML) | `project_probe/detect_rust_test.go` / `detect_python_test.go` (probe-level `t.TempDir` + `os.WriteFile`, `t.Parallel`) | exact |
| `project_probe/detect_java_kotlin_test.go` (new) | test | file-I/O fixtures + pure-content parse rows | `project_probe/detect_python_test.go` (probe-level table, `TestDetectPython` rows) + `readme_test.go:127-158` (`TestFirstRealParagraph` pure `[]byte` rows) | exact |
| `project_probe/registry.go` (MOD) | config (registry literal) | request-response (cascade dispatch) | itself — `registry.go:14-22` (detectors literal; fill indices 2/5 in place) | exact |
| `project_probe/registry_test.go` (MOD) | test | — | itself — `registry_test.go:70-79` (`TestDetectorPositions` — flip 2 in P01, 5 in P02) | exact |
| `project_probe/doc.go` (MOD) | docs | — | itself — `doc.go:35-49` (detector-list paragraph; all-seven state) | exact |
| `project_probe/detect_php_test.go` (MOD) | test | file-I/O fixtures | itself — `detect_php_test.go:172-292` (`TestProbe_CascadePrecedence` table; append C#/Java rows) | exact |

**Dropped-files note (revision, D-disc-8):** `project_probe/xml.go` and `project_probe/xml_test.go` are NOT created by any plan. RESEARCH.md D-disc-8 recommends no separate `xml_test.go`, and the shared-helper design itself was dropped in pass-3 revision — `xml.go`/`readXMLManifest` do NOT exist and must not be created: an uncalled unexported helper sits at 0% file coverage, failing the `.testcoverage-quick.yml` file:70 gate with no project_probe override (plan 13-01's verify is `test ! -f project_probe/xml.go`). The XML decode is a 3-line inline `_ = xml.Unmarshal(content, &v)` decode-error-ignored inside each detector; all behavior tests live at detector level (the Phase 12 `toml_test.go` existed because that reader was 349 lines of real logic — no such file exists here).

**Tracked-source gate (#3645):** every analog above verified git-tracked (`git ls-files -- project_probe/...` non-empty for all 15 files checked). No mirrors, no submodules. All excerpts below are from tracked source.

---

## Pattern Assignments

### `project_probe/detect_csharp.go` inline XML decode + `readFirstManifest` (utility, file-I/O → decode)

**Analog:** `project_probe/json.go` — **shape-informed, NOT reproduced**: `readJSONManifest` (json.go lines 11-17) inspired an XML twin, but the shared `readXMLManifest` helper was dropped in revision (an uncalled unexported helper → 0% file coverage → fails the file:70 gate). The decode is INLINE in each detector (D-09 presence-match degrade):
```go
_ = xml.Unmarshal(content, &proj) // decode-error-ignored — zero struct → presence-match degrade (D-09)
```

**Blueprint:** RESEARCH Pattern 2 skeleton (13-RESEARCH.md:198-224) — `readFirstManifest(folder, suffix string) (content []byte, ok bool)`. The discovery loop is the one genuinely new mechanism of the phase (`.csproj` is a variable filename — `readManifest` opens a concrete path, a glob string fails at `os.OpenFile`):

```go
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

**Precedents for the discovery loop:**
- `os.ReadDir` + entry-name map: `readme.go:30-37` (readmeDescription) — the exact-case-against-entries pattern; Phase 11 D-08 exact-case precedent
- `os.ReadDir` at the Probe boundary: `probe.go:34`
- Sorted-entries contract: `os.ReadDir` sorts by filename (deterministic first-candidate)

**Behavior contract:**
- Skip directories (`e.IsDir()`) — a subdirectory named `foo.csproj` never triggers (D-04 root-scoped)
- Exact-case suffix match — `.CsProj` variants → non-match (A6)
- Every candidate routes through `readManifest` — the WR-01 FIFO gate (manifest.go:29-43) applies per candidate; a rejected candidate falls through to the next (A9)
- Ignore list NOT consulted — same visibility as every other detector; `hasContent` already gates `Probe` (probe.go:39-42)

**Error handling:** none — missing/oversized/malformed → `false` / `(nil, false)`, never an error, never a panic. BOM strip inside readManifest (D-02); `xml.Unmarshal` tolerates a leading BOM anyway (verified Go 1.27, belt-and-suspenders).

---

### `project_probe/detect_csharp.go` (detector, file-I/O → transform)

**Analog:** `project_probe/detect_rust.go` — **exact** detector shape (presence-match, folder-base fallback, DATA-04 chain)

**Presence-match + never-fail degrade + chain skeleton** (detect_rust.go lines 18-35) — copy this shape; swap `readManifest(folder, "Cargo.toml")` → `readFirstManifest(folder, ".csproj")` and `readTOMLSection` → `xml.Unmarshal` into `csprojManifest`:
```go
func detectRust(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "Cargo.toml")
	if !ok {
		return ProjectData{}, false // D-09: never-fail degrade
	}
	fields := readTOMLSection(content, "package") // D-06: single section read
	data := ProjectData{Language: LanguageRust}
	data.Name = fields["name"]
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = fields["version"] // "" for version.workspace = true (D-07 — never resolved)
	data.Description = fields["description"]
	if data.Description == "" {
		data.Description = readmeDescription(folder) // DATA-04 — also for description.workspace = true
	}
	return data, true
}
```

**C#-specific deltas (RESEARCH Pattern 3, 13-RESEARCH.md:246-317):**
- Decode structs — **two unexported types**, tags carry NO namespace (D-01 local-name matching; probe rows A/B/I):
```go
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
```
- **Collect-first-then-chain across ALL PropertyGroups** (probe row J: multiple groups are the norm — Debug/Release `Condition` groups). First non-empty per field in document order, THEN apply the D-03 chain (`AssemblyName` → `RootNamespace` → folder base; `Version` → `VersionPrefix` → `""`). AssemblyName priority must not depend on which group holds it (D-disc-6)
- `_ = xml.Unmarshal(content, &proj)` — decode error → zero struct → **presence-match degrade** (D-09: garbage .csproj still claims `LanguageCSharp` with empty fields, folder-base Name)
- Version never fabricated: MSBuild's implicit `VersionPrefix = 1.0.0` default is never reported — absent → `""` (D-03/DATA-03). MSBuild `$(...)` placeholders reported RAW, never resolved (A2)
- Description: `csproj <Description>` → `readmeDescription(folder)` → empty (A1 — full DATA-04 chain interpretation)
- Language constant (project.go:38-39): `LanguageCSharp Language = "C#/.NET"`
- Imports: `encoding/xml`, `os`, `path/filepath`, `strings` (filepath per detect_rust.go:3-5; os+strings for the discovery loop)
- Doc comments on all unexported types/funcs (repo convention; json.go/readme.go precedent)

**Error handling:** never-fail — discovery failure → `(ProjectData{}, false)`; malformed XML → empty fields → still matches on presence (D-09); panic safety inherited from `callDetector` (registry.go:44-53) — no internal recover.

---

### `project_probe/detect_java_kotlin.go` (detector, file-I/O → transform)

**Analog:** `project_probe/detect_php.go` (branch on helper result) + `project_probe/detect_go.go:43-68` (`parseGoMod` line parser for `parseRootProjectName`)

**Two-branch detector shape** (detect_php.go lines 18-38 precedent — helper-guarded branch; here the branch is pom.xml-first, settings.gradle only-when-no-pom):

**Decode structs (RESEARCH Pattern 4, 13-RESEARCH.md:331-349)** — root is the LOWERCASE `<project>` (XML is case-sensitive — this struct never doubles for .csproj):
```go
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
```

**D-05 chains (from 13-RESEARCH.md:390-425):** Name `<name>` → `<artifactId>` → `filepath.Base(folder)`; Version `<version>` → when `""` and `pom.Parent != nil` → `pom.Parent.Version` (**single level** — `<parent><version>` lives in the CHILD's own file; Maven requires it; no sibling/relativePath traversal, root-scoped D-07/ROBT-05); Description `<description>` → `readmeDescription(folder)` → empty. `_ = xml.Unmarshal(content, &pom)` — presence-match degrade (D-09: present-but-garbage pom → Java with folder-base Name, and the settings.gradle fallback does NOT fire).

**Line-parser precedent** (detect_go.go lines 43-51) — `parseRootProjectName(content []byte) string` copies the `strings.Split` + `TrimSpace` + exact-match loop skeleton:
```go
func parseGoMod(content []byte) (modulePath, goVersion string) {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		...
```
**parseRootProjectName deltas (D-disc-3/4, RESEARCH 13-RESEARCH.md:361-381):**
- Exact **left-of-=** key equality: `"rootProject.name"` or `"settings.rootProject.name"` — a commented line `// rootProject.name = 'x'` fails the exact match and can never fabricate (anti-pattern: `strings.HasPrefix`)
- Value = first quoted literal (`'` or `"` both legal Groovy; double quotes interpolate — never evaluate)
- Remainder after the closing quote must be empty/whitespace/`//`-comment; `'a' + 'b'` concatenation → not stored, never partial (Phase 12 D-disc-6 precedent)
- First match wins (parseGoMod precedent, D-disc-4)
- BOM: content already stripped by readManifest (manifest.go:50)

**settings.gradle branch (D-06):** ONLY when `readManifest(folder, "pom.xml")` returns false. `readManifest(folder, "settings.gradle")` → `parseRootProjectName(content)` → folder-base fallback; `readmeDescription(folder)` (no description field). Language = `LanguageJava` for BOTH branches (project.go:44-45 — `LanguageJava Language = "Java"`; Kotlin-distinct value is v2 REFN-02). `settings.gradle.kts` NOT read (A5, D-06 literal).

**Error handling:** never-fail — no pom.xml AND no settings.gradle → `(ProjectData{}, false)`; malformed pom → presence-match Java; malformed settings.gradle → folder-base Name (presence-match on the fallback arm too).

---

### `project_probe/detect_csharp_test.go` (test, file-I/O fixtures)

**Analog:** `project_probe/detect_rust_test.go` / `detect_python_test.go` — **exact**

**Inline-fixture pattern** (detect_python_test.go lines 20-35 + RESEARCH fixture 13-RESEARCH.md:589-615) — `t.Parallel()` safe (TempDir-only, no global state); raw-string XML literals (XML contains no backticks — D-disc-7):
```go
func TestProbe_PythonEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte("[project]\nname = \"acme\"\nversion = \"1.2.3\"\ndescription = \"A Python library.\"\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguagePython, data.Language)
	...
```

**Required rows (from RESEARCH 13-RESEARCH.md:589-615 + Pitfalls 1-4):**
- End-to-end SDK-style: `<Project Sdk="Microsoft.NET.Sdk">` with AssemblyName/Version/Description → `LanguageCSharp`, name/version/description verbatim — **fixture named `MyApp.csproj`, NOT `.csproj`** (Pitfall 2: variable filename)
- Old-style namespaced: `<Project ToolsVersion="15.0" xmlns="...msbuild/2003">` — same expectations (D-01, probe row B)
- BOM row: `"\xEF\xBB\xBF"` prefix (detect_python_test.go:239-253 BOM row precedent, Phase 11 pattern)
- Multi-PropertyGroup: Version in the LAST group (Pitfall 3 — collect-across-groups is load-bearing)
- Malformed row: `<Project><PropertyGroup><AssemblyName>X` → still `LanguageCSharp`, folder-base Name (D-09 presence)
- Root-scope row: `.csproj` in a subdirectory → non-match for parent (D-04)
- Name chain: RootNamespace-only → RootNamespace; neither → folder base (DATA-02)
- Version chain: VersionPrefix-only → VersionPrefix; neither → `""` (never `1.0.0`)
- `TestDetectCSharp_MissingManifest` — `(ProjectData{}, false)` (detect_python_test.go:165-172 precedent)
- FIFO/readability edge (optional): valid `b.csproj` beats FIFO `a.csproj` (A9)

### `project_probe/detect_java_kotlin_test.go` (test, file-I/O fixtures + pure parse rows)

**Analog:** `project_probe/detect_python_test.go` (probe-level table) + `readme_test.go:127-158` (pure-content rows) — **exact**

**Probe-level table shape** (detect_python_test.go `TestDetectPython` lines 318-395 — table rows with `folder func(t *testing.T) string` column, `wantName ""` → `filepath.Base(folder)`); **pure parse rows for `parseRootProjectName`** mirror `TestFirstRealParagraph` (readme_test.go:127-158, pure `[]byte` in → string out).

**Required rows (RESEARCH 13-RESEARCH.md:617-649 + Pitfalls 5-8):**
- Canonical pom: `<project xmlns="http://maven.apache.org/POM/4.0.0">` with name/artifactId/version/description (probe row F)
- SC2 child-module: `<parent><version>2.0.0</version></parent>` + no own `<version>` → Version `"2.0.0"` (single-level; never read `../pom.xml` — Pitfall 5)
- Name chain: no `<name>` → artifactId; neither → folder base (Maven's own default mirrors this)
- SC3 gradle fallback: `rootProject.name = 'acme-tool'` only (no pom.xml) → `LanguageJava`, Name "acme-tool"; `settings.gradle` + `pom.xml` → pom wins (D-06)
- Gradle parse rows (pure content): single quotes, double quotes, `settings.rootProject.name`, commented line → `""`, `'a' + 'b'` → `""`, `'x' // comment` → "x", unquoted → `""`, unrelated line → `""`
- Malformed pom → presence-match Java (D-09); pom present-but-garbage → gradle fallback does NOT fire (D-06 asymmetry)
- Root-scope: pom.xml in a subdirectory → non-match (D-07)
- `TestDetectJavaKotlin_MissingManifest` — `(ProjectData{}, false)`

### `project_probe/registry.go` (MOD — config, registry literal)

**Analog:** itself — `registry.go:14-22` — **exact**

**Slot-fill edit (D-08, RESEARCH Pattern 5, 13-RESEARCH.md:440-451)** — replace the two `nil` entries in place, never reorder:
```go
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
Also refresh the stale comment block at registry.go:8-13 ("empty slots stay nil until their phase fills them" → all seven live). **No `runDetectors`/`callDetector` change** — nil-skip (registry.go:28-39) and panic recovery (44-53) already in place.

---

### `project_probe/registry_test.go` (MOD — test)

**Analog:** itself — `registry_test.go:70-79` — **exact**

**TestDetectorPositions flip, ONE assertion per plan (Pitfall 9)** — interim after P01: `detectors[2]` Nil→NotNil, `detectors[5]` stays Nil; final after P02: `detectors[5]` Nil→NotNil:
```go
func TestDetectorPositions(t *testing.T) { //nolint:paralleltest // reads global detectors
	assert.Len(t, detectors, 7)
	assert.NotNil(t, detectors[0]) // Go
	assert.NotNil(t, detectors[1]) // Python — Phase 12
	assert.NotNil(t, detectors[2]) // C#/.NET — Phase 13 (was Nil)
	assert.NotNil(t, detectors[3]) // JS/TS
	assert.NotNil(t, detectors[4]) // Rust — Phase 12
	assert.NotNil(t, detectors[5]) // Java/Kotlin — Phase 13 (was Nil)
	assert.NotNil(t, detectors[6]) // PHP
}
```
Keep the file's no-`t.Parallel()` discipline (registry_test.go:12-14 — registry tests mutate the package-level `detectors` slice; AGENTS.md global-state rule).

---

### `project_probe/doc.go` (MOD — docs)

**Analog:** itself — `doc.go:35-49` — **exact**

**Detector-list paragraph refresh (D-08)** — the "Detector implementations land across phases 11-13" paragraph currently says "slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13" (doc.go:35-49). Replace with the all-seven state plus one sentence per new detector: .csproj presence → `LanguageCSharp` with `AssemblyName`→`RootNamespace`→folder-base Name, `Version`→`VersionPrefix`→empty; pom.xml presence → `LanguageJava` with `<name>`/`<artifactId>` and single-level `<parent><version>` inheritance; `settings.gradle` `rootProject.name` fallback only when no pom.xml. Keep the existing never-fail contract paragraphs (doc.go:7-17) untouched.

---

### `project_probe/detect_php_test.go` (MOD — test)

**Analog:** itself — `detect_php_test.go:172-292` (`TestProbe_CascadePrecedence`) — **exact**

**Append 5 cascade rows (P02 only, mirroring Phase 12's P03; RESEARCH 13-RESEARCH.md:453-459)** to the existing table (same struct: `name`, `setup func(t *testing.T) string`, `want`, `wantErr`):

1. `.csproj + package.json` → `LanguageCSharp` (index 2 beats 3)
2. `.csproj + pom.xml` → `LanguageCSharp` (index 2 beats 5)
3. `pom.xml + composer.json` → `LanguageJava` (index 5 beats 6)
4. `settings.gradle + composer.json` → `LanguageJava` (index 5 beats 6 **via the fallback** — pins that the gradle fallback participates in cascade order)
5. `settings.gradle only` → `LanguageJava` (D-06 fallback end-to-end)

Row shape precedent — detect_php_test.go:203-223 (`go_mod_and_package_json` row: two `os.WriteFile` calls, expected `LanguageGo`).

---

## Shared Patterns

### Never-fail presence-match contract
**Source:** `project_probe/detect_go.go:14-29`, `detect_rust.go:18-35`, `detect_python.go:15-35`
**Apply to:** `detect_csharp.go`, `detect_java_kotlin.go`
```go
content, ok := readManifest(folder, "Cargo.toml")
if !ok {
	return ProjectData{}, false // D-09: never-fail degrade
}
```
Manifest presence = match; parse failure → empty/fallback fields; read failure → `(ProjectData{}, false)`. Never an error return; panic containment inherited from `callDetector` (registry.go:44-53) — no internal recover.

### Manifest I/O safety
**Source:** `project_probe/manifest.go:20-51`
**Apply to:** all reads in both detectors (via `readManifest`/`readFirstManifest`; the XML decode itself is INLINE after the read — the `readXMLManifest` helper was dropped in revision: uncalled → 0% file coverage → file:70 gate failure)
1 MB cap (`maxManifestSize`, line 13), BOM strip (line 50), O_NONBLOCK + `Mode().IsRegular()` WR-01 FIFO gate (lines 29-43). **Never `os.ReadFile`** — `readFirstManifest` routes every .csproj candidate through this gate.

### Description fallback chain (DATA-04)
**Source:** `project_probe/readme.go:29-48` (`readmeDescription` — exact-case candidates, README.md → README.rst → README)
**Apply to:** both detectors — manifest `<Description>`/`<description>` → `readmeDescription(folder)` → empty.

### Name/Version chains (DATA-02/DATA-03)
**Source:** `project_probe/detect_rust.go:25-29` (folder-base fallback), `project.go:38-45` (Language constants)
**Apply to:** both detectors — `filepath.Base(folder)` fallback (Windows-safe, ROBT-04); versions never fabricated (MSBuild implicit `1.0.0` and Maven Super POM defaults → `""`); placeholders reported raw.

### Test parallelism discipline
**Source:** `project_probe/registry_test.go:12-14` vs `detect_python_test.go:12-14`
**Apply to:** all test files — fixture-only tests (`t.TempDir()` writes) MAY `t.Parallel()`; registry-mutating tests (TestDetectorPositions) MUST NOT (`//nolint:paralleltest`, AGENTS.md global-state rule).

### Inline raw-string fixtures
**Source:** `project_probe/detect_python_test.go:20-35`, `detect_rust_test.go` BOM rows; RESEARCH 13-RESEARCH.md:588-615
**Apply to:** `detect_csharp_test.go`, `detect_java_kotlin_test.go` — `t.TempDir()` + `os.WriteFile` + Go raw-string XML (XML contains no backticks, safe — D-disc-7); BOM rows via `"\xEF\xBB\xBF"` prefix.

### XML decode degradation
**Source:** verified behavior (RESEARCH Pattern 1, 12-case probe on Go 1.27) + `encoding/xml` docs
**Apply to:** both detectors — `xml.Unmarshal` returns errors, never panics, on malformed input; unknown elements discarded; missing elements → zero value; pointer fields allocated only when present; case-sensitive matching; XMLName tag without namespace matches by LOCAL name (namespace-agnostic).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| — | — | — | Every new file has an exact or role-match analog in `project_probe/`. The only new mechanism (`.csproj` discovery) is grounded in `os.ReadDir` precedents at `probe.go:34` and `readme.go:30-37`. |

## Metadata

**Analog search scope:** `project_probe/` (all 29 files globbed; 15 read or excerpted: json.go, manifest.go, readme.go, project.go, probe.go, registry.go, registry_test.go, doc.go, detect_go.go, detect_python.go, detect_rust.go, detect_php.go, detect_python_test.go, detect_php_test.go, toml.go; test-function inventory via grep across all test files)
**Files scanned:** 29 (project_probe/*.go)
**Pattern extraction date:** 2026-09-29
**Tracked-source verification:** `git ls-files` non-empty for all 15 named analog files (no mirrors, no submodules)
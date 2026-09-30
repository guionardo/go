# Phase 12: TOML Subset + Python/Rust Detectors - Pattern Map

**Mapped:** 2026-09-29
**Files analyzed:** 10 (3 new runtime, 1 new shared helper, 3 new test files, 3 modified)
**Analogs found:** 10 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `project_probe/toml.go` (new) | utility (unexported parse helper) | transform (text → map) | `project_probe/json.go` (never-fail helper shape) + `project_probe/detect_go.go:43-68` (parseGoMod line-based parser over readManifest output) | role-match (skip-state machine is new mechanics) |
| `project_probe/detect_python.go` (new) | detector (cascade entry, pure func) | file-I/O → text transform | `project_probe/detect_go.go` (presence-match, folder-base fallback) | exact |
| `project_probe/detect_rust.go` (new) | detector (cascade entry, pure func) | file-I/O → text transform | `project_probe/detect_go.go` | exact |
| `project_probe/toml_test.go` (new) | test (pure-content table, no files) | — | `project_probe/readme_test.go:127-158` (TestFirstRealParagraph — pure `[]byte` rows + `t.Parallel`) | exact |
| `project_probe/detect_python_test.go` (new) | test | file-I/O fixtures | `project_probe/detect_go_test.go` (probe-level `t.TempDir` + `os.WriteFile`) | exact |
| `project_probe/detect_rust_test.go` (new) | test | file-I/O fixtures | `project_probe/detect_go_test.go` | exact |
| `project_probe/registry.go` (MOD) | config (registry literal) | request-response (cascade dispatch) | itself — `registry.go:14-22` (detectors literal) | exact |
| `project_probe/registry_test.go` (MOD) | test | — | itself — `registry_test.go:70-79` (TestDetectorPositions) | exact |
| `project_probe/doc.go` (MOD) | docs | — | itself — `doc.go:35-46` (detector-list paragraph) | exact |
| `project_probe/detect_php_test.go` (MOD) | test | file-I/O fixtures | itself — `detect_php_test.go:172-256` (TestProbe_CascadePrecedence) | exact |

**Tracked-source gate (#3645):** every analog above verified git-tracked (`git ls-files project_probe/` non-empty for all 23 files). No mirrors, no submodules. All excerpts below are from tracked source.

---

## Pattern Assignments

### `project_probe/toml.go` (utility, transform: text → map)

**Analog:** `project_probe/json.go` (helper shape) + `project_probe/detect_go.go:43-68` (line-based parsing mechanics)

**Unexported never-fail helper shape** (json.go lines 11-17) — `readTOMLSection` has the same unexported status and consumes `readManifest` output identically:
```go
func readJSONManifest(folder, name string, v any) bool {
	content, ok := readManifest(folder, name)
	if !ok {
		return false
	}
	return json.Unmarshal(content, v) == nil
}
```

**Line-based parser mechanics** (detect_go.go lines 43-51) — the `strings.Split(content, "\n")` + `TrimSpace` loop is the Phase 11 precedent; the reader adds quote-aware value extraction and skip states on top:
```go
func parseGoMod(content []byte) (modulePath, goVersion string) {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		...
```

**Blueprint:** RESEARCH Pattern 1 skeleton (12-RESEARCH.md:199-243) is the shape to implement — `readTOMLSection(content []byte, section string) map[string]string` with three loop states (`skipDelim` multi-line string, `skipClose` bracket, `current != section` guard), exact header equality, key/value split at first `=`, quote-aware scanning, strict degrade (unsupported values NOT stored — D-disc-2, missing-key lookup reads `""`). Executor discretion on helper split (parseKeyValue/headerName/enterSkip); keep under ~110 lines with comments.

**Behavior contract the tests must pin (12-RESEARCH.md:492-508 edge-case matrix + Pattern 1 points 1-8):**
- Exact header equality only: `[project]` ≠ `[project.optional-dependencies]`; `[tool.poetry]` ≠ `[tool.poetry.dependencies]` (P2 — anti-pattern: prefix matching)
- Global skip states: multi-line strings/arrays/inline tables in ANY section suppress header/keyval parsing until the closing delimiter (D-disc-4, P3 — prevents fabrication)
- Quoted-value-first extraction: `description = "hello # world"` keeps `#` inside quotes (P1); remainder after closing quote must be whitespace/comment or the key is NOT stored (D-disc-6)
- Bare-key form check before `=` split: a `.` in the key → dotted key → not stored (this is how `version.workspace = true` degrades, D-07); quoted keys must be exactly the target names
- `TrimSpace` per line handles CRLF (P5); BOM already stripped by readManifest (manifest.go:50)
- Duplicate keys: last wins (A5, mirrors encoding/json)
- Never panics: bounds-checked scanning only; `callDetector` recover stays the outer net

**Error handling:** none — D-03: section missing → empty map, never an error. No panics by construction.

---

### `project_probe/detect_python.go` (detector, file-I/O → text transform)

**Analog:** `project_probe/detect_go.go` — **exact** (presence-match rule D-09 is detectGo's D-disc-1; folder-base fallback; README description)

**Presence-match + never-fail degrade + DATA-02/DATA-04 chains** (detect_go.go lines 14-29) — copy this shape; swap `go.mod`→`pyproject.toml`, `parseGoMod`→`readTOMLSection(content, "project")` with the `[tool.poetry]` fallback (D-04):
```go
func detectGo(folder string) (ProjectData, bool) {
	content, ok := readManifest(folder, "go.mod")
	if !ok {
		return ProjectData{}, false // D-12: never-fail degrade
	}

	name, version := parseGoMod(content)
	data := ProjectData{Language: LanguageGo}
	data.Name = name
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
	data.Version = version                       // raw go directive or "" (D-02 ...)
	data.Description = readmeDescription(folder) // go.mod has no description field (D-03)
	return data, true
}
```

**Python-specific deltas (from 12-RESEARCH.md Pattern 2, lines 253-281):**
- Manifest: `readManifest(folder, "pyproject.toml")` — never `os.ReadFile` (P5)
- Fields: `fields := readTOMLSection(content, "project")`; `if len(fields) == 0 { fields = readTOMLSection(content, "tool.poetry") }` — the `len==0` guard covers both "section absent" and "section present but everything unsupported" (D-disc-3); NO per-field mixing across sections
- Version is the raw string or `""` — `dynamic = ["version"]` is an array → not stored → `""` (DATA-03, never fabricated)
- Language constant (project.go lines 32-33): `LanguagePython Language = "Python"`
- Imports: `path/filepath` only (detect_go.go lines 3-6 precedent)

**Error handling:** never-fail — read failure → `(ProjectData{}, false)`; malformed TOML → strict empty fields → still matches on presence (D-09), fallbacks fill Name/Description. Panic safety inherited from `callDetector` (registry.go:44-53) — no internal recover.

---

### `project_probe/detect_rust.go` (detector, file-I/O → text transform)

**Analog:** `project_probe/detect_go.go` — **exact** (same shape as detect_python.go minus the section fallback)

**Shape:** identical to detect_python.go with `readManifest(folder, "Cargo.toml")` and a single `readTOMLSection(content, "package")` (12-RESEARCH.md Pattern 3, lines 297-322). The `[package]` arrays (`authors = [...]`) and unquoted booleans (`publish = false`) are legal-but-unsupported — the reader's per-key degrade must not disturb the section state (P4).

**Rust-specific deltas:**
- `version.workspace = true` / `description.workspace = true` are dotted keys → not stored → `""` (D-07); NEVER read `[workspace.package]`, never resolve from a workspace root (anti-pattern; ROBT-05)
- `description.workspace = true` → `""` → README fallback fires (same DATA-04 chain as any absent description)
- A virtual manifest (`[workspace]` without `[package]`) still matches on presence with folder-base Name (D-09)
- Language constant (project.go lines 41-42): `LanguageRust Language = "Rust"`
- Imports: `path/filepath` only

**Error handling:** never-fail — identical to detect_python.go.

---

### `project_probe/toml_test.go` (test, pure-content table — no files)

**Analog:** `project_probe/readme_test.go:127-158` (TestFirstRealParagraph) — **exact**: pure `[]byte` rows, `t.Parallel()` everywhere (no global state, no TempDir)

**Pure-content table mechanics to copy** (readme_test.go lines 130-157):
```go
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"badge_only", "[![logo](x)](y)\n", ""},
		...
		{"empty", "", ""},
		{"whitespace_only", "\n   \n\t\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, firstRealParagraph([]byte(tt.content)))
		})
	}
```

**Delta for the reader:** rows carry `section string` + `want map[string]string`; call `readTOMLSection([]byte(tt.content), tt.section)`. `toml_test.go` MUST be `package projectprobe` (internal) to reach the unexported `readTOMLSection` (12-RESEARCH.md:175). Row list is the RESEARCH edge-case matrix (12-RESEARCH.md:492-508) — every degrade case (multiline string containing `name = "evil"`, `authors = [` multi-line array, `version = {` inline table, `version.workspace = true`, `dynamic = ["version"]`, `publish = false`, `name = "foo" garbage`, trailing-comment headers, `[ project ]` padding, `[[tool.poetry.source]]` non-match, duplicate keys last-wins, CRLF, BOM-prefixed content, empty/comment-only/whitespace-only, 200-key section) plus a happy-path `[project]` full read and the D-03 missing-section empty-map row. This file is the Phase 12 blocker's fixture-driven validation (STATE.md).

---

### `project_probe/detect_python_test.go` / `project_probe/detect_rust_test.go` (tests, file-I/O fixtures)

**Analog:** `project_probe/detect_go_test.go` — **exact** (probe-level `t.TempDir()` + `os.WriteFile` inline fixtures, `t.Parallel()` safe per the file header at detect_go_test.go:12-14)

**Probe-level end-to-end row** (detect_go_test.go lines 21-41 — copy shape; pyproject.toml/Cargo.toml + README.md fixtures):
```go
func TestProbe_GoEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "go.mod"),
		[]byte("module example.com/acme\n\ngo 1.26.4\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "README.md"),
		[]byte("[![badge](https://example.com/x.svg)](https://example.com)\n\n# Acme\n\nA CLI for acme.\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageGo, data.Language)
	assert.Equal(t, "example.com/acme", data.Name)
	assert.Equal(t, "1.26.4", data.Version)
	assert.Equal(t, "A CLI for acme.", data.Description)
}
```

**Table-driven rows** (detect_go_test.go lines 105-256 — `TestDetectGo` shape: `folder func(t *testing.T) string` column, `""` wantName = folder-base fallback; call the detector directly, e.g. `pd, ok := detectPython(folder)`). BOM row pattern from detect_javascript_test.go:108-122 (TestProbe_JSBOM — `"\xEF\xBB\xBF"` prefix). Presence row pattern from detect_go_test.go:64-79 (TestProbe_GoPresence — garbage content still matches, folder-base Name, `""` Version).

**Python-specific rows:** PEP 621 full `[project]`; `[tool.poetry]`-only legacy fallback; both-present → `[project]` wins (D-disc-3); `[project]` with `dynamic = ["version"]` → Version `""`; `[project.optional-dependencies]` key leakage isolation (P2); `[project]`-absent + `[tool.poetry.dependencies]`-present → legacy read stays clean; malformed pyproject.toml → still `LanguagePython` (presence, D-09 — contrast with JS's parse-success non-match); missing pyproject.toml → no match.

**Rust-specific rows:** `[package]` full; `version.workspace = true` → Version `""` (SC3, D-07); `description.workspace = true` → `""` → README fallback fires; `authors = ["a","b"]` + `version = "1.2.3"` → Version intact (P4); virtual manifest (`[workspace]` only) → matches, folder-base Name; `publish = false` in `[package]` → degrades, no state disturbance; malformed Cargo.toml → still matches on presence (D-09); missing Cargo.toml → no match.

---

### `project_probe/registry.go` (MODIFIED — fill indices 1 and 4)

**Analog:** itself — `registry.go:14-22` (the 7-slot literal)

**Surgical edit** (registry.go lines 14-22 — replace the two `nil` entries in place; D-08 — no reordering, order is the contract):
```go
var detectors = []detectorFunc{
	detectGo,  // index 0 — Go (Phase 11)
	nil,       // index 1 — Python (Phase 12)   ← becomes detectPython
	nil,       // index 2 — C#/.NET (Phase 13)
	detectJS,  // index 3 — JavaScript/TypeScript (Phase 11)
	nil,       // index 4 — Rust (Phase 12)     ← becomes detectRust
	nil,       // index 5 — Java/Kotlin (Phase 13)
	detectPHP, // index 6 — PHP (Phase 11)
}
```

Final state per 12-RESEARCH.md Pattern 4 (lines 339-349): `detectPython` at 1, `detectRust` at 4, comments updated to "Phase 12". `runDetectors` nil-skip (registry.go:28-39) and `callDetector` recover (registry.go:44-53) need NO changes — nil-skip already handles empty slots; recover is already in place.

---

### `project_probe/registry_test.go` (MODIFIED — TestDetectorPositions)

**Analog:** itself — `registry_test.go:70-79`

**Interim flip (after P02 fills slot 1):** line 73 `assert.Nil(t, detectors[1])` → `assert.NotNil(t, detectors[1]) // Python — Phase 12`; line 76 stays `assert.Nil` (Rust not yet filled).
**Final flip (after P03 fills slot 4):** line 76 `assert.Nil(t, detectors[4])` → `assert.NotNil(t, detectors[4]) // Rust — Phase 12`. Final state pinned in 12-RESEARCH.md:471-482. Keep the `//nolint:paralleltest` comment and the file header note (registry_test.go:12-14 — no `t.Parallel()` in this file; global-state mutation).

---

### `project_probe/doc.go` (MODIFIED — detector-list paragraph)

**Analog:** itself — `doc.go:35-46`

**Refresh the paragraph** (doc.go lines 35-46): change "currently holds live detectors at positions 0 (Go), 3 (JS/TS), and 6 (PHP)" to name the Phase 12 additions — live at 0 (Go), 1 (Python), 3 (JS/TS), 4 (Rust), 6 (PHP); slots 2 (C#/.NET) and 5 (Java/Kotlin) fill in Phase 13. Add one sentence per new detector mirroring the existing DETC-03/DETC-04 sentences (pyproject.toml presence → LanguagePython with `[project]`/`[tool.poetry]` fields; Cargo.toml presence → LanguageRust with `[package]` fields, workspace-inherited versions degrade to empty).

---

### `project_probe/detect_php_test.go` (MODIFIED — TestProbe_CascadePrecedence rows)

**Analog:** itself — `detect_php_test.go:172-256`

**Add two rows** to the existing `tests` table (shape: `setup func(t *testing.T) string` + `want Language`; subtest asserts `Probe(folder)` Language — detect_php_test.go lines 246-255):
1. `pyproject.toml + Cargo.toml` → `LanguagePython` (index 1 beats 4 — D-08 order)
2. `broken package.json + valid pyproject.toml` → `LanguagePython` (index 1 beats 3 — Python matches on presence D-09 while JS needs parse success D-04)

Both rows exactly mirror the existing `"broken_package_json_valid_composer_json"` row mechanics (detect_php_test.go:184-202).

---

## Shared Patterns

### Never-fail degrade (manifest read failure → non-match)
**Source:** `project_probe/detect_go.go:15-18` (and `detect_javascript.go:25-27`)
**Apply to:** detect_python.go, detect_rust.go
```go
	content, ok := readManifest(folder, "go.mod")
	if !ok {
		return ProjectData{}, false // D-12: never-fail degrade
	}
```

### DATA-02 name chain (folder-base fallback)
**Source:** `project_probe/detect_go.go:23-25`
**Apply to:** both new detectors
```go
	data.Name = fields["name"]
	if data.Name == "" {
		data.Name = filepath.Base(folder) // DATA-02: name chain fallback
	}
```

### DATA-04 description chain (README first paragraph)
**Source:** `project_probe/readme.go:29-48` (`readmeDescription` — consumed unchanged; call at `detect_go.go:27`)
**Apply to:** both new detectors — `if data.Description == "" { data.Description = readmeDescription(folder) }`

### Manifest I/O gate (1 MB cap + BOM strip + WR-01 FIFO gate)
**Source:** `project_probe/manifest.go:20-51` (`readManifest` — consumed unchanged; O_NONBLOCK + `Mode().IsRegular()` at lines 29-43, BOM strip at line 50)
**Apply to:** toml.go's callers and every detector test fixture — never `os.ReadFile` in detectors (P5)

### `t.Parallel()` discipline
**Source:** `project_probe/registry_test.go:12-14` (NO parallel — mutates global `detectors`); `project_probe/detect_go_test.go:12-14` (parallel-safe — TempDir fixtures only); `project_probe/readme_test.go:127-158` (parallel-safe — pure content)
**Apply to:** toml_test.go (parallel — pure), detect_python_test.go/detect_rust_test.go (parallel — TempDir only), registry_test.go/detect_php_test.go cascade rows (serial per file-header convention)

### Panic containment (outer net, no per-detector recover)
**Source:** `project_probe/registry.go:44-53` (`callDetector`)
**Apply to:** both new detectors — the reader is designed never to panic; `callDetector`'s recover remains the only safety net.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (none) | — | — | Every file has a tracked analog. `toml.go`'s skip-state machine mechanics are new (parseGoMod is single-line, no skip states) — the planner should use 12-RESEARCH.md Pattern 1 (lines 177-247) as the behavioral blueprint on top of the parseGoMod/json.go shape. |

## Metadata

**Analog search scope:** `project_probe/` (23 tracked files), `.planning/phases/11-*/11-PATTERNS.md`, `.planning/phases/12-*/12-RESEARCH.md`
**Files scanned:** 24
**Pattern extraction date:** 2026-09-29
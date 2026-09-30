# Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback - Pattern Map

**Mapped:** 2026-09-29
**Files analyzed:** 12 (5 new runtime, 1 new shared helper, 3 modified, 4 new test files — README helper + detector tests counted as fixtures)
**Analogs found:** 11 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `project_probe/detect_go.go` (new) | detector (cascade entry, pure func) | file-I/O → text transform | `project_probe/registry.go` (detectorFunc contract) + `project_probe/manifest.go` (readManifest) | exact |
| `project_probe/detect_js.go` (new) | detector (cascade entry, pure func) | file-I/O → JSON decode | same as above + `httptest_mock/setup.go` (unmarshalMock) | exact |
| `project_probe/detect_php.go` (new) | detector (cascade entry, pure func) | file-I/O → JSON decode | mirror of `detect_js.go` (same shape, `composer.json`/`LanguagePHP`) | exact (mirror) |
| `project_probe/json.go` (new) | utility (shared helper) | file-I/O → JSON decode | `httptest_mock/setup.go:277-298` (unmarshalMock) + `project_probe/manifest.go:19-35` (readManifest wrapper) | role-match |
| `project_probe/readme.go` (new) | utility (shared helper) | file-I/O → text extraction | `project_probe/manifest.go:19-35` (readManifest candidate reads) | partial (heuristic is new) |
| `project_probe/registry.go` (MOD) | config (registry literal) | request-response (cascade dispatch) | itself — `registry.go:11` (detectors var), `registry.go:15-23` (runDetectors) | exact |
| `project_probe/doc.go` (MOD) | docs | — | itself — `doc.go:35-37` (stale "registry is empty" paragraph) | exact |
| `project_probe/registry_test.go` (MOD) | test | — | itself — `registry_test.go:12-25` (no-parallel note, save/restore, empty-registry test) | exact |
| `project_probe/detect_go_test.go` (new) | test | file-I/O fixtures | `project_probe/manifest_test.go:23-133` (table-driven inline fixtures) | exact |
| `project_probe/detect_js_test.go` (new) | test | file-I/O fixtures | `project_probe/manifest_test.go:23-133` | exact |
| `project_probe/detect_php_test.go` (new) | test | file-I/O fixtures | `project_probe/manifest_test.go:23-133` | exact |
| `project_probe/readme_test.go` (new) | test | file-I/O fixtures | `project_probe/manifest_test.go` (fixture mechanics) + `project_probe/probe_test.go:22-77` (table shape) | role-match |

**Tracked-source gate (#3645):** every analog above verified git-tracked (`git ls-files` non-empty for `project_probe/`, `cache/`, `httptest_mock/`). The deleted `project_detector/` package is **untracked/removed** — never reference it as an analog (historical only, per RESEARCH). All excerpts below are from tracked source.

---

## Pattern Assignments

### `project_probe/detect_go.go` (detector, file-I/O → text transform)

**Analog:** `project_probe/registry.go` (contract + dispatch) + `project_probe/manifest.go` (I/O)

**Detector contract to implement** (registry.go lines 3-6):
```go
// detectorFunc reports a project match for folder. It must never return an
// error: a non-match is (ProjectData{}, false). A recovered panic is treated
// as a non-match so Probe can never fail on content (D-03).
type detectorFunc func(folder string) (ProjectData, bool)
```

**Manifest I/O pattern — never-fail degrade (manifest.go lines 19-35, key lines 23-34):**
```go
f, err := os.Open(filepath.Join(folder, name))
if err != nil {
    return nil, false
}
defer f.Close() //nolint: errcheck

content, err = io.ReadAll(io.LimitReader(f, maxManifestSize+1))
if err != nil || len(content) > maxManifestSize {
    return nil, false
}

return bytes.TrimPrefix(content, utf8BOM), true
```

**Detector shape — RESEARCH Pattern 2/Code Examples (research.md:478-495) is the blueprint** (parseGoMod + presence-match; use `filepath.Base(folder)` for DATA-02 name fallback, `readmeDescription(folder)` for D-03 description). The `LanguageGo` constant to set (project.go lines 30-31):
```go
// LanguageGo identifies a Go project.
LanguageGo Language = "Go"
```

**Error handling:** none — never-fail. `readManifest` returns `(nil, false)`; detector returns `(ProjectData{}, false)`. Panic safety is inherited from `callDetector` (registry.go lines 28-37) — do NOT add try/catch-style handling inside the detector.

**Validation:** go.mod parse via `strings.Fields` + exact first-token match (per RESEARCH Pitfall 6: `gopher 1.2`, `modulex y`, `toolchain go1.26.4` must NOT match; strip only outer quotes on `module` value; `go` directive reported raw, never normalized — D-02).

---

### `project_probe/detect_js.go` (detector, file-I/O → JSON decode)

**Analog:** `project_probe/registry.go` (contract) + `httptest_mock/setup.go` (JSON decode precedent)

**JSON decode pattern** (httptest_mock/setup.go lines 277-298, key lines 282-291):
```go
var mock Mock
if data[0] == '{' {
    err = json.Unmarshal(data, &mock)
} else {
    err = yaml.Unmarshal(data, &mock)
}

if err != nil {
    return nil, fmt.Errorf("failed to unmarshal json/yaml: %w", err)
}
```
Phase 11's variant has no error return — decode failure means `(ProjectData{}, false)` (D-04). The struct shape (RESEARCH Pattern 3, research.md:299-319): anonymous struct with `Name`/`Version`/`Description` string fields and **no `Private bool` field** (D-06 is pinned by a test row, not a code branch).

**Language constant** (project.go lines 36-37):
```go
// LanguageJavaScript identifies a JavaScript project.
LanguageJavaScript Language = "JavaScript"
```

**DATA-02/DATA-04 chains:** `data.Name = m.Name; if data.Name == "" { data.Name = filepath.Base(folder) }`; `data.Description = m.Description; if data.Description == "" { data.Description = readmeDescription(folder) }`.

---

### `project_probe/detect_php.go` (detector, file-I/O → JSON decode)

**Analog:** `project_probe/detect_js.go` (mirror — write one, copy the other; RESEARCH Pattern 3 says "PHP is identical with `composer.json`, `LanguagePHP`, and the full `vendor/package` name (D-07)").

**Language constant** (project.go lines 48-49):
```go
// LanguagePHP identifies a PHP project.
LanguagePHP Language = "PHP"
```

Version stays `""` when absent — that is the ecosystem norm (Packagist infers from tags); never fabricate.

---

### `project_probe/json.go` (utility, file-I/O → JSON decode)

**Analog:** `httptest_mock/setup.go` (unmarshalMock) + `project_probe/manifest.go` (wrapper shape)

**Shared helper shape** (5 lines, RESEARCH Pattern 3 research.md:278-284):
```go
func readJSONManifest(folder, name string, v any) bool {
    content, ok := readManifest(folder, name)
    if !ok {
        return false
    }
    return json.Unmarshal(content, v) == nil
}
```
Imports: `encoding/json` — matches httptest_mock/setup.go import usage (line 284). Never call `json.Unmarshal` on raw `os.ReadFile` output — the BOM strip in `readManifest` (manifest.go line 34) is load-bearing (RESEARCH Pitfall 1).

---

### `project_probe/readme.go` (utility, file-I/O → text extraction)

**Analog:** `project_probe/manifest.go` (readManifest for each candidate read)

**Candidate-read loop pattern** — reuse `readManifest(folder, name)` per candidate (manifest.go lines 19-35), exactly like the manifest reads, but ordered: `README.md` → `README.rst` → `README` (D-08, exact-case). First `ok=true` wins; none → `""` (D-10). Note the current `readManifest` doc comment says "is a regular file" (manifest.go lines 16-18) but the code does NOT yet enforce `Mode().IsRegular()` — the WR-01 FIFO gate is an open question (RESEARCH OQ-1); if the planner includes it, add the `f.Stat()` check after `os.Open` at manifest.go line 23.

**Imports pattern:** `bytes`, `strings`, `path/filepath` (stdlib-only mandate, STACK.md) — matches manifest.go lines 3-8 import block style.

**No in-repo analog for the heuristic itself** — see "No Analog Found" below; the extraction matrix contract is RESEARCH Pitfall 5 (research.md:398-402).

---

### `project_probe/registry.go` (MODIFIED — config, cascade dispatch)

**Analog:** itself. Two surgical edits:

**Edit 1 — replace the empty literal (registry.go line 11) with the nil-slot literal** (RESEARCH Pattern 1, research.md:198-206):
```go
var detectors = []detectorFunc{
    detectGo,  // index 0 — Go (Phase 11)
    nil,       // index 1 — Python (Phase 12)
    nil,       // index 2 — C#/.NET (Phase 13)
    detectJS,  // index 3 — JavaScript/TypeScript (Phase 11)
    nil,       // index 4 — Rust (Phase 12)
    nil,       // index 5 — Java/Kotlin (Phase 13)
    detectPHP, // index 6 — PHP (Phase 11)
}
```
Update the comment block above the var (lines 8-10) to reflect the live entries.

**Edit 2 — nil-skip in runDetectors (registry.go lines 15-23):**
```go
for _, fn := range detectors {
    if fn == nil { // empty slot (Phase 12/13) — skip
        continue
    }
    if pd, ok := callDetector(fn, folder); ok {
        return pd, true
    }
}
```
The nil-skip must come before `callDetector` — a nil entry would panic inside `callDetector`'s `fn(folder)` call, get swallowed as a non-match, and spam recovery paths (RESEARCH Pattern 1 rationale).

**Keep unchanged:** `detectorFunc` type (lines 3-6), `callDetector` panic recovery (lines 28-37).

---

### `project_probe/doc.go` (MODIFIED — docs)

**Analog:** itself, lines 35-37 — the stale closing paragraph:
```go
// Detector implementations land in phases 11-13: the ordered registry
// (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) is empty in this
// phase, so every content-bearing folder currently yields LanguageUnknown.
```
Rewrite to name the three live detectors (Go/JS/PHP) and the shared `readmeDescription` fallback. The rest of doc.go (never-fail contract, usage, sentinels — lines 1-34) is unchanged.

---

### `project_probe/registry_test.go` (MODIFIED — test)

**Analog:** itself. Two changes per RESEARCH OQ-3/Pitfall 4:

1. **Refresh stale comments** — `TestRunDetectors_EmptyRegistry`'s comment (lines 16-17, "pins the Phase-10 default state") is now false; the test still passes (empty cascade input `""` matches nothing) but the comment must be updated.
2. **Add `TestDetectorPositions`** (RESEARCH Code Examples, research.md:464-476):
```go
func TestDetectorPositions(t *testing.T) { //nolint:paralleltest // reads global detectors
    assert.Len(t, detectors, 7)
    assert.NotNil(t, detectors[0]) // Go
    assert.Nil(t, detectors[1])    // Python — Phase 12
    assert.Nil(t, detectors[2])    // C#/.NET — Phase 13
    assert.NotNil(t, detectors[3]) // JS/TS
    assert.Nil(t, detectors[4])    // Rust — Phase 12
    assert.Nil(t, detectors[5])    // Java/Kotlin — Phase 13
    assert.NotNil(t, detectors[6]) // PHP
}
```

**Save/restore pattern to keep for all existing tests** (registry_test.go lines 18-20):
```go
original := detectors
defer func() { detectors = original }()
```
And the file-level no-parallel note (lines 12-14) governs all registry-mutating tests — extend it to new detector tests that mutate `detectors` (AGENTS.md global-state rule).

---

### `project_probe/detect_go_test.go`, `detect_js_test.go`, `detect_php_test.go` (new tests)

**Analog:** `project_probe/manifest_test.go` — the table-driven inline-fixture pattern (repo convention, TESTING.md: no `testdata/` dirs; inline `t.TempDir()` + `os.WriteFile`).

**Table shape** (manifest_test.go lines 26-31):
```go
tests := []struct {
    name   string
    file   string
    setup  func(t *testing.T) (string, []byte) // returns folder + expected content (nil when !wantOK)
    wantOK bool
}{...}
```

**Fixture mechanics** (manifest_test.go lines 41-49):
```go
folder := t.TempDir()
require.NoError(t, os.WriteFile(filepath.Join(folder, manifestName), content, 0o600))
```

**Key rows to pin per RESEARCH** (from the verified behavior tables, research.md:287-294 and Pitfalls 5-7):
- BOM'd manifest: `[]byte("\xEF\xBB\xBF{...}")` (mirror manifest_test.go lines 66-79 `bom_stripped` row — BOM strip is load-bearing for JSON, RESEARCH Pitfall 1)
- `"private": true` without version → Version `""` (D-06 row, no code branch)
- `"name": "@scope/pkg"` preserved verbatim; `"description": null` → `""` → README fallback triggers
- go.mod rows: quoted module `module "example.com/dq"`, trailing comment, `go 1.21rc1`, `toolchain go1.26.4` must NOT set Version, `gopher 1.2`/`modulex y` non-matches (Pitfall 6)
- Malformed JSON → `false` (detector degrades, D-04)
- Detector tests may `t.Parallel()` if they do NOT touch the `detectors` slice (manifest_test.go line 24 is parallel; registry_test.go is not — same rule)

---

### `project_probe/readme_test.go` (new test)

**Analog:** `project_probe/probe_test.go` (table with `setup func(t) string`) + `manifest_test.go` (fixture mechanics).

**Extraction matrix is the contract** — RESEARCH Pitfall 5 (research.md:401) lists every row: badge-only line, badge+text line (NOT skipped), `- [TOC](#x)`, `1. [TOC](#x)`, rst `Title\n=======`, markdown setext `Title\n---`, ATX `# Title`, `...` line, `---` between blank lines, multi-line paragraph (joined with single space, D-disc-4), HTML comment preamble, empty README, whitespace-only README, lowercase `readme.md` alone → `""` (D-08 exact-case, Pitfall 7). Fixtures inline as string constants in the table (D-disc-6).

---

## Shared Patterns

### Never-fail `(ProjectData, bool)` contract
**Source:** `project_probe/registry.go:3-6` + `project_probe/manifest.go:16-18`
**Apply to:** All three detectors + `readJSONManifest` + `readmeDescription`
```go
// detectorFunc reports a project match for folder. It must never return an
// error: a non-match is (ProjectData{}, false).
type detectorFunc func(folder string) (ProjectData, bool)
```
Every failure mode (missing file, malformed JSON, unparseable go.mod, no README) degrades to a non-match or `""` — never an error, never a panic.

### Manifest I/O — `readManifest` is the only file access
**Source:** `project_probe/manifest.go:19-35`
**Apply to:** All six call sites (3 manifests × 3 README candidates)
```go
return bytes.TrimPrefix(content, utf8BOM), true
```
1 MB cap (line 12) + BOM strip (line 14) inherited by every detector — do not call `os.ReadFile` or `json.Unmarshal` on raw bytes anywhere in detectors.

### Panic containment — cascade dispatch
**Source:** `project_probe/registry.go:28-37`, precedent `cache/singleflight.go:73-88` (`callSetter` recover at lines 81-85)
**Apply to:** `runDetectors` (with the new nil-skip before `callDetector`)
```go
defer func() {
    if r := recover(); r != nil {
        // treat as non-match (never re-panic)
        pd, ok = ProjectData{}, false
    }
}()
```

### Detector body shape (name/version/description chains)
**Source:** RESEARCH Pattern 3 (research.md:299-319) + `project_probe/project.go` constants
**Apply to:** All three detectors — `Language` set first (constants at project.go:30-31 Go, 36-37 JS, 48-49 PHP), then DATA-02 chain (`Name` → `filepath.Base(folder)`), then DATA-04 chain (`Description` → `readmeDescription(folder)`); `Version` raw or `""` (DATA-03, never normalized/fabricated).

### Merge rule — detector output overrides Unknown defaults
**Source:** `project_probe/probe.go:44-49` — unchanged, but the new detectors feed it:
```go
if pd, ok := runDetectors(clean); ok {
    data.Language = pd.Language
    data.Name = pd.Name
    data.Version = pd.Version
    data.Description = pd.Description
}
```

### Test fixtures — inline `t.TempDir()` + `os.WriteFile`
**Source:** `project_probe/manifest_test.go:26-31,41-49` (repo convention, TESTING.md; no `testdata/` dirs exist in the repo)
**Apply to:** All new test files
```go
setup  func(t *testing.T) (string, []byte) // returns folder + expected content
...
folder := t.TempDir()
require.NoError(t, os.WriteFile(filepath.Join(folder, manifestName), content, 0o600))
```

### No `t.Parallel()` on global-state mutation
**Source:** `project_probe/registry_test.go:12-14` (AGENTS.md mid/collectFuncs precedent)
**Apply to:** `TestDetectorPositions` and any test that mutates the `detectors` slice; detector unit tests that only touch `t.TempDir()` folders may parallelize (manifest_test.go line 24).

### Coverage gate known-red
**Source:** `.testcoverage-quick.yml` thresholds + RESEARCH Pitfall 2 (research.md:380-384)
**Apply to:** Every commit — `make coverage-quick` fails on `release/update.go` 68.9% (pre-existing, deferred). Assert project_probe rows (pkg ≥80%, file ≥70%) + total ≥75% pass; document the known-red in commit messages.

---

## No Analog Found

Files/behaviors with no close match in the codebase (planner should use RESEARCH.md patterns):

| File / Behavior | Role | Data Flow | Reason |
|-----------------|------|-----------|--------|
| `firstRealParagraph` heuristic (in `readme.go`) | utility | text extraction | No existing code does line-heuristic text extraction; RESEARCH Pattern 4 skeleton (research.md:431-460) is the blueprint, and the behavior matrix (research.md:398-402) is the test contract. `httptest_mock` YAML parsing is the closest spirit but different domain |
| Nil-slot registry literal representation | config | — | The Phase 10 `var detectors = []detectorFunc{}` (registry.go:11) has no precedent for placeholder slots; RESEARCH Pattern 1 (research.md:193-220) is the design |
| DATA-02 folder-base fallback | utility | transform | `filepath.Base` stdlib — no in-repo wrapper exists; A7 edge rows (`sub/dir` → `dir`) come from RESEARCH (research.md:520) |

## Metadata

**Analog search scope:** `project_probe/` (11 files, whole package read), `cache/` (singleflight.go panic-recovery precedent), `httptest_mock/` (setup.go JSON decode precedent)
**Files scanned:** 14 (project_probe 11 + cache/singleflight.go + httptest_mock/setup.go + httptest_mock/request.go grep)
**Tracked-source gate:** all analog paths verified via `git ls-files`; `project_detector/` excluded (deleted/untracked)
**Pattern extraction date:** 2026-09-29
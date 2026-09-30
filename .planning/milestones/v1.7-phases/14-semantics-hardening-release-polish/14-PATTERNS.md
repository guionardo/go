# Phase 14: Semantics, Hardening, and Release Polish - Pattern Map

**Mapped:** 2026-09-29
**Files analyzed:** 11 (1 new test file + 3 new testdata fixture dirs, 8 modified)
**Analogs found:** 10 / 11 (fuzz corpus fixtures have no codebase analog — RESEARCH Pattern 2 is the only source)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `project_probe/fuzz_test.go` (NEW) | test (fuzz targets ×3, internal package) | file-I/O → detector parse paths (`readManifest` → decode → chains); no assertions on results | `project_probe/detect_csharp_test.go:22-46` (fixture-write body: `t.TempDir` + `os.WriteFile` + detector call) — fuzz mechanism itself has no repo analog (grep `^func Fuzz` → zero hits); use RESEARCH Pattern 1 verified skeleton | role-match (fixture body) / no-analog (fuzz harness) |
| `project_probe/testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/` (NEW) | fixture (seed corpus) | — (consumed by `internal/fuzz.ReadCorpus` at test start) | none — corpus files must be `go test fuzz v1`-encoded (RESEARCH Pattern 2 verified format; raw bytes fail the build) | no analog |
| `project_probe/readme.go` (MOD) | utility (pure funcs `isBadgeLine`, `firstRealParagraph`) | transform (README text → first paragraph) | itself — `readme.go:113-135` (isBadgeLine), `readme.go:55-86` (firstRealParagraph switch) | exact |
| `project_probe/detect_csharp.go` (MOD) | detector (cascade entry, pure func) | file-I/O → discovery (`readFirstManifest`) → inline decode → transform | itself — `detect_csharp.go:76-93` (collect loop), `detect_csharp.go:25` (exact-name guard), `detect_csharp.go:71` (decode-error comment) | exact |
| `project_probe/detect_java_kotlin.go` (MOD) | detector (cascade entry, pure func) | file-I/O → inline decode → transform | itself — `detect_java_kotlin.go:97-111` (Name/Version/Description chains) | exact |
| `project_probe/registry_test.go` (MOD) | test | — (global-state mutation: `detectors` slice) | itself — `registry_test.go:21-28` (`TestRunDetectors_EmptyRegistry` save/restore shape; `TestRunDetectors_OrderAndFirstMatch:33-46` shows the `[]detectorFunc{}` literal form) | exact |
| `project_probe/readme_test.go` (MOD) | test | pure-content matrix rows | itself — `readme_test.go:127-158` (`TestFirstRealParagraph` matrix: name/content/want rows, `t.Parallel` subtests) | exact |
| `project_probe/detect_csharp_test.go` (MOD) | test | file-I/O fixtures + table rows | itself — `detect_csharp_test.go:362-419` (`TestDetectCSharp` table with `folder func(t) string` column), `detect_csharp_test.go:188-218` (`TestProbe_CSharpVersionPrefixChain` content→wantVer table) | exact |
| `project_probe/detect_java_kotlin_test.go` (MOD) | test | file-I/O fixtures + table rows | `project_probe/detect_csharp_test.go:362-419` (same detector-level table shape) | exact |
| `release/update_test.go` (MOD) | test | request-response (httptest mock GitHub API) | itself — `update_test.go:87-132` (`TestCheckForUpdate_NewerVersion`: `mu` lock + `githubAPIBase` save/restore + `httptest.NewServer` handler), `update_test.go:247-280` (`TestDownloadUpdate`: asset with digest + TempDir) | exact |
| `README.md` (MOD) | docs | — | itself — `README.md:13-36` (sorted package table; new row between `pathtools` :31 and `reflecttools` :32), `README.md:44-54` (`### Package brdocs` per-package section shape) | exact |
| `project_probe/doc.go` (MOD) | docs | — | itself — `doc.go:35-53` (detector-list paragraph; append Version-semantics contract after it, before `package projectprobe` at :54) | exact |

**Tracked-source gate (#3645):** every analog above verified git-tracked (`git ls-files -- project_probe/... release/... README.md` non-empty for all 10 named files — confirmed at mapping time). No mirrors, no submodules. All excerpts below are from tracked source.

**Coverage-gate invariant (Pitfall 4):** `release/update.go` is NOT modified by any plan — the gate closes with test-only additions (74 stmts / 51 covered = 68.9%; one more covered statement closes the 70% file gate; the no-options row covers 13).

---

## Pattern Assignments

### `project_probe/fuzz_test.go` (test — fuzz targets, file-I/O → detector parse paths)

**Analog:** `project_probe/detect_csharp_test.go` (fixture-write body) + RESEARCH Pattern 1 skeleton (no repo fuzz precedent — grep `^func Fuzz` → zero matches)

**File header / package:** `package projectprobe` (internal — unexported detectors reachable). Imports: `os`, `path/filepath`, `testing` (RESEARCH.md:210-214).

**Fixture-write body pattern** (copy from `detect_csharp_test.go:24-38`, `t.TempDir` + `os.WriteFile` + `0o600`):
```go
folder := t.TempDir()
require.NoError(t, os.WriteFile(
	filepath.Join(folder, "MyApp.csproj"),
	[]byte(`...`),
	0o600,
))
```
In the fuzz body the `require` becomes `t.Skipf` on write error (RESEARCH.md:226-229):
```go
dir := t.TempDir() // works in seed AND fuzz mode (verified go1.27.0)
if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o600); err != nil {
	t.Skipf("write: %v", err)
}
detectJS(dir) // no-panic is the contract; result ignored
```

**Fuzz-target skeleton** (RESEARCH.md:221-236, verified live) — one file, three targets, same shape:
- `FuzzJSONManifest`: writes `package.json` → `detectJS(folder)`; writes `composer.json` → `detectPHP(folder)`
- `FuzzTOMLManifest`: writes `pyproject.toml` → `detectPython(folder)`; writes `Cargo.toml` → `detectRust(folder)`
- `FuzzXMLManifest`: writes `MyApp.csproj` → `detectCSharp(folder)`; writes `pom.xml` → `detectJavaKotlin(folder)` — file name must use the exact-name guard shape (never `.csproj` alone, 13 IN-03)

Detector signatures (all `func(folder string) (ProjectData, bool)`): `detectJS` (detect_javascript.go:19), `detectPHP` (detect_php.go:18), `detectPython` (detect_python.go:15), `detectRust` (detect_rust.go:18), `detectCSharp` (detect_csharp.go:65), `detectJavaKotlin` (detect_java_kotlin.go:90).

**Executor contract (Pitfall 3):** NEVER assert on the detector bool inside `f.Fuzz` — presence-match (TOML/XML) returns `ok=true` on garbage, parse-success (JSON) returns `false`; both correct. No-panic needs no assertion. `t.Parallel()` forbidden in fuzz targets. Keep 3-5 `f.Add` seeds per target AND the corpus files (both run under plain `go test`). Never `-fuzz` in CI/plan/Makefile.

### `project_probe/testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/` (fixture — seed corpus)

**No codebase analog.** The format is the phase's #1 trap (RESEARCH Pattern 2, verified): corpus files are `go test fuzz v1`-ENCODED, not raw manifest bytes — raw bytes fail the whole package build with `unmarshal: must include version and at least one value`.

```text
go test fuzz v1
[]byte(`{"name":"acme","version":"1.2.3"}`)
```
```text
go test fuzz v1
[]byte("<Project>\n  <Version> 1.2.3 </Version>\n</Project>")
```

Encoding rules (RESEARCH.md:261-266): one value per line; backtick literal CANNOT span lines (use `\n` escapes in quoted literals); BOM needs escaped literal `[]byte("\xef\xbb\xbf{\"name\":\"bom\"}")`; filenames are free-form (readable names OK — `ReadCorpus` reads every file). Seed inventory per RESEARCH.md:268: real-world shapes (package.json name/version/description; composer.json vendor/package without version; pyproject `[project]` + `[tool.poetry]` + `dynamic = ["version"]`; Cargo `version.workspace = true`; SDK + old-style .csproj; pom.xml with `<parent><version>` and no own version) PLUS pinned malformed shapes (truncated JSON/XML, BOM-prefixed, whitespace-only XML elements = 13 WR-01 shape, 6-quote close+reopen TOML = 12 CR-01 shape, bracket-in-string TOML). Verify with `go test ./project_probe/...` before commit — green means the corpus is well-formed.

### `project_probe/readme.go` (utility — WR-01 plain badge, WR-02 comment-block state)

**Analog:** itself.

**Fix 1 — plain-badge remainder (11 WR-01)** — `isBadgeLine` final return, current `readme.go:134`:
```go
return strings.TrimSpace(line) == "" || strings.Trim(line, "!") == ""
```
The `![`-presence guard at `readme.go:115` keeps prose (`!!!`, `!important`) out; a stripped `![logo](x)` leaves exactly `"!"` → badge.

**Fix 2 — comment-block state (11 WR-02)** — `firstRealParagraph` switch, `readme.go:58-80`. `inComment` must be the FIRST case so blank lines/comment bodies inside the block are consumed (a blank line must not return a partial paragraph). The existing single-line case at `readme.go:74` stays untouched:
```go
var inComment bool
for _, line := range strings.Split(string(content), "\n") {
	trimmed := strings.TrimSpace(line)
	switch {
	case inComment:
		if strings.Contains(trimmed, "-->") {
			inComment = false
		}
		prev = ""
	case strings.HasPrefix(trimmed, "<!--") && !strings.HasSuffix(trimmed, "-->"):
		inComment = true // multi-line comment opens — consume until "-->"
		prev = ""
	// ... existing cases unchanged (blank, underline, badge/TOC/heading/single-line comment, default)
```

**Test rows** — append to the `TestFirstRealParagraph` matrix (`readme_test.go:135-149`, `name/content/want` rows, `t.Parallel` subtests at :151-157):
```go
{"plain_image_badge", "![logo](x)\npara\n", "para"},
{"multi_line_html_comment", "<!--\nBanner\n-->\npara\n", "para"},
```
(plus a shields.io-style URL row per RESEARCH.md:302).

### `project_probe/detect_csharp.go` (detector — 13 WR-01 trim, 13 IN-03 exact-name guard, 13 IN-01 comment)

**Analog:** itself.

**Trim per field** — inside the collect loop, current `detect_csharp.go:76-93` (5 assignments; `strings` already imported at :7):
```go
for _, pg := range proj.PropertyGroups {
	if assemblyName == "" {
		assemblyName = strings.TrimSpace(pg.AssemblyName)
	}
	// ... same for rootNamespace, version, versionPrefix, description
}
```
This is decode hygiene, NOT version normalization (Pitfall 2) — the D-03 audit and doc.go contract must say so explicitly.

**Exact-name guard (13 IN-03)** — `readFirstManifest` loop, current `detect_csharp.go:25`:
```go
if e.IsDir() || len(e.Name()) <= len(suffix) || !strings.HasSuffix(e.Name(), suffix) {
	continue
}
```
A file literally named `.csproj` no longer passes `HasSuffix` (len check first). Fixture row: file named exactly `.csproj` → non-match, folder falls through to next detector.

**Comment reword (13 IN-01)** — `detect_csharp.go:71`, replace "decode error → zero struct" with the verified behavior:
```go
_ = xml.Unmarshal(content, &proj) // decode error → whatever decoded before the error point survives; the rest degrades to empty (IN-01)
```

**Test rows** — `detect_csharp_test.go:362-419` (`TestDetectCSharp` table: `name/content/folder/wantName/wantVer/wantDesc` with `wantName == ""` → `filepath.Base(folder)`): padded values → trimmed output; whitespace-only `<Version> </Version>` + `<VersionPrefix>` → prefix chain; whitespace-only `<name> </name>` → folder-base fallback; exact-`.csproj` name → no match.

### `project_probe/detect_java_kotlin.go` (detector — 13 WR-01 trim, 13 IN-01 comment)

**Analog:** itself.

**Trim before chains** — current `detect_java_kotlin.go:97-111` (apply `strings.TrimSpace` to every decoded field BEFORE chain resolution; `strings` already imported at :7):
```go
data.Name = strings.TrimSpace(pom.Name) // <name> — Maven's optional display name
if data.Name == "" {
	data.Name = strings.TrimSpace(pom.ArtifactID) // <artifactId> — the required coordinate
}
if data.Name == "" {
	data.Name = filepath.Base(folder) // DATA-02
}
data.Version = strings.TrimSpace(pom.Version)
if data.Version == "" && pom.Parent != nil {
	data.Version = strings.TrimSpace(pom.Parent.Version) // D-05: single-level parent inheritance (same file)
}
data.Description = strings.TrimSpace(pom.Description)
```

**Comment reword (13 IN-01)** — `detect_java_kotlin.go:95`: same "whatever decoded before the error point survives" wording. Optional: truncated-after-complete-group test row (doubles as a fuzz seed).

**Test rows** — same shape as `detect_csharp_test.go:362-419`: padded values → trimmed; whitespace-only `<version> </version>` with `<parent><version>1.2.3</version></parent>` → `1.2.3` (parent fires); whitespace-only `<name> </name>` → artifactId/folder base.

### `project_probe/registry_test.go` (test — 12/13 WR-02 explicit empty slice)

**Analog:** itself.

**`TestRunDetectors_EmptyRegistry`** — current `registry_test.go:21-28` saves/restores `detectors` but never mutates it (runs the production 7-slot registry against `""`). Inject the empty slice and use a folder that cannot collide (the `[]detectorFunc{}` literal form is already in this file at :37):
```go
func TestRunDetectors_EmptyRegistry(t *testing.T) { //nolint:paralleltest // global-state mutation: detectors slice
	original := detectors
	detectors = []detectorFunc{} // explicit: no live detectors
	defer func() { detectors = original }()

	pd, ok := runDetectors("x")
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}
```
Keep the file's no-`t.Parallel()` discipline (`registry_test.go:12-14` note). Do NOT re-edit the `TestDetectorPositions` comment (`registry_test.go:65-69`) — already refreshed in Phase 13.

### `project_probe/*_test.go` (test — fold-in rows + comment refresh)

**Analog:** `readme_test.go:127-158` (pure matrix) and `detect_csharp_test.go:362-419` (detector table) — both shown above. All new rows follow the repo convention: inline fixtures via `t.TempDir` + `os.WriteFile` (0o600), `t.Parallel()` where no global state is touched, `testify` assert/require, `//nolint:funlen` on growing tables (`readme_test.go:23,127` precedent).

### `release/update_test.go` (test — coverage-gate closure, request-response via httptest)

**Analog:** itself.

**Global-state discipline (mandatory for every new row)** — `TestCheckForUpdate_NewerVersion`, `update_test.go:87-124`: `mu.Lock()`/`defer mu.Unlock()` + `githubAPIBase` save/restore + `httptest.NewServer` handler. The no-options row needs `mu` too (it mutates `githubAPIBase`). All rows keep `t.Parallel()` AFTER the lock (existing rows call it first — keep the same order).

**The mandatory row — `TestCheckForUpdate_NoOptions`** (RESEARCH Pattern 3; closes the gate: 51 + 13 = 64/74 = 86.5%):
```go
func TestCheckForUpdate_NoOptionsDerivesOwnerRepo(t *testing.T) { //nolint:paralleltest // global githubAPIBase
	mu.Lock()
	defer mu.Unlock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/guionardo/go/releases/latest", r.URL.Path) // derived from module path
		_, _ = fmt.Fprint(w, `{"tag_name": "v2.0.0", "name": "v2.0.0", "assets": []}`)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	rel, newer, err := CheckForUpdate(context.Background(), "v1.0.0") // NO options → module derivation
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.True(t, newer)
}
```
Asserts the derived path in the handler — pins `getCurrentModule()` → `url.Parse` → `words[1]/words[2]` derivation (`update.go:59-81`, the 13 dead statements).

**Seven recommended error-path rows** (cross-platform ENOTDIR triggers — NO chmod, AGENTS.md Windows rule; each ~10 lines mirroring the shapes above):

| Row | Trigger | Covers (update.go) |
|-----|---------|---------------------|
| `TestCheckForUpdate_RequestCreationError` | `githubAPIBase = "http://exa mple.com"` (space in host → `NewRequestWithContext` fails) | 91-92 |
| `TestCheckForUpdate_NetworkError` | `httptest.NewServer` then `server.Close()`; use its URL → connection refused | 102-103 |
| `TestCheckForUpdate_InvalidReleaseJSON` | server returns `not-json` with 200 | 112-113 |
| `TestCheckForUpdate_InvalidReleaseVersion` | `tag_name: "not-a-version"` | 117-118 |
| `TestDownloadUpdate_MkdirAllError` | `targetDir = filepath.Join(existingFile, "sub")` → ENOTDIR | 130-131 |
| `TestDownloadUpdate_CreateError` | `Asset{Name: "sub/" + goos + "_" + goarch + ".tar.gz"}` (slash in matched name) | 136-137 |
| `TestDownloadUpdate_DigestMismatch` | server content digest ≠ `asset.Digest` → `os.Remove` cleanup | 141-144 |

`TestDownloadUpdate` (update_test.go:247-280) is the digest-row analog: asset built with `digest.FromBytes(content)` + `BrowserDownloadURL: server.URL + "/download"` + `dir := t.TempDir()`.

### `README.md` (docs — D-10 package index row)

**Analog:** itself.

**Row** — insert alphabetically between `pathtools` (README.md:31) and `reflecttools` (README.md:32), keeping the `| [name](#package-<dir>) | \`import\` | description |` anchor format:
```markdown
| [projectprobe](#package-project_probe) | `project_probe` | Best-effort project detection across 7 languages (manifest-first, stdlib-only, never fails) |
```

**Section** — follow the `### Package brdocs` shape (README.md:44-54: import line `Import \`github.com/guionardo/go/project_probe\``, one-paragraph description of the never-fail contract + 7-detector cascade, a `Probe` usage snippet). Row is the lock; section wording is executor discretion.

### `project_probe/doc.go` (docs — D-02 Version-semantics contract)

**Analog:** itself.

**Insert** the Version-semantics contract section after the detector-list paragraph (`doc.go:35-53`), before `package projectprobe` (`doc.go:54`) — RESEARCH Pattern 7 verified wording:
```go
// # Version semantics (DATA-03)
//
// Version is always the raw manifest string — verbatim, never normalized,
// never fabricated. Absent or unsupported values yield "". The rules:
//
//   - go.mod: the "go" directive, reported as the toolchain floor — the
//     minimum Go version the module was written for, NOT the release
//     version; "toolchain" lines are never the directive.
//   - pyproject.toml: the [project] version; a "dynamic" version array
//     degrades to "" — never resolved, never guessed.
//   - Cargo.toml: the [package] version; version.workspace = true degrades
//     to "" — never resolved from a workspace root.
//   - .csproj: <Version>, falling back to <VersionPrefix>; the MSBuild
//     implicit 1.0.0 default is never reported.
//   - pom.xml: <version>, inheriting <parent><version> when absent (single
//     level, from the same file); the Maven Super POM 4.0.0 default is
//     never reported.
//   - package.json / composer.json: the "version" field verbatim.
//
// Placeholders ($(...) in MSBuild, ${...} in Maven) are reported raw, never
// resolved. XML element text is whitespace-trimmed at decode time (decode
// hygiene — the value itself is still never normalized).
```
Precedent for the section style: `doc.go:7-17` (`# Never-fail contract`).

---

## Shared Patterns

### Test fixtures: inline `t.TempDir` + `os.WriteFile`, never `testdata/` for runtime fixtures
**Source:** `project_probe/detect_csharp_test.go:24-38`, `readme_test.go:15-18` (writeReadme helper)
**Apply to:** all new/modified project_probe test rows; release tests use `t.TempDir()` for `DownloadUpdate` targets (`update_test.go:271`).
```go
folder := t.TempDir()
require.NoError(t, os.WriteFile(filepath.Join(folder, "MyApp.csproj"), []byte(content), 0o600))
```
The ONE exception: `testdata/fuzz/` corpus files (encoded per RESEARCH Pattern 2) — they are Go-fuzz fixtures, not test fixtures.

### Global-state mutation discipline (no `t.Parallel`)
**Source:** `project_probe/registry_test.go:12-14,21` (`detectors` slice, `//nolint:paralleltest`); `release/update_test.go:87-90` (`mu` + `githubAPIBase`)
**Apply to:** `registry_test.go` edit, all new `release/update_test.go` rows. Tests touching only `t.TempDir` fixtures MAY use `t.Parallel()` (`detect_csharp_test.go:12-14`).

### Never-fail degrade + presence-match
**Source:** `detect_csharp.go:65-69`, `detect_java_kotlin.go:90-125`
**Apply to:** fuzz bodies (no-panic is the invariant; detector bools never asserted — Pitfall 3) and the doc.go contract. All 7 detectors return `(ProjectData, bool)` — never an error from content.

### `//nolint:funlen` on growing test tables
**Source:** `readme_test.go:23,127`, `detect_csharp_test.go:362`
**Apply to:** `TestFirstRealParagraph`, `TestDetectCSharp`, `TestDetectJavaKotlin` after adding the fold-in rows.

### Coverage gate
**Source:** `.testcoverage-quick.yml` (file:70 / pkg:80 / total:75) via `make coverage-quick`
**Apply to:** every plan's verify step. `release/update.go` must NOT be modified (test-only closure); project_probe edits must not introduce uncalled unexported helpers (Phase 13 D-disc-8 lesson: an uncalled helper → 0% file coverage → gate failure — the XML trim is inline `strings.TrimSpace` assignments, NOT a `readXMLManifest` helper).

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `project_probe/fuzz_test.go` (fuzz mechanism) | test | file-I/O → detector parse paths | No `Fuzz` functions exist in the repo (grep `^func Fuzz` → zero hits); RESEARCH Pattern 1 skeleton (14-RESEARCH.md:205-245) is the verified source — the fixture-write body pattern still comes from `detect_csharp_test.go` |
| `project_probe/testdata/fuzz/Fuzz{JSON,TOML,XML}Manifest/*` | fixture | — | `go test fuzz v1`-encoded corpus format (14-RESEARCH.md Pattern 2, verified live) — no repo precedent; raw bytes in these dirs FAIL the build |

## Metadata

**Analog search scope:** `project_probe/` (all 25 tracked files scanned for detector/test/doc analogs), `release/` (update.go + update_test.go + self_update_test.go shapes), repo root (`README.md`, `go.mod`, `.testcoverage-quick.yml`), prior-phase artifacts (13-PATTERNS.md format precedent).
**Files scanned:** 12 (10 read in full; grep-verified detector signatures + zero-fuzz-search across the repo)
**Pattern extraction date:** 2026-09-29
**Verification notes:** all analogs git-tracked (`git ls-files` non-empty); fuzz-corpus format and coverage math verified by phase research on go1.27.0 (14-RESEARCH.md Confidence: HIGH).
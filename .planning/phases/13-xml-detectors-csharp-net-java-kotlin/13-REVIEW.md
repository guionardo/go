---
phase: 13-xml-detectors-csharp-net-java-kotlin
reviewed: 2026-09-29T22:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - project_probe/detect_csharp.go
  - project_probe/detect_csharp_test.go
  - project_probe/detect_java_kotlin.go
  - project_probe/detect_java_kotlin_test.go
  - project_probe/registry.go
  - project_probe/registry_test.go
  - project_probe/doc.go
  - project_probe/detect_php_test.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 13: Code Review Report

**Reviewed:** 2026-09-29T22:00:00Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

The final two detectors (`detectCSharp` at registry index 2, `detectJavaKotlin` at index 5) complete the 7-slot cascade. Reviewed against the locked contracts D-01..D-10 in 13-CONTEXT.md / 13-RESEARCH.md. Verification performed: full `go test ./project_probe/...` suite (236 tests pass), plus an executable probe of the production decode shape against adversarial inputs (pretty-printed XML, whitespace-only elements, truncated documents, entity decoding).

The load-bearing contracts hold: XMLName local-name matching is namespace-agnostic for both roots (D-01), BOM is stripped at both boundaries (D-02), presence-match never fails and decode errors never panic (D-09), root-scoped discovery is single-level with the FIFO gate per candidate (D-04/D-07, WR-01 closed), registry order and 7-of-7 liveness are correct and pinned (D-08/D-10), `parseRootProjectName` never fabricates from comments or concatenations (D-06, D-disc-3), and versions are never fabricated (D-03/D-05). Cascade precedence rows for C#/Java are correctly appended and pass.

Two warnings: (1) `encoding/xml` does not trim element text — padded and whitespace-only values flow verbatim into `ProjectData` and block the D-03/D-05 fallback chains (verified live: `<Version> 1.2.3 </Version>` → `" 1.2.3 "`, `<name> </name>` → `" "` which suppresses the artifactId/folder-base fallback); (2) the 12-REVIEW WR-01 fragility in `TestRunDetectors_EmptyRegistry` was touched but not fixed — the test still runs the production 7-detector registry against the package directory, and its comment now claims an injection that does not exist. Three info items: the "decode error → zero struct" comments overstate the behavior (closed elements before the error point survive, verified), the gradle parser degrades legal Groovy block-comment and escaped-quote forms, and a file literally named `.csproj` passes the suffix filter.

## Warnings

### WR-01: XML element text is not whitespace-trimmed — padded/whitespace-only values flow verbatim and block the fallback chains

**File:** `project_probe/detect_csharp.go:76-110`, `project_probe/detect_java_kotlin.go:97-111`

**Issue:** `encoding/xml` copies character data verbatim — it does not trim. Verified against the production decode shape (Go 1.27.0):

| Input | Decoded field | Reported result |
|---|---|---|
| `<AssemblyName>\n      MyApp\n    </AssemblyName>` (pretty-printed) | `"\n      MyApp\n    "` | Name carries newlines/indentation |
| `<Version> 1.2.3 </Version>` | `" 1.2.3 "` | padded Version |
| `<Description>Line1\nLine2</Description>` | `"Line1\nLine2"` | embedded newline in Description |
| `<name> </name>` (whitespace-only) | `" "` | **non-empty** → the artifactId/folder-base fallback never fires; Name = `" "` |
| `<version>\t</version>` (whitespace-only) | `"\t"` | **non-empty** → the `<parent><version>` inheritance (D-05) and the never-fabricate `""` (D-03) never fire; Version = `"\t"` |

The whitespace-only cases are a genuine chain-contract violation (D-03/D-05: "when absent" — a whitespace-only element is treated as present), and the padded cases leak formatting into `ProjectData` — unlike every sibling reader in the package, which trims by construction (`readTOMLSection` trims values at toml.go:90,112,119; JSON values can't be padded; `parseGoMod` tokenizes). Pretty-printed/multi-line XML is a common real-world shape (IDE-reformatted poms, hand-formatted MSBuild property groups), so this is not a contrived corner.

**Fix:** trim each decoded field before chain resolution (or centrally after decode):

```go
// detect_java_kotlin.go (strings already imported)
data.Name = strings.TrimSpace(pom.Name)
if data.Name == "" {
    data.Name = strings.TrimSpace(pom.ArtifactID)
}
...
data.Version = strings.TrimSpace(pom.Version)
if data.Version == "" && pom.Parent != nil {
    data.Version = strings.TrimSpace(pom.Parent.Version)
}
data.Description = strings.TrimSpace(pom.Description)

// detect_csharp.go — trim inside the collect loop:
for _, pg := range proj.PropertyGroups {
    if assemblyName == "" {
        assemblyName = strings.TrimSpace(pg.AssemblyName)
    }
    // ... same for rootNamespace, version, versionPrefix, description
}
```

Add one test row per shape: padded values → trimmed output; whitespace-only `<Version> </Version>` + `<parent><version>` → parent version; whitespace-only `<name> </name>` → artifactId/folder base.

### WR-02: `TestRunDetectors_EmptyRegistry` runs the production registry against the package directory — comment now claims an injection that does not exist (12-REVIEW WR-01 carry-over)

**File:** `project_probe/registry_test.go:21-28`

**Issue:** 12-REVIEW WR-01 recommended injecting an explicit empty slice; the Phase 13 edit rewrote the comment instead of the test, and the new comment is factually wrong in both directions: "the injected slice below never matches" — there is no injected slice; "this test replaces the slice wholesale" — it does not (only `original := detectors` + restore). The test executes the **production 7-detector registry** against `runDetectors("")`, which resolves relative manifest paths in the test process CWD (the package dir). It passes only because `project_probe/` happens to contain none of the eight manifest names — and the Phase 13 additions made it *more* fragile: `detectCSharp` now matches any `.csproj` in the package dir via ReadDir discovery, and `detectJavaKotlin` matches `pom.xml`/`settings.gradle`. A tooling-added fixture at package root would silently flip this test to a match with a confusing failure.

**Fix:** make the test independent of the package directory:

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

## Info

### IN-01: "decode error → zero struct" comments overstate — closed elements before the error point survive

**File:** `project_probe/detect_csharp.go:71`, `project_probe/detect_java_kotlin.go:95`

**Issue:** Verified: `<Project><PropertyGroup><AssemblyName>X</AssemblyName></PropertyGroup><PropertyGroup><Version>1.0` (truncated) returns an error but decodes `PropertyGroups=[{AssemblyName:"X"}]` — the complete group survives, so the truncated file reports Name="X", not the folder-base fallback the comment promises. The behavior is acceptable presence-match degradation (the decoded value is real data, not fabrication), but the comments ("decode error → zero struct") are inaccurate for this input class, and the tests only pin the truncation-mid-element shape.

**Fix:** reword both comments to "decode error → whatever decoded before the error point survives; the rest degrades to empty" and optionally add a row pinning the truncated-after-complete-group shape.

### IN-02: `parseRootProjectName` degrades legal Groovy forms — block comments and escaped quotes after the literal

**File:** `project_probe/detect_java_kotlin.go:69-73`

**Issue:** The remainder guard only accepts empty or a `//` prefix, so a legal Groovy block comment (`rootProject.name = 'x' /* comment */`) degrades to `""` → folder-base Name. Likewise `rootProject.name = 'it\'s'` (escaped quote inside the literal) finds the escaped quote first and degrades — safe (never partial, never fabricated) but both are valid settings.gradle syntax that yield folder-base instead of the literal. Consistent with the locked D-disc-3 contract; flagging only so the degrade is a decision, not an accident.

**Fix:** optionally accept `/*`-prefixed remainders and skip backslash-escaped quotes when scanning for the closing quote; at minimum add test rows pinning both degrades.

### IN-03: A file literally named `.csproj` passes the suffix filter

**File:** `project_probe/detect_csharp.go:25`

**Issue:** `strings.HasSuffix(e.Name(), ".csproj")` is true for the bare name `".csproj"` — an accidentally-created empty/hidden file named exactly `.csproj` (no project name) is read, fails decode, and claims `LanguageCSharp` via presence-match with folder-base Name. Contrived and harmless, but trivially avoidable.

**Fix:** `if e.IsDir() || len(e.Name()) <= len(suffix) || !strings.HasSuffix(e.Name(), suffix) { continue }`.

---

_Reviewed: 2026-09-29T22:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
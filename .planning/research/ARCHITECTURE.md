# Architecture Research

**Domain:** Go stdlib-only project detection library (`project_probe` package)
**Researched:** 2026-09-28
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Public API (probe.go)                         │
│  Probe(folder string) (ProjectData, error)                           │
│  ProjectData{Folder, Language, Name, Version, Description}           │
│  Language constants (LanguageGo, LanguagePython, ... LanguageUnknown)│
├─────────────────────────────────────────────────────────────────────┤
│                        Registry (registry.go)                        │
│  os.ReadDir ONCE → filter ignored dirs → iterate ordered detectors   │
│  first match wins → Unknown fallback (never errors)                  │
├─────────────────────────────────────────────────────────────────────┤
│              Per-language detectors (detector_*.go ×7)               │
│  Go │ Python │ JS/TS │ .NET │ Rust │ Java/Kotlin │ PHP               │
│  each: root marker lookup + manifest parse + field fallbacks         │
├─────────────────────────────────────────────────────────────────────┤
│                Shared parsing helpers (stdlib-only)                  │
│  toml.go (minimal section-aware TOML subset)   readme.go (first ¶)   │
│  encoding/json (package.json, composer.json)   encoding/xml (pom,    │
│  .csproj)   bufio text scan (go.mod, gradle, .sln)                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `probe.go` | `ProjectData` struct, language constants, `Probe()` public API, doc.go | Exported types + one exported function |
| `registry.go` | Single `os.ReadDir`, ignored-dir filter, ordered `[]detectorFunc` iteration, Unknown fallback | ~60-line private loop; detectors slice is private, unexported |
| `detector_<lang>.go` (×7) | Claim a folder: check root markers, parse manifest, apply name/version/description fallbacks | One `detectX(folder string, entries []os.DirEntry) (ProjectData, bool)` per language |
| `toml.go` | Minimal TOML subset parser: track current `[section]`, extract `key = "value"` | Section-aware line scanner, ~80 lines; shared by Python + Rust detectors |
| `readme.go` | README first-paragraph extraction for Description fallback | `README.md`/`README` scan: skip H1/title, take first non-empty paragraph |
| `ignore.go` | Vendored/generated dir exclusion list | `map[string]struct{}` (node_modules, vendor, target, .venv, ...) — inherited from sample |

## Recommended Project Structure

```
project_probe/
├── doc.go                   # package documentation (repo convention: every package has one)
├── probe.go                 # ProjectData, Language constants, Probe() entry point
├── registry.go              # ordered detector slice + loop + Unknown fallback
├── ignore.go                # ignored directory names
├── readme.go                # README first-paragraph description fallback
├── toml.go                  # minimal section-aware TOML subset parser (shared)
├── detector_go.go           # go.mod (module + go directive)
├── detector_python.go       # pyproject.toml ([project] / [tool.poetry]) + setup.py marker
├── detector_javascript.go   # package.json (+ tsconfig.json refines language)
├── detector_dotnet.go       # *.sln / *.slnx / *.csproj (root level only)
├── detector_rust.go         # Cargo.toml ([package])
├── detector_java.go         # pom.xml / build.gradle / build.gradle.kts
├── detector_php.go          # composer.json
├── detector_*_test.go       # table-driven tests per language
└── testdata/                # fixture manifests + READMEs per language
    ├── go/go.mod
    ├── python/pyproject.toml
    ├── node/package.json
    ├── ...
```

### Structure Rationale

- **detector_<lang>.go per language:** mirrors the deprecated sample's proven file-per-language layout; adding Ruby or Terraform later = one new file + one registry entry. No changes to existing detectors.
- **registry.go as a separate tiny file:** makes the ordered slice visible and editable in one place; the ordering *is* the priority policy and deserves its own review surface.
- **Shared helpers (`toml.go`, `readme.go`) as separate files:** both are reused across ≥2 languages; colocating them in one detector file would force cross-language imports.
- **testdata/ fixtures over inline strings:** manifest formats are stable, verbose, and benefit from realistic fixtures; mirrors `httptest_mock`'s file-based test data approach already used in this repo.
- **Package name `projectprobe`** (directory `project_probe`): matches repo convention — `path_tools/` → `package pathtools`, `time_tools/` → `package timetools`. Underscores are stripped; the Go package name must be one lowercase word.

## Architectural Patterns

### Pattern 1: Ordered Detector Registry with `(T, bool)` contract

**What:** A private, ordered slice of detector functions, each returning `(ProjectData, bool)` — matched or not. The registry iterates in order, first match wins, then falls through to `LanguageUnknown` with no error.

**When to use:** Any project-type detection with a fixed, known set of languages and a single-result API. This is the ecosystem consensus: GitHub Linguist's ordered strategy pipeline, the Rust `project-detect` crate's ordered marker checks, `opendray/projectscan`'s `[]detectorFunc{detectGo, detectNode, ...}` each returning `(Stack, bool)`, and `vership`'s ordered if-chain all converge on it.

**Trade-offs:** Ordering is a global policy decision (it encodes priority); collisions between languages are resolved silently by position rather than by confidence scoring. Acceptable because manifest-marker collisions are rare and the result is stable/deterministic. Cheaper and simpler than a scoring system; scoring adds no value when markers are near-unique.

**Example:**
```go
// registry.go
type detectorFunc func(folder string, entries []os.DirEntry) (ProjectData, bool)

// Order IS the priority policy. More specific markers first.
var detectors = []detectorFunc{
    detectGo,       // go.mod
    detectRust,     // Cargo.toml
    detectDotnet,   // *.sln, *.slnx, *.csproj
    detectJava,     // pom.xml, build.gradle(.kts)
    detectPython,   // pyproject.toml, setup.py
    detectJavaScript, // package.json
    detectPHP,      // composer.json
}

func Probe(folder string) (ProjectData, error) {
    asserted, err := filepath.Abs(folder)
    if err != nil {
        return ProjectData{}, err
    }
    entries, err := os.ReadDir(asserted) // read ONCE
    if err != nil {
        return ProjectData{}, err // only I/O failure is an error
    }
    entries = filterIgnored(entries)
    if !hasAtLeastOneFile(entries) {
        return unknownProject(asserted, entries), nil // empty → Unknown, no error
    }
    for _, detect := range detectors {
        if data, ok := detect(asserted, entries); ok {
            return data, nil
        }
    }
    return unknownProject(asserted, entries), nil // no match → Unknown, no error
}
```

### Pattern 2: Single Directory Read, Entries Shared with Detectors

**What:** The registry calls `os.ReadDir` exactly once and passes the resulting `[]os.DirEntry` into every detector. Detectors check markers against the entries slice instead of re-listing the directory (or calling `filepath.Glob`, which re-reads).

**When to use:** Always — it is a strict improvement over the sample, where each `files.FindFirst` call re-listed the directory (7 detectors × 1+ listing each). It also gives the registry the single source of truth for the "is the folder non-empty" check and the ignored-dir filter.

**Trade-offs:** Detectors are limited to the root-level listing (no lazy re-listing). That is exactly the behavior we want — see Pattern 4. If a future detector genuinely needs subdir scans (e.g., `src/` markers), it can call `os.ReadDir` itself; the contract doesn't forbid it, but the root-first design makes such needs rare.

**Example:**
```go
func detectGo(folder string, entries []os.DirEntry) (ProjectData, bool) {
    if !hasFile(entries, "go.mod") {
        return ProjectData{}, false
    }
    name, goVer := readGoMod(filepath.Join(folder, "go.mod")) // text scan
    return ProjectData{
        Folder:   folder,
        Language: LanguageGo,
        Name:     firstNonEmpty(name, filepath.Base(folder)),
        Version:  goVer, // go directive; "" if absent
    }, true
}
```

### Pattern 3: Per-Language Self-Contained Detector Files + Shared Minimal Parsers

**What:** Each language owns its markers, manifest structs, and parser functions inside its own `detector_<lang>.go`. Only genuinely cross-language machinery (TOML subset parser, README extraction, ignore list) lives in shared files.

**When to use:** Always for this package. Keeps every language's full behavior (claiming + parsing + fallbacks) reviewable in one file and makes language addition a copy-modify-extend exercise.

**Trade-offs:** Slight duplication of shape (`ProjectData{...}` literals repeated per file) — acceptable; it keeps each file independently readable. Do not over-abstract into a shared `ManifestParser` interface; the manifest formats differ too much (JSON vs XML vs TOML vs free text) for an interface to earn its complexity at 7 languages.

**Example (detector_python.go):**
```go
func detectPython(folder string, entries []os.DirEntry) (ProjectData, bool) {
    pyproject := hasFile(entries, "pyproject.toml")
    if !pyproject && !hasFile(entries, "setup.py") {
        return ProjectData{}, false
    }
    name, version, description := "", "", ""
    if pyproject {
        name, version, description = parsePyProject(filepath.Join(folder, "pyproject.toml"))
    }
    return ProjectData{
        Folder:      folder,
        Language:    LanguagePython,
        Name:        firstNonEmpty(name, filepath.Base(folder)),
        Version:     version,
        Description: firstNonEmpty(description, readmeDescription(folder, entries)),
    }, true
}
```

### Pattern 4: Root-Scoped Marker Detection — Order Encodes Priority

**What:** Every detector claims a folder only from markers present at the probed folder's root. No recursive or subdirectory probing for markers. Registry order is the conflict-resolution mechanism (specific markers before generic ones).

**When to use:** Always. Real-world evidence: `vership` resolves "repo has both go.mod and a tooling package.json" by ordering (private package.json is checked last so go.mod wins); the Rust `project-detect` crate checks language-specific files first, generic build systems last. Linguist's strategy pipeline is the same idea at file level.

**Trade-offs:** A nested project (e.g., a Go repo with `frontend/package.json`) reports as Go only — the nested project is out of scope for a single-folder probe. This is the correct behavior for the milestone ("folder's project"), and multi-project discovery is an explicit future feature (see Scaling).

**Example:** The sample's Dotnet detector searched one level of subdirectories for `*.csproj` — do not carry this over. A `.sln`/`.slnx`/`.csproj` at root is the .NET marker; a `.csproj` in a subdir belongs to a nested project and must not claim the parent folder.

### Pattern 5 (Rejected): Plugin-Style Registration

**What:** Exported `Register(lang string, d DetectorFunc)` with a mutable global registry (IntelliJ's `FileTypeDetector` extension points; `projectdetect`'s `LoadFromFile` YAML custom types).

**Why rejected here:** (1) The milestone's extensibility need is *internal* — adding Ruby/Terraform later is a source change, not a consumer need. (2) A mutable global registry introduces init-order concerns, duplicate-registration panics, and test isolation hazards for zero current benefit. (3) This monorepo's philosophy is minimal, stable APIs; an exported registration surface commits the API to compatibility forever. (4) Every real-world single-binary detector tool (Linguist, vership, project-detect, opendray) uses a private ordered list; plugin registration appears only in IDE platforms (IntelliJ, VS Code) that must host third-party extensions.

**If consumers ever need custom languages:** revisit as a separate milestone with an explicit `Register` API + duplicate detection; the `detectorFunc` contract makes this additive without breaking the built-ins.

## Data Model Design

### `ProjectData` — one struct, five fields

```go
type ProjectData struct {
    Folder      string // absolute path of the probed folder
    Language    string // one of the Language* constants; LanguageUnknown if nothing matched
    Name        string // manifest name → fallback folder base name (path.Base)
    Version     string // manifest version; "" when the format has none
    Description string // manifest description → fallback README first paragraph
}
```

| Field | Primary source | Fallback chain | Notes |
|-------|----------------|----------------|-------|
| `Folder` | absolute probed path | — | never empty on success |
| `Language` | winning detector | `LanguageUnknown` | never empty; no error for Unknown |
| `Name` | manifest name | `path.Base(folder)` | dispat scanner confirms "stated name > root manifest > folder base" |
| `Version` | manifest version | `""` | go.mod has no project version — the `go` directive is used (validated by opendray reading the `go` line as version) |
| `Description` | manifest description | README first paragraph → `""` | new field; manifest description wins when present |

### Per-language field sources

| Language | Marker(s) | Name | Version | Description |
|----------|-----------|------|---------|-------------|
| Go | `go.mod` | `module` line | `go` directive | none in manifest → README fallback |
| Python | `pyproject.toml`, `setup.py` | `[project].name` / `[tool.poetry].name` | `[project].version` / `[tool.poetry].version` | `[project].description` |
| JS/TS | `package.json` (+ `tsconfig.json`) | `name` | `version` | `description` |
| C#/.NET | `*.sln`, `*.slnx`, `*.csproj` (root) | sln project name / `<AssemblyName>`/`<PackageId>` / csproj basename | `<Version>` | `<Description>` |
| Rust | `Cargo.toml` | `[package].name` | `[package].version` | `[package].description` |
| Java/Kotlin | `pom.xml`, `build.gradle`, `build.gradle.kts` | `<artifactId>` / gradle `name`/`group` lines | `<version>` / gradle `version` line | `<description>` / gradle `description` line |
| PHP | `composer.json` | `name` | `version` | `description` |

Notes:
- **Language constants** replace the sample's `*ProjectType` constants: `LanguageGo = "go"`, `LanguagePython = "python"`, `LanguageJavaScript = "javascript"`, `LanguageTypeScript = "typescript"` (refinement when `tsconfig.json` present — optional nuance, decide in roadmap), `LanguageDotnet = "dotnet"`, `LanguageRust = "rust"`, `LanguageJava = "java"`, `LanguageKotlin = "kotlin"` (refinement when `build.gradle.kts` present — optional), `LanguagePHP = "php"`, `LanguageUnknown = "unknown"`. Keep the values lowercase single words for downstream pattern-matching.
- **`Type` is renamed to `Language`** — new package, no compatibility burden with the deprecated sample; do not keep a `Type` alias.
- **Version semantics:** best-effort string, never parsed/normalized (no semver validation — `hashicorp/go-version` is a cache-package dependency and out of scope here; a raw string is the right contract for a probe).
- **Description:** manifest `description` field wins; only when absent or empty fall back to README first paragraph. This honors the milestone's "manifest+README fallback" chain and keeps README parsing out of the hot path.

### README first-paragraph extraction (readme.go)

1. Look for `README.md` (then `README`) among root entries — case-sensitive, conventional names only; do not glob.
2. Scan lines: skip the first line if it is a heading (`# `, `## `, or an H1 setext `===` underline).
3. Skip blank lines and lines starting with badges/images (`![]`, `<img`, `https://` shield URLs — heuristic, keep minimal).
4. Take the first non-empty paragraph (lines until a blank line). Return as-is, no truncation (paragraphs are short); trim whitespace.

Confidence: MEDIUM (ecosystem convention; not verified against a specific primary source — verify during implementation with real-world README fixtures).

## Data Flow

### Probe flow

```
Probe("/path/to/folder")
    ↓
filepath.Abs → os.ReadDir ── error? ──→ return err        (only I/O failure errors)
    ↓
filter ignored dirs (node_modules, vendor, target, ...)
    ↓
at least one non-dir entry? ── no ──→ ProjectData{Unknown, base(folder)}   (no error)
    ↓ yes
for each detector in [go, rust, dotnet, java, python, js, php]:
    detect(folder, entries) ── ok? ──→ ProjectData{Language, ...}          (first match wins)
    ↓ no match
ProjectData{Unknown, Name: base(folder)}                                    (no error)
```

### Per-detector flow (manifest → fallbacks)

```
marker found in entries
    ↓
parse manifest (json | xml | toml-subset | text scan)
    ↓ malformed / missing field? → fall back per field:
Name:        manifest name  →  path.Base(folder)
Version:     manifest value →  ""
Description: manifest value →  README first paragraph →  ""
```

### Key data flows

1. **Detection flow:** one ReadDir feeds all detectors; each detector claims via root markers only; ordering resolves multi-manifest folders.
2. **Field fallback flow:** every field degrades independently (name → folder base; description → README; version → empty). A folder with a valid marker but a malformed manifest still yields a usable `ProjectData` with `Language` set — never an error.

## Collect-All vs First-Match: Does the Unknown-without-error Contract Change the Registry?

**No — and the reason is precise.** The Unknown-without-error contract changes exactly one thing: the registry's *fall-through* path. The sample's `DetectProject` both returns `UnknownProjectType` data AND returns an error — a contradiction. The fix is to make the loop fall through to the Unknown default instead of returning an error:

- The detector contract changes from `(ProjectData, error)` (error-as-control-flow: `os.ErrNotExist` means "not me") to `(ProjectData, bool)` — validated by opendray and the Rust `project-detect` crate.
- The registry loop stays first-match-wins. **Collecting ALL matches is not required by the contract**, and the milestone's API is a single report per folder. Multi-match (monorepo roots with several manifests) is genuinely ambiguous; the ecosystem answers are "first-match with careful ordering" (vership, project-detect) or "return all matches" (projectdetect, opendray — both of which expose multi-result APIs).
- If a future milestone wants `ProbeAll(folder) ([]ProjectData, error)` (multi-language monorepo roots), the `(ProjectData, bool)` contract supports it **without touching any detector**: the registry loop collects instead of returning on first match. That is the entire extensibility surface needed. Do not build it now.

**Recommendation:** first-match, with ordering as the documented priority policy, and a doc comment on the registry stating "order is intentional — specific markers before generic, stable across releases."

## Extensibility Path: Adding Ruby / Terraform Later

| Step | What | Touches |
|------|------|---------|
| 1 | Create `detector_ruby.go`: markers `Gemfile`, `*.gemspec` (root); name from gemspec `name`/`Gemfile` parse (text scan — gemspec is Ruby code; regex `name\s*=\s*["']`); version from gemspec `version`; description from gemspec `description` | 1 new file |
| 2 | Add `detectRuby` to the registry slice with a comment on ordering intent | registry.go, 1 line |
| 3 | Create `detector_terraform.go`: markers `*.tf` (root glob over entries — no subdir walk); language `terraform`; no name/version/description in manifests → all fields fall back (name → folder base, others empty) | 1 new file |
| 4 | `testdata/ruby/`, `testdata/terraform/` fixtures + table-driven tests | testdata + 1 test file |

**Design rules that keep this cheap:**
- Detectors never import each other; the registry is the only coupling point.
- `toml.go` (if a future language uses TOML, e.g. Terraform's `.terraform.lock.hcl` is HCL — NOT TOML — so note: Terraform needs no TOML; a future `cargo`-like format would reuse it).
- Ordering rule of thumb for placement: language-specific build manifests before generic ones; new languages are appended near siblings (Ruby next to Python; Terraform at the end as a file-glob detector, since `*.tf` is a weaker signal than a named manifest).
- Every new detector must be testable in isolation: the `(folder, entries)` signature means tests can construct `os.DirEntry`-equivalent fixtures via a real `t.TempDir()` — table-driven with `testdata/` subdirs.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 7 languages, single folder | Current design: one ReadDir + linear scan of marker names. Sub-microsecond per detector beyond the initial listing. |
| 15-20 languages | Still fine. Detectors are O(entries) or O(1); no per-language I/O beyond the manifest read of the *winner*. Registry grows but remains linear. |
| Recursive discovery (find all projects under a tree) | Not in scope for v1.7. When needed, it is a *caller-side* concern (walk + Probe per dir, or a future `Find(ctx, root, opts)` API like `projectdetect`'s). Do not build walking into the probe. |
| Multi-match roots (monorepo) | Deferred; `ProbeAll` is additive later (see Collect-All section). |

### Scaling Priorities

1. **First bottleneck (never in practice):** a folder with tens of thousands of entries — the single ReadDir dominates. Mitigation if ever needed: skip the `hasAtLeastOneFile` scan by checking for any non-dir during the same loop that filters ignored dirs (already the case in the design above — one pass does both).
2. **Second: nothing.** The package performs one directory listing + one manifest read per probe. No caching, no concurrency needed; do not add them.

## Anti-Patterns

### Anti-Pattern 1: Error-as-Control-Flow Detector Contract

**What people do:** The sample's `DetectorFunc(folder) (ProjectData, error)` returns `os.ErrNotExist` to mean "not my language", and the registry tests `err == nil` to mean "matched".

**Why it's wrong:** Errors are for exceptional conditions, not normal control flow. Every real-world successor (opendray `(Stack, bool)`, IntelliJ `FileTypeDetector` returning `null`, Rust `Option<ProjectKind>`) uses a match/no-match signal. It also made the sample's "Unknown without error" impossible to express cleanly.

**Do this instead:** `(ProjectData, bool)`; errors are reserved for I/O failures in `Probe` itself (folder missing/unreadable).

### Anti-Pattern 2: Subdirectory Probing for Markers

**What people do:** The sample's Dotnet detector searched one level down for `*.csproj` (`files.FindFirstInSubdirs`), and the JS detector pattern (`"*.py"`-style) matched loose source files.

**Why it's wrong:** A folder containing any nested project gets claimed by the wrong language (a JS repo with a `vendor/*.csproj` becomes "dotnet"). False positives compound with first-match-wins.

**Do this instead:** Root-scoped markers only (Pattern 4). The probe answers "what is *this* folder", not "what is inside it".

### Anti-Pattern 3: Naive Regex TOML Parsing

**What people do:** `regexp.MustCompile(`^name\s*=\s*"(.+)"`)` over the whole file — the mex/scanner TypeScript tool does exactly this.

**Why it's wrong:** `pyproject.toml` keeps `name`/`version` under `[project]` (or `[tool.poetry]`), `Cargo.toml` under `[package]`. An unanchored regex picks up `dependencies` entries, `[tool.ruff]` config, or the wrong section's `name`. Also, `^name` fails if keys are indented.

**Do this instead:** Section-aware TOML subset scanner (track current `[section]`, only read `key = "value"` under allowed sections). ~80 lines, fully unit-testable; handles comments, quoted values, and basic escapes. See `toml.go`.

### Anti-Pattern 4: Plugin-Style Global Registration

**What people do:** Export `Register(name string, detector DetectorFunc)` into a package-level mutable slice.

**Why it's wrong:** Global mutable state: duplicate-registration panics, init-order surprises, test pollution (the AGENTS.md note about `collectFuncs` race in `mid/` is the same class of hazard). It also freezes the API surface for a need (external custom languages) that does not exist yet.

**Do this instead:** Private ordered slice (Pattern 1); revisit plugin registration only if a consumer need emerges (Pattern 5).

### Anti-Pattern 5: Erroring on Unknown / Empty Folders

**What people do:** The sample returns `errors.New("folder is empty")` and `errors.New("could not detect project type")` alongside Unknown data.

**Why it's wrong:** The milestone contract is best-effort, never failing. An empty folder or a folder with unrecognized content is a *valid answer*, not a failure — consumers would have to special-case errors to display the Unknown state.

**Do this instead:** `LanguageUnknown` + folder-base `Name`, `nil` error. The only errors: `folder` doesn't exist or `os.ReadDir` fails.

### Anti-Pattern 6: External Dependencies Leaking into the New Package

**What people do:** Copying the sample verbatim, which imports `github.com/guionardo/gs-dev/pkg/tools/files` and `github.com/BurntSushi/toml` (both unresolved in the module — the sample doesn't even compile today).

**Why it's wrong:** The milestone mandates stdlib-only. TOML has no stdlib parser (verified against Go 1.26 release notes), so the subset scanner replaces BurntSushi; `files.AssertDirectory`/`files.FindFirst` become `filepath.Abs` + `os.ReadDir` + a small `hasFile(entries, name)` helper.

**Do this instead:** A `#nosec G304` comment (path comes from caller-supplied folder, read-only) on manifest `os.Open` calls, mirroring the sample's own annotation; stdlib-only imports, enforced by CI's dependency checks if any (otherwise by review).

## Integration Points

### Public API Surface (new package, no changes to existing packages)

```go
package projectprobe // directory: project_probe/ (repo convention: underscores stripped)

type ProjectData struct {
    Folder      string
    Language    string
    Name        string
    Version     string
    Description string
}

const (
    LanguageUnknown    = "unknown"
    LanguageGo         = "go"
    LanguagePython     = "python"
    LanguageJavaScript = "javascript"
    LanguageTypeScript = "typescript" // only if tsconfig refinement is adopted
    LanguageDotnet     = "dotnet"
    LanguageRust       = "rust"
    LanguageJava       = "java"
    LanguageKotlin     = "kotlin"     // only if gradle.kts refinement is adopted
    LanguagePHP        = "php"
)

// Probe reports the project described by the folder's contents.
// It never fails for unknown content: unknown/empty folders return
// LanguageUnknown with a nil error. The only errors are I/O failures
// (missing folder, unreadable directory).
func Probe(folder string) (ProjectData, error)
```

**Deliberately NOT exported (v1.7):** `Register`, `Detector` interface, `ProbeAll`, `Find` (recursive discovery), version normalization helpers. Each is additive later without breaking this surface.

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `Probe` ↔ registry | direct call | registry is unexported; tests exercise it via `Probe` plus per-detector unit tests |
| registry ↔ detectors | `detect(folder, entries) (ProjectData, bool)` | the only contract between layers; detectors never call each other |
| detectors ↔ `toml.go`/`readme.go` | direct calls | shared helpers are unexported package-level funcs |
| `project_probe` ↔ rest of monorepo | none | zero imports from sibling packages (stdlib-only); consumers import `github.com/guionardo/go/project_probe` |

### Repo Integration Points

| Integration | Type | What |
|-------------|------|------|
| Root `README.md` package index | modify | add `project_probe` row (repo convention: every package listed) |
| `doc.go` | new | package documentation, following the 21-package convention |
| Coverage CI (`.testcoverage-quick.yml`, Makefile) | modify | new package must meet thresholds (packages ≥80%, files ≥70%, total ≥75% for quick; full suite 95%+); testdata-driven table tests must cover fallback branches |
| Deprecated `project_detector/` | untouched | leave as-is; it does not compile in the module and is superseded |
| `cmd/` | none | only `example-updater` exists; no CLI consumes detection today |

## Build Order Recommendation

Dependency-driven, each step leaves the package green under `make coverage-quick`:

1. **Skeleton + Unknown contract:** `doc.go`, `probe.go` (`ProjectData`, constants, `Probe`), `registry.go` with `(ProjectData, bool)` loop, `ignore.go`, empty-folder → Unknown. Tests: Probe on empty dir, missing dir (error), unknown dir (Unknown, nil error). *This establishes the never-failing contract before any language exists.*
2. **Shared helpers:** `toml.go` section-aware subset parser + `readme.go` first-paragraph + table tests with `testdata/` READMEs. *Both helpers are independently testable and unblock two language families each.*
3. **Text-scan detectors (easiest, no parsers):** Go (`go.mod`), JavaScript/PHP (`encoding/json`), .NET (`.sln`/`.slnx` text + `.csproj` XML). *JSON/XML are stdlib; validate the fallback chains early.*
4. **XML + TOML detectors:** Java (`pom.xml`), .NET `<Version>`/`<Description>` refinement, Python + Rust (TOML). *TOML subset parser gets its first real consumers; fixture-driven.*
5. **Polish + docs:** doc.go prose, root README index row, fallback-branch coverage pass, `go vet`/linting (G304 annotations on manifest reads).

**Why this order:** the Unknown-without-error contract is the riskiest design bet, so it lands first; shared parsers before their consumers avoids two languages re-implementing the same helper; easiest parsers first de-risks the detector contract shape before the TOML complexity arrives.

## Sources

Primary sources (read via search highlights; confidence per finding):

- Go 1.26 release notes (go.dev/doc/go1.26, Feb 2026) — **no `encoding/toml` in stdlib** (new: crypto/hpke, simd/archsimd, runtime/secret). HIGH (official).
- golang/go issues #60791 (common struct tag), #68361 (name alias) — TOML-tag proposals closed/frozen, no stdlib TOML adoption. HIGH (primary).
- GitHub Linguist `lib/linguist.rb` — ordered `STRATEGIES` pipeline, candidate narrowing; `heuristics.yml` ordered first-match rules. HIGH (primary source code).
- JetBrains intellij-community `FileTypeRegistry.java` — plugin extension points; `FileTypeDetector` returns null, "must refrain from throwing exceptions". HIGH (primary source code).
- `richardwooding/projectdetect` (Go) — indicator-based registry, returns ALL matches, YAML custom types. MEDIUM (primary, single project).
- Rust `project-detect` crate (`docs.rs` source) — ordered marker checks, language-specific first, generic build systems later, `detect_nearest` walks up. HIGH (primary source code).
- `vership` `project/detect.rs` — ordering as ambiguity resolution; private package.json treated as tooling (checked last); maturin compound type before plain Rust. HIGH (primary source code, directly relevant).
- `opendray/projectscan` — `[]detectorFunc` returning `(Stack, bool)`; go.mod `go` line read as version; ignored-dir map. HIGH (primary source code — validates both the bool contract and the go-directive-as-version choice).
- `dispat` scanner docs — ecosystem-neutral `Manifest{Path, Name, Version, ...}`; name preference "stated > root > nested"; workspace-dir skip lists. MEDIUM.
- mex `scanner/manifest.ts` — naive `^name\s*=` TOML regex (the anti-pattern), ordered manifest list. MEDIUM.
- `BurntSushi/toml` v1.6 / `pelletier/go-toml` v2 — ecosystem TOML status. HIGH (registry pages).

Confidence: architecture shape HIGH (converging primary sources across 6+ implementations); README-fallback specifics MEDIUM (convention, not verified against a dedicated source — flag for phase research).

---
*Architecture research for: v1.7 project_probe package*
*Researched: 2026-09-28*
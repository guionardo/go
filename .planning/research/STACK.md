# Stack Research: project_probe — Best-Effort Project Detection

**Domain:** Best-effort project detection library (folder → language/name/version/description) inside a stdlib-first Go utility monorepo
**Project:** `github.com/guionardo/go` — milestone v1.7 `project_probe`
**Researched:** 2026-09-28
**Confidence:** HIGH

> This document is the milestone-scoped stack research for v1.7. It supersedes the repo-level
> STACK.md (2026-07-21, kept in git history) whose general guidance (Go 1.26 stdlib-first,
> minimal deps, testify, coverage gates) remains valid and is assumed here.

## Executive Summary

`project_probe` needs **zero new runtime dependencies**. Every manifest format in scope
(go.mod, package.json, composer.json, pyproject.toml, Cargo.toml, .csproj/.slnx/pom.xml, .sln,
build.gradle) is parseable with Go 1.26 stdlib — `os`, `bufio`, `strings`, `encoding/json`,
`encoding/xml`, `path/filepath`, and a small `regexp` usage — **plus one unexported
hand-rolled TOML-subset reader** (~80 lines) for `pyproject.toml` and `Cargo.toml`.

Two verified facts drive the key decisions:

1. **Go stdlib has no TOML parser** — verified locally against Go 1.27.0 (`go doc encoding/toml`
   fails; nothing added in 1.24/1.25/1.26). `pyproject.toml`/`Cargo.toml` are **TOML, not YAML** —
   so the already-present `gopkg.in/yaml.v3` (used by `config/`) is useless here; the "is a YAML
   dep justified?" question is a category error. The choice is: external TOML dep vs. hand-rolled
   subset. For exactly 3 string fields (`name`, `version`, `description`) under 3 known tables
   (`[project]`, `[tool.poetry]`, `[package]`), a subset reader is correct; a full TOML
   dependency becomes justified only at the deferred framework-detection milestone (dependencies
   parsing) — defer the dep, not the field extraction.
2. **The mature Go ecosystem pattern is manifest-first ordered cascades, not content-based
   language detection.** GitHub Linguist's strategy cascade (Modeline→Filename→Shebang→Extension→
   XML→Manpage→Heuristics→Classifier, first strategy to yield exactly one candidate wins) and
   Snyk's ordered `DETECTABLE_FILES` basename list (first existing manifest wins) both converge on
   the same design: ordered, deterministic, first-match, unknown = typed fallback value **never
   an error**. `go-enry` (the Go linguist port) is per-file content classification and knows
   nothing about manifests — explicitly not the right abstraction.

The deprecated `project_detector/` sample (ordered per-language detectors, first-match wins,
`Unknown` fallback) is structurally the right pattern but depends on the unavailable `gs-dev`
module and BurntSushi/toml, and **does not compile** (verified: 3 build errors). It must be
removed in this milestone to restore `go build ./...` / lint / coverage CI cleanliness — its
detector logic is the design reference, its dependencies are not.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.26.4 (go.mod) | Language + toolchain | Repo constraint; local toolchain 1.27.0. All stdlib features used are ≥1.24 (`strings.SplitSeq`), safe on both. |
| `os` | stdlib | `ReadDir`, `ReadFile`, `Stat` | Folder listing, manifest reads, folder-existence check for the error path. No `ioutil`. |
| `bufio` | stdlib | `Scanner` for line-oriented reads | go.mod (`module`/`go` lines), README first-paragraph extraction, .sln `Project(...)` lines. Bounded token size via `Scanner.Buffer`. |
| `strings` | stdlib | `CutPrefix`, `Fields`, `TrimSpace`, `TrimSuffix`, `SplitSeq` | go.mod directives, .sln parsing, gradle `version = 'x'`, README normalization. `SplitSeq` iterator form is the Go 1.24+ idiom (already used in repo). |
| `encoding/json` | stdlib | `package.json`, `composer.json` | Typed struct decode for `name`/`version`/`description`. Both files are strict JSON — no JSONC concern. |
| `encoding/xml` | stdlib | `.csproj`, `.slnx`, `pom.xml` | Struct decode by local element names; tolerant of the `xmlns` namespaces found in real `.csproj`/`pom.xml` files (namespace-agnostic local-name matching). |
| `path/filepath` | stdlib | `Base`, `Glob`, `WalkDir`, `Join` | Manifest discovery (`*.csproj`, `*.sln`), folder-name fallback for Name, bounded 1-level subdir scan for .NET projects (skip `.git`, `node_modules`, `vendor`, …). |
| `regexp` | stdlib | Minimal, compiled-once patterns | TOML value extraction edge cases (single-quoted strings, inline comments after values), gradle version lines. Prefer `strings` first; use regexp only where line-splitting is insufficient. |
| `errors` | stdlib | Sentinel errors | `ErrNotDir`-style sentinel for the only real error path (folder missing/unreadable). Unknown detection is **not** an error. |

### Supporting Libraries (runtime: NONE)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| *(none)* | — | — | All manifest parsing is stdlib + unexported TOML subset reader. Zero new entries in `go.mod`. |

The only new code-level "library" is an **unexported `toml.go` subset reader** (package-private,
~80 lines, table tracked + `key = "value"` string extraction). It is deliberately NOT a public
API and NOT a full TOML parser. See "Stack Patterns by Variant" for its exact contract.

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `make coverage-quick` | Coverage gate before every commit | Runs `go test ./...` + `go-test-coverage` — new package is included automatically. Thresholds (`.testcoverage-quick.yml`): package ≥80, file ≥70, total ≥75. |
| `.testcoverage-quick.yml` | Per-package threshold config | **No override needed** — best-effort detection is highly testable via table-driven tests + `testdata/` fixture folders (`testdata` is excluded from coverage automatically). |
| `golangci-lint` (gosec) | Security linting | Manifest files are opened via dynamic paths (`os.Open` of discovered files) → gosec `G304` fires. Repo precedent exists: `#nosec G304 -- path comes from local discovery in the inspected folder.` annotations in the sample; replicate. |
| `doc.go` convention | Package docs | Every repo package has `doc.go` with `// Package project_probe provides ...` + feature list + example. Required for lint + README consistency. |
| Go doc comments | All exported symbols | Repo rule ("Go doc comments for all exported symbols"); enforced by lint. |
| README.md package index | Public surface | v1.5 restructured README with a package index table — add `project_probe` row in this milestone. |
| `commitlint` | Conventional commits | `feat(project_probe): ...` scope. |

## Installation

```bash
# No new runtime dependencies.
# Verify the module stays clean after implementation:
go mod tidy && git diff --stat go.mod go.sum   # expect: no changes

# Quality gates (existing targets, no changes needed):
make lint
make coverage-quick
```

## The TOML Decision (explicit)

`pyproject.toml` (Python, PEP 621) and `Cargo.toml` (Rust) are **TOML**. Go stdlib has no TOML
(verified on Go 1.27.0). The candidates:

| Option | Verdict | Rationale |
|--------|---------|-----------|
| **Hand-rolled subset reader** (unexported) | **RECOMMENDED for v1.7** | Only 3 string fields under 3 known tables. Best-effort semantics make a miss acceptable (falls back to folder name / empty). ~80 lines, fully testable, zero deps, zero supply-chain surface. |
| `BurntSushi/toml` v1.6.0 | Defer | Reflection API like `encoding/json`, TOML v1.1.0, ~5k★, 38k importers; what Trivy/go-dep-parser use for `pyproject.toml`. **Add it at the framework-detection milestone** when `[project.dependencies]` / `[tool.poetry.dependencies]` / `[package.dependencies]` parsing makes a full parser genuinely valuable. |
| `pelletier/go-toml/v2` v2.4.x | Defer | TOML v1.1.0, stdlib-like API, 5-8× faster unmarshal than BurntSushi; the modern alternative if the dep is ever added — same deferral boundary. |
| `gopkg.in/yaml.v3` | **Never for this** | Already a direct dep (via `config/`) but YAML ≠ TOML — cannot parse `pyproject.toml`/`Cargo.toml`. Category error; no amount of "it's already in go.mod" makes it work. |

**Deferral boundary (write it in the code):** the day `project_probe` needs dependency lists
(framework detection), add `pelletier/go-toml/v2` (or BurntSushi) and delete `toml.go`. Until
then the subset reader is the smaller, cheaper, testable truth.

## The go.mod Parsing Decision (explicit)

| Option | Verdict | Rationale |
|--------|---------|-----------|
| **Line-scan with `bufio` + `strings.CutPrefix`** | **RECOMMENDED for v1.7** | `module <path>` (optionally quoted) and `go <version>` are simple, stable line formats. 10 lines total, trivially tested (quote + comment cases). Matches "minimal external dependencies" and the repo's own `config/` philosophy. |
| `golang.org/x/mod/modfile` (v0.38.0 already indirect in module graph) | Documented upgrade path | The Go team's canonical go.mod parser, used by the toolchain itself; 1,706 importers. `modfile.ModulePath(data)` is a tolerant extractor (returns `""` if absent) — a perfect best-effort primitive. **Cost:** promotes an indirect dep to direct. Revisit if go.work handling or toolchain-exact semantics are ever needed (not in v1.7 scope). |

## Per-Language Manifest → Reader Mapping (v1.7)

| Language | Manifest(s) | Reader | Name | Version | Description |
|----------|-------------|--------|------|---------|-------------|
| Go | `go.mod` | `bufio` + `strings` (line scan) | `module` line | `go` directive | none → README fallback |
| Python | `pyproject.toml` | TOML subset reader | `[project].name` → `[tool.poetry].name` | `[project].version` → `[tool.poetry].version` | `[project].description` → README fallback |
| JS/TS | `package.json` | `encoding/json` (struct) | `name` | `version` | `description` → README fallback |
| C#/.NET | `*.sln` / `*.slnx` / `*.csproj` (1-level subdir scan) | `strings` (.sln `Project("...") = "Name", ...`), `encoding/xml` (.slnx `<Project Path=.../>`, .csproj `<Version>`/`<Description>`) | sln first project name, else csproj basename | csproj `<Version>` (empty if absent) | csproj `<Description>` → README fallback |
| Rust | `Cargo.toml` | TOML subset reader | `[package].name` | `[package].version` | `[package].description` → README fallback |
| Java/Kotlin | `pom.xml`, `build.gradle(.kts)` | `encoding/xml` (pom), line scan (gradle `version = 'x'` / `version = "x"`) | pom `<name>` → gradle `rootProject.name`/archiveBaseName → folder | pom direct-child `<version>` (skip `<parent><version>`) → gradle `version =` | pom `<description>` → README fallback |
| PHP | `composer.json` | `encoding/json` (struct) | `name` | `version` (usually absent — empty is correct) | `description` → README fallback |
| *any* | README fallback (cross-cutting) | `os.ReadFile` + `bufio`/`strings` | `path/filepath.Base(folder)` when manifest lacks name | empty when manifest lacks version | First non-empty paragraph of `README.md` (also `README`, `readme.md`), trimmed; capped length |

**Detection precedence (documented, deterministic):** ordered cascade Go → Python → C#/.NET →
JS/TS → Rust → Java/Kotlin → PHP (same relative order as the sample; Go first because `go.mod`
is unambiguous and this is a Go monorepo). First manifest match wins; unknown folder →
`Language: Unknown`, `Name: folder base`, no error. Framework detection (dependencies-based) is
explicitly out of scope.

## Mature API Shape (reference for ARCHITECTURE.md)

| Ecosystem reference | API pattern | What to copy |
|---------------------|-------------|--------------|
| GitHub Linguist (source-verified) | `STRATEGIES = [Modeline, Filename, Shebang, Extension, XML, Manpage, Heuristics, Classifier]` — cascade narrows candidates; first strategy yielding exactly one wins; `nil` if unresolved | Ordered cascade with first-match termination; unknown is a value, never an error |
| Snyk detector | Ordered `DETECTABLE_FILES` basename list, first existing file wins | Manifest basename → language mapping table |
| `aquasecurity/go-dep-parser` (→ Trivy) | One `Parse(r io.Reader)` per manifest format, keyed by filename | Per-manifest parse funcs; do NOT take the dependency (absorbed into Trivy, standalone archived) |
| `go-enry` | `GetLanguage(filename, content) string` — returns `""` for unknown, never errors; `IsVendor`/`IsBinary` helpers | The "return a fallback value, not an error" contract; the vendor-dir skip hygiene (`.git`, `node_modules`, `vendor`) |
| `x/mod/modfile` | `ModulePath(mod []byte) string` — tolerant, `""` if absent | Same tolerant-extraction contract inside our line readers |

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Hand-rolled TOML subset (unexported) | `BurntSushi/toml` v1.6.0 | Framework-detection milestone (dependency parsing) — full parser needed |
| Hand-rolled TOML subset (unexported) | `pelletier/go-toml/v2` v2.4.x | Same deferral boundary; prefer this over BurntSushi if/when added (speed, maintained API) |
| Line-scan go.mod | `golang.org/x/mod/modfile` | If go.work parsing or toolchain-exact semantics are required later |
| Manifest-first ordered cascade | `go-enry` v2 | If per-file language classification (byte-weighted language stats) ever becomes a feature — it is not |
| Manifest-first ordered cascade | GitHub Linguist (Ruby) | Never — wrong language, and content-based, not manifest-based |
| stdlib `encoding/xml` | `antchfx/xmlquery` / `etree` | If XPath-style queries were needed — they are not; local-name struct decode suffices for csproj/slnx/pom |
| stdlib `path/filepath` | `gobwas/glob` / `doublestar` | `filepath.Glob` covers `*.csproj`/`*.sln`; `**` semantics not needed |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `gopkg.in/yaml.v3` for pyproject.toml/Cargo.toml | YAML cannot parse TOML — wrong format entirely, regardless of it already being a dep | TOML subset reader |
| `BurntSushi/toml` / `pelletier/go-toml/v2` in v1.7 | 2 fields × 2 files; violates minimal-deps for no robustness gain under best-effort semantics | Hand-rolled subset; add the dep at framework milestone |
| `go-enry` | Content/extension-based per-file classification with an embedded data package (~MBs, generated from linguist YAML); knows nothing about manifest metadata; some heuristics need oniguruma | Ordered manifest cascade + README fallback |
| `aquasecurity/go-dep-parser` | Absorbed into Trivy ("Moved to the dependency package in Trivy"); dependency/lockfile parsing is out of v1.7 scope | Its architecture (per-format parser keyed by filename) as a design reference |
| `golang.org/x/mod/modfile` in v1.7 | Promotes indirect dep to direct for two trivial line formats | bufio line-scan; documented upgrade path |
| Framework-detection libs (e.g. parsing `dependencies` blocks) | Explicitly deferred to a later milestone per PROJECT.md | Dependencies-based detection later, on top of the same manifest readers |
| `fsnotify`, glob libs, `samber/lo`, any kitchen-sink | No watch/recursive-glob/lo-utility need; contradicts repo constraint | stdlib only |

## Stack Patterns by Variant

**If a manifest is line-oriented (go.mod, .sln, build.gradle):**
- Use `bufio.Scanner` (bounded) + `strings.CutPrefix`/`Fields`; reserve `regexp` for quoted-value
  extraction only. Skip blank lines and `//`/`#` comments.

**If a manifest is TOML (pyproject.toml, Cargo.toml):**
- Use the unexported subset reader: track current table header (`[project]`, `[tool.poetry]`,
  `[package]`); extract `name`/`version`/`description` string values in double **or** single
  quotes; ignore `#` comments and multiline `"""` values (miss → fallback, acceptable).
- Poetry fallback (`[tool.poetry]`) matters: many real pyproject.toml files predate PEP 621.

**If a manifest is JSON (package.json, composer.json):**
- Decode into a small struct (`Name`, `Version`, `Description string`) with `json.Unmarshal`;
  tolerate unknown fields (default behavior) and absent fields (zero values).

**If a manifest is XML (.csproj, .slnx, pom.xml):**
- Decode into a struct with `xml:"..."` tags matching local names; do NOT match namespaces —
  `encoding/xml` matches local names when tags omit namespace, which handles real-world `xmlns`
  attributes. For pom.xml, read only the direct-child `<version>` (skip `<parent><version>`).

**If the folder has no manifest or manifest lacks the field:**
- Language → `Unknown` (no error); Name → `filepath.Base(folder)`; Version → `""`;
  Description → README first-paragraph fallback (shared helper, all languages).

**If the folder does not exist / cannot be listed:**
- Return an error (the ONE error path), per `os.ReadDir`/`os.Stat` failure — this matches the
  sample's `AssertDirectory` behavior without the "could not detect" error.

**If scanning subdirectories (C#/.NET only):**
- Bound depth to 1 and skip the standard ignore set (`.git`, `node_modules`, `vendor`, `.venv`,
  `__pycache__`, `dist`, `build`, `target`, `bin`, `obj`, `.tox`, `.idea`, `.vscode`) — linguist
  vendor-hygiene pattern, prevents false positives in nested toolchains.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| go.mod `go 1.26.4` | local Go 1.27.0 | Both fine; used stdlib features are ≥1.24 (`strings.SplitSeq`, `os.ReadDir`) |
| `strings.CutPrefix` | Go 1.20+ | go.mod directive parsing |
| `strings.SplitSeq` | Go 1.24+ | .sln/README iteration (already used in repo sample) |
| `bufio.Scanner.Buffer` | Go 1.1+ | Cap token size for README paragraphs (e.g. 64KB) |
| `encoding/xml` local-name matching | stdlib | No namespace-tag matching → robust to `xmlns` in csproj/pom |
| New package + `make coverage-quick` | `.testcoverage-quick.yml` | Package ≥80 / file ≥70 / total ≥75 — no override entry needed; `testdata/` fixtures are auto-excluded |
| **Deprecated `project_detector/`** | **BREAKS the module** | Verified: 3 build errors (missing `gs-dev`, `BurntSushi/toml` go.sum entry). `go build ./...`, `go vet`, lint, and coverage all trip on it. **Delete the folder in this milestone** (its logic is superseded by `project_probe`); this is a stack-level prerequisite, not an option. |

## Sources

- `pkg.go.dev/golang.org/x/mod/modfile` (v0.41.0) — modfile API surface; MEDIUM (official docs, cross-checked with module graph)
- Local toolchain Go 1.27.0 — `go doc encoding/toml` fails ⇒ no stdlib TOML; **HIGH** (primary verification)
- `pkg.go.dev/github.com/BurntSushi/toml` v1.5.0/v1.6.0 + repo — TOML 1.1, reflection API, 38k importers; MEDIUM
- `pkg.go.dev/github.com/pelletier/go-toml/v2` v2.4.1 + repo — TOML 1.1, stdlib-like API, benchmark table; MEDIUM
- `github.com/aquasecurity/go-dep-parser` + `pkg.go.dev` directory listing — per-format parser inventory, "Moved to the dependency package in Trivy"; MEDIUM
- `github.com/github-linguist/linguist` `lib/linguist.rb` (source) — STRATEGIES cascade, `detect` returns nil on unresolved; MEDIUM (source-verified)
- `pkg.go.dev/github.com/go-enry/go-enry/v2` + repo — `GetLanguage(filename, content)`, Is* helpers, data generation; MEDIUM
- Repo evidence — `project_detector/` build failure (3 errors), `go.mod` (yaml.v3 direct, x/mod indirect v0.38.0), `.testcoverage-quick.yml`, Makefile, `flow/doc.go` convention; **HIGH** (primary)
- Prior research cache — Snyk `DETECTABLE_FILES` ordered-basename pattern, mise/asdf tree-walk version-file discovery; MEDIUM

---
*Stack research for: v1.7 project_probe milestone*
*Researched: 2026-09-28*

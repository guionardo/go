# Domain Pitfalls: Best-Effort Project Detection & Manifest Parsing

**Domain:** Go utility library — folder probing, manifest parsing, language detection
**Researched:** 2026-09-28
**Confidence:** HIGH — stdlib behaviors verified locally on Go 1.26.4 and 1.27.0; ecosystem conventions cross-checked (linguist/enry, go-toml v2, BurntSushi/toml, golang.org/x/mod/modfile)
**Scope:** Adding the `project_probe` package (v1.7) to `github.com/guionardo/go`, replacing the deprecated `project_detector/` sample. Detects language, name, version, description across Go, Python, JS/TS, C#/.NET, Rust, Java/Kotlin, PHP. Best-effort: `Unknown` type, no error.

---

## Critical Pitfalls

Mistakes that cause rewrites, silently wrong data, or break the "best-effort, never failing" contract.

### Pitfall 1: API Contract Ambiguity — Error vs Unknown

**What goes wrong:** `Probe(folder) (ProjectData, error)` ships without a written contract for when the error is set. Two failure modes get conflated: (a) "folder is not a project" — benign, must yield `Unknown` with nil error; and (b) "cannot read folder" — permission denied, missing directory, genuinely broken. The old `DetectProject` conflated both: it returned `errors.New("folder is empty")` and `errors.New("could not detect project type")` for benign cases, and used `os.ErrNotExist` as a control-flow signal ("not my type") inside detectors.

**Why it happens:** Best-effort tools start with "never fail", then the author discovers real I/O failures and starts returning errors for those — without redefining what the error channel means. The distinction is never written down, and callers end up string-matching error messages.

**How to avoid:**
- Write the contract in `doc.go` on day one: `Probe(folder) (ProjectData, error)` where **error is reserved for hard failures only** (folder unreadable, not a directory). Not-a-project ⇒ `ProjectData{Language: Unknown, Name: filepath.Base(folder)}` with nil error.
- Make detectors error-free internally: return `(data, matched bool)`, never error-as-control-flow (the old `os.ErrNotExist` pattern made "empty folder" a hard error).
- Test the contract explicitly: empty folder → Unknown + nil error; nonexistent folder → error; unreadable folder → error.
- Keep detector ordering in an **ordered slice** (the old `detectors` list), never a map iteration — Go map iteration order is random, which makes results nondeterministic.

**Warning signs:** Callers switching on `errors.Is(err, os.ErrNotExist)`; tests asserting on error strings; the package doc unable to state "never fails" truthfully; any `map` used as the detector registry.

**Phase to address:** Package foundation phase (API design + `doc.go`) — must precede all detector work. The contract shapes every later phase.

---

### Pitfall 2: Naive Line-Scanning Silently Produces Wrong Values (BOM, Quotes, Comments)

**What goes wrong:** Line scanners miss values silently — no error, no fallback, wrong data reported as if true. **Verified locally on Go 1.26.4:**
1. A `go.mod` starting with a UTF-8 BOM makes `strings.CutPrefix(line, "module ")` fail on line 1 — the module name is silently lost (the old `readGoProjectNameFromGoMod` does exactly this).
2. A quoted module directive `module "example.com/foo"` (legal since Go 1.17) extracts the name **with quotes**: `"example.com/foo"`.
3. A trailing `// comment` on the module line keeps the comment inside the value.
The same class of bug hits hand-rolled TOML scanners for `pyproject.toml` and `Cargo.toml` (comments, quoted strings, `version.workspace = true`).

**Why it happens:** Line-based parsing of "just the first directive" looks trivial and the happy path works; the failure modes are invisible because nothing errors.

**How to avoid:**
- Strip the BOM once at read time (`bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})`) in a single shared `readManifest` helper, and test every parser with a BOM-prefixed fixture.
- For `go.mod`: prefer `golang.org/x/mod/modfile.ModulePath` — the Go toolchain's own parser (1,700+ importers), explicitly "tolerant of unrelated problems in the go.mod file", handles quotes, comments, and BOM. This is a deliberate, documented exception to stdlib-only — or replicate its three rules in a unit-tested helper (BOM strip + `strconv.Unquote` + comment trim).
- For TOML: constrain extraction to known table roots (`[project]` / `[package]` / `[tool.poetry]`) and strict scalar rules: only single-line `key = "value"`; multiline strings, dotted keys, inline tables ⇒ **empty field, never a guess**.
- Every extraction returns `(value string, ok bool)`; "not ok" must propagate to an empty field, never a partial value.

**Warning signs:** A parser returning a name containing `"` or `//`; fixture corpus missing a BOM case; no test asserting "malformed input ⇒ empty field, nil error"; any parser that logs or returns parse errors instead of empty fields.

**Phase to address:** Every manifest-parsing phase (Go line parser, TOML parsers). The shared `readManifest` helper belongs in the foundation phase.

---

### Pitfall 3: `encoding/xml` Root-Element Trap — Silent All-Empty Parse

**What goes wrong:** The natural struct pattern for `.csproj`/`pom.xml` — `struct{ Project struct{...} `xml:"Project"` }` — silently matches NOTHING. **Verified locally on Go 1.26.4:** the top-level struct IS the root element; the root name must be captured via `XMLName xml.Name `xml:"Project"`` and fields match the root's **children** by local name (namespace-agnostic — both the csproj `xmlns` and pom `xmlns` parse fine, as do prefixed namespaces). The broken pattern yields all-empty fields with nil error — the worst failure mode for best-effort parsing, because it looks like "valid but empty project" instead of "parse failed".

**Why it happens:** Developers port patterns from DOM/ElementTree mental models where the root is a node you match; Go's decoder is positional (the struct you pass IS the root).

**How to avoid:**
- Use the `XMLName` pattern in every XML parser (verified working with namespaces, BOM, XML declaration, and attributes via `xml:"Condition,attr"`).
- Assert parse quality: after unmarshal, if no expected field was populated, treat the manifest as unparsed (empty fields), not as a valid manifest with empty values.
- Golden fixtures of real-world files: SDK-style `.csproj` with and without `xmlns`; `pom.xml` with `<parent>` and with `<modules>`.

**Warning signs:** XML parser unit tests that never assert a non-empty value; structs wrapping the root element in a field; a `.csproj` fixture that parses to zero PropertyGroups without a test noticing.

**Phase to address:** C#/.NET and Java/Kotlin detector phases (both parse XML).

---

### Pitfall 4: Version Semantics — Language Directives, Inherited and Missing Versions

**What goes wrong:** `Version` is populated with values that mean different things per language, or fabricated when absent:
- **go.mod**: the `go 1.26` directive is the minimum *language* version, not a project release version — Go modules declare no version in `go.mod` (versions come from git tags). The milestone spec maps "go.mod go directive" → Version; flag this for a decision: report it as a separate `GoVersion` field or leave `Version` empty for Go, otherwise consumers see "1.26" as a release version.
- **pom.xml**: child modules without `<version>` inherit from `<parent><version>` — **verified locally**: `artifactId` matched, root `version` was empty, `parent.version` = "2.0.0". Naive parsers report empty version for inheriting Maven modules.
- **package.json**: `version` is optional (private packages omit it); values are sometimes non-semver ("1.0", date-based).
- **pyproject.toml**: `dynamic = ["version"]` — computed at build time, absent from the file; Poetry stores it under `[tool.poetry]` instead of PEP 621 `[project]`.
- **Cargo.toml**: `version.workspace = true` — inherited from the workspace root's `[workspace.package]`.
- **composer.json**: `version` is rare — git tags are the source of truth.

**How to avoid:**
- Empty version = empty string. Never fabricate "0.0.0" or "unknown" — misleads consumers and breaks downstream semver comparisons.
- Report the **raw string** as it appears (v-prefix, prerelease, build metadata preserved). Normalization is a later decision — `hashicorp/go-version` is already a repo dependency (v1.5 decision) if ordering ever becomes necessary.
- Per-language fallback chain, documented in code: Java: root `<version>` → `<parent><version>` → empty; Cargo: `[package] version` → `version.workspace` (resolve from workspace root manifest) → empty; Python: PEP 621 `[project] version` → Poetry `[tool.poetry] version` → empty (never read `dynamic`).
- Log the go-directive-as-version decision in PROJECT.md Key Decisions when the phase lands.

**Warning signs:** Tests asserting `Version == "0.0.0"` for versionless projects; `Version` populated from the go directive without a decision record; no pom-parent fixture; no Cargo workspace fixture.

**Phase to address:** Version extraction phase (needs cross-file resolution for Cargo workspaces and Maven parents — comes after per-language name detection).

---

### Pitfall 5: Walk Performance — Unpruned Vendored Directories

**What goes wrong:** Probing a real project with `node_modules` (50–100k files), `target/`, or `.gradle/` takes minutes instead of milliseconds; tests become slow and flaky. The old code checks `ignoreDirs` only at the **top level** — a nested `apps/web/node_modules` is walked in full. The old ignore list also misses common dirs: `.next`, `.nuxt`, `.gradle`, `venv` (no dot — the most common Python venv name!), `env`, `Pods`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `htmlcov`, `.terraform`, `bower_components`, `jspm_packages`, `CMakeFiles`, `.cache`, `.yarn`.

**Why it happens:** The ignore list is applied once at the root instead of at every directory level; and the list reflects the author's own ecosystems, not the 7 target ecosystems plus common build tools.

**How to avoid:**
- `filepath.WalkDir` + `return fs.SkipDir` for ignored dir names at **every level**, before reading the directory. **Verified locally:** WalkDir does not follow symlinks (uses Lstat) — symlink cycles cannot occur; a symlink-to-dir is visited as one entry with `Type()&fs.ModeSymlink` set and is not descended into. Document this behavior (a symlinked `shared` → real project will be skipped).
- Stop early: best-effort first-match — check the root's own `os.ReadDir` before any descent, and return as soon as the first manifest matches.
- Keep the ignore set as one exported, documented variable so users can extend it.
- Add a benchmark/fixture test: a tree with 10k+ files asserting the probe completes quickly.

**Warning signs:** Probe on a repo with `node_modules` takes >1s; walk visits files inside `.git`; test trees never contain nested vendor dirs; the walker reads directory contents before checking the ignore list.

**Phase to address:** Walker phase (foundation) — before any detector depends on it.

---

### Pitfall 6: `path` vs `filepath` on Windows — Silent Name Loss

**What goes wrong:** **Verified locally:** `path.Base("C:\dev\proj\go.mod")` returns the WHOLE path (the `path` package splits on `/` only). The old `detector_go.go` check `path.Base(projectFile) == "go.mod"` therefore fails on Windows — the module name is silently lost and the name falls back to the folder name. Any `path` usage in the new package (Base/Join/Dir on discovered file paths) breaks the Windows CI leg the same way.

**Why it happens:** `path` is for URL-style paths. On macOS/Linux both packages behave identically, so the bug ships unnoticed — exactly what happened to the old sample.

**How to avoid:**
- `filepath` everywhere for filesystem paths; ban `path` in `project_probe` (code-review checklist item; the repo already has a Windows CI matrix).
- Add a pure-string unit test for the Windows path shape: `filepath.Base(`C:\proj\go.mod`) == "go.mod"` — needs no OS, verifies separator handling in the exact function used.
- Do not rely on the Windows CI leg to catch it — CI only runs on PRs.

**Warning signs:** `import "path"` in any `project_probe` file; `path.Base`/`path.Join` on walk-discovered paths; go.mod name extraction returning the full absolute path on Windows.

**Phase to address:** Walker/detector foundation phase.

---

### Pitfall 7: Monorepos & Nested Projects — Undefined Manifest Precedence

**What goes wrong:** A monorepo root has `package.json` AND `apps/api/package.json` AND `apps/web/go.mod`. Which one wins? The old `FindFirst` returns the first file found in walk order — deterministic only because `os.ReadDir` sorts entries, but the rule is undocumented and depth-first order surprises users (a nested module's manifest can win over the root's). `go.work`-only Go workspaces (no `go.mod` at root) are missed entirely. Python has three competing manifest generations: `pyproject.toml` (PEP 621), Poetry `[tool.poetry]`, and legacy `setup.py`/`setup.cfg`.

**Why it happens:** Detection looks like "find any manifest", so the precedence question is deferred until users file issues about wrong answers.

**How to avoid:**
- Define and document the rule: **root-first, then shallowest-first depth-first walk with sorted entries**; within a folder, per-language manifest precedence (`pyproject.toml` > `setup.cfg` > `setup.py`; `.sln` > `.csproj` etc.).
- Detector ordering stays an ordered slice (never map iteration — random order).
- Recognize `go.work` as a Go signal (Language = go; name from first `use` module path or folder name).
- Fixture tests: monorepo tree (root manifest + nested manifests), git submodule dir containing its own `.git` (must be ignored), `go.work`-only root, PEP 621 vs Poetry `pyproject.toml`.

**Warning signs:** Tests only cover single-manifest dirs; probe output differs between runs on the same tree (map ordering leak); no documented precedence rule; `go.work` root reported as Unknown.

**Phase to address:** Walker phase (precedence + ordering tests) plus each per-language detector phase (manifest precedence within a language).

---

### Pitfall 8: Manifest Robustness — BOM, JSONC, Size Caps, Encoding, Fuzzing

**What goes wrong:** Real-world manifests violate strict parsers: **verified locally** — `json.Unmarshal` fails on a UTF-8 BOM ("invalid character '\ufeff' looking for beginning of value") and on trailing commas/JSONC ("invalid character '}'..."). Hand-edited `package.json` files with trailing commas are common. A 10 MB minified README or malformed manifest read whole into memory; UTF-16 legacy manifests produce mojibake strings instead of empty fields. `encoding/xml` tolerates BOM and declarations (verified) — so the JSON path needs the BOM strip that XML does not.

**Why it happens:** `encoding/json` is strict by spec; the tool author assumes manifests are well-formed because the ecosystem tooling requires it — but probe tools run against broken, hand-edited, or partially-written files.

**How to avoid:**
- One shared `readManifest` helper: size cap (1 MB) → `os.ReadFile` → BOM strip → parse; **parse failure ⇒ empty data, never an error** (best-effort contract).
- Fuzz every parser with Go's built-in fuzzing (`FuzzXxx`): seed corpus of real manifests (go.mod, package.json, pyproject.toml, Cargo.toml, .csproj, pom.xml, composer.json). Seeds run during normal `go test` (they count toward the coverage gate); full fuzzing is a separate `make fuzz` target.
- If content fails `utf8.Valid`, treat as unparseable (empty fields) — never report mojibake.
- gosec: `os.ReadFile` on walk-discovered paths triggers G304 — keep the `#nosec G304` comment with the old justification ("path comes from local discovery in the inspected folder").

**Warning signs:** No fuzz targets; no BOM fixture for JSON parsers; `os.ReadFile` without a size cap; parsers returning errors to the caller instead of empty fields.

**Phase to address:** Parser hardening — fold into each parser phase (fixtures + fuzz seeds per language), with a dedicated hardening pass at the end of the milestone.

---

### Pitfall 9: README Description — Naive First-Paragraph Extraction

**What goes wrong:** The "first paragraph" of most READMEs is the **title line** (duplicated as the name), a row of badge images (no text at all), or a table of contents. README files vary in name (README, README.md, README.rst, README.txt) and markup — reStructuredText uses `====` underline headings, not `#`. Reading the whole file is wasteful for a one-paragraph description.

**Why it happens:** "First paragraph" seems self-evident until real READMEs are examined; the title line is the most common first line and the naive split lands on it.

**How to avoid:**
- Skip heading lines (`# ..`, `====` underlines), image-only lines, HTML comments, and ToC blocks; take the first non-empty text paragraph after the title; truncate to a sane length (e.g. 200 chars).
- Case-insensitive README name matching; support the common variants per language (Rust/Cargo projects often have no README; fall back to manifest description first, README second — the milestone already orders description: manifest → README fallback).
- Cap read size (first 64 KB is plenty).
- Fixtures: README whose first paragraph is the title; README starting with a badge; `.rst` README with underline headings.

**Warning signs:** `description == name` for most fixtures; description containing markdown syntax (`#`, `![]`); README.rst returning markup text; description from a 5 MB README.

**Phase to address:** Description extraction phase (after name/version detection).

---

### Pitfall 10: Dead Sample Left in Repo — `project_detector/` Does Not Build

**What goes wrong:** **Verified:** `project_detector/` is untracked and imports `github.com/guionardo/gs-dev` and `github.com/BurntSushi/toml`, which are **not in go.mod** — `go build ./...` fails locally with 3 errors. CI stays green only because the directory is untracked. But: golangci-lint and pre-commit scan the filesystem and will lint/vet it (gofmt, G304, deprecation noise); a `git add .` commits it and all 3 CI legs go red; downstream consumers running `go test ./...` on the module break.

**Why it happens:** Sample code copied from another project (gs-dev) as a reference; "deprecated" was declared in prose but never made executable — no build tag, no removal.

**How to avoid:**
- The milestone **must delete `project_detector/`** — its knowledge (ignore list, ordered detectors, Unknown fallback) is ported into `project_probe` with tests, never by copying files.
- Verify the repo builds clean (`go build ./...` and `make coverage-quick`) in the same phase that creates `project_probe`.
- If any old behavior must be preserved, port it test-first; the git history retains the sample.

**Warning signs:** `git status` shows `project_detector` untracked; `go build ./...` fails locally; README still lists the sample as a package; pre-commit/lint output mentions project_detector files.

**Phase to address:** Package foundation phase — the deletion lands with (or immediately before) the first `project_probe` commit.

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hand-rolled TOML subset parser (stdlib-only) | Zero new dependencies | Breaks on legal TOML (multiline strings, dotted keys, inline tables) — silently empty fields | Acceptable for best-effort IF strict degrade-to-empty is tested; revisit when framework detection (dependency-based, later milestone) needs real TOML — then take `pelletier/go-toml/v2` |
| Raw version passthrough (no semver normalization) | Simple, lossless | Consumers must normalize themselves; no version ordering | Correct for v1.7; normalization later via `hashicorp/go-version` (already a repo dep) |
| Depth-first first-match walk | Simple walker | Nested manifests can shadow root intent; deep scans on monorepos | Acceptable with the documented root-first rule + monorepo fixture tests |
| Swallowing unreadable-subdir errors | Quiet output | Projects silently reported Unknown | Acceptable ONLY for subdirectories; root-folder errors must surface per the API contract |
| Copying old detector code into the new package | Fast start | Two divergent implementations drift; dead code retains bugs (path.Base, BOM) | **Never** — port behavior test-first, then delete the sample |
| Ignore-list as a package-private constant | Simple | Users with exotic build dirs get slow probes and can't extend | Acceptable for v1.7; export the set when the first user asks |

## Integration Gotchas

Common mistakes when connecting to existing repo infrastructure.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `go.mod` name extraction | Naive `CutPrefix("module ")` | `golang.org/x/mod/modfile.ModulePath` (toolchain parser) or a BOM+Unquote+comment-trim helper |
| `pyproject.toml` parsing | Reaching for `gopkg.in/yaml.v3` (already a repo dep) | pyproject.toml is **TOML, not YAML** — table-scoped hand-rolled reader or go-toml/v2 |
| Coverage gate (`make coverage-quick`: pkg ≥80%, total ≥75%; CI 95%) | Parser error paths untested → gate fails on PR | Table-driven fixture tests + fuzz seeds (run during `go test`, count toward coverage) |
| Windows CI leg | `path` instead of `filepath` | `filepath` everywhere + pure-string Windows-path unit test |
| Symlink fixtures on Windows | `os.Symlink` needs admin/Developer Mode → CI fails | `t.Skip` symlink tests on Windows, or build symlinks via junctions |
| Old package type names (`"nodejs"` for JS) | New `Language` enum drifts from old values | Document the mapping (or keep compatible string constants) in the migration note |
| golangci-lint / pre-commit on untracked files | `project_detector` lint noise and gofmt failures | Delete the sample in the foundation phase |
| README package index (main README table) | New package missing from index | Update README table in the same phase that ships the package |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Unpruned recursive walk | Probe takes seconds on repos with `node_modules`/`target`/`.gradle` | `fs.SkipDir` on the ignore list at EVERY level, before reading dir contents | Dirs with >10k files (node_modules, .gradle, target) |
| Whole-file read of huge manifests/READMEs | Memory spikes; reading package-lock.json needlessly | Size cap (1 MB) via `readManifest`; 64 KB LimitReader for README | Files >1 MB |
| Full-tree scan when root already has a manifest | Unnecessary descent into subprojects | Check root `os.ReadDir` first; stop on first match | Any repo with a root manifest |
| Reading lockfiles (`package-lock.json`, `Cargo.lock`, `go.sum`) | Wasted IO; false-positive detection | Manifest whitelist — never probe lockfiles | Always (they are huge) |
| Per-detector re-walking the tree | Each of 7 detectors walks independently → 7× IO | One shared walk; detectors consume the collected candidate files | Monorepos with many files |

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| `os.ReadFile` on walk-discovered paths (gosec G304) | CI lint failure; flagged as arbitrary-file-read | `#nosec G304` with the existing justification comment (paths from local discovery, not user input) |
| Unbounded file read | Memory DoS when probing a tree containing a 100 MB manifest | Size cap in `readManifest` before `ReadFile` |
| Symlink traversal | Following links out of the probed tree (e.g. into /etc) | WalkDir doesn't follow symlinks (verified); add a fixture test asserting the behavior |
| Reporting raw paths | Callers trust the `Folder` string and pass it to file APIs | `filepath.Abs` + `Clean` before returning; document that paths are local |
| Panic on malformed input | Parser panics (nil map write, slice bounds) crash the caller | `defer recover` in `Probe` is a last resort; fuzz targets catch panics before release |

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| `go` directive reported as Version | Downstream consumers see "1.26" as a release version | Version empty for Go, or a dedicated `GoVersion` field (decision record) |
| Composer `vendor/name` reported as project name | Name like "acme/widget" surprises users | Strip the vendor prefix or document the format; prefer the `name` portion |
| Description equals title line | Name duplicated as description | Skip headings/badges/ToC; take the first real text paragraph |
| Unknown-with-error | Callers can't distinguish "no project" from "broken" | Error only for hard I/O failures; Unknown is nil-error |
| Version with v-prefix for some languages, without for others | Inconsistent-looking output | Raw passthrough + document per-language semantics in the type docs |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **BOM handling:** every parser tested with a BOM-prefixed fixture (JSON fails without strip — verified; XML tolerates; line parsers silently miss — verified)
- [ ] **go.work-only workspace** detected as Go, not Unknown
- [ ] **pom.xml child without `<version>`** inherits from `<parent><version>` (fixture asserts inherited value — verified behavior)
- [ ] **csproj conditional PropertyGroup** (Debug-only `Condition`) not chosen over the unconditional one
- [ ] **Cargo `version.workspace = true`** resolved from the workspace root manifest
- [ ] **package.json without name/version** → folder-name fallback + empty version (no "0.0.0")
- [ ] **No `path` import** in the package; Windows-path unit test present
- [ ] **Determinism:** same tree → same result across runs (sorted walk + ordered detector slice, no map iteration)
- [ ] **Contract test:** empty folder → Unknown + nil error; unreadable folder → error
- [ ] **README variants** (README, README.md, README.rst, badge-first) all yield sensible descriptions
- [ ] **`project_detector/` deleted**; `go build ./...` green; `make coverage-quick` green
- [ ] **Fuzz targets** exist with real-manifest seed corpora (run in normal `go test`)

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Silent wrong manifest values shipped | MEDIUM | Add assertion tests ("at least one field populated per fixture"); add BOM/quotes fixtures; release a patch |
| Probe slow on real repos | LOW | Add missing ignore dirs + SkipDir; add the 10k-file benchmark fixture to prevent regression |
| Wrong version semantics shipped | HIGH (API/UX) | Keep raw string; document per-language semantics; rename/redocument fields only in a next major version |
| `project_detector/` committed accidentally | MEDIUM | `git rm -r project_detector/`; red CI is the alarm — it fails fast on all 3 OSes |
| Parser panic found by fuzz | LOW | Fix + add the failing input as a regression fixture before the next release |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls (proposed v1.7 phase structure).

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| C1 API contract (error vs Unknown) | Phase 1: Package foundation — API, `doc.go`, `readManifest` helper, delete `project_detector/` | Contract tests (Unknown nil-error, I/O error paths); `go build ./...` + `make coverage-quick` green |
| C10 dead sample in repo | Phase 1 (same as above) | `git status` clean of project_detector; lint output clean |
| C6 `path` vs `filepath` | Phase 1–2: foundation + walker | No `path` import; Windows-path unit test; windows CI leg |
| C5 walk performance | Phase 2: Walker + ignore list + early-stop | 10k-file fixture benchmark; nested vendor dirs pruned; symlink fixture |
| C7 monorepo/nested precedence | Phase 2: walker ordering + precedence tests | Monorepo fixture; determinism test; go.work fixture |
| C2 line-scanning (BOM/quotes/comments) | Phase 3: Go + JSON parsers (shared helper from Phase 1) | BOM fixture, quoted-module fixture, comment fixture per parser |
| C8 robustness + fuzz | Phases 3–7: each parser phase + final hardening pass | Fuzz targets with real-manifest seeds; size-cap test; utf8.Valid test |
| C3 XML root trap | Phase 5: XML parsers (C#/.NET, Java/Kotlin) | Golden csproj/pom fixtures with namespace + BOM; non-empty assertion |
| C4 version semantics | Phase 6: Version extraction (needs cross-file: Maven parent, Cargo workspace) | Per-language version fixtures; decision record for go directive |
| C9 README description | Phase 6: Description extraction | README variant fixtures; description ≠ name assertion |
| TOML stdlib gap (decision) | Phase 4: TOML parsers (Python, Rust) — decide hand-rolled vs go-toml/v2 | Decision record; workspace-inheritance fixture; strict degrade-to-empty tests |

## Sources

- **Local verification experiments** (Go 1.26.4 and 1.27.0, run 2026-09-28): JSON/XML BOM behavior, JSONC rejection, `encoding/xml` root-element pattern + namespace handling + pom parent-version inheritance, `filepath.WalkDir` symlink behavior, `os.ReadDir` symlink reporting, `path.Base` on Windows paths, go.mod BOM/quoted-module parsing. Digests stored in `.planning/research/.cache/` (keys 3ec06cb3…, 9e7bd084…, 24cd49f7…, de674c6e…, f1d67377…).
- **pkg.go.dev/golang.org/x/mod/modfile** — canonical go.mod parser used by the Go toolchain; `ModulePath` tolerant extraction (digest 6216feed…).
- **github.com/pelletier/go-toml/v2** and **github.com/BurntSushi/toml** — TOML library landscape; BurntSushi maintainer's README recommends alternatives ("fallen behind the upstream TOML specification"); go-toml/v2 TOML 1.1.0, 5–10× faster, actively maintained (digest 9374a87e…).
- **github.com/go-enry/go-enry** — strategy-cascade detection, `IsVendor` filtering, "unknown is empty, not error" semantics; wrong abstraction for project-level metadata (digest 36290e0c…).
- **GitHub linguist strategy cascade** — ordered strategies, first single-candidate wins, vendor.yml hygiene, 50 KB content heuristics (digest 14e59e20…).
- **aquasecurity/go-dep-parser** — per-format manifest-parser architecture reference (digest 0de693ec…).
- **In-repo evidence:** `project_detector/` (untracked, non-building sample; `path.Base` bug; top-level-only ignoreDirs; error-as-control-flow) and CI workflow `.github/workflows/go.yml` (3-OS matrix, coverage gate).

---
*Pitfalls research for: project_probe (v1.7) — best-effort project detection and manifest parsing*
*Researched: 2026-09-28*

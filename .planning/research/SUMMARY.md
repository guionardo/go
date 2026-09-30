# Project Research Summary

**Project:** `github.com/guionardo/go` — milestone v1.7 `project_probe` package
**Domain:** Best-effort project detection library (folder → language/name/version/description) inside a stdlib-first Go utility monorepo
**Researched:** 2026-09-28
**Confidence:** HIGH (with MEDIUM pockets — see Confidence Assessment)

## Executive Summary

`project_probe` is a small, best-effort Go library that reads a folder's contents and reports the project's language, name, version, and description across 7 languages (Go, Python, JS/TS, C#/.NET, Rust, Java/Kotlin, PHP) — never failing: unknown content yields a typed `Unknown` value with nil error, and only hard I/O failures (missing/unreadable folder) return errors. Research across the ecosystem (GitHub Linguist, Snyk, vership, opendray/projectscan, Rust `project-detect`, IntelliJ) converges on one design: **an ordered cascade of manifest-first detectors, first match wins, unknown = fallback value never an error**. The mature ecosystem API is a single `Probe(folder) (ProjectData, error)` entry point backed by a private ordered detector registry with a `(ProjectData, bool)` contract — not error-as-control-flow, not plugin registration.

The recommended approach is **zero new runtime dependencies**. Every manifest format in scope parses with Go 1.26 stdlib (`os`, `bufio`, `strings`, `encoding/json`, `encoding/xml`, `path/filepath`, minimal `regexp`) plus one unexported section-aware TOML-subset reader (~80 lines) for `pyproject.toml`/`Cargo.toml` — Go stdlib has no TOML parser (verified on Go 1.27.0), and the already-present `yaml.v3` is a category error for TOML. A full TOML dependency (`pelletier/go-toml/v2` or BurntSushi) is explicitly deferred to the framework-detection milestone. The deprecated `project_detector/` sample (which does not compile — 3 verified build errors) must be deleted as a stack-level prerequisite; its logic is the design reference, its dependencies are not.

Key risks and mitigations: (1) **API contract ambiguity** (error vs Unknown) — write the contract in `doc.go` on day one, error reserved for I/O failures only; (2) **silent wrong values from naive parsing** (BOM, quotes, comments, XML root trap) — one shared `readManifest` helper (size cap + BOM strip), `XMLName` pattern for XML, strict degrade-to-empty for TOML, all verified by locally-run experiments; (3) **version semantics** — the go.mod `go` directive is a language floor, not a release version; report raw strings, never fabricate, and record the decision; (4) **Windows `path` vs `filepath`** — ban `path`, add a pure-string Windows-path test. Confidence is HIGH overall: stack, architecture, and pitfalls research all include locally-verified experiments (Go 1.26.4/1.27.0) and converging primary sources; features research is MEDIUM (cross-checked across 10+ tools, some single-source claims).

## Key Findings

### Recommended Stack

Stdlib-only parsing with one hand-rolled unexported TOML subset reader. Verified facts driving decisions: Go stdlib has **no TOML parser** (checked on Go 1.27.0); `pyproject.toml`/`Cargo.toml` are TOML, so the existing `yaml.v3` dep cannot help; the mature pattern is manifest-first ordered cascades (Linguist, Snyk), not content-based detection (`go-enry` is the wrong abstraction). The deprecated `project_detector/` sample is the structural design reference but breaks the build (3 errors: missing `gs-dev`, missing BurntSushi go.sum entry) — delete it in this milestone.

**Core technologies:**
- Go 1.26.4 (go.mod; local 1.27.0): language + toolchain — repo constraint; all features used are ≥1.24 (`strings.SplitSeq`)
- `os`/`bufio`/`strings`: folder listing, line-oriented manifest reads (go.mod, .sln, gradle, README) — stdlib, bounded scanners
- `encoding/json`: `package.json`, `composer.json` — typed struct decode, strict JSON (no JSONC concern)
- `encoding/xml`: `.csproj`, `.slnx`, `pom.xml` — local-name matching is namespace-agnostic (handles real `xmlns`); use the `XMLName` pattern (root-trap verified)
- `path/filepath`: discovery + Windows-safe path handling — `path` package is banned (silent Windows breakage verified)
- Unexported `toml.go` subset reader (~80 lines): section-tracked `key = "value"` extraction for `[project]`/`[tool.poetry]`/`[package]` — strict degrade-to-empty on anything else; **defer** `pelletier/go-toml/v2` to the framework-detection milestone
- `errors`: sentinel `ErrNotDir`-style error for the one real error path (folder missing/unreadable)
- Dev tools (existing, no changes): `make coverage-quick` (pkg ≥80/file ≥70/total ≥75 — no override needed; `testdata/` auto-excluded), golangci-lint with `#nosec G304` annotations on manifest reads, `doc.go` convention, README package index row, commitlint `feat(project_probe):`

### Expected Features

Manifest-first detection with per-field fallback chains is table stakes; the never-fail `Unknown` contract and README-description fallback are the differentiators. Version is always the raw manifest string, empty when absent/dynamic — never fabricated, never normalized.

**Must have (table stakes):**
- 7 manifest-first language detectors (Go, Python, JS/TS, C#/.NET, Rust, Java/Kotlin, PHP), ordered, first-match wins — the core feature
- Name extraction: manifest name → folder base fallback
- Version extraction: raw manifest value, empty when absent/dynamic (go.mod `go` directive semantics = P1 documentation decision)
- Description: manifest field → README first-paragraph fallback → empty (README fallback required because go.mod and gradle have no description field)
- `Unknown` type with nil error for unmatched folders — best-effort contract
- Folder validation + ignore-list hygiene (vendored/build/IDE dirs must not masquerade as markers)

**Should have (competitive):**
- Never-fail `Probe(folder) (ProjectData, error)` — error reserved for I/O failures; display tooling can call it unconditionally
- Stdlib-only with minimal TOML subset parser — repo constraint honored; malformed TOML → Unknown, never error
- README first-paragraph description fallback — only description source for 2 of 7 languages
- Deterministic detector ordering — predictable output, trivially testable

**Defer (v2+ / P2-P3):**
- P2: `tsconfig.json` as TS confirmation, `settings.gradle` `rootProject.name`, lockfile secondary confirmation, .csproj selection rule
- P3/deferred milestone: framework detection (dependencies-based), extension-count fallback, monorepo/workspace detection, version normalization, detection-depth scanning
- Anti-features (never for v1.7): executing build tools, network calls, recursive/symlink-following discovery, full TOML dep, version normalization, parsing `build.gradle` code as data, lockfile-first detection

### Architecture Approach

A private, ordered slice of per-language detectors, each with a `detectX(folder, entries) (ProjectData, bool)` contract; the registry does one `os.ReadDir`, filters ignored dirs, iterates detectors first-match-wins, then falls through to `LanguageUnknown` with nil error. Detectors are root-scoped only (no subdir probing), self-contained per language file, and share three unexported helpers: `toml.go`, `readme.go`, `ignore.go`. `ProjectData{Folder, Language, Name, Version, Description}` with independent per-field fallbacks. `(ProjectData, bool)` was validated across opendray, Rust project-detect, vership; plugin-style registration was rejected (global mutable state, no consumer need).

**Major components:**
1. `probe.go` — `ProjectData`, language constants, `Probe()` entry point, `doc.go` (contract written here)
2. `registry.go` — single `os.ReadDir`, ignored-dir filter, ordered `[]detectorFunc` loop, Unknown fallback (~60 lines)
3. `detector_<lang>.go` ×7 — root marker claim + manifest parse + per-field fallbacks, one file per language
4. `toml.go` — section-aware TOML subset scanner (shared by Python + Rust)
5. `readme.go` — README first-paragraph extraction (description fallback)
6. `ignore.go` — vendored/generated dir exclusion set

Package name `projectprobe` (directory `project_probe/`, repo convention). Build order from research: skeleton+Unknown contract → shared helpers → text/JSON detectors → XML+TOML detectors → polish.

### Critical Pitfalls

1. **API contract ambiguity (error vs Unknown)** — write the contract in `doc.go` day one: error reserved for hard I/O failures; detectors return `(data, bool)`, never error-as-control-flow; empty folder → Unknown + nil error; ordered slice, never map iteration (nondeterministic)
2. **Naive line-scanning silently produces wrong values** (BOM, quoted module directives, trailing comments — all verified on Go 1.26.4) — one shared `readManifest` helper: size cap (1 MB) → BOM strip → parse; parse failure ⇒ empty fields, never an error; every parser gets a BOM fixture
3. **`encoding/xml` root-element trap** — struct-with-wrapped-root silently parses to all-empty fields (verified); use `XMLName xml.Name` pattern matching root children by local name; assert at least one field populated per fixture
4. **Version semantics** — go directive is a language floor (not a release version; decision record required); pom child inherits `<parent><version>` (verified); Cargo `version.workspace = true`; pyproject `dynamic = ["version"]` — report raw, empty when absent, never fabricate "0.0.0"
5. **Unpruned vendored dirs / Windows `path` vs `filepath`** — `fs.SkipDir` hygiene at every level if any subdir scan exists (10k-file benchmark fixture); ban `path` imports (verified: `path.Base` on Windows paths returns the whole path), pure-string Windows-path unit test

## Implications for Roadmap

Suggested phase structure (synthesized from ARCHITECTURE.md build order + PITFALLS.md phase mapping; each phase leaves the package green under `make coverage-quick`):

### Phase 1: Package Foundation — API Contract + Repo Cleanup
**Rationale:** The Unknown-without-error contract is the riskiest design bet and shapes every later phase; the dead `project_detector/` sample must be removed before any code lands (verified: `go build ./...` fails).
**Delivers:** `doc.go` with the written error/Unknown contract, `probe.go` (`ProjectData`, 7+Unknown language constants, `Probe`), `registry.go` with `(ProjectData, bool)` ordered loop, `ignore.go`, shared `readManifest` helper (1 MB cap + BOM strip); `project_detector/` deleted; contract tests (empty folder → Unknown+nil, missing folder → error, unknown folder → Unknown+nil); `filepath`-only discipline + Windows-path unit test.
**Addresses:** FEATURES table stakes — Unknown contract, folder validation, ignore hygiene.
**Avoids:** PITFALLS C1 (API contract), C10 (dead sample), C6 (`path` vs `filepath`), foundation of C8 (readManifest).
**Research flag:** None — well-documented, standard patterns. Skip research-phase.

### Phase 2: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback
**Rationale:** Easiest parsers first (line scan + `encoding/json` are stdlib-trivial) validate the detector contract shape before TOML complexity arrives; `readme.go` lands here because go.mod has no description field, so the fallback chain is exercised immediately.
**Delivers:** `detector_go.go` (go.mod module + go directive), `detector_javascript.go` (package.json), `detector_php.go` (composer.json), `readme.go`; BOM/quoted-module/comment fixtures; go-directive-as-version decision record in PROJECT.md.
**Addresses:** FEATURES P1 — Go/JS/PHP detectors, name/version/description chains, README fallback.
**Avoids:** PITFALLS C2 (line-scanning BOM/quotes/comments), C9 (naive README first-paragraph), C8 seeds for JSON.
**Research flag:** None — stdlib behaviors verified locally; standard patterns. Skip research-phase.

### Phase 3: TOML Subset + Python/Rust Detectors
**Rationale:** The TOML subset reader is the only genuinely novel parsing code; land it with its two real consumers so its strict degrade-to-empty contract is proven by fixtures, not by theory.
**Delivers:** `toml.go` (section-aware: `[project]`, `[tool.poetry]`, `[package]`; single-line `key = "value"` only; anything else → empty field), `detector_python.go`, `detector_rust.go`; TOML decision record (hand-rolled now; `pelletier/go-toml/v2` at framework milestone); Cargo `version.workspace` semantics decision.
**Addresses:** FEATURES P1 — Python/Rust detectors, TOML subset parser (differentiator).
**Avoids:** PITFALLS C2-TOML variant, Anti-Pattern 3 (naive regex TOML), C8 (fuzz seeds from real manifests).
**Research flag:** Phase 3 — MEDIUM-confidence area: the strict degrade-to-empty behavior on legal-but-unsupported TOML (dotted keys, multiline strings, inline tables, workspace inheritance) needs fixture-driven validation during planning. Use `/gsd-plan-phase --research-phase 3`.

### Phase 4: XML Detectors — C#/.NET + Java/Kotlin
**Rationale:** XML is the last parser family; the verified `XMLName` pattern and golden namespace fixtures de-risk both detectors. Includes the unresolved .NET marker-precedence decision.
**Delivers:** `detector_dotnet.go` (`.sln`/`.slnx`/`.csproj`), `detector_java.go` (`pom.xml`/`build.gradle(.kts)`); golden fixtures with `xmlns` + BOM; pom `<parent><version>` inheritance fixtures; non-empty-field assertions per fixture; per-language manifest precedence tests (sln > csproj, pyproject > setup.py).
**Addresses:** FEATURES P1 — C#/.NET and Java/Kotlin detectors (highest-complexity pair).
**Avoids:** PITFALLS C3 (XML root trap), C7 (manifest precedence, monorepo fixtures).
**Research flag:** Phase 4 — two open items: (a) **.NET subdir scan conflict**: STACK.md's per-language mapping includes a bounded 1-level subdir scan for `*.csproj`, while ARCHITECTURE.md (Pattern 4) and PITFALLS.md (Anti-Pattern 2) explicitly reject subdir probing ("a `.csproj` in a subdir belongs to a nested project"). Recommendation: root-scoped markers only, per the 2-of-3 consensus and the milestone's "what is *this* folder" contract; revisit only if .NET misclassification reports arrive. (b) **Detector ordering**: STACK.md cascade (Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP) vs ARCHITECTURE.md example order (go, rust, dotnet, java, python, js, php) differ — pick one documented order (recommend STACK's, matching the sample's relative order with Go first) during planning.

### Phase 5: Semantics, Hardening, and Release Polish
**Rationale:** Version/description semantics need cross-file and cross-language resolution that individual detector phases can't fully own; fuzzing and fixture breadth are the last defense against silent-wrong-value failures; the package must ship with docs and CI integration complete.
**Delivers:** Consolidated version-semantics decision records (go directive; Cargo workspace resolution; Maven parent inheritance — raw string, never fabricated); README variant fixtures (rst underline headings, badge-first, ToC); fuzz targets with real-manifest seed corpora for all parsers (run during normal `go test`, count toward coverage); 10k-entry benchmark fixture; `doc.go` prose + README package index row; `go vet`/lint pass (G304 annotations).
**Addresses:** FEATURES P1 completion + "Looks Done But Isn't" checklist (PITFALLS).
**Avoids:** PITFALLS C4 (version semantics), C8 (robustness + fuzz), C9 remainder, performance traps.
**Research flag:** Phase 5 (light) — Cargo workspace cross-file resolution design has no single verified source; validate during planning. Skip research-phase for the rest (standard hardening patterns).

### Phase Ordering Rationale
- **Contract first, parsers later:** the Unknown-without-error contract is the riskiest bet and every later phase builds on it; `readManifest` (BOM+cap) must exist before any parser (PITFALLS C2/C8)
- **Easiest parsers before harder ones:** text/JSON → TOML → XML de-risks the `(ProjectData, bool)` detector shape before the novel TOML subset and XML trap arrive (ARCHITECTURE build order)
- **Shared helpers before consumers:** `toml.go`/`readme.go` land before or with their first consumers so two language families don't re-implement the same helper
- **Version/description semantics consolidated at the end:** decisions (go directive, workspace inheritance) need cross-file evidence from all detector phases; the PITFALLS mapping converges on this as a final phase
- **Cleanup is prerequisite, not optional:** `project_detector/` deletion lands in Phase 1 because it breaks the build and CI cleanliness (verified)

### Research Flags
Needs research during planning (`/gsd-plan-phase --research-phase N`):
- **Phase 3:** TOML strict-degrade-to-empty edge cases (dotted keys, multiline strings, inline tables, `version.workspace`) — MEDIUM confidence, needs fixture-driven validation
- **Phase 4:** .NET marker precedence + root-only vs 1-level subdir scan (STACK vs ARCHITECTURE/PITFALLS conflict); detector ordering policy (two candidate orders in research)
- **Phase 5 (light):** Cargo workspace cross-file version resolution design

Standard patterns (skip research-phase):
- **Phase 1:** API contract, registry loop, ignore list — well-documented, locally verified
- **Phase 2:** go.mod/package.json/composer.json parsing + README extraction — stdlib behaviors verified by local experiments

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Stdlib TOML absence verified locally on Go 1.27.0; go.mod evidence (broken sample, dep graph); modfile/BurntSushi/go-toml docs cross-checked |
| Features | MEDIUM | Cross-checked across Snyk source, Linguist, PEP 621, Composer/MSBuild docs, langforge, repo-scanner, detect-stack, Understand-Anything; single-source claims (detect-stack, SherlockIO) marked LOW |
| Architecture | HIGH | Converging primary sources across 6+ implementations (Linguist, opendray, vership, Rust project-detect, projectdetect, IntelliJ); README-fallback specifics MEDIUM |
| Pitfalls | HIGH | Stdlib behaviors verified in local experiments on Go 1.26.4/1.27.0 (digests in `.planning/research/.cache/`); ecosystem conventions cross-checked |

**Overall confidence:** HIGH (features research is the MEDIUM outlier; all three other areas include locally-verified experiments)

### Gaps to Address

- **.NET subdir scan (STACK vs ARCHITECTURE/PITFALLS conflict):** resolved in synthesis in favor of root-scoped markers (2-of-3 consensus); confirm during Phase 4 planning
- **Detector ordering discrepancy:** STACK.md cascade vs ARCHITECTURE.md example order differ; fix one documented order during roadmap/planning (recommend: Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP)
- **go.mod `go` directive as Version:** P1 documentation decision — recommend reporting the raw directive with a documented "language floor, not release version" semantic (opendray precedent + milestone spec) recorded in PROJECT.md; alternative is a dedicated `GoVersion` field. Decide in Phase 2.
- **`path_tools` reuse (FEATURES) vs stdlib-only sibling imports (ARCHITECTURE/STACK):** resolved in favor of stdlib-only (`filepath.Abs` + `os.ReadDir` + small `hasFile` helper); `path_tools` remains an acceptable alternative if team prefers in-repo reuse — confirm during planning
- **Language constants refinement (TypeScript/Kotlin):** ARCHITECTURE lists optional refinement constants; recommend shipping 7 constants + `Unknown` in v1.7, refining via `tsconfig.json`/`build.gradle.kts` signals as P2 (matches FEATURES P2 triggers)
- **README fallback specifics:** rst underline headings, badge/ToC skipping heuristics are convention-level (MEDIUM); validate with real-world README fixtures during Phase 2 implementation
- **`project_detector/` deletion:** ARCHITECTURE.md's integration table says "leave as-is" but STACK.md + PITFALLS.md (verified build break) mandate deletion — treat deletion as authoritative; note ARCHITECTURE row as stale

## Sources

### Primary (HIGH confidence)
- Local toolchain experiments Go 1.26.4/1.27.0 (2026-09-28) — stdlib TOML absence (`go doc encoding/toml` fails), JSON/XML BOM behavior, `encoding/xml` root-element pattern + namespaces, pom parent-version inheritance, `filepath.WalkDir` symlink behavior, `path.Base` on Windows paths, go.mod BOM/quoted-module parsing (digests in `.planning/research/.cache/`)
- In-repo evidence — `project_detector/` build failure (3 errors), `go.mod` (yaml.v3 direct, x/mod indirect), `.testcoverage-quick.yml`, Makefile, CI workflow (3-OS matrix), package conventions (`doc.go`, README index)
- golang/go issues #60791, #68361 — no stdlib TOML adoption; Go 1.26 release notes (go.dev/doc/go1.26)
- GitHub Linguist `lib/linguist.rb` — ordered `STRATEGIES` cascade, first-single-candidate wins
- Rust `project-detect` crate + `vership` `project/detect.rs` + `opendray/projectscan` — ordered marker checks, `(Stack, bool)` contract, ordering-as-ambiguity-resolution, go-directive-as-version, ignored-dir maps
- JetBrains intellij-community `FileTypeRegistry.java` — plugin registration pattern (rejected), "refrain from throwing exceptions"

### Secondary (MEDIUM confidence)
- `pkg.go.dev/golang.org/x/mod/modfile` v0.41.0 — `ModulePath` tolerant extraction (upgrade path for go.mod parsing)
- `github.com/pelletier/go-toml/v2` v2.4.x + `github.com/BurntSushi/toml` v1.6 — TOML library landscape (deferral boundary; go-toml/v2 preferred if dep ever added)
- Snyk CLI source `detect.ts` — ordered `DETECTABLE_FILES` basename list, first-match
- `github.com/go-enry/go-enry/v2` — unknown-is-empty contract, vendor hygiene (wrong abstraction for this task)
- `github.com/aquasecurity/go-dep-parser` — per-format parser architecture reference (absorbed into Trivy; do not take as dep)
- PEP 621 pyproject.toml spec, Composer schema/Packagist, .NET SDK MSBuild properties docs — per-language version/name semantics
- `rios0rios0/langforge`, `@codegeneai/repo-scanner`, `nickpending/llmcli-tools` (language-detect), `Lum1104/Understand-Anything`, mise docs — feature landscape cross-checks
- Prior research cache — Snyk pattern, mise/asdf version-file discovery

### Tertiary (LOW confidence)
- `Luum-Home/luum-cognitive-os` (detect-stack) — name priority chain (single source)
- `griffincancode/sherlockio` — lockfile exclusion rationale (single source)
- `richardwooding/projectdetect` — indicator-based registry, multi-match API (single project, used only as contrast)
- `dispat` scanner docs — `Manifest{Path, Name, Version}` shape, name preference chain (MEDIUM, used for data model)

---
*Research completed: 2026-09-28*
*Ready for roadmap: yes*

# Feature Research

**Domain:** Best-effort project detection library (`project_probe`, 7 languages, Go stdlib-only)
**Researched:** 2026-09-28
**Confidence:** MEDIUM (cross-checked across Snyk source, langforge, repo-scanner, detect-stack, Understand-Anything, Linguist docs, PEP 621, Composer/Packagist, MSBuild docs; single-source claims marked LOW)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Manifest-first language detection (7 markers) | Every detection tool (Snyk, langforge, language-detect, repo-scanner, detect-stack) uses marker files as the primary signal; ordered list, first-match wins | LOW–HIGH per language | go.mod LOW (line-based); package.json/composer.json LOW (JSON); pyproject.toml/Cargo.toml MEDIUM (no stdlib TOML — needs minimal subset parser); .csproj MEDIUM (XML, multiple files); Java HIGH (pom.xml easy, build.gradle is code) |
| Name extraction from manifest, folder fallback | Universal name-priority chain across all surveyed tools: manifest name → module/package path segment → folder name | LOW | go.mod `module` last segment; composer `vendor/package`; pom `<artifactId>`; .csproj `PackageId`→`AssemblyName`→filename; settings.gradle `rootProject.name`; pyproject `[project].name` (required by PEP 621, never dynamic) |
| Version extraction, best-effort (empty when absent) | Every ecosystem makes version optional or derived: pyproject `dynamic=["version"]`, composer omitted by design, pom parent-inherited, .csproj defaults 1.0.0, package.json often absent for private packages | LOW | Never fabricate: report raw field only. `go.mod` has NO project version — `go` directive is a **toolchain floor** (mise docs explicitly: floors are not version requests). Decision needed: report it with documented semantics or leave empty |
| Description from manifest | Manifests carrying description: package.json, composer.json (required for Packagist), pyproject `[project].description`, Cargo.toml `[package].description`, .csproj `<Description>`, pom `<description>` | LOW | go.mod and build.gradle have no description field — README fallback (below) covers these |
| Unknown type without error | Best-effort contract for display tooling; diverges from Snyk (throws `NoSupportedManifestsFoundError`) intentionally | LOW | Unknown + folder-name fallback, never fail. Terminal state of the detector chain |
| Folder validation + empty-folder handling | Existing deprecated sample behavior; reuse `path_tools` (in-repo) instead of the old gs-dev dependency | LOW | Old code returns error on empty folder — keep error for empty/missing folder (input error), Unknown for undetectable content |
| Ignore-list hygiene (vendored/build/IDE dirs) | Linguist's vendored/generated/docs exclusion is table stakes for not misfiring on `node_modules`, `vendor`, `target`, `dist`, `.git` | LOW | Deprecated sample's `ignoreDirs` already implements this; keep, extend per language |
| README first-paragraph fallback for description | Without it, description is empty for Go and Gradle (2 of 7 languages have no manifest description). Cross-checked: Understand-Anything reads README head (~10 lines); detect-stack synthesizes from README | LOW | First non-empty paragraph, strip markdown heading/HTML. Mild differentiator vs pure manifest detectors (Snyk/langforge extract no description at all) |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Never-fail `Unknown` contract | Display tools (prompts, shell scripts, status lines) can call Probe() unconditionally; Snyk/older sample error out on unknown folders | LOW | Single `Probe(folder) (ProjectData, error)` — error reserved for I/O-level failures only |
| Stdlib-only with minimal TOML subset parser | Repo constraint "minimal external dependencies"; no `encoding/toml` exists in Go 1.26 stdlib (verified by absence; pelletier/go-toml v2 is the external standard) | MEDIUM | Parse only scalar keys under `[project]`, `[tool.poetry]`, `[package]`; skip arrays/inline tables; malformed TOML → Unknown, never error |
| README fallback description | Only description source for Go/Gradle projects; keeps 7/7 languages with a usable Description | LOW | See table stakes row |
| Deterministic detector ordering + priority chain | Predictable output, trivially testable; Snyk's first-match-wins validated in production | LOW | Order: manifest markers checked in fixed sequence; document conflict behavior (e.g., folder with both package.json and go.mod → first in order wins, flagged as monorepo edge) |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Executing build tools (`dotnet`, `gradle`, `mvn`, `npm`, `cargo`, `composer`) to resolve dynamic versions | "Get the real version" — resolves pyproject dynamic, gradle code versions, .csproj defaults | Slow (seconds–minutes), side effects (writes bin/obj, network), tool not installed, breaks offline/CI | Static parse only; empty Version when dynamic; document the semantic |
| Network calls (registry/API lookups for latest version, GitHub API) | Fresh version data | Offline-first library contract; latency; rate limits | Never network; version = manifest only |
| Recursive descent / symlink following for manifests | Catch nested projects | Escape root, symlink cycles, monorepo ambiguity explosion | Root-level scan only (`os.ReadDir`, no follow); monorepo detection deferred |
| Full TOML parser dependency | "Correctness" | Violates stdlib-only constraint; yanking BurntSushi/pelletier adds the only external dep for 2 fields × 2 files | Minimal subset parser (~100 LOC), documented TOML-1.0-scalar limitation |
| Extension-count language detection fallback | Detect language when no manifest (language-detect does this as Phase 2) | Weak signals, HIGH complexity, needs thresholds + depth limits + hidden-dir skipping; conflicts with Unknown contract | Keep Unknown; revisit as later milestone |
| Framework detection from dependencies | "What web framework?" | Explicitly deferred by milestone; requires per-ecosystem dependency tables (React/Spring/Express...), parse cost | Later milestone (needs dependency parsing of manifests) |
| Version normalization (semver parse, strip `v`, PEP 440, resolve .csproj default 1.0.0) | Consistent output | Fabricates values (1.0.0 default is indistinguishable from intended), munges raw data, each ecosystem has different rules | Return raw manifest value verbatim; empty when absent; consumers normalize |
| Parsing `build.gradle` content for name/version | "Gradle projects need versions too" | It's Groovy/Kotlin code — `version = "1.0"` is an assignment, not data; parsing is unreliable | Use `settings.gradle(.kts)` `rootProject.name` for name; no version; no description |
| Lockfile-first detection (package-lock.json, composer.lock, Cargo.lock) | Snyk uses lockfiles as primary package-manager signals | Lockfiles describe dependency state, not the project; SherlockIO explicitly excludes them from language stats | Manifests only; lockfiles at most secondary confirmation later |
| Multiple-manifest disambiguation (.csproj in subdirs, monorepos) | "Which project?" | First-match is wrong for true monorepos; per-project selection is a separate feature | Deterministic first-match (documented); multi-project probing deferred |

## Feature Dependencies

```
Language detection (7 detectors)
    └──requires──> Folder validation + ignore-list hygiene (path_tools reuse)

Name extraction
    └──requires──> Language detection (which manifest to read)
    └──enhances──> Folder-name fallback (always available)

Version extraction
    └──requires──> Language detection
    └──enhances──> Description extraction (same manifest read)

Description extraction
    └──requires──> Language detection
    └──enhances──> README fallback (independent read)

Unknown type (terminal state)
    └──requires──> All detectors (exhausted, no match)

README fallback
    └──enhances──> Description extraction (fills go.mod/build.gradle gap)
```

### Dependency Notes

- **All extraction requires language detection:** the manifest to read is unknown until the language is decided — detection phase and metadata phase must be sequential within one `Probe` call.
- **Ignore-list hygiene gates detection:** vendored dirs (node_modules, vendor, target) must be skipped before marker checks so a vendored package.json doesn't masquerade as the project.
- **README fallback enhances description only:** independent of manifest parsing; needs its own error tolerance (no README → empty description).
- **Depends on existing in-repo packages:** `path_tools` (folder assertion — replaces the deprecated sample's `github.com/guionardo/gs-dev/pkg/tools/files` import); `flow` optional for Default-style fallbacks. No new external dependencies.

## MVP Definition

### Launch With (v1)

- [x] 7 language detectors (Go, Python, JS/TS, C#/.NET, Rust, Java/Kotlin, PHP) — manifest-first, ordered, first-match — [essential: the core feature]
- [x] Name extraction: manifest name → folder name — [essential: stated milestone feature]
- [x] Version extraction: manifest version, raw, empty when absent/dynamic — [essential: stated milestone feature; semantics documented for go.mod `go` directive]
- [x] Description extraction: manifest description → README first paragraph → empty — [essential: stated milestone feature; README fallback needed for Go/Gradle]
- [x] `Unknown` type without error for unmatched folders — [essential: best-effort contract]
- [x] Folder validation via `path_tools` + ignore-list hygiene — [essential: correct input handling]
- [x] Stdlib-only: minimal TOML subset parser for pyproject.toml/Cargo.toml — [essential: repo constraint]

### Add After Validation (v1.x)

- [ ] `tsconfig.json` as TypeScript confirmation signal (mise reads it for node; @4meta5 and language-detect use it) — [trigger: JS/TS misclassification reports]
- [ ] `settings.gradle(.kts)` `rootProject.name` for Gradle name (pom-less Java/Kotlin projects) — [trigger: Gradle-only projects reporting folder name]
- [ ] Lockfile-based secondary confirmation (Cargo.lock/composer.lock/package-lock.json) — [trigger: manifest-less but lockfile-present folders]
- [ ] Deterministic .csproj selection when multiple exist (nearest-to-root rule) — [trigger: multi-project .NET folders]

### Future Consideration (v2+)

- [ ] Framework detection (dependencies-based) — [explicitly deferred by milestone; needs per-ecosystem dependency tables]
- [ ] Extension-count language fallback — [weak signals; conflicts with Unknown contract; needs thresholds]
- [ ] Monorepo/workspace detection (pnpm workspaces, go.work, .sln) — [separate product; repo-scanner-level scope]
- [ ] Version normalization + semver validation — [consumers normalize; keep raw for v1]
- [ ] Detection-depth scanning — [Snyk `--detection-depth`; complexity without clear v1 value]

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Go detector (go.mod) | HIGH | LOW | P1 |
| JS/TS detector (package.json) | HIGH | LOW | P1 |
| PHP detector (composer.json) | MEDIUM | LOW | P1 |
| Python detector (pyproject.toml) | HIGH | MEDIUM (TOML subset) | P1 |
| Rust detector (Cargo.toml) | MEDIUM | MEDIUM (TOML subset) | P1 |
| C#/.NET detector (.csproj) | MEDIUM | MEDIUM (XML + multi-file) | P1 |
| Java/Kotlin detector (pom.xml + build.gradle) | MEDIUM | HIGH (gradle is code; name from settings.gradle only) | P1 |
| Name extraction + folder fallback | HIGH | LOW | P1 |
| Version extraction (raw, empty-if-absent) | HIGH | LOW | P1 |
| Description extraction (manifest → README) | HIGH | LOW | P1 |
| Unknown type, no error | HIGH | LOW | P1 |
| Ignore-list hygiene | HIGH | LOW | P1 |
| Minimal TOML subset parser | HIGH | MEDIUM | P1 |
| go.mod `go`-directive version semantics decision | HIGH | LOW | P1 (documentation decision) |
| tsconfig.json confirmation | MEDIUM | LOW | P2 |
| settings.gradle rootProject.name | MEDIUM | LOW | P2 |
| Lockfile secondary confirmation | LOW | LOW | P2 |
| .csproj selection rule | LOW | MEDIUM | P2 |
| Framework detection | HIGH | HIGH | P3 (deferred milestone) |
| Extension-count fallback | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | GitHub Linguist | Snyk | mise/asdf | langforge (Go) | Understand-Anything | Our Approach |
|---------|-----------------|------|-----------|----------------|---------------------|--------------|
| Language detection | Per-file byte stats (8 strategies); **no manifests** | Ordered basename list, first-match | Runtime version files only (`.nvmrc`, `.tool-versions`, package.json, pyproject.toml) | Manifest list + extension classifier, registry | Manifest + LLM narrative | Ordered 7-marker list, first-match, Unknown fallback |
| Name | N/A | From dep tree root package | N/A | Per-ecosystem readers | Priority chain → dirname | Manifest name → folder |
| Version | N/A | From lockfiles/registry | Runtime pins (floors explicitly excluded) | Per-ecosystem version readers | From manifests | Raw manifest field, empty if dynamic/absent |
| Description | N/A | N/A | N/A | N/A | Manifest + README head (~10 lines) | Manifest → README first paragraph |
| Error behavior | Never (stats only) | Throws when no manifest | Silent no-op | Returns detection result | Narrative, never fails | `Unknown`, never error |
| Stdlib/offline | Ruby lib, offline | Node CLI, online | Rust CLI, offline | Go, offline | Node + LLM | Go stdlib-only, offline |
| Vendored-code hygiene | Full vendor/generated/docs exclusion | N/A (dependency scanner) | N/A | N/A | Ignore-list | ignoreDirs list (from deprecated sample) |

## Sources

- GitHub Linguist: [how-linguist-works.md](https://github.com/github-linguist/linguist/blob/main/docs/how-linguist-works.md), [repository.rb](https://github.com/github/linguist/blob/master/lib/linguist/repository.rb), [overrides.md](https://github.com/github-linguist/linguist/blob/master/docs/overrides.md) — MEDIUM (primary docs, verified)
- Snyk CLI source: [detect.ts](https://github.com/snyk/snyk/blob/master/src/lib/detect.ts) (DETECTABLE_FILES ordered list, first-match, basename→package-manager table), [package-managers.ts](https://github.com/snyk/cli/blob/6646233c/src/lib/package-managers.ts) — MEDIUM (primary source, verified)
- mise: [dev-tools](https://mise.jdx.dev/dev-tools/), [configuration.html](https://mise.jdx.dev/configuration.html) (idiomatic version files, floors-not-versions) — MEDIUM (primary docs, verified)
- oh-my-zsh nvm plugin: [nvm.plugin.zsh](https://github.com/ohmyzsh/ohmyzsh/blob/master/plugins/nvm/nvm.plugin.zsh) (.nvmrc chpwd autoload) — MEDIUM (primary source, verified)
- PEP 621 pyproject.toml: [peps.python.org/pep-0621](https://peps.python.org/pep-0621/), [packaging.python.org pyproject-toml spec](https://packaging.python.org/en/latest/specifications/pyproject-toml/) (name required, version required-or-dynamic, description optional) — MEDIUM (spec, verified)
- Composer schema: [getcomposer.org/doc/04-schema.md](https://getcomposer.org/doc/04-schema.md) (version optional, omit when VCS-derived), [packagist.org about](https://packagist.org/about?type=composer) (versions from tags) — MEDIUM (primary docs, verified)
- .NET SDK MSBuild properties: [learn.microsoft.com msbuild-props](https://learn.microsoft.com/en-us/dotnet/core/project-sdk/msbuild-props), [NuGet pack MSBuild](https://learn.microsoft.com/en-us/nuget/create-packages/creating-a-package-msbuild) (PackageId→AssemblyName, Version default 1.0.0, Description) — MEDIUM (primary docs, verified)
- langforge (Go detection lib): [rios0rios0/langforge](https://github.com/rios0rios0/langforge) (detection files + version files per ecosystem) — MEDIUM (repo README, verified)
- repo-scanner: [@codegeneai/repo-scanner](https://registry.npmjs.org/@codegeneai/repo-scanner) (runtime sources incl. go.mod/package.json#engines) — MEDIUM
- language-detect: [nickpending/llmcli-tools](https://github.com/nickpending/llmcli-tools/tree/main/packages/language-detect) (two-phase: markers → extension counts) — MEDIUM
- Understand-Anything project-scanner: [Lum1104/Understand-Anything](https://github.com/Lum1104/Understand-Anything) (name priority chain, README head fallback) — MEDIUM
- detect-stack skill: [luum-cognitive-os](https://github.com/Luum-Home/luum-cognitive-os) (name priority: package.json → go.mod → dirname) — LOW (single source, secondary)
- SherlockIO: [griffincancode/sherlockio](https://github.com/griffincancode/sherlockio) (two-stage filtering, lockfile exclusion) — LOW (single source)
- Go stdlib TOML absence: [pelletier/go-toml](https://github.com/pelletier/go-toml) (TOML v1.1.0, v2.4.0 2026-06), [golang/go#60791](https://github.com/golang/go/issues/60791) (struct-tag proposal, not TOML package) — MEDIUM (verified by absence across golang.org + ecosystem)
- Awesome-copilot stack-detection reference (manifest field map, runtime version sources): [github/awesome-copilot](https://github.com/github/awesome-copilot) — MEDIUM

---
*Feature research for: project_probe — best-effort project detection (7 languages)*
*Researched: 2026-09-28*
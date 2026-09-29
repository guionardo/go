# Requirements: go

**Defined:** 2026-09-28
**Core Value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.

## v1.7 Requirements — project_probe

Best-effort project detection: read a folder's contents and report language, name, version, and description across 7 languages. Stdlib-only, never fails — unknown content yields a typed Unknown value.

### Foundation

- [x] **FND-01**: Delete deprecated `project_detector/` sample in its own commit — it fails `go build ./...` (missing gs-dev, BurntSushi go.sum)
- [x] **FND-02**: `project_probe/` package skeleton with `doc.go` documenting the never-fail contract (error reserved for hard I/O failures; Unknown = nil error)
- [x] **FND-03**: `Probe(folder) (ProjectData, error)` entry point backed by private ordered detector registry with `(ProjectData, bool)` contract

### Detection

- [x] **DETC-01**: Ordered manifest-first cascade, first match wins, root-scoped only (no subdir probing)
- [x] **DETC-02**: Go detector — go.mod module → name, `go` directive → Version (documented as toolchain floor)
- [x] **DETC-03**: JS/TS detector — package.json name/version/description
- [x] **DETC-04**: PHP detector — composer.json name/version/description
- [x] **DETC-05**: Python detector — pyproject.toml `[project]` + legacy `[tool.poetry]`, PEP 621 fields
- [x] **DETC-06**: Rust detector — Cargo.toml `[package]` fields
- [x] **DETC-07**: C#/.NET detector — .csproj XML (namespace-agnostic, XMLName pattern)
- [ ] **DETC-08**: Java/Kotlin detector — pom.xml (parent version inheritance), settings.gradle `rootProject.name` fallback
- [x] **DETC-09**: Unknown folders → `LanguageUnknown` value, nil error

### Data Model

- [x] **DATA-01**: `ProjectData{Folder, Language, Name, Version, Description}` with independent per-field fallbacks
- [x] **DATA-02**: Name chain: manifest name → folder base
- [ ] **DATA-03**: Version: raw manifest string, empty when absent/dynamic, never fabricated
- [x] **DATA-04**: Description: manifest → README first-paragraph → empty

### Robustness

- [x] **ROBT-01**: Ignore-list hygiene (vendored/build/IDE dirs) so they never masquerade as markers
- [x] **ROBT-02**: Shared `readManifest` helper — size cap + BOM strip
- [x] **ROBT-03**: Unexported section-aware TOML-subset reader (~80 lines) for pyproject/Cargo; strict degrade-to-empty
- [x] **ROBT-04**: Stdlib-only imports; `path/filepath` not `path`; Windows-safe
- [ ] **ROBT-05**: No build-tool execution, no network, no symlink following, no version normalization (anti-features enforced)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Framework Detection

- **FRAM-01**: Detect framework from dependencies (package.json deps → React/Vue/Express; pyproject → Django/FastAPI; go.mod → Gin/Echo)
- **FRAM-02**: Replace TOML-subset reader with `pelletier/go-toml/v2` when dependency parsing justifies the dep

### Refinements

- **REFN-01**: tsconfig.json presence → distinct `typescript` language value
- **REFN-02**: settings.gradle `rootProject.name` for Java/Kotlin (already in scope) — build.gradle.kts → `kotlin` language value
- **REFN-03**: Lockfile secondary confirmation (package-lock.json, Cargo.lock, go.sum)
- **REFN-04**: Multiple `.csproj`/`.sln` selection rule for .NET monorepo layouts

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Executing build tools (gradle, dotnet, npm) | Anti-feature — reads folder contents only |
| Network calls | Anti-feature — offline detection |
| Symlink following / recursive discovery | Anti-feature — root-scoped only |
| Version normalization | Anti-feature — raw manifest strings, empty when absent |
| Extension-count/content-based detection (go-enry style) | Wrong abstraction — manifest-first is the ecosystem pattern |
| Framework detection | Deferred to v2 (FRAM-01) |
| Full TOML dependency | Deferred to v2 — stdlib-only constraint for v1.7 |
| Monorepo/workspace detection | Complex; not core to detection value |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FND-01 | Phase 10 | Complete |
| FND-02 | Phase 10 | Complete |
| FND-03 | Phase 10 | Complete |
| DETC-01 | Phase 10 | Complete |
| DETC-02 | Phase 11 | Complete |
| DETC-03 | Phase 11 | Complete |
| DETC-04 | Phase 11 | Complete |
| DETC-05 | Phase 12 | Complete |
| DETC-06 | Phase 12 | Complete |
| DETC-07 | Phase 13 | Complete |
| DETC-08 | Phase 13 | Pending |
| DETC-09 | Phase 10 | Complete |
| DATA-01 | Phase 10 | Complete |
| DATA-02 | Phase 11 | Complete |
| DATA-03 | Phase 14 | Pending |
| DATA-04 | Phase 11 | Complete |
| ROBT-01 | Phase 10 | Complete |
| ROBT-02 | Phase 10 | Complete |
| ROBT-03 | Phase 12 | Complete |
| ROBT-04 | Phase 10 | Complete |
| ROBT-05 | Phase 14 | Pending |

**Coverage:**

- v1 requirements: 21 total (Foundation 3, Detection 9, Data Model 4, Robustness 5)
- Mapped to phases: 21
- Unmapped: 0 ✓

---
*Requirements defined: 2026-09-28*
*Last updated: 2026-09-28 after roadmap creation*

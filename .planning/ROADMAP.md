# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- ✅ **v1.5 Self-Update** — Phase 4 (shipped 2026-07-21)
- ✅ **v1.6 Cache Dedup** — Phases 5-9 (shipped 2026-08-08)
- 🚧 **v1.7 Project Probe** — Phases 10-14 (in progress)

## Phases

<details>
<summary>✅ v1.4 Core Packages (Phase 1) — SHIPPED 2026-07-21</summary>

- [x] **Phase 1: Cache Package** - Generic Cache over 5 backends (3/3 plans) — completed 2026-07-21

</details>

<details>
<summary>✅ v1.5 Self-Update (Phase 4) — SHIPPED 2026-07-21</summary>

- [x] **Phase 4: Release self-update with swapper binary** - Self-update mechanism (3/3 plans) — completed 2026-07-21

</details>

<details>
<summary>✅ v1.6 Cache Dedup (Phases 5-9) — SHIPPED 2026-08-08</summary>

**Milestone Goal:** Eliminate duplicate setter work in cache misses and reduce round trips — via singleflight GetOrSet across all 5 providers, batch operations, and a benchmark suite.

- [x] **Phase 5: Shared singleflight helper** - Shared `singleflightGetOrSet` helper in the cache package: setter-only wrap, DoChan cancel variant, panic recovery, Get double-check (completed 2026-08-06)
- [x] **Phase 6: Provider integration** - All 5 providers delegate GetOrSet through the helper; error-prefix parity, delete-during-flight tombstone guard, race/TTL parity
- [x] **Phase 7: Batch operations** - MGet/MSet/MDel on all 5 providers with provider-appropriate batching (single-lock, GetMulti, pipelines, query batch) — completed 2026-08-08
- [x] **Phase 8: Benchmark suite** - Thundering-herd dedup and batch batching benchmarks, runnable via `make benchmark` — completed 2026-08-08
- [x] **Phase 9: Post-v1.6 cleanup** - Config HTTP endpoint, httptest mock ServeMux routing, Windows header matching fix — completed 2026-08-08

</details>

### 🚧 v1.7 Project Probe (In Progress)

**Milestone Goal:** A stdlib-only `project_probe` package that reads a folder's contents and reports the project's language, name, version, and description — best-effort, never failing.

- [x] **Phase 10: Package Foundation — API Contract + Repo Cleanup** - Never-fail Probe contract, ProjectData model, ordered detector registry, ignore list, shared readManifest; delete project_detector/ (completed 2026-09-29)
- [x] **Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback** - Go, JS/TS, PHP detectors with name/version/description chains and README first-paragraph fallback (completed 2026-09-29)
- [x] **Phase 12: TOML Subset + Python/Rust Detectors** - Section-aware TOML-subset reader + Python (pyproject/poetry) and Rust (Cargo) detectors (completed 2026-09-29)
- [ ] **Phase 13: XML Detectors — C#/.NET + Java/Kotlin** - .csproj and pom.xml/settings.gradle XML detectors, namespace-agnostic and root-scoped
- [ ] **Phase 14: Semantics, Hardening, and Release Polish** - Version-semantics decisions, anti-feature audit, fuzz + fixtures, docs, coverage gate

## Phase Details

### Phase 10: Package Foundation — API Contract + Repo Cleanup

**Goal**: The `project_probe` package exists with a documented never-fail contract, deterministic ordered registry, and shared safe-manifest helpers — and the broken `project_detector/` sample is gone.
**Depends on**: Nothing (v1.7 start; follows Phase 9)
**Requirements**: FND-01, FND-02, FND-03, DETC-01, DETC-09, DATA-01, ROBT-01, ROBT-02, ROBT-04
**Success Criteria** (what must be TRUE):

  1. `go build ./...` passes across the module — the deprecated `project_detector/` sample is deleted in its own commit and CI stays green.
  2. User calls `projectprobe.Probe(folder)` and receives `ProjectData{Folder, Language, Name, Version, Description}`: a missing/unreadable folder returns an error; an empty or unrecognized folder returns `LanguageUnknown` with nil error — the probe never fails on content.
  3. Probing the same folder repeatedly returns identical results — the detector cascade is deterministic, ordered, first-match-wins, and root-scoped only (no subdir probing).
  4. A folder containing only vendored/build/IDE directories (node_modules/, vendor/, .git/, dist/, .idea/) is reported Unknown — ignore-list hygiene.
  5. Manifest reads are size-capped (1 MB) and BOM-stripped with no panics on pathological input; all path handling uses filepath — Windows-safe, no `path` imports.

**Plans**: 3/3 plans complete

Plans:
**Wave 1**

- [x] 10-01-PLAN.md — Delete deprecated project_detector sample so the module builds (FND-01)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 10-02-PLAN.md — Probe contract tracer: model, sentinels, entry point, ordered registry, ignore gate, never-fail docs (FND-02, FND-03, DETC-01, DETC-09, DATA-01, ROBT-01, ROBT-04)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 10-03-PLAN.md — Shared readManifest helper: 1 MB cap + BOM strip, never-fail (ROBT-02)

### Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback

**Goal**: Go, JavaScript/TypeScript, and PHP projects are detected end-to-end, with name/version/description chains and README first-paragraph fallback.
**Depends on**: Phase 10
**Requirements**: DETC-02, DETC-03, DETC-04, DATA-02, DATA-04
**Success Criteria** (what must be TRUE):

  1. User probes a folder with go.mod and gets Language=Go, Name=module path, Version=go directive (raw string, documented as toolchain floor, not a release version); BOM and quoted module lines parse correctly.
  2. User probes a folder with package.json and gets Language=JavaScript with Name/Version/Description from the manifest.
  3. User probes a folder with composer.json and gets Language=PHP with Name/Version/Description from the manifest.
  4. Name falls back to the folder base when the manifest has no name; Description falls back to the README's first paragraph when the manifest has none, and is empty when no README exists.
  5. README fallback skips badges, tables of contents, and rst-style underline headings, extracting the first real paragraph.

**Plans**: 3/3 plans complete

Plans:
**Wave 1**

- [x] 11-01-PLAN.md — README description fallback helper (candidates + first-real-paragraph extraction) + WR-01 regular-file gate (DATA-04)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 11-02-PLAN.md — Go detector end-to-end through Probe + registry 7-slot nil literal (DETC-02, DATA-02, DATA-04)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 11-03-PLAN.md — JS/TS + PHP JSON detectors, cascade precedence, final registry positions (DETC-03, DETC-04, DATA-02, DATA-04)

### Phase 12: TOML Subset + Python/Rust Detectors

**Goal**: Python and Rust projects are detected via the unexported section-aware TOML-subset reader.
**Depends on**: Phase 11
**Requirements**: DETC-05, DETC-06, ROBT-03
**Success Criteria** (what must be TRUE):

  1. User probes a folder with pyproject.toml `[project]` and gets Language=Python with PEP 621 name/version/description.
  2. User probes a Poetry-managed folder (`[tool.poetry]`) and gets Language=Python with poetry name/version/description.
  3. User probes a folder with Cargo.toml `[package]` and gets Language=Rust with name/version/description; `version.workspace = true` yields an empty Version — never fabricated.
  4. Malformed or unsupported TOML (dotted keys, multiline strings, inline tables) degrades strictly to empty fields — Unknown or partial data, nil error, no panic.

**Plans**: 4/4 plans complete + 1 gap-closure plan

Plans:
**Wave 1**

- [x] 12-01-PLAN.md — TOML-subset reader readTOMLSection + strict-degrade matrix, global skip states (ROBT-03)
- [x] 12-04-PLAN.md — gap closure: quote-aware skip-state scanning (CR-01), restores SC4 / 12-01 truth 4 / 12-02 truth 5

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 12-02-PLAN.md — Python detector [project]/[tool.poetry] + registry slot 1, interim position flip (DETC-05)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 12-03-PLAN.md — Rust detector [package] + registry slot 4, final 5-of-7 state, doc.go refresh, cascade rows (DETC-06)

### Phase 13: XML Detectors — C#/.NET + Java/Kotlin

**Goal**: C#/.NET and Java/Kotlin projects are detected through XML manifests — namespace-agnostic and root-scoped.
**Depends on**: Phase 12
**Requirements**: DETC-07, DETC-08
**Success Criteria** (what must be TRUE):

  1. User probes a folder with .csproj and gets Language=C#/.NET with name/version extracted via XMLName local-name matching — correct with or without xmlns, and with BOM.
  2. User probes a folder with pom.xml and gets Language=Java with name/version; child modules inherit `<parent><version>` from the parent POM.
  3. User probes a folder with settings.gradle (rootProject.name) and no pom.xml and gets Java/Kotlin with name from the gradle file.
  4. A .csproj or pom.xml in a subdirectory never triggers detection for the parent folder — root-scoped markers only.

**Plans**: TBD

### Phase 14: Semantics, Hardening, and Release Polish

**Goal**: Version semantics are resolved across all 7 detectors; the package is hardened (fuzz, fixtures), documented, and meets coverage thresholds.
**Depends on**: Phase 13
**Requirements**: DATA-03, ROBT-05
**Success Criteria** (what must be TRUE):

  1. Version is always the raw manifest string across all detectors: go.mod `go` directive (documented toolchain floor), Maven parent inheritance, Cargo `version.workspace` and pyproject `dynamic` → empty; never normalized, never fabricated.
  2. Package-wide audit confirms anti-features are absent: no build-tool execution, no network calls, no symlink following, no version normalization in any code path.
  3. Fuzz targets seeded with real manifests run under normal `go test` — malformed JSON/XML/TOML inputs never panic and never crash the probe.
  4. The package ships complete: `make coverage-quick` passes, doc.go contract finalized, README package index row added, `go vet` and lint clean.

**Plans**: TBD

## Progress

**Execution Order:** Phases execute in numeric order: 10 → 11 → 12 → 13 → 14

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Cache Package | v1.4 | 3/3 | Complete | 2026-07-21 |
| 4. Self-Update | v1.5 | 3/3 | Complete | 2026-07-21 |
| 5. Shared singleflight helper | v1.6 | 3/3 | Complete | 2026-08-06 |
| 6. Provider integration | v1.6 | — | Complete | 2026-08-08 |
| 7. Batch operations | v1.6 | 4/4 | Complete | 2026-08-08 |
| 8. Benchmark suite | v1.6 | 3/3 | Complete | 2026-08-08 |
| 9. Post-v1.6 cleanup | v1.6 | 1/1 | Complete | 2026-08-08 |
| 10. Package Foundation | v1.7 | 3/3 | Complete    | 2026-09-29 |
| 11. Text/JSON Detectors | v1.7 | 3/3 | Complete    | 2026-09-29 |
| 12. TOML Subset + Python/Rust | v1.7 | 4/4 | Complete    | 2026-09-29 |
| 13. XML Detectors | v1.7 | 0/TBD | Not started | - |
| 14. Semantics, Hardening, Polish | v1.7 | 0/TBD | Not started | - |

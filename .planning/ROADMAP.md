# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- ✅ **v1.5 Self-Update** — Phase 4 (shipped 2026-07-21)
- ✅ **v1.6 Cache Dedup** — Phases 5-9 (shipped 2026-08-08)
- ✅ **v1.7 Project Probe** — Phases 10-14 (shipped 2026-09-30)

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

- [x] **Phase 5: Shared singleflight helper** - Shared `singleflightGetOrSet` helper in the cache package (3/3 plans) — completed 2026-08-06
- [x] **Phase 6: Provider integration** - All 5 providers delegate GetOrSet through the helper — completed 2026-08-08
- [x] **Phase 7: Batch operations** - MGet/MSet/MDel on all 5 providers — completed 2026-08-08
- [x] **Phase 8: Benchmark suite** - Thundering-herd dedup and batch benchmarks — completed 2026-08-08
- [x] **Phase 9: Post-v1.6 cleanup** - Config HTTP endpoint, httptest mock ServeMux routing, Windows header fix — completed 2026-08-08

</details>

<details>
<summary>✅ v1.7 Project Probe (Phases 10-14) — SHIPPED 2026-09-30</summary>

- [x] **Phase 10: Package Foundation — API Contract + Repo Cleanup** - Never-fail Probe contract, ProjectData model, ordered detector registry, ignore list, shared readManifest; delete project_detector/ (3/3 plans) — completed 2026-09-29
- [x] **Phase 11: Text/JSON Detectors — Go, JS/TS, PHP + README Fallback** - Go, JS/TS, PHP detectors with name/version/description chains and README first-paragraph fallback (3/3 plans) — completed 2026-09-29
- [x] **Phase 12: TOML Subset + Python/Rust Detectors** - Section-aware TOML-subset reader + Python (pyproject/poetry) and Rust (Cargo) detectors (4/4 plans) — completed 2026-09-29
- [x] **Phase 13: XML Detectors — C#/.NET + Java/Kotlin** - .csproj and pom.xml/settings.gradle XML detectors, namespace-agnostic and root-scoped (2/2 plans) — completed 2026-09-29
- [x] **Phase 14: Semantics, Hardening, and Release Polish** - Version-semantics decisions, anti-feature audit, fuzz + fixtures, docs, coverage gate (4/4 plans) — completed 2026-09-29

</details>

## Next Milestone

Roadmap for the next milestone will be created with `/gsd-new-milestone` (questioning → research → requirements → roadmap).

See `.planning/milestones/v1.7-ROADMAP.md` for the archived v1.7 detail.
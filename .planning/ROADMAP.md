# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- ✅ **v1.5 Self-Update** — Phase 4 (shipped 2026-07-21)
- ✅ **v1.6 Cache Dedup** — Phases 5-9 (shipped 2026-08-08)
- ✅ **v1.7 Project Probe** — Phases 10-14 (shipped 2026-09-30)
- ✅ **v1.8 SQLite Cache Backend** — Phases 15-17.1 (shipped 2026-10-27) — [archive](./milestones/v1.8-ROADMAP.md)

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

<details>
<summary>✅ v1.8 SQLite Cache Backend (Phases 15-17.1) — SHIPPED 2026-10-27</summary>

**Milestone goal:** an embedded, persistent SQLite-backed cache provider (`cache/sqlite`) — durable local caching for CLIs and small services with no external infrastructure, safe for multi-process access via WAL.

- [x] **Phase 15: Provider Foundation + Core Cache Semantics** - Constructor with three location modes, pinned pool, hardened DSN, primitive Cache parity, TTL expiry + sweeps, verified WAL pragmas, storage controls (3/3 plans) — completed 2026-10-08
- [x] **Phase 16: Batch Surface + Multi-Process Hardening** - MGet/MSet/MDel with v1.6 semantics; two-process WAL safety proven; bounded contention, race-clean, CI two-process E2E (5/5 plans) — completed 2026-10-26
- [x] **Phase 17: Delivery Hardening** - CGO-free three-OS CI matrix, no-override coverage gates, doc.go + runnable example + README, benchmark entries (3/3 plans) — completed 2026-10-26
- [x] **Phase 17.1: Error sentinel identity + MGet contract** - Single provider prefix, dual-sentinel `errors.Is`, honest `MGet` exception, sqlite in the shared conformance E2E (2/2 plans, inserted) — completed 2026-10-27

Full detail, decisions and evidence: [v1.8 archive](./milestones/v1.8-ROADMAP.md)

</details>

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Cache Package | v1.4 | 3/3 | Complete | 2026-07-21 |
| 4. Self-Update | v1.5 | 3/3 | Complete | 2026-07-21 |
| 5. Shared singleflight helper | v1.6 | 3/3 | Complete | 2026-08-06 |
| 6. Provider integration | v1.6 | — | Complete | 2026-08-08 |
| 7. Batch operations | v1.6 | 4/4 | Complete | 2026-08-08 |
| 8. Benchmark suite | v1.6 | 3/3 | Complete | 2026-08-08 |
| 9. Post-v1.6 cleanup | v1.6 | 1/1 | Complete | 2026-08-08 |
| 10. Package Foundation | v1.7 | 3/3 | Complete | 2026-09-29 |
| 11. Text/JSON Detectors | v1.7 | 3/3 | Complete | 2026-09-29 |
| 12. TOML Subset + Python/Rust | v1.7 | 4/4 | Complete | 2026-09-29 |
| 13. XML Detectors | v1.7 | 2/2 | Complete | 2026-09-29 |
| 14. Semantics, Hardening, Polish | v1.7 | 4/4 | Complete | 2026-09-29 |
| 15. Provider Foundation + Core Semantics | v1.8 | 3/3 | Complete | 2026-10-08 |
| 16. Batch + Multi-Process Hardening | v1.8 | 5/5 | Complete | 2026-10-26 |
| 17. Delivery Hardening | v1.8 | 3/3 | Complete | 2026-10-26 |
| 17.1 Error sentinel + MGet contract | v1.8 | 2/2 | Complete | 2026-10-27 |

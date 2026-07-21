# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- 📋 **v1.5 Self-Update** — Self-update with swapper binary (Phase 4)

## Phases

<details>
<summary>✅ v1.4 Core Packages (Phase 1) — SHIPPED 2026-07-21</summary>

- [x] Phase 1: Cache Package (3/3 plans) — completed 2026-07-21

</details>

### 📋 v1.5 Self-Update (Planned)

- [ ] Phase 4: Release self-update with swapper binary

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Cache Package | v1.4 | 3/3 | Complete | 2026-07-21 |
| 4. Self-Update | v1.5 | 0/3 | Not started | - |

### Phase 4: Release self-update with swapper binary

**Goal:** Provide a self-update mechanism — detect current version, download release artifact, verify checksums, swap binary via embedded swapper with rollback
**Requirements**: UPD-01, UPD-02, UPD-03, UPD-04, UPD-05, UPD-06, UPD-07, UPD-08
**Plans:** 3 plans

Plans:

- [ ] 04-01-PLAN.md — Detection + Download + Core API (version parsing, update check, download, checksum verification, functional options)
- [ ] 04-02-PLAN.md — Swapper Binary (atomic swap with backup/rollback, cross-platform exec, checksum re-verify)
- [ ] 04-03-PLAN.md — Integration + CLI (go:embed, self-update orchestrator, example CLI, Makefile targets)

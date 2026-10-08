# Roadmap: go

## Milestones

- ✅ **v1.4 Core Packages** — Phase 1 (shipped 2026-07-21)
- ✅ **v1.5 Self-Update** — Phase 4 (shipped 2026-07-21)
- ✅ **v1.6 Cache Dedup** — Phases 5-9 (shipped 2026-08-08)
- ✅ **v1.7 Project Probe** — Phases 10-14 (shipped 2026-09-30)
- 🚧 **v1.8 SQLite Cache Backend** — Phases 15-17 (in progress)

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

### 🚧 v1.8 SQLite Cache Backend (In Progress)

**Milestone Goal:** Add an embedded, persistent SQLite-backed cache provider (`cache/sqlite`) — durable local caching for CLIs and small services with no external infrastructure, safe for multi-process access via WAL.

- [x] **Phase 15: Provider Foundation + Core Cache Semantics** - Sqlite provider constructor (3 location modes, pinned pool, hardened DSN), primitive Cache surface with provider parity, TTL expiry + sweeps, verified WAL pragmas, storage lifecycle controls (completed 2026-10-08)
- [ ] **Phase 16: Batch Surface + Multi-Process Hardening** - MGet/MSet/MDel with v1.6 semantics; two-process WAL safety proven by spike; bounded contention + race-clean; CI two-process E2E
- [ ] **Phase 17: Delivery Hardening** - CGO-free three-OS CI matrix, no-override coverage gates, doc.go + runnable example + README row, benchmark entries

## Phase Details

### Phase 15: Provider Foundation + Core Cache Semantics

**Goal**: A working, durable `cache/sqlite` provider — `sqlite.New[K, V]` constructs a `cache.BatchCache[K, V]` in all three location modes; Get/Set/Delete/GetOrSet/Close match the five existing providers' contracts; TTL expiry and sweeps are enforced; WAL pragmas are verified on every pooled connection; and long-running storage controls are available.
**Depends on**: Nothing (v1.8 start; follows Phase 14 — builds on the shipped v1.6 cache architecture; `cache/` root and the five providers stay frozen)
**Requirements**: PROV-01, PROV-02, PROV-03, PROV-04, STOR-01, STOR-02, STOR-03, STOR-04, STOR-05, STOR-06, STOR-07, TTL-01, TTL-02, TTL-03, TTL-04
**Success Criteria** (what must be TRUE):

  1. Caller calls `sqlite.New[K, V](opts...)` and receives a `cache.BatchCache[K, V]` built on the sqlite provider + `cache.NewConcreteCache`, in three location modes — default `os.UserCacheDir()` + cache name, explicit path override, and `:memory:`; both file and memory modes pin the pool to exactly one connection; paths containing DSN metacharacters (`?`, `#`) are rejected.
  2. Get/Set/Delete/GetOrSet/Close match the five existing providers' contracts: `fmt.Sprint` keys, JSON values, misses wrapping `cache.ErrMiss`, context-aware SQL, idempotent `Close`, `cache/sqlite:` error prefix — and `GetOrSet` dedups through the shared `SingleflightGetOrSet` with no second dedup layer.
  3. Data survives restarts: reopening an existing cache file preserves unexpired entries; `Close` performs a clean shutdown (final checkpoint; `-wal`/`-shm` removed on last connection close); schema bootstrap is idempotent under `BEGIN IMMEDIATE` (`CREATE TABLE/INDEX IF NOT EXISTS`, fixed three-column schema).
  4. Expiry behaves with TTL parity: per-call TTL wins, zero/absent falls back to the provider default, none means no expiry — stored as an absolute UnixNano timestamp that survives restarts; expired entries are never returned and reads never delete rows; a best-effort sweep runs on open, and an optional periodic sweep interval (mem/postgres parity) reclaims entries in long-running processes.
  5. Every pooled connection carries the DSN-carried pragmas — `journal_mode=WAL` (file mode), `busy_timeout=5000`, `synchronous=NORMAL`, immediate transaction lock — verified by read-back (`journal_mode` is `wal` for file databases); callers can tune `wal_autocheckpoint` and trigger an explicit optimize (checkpoint and/or `VACUUM`) to reclaim disk space.

**Plans**: 3/3 plans complete
Plans:
**Wave 1**

- [x] 15-01-PLAN.md — Provider foundation: pinned dependency, constructor + location modes, primitive CRUD contracts, error taxonomy (tracer-led; batch placeholders with Phase 16 TODO)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 15-02-PLAN.md — TTL expiry, filter-only reads, sweep-on-open + opt-in periodic sweeper, durability lifecycle (sidecar removal, reopen persistence)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 15-03-PLAN.md — Storage controls (`Optimizable` checkpoint/vacuum, `WithAutoCheckpoint`), docs, example, README rows, repo-wide coverage gate

### Phase 16: Batch Surface + Multi-Process Hardening

**Goal**: The `BatchCache` surface is complete with v1.6 semantics, and the milestone's headline promise is proven — two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`.
**Depends on**: Phase 15
**Requirements**: BATCH-01, BATCH-02, BATCH-03, CONC-01, CONC-02, CONC-03, CONC-04, CONC-05
**Success Criteria** (what must be TRUE):

  1. `MGet` retrieves multiple keys via chunked `IN` queries — missing, expired, and undecodable entries are silently skipped (bounded chunk size); `MDel` deletes via chunked `IN` lists and is idempotent for missing keys.
  2. `MSet` writes all pairs in a single transaction with one prepared upsert (`INSERT ... ON CONFLICT DO UPDATE`), one TTL for the batch — and the batch is atomic: any error rolls back the whole batch.
  3. Two OS processes sharing one cache file can read and write without corruption or hard failure under contention — verified by the two-process spike (helper-process pattern), which is the phase exit criterion.
  4. Contention surfaces at `BEGIN` within `busy_timeout` as a bounded error (immediate transaction lock mode; no unbounded auto-retry), the provider passes `go test -race` under concurrent goroutines on the pinned single-connection pool, and package docs state WAL's constraints (local storage only, same-host sharing).
  5. A CI E2E test spawns two OS processes sharing one temp database and asserts no corruption and correct behavior under concurrent writes.

**Plans:** 4 plans

Plans:
**Wave 1**

- [ ] 16-01-PLAN.md — Two-process spike: raw vs bounded-retry policy arms on one cache file; 16-SPIKE-FINDINGS.md evidence record (D-11; CONC-01 exit-criterion precondition)
- [ ] 16-02-PLAN.md — Provider hardening: bounded open retry closes DI-15-01 (D-06) + journal_size_limit DSN (D-08) + CONC-02 bounded-contention proof

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 16-03-PLAN.md — Batch surface: chunked MGet/MDel + atomic prepared-upsert MSet (BATCH-01..03, D-01..D-05) + race-clean CONC-03

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 16-04-PLAN.md — Two-process E2E (contention + crash arms), 3-OS CI steps + race gate (D-09/D-10, CONC-05), CONC-04 docs

### Phase 17: Delivery Hardening

**Goal**: The package ships per repo standards — CGO-free builds and tests green across the three-OS CI matrix, coverage gates pass with no override, docs and a runnable example exist, and benchmarks register the backend against mem and per-key loops.
**Depends on**: Phase 16
**Requirements**: QUAL-01, QUAL-02, QUAL-03, QUAL-04
**Success Criteria** (what must be TRUE):

  1. The package builds and tests with `CGO_ENABLED=0` on Linux, macOS, and Windows (CI matrix green); `modernc.org/sqlite` v1.60.1 and the exact `modernc.org/libc` pin are in `go.mod`.
  2. Tests cover unit, `:memory:`, reopen/persistence, concurrent, and sweep scenarios; coverage gates pass with no `.testcoverage-quick.yml` override (package ≥80%, file ≥70%, total ≥75%).
  3. `doc.go` documents the provider (TTL, reopen persistence, `:memory:`, multi-process constraints), a runnable example exists, and the README package index is updated.
  4. Benchmark entries measure the sqlite backend against mem and per-key loops, following the v1.6 benchmark suite conventions.

**Plans**: TBD

## Progress

**Execution Order:** Phases execute in numeric order: 15 → 16 → 17

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
| 15. Provider Foundation + Core Semantics | v1.8 | 3/3 | Complete    | 2026-10-08 |
| 16. Batch + Multi-Process Hardening | v1.8 | 0/— | Not started | - |
| 17. Delivery Hardening | v1.8 | 0/— | Not started | - |

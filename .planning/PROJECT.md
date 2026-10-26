# go - Golang tools, examples, and packages

## What This Is

A collection of reusable Go utility packages — config, data structures, validators, and CLI helpers — authored by Guionardo and published as `github.com/guionardo/go`. It serves as both a personal toolkit and a public Go module.

## Core Value

Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.

## Business Context

<!-- OPTIONAL — only for monetized or customer-facing projects. Delete this section otherwise. -->

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

- ✓ Typed configuration provider with YAML profiles, env var overrides, and validation — `config/` — existing
- ✓ Generic `Set[T comparable]` with union, intersect, diff, filter, marshal, SQL scan — `set/` — existing
- ✓ Immutable `Fraction` type with arithmetic operations — `fraction/` — existing
- ✓ Generic ternary (`If`) and zero-value default (`Default`) helpers — `flow/` — existing
- ✓ CPF and CNPJ validation (Brazilian documents) — `br_docs/` — existing
- ✓ Cross-platform machine identifier (Linux, macOS, Windows) — `mid/` — existing
- ✓ File/directory path utilities (existence, creation, Go root detection) — `path_tools/` — existing
- ✓ Quoted shell argument parsing and case-insensitive env var lookup — `shell_tools/` — existing
- ✓ Time string parser with auto-prioritizing layout list — `time_tools/` — existing
- ✓ Reflection utilities for zero-value checking — `reflect_tools/` — existing
- ✓ HTTP mock server for tests with request matching from code or files — `httptest_mock/` — existing
- ✓ GitHub latest release fetcher with asset download and digest verification — `release/` — existing
- ✓ CI pipeline with golangci-lint, pre-commit, commitlint, coverage enforcement, vulncheck — existing
- ✓ Generic `Cache[K, V]` abstraction over 5 backends — `cache/` — v1.0: in-memory, Redis, Memcache, Postgres, Valkey
- ✓ `SingleflightGetOrSet[K, V]` shared helper deduplicating concurrent GetOrSet misses — `cache/` — v1.6
- ✓ Provider integration — all 5 providers delegate GetOrSet through the shared helper — `cache/` — v1.6
- ✓ `BatchCache[K, V]` with MGet/MSet/MDel across all 5 providers — `cache/` — v1.6
- ✓ Benchmark suite quantifying dedup and batching wins — `cache/` — v1.6
- ✓ Complete self-update mechanism with version detection, SHA256 verification, atomic swap, and relaunch — `release/` — v1.5
- ✓ `project_probe` package foundation — never-fail `Probe` contract, `ProjectData` model, `Language` type, error sentinels, ordered detector registry, ignore list, `readManifest` — `project_probe/` — v1.7 (Phase 10)
- ✓ Go, JavaScript/TypeScript, and PHP detectors — go.mod/package.json/composer.json parsing, README first-paragraph fallback, name/description chains — `project_probe/` — v1.7 (Phase 11)
- ✓ TOML-subset reader + Python/Rust detectors — `readTOMLSection` with quote-aware skip states (strict degrade, never fabricates), pyproject.toml [project]/[tool.poetry], Cargo.toml [package] — `project_probe/` — v1.7 (Phase 12)
- ✓ All 7 detectors live — Go, Python, C#/.NET, JS/TS, Rust, Java/Kotlin, PHP; C#/.NET via .csproj (XMLName namespace-agnostic), Java/Kotlin via pom.xml (parent version inheritance) + settings.gradle fallback — `project_probe/` — v1.7 (Phase 13)
- ✓ SQLite cache provider core and batch surface — `sqlite.New[K,V]` (3 location modes, pinned 1-conn pool, hardened DSN), primitive `Cache` parity, TTL expiry + sweeps, storage controls, chunked `MGet`/`MDel` + single-transaction `MSet`, and proven multi-process WAL safety (bounded busy-only open retry, `journal_size_limit`, 3-OS two-process E2E) — `cache/sqlite/` — v1.8 (Phases 15–16)

### Active

<!-- Current scope. Building toward these. -->

- [ ] SQLite provider delivery hardening — contributor docs (`doc.go`), runnable example, README row, benchmarks vs mem/per-key loops, no-override coverage gates, CGO-free 3-OS matrix (v1.8 Phase 17)
- [ ] String utilities package (v2 backlog — truncation, padding, join/split)
- [ ] Retry package with backoff strategies and jitter support (v2 backlog)
- [ ] Framework detection for project_probe (v2 — dependencies-based)

## Current Milestone: v1.8 SQLite Cache Backend

**Goal:** Add an embedded, persistent SQLite-backed cache provider (`cache/sqlite`) — durable local caching for CLIs and small services with no external infrastructure, safe for multi-process access via WAL.

**Target features:**
- SQLite provider implementing the existing `cacher[K, V]` primitive + `BatchCache[K, V]` surface (Get/Set/Delete/GetOrSet/Close, MGet/MSet/MDel) via `NewConcreteCache`
- Pure-Go `modernc.org/sqlite` driver (CGO-free; Linux/macOS/Windows CI)
- Multi-process safety: WAL journal mode + `busy_timeout`
- Location modes: default `os.UserCacheDir` path via cache name; `:memory:` when path empty; explicit path override
- TTL expiry: lazy delete on read + sweep of expired rows on open
- JSON values / `fmt.Sprint` key parity with existing providers; tests, coverage gates, doc.go, example

## Current State

**v1.7 Project Probe — SHIPPED 2026-09-30**

The `project_probe` package ships complete: never-fail `Probe(folder)` detecting all 7 languages (Go, Python, C#/.NET, JS/TS, Rust, Java/Kotlin, PHP) with name/version/description chains, README first-paragraph fallback, strict-degrade TOML reader (quote-aware skip states — SC4 fabrication gap closed), fuzz-hardened (3 targets, 20-seed corpus), doc.go Version-semantics contract, and the anti-feature audit. 820+ tests across 25 packages; `make coverage-quick` green (80.7%); UAT 12/12; security 16/16 threats closed.

**Backlog (deferred from earlier milestones):** String utilities package; Retry package with backoff strategies and jitter support

**Backlog (deferred from earlier milestones):** String utilities package; Retry package with backoff strategies and jitter support

## Completed Milestones

### v1.6 Cache Dedup — completed 2026-08-08

Eliminated duplicate setter work in cache misses and reduced round trips via singleflight GetOrSet across all 5 providers, batch operations (MGet/MSet/MDel), and a benchmarking suite.

**Key results:**
- Singleflight dedup: setter runs exactly once for N concurrent misses (vs N× without)
- Batch MGet: 10-17% faster than per-key loop; MDel: up to 49% faster
- All 5 providers use optimal strategies (pipeline, GetMulti, SendBatch, single-lock)

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

| Feature | Reason |
|---------|--------|
| Slices utility package | Go 1.26 stdlib `slices` package covers common operations — not needed |

## Current State

**v1.8 — SQLite Cache Backend** (IN PROGRESS: Phase 15 ✓, Phase 16 ✓, Phase 17 pending)

`cache/sqlite` is functionally complete: pure-Go `modernc.org/sqlite` provider, three location modes, pinned single-connection pool, hardened DSN with pragma read-back, TTL expiry + sweeps, `Optimizable` storage controls, the full `BatchCache` surface (chunked MGet/MDel, atomic MSet), and multi-process WAL safety proven by a two-process E2E running on the 3-OS CI matrix. Phase 16 UAT 1/1 with G-16-1 reconciled in CI (PR #51); security 13/13 threats closed. Remaining: Phase 17 delivery hardening (docs/example/README, benchmarks, CGO-free matrix, no-override coverage).

**Previous: v1.7 — Project Probe** (SHIPPED 2026-09-30)

The `project_probe` package ships complete: never-fail `Probe(folder)` detecting all 7 languages with name/version/description chains, README first-paragraph fallback, strict-degrade TOML reader (quote-aware skip states), fuzz-hardened, doc.go Version-semantics contract, anti-feature audit. 820+ tests / 25 packages; `make coverage-quick` green (80.7%); UAT 12/12; security 16/16 threats closed.

**Previous: v1.6 — Cache Dedup** (shipped 2026-08-08)

All five cache providers now share a common `cacher` / `concreteCache` architecture with singleflight-wrapped `GetOrSet`, `BatchCache[K,V]` interface with MGet/MSet/MDel using provider-optimal strategies, and a benchmark suite proving the dedup and batching wins. Built on 9 validated spikes from 2026-08-06.

**Next:** String utilities or retry package — see Active requirements.

**Previous: v1.5 — Self-Update** (shipped 2026-07-21)

The `release` package now provides a complete self-update mechanism: version detection from build info, GitHub release checking, platform-specific asset download with SHA256 verification, atomic binary replacement via an embedded swapper (cross-platform), and automatic relaunch. All 21 library packages have proper `doc.go` documentation files. The main README was restructured with a package index table.

**Previous: v1.4 — Core Packages** (shipped 2026-07-21)

Shipped the generic `Cache[K, V]` package with 5 backends. ~13 utility packages with 6,600+ lines of code across the cache subsystem.

**Tech stack:** Go 1.26, testcontainers-go for E2E, go-redis/v9, valkey-go, gomemcache, pgx/v5, hashicorp/go-version

## Context

This is a personal Go monorepo of utility packages published as `github.com/guionardo/go`. The codebase follows Go standard library idioms with minimal external dependencies. Testing uses `testify` assertions with `httptest` for HTTP tests. CI enforces 95% total coverage, conventional commits, and comprehensive linting.

## Constraints

- **[Language]**: Go 1.26 only — all packages must follow Go stdlib idioms
- **[Dependencies]**: Minimal external dependencies — prefer stdlib solutions
- **[Compatibility]**: Must support Linux, macOS, and Windows (tested in CI matrix)
- **[Quality]**: Must maintain 95%+ total test coverage and pass all linters

## Key Decisions

<!-- Decisions that constrain future work. Add throughout project lifecycle. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go stdlib over frameworks | Keep dependencies minimal for a utility library | ✓ Good |
| Monorepo of independent packages | Each package is usable independently via `go get` | ✓ Good |
| Conventional commits + pre-commit | Enforce consistent commit history and code quality | ✓ Good |
| Generic Cache interface over 5 backends | Swap providers without code changes; memory cache for zero-dep testing | ✓ Good (v1.0) |
| Self-update via embedded swapper | Spawn → exit → swap → exec avoids file-in-use locks on Windows | ✓ Good (v1.5) |
| hashicorp/go-version for semver | Replaces custom parser; handles prereleases, pseudo-versions, build metadata | ✓ Good (v1.5) |
| Two-phase SHA256 verification | go-digest at download + stdlib at swap protects against corruption mid-flight | ✓ Good (v1.5) |
| singleflight wraps only the setter | Dedups concurrent misses without locking the fast-path Get | ✓ Validated (v1.6 spikes) |
| Shared GetOrSet helper over per-provider bodies | Eliminates 5× duplicated miss→setter→Set code | ✓ Validated (v1.6 spikes) |
| Never-fail Probe contract (error only for hard I/O) | Best-effort detection; content never fails | ✓ Good (v1.7, Phase 10) |
| syscall errno → sentinel mapping (ENOENT/ENOTDIR/EACCES/EPERM) | Cross-platform error discrimination without string matching | ✓ Good (v1.7, Phase 10) |
| Typed `Language` constants with display values | Discoverable API, switchable, extensible (v2 typescript) | ✓ Good (v1.7, Phase 10) |
| `Folder` = filepath.Clean(as-given), never absolutized | Deterministic tests, Windows-safe, caller gets what they asked for | ✓ Good (v1.7, Phase 10) |
| Panic-recovery at registry dispatch | A panicking detector is a non-match, never an error (cache callSetter precedent) | ✓ Good (v1.7, Phase 10) |
| readManifest 1 MB cap + BOM strip, `([]byte, bool)` | Never-fail contract; no panic on pathological input | ✓ Good (v1.7, Phase 10) |
| go.mod `go` directive as Version (toolchain floor, raw) | Never fabricated or normalized; documented semantics | ✓ Good (v1.7, Phase 11) |
| Always-JavaScript for package.json | Distinct `typescript` value deferred to v2 (REFN-01) | ✓ Good (v1.7, Phase 11) |
| README first-real-paragraph fallback (README.md → README.rst → README) | Description chain when manifest lacks one; skips badges/TOC/headings | ✓ Good (v1.7, Phase 11) |
| README extraction skip predicates | Badges, TOC lists, rst/setext underline headings, ATX, HTML comments | ✓ Good (v1.7, Phase 11) |
| `_javascript` filename over `_js` | `_js` suffix is a legacy GOARCH build constraint (silently excludes from non-js builds) | ✓ Good (v1.7, Phase 11) |
| Quote-aware skip-state scanning in TOML reader | Whole-line delimiter detection fabricated data from malformed TOML (SC4); closesMultiLine/clearsBracket/opensMultiLine + pendingMLS fix it | ✓ Good (v1.7, Phase 12, gap closure) |
| `version.workspace = true` → empty Version (no resolution) | Dotted-key classification degrades naturally; never fabricated, never resolved | ✓ Good (v1.7, Phase 12) |
| Pinned 1-connection pool for both file and `:memory:` modes | The single most important correctness decision — eliminates intra-process contention on the shared file | ✓ Good (v1.8, Phase 15) |
| Bounded busy-only open retry (5 ms backoff / 5 s budget, `Code()&0xff==5`) at the connection-establishment seam | Cross-process WAL conversion can return immediate SQLITE_BUSY; retry only there, never on ordinary queries | ✓ Good (v1.8, Phase 16, CONC-01/02 — 30/30 policy arm vs 8/10 raw failures) |
| Two-process E2E on all three OSes; `-race` gated to non-Windows runners | Proves the milestone's headline promise in CI; Windows lacks a C toolchain for `-race` in this repo | ✓ Good (v1.8, Phase 16, CONC-05) |
| Close-then-remove on download error paths (`f.Close()` before `removeFile`), cleanup failure surfaced via `errors.Join` | Windows denies `DeleteFile` on an open handle (no `FILE_SHARE_DELETE`); POSIX semantics masked the leak that broke CI | ✓ Good (v1.8, Phase 16 gap G-16-1) |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-10-26 after Phase 16 (Batch Surface + Multi-Process Hardening) completed*

# go - Golang tools, examples, and packages

## What This Is

A collection of reusable Go packages — configuration, data structures, validators, caching, project detection, and CLI self-update — authored by Guionardo and published as `github.com/guionardo/go`. It serves as both a personal toolkit and a public Go module.

## Core Value

Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.

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
- ✓ Quoted shell argument parsing and environment lookup — `shell_tools/` — existing
- ✓ Time string parser with auto-prioritizing layout list — `time_tools/` — existing
- ✓ Reflection utilities for zero-value checking — `reflect_tools/` — existing
- ✓ HTTP mock server for tests with request matching from code or files — `httptest_mock/` — existing
- ✓ GitHub latest release fetcher with asset download and digest verification — `release/` — existing
- ✓ CI pipeline with golangci-lint, pre-commit, commitlint, coverage enforcement, vulncheck — existing
- ✓ Generic `Cache[K, V]` abstraction over 5 backends — `cache/` — v1.0
- ✓ `SingleflightGetOrSet[K, V]` shared helper deduplicating concurrent GetOrSet misses — `cache/` — v1.6
- ✓ `BatchCache[K, V]` with MGet/MSet/MDel across all 5 providers — `cache/` — v1.6
- ✓ Benchmark suite quantifying dedup and batching wins — `cache/` — v1.6
- ✓ Complete self-update mechanism with version detection, SHA256 verification, atomic swap, and relaunch — `release/` — v1.5
- ✓ `project_probe` — never-fail `Probe`, 7 language detectors, README fallback, strict-degrade TOML reader — `project_probe/` — v1.7
- ✓ SQLite cache provider — `sqlite.New[K,V]` (3 location modes, pinned 1-conn pool, hardened DSN), primitive `Cache` parity, TTL + sweeps, storage controls, full `BatchCache` surface, multi-process WAL safety with a bounded busy-only open retry — `cache/sqlite/` — v1.8 (Phases 15–16)
- ✓ Delivery hardening — CGO-free three-OS CI matrix that asserts its own claim, no-override coverage gates, `doc.go` + runnable example + package README, benchmarks against mem and per-key loops — `cache/sqlite/` — v1.8 (Phase 17)
- ✓ Honest error contract — one provider prefix per error, `sqlite.ErrClosed` asserted through both `sqlite.ErrClosed` and `cache.ErrClosed`, `MGet`'s deferred-error exception documented and logged at the level it deserves, sqlite in the shared conformance E2E — `cache/` — v1.8 (Phase 17.1)
- ✓ Documentation regime — root README as an index, 18 package READMEs, 23 of 26 packages with `// Output:`-asserted examples, and drift made a defect in `AGENTS.md` — repo-wide — v1.8 closeout

### Active

<!-- Current scope. Building toward these. -->

- [ ] String utilities package (v2 backlog — truncation, padding, join/split)
- [ ] Retry package with backoff strategies and jitter support (v2 backlog)
- [ ] Framework detection for `project_probe` (v2 — dependencies-based)

### Out of Scope

| Feature | Reason |
|---------|--------|
| Slices utility package | Go 1.26 stdlib `slices` package covers common operations — not needed |
| Arbitrary SQL/ORM surface on `cache/sqlite` | Turns a cache backend into a database wrapper; breaks provider-swap abstraction |
| Retry middleware beyond `busy_timeout` | Hides pathological lock holders and multiplies latency; errors must surface |
| Cross-host / network-filesystem cache sharing | WAL needs shared memory — use redis/valkey/postgres for shared infrastructure |

## Current State

**v1.8 SQLite Cache Backend — SHIPPED 2026-10-27**

`cache/sqlite` ships complete: a pure-Go `modernc.org/sqlite` provider with three location modes, a pinned single-connection pool, a hardened DSN with pragma read-back, TTL expiry plus sweeps, `Optimizable` storage controls, the full `BatchCache` surface, and multi-process WAL safety proven by a two-process E2E running on the three-OS CI matrix. The delivery phase added a CGO-free matrix that asserts its own claim, benchmarks placed against mem and per-key loops, and provider docs. Phase 17.1 then made the error contract honest (single prefix, dual-sentinel `errors.Is`, documented `MGet` exception) and registered sqlite in the shared conformance harness.

**Quality:** 25,914 lines of Go across 26 packages · coverage 82.4% total, `cache/sqlite` 92.7% (no override) · 3-OS CI green · 23 of 26 packages carry runnable examples · milestone audit `passed`, 27/27 requirements, 0 open artifacts.

**Next:** no milestone is active — start one with `/gsd-new-milestone` (the v2 backlog above is the natural seed).

## Completed Milestones

### v1.8 SQLite Cache Backend — 2026-10-27
Added an embedded, durable cache backend with no external infrastructure, and proved its multi-process safety in CI. 4 phases, 13 plans, 34 tasks; 7 merged PRs over 20 days. Archive: `milestones/v1.8-ROADMAP.md`.

### v1.7 Project Probe — 2026-09-30
Never-fail project detection across 7 languages with manifest chains, README fallback, and a strict-degrade TOML reader. Fuzz-hardened, 820+ tests.

### v1.6 Cache Dedup — 2026-08-08
Singleflight dedup of concurrent misses, batch operations across all 5 providers, and the benchmark suite that quantified both (MGet 10-17% faster; MDel up to 49%).

### v1.5 Self-Update — 2026-07-21
Version detection, GitHub release checking, SHA256 verification, atomic swap via an embedded swapper, relaunch.

### v1.4 Core Packages — 2026-07-21
The generic `Cache[K, V]` package with 5 backends and ~13 utility packages.

**Tech stack:** Go 1.26 · `modernc.org/sqlite` (CGO-free) · go-redis/v9 · valkey-go · gomemcache · pgx/v5 · hashicorp/go-version · testcontainers-go for E2E

## Context

A personal Go monorepo published as `github.com/guionardo/go`, following Go standard-library idioms with minimal external dependencies. Tests use `testify`, `httptest`, and real temp directories; `make coverage-quick` gates at packages ≥80% / files ≥70% / total ≥75% with no per-package overrides for `cache/sqlite`. CI runs Linux, macOS and Windows with `CGO_ENABLED=0`, plus a two-process E2E and a race-detector lane where a C toolchain exists.

Documentation is a contract, not decoration: every package carries a `doc.go`, a runnable `example_test.go` with `// Output:` assertions, and its own `README.md`; the root README is an index. See `AGENTS.md` § Documentation.

## Constraints

- **[Language]**: Go 1.26 only — all packages follow Go stdlib idioms
- **[Dependencies]**: Minimal external dependencies — prefer stdlib solutions; the cache's sixth backend stays CGO-free
- **[Compatibility]**: Linux, macOS, and Windows (three-OS CI matrix)
- **[Quality]**: `make coverage-quick` green before every commit; golangci-lint at zero issues; Conventional Commits

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go stdlib over frameworks | Keep dependencies minimal for a utility library | ✓ Good |
| Monorepo of independent packages | Each package usable independently via `go get` | ✓ Good |
| Conventional commits + pre-commit | Consistent history and code quality | ✓ Good |
| Generic Cache interface over 6 backends | Swap providers without code changes | ✓ Good (v1.0, extended v1.8) |
| Self-update via embedded swapper | Spawn → exit → swap → exec avoids file-in-use locks on Windows | ✓ Good (v1.5) |
| Two-phase SHA256 verification | go-digest at download + stdlib at swap protects against mid-flight corruption | ✓ Good (v1.5) |
| singleflight wraps only the setter | Dedups concurrent misses without locking the fast-path Get | ✓ Validated (v1.6) |
| Never-fail Probe contract | Best-effort detection; content never fails | ✓ Good (v1.7) |
| Pinned 1-connection pool (file and `:memory:`) | The single most important correctness decision — eliminates intra-process file contention | ✓ Good (v1.8, Phase 15) |
| Bounded busy-only open retry (5 ms / 5 s, busy code at the connection seam) | Cross-process WAL conversion can return immediate SQLITE_BUSY; retry only there | ✓ Good (v1.8, CONC-01/02: 30/30 vs 8/10 raw) |
| Two-process E2E on three OSes; `-race` gated to non-Windows | Proves the headline promise in CI; Windows lacks a C toolchain for `-race` here | ✓ Good (v1.8, Phase 16) |
| Close-then-remove on download error paths | Windows denies `DeleteFile` on an open handle; POSIX masked the leak | ✓ Good (v1.8, G-16-1) |
| Sentinels wrap the cache-level sentinel; one prefix per error | `errors.Is` works for both `sqlite.ErrClosed` and `cache.ErrClosed` without a custom type | ✓ Good (v1.8, Phase 17.1) |
| `MGet` keeps its frozen signature and documents the exception | The interface is frozen; returning `nil` or adding `MGetErr` would change a published contract for a logging concern | ✓ Good (v1.8, Phase 17.1) |
| Root README is an index; detail lives in package READMEs | Keeps the entry point scannable and each package's docs next to its code | ✓ Good (v1.8 closeout) |
| One `cache/README.md` for the backend matrix | The backends differ by driver, not by story; six READMEs would be six things to keep true | ✓ Good (v1.8 closeout) |
| Benchmarks must measure what they claim | A bounded key ring and a real miss path; growing keys made file-mode runs measure the database, not the operation | ✓ Good (v1.8 closeout) |
| After a squash-merge, `git reset --hard origin/main` | Planning commits land on `main` before the phase branch, so the squash rewrites their SHAs and `pull` cannot fast-forward | ✓ Good (recorded in AGENTS.md) |

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
*Last updated: 2026-10-27 at the v1.8 milestone close*

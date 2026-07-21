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

### Active

<!-- Current scope. Building toward these. -->

- [ ] String utilities package (truncation, padding, join/split)
- [ ] Retry package with backoff strategies and jitter support

### Out of Scope

<!-- Explicit boundaries. Includes reasoning to prevent re-adding. -->

| Feature | Reason |
|---------|--------|
| Slices utility package | Go 1.26 stdlib `slices` package covers common operations — not needed |

## Current State

**v1.0 — Core Packages** (shipped 2026-07-21)

The first planned milestone shipped the generic `Cache[K, V]` package with 5 backends. The codebase has ~13 utility packages with 6,600+ lines of code across the cache subsystem. CI enforces linting, conventional commits, and E2E test separation via build tags.

**Tech stack:** Go 1.26, testcontainers-go for E2E, go-redis/v9, valkey-go, gomemcache, pgx/v5

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
*Last updated: 2026-07-21 after v1.0 milestone*

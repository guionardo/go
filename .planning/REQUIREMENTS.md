# Requirements: go - Golang tools, examples, and packages

**Defined:** 2026-07-21
**Core Value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently

## v1 Requirements

### Strings Package

- **STRNG-01**: Package provides string truncation utilities
- **STRNG-02**: Package provides padding/alignment utilities
- **STRNG-03**: Package provides join/split utilities

### Retry Package

- **RETRY-01**: Package provides retry with backoff strategies
- **RETRY-02**: Package provides jitter support

### Cache Package

- [x] **CACHE-01**: Package provides generic `Cache[K, V]` interface with `Get`, `Set`, `Delete`, `GetOrSet` methods accepting `context.Context`
- [x] **CACHE-02**: `Set` accepts per-key TTL (optional); falls back to provider-level default TTL if not specified
- [x] **CACHE-03**: All errors are wrapped and returned (not swallowed)
- [x] **CACHE-04**: Provides in-memory provider (`cache/mem`)
- [x] **CACHE-05**: Provides Redis provider (`cache/redis`)
- [x] **CACHE-06**: Provides Memcache provider (`cache/memcache`)
- [x] **CACHE-07**: Provides Postgres provider (`cache/postgres`)
- [x] **CACHE-08**: Provides Valkey provider (`cache/valkey`)
- [x] **CACHE-09**: Each provider lives in its own sub-package importable independently
- [x] **CACHE-10**: Package includes runnable examples
- [x] **CACHE-11**: Package follows project conventions (lint, 95%+ test coverage, naming)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Monadic types (Option, Either, Result) | Over-engineered for Go; community consensus against it |
| DI container | Stdlib interfaces + constructor injection is sufficient |
| Logger abstraction | `log/slog` in stdlib covers this |
| Full lodash clone | Contradicts minimal-dependency constraint; `samber/lo` already exists |
| CLI framework | This is a library module, not a CLI tool |
| Multi-module split | Premature at current scale (~12 packages) |
| Slices utility package | Not needed — Go 1.26 stdlib `slices` package covers common operations |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CACHE-01 | Phase 1 | Complete |
| CACHE-02 | Phase 1 | Complete |
| CACHE-03 | Phase 1 | Complete |
| CACHE-04 | Phase 1 | Complete |
| CACHE-05 | Phase 1 | Complete |
| CACHE-06 | Phase 1 | Complete |
| CACHE-07 | Phase 1 | Complete |
| CACHE-08 | Phase 1 | Complete |
| CACHE-09 | Phase 1 | Complete |
| CACHE-10 | Phase 1 | Complete |
| CACHE-11 | Phase 1 | Complete |

**Coverage:**

- v1 requirements: 13 total
- Mapped to phases: 11
- Unmapped: 2 ✓ (Strings, Retry — future scope)

---
*Requirements defined: 2026-07-21*
*Last updated: 2026-07-21 after initial definition*

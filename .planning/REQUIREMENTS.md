# Requirements: go - Golang tools, examples, and packages

**Defined:** 2026-07-21
**Core Value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently

## v1 Requirements

### Slices Package

- [ ] **SLICE-01**: Package provides generic `Filter` function for slices
- [ ] **SLICE-02**: Package provides generic `Map` function for slices
- [ ] **SLICE-03**: Package provides generic `Reduce` function for slices
- [ ] **SLICE-04**: Package provides generic `Contains` function for slices
- [ ] **SLICE-05**: Package provides generic `Chunk` function for slices
- [ ] **SLICE-06**: Package provides generic `Uniq` function for slices
- [ ] **SLICE-07**: Package provides generic `Shuffle` function for slices
- [ ] **SLICE-08**: Package has `doc.go` with package documentation
- [ ] **SLICE-09**: Package has `example_test.go` with runnable examples
- [ ] **SLICE-10**: Package follows project conventions (lint, test coverage, naming)

## v2 Requirements

### Strings Package

- **STRNG-01**: Package provides string truncation utilities
- **STRNG-02**: Package provides padding/alignment utilities
- **STRNG-03**: Package provides join/split utilities

### Retry Package

- **RETRY-01**: Package provides retry with backoff strategies
- **RETRY-02**: Package provides jitter support

### Cache Package

- [ ] **CACHE-01**: Package provides generic `Cache[K, V]` interface with `Get`, `Set`, `Delete`, `GetOrSet` methods accepting `context.Context`
- [ ] **CACHE-02**: `Set` accepts per-key TTL (optional); falls back to provider-level default TTL if not specified
- [ ] **CACHE-03**: All errors are wrapped and returned (not swallowed)
- [ ] **CACHE-04**: Provides in-memory provider (`cache/mem`)
- [ ] **CACHE-05**: Provides Redis provider (`cache/redis`)
- [ ] **CACHE-06**: Provides Memcache provider (`cache/memcache`)
- [ ] **CACHE-07**: Provides Postgres provider (`cache/postgres`)
- [ ] **CACHE-08**: Provides Valkey provider (`cache/valkey`)
- [ ] **CACHE-09**: Each provider lives in its own sub-package importable independently
- [ ] **CACHE-10**: Package includes runnable examples
- [ ] **CACHE-11**: Package follows project conventions (lint, 95%+ test coverage, naming)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Monadic types (Option, Either, Result) | Over-engineered for Go; community consensus against it |
| DI container | Stdlib interfaces + constructor injection is sufficient |
| Logger abstraction | `log/slog` in stdlib covers this |
| Full lodash clone | Contradicts minimal-dependency constraint; `samber/lo` already exists |
| CLI framework | This is a library module, not a CLI tool |
| Multi-module split | Premature at current scale (~12 packages) |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SLICE-01 | Phase 1 | Pending |
| SLICE-02 | Phase 1 | Pending |
| SLICE-03 | Phase 1 | Pending |
| SLICE-04 | Phase 1 | Pending |
| SLICE-05 | Phase 1 | Pending |
| SLICE-06 | Phase 1 | Pending |
| SLICE-07 | Phase 1 | Pending |
| SLICE-08 | Phase 1 | Pending |
| SLICE-09 | Phase 1 | Pending |
| SLICE-10 | Phase 1 | Pending |
| CACHE-01 | Phase 2 | Pending |
| CACHE-02 | Phase 2 | Pending |
| CACHE-03 | Phase 2 | Pending |
| CACHE-04 | Phase 2 | Pending |
| CACHE-05 | Phase 2 | Pending |
| CACHE-06 | Phase 2 | Pending |
| CACHE-07 | Phase 2 | Pending |
| CACHE-08 | Phase 2 | Pending |
| CACHE-09 | Phase 2 | Pending |
| CACHE-10 | Phase 2 | Pending |
| CACHE-11 | Phase 2 | Pending |

**Coverage:**
- v1 requirements: 10 total
- v2 requirements: 13 total
- Mapped to phases: 23
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-21*
*Last updated: 2026-07-21 after initial definition*

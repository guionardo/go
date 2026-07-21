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

**Coverage:**
- v1 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-21*
*Last updated: 2026-07-21 after initial definition*

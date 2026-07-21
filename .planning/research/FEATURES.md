# Feature Landscape: Go Utility Monorepo Organization

**Domain:** Go utility libraries / developer toolkits  
**Researched:** 2026-07-21

## Table Stakes

Features a well-organized Go utility monorepo must have. Missing these = project feels unprofessional.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Single `go.mod` | Simple, standard | Trivial | Already done |
| `internal/` for private packages | Go standard practice for >1y | Low | Not yet done for `config/*` sub-packages |
| README per package with examples | Users land on individual pkg.go.dev pages | Low | Partially done — `httptest_mock/README.md` exists, others use doc comments |
| Root `README.md` listing packages | First thing users see on GitHub | Low | Exists (STRUCTURE.md confirms) |
| Conventional commit history | Versioning tooling (release-please, goreleaser) depends on it | Low | Done — `.commitlint.yaml` configured |
| Linting CI | Go community expects `golangci-lint` | Low | Done — `.golangci.yml` with 42 linters |
| Test coverage enforcement | Industry standard for public libraries | Low | Done — `.testcoverage.yml` at 95% |
| Cross-platform CI | Utility libs should support Linux/macOS/Windows | Low | Done — `.github/workflows/go.yml` |

## Differentiators

Features that set a well-organized monorepo apart. Not universally expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| `doc.go` per package | Canonical Go way to document packages; shows on pkg.go.dev | Low | Not yet done — consider adding |
| `example_test.go` per package | Runnable, testable documentation on pkg.go.dev | Low | Done in `flow/`, `fraction/`, `set/`, `shell_tools/`, `time_tools/` — should be everywhere |
| Package template / scaffolding | Lowers friction for new contributors; enforces conventions | Medium | Consider `_template/` directory with boilerplate |
| Go workspace (`go.work`) | Enables local multi-module development without commitment | Low | Not yet — would be needed if splitting to multi-module |
| README badges (coverage, go report, pkg.go.dev) | Social proof for library adoption | Low | Check current README |
| CLI tools in `cmd/` | Useful companion CLI utilities (e.g., config validator, release tool) | Medium | Currently pure library — only add if a clear CLI use case emerges |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| `pkg/` catch-all directory | Go team advises against it for libraries; adds nesting without value | Flat packages at root |
| Monolithic single `package go` | Forces unrelated abstractions into one namespace (anti-Lodash for unrelated utilities) | Separate packages per utility |
| `init()` functions for setup | Side effects at import time surprise users and break testability | Lazy init with `sync.Once` or explicit setup |
| Third-party build system (Bazel, Please) | Overkill for a 12-package personal monorepo | `Makefile` + `go build`/`go test` is sufficient |
| Code generation framework | Adds build complexity; rarely worth it for utility libs | Hand-written Go + generics is idiomatic |

## Feature Dependencies

```
Package naming standardization → New package template
    (Need naming conventions before template can enforce them)

config/internal/ sub-package move → Stable config API
    (Should not move to internal while still iterating on public surface)

Multi-module split ← Only if version coupling becomes painful
    (go.work workspace → separate go.mods per package)
```

## MVP Recommendation

Current state is already well-structured. Prioritize:

1. **Move config sub-packages to `internal/`** — Correctness/API stability
2. **Add `doc.go` to every package** — Discoverability on pkg.go.dev
3. **Standardize naming conventions** — Consistency for future packages
4. **Create package template** — Institutionalize conventions

Defer: Multi-module split — revisit only when version coupling causes real pain.

## Sources

- Go official module layout guidance: https://go.dev/doc/modules/layout (HIGH confidence)
- Go blog on package naming: https://go.dev/blog/package-names (HIGH confidence)
- samber/lo conventions: https://github.com/samber/lo (HIGH confidence)
- uber-go/guide: https://github.com/uber-go/guide (HIGH confidence)
- Current project codebase at `/Users/guionardo/dev/go` (HIGH confidence)

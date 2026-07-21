# Technology Stack

**Project:** `github.com/guionardo/go` — Go Utility Collection
**Researched:** 2026-07-21
**Overall Stack Confidence:** HIGH

## Executive Summary

This project is a Go 1.26 monorepo of independent utility packages with minimal external dependencies. The stack philosophy — **stdlib-first, minimal deps, generics where natural** — is correct for 2025-2026. Go's rapid evolution across 1.24→1.26 has eliminated several common reasons for third-party utility packages. This document recommends keeping the existing dependency strategy while selectively adding one test-only dependency (`go-cmp` for structural comparison) and ruthlessly avoiding kitchen-sink libraries like `samber/lo`.

## Recommended Stack

### Core Language
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.26.4 | Source language, compiler, runtime | Green Tea GC is now default (10-40% less GC overhead). Swiss Tables in stdlib map. `errors.AsType[T]()` available. `sync.WaitGroup.Go()` convenience. All code should target go 1.26 in go.mod. |
| Standard library | Go 1.26 | All core functionality | The gap between stdlib and popular third-party packages continues to narrow. `sync.Map` is now hash-trie based. `strings.Lines/SplitSeq` provide iterator-based string splitting. `os.Root` provides chroot-style filesystem access. `weak` package enables canonicalization maps. |
| Go modules | Go 1.26 | Dependency management | `go.mod` `tool` directive for tracking build tools. `go fix` now includes modernizers for automated stdlib migration. |

### Existing Dependencies (Keep)

| Dependency | Version | Purpose | Confidence | Why Keep |
|------------|---------|---------|------------|----------|
| `github.com/stretchr/testify` | v1.11.1 | Test assertions | HIGH | De-facto standard, stable v1 API. |
| `github.com/go-playground/validator/v10` | v10.30.3 | Struct validation | HIGH | Best-in-class, used by `config/`. |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML marshaling | HIGH | Still the de-facto YAML library. |
| `golang.org/x/sync` | v0.22.0+ | Errgroup, Semaphore, Singleflight | HIGH | Officially maintained by Go team. |
| `github.com/opencontainers/go-digest` | v1.0.0 | Content digest verification | MEDIUM | Only used by `release/`. Consider replacing with stdlib `crypto/sha256`. Keep for now. |

### New Test-Only Dependency (Add)

| Dependency | Version | Purpose | Confidence | Why Add |
|------------|---------|---------|------------|---------|
| `github.com/google/go-cmp` | v0.7.0+ | Structural comparison in tests | HIGH | Significantly better than `reflect.DeepEqual`. Provides readable diff output, custom comparers (float tolerance), unexported field options. v0.7.0 released Feb 2025 (4.7k ★). **Test-only dependency** — no impact on library consumers. |

### Tooling (Keep & Upgrade)

| Tool | Version | Purpose | Why |
|------|---------|---------|-----|
| `golangci-lint` | latest (v1.64+) | Comprehensive linting | Enable `copylock` for Go 1.25+ 3-clause loop mutex checks. |
| `go-test-coverage` | v2 | Coverage enforcement | Works well with Go 1.26. |
| `pre-commit` | latest | Git hooks | Fix Linux-only install in Makefile (macOS compat). |
| `commitlint` | latest | Conventional commits | Keep. |
| `govulncheck` | Go 1.24+ | Vulnerability scanning | Now integrated with `go` toolchain. |

## Libraries to NOT Add (And Why)

### `samber/lo` (21.4k ★)
**Assessment:** DO NOT ADD
**Rationale:** Adding it would: (1) contradict the minimal-dependency constraint, (2) overlap with existing `flow/` package (`lo.Ternary` ≈ `flow.If`, `lo.Coalesce` ≈ `flow.Default`), (3) pull in 930+ commits for functionality easily written as small generics. **Alternative:** implement specific helpers as small standalone generics when a genuine need arises — same pattern used for `Set[T]` and `flow/` helpers.

### `hashicorp/go-multierror`
**Assessment:** DO NOT ADD
**Rationale:** `errors.Join` (Go 1.20+) provides error aggregation natively. Go 1.26 adds `errors.AsType[T]()` for type-safe error unwrapping. The project already uses `errors.Join`.

### `uber-go/zap` or `rs/zerolog`
**Assessment:** DO NOT ADD
**Rationale:** The project already uses `log/slog` (Go 1.21+). For a library, `slog` is the correct choice — it forces no logging dependency on consumers. `slog.DiscardHandler` (Go 1.24) covers disabling output. `slog.NewMultiHandler` (Go 1.26) covers fan-out. Always use `slog` for libraries.

### `uber-go/dig` / `samber/do` / `google/wire`
**Assessment:** DO NOT ADD
**Rationale:** DI is an application-level concern. This is a library/utility module with no application runtime.

### `spf13/cobra` / `spf13/viper`
**Assessment:** DO NOT ADD
**Rationale:** CLI and application configuration. The `config/` package already provides typed configuration.

### `golang.org/x/exp`
**Assessment:** DO NOT USE
**Rationale:** `x/exp` packages are unstable. Target Go 1.26 stable. Key `x/exp/slices` functions are now in stdlib `slices` since Go 1.21.

## New Package Patterns

For any NEW packages, follow these 2025-2026 Go idioms:

### Pattern: Generic utility with `iter.Seq`
Provide `iter.Seq` methods for iteration on collection/container types:

```go
func (s Set[T]) All() iter.Seq[T] {
    return func(yield func(T) bool) {
        for k := range s.items {
            if !yield(k) { return }
        }
    }
}
```

Apply this to the existing `set` package and any future containers.

### Pattern: `errors.AsType[T]()` for Go 1.26+
Go 1.26's `errors.AsType[T any]() bool` replaces the clunky `errors.As(err, &target)` pattern:

```go
if myErr, ok := errors.AsType[*MyError](err); ok {
    // use myErr directly, no extra variable needed
}
```

### Pattern: `sync.WaitGroup.Go()` for goroutines
Go 1.25's `wg.Go(func())` eliminates the `wg.Add(1); defer wg.Done()` boilerplate.

### Pattern: `testing.B.Loop()` for benchmarks
Use `for b.Loop()` instead of `for range b.N`. Setup runs once, not b.N times.

### Pattern: `testing.T.Context()` for test timeouts
Go 1.24's `t.Context()` returns a context canceled when the test completes.

### Pattern: `slog` with `slog.DiscardHandler` for library logging
Any new package that needs logging must use `slog.Logger` passed as parameter. Never create package-level loggers. Use `slog.New(slog.DiscardHandler)` in tests.

### Pattern: `go-cmp` for struct comparison in tests
Use `cmp.Diff(want, got)` instead of `reflect.DeepEqual(want, got)` for readable diff output:

```go
import "github.com/google/go-cmp/cmp"
if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("mismatch (-want +got):\n%s", diff)
}
```

## Go Version Feature Matrix

| Feature | Go Version | Relevance |
|---------|------------|-----------|
| Generic type parameters | 1.18 | Foundation — used by `Set[T]`, `Provider[T]`, `flow.If` |
| `any` alias | 1.18 | Used throughout |
| `encoding/json` `omitempty` | 1.20 | Already in use |
| `errors.Join` | 1.20 | Error aggregation |
| `log/slog` | 1.21 | Logging (used in config, httptest_mock) |
| `slices` package | 1.21 | Slice operations |
| `maps` package | 1.21 | Map operations |
| `iter.Seq` / range-over-func | 1.23 | Container iteration (adopt in `set`) |
| Swiss Tables map | 1.24 | Default — affects all map-backed packages |
| `testing.B.Loop()` | 1.24 | Benchmark pattern |
| `slog.DiscardHandler` | 1.24 | Silence logs in tests |
| `testing.T.Context()` | 1.24 | Test-scoped context |
| `weak` package | 1.24 | Weak pointers |
| `os.Root` | 1.24 | Directory-scoped fs ops |
| `strings.Lines/SplitSeq` | 1.24 | Iterator-based string splitting |
| `errors.AsType[T]()` | 1.25 | Type-safe error unwrapping |
| `sync.WaitGroup.Go()` | 1.25 | Concurrent goroutine launching |
| `testing/synctest` | 1.25 | Concurrent code testing |
| `go fix` modernizers | 1.26 | Automated code migration |
| `slog.NewMultiHandler` | 1.26 | Log routing to multiple handlers |
| `reflect.Type.Fields()` iterators | 1.26 | Reflection without index loops |

## Implementation Guidance

### For Existing Package Updates
1. **`set` package**: Add `All() iter.Seq[T]` method for range-over-func compatibility
2. **`config` package**: Use `errors.AsType[T]()` for validation error handling; fix the silent-error swallowing (CONCERNS.md)
3. **`time_tools`**: Consider copy-on-write pattern instead of mutex promotion for layout list
4. **`release` package**: Fix critical bugs (unused request, missing body close, no timeout)
5. **Testing across all packages**: Use `t.Context()` instead of `context.Background()` in tests
6. **Benchmarks**: Convert to `for b.Loop()` pattern where applicable
7. **General**: Run `go fix` (Go 1.26 modernizers) to auto-migrate patterns

### For New Package Decision Flow
When considering a new package, ask:
1. **Can stdlib (Go 1.26) do it?** → If yes, don't write a package. Document the pattern instead.
2. **Is it generically reusable across 3+ projects?** → If yes, consider adding.
3. **Does it require external dependencies?** → If yes, strongly reconsider.
4. **Is it a wrapper around an external API?** → If yes, design as thin adapter (like `release/`).
5. **Does it duplicate `samber/lo` functionality?** → If yes, implement as single function, not package.

### Version Pinning Strategy
- All dependencies pinned in `go.sum`
- `golang.org/x/sync` — keep updated (minor versions)
- `yaml.v3` — stable, no upgrade concerns
- `testify` — keep within v1.x (no v2 exists)
- `go-playground/validator` v10 — stable, upgrade only for security fixes

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| **Test comparisons** | `gotest.tools/v3/assert` + `go-cmp` | `testify/assert` (existing) | Keep `testify`. Add `go-cmp` alongside, don't replace. |
| **Logging** | `log/slog` (stdlib) | `uber-go/zap` | Library must not force logging on consumers. |
| **Error aggregation** | `errors.Join` (stdlib) | `hashicorp/go-multierror` | `errors.Join` + `errors.AsType[T]()` covers all needs. |
| **Generic utilities** | In-house generics | `samber/lo` | Contradicts minimal-dependency philosophy. |
| **Configuration** | Current `config/` package | `spf13/viper` | Viper is application-focused, not library-friendly. |
| **HTTP testing** | Current `httptest_mock/` | `jarcoal/httpmock` | In-house package is a project differentiator. |
| **DI container** | None | `google/wire`, `uber-go/dig` | Not applicable to a utility library. |
| **SQL driver helpers** | None | `jmoiron/sqlx` | Out of scope. Let downstream decide. |
| **Time parsing** | Current `time_tools/` | `lotus` / `dateparse` | Existing solution follows project pattern. |

## Sources

- Go 1.24 Release Notes (Feb 2025) — `tip.golang.org/doc/go1.24` (HIGH confidence, official source)
- Go 1.25 Release Notes (Aug 2025) — `tip.golang.org/doc/go1.25` (HIGH confidence, official source)
- Go 1.26 Release Notes (Feb 2026) — `go.dev/doc/go1.26` (HIGH confidence, official source)
- `golang.org/x/sync` v0.22.0 — `pkg.go.dev/golang.org/x/sync` (HIGH confidence, official source)
- `samber/lo` v1.53.0 — `github.com/samber/lo` (HIGH confidence, official repo)
- `google/go-cmp` v0.7.0 — `github.com/google/go-cmp` (HIGH confidence, official repo)
- `samber/do` v2.1.0 — `github.com/samber/do` (HIGH confidence, official repo)
- Project stack — `.planning/codebase/STACK.md` (HIGH confidence, project source)
- Project concerns — `.planning/codebase/CONCERNS.md` (HIGH confidence, project source)

---

*Stack research conducted: 2026-07-21. Review after Q1 2027 or when Go 1.27 is released.*
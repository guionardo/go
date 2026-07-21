# Architecture Patterns for Go Utility Monorepos

**Domain:** Go utility libraries / developer toolkits  
**Researched:** 2026-07-21  
**Mode:** Ecosystem (architecture dimension)  

## Executive Summary

Go utility monorepos follow one of six distinct organizational patterns, ranging from flat single-package repos (samber/lo) to multi-module meta-repos (golang.org/x). The `github.com/guionardo/go` project currently uses the **flat package-per-utility** pattern — each utility is a top-level package directory under a single Go module. This pattern occupies a specific niche: it offers lower discoverability friction than uber-go's multi-repo approach while maintaining stronger package isolation than samber/lo's flat single-package. The research identifies a gap: as the collection grows past ~15 packages, navigation and discoverability degrade without some form of categorization (grouping, naming conventions, or a root package pattern).

## The Six Major Patterns

### Pattern 1: Flat Single-Package Monorepo

**Used by:** `samber/lo` (21.4k stars)  
**Structure:**
```
lo/
  go.mod
  slice.go          # package lo
  map.go            # package lo
  find.go           # package lo
  condition.go      # package lo
  ...
  internal/         # private helpers
  mutable/          # separate package (mutable variants)
  parallel/         # separate package
  it/               # separate package (iterators)
```

**Key characteristics:**
- Everything is `package lo` — users import `github.com/samber/lo`
- Utility functions split across files by *concern* (slice.go, map.go, find.go)
- Sub-packages only for genuinely different abstractions (mutable collections, iterators)
- One `go.mod` at root, one module

**When it fits:** Library with a tightly cohesive API surface (like Lodash). All functions share the same namespace and mental model.

**When it doesn't:** Unrelated utilities (config, document validation, HTTP mocking) forced into one package — violates Go idiom of separate packages for separate concerns.

### Pattern 2: Flat Package-Per-Utility (This Project's Pattern)

**Used by:** `github.com/guionardo/go`, `hashicorp/go-multierror`, `hashicorp/go-uuid`, `hashicorp/go-retryablehttp`  
**Structure:**
```
go/
  go.mod
  set/              # package set
  config/           # package config
  fraction/         # package fraction
  flow/             # package flow
  ...
  config/
    environment/    # sub-package config/environment
    profile/        # sub-package config/profile
    merger/         # sub-package config/merger
    validation/     # sub-package config/validation
```

**Key characteristics:**
- Each top-level directory is a separate Go package (different `package` declaration)
- One `go.mod` at root, one module — all packages versioned together
- Sub-packages for complex utilities (config/ has internal structure)
- Minimal cross-package dependencies (packages are independent)

**When it fits:** Collection of *independent* utilities. Each package is individually useful. The `go get` UX is clean: `go get github.com/user/go/set`.

**Tradeoffs:**
- + Each package is independently usable and testable
- + Simple mental model: one directory = one import path
- + Easy to add new packages without touching existing ones
- - No discoverability structure at root level (just an alphabetical list of directories)
- - Version coupling: `go get` fetches all packages even when you need one
- - Package naming collisions can happen (distinct namespace each time)

### Pattern 3: Core + Sub-Packages + Internal

**Used by:** `rs/zerolog` (12.5k stars)  
**Structure:**
```
zerolog/
  go.mod
  log.go            # package zerolog (core)
  event.go          # package zerolog
  context.go        # package zerolog
  ...
  hlog/             # sub-package for net/http integration
  pkgerrors/        # sub-package for error stacktrace
  diode/            # sub-package for lock-free writer
  internal/         # private helpers
  cmd/              # CLI tools (separate main packages)
    zed/            # command-line tool
    ...
```

**Key characteristics:**
- Core type in root package, optional integrations as sub-packages
- `internal/` for implementation details not meant for public consumption
- `cmd/` for CLI tooling (rare in pure utility libs, but present when a CLI is useful)
- Optional packages import the core package

**When it fits:** A central utility (like a logger) with optional integrations (HTTP middleware, error handlers). Users get the core with `go get` and opt into integrations.

### Pattern 4: Multi-Module Monorepo

**Used by:** `golang.org/x/*` (Go team's sub-repositories)  
**Structure:**
```
x/
  sync/
    go.mod           # module golang.org/x/sync
    sema/
    errgroup/
    singleflight/
  crypto/
    go.mod           # module golang.org/x/crypto
    ssh/
    openpgp/
    ...
  net/
    go.mod           # module golang.org/x/net
    http2/
    websocket/
    ...
```

**Key characteristics:**
- Each top-level directory is its own Go module with its own `go.mod`
- Independent versioning per module
- Shared internal tooling via workspace or cross-module `replace` directives
- Usually backed by a monorepo tool (gazelle, please, etc.)

**When it fits:** Large org with multiple related but independently versioned products (Go team, Tailscale, Kubernetes).

**Tradeoffs:**
- + Independent versioning — update one module without affecting others
- + Clean `go get` — fetches only the module you need
- - Requires monorepo tooling for cross-cutting changes
- - Higher cognitive overhead: N go.mod files to maintain
- - Cross-module refactors are painful

### Pattern 5: Separate Repos Per Utility

**Used by:** `uber-go` (zap, fx, config, automaxprocs, etc.), `hashicorp` (go-multierror, go-uuid, go-retryablehttp)  
**Structure:**
```
github.com/uber-go/zap/        # one repo
github.com/uber-go/fx/         # another repo
github.com/uber-go/config/     # another repo
...
```

**Key characteristics:**
- Every utility gets its own repository and module
- Independent issue tracking, releases, CI, community
- Organization-level discoverability (org README lists repos)

**When it fits:** Large organization or when utilities have different maintainers, release cadences, and user bases.

**Tradeoffs:**
- + Maximum independence
- + Can have different license, maintainers, CI
- - Massive operational overhead (N repos to manage)
- - Users must discover each utility separately
- - Cross-cutting changes require N PRs

### Pattern 6: Standard `cmd/` + `pkg/` Layout

**Used by:** Many Go applications that also export packages, Kubernetes, Docker, Helm  
**Structure:**
```
project/
  go.mod
  cmd/              # binaries (each dir = package main)
    server/
    client/
  pkg/              # public library code
    auth/
    config/
  internal/         # private implementation
```

**When it fits:** Applications that also happen to export useful packages. Not recommended for pure utility collections.

## Where This Project Sits

`github.com/guionardo/go` uses **Pattern 2 (Flat Package-Per-Utility)** with a minor variation:
- 12 independent packages at root
- `config/` has sub-packages (Pattern 3 influence) but no `internal/` barrier
- No `cmd/` — pure library
- No multi-module — single `go.mod`

This is the **right pattern for a personal utility collection** at its current scale (12 packages). It matches how `hashicorp` structures its individual `go-*` repos, but consolidated into one module for maintenance efficiency.

## Recommended Evolution Strategy

As the collection grows, these structural pressures will emerge:

### At 15–20 packages: Add naming categorization
- **Problem:** `ls` at root shows 15+ directory names with no logical grouping
- **Solution:** Introduce a prefix or suffix convention for related packages, e.g.:
  - `docs_br` instead of `br_docs` (moves `br` to prefix for grouping)
  - Or accept alphabetical interleaving as the cost of flat structure

### At 20–30 packages: Consider `internal/` for complex packages
- **Problem:** `config` has 4 sub-packages that are all public — external users can import `config/environment` directly, coupling them to internal structure
- **Solution:** Move `config/environment`, `config/merger`, `config/profile`, `config/validation` under `config/internal/`. They become `config/internal/environment`, etc. — not importable by external modules. This matches Go official guidance and rs/zerolog's approach.

### At 30+ packages or different release cadences: Evaluate multi-module
- **Problem:** A bug fix in `set/` forces a version bump for all 30+ packages
- **Solution:** Split into separate Go modules. This is a **significant** restructuring — only warranted when there's actual pain from version coupling.

## Component Boundaries (Current Project)

### Package Independence Analysis

```
Independent (no imports from sibling packages):
  br_docs, flow, fraction, mid, path_tools,
  reflect_tools, release, set, shell_tools, time_tools

Has internal dependencies:
  config → config/environment, config/profile,
           config/merger, config/validation
  httptest_mock → flow, reflect_tools (small, for utility helpers)
```

### Implicit Coupling Risks

| Risk | Current State | Recommendation |
|------|--------------|----------------|
| `config/` sub-package explosion | 4 sub-packages, all public | Evaluate `internal/` barrier |
| `httptest_mock` import chain | Imports `flow` and `reflect_tools` | Acceptable — these are stable, generic helpers |
| Package naming inconsistency | Mix of singular (`set/`, `flow/`), compounded (`path_tools/`, `shell_tools/`, `reflect_tools/`, `time_tools/`), and underscore (`br_docs/`, `httptest_mock/`) | Standardize on one pattern for new packages |

### Suggested Package Categorization

```
Data Structures:
  set/          — generic Set[T comparable]
  fraction/     — immutable Fraction type
  ...           — (future: ordered map, ring buffer, etc.)

Control Flow:
  flow/         — If[T], Default[T]
  ...           — (future: Try[T], pipeline, etc.)

I/O / System:
  path_tools/   — file/directory operations
  shell_tools/  — shell argument parsing, env vars
  mid/          — machine identifier
  time_tools/   — time parsing
  release/      — GitHub release fetcher

Validation / Domain:
  br_docs/      — Brazilian document validation
  reflect_tools/— reflection utilities
  ...           — (future: email, phone, URL validators)

Infrastructure:
  config/       — typed configuration provider
  httptest_mock/— HTTP mock server
```

*Note: The project does not actually organize packages into subdirectories — this categorization is a conceptual grouping to guide future package naming decisions.*

## Data Flow Between Components

### Current State

```
External Go Project
    │
    ├── go get github.com/guionardo/go/set
    ├── go get github.com/guionardo/go/config
    ├── go get github.com/guionardo/go/httptest_mock
    └── ... (each package imported independently)
```

There is **no cross-package data flow** in the library itself — each package is a leaf. The data flow happens *inside* consuming projects that combine multiple packages.

### Internal Data Flow (config package only)

```
External caller
    │
    ▼
config.NewProvider[T]()
    │
    ├── config/internal/options.go → functional options configure Provider
    │
    ▼
Provider.GetConfiguration()
    │
    ├── config/profile.GetScopedProfileContent() → reads YAML
    │   └── config/merger.MergeMaps() → deep-merges scopes
    │
    ├── config/environment.ParseEnvironment() → env var overrides
    │
    └── config/validation.ValidateConfig() → struct validation
```

### Data Flow (httptest_mock package only)

```
Test code → SetupServer() → MockHandler
    │
    ├── NewMock() registers Mock handler
    │
    ▼
Incoming HTTP request → MockHandler.ServeHTTP()
    │
    ├── Mock.Matches() checks method, path, query, headers, body
    │   └── flow/helpers used internally for matching logic
    │
    ├── Match found → Mock.WriteResponse() → status, headers, body
    │
    └── No match → 404 response
```

## Informing Build Order

Because packages are **independent**, there is no required build order — each package can be built, tested, and released independently. The only structural dependency is:

1. **Layer 0 (stable base, no project deps):** `flow`, `reflect_tools`, `set`, `fraction` — Generic utility types with zero internal dependencies
2. **Layer 1 (platform/system deps):** `path_tools`, `shell_tools`, `mid`, `time_tools`, `br_docs` — System-level utilities, may depend on Layer 0 for implementation convenience
3. **Layer 2 (complex utilities):** `config`, `httptest_mock`, `release` — Multi-file/multi-subpackage utilities that depend on stdlib + Layer 0

### Cross-cutting concerns (apply to all layers):

- **CI/CD:** One pipeline, all packages built/tested together
- **Documentation:** `README.md` per package + root overview
- **Versioning:** Single semver version for all packages (current model)

## Patterns to Keep

| Pattern | Where Used | Why Keep |
|---------|-----------|----------|
| Functional Options | `config/options.go` | Idiomatic Go for builder-like configuration |
| Generic structs | `Set[T]`, `Provider[T]` | Go 1.18+ generics reduce boilerplate |
| Co-located tests | Every `_test.go` | Standard Go convention, enables `go test ./...` |
| Example tests | `example_test.go` | Runnable documentation, golden files |
| Build tags | `_darwin.go`, `_linux.go` | Clean cross-platform separation |
| Sentinel errors | `ErrDivideByZero`, etc. | Testable error identity |
| Value types | `Fraction` | Immutable by design, no pointer receiver on accessors |

## Patterns to Consider Adding

| Pattern | Reason | Priority |
|---------|--------|----------|
| `internal/` for sub-packages | Encapsulate `config/environment`, etc. | Medium — recommended when config's API stabilizes |
| `go.work` for development | Enables multi-module without separating repos | Low — only if splitting into multiple modules |
| `doc.go` per package | Package-level documentation for pkg.go.dev | Low — README.md covers this, but doc.go is the canonical Go way |

## Anti-Patterns to Avoid

| Anti-Pattern | Why Avoid | Instead |
|-------------|-----------|---------|
| `pkg/` directory as catch-all | `pkg/` serves no purpose in a pure-lib monorepo. Go team advises against it for library modules. | Flat packages at root |
| Internal cross-package coupling | Two sibling packages importing each other creates tight coupling and testing complexity | Keep packages independent; extract shared logic into a new package or use duplication |
| `init()` functions | Side effects at import time break testability and surprise users | Lazy init, explicit setup, or `sync.Once` |
| `internal/` at root for everything | Over-encapsulation hurts usability for a utility library | Use `internal/` only for genuinely private implementation details |
| Monolithic single package | Forcing unrelated concerns into one package (counterexample needed) | Separate packages per concern |

## Scalability Considerations

| Metric | Current (12 pkgs) | 25+ packages | 50+ packages |
|--------|-------------------|--------------|--------------|
| `go test ./...` time | Fast (< 5s) | Acceptable (< 15s) | May need CI parallelization |
| Discoverability | `ls` shows all | Alphabetical browse OK | Categorization needed |
| Version coupling | Negligible | Minor friction | May motivate multi-module split |
| Import UX | `go get github.com/user/go/pkg` | Same | Same |
| Maintenance overhead | Low (single CI, single release) | Low-Medium | Medium — consider tooling |

## Sources

- Go official documentation on module organization: https://go.dev/doc/modules/layout (HIGH confidence)
- samber/lo repository structure: https://github.com/samber/lo (HIGH confidence)
- rs/zerolog repository structure: https://github.com/rs/zerolog (HIGH confidence)
- hashicorp/go-multierror: https://github.com/hashicorp/go-multierror (HIGH confidence)
- Uber Go organization: https://github.com/uber-go (HIGH confidence)
- golang.org/x sub-repositories: https://pkg.go.dev/golang.org/x (HIGH confidence)
- go.dev blog post on module layout and internal packages: https://go.dev/doc/modules/layout (HIGH confidence)

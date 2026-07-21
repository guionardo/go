<!-- refreshed: 2026-07-21 -->
# Architecture

**Analysis Date:** 2026-07-21

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                      Top-Level Go Packages (Utilities)                   │
├────────────┬──────────┬───────────┬──────────┬──────────┬───────────────┤
│  config/   │  flow/   │ fraction/ │  set/    │  mid/    │  httptest_    │
│ type-safe  │  generic │ immutable │ generic  │ cross-   │  mock/        │
│ config     │  ternary │ fraction  │ Set[T]   │ platform │ HTTP mock     │
│ provider   │  &       │ & math    │ with     │ machine  │ server for    │
│            │  default │ ops       │ SQL scan │ ID       │ tests         │
│            │  value   │           │ & marshal│          │               │
├────────────┼──────────┼───────────┼──────────┼──────────┼───────────────┤
│ br_docs/   │ path_    │ shell_    │ reflect_ │ time_    │ release/      │
│ Brazilian  │ tools/   │ tools/    │ tools/   │ tools/   │ GitHub        │
│ document   │ file/dir │ quoted    │ zero-    │ flexible │ release       │
│ validation │ path ops │ shell arg │ value    │ time     │ fetcher       │
│ (CPF/CNPJ) │          │ parsing   │ check    │ parser   │               │
└────────────┴──────────┴───────────┴──────────┴──────────┴───────────────┘
         │                  │                    │
         ▼                  ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Go Standard Library                                │
│  net/http, os, reflect, encoding/json, database/sql, regexp, sync,       │
│  iter, log/slog, runtime/debug                                           │
└─────────────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  External Dependencies (minimal)                                         │
│  github.com/stretchr/testify — testing assertions                       │
│  github.com/go-playground/validator/v10 — struct validation             │
│  gopkg.in/yaml.v3 — YAML parsing                                        │
│  golang.org/x/sync — synchronization primitives                         │
│  github.com/opencontainers/go-digest — content digest verification      │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Package | Responsibility | Path |
|---------|----------------|------|
| config | Generic typed configuration provider — loads YAML profiles + env vars + validation | `config/` |
| flow | Generic ternary (`If`) and zero-value default (`Default`) helpers | `flow/` |
| fraction | Immutable `Fraction` type with arithmetic (add, subtract, multiply, divide) | `fraction/` |
| httptest_mock | HTTP mock server with request matching (path, query, headers, body) from code or files | `httptest_mock/` |
| mid | Cross-platform machine identifier (Linux hostnamectl/dbus, macOS system_profiler, Windows registry) | `mid/` |
| path_tools | File/directory existence, path creation, Go root folder detection, PATH search | `path_tools/` |
| reflect_tools | Reflection utilities — zero-value checking across all Go types | `reflect_tools/` |
| release | GitHub latest release fetcher + asset download with digest verification | `release/` |
| set | Generic `Set[T comparable]` backed by `map[T]struct{}` — union, intersect, diff, filter, marshal, SQL scan | `set/` |
| shell_tools | Quoted shell argument parsing (`QuotedShellArgs`) and case-insensitive env var lookup | `shell_tools/` |
| time_tools | Time string parser with auto-prioritizing layout list (promotes successful templates) | `time_tools/` |
| br_docs | Brazilian document validation — CPF and CNPJ check-digit verification | `br_docs/` |

## Pattern Overview

**Overall:** Flat package-per-utility library monorepo

**Key Characteristics:**
- **No `cmd/` or `main()` entry points** — pure library intended for import by other projects
- **Flat top-level packages** — each package is a standalone utility with no internal interdependency (except `config/` → `config/environment`, `config/profile`, `config/merger`, `config/validation`)
- **Generics-first design** — Go 1.26 generics used for type-safe containers (`Set[T]`, `Provider[T]`, flow helpers)
- **Minimal external dependencies** — only 5 direct dependencies, relying heavily on the Go standard library
- **Cross-platform support via build tags** — `mid/` and `path_tools/` use OS-specific files (`_darwin.go`, `_linux.go`, `_windows.go`)
- **Functional options pattern** — used in `config/` for provider configuration (`WithScope`, `WithProfilesPath`, `WithLogger`, etc.)
- **Interface-based extensibility** — `Mocker` interface for mock implementations, `Validator` for self-validating types, `Logger` abstraction, `CustomHandlerFunc`
- **Co-located tests** — every `.go` file has a corresponding `_test.go` and `example_test.go` where applicable

## Layers

**Utility Packages (12 top-level packages):**
- Purpose: Provide reusable Go library code
- Location: Root directory — each package is a direct subdirectory
- Contains: Pure functions, generic types, interfaces
- Depends on: Go standard library + minimal external packages
- Used by: External Go projects importing `github.com/guionardo/go`

**Sub-packages (config internals):**
- Purpose: Support the `config` package with layered responsibilities
- Location: `config/environment/`, `config/profile/`, `config/merger/`, `config/validation/`
- Contains: Env var parsing, YAML profile loading, deep-merge, struct validation
- Depends on: `config` root package, external libs (validator, yaml.v3)
- Used by: `config` package only

**CI/Dev Tooling Layer:**
- Purpose: Maintain code quality, run tests, enforce conventions
- Location: `.github/workflows/go.yml`, `Makefile`, `.pre-commit-config.yaml`, `.golangci.yml`, `.testcoverage.yml`
- Contains: CI pipeline, linter config, test coverage thresholds, pre-commit hooks
- Depends on: Go toolchain, golangci-lint, go-test-coverage

## Data Flow

### Configuration Loading (config package)

1. `NewProvider[T]()` creates provider with functional options (`config/provider.go:41`)
2. `GetConfiguration()` called — first call acquires write lock (`config/provider.go:62`)
3. `loadStaticConfiguration()` reads YAML profile via `profile.GetScopedProfileContent()` (`config/provider.go:112`)
4. Profile YAML unmarshaled into struct `T` (`config/provider.go:122`)
5. `environment.ParseEnvironment()` overrides struct fields via `env` tags (`config/provider.go:127`)
6. `validateConfiguration()` runs validation via `validate` tags or `Validator` interface (`config/provider.go:131`)
7. Config cached with `sync.RWMutex` — subsequent calls return cached value (`config/provider.go:64`)

### HTTP Mock Matching (httptest_mock)

1. `SetupServer()` creates `httptest.Server` with `MockHandler` (`httptest_mock/setup.go:41`)
2. Incoming request hits `MockHandler.ServeHTTP()` (`httptest_mock/handler.go:56`)
3. Iterates registered `Mocker` instances calling `Matches()` (`httptest_mock/handler.go:69-70`)
4. `Request.match()` checks method → path (with params) → query params → headers → body (`httptest_mock/request.go:69`)
5. Full match → `WriteResponse()` writes status, headers, body with optional delay (`httptest_mock/response.go:39`)
6. Partial match → configurable via `WithAcceptingPartialMatch()` (`httptest_mock/handler.go:89`)
7. No match → 404 (`httptest_mock/handler.go:110`)
8. `Assert()` verifies expected hit counts at test end (`httptest_mock/handler.go:149`)

### Profile Loading with Scope Layering (config/profile)

1. `GetScopedProfileContent(basePath, defaultScope, scope)` called (`config/profile/profile.go:17`)
2. `getProfileFiles()` resolves YAML files for both scopes (`config/profile/profile.go:64`)
3. `readProfileMap()` decodes each YAML to `map[string]any` (`config/profile/profile.go:47`)
4. `merger.MergeMaps()` recursively deep-merges scope into default, scope wins (`config/profile/profile.go:42`)
5. Merged map marshaled back to YAML bytes and returned (`config/profile/profile.go:23`)

**State Management:**
- **`config.Provider[T]`** — uses `sync.RWMutex` for thread-safe read/write, caches config after first load
- **`time_tools` layouts** — uses `sync.RWMutex` for concurrent read/write to global layout list
- **`httptest_mock.Mock`** — uses individual mutexes per concern (`matchMu`, `responseMu`, `assertionLock`)
- **`httptest_mock.MockHandler`** — uses `sync.RWMutex` for mock list access
- No global mutable state in packages other than the ones above (stateless functions preferred)

## Key Abstractions

**`config.Provider[T any]`:**
- Purpose: Generic typed configuration provider — loads from YAML profiles + env vars, caches, validates
- Location: `config/provider.go`
- Pattern: Generic struct + functional options + RWMutex + lazy loading
- Constraints: `T` must be a struct (checked via `reflect.TypeFor[T]().Kind()`)

**`httptestmock.Mocker` (interface):**
- Purpose: Contract for HTTP request mock matching and response
- Location: `httptest_mock/interfaces.go`
- Pattern: Interface with 12 methods — matching, response, assertions, logging, data extraction
- Implementations: `*Mock` in `httptest_mock/mock.go`
- Extension: `CustomHandlerFunc` for custom response logic

**`set.Set[T comparable]`:**
- Purpose: Type-safe generic set backed by `map[T]struct{}`
- Location: `set/set.go`
- Pattern: Type alias over map with methods — Union, Diff, Intersection, Filter (iter.Seq), marshal, SQL scan
- Integration: `database/sql.Scanner` and `driver.Valuer` for DB persistence

**`fraction.Fraction`:**
- Purpose: Immutable fraction type — always simplified, never zero denominator
- Location: `fraction/fraction.go`
- Pattern: Value type (no pointer receivers on accessors), always reduced via GCD, `FromFloat64` float→fraction conversion
- Safety: `New()` returns error on zero denominator, `Divide()` returns error on zero divisor

**`validation.Validator` (interface):**
- Purpose: Self-validation contract for configuration types
- Location: `config/validation/validator.go`
- Pattern: Single-method interface — types implementing `Validate() error` get custom validation

## Entry Points

**Public API — function surface per package (all are entry points):**

| Package | Key Exported Functions/Types |
|---------|----------------------------|
| `br_docs` | `IsCPF(doc) bool`, `IsCNPJ(doc) bool`, `RemoveNonDigitAndLetters(value *string)` |
| `config` | `NewProvider[T](opts...) *Provider[T]`, `Provider.GetConfiguration()`, `Provider.UpdateConfiguration()` |
| `flow` | `If[T](cond, t, f) T`, `Default[T](value, zero) T` |
| `fraction` | `New[T,K](num, den) Fraction`, `FromFloat64(f) Fraction`, `Fraction.Add/Subtract/Multiply/Divide/Equal/Float64` |
| `httptest_mock` | `NewMock(method, path) *Mock`, `SetupServer(t, opts...)`, `GetMocksFrom(paths...)`, `Mock.FastServe(t)` |
| `mid` | `MachineID() string` |
| `path_tools` | `DirExists(path) bool`, `FileExists(path) bool`, `CreatePath(path) error`, `GetRootFolder(base)`, `FindFileInPath(file) string` |
| `reflect_tools` | `IsZeroValue(val) bool` |
| `release` | `GetLatestRelease(owner, repo) *Release`, `GetThisLatestRelease()`, `Asset.Download(w)` |
| `set` | `New[T](vals...) Set[T]`, `Set.Has/HasAll/Add/Remove/Union/Diff/Intersection/Filter/ToArray/Equals/Clear` |
| `shell_tools` | `NewQuotedShellArgs(s) QuotedShellArgs`, `GetEnv(name) string` |
| `time_tools` | `Parse(s) time.Time`, `SetLayouts(layouts)` |

## Architectural Constraints

- **No `internal/` package isolation** — all packages are public and importable by any Go project
- **No `main()` functions** — this is a library-only module with no executable targets
- **Generics constraint: `Set[T comparable]`** — elements must be comparable (maps/dicts not accepted)
- **Generics constraint: `Provider[T]`** — `T` must be a struct (checked at runtime with `panic`)
- **`config` internal sub-packages** — separated by concern but all in the same module, no `internal/` visibility restriction
- **Threading:** `sync.RWMutex` used in `config.Provider`, `time_tools` layout list, `httptest_mock.Mock` and `MockHandler` for concurrent safety
- **Global state:** `time_tools` maintains global `layouts` slice with `sync.RWMutex`; `config` has a package-level `slog.Logger` singleton via `sync.Once`
- **Circular imports:** None detected — the `config` → `config/environment|profile|merger|validation` hierarchy is acyclic; `httptest_mock` imports `flow` and `reflect_tools` only

## Error Handling

**Strategy:** Return errors as values — no panics in production code paths. Panics used only for programming errors (e.g., `NewProvider[string]()` panics with "configuration type must be a struct").

**Patterns:**
- Sentinel errors: `ErrDivideByZero`, `ErrZeroDenominator`, `ErrInvalid`, `ErrOutOfRange`, `ErrNotAGoProject`, `ErrTimeParser`
- Error wrapping: `fmt.Errorf("...: %w", err)` with `errors.Join()` for aggregated errors
- Deferred panic recovery: `ParseEnvironment` recovers panics from reflect operations; `MockHandler.ServeHTTP` recovers handler panics
- Validation errors: `go-playground/validator` provides structured field validation errors
- Graceful degradation: If profile loading fails, config provider continues with env vars only (logs error, doesn't fail)

## Cross-Cutting Concerns

**Logging:** Uses `log/slog` throughout. Config package provides a `Logger` interface abstraction with adapters for `slog.Logger` and custom loggers. Debug logging gated behind `WithDebugLogger()` option. Sensitive fields redacted via `safe` struct tag (masked as `********`).

**Validation:** Two-tier — `go-playground/validator` via struct tags (`validate:"required"`) and optional `Validator` interface for custom logic. Config provider validates both top-level and nested structs recursively.

**Marshaling:** YAML for configuration profiles (`gopkg.in/yaml.v3`). JSON for mock definitions (`httptest_mock`), internal mock data, and GitHub release API responses. Set package supports both JSON and YAML marshaling.

---

*Architecture analysis: 2026-07-21*

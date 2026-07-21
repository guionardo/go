# Coding Conventions

**Analysis Date:** 2026-07-21

## Naming Patterns

**Files:**
- `snake_case.go` — all source files use lowercase with underscores separating words (e.g., `shell_args.go`, `string_parts.go`, `find_file_path.go`)
- Test files mirror source: `*_test.go`
- Example test files: `example_test.go` in each package
- Build-tag files use platform suffix: `path_tool_darwin.go`, `path_tool_linux.go`, `path_tool_windows.go`

**Functions:**
- `camelCase` for unexported functions (e.g., `setMatchLog`, `readMock`, `unmarshalMock`, `calcCadastroDigit`)
- `PascalCase` for exported functions (e.g., `GetLatestRelease`, `CreatePath`, `SetupServer`, `IsCPF`)
- Generic functions prefer single-letter type params: `T`, `K` — convention `[T comparable]`, `[T, K integer]`

**Variables:**
- `camelCase` for local variables (e.g., `tmp`, `got`, `cfg`, `profilePath`)
- Short variable names for short scopes: `s` (set), `f` (fraction), `tt` (table test), `got`/`want` (test results)
- Unexported package-level vars: `camelCase` (e.g., `validate`, `layouts`, `emptyStruct`, `zeroValue`)

**Types:**
- `PascalCase` for exported types: `Set[T]`, `Fraction`, `Provider[T]`, `Release`, `Mock`, `Request`
- `PascalCase` for unexported types too: `provider`, `RequestMatchLevel`
- Interface suffix: none — exported interfaces use descriptive names like `Mocker`, `Validator`, `Logger`

**Constants:**
- `PascalCase` for exported sentinel errors: `ErrDivideByZero`, `ErrInvalid`, `ErrZeroDenominator`
- `PascalCase` for exported consts: `DefaultScope`, `MatchLevelNone`, `MatchLevelFull`
- `camelCase` for unexported consts: `emptyStruct`, `readDataPathParamPrefix`, `noMatchEmoji`, `kvCount`

## Code Style

**Formatting:**
- `gofmt` with `-s` simplification, enforced via pre-commit (`gofmt -l -w -s`)
- `golines` with `--max-len=120` for line wrapping, `--chain-split-dots` enabled
- `goimports` for import sorting and grouping
- Go version: 1.26.4
- Line length: 120 characters max (enforced by `golangci-lint` lll linter)
- Tab indentation (1 tab = 4 spaces in golines config)

**Linting:**
- `golangci-lint` v2 with extensive linter suite (50+ linters enabled)
- Configuration in `.golangci.yml` with `new-from-merge-base: main` for incremental linting
- `//nolint:linter` annotations used for intentional violations (e.g., `//nolint:funlen`, `//nolint:cyclop`, `//nolint:mnd`)
- Key linters: `cyclop` (max-complexity: 10), `funlen`, `gocognit`, `godoclint`, `gosec`, `govet`, `staticcheck`, `testifylint`, `paralleltest`, `wsl_v5`, `wsl_v5`
- `gofmt` rewrite rules: `interface{}` → `any`, `a[b:len(a)]` → `a[b:]`

**Pre-commit hooks:** `.pre-commit-config.yaml` enforces:
1. No commit to `main` branch
2. Trailing whitespace removal
3. End-of-file fixer
4. YAML validity
5. Large file check
6. Commitlint (conventional commits)
7. `go mod tidy`
8. `gofmt -l -w -s`
9. `go test ./...`
10. `golangci-lint run`
11. `go-vulncheck`

## Import Organization

**Order:**
1. Standard library packages (no blank line separation within)
2. External/third-party packages (separated by blank line from stdlib)
3. Internal project packages (`github.com/guionardo/go/...` — separated by blank line from third-party)

**Example from `config/provider.go`:**
```go
import (
    "log/slog"
    "reflect"
    "sync"

    "github.com/guionardo/go/config/environment"
    "github.com/guionardo/go/config/profile"
    "gopkg.in/yaml.v3"
)
```

**Path Aliases:**
- External test packages import their own package with explicit aliasing when the import path differs from the package name:
```go
import (
    httptestmock "github.com/guionardo/go/httptest_mock"
    shelltools "github.com/guionardo/go/shell_tools"
    timetools "github.com/guionardo/go/time_tools"
    reflecttools "github.com/guionardo/go/reflect_tools"
)
```
- This is used when the module path segment contains underscores or doesn't match the declared `package` name.

## Type Declarations

**Grouped types preferred:**
```go
type (
    Set[T comparable] map[T]struct{}
)
```

**Multiple related types in single group:**
```go
type (
    Mock struct { ... }
    RequestMatchLevel uint8
)
```

**Declaration order enforced by linter:** `type` → `const` → `var` → `func` (per `decorder` linter config)

**Composite literals always use field names** (no positional initialization):
```go
// Good
srv := &http.Server{
    Addr:         ":8080",
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
}

// Never positional
srv := &http.Server{":8080", 5 * time.Second, ...}
```

## Error Handling

**Patterns:**
- Sentinel errors defined as package-level `var` with `errors.New()` or `fmt.Errorf()`:
```go
var (
    ErrDivideByZero = errors.New("denominator cannot be zero")
    ErrTimeParser   = errors.New("failed to parse time.Time value")
)
```
- Error wrapping with `%w` for propagation: `fmt.Errorf("parsing: %w", err)`
- Error joining with `errors.Join` for aggregating multiple errors:
```go
err = errors.Join(err, readErr)
```
- Early return on error — happy path unindented:
```go
if err != nil {
    return nil, fmt.Errorf("failed to read profile: %w", err)
}
```
- Panic used only in unrecoverable situations (constructor contract violations):
```go
if typeOf.Kind() != reflect.Struct {
    panic("configuration type must be a struct")
}
```
- Recover used sparingly — only in packages that wrap external input parsing:
```go
defer func() {
    if panicErr := recover(); panicErr != nil {
        err = fmt.Errorf("panic: %v", panicErr)
    }
}()
```

## Logging

**Framework:** `log/slog` (standard library structured logging)

**Patterns:**
- Package-level logger initialization with `sync.Once`:
```go
var (
    logger     *slog.Logger
    loggerOnce sync.Once
)

func log() *slog.Logger {
    loggerOnce.Do(func() {
        logger = slog.With(slog.String("module", "config"))
    })
    return logger
}
```
- Structured attributes using `slog.String`, `slog.Int`, `slog.Any`, `slog.Group`
- Discard handler used in tests: `slog.New(slog.DiscardHandler)`
- Custom `Logger` interface defined in some packages for testability (see `config.Logger`)

**When to log:**
- Configuration initialization events at `Info` level
- Errors during loading/matching at `Error` level
- Debug-level logging gated (explicit `WithDebugLogger()` option, not used in production)
- Sensitive values redacted with `safe:"true"` struct tags → logged as `"********"`

## Comments

**Package comments:**
- Required on all packages (enforced by `godoclint`)
- Format: `// Package <name> provides <summary>.`
- Used to document purpose and usage patterns

**Function comments:**
- Present on all exported functions and methods
- Start with function name (godoc convention): `// New creates a new fraction...`
- Focus on *what* and *why*, not *how*
- Some longer doc comments include Parameters/Returns sections

**Linter-note comments:**
- `//nolint:linter1,linter2` at function level for intentional violations
- `// nocover` on unreachable code paths
- `// #nosec G705` for gosec suppressions with rationale

**TODOs:**
- Not prevalent in source code — tracked externally (no `TODO`/`FIXME` comments found in production code)

## Function Design

**Size:**
- Not enforced by a fixed line limit per function, but `funlen` linter is active
- Functions with many cases suppress linter: `//nolint:funlen` (seen on large test functions and type-switch functions like `FromFloat64`)

**Parameters:**
- Functions with 4+ parameters prefer per-line argument layout:
```go
func isCadastro(
    doc string,
    pattern *regexp.Regexp,
    size int,
    position int,
) bool {
```
- Functions with many parameters use options struct pattern (see `SetupServer` with `func(*MockHandler)` options)
- `context.Context` first parameter when present

**Return Values:**
- Tuple returns with `(value, error)` as the standard pattern
- Named returns are rare — most functions use un-named return values
- Short-lived error variables preferred: `if err != nil { return ..., err }`

## Module Design

**Exports:**
- Unexport aggressively — only what's needed by consumers
- Internal helpers and test utilities are unexported
- Interface types exported with small surface area (e.g., `Mocker` with 12 methods)
- Generic types use `[T comparable]` constraint for map-key types, `[T any]` for other uses

**Package organization:**
- Single concern per package (e.g., `set`, `fraction`, `flow`, `mid`, `release`)
- `config/` package split into sub-packages: `config/environment`, `config/profile`, `config/validation`, `config/merger`
- Sub-packages import only what they need from parent — no circular dependencies

**Constructor functions:**
- Use `New` prefix: `New[T](values ...T)`, `NewProvider[T](options...)`
- Functional options pattern for configuration: `func(*provider) providerOption`
- Builder pattern for fluent API (see `httptest_mock/builder.go`):
```go
mock := httptestmock.NewMock("GET", "/hello").
    WithResponseStatus(200).
    WithResponseBody("Hello, World!")
```

## Concurrent Patterns

**Mutex usage:**
- `sync.RWMutex` for read-prevalent scenarios (e.g., `Provider[T]` configuration)
- `sync.Mutex` for write-only or balanced scenarios (e.g., `Mock` assertion tracking)
- Lock ordering consistent within files
- Always `defer` unlock immediately after lock

**Lazy initialization:**
- `sync.Once` for singleton logger and similar patterns

---

*Convention analysis: 2026-07-21*

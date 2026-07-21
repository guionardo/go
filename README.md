# go

Golang tools, examples, and packages

[![Go Reference](https://pkg.go.dev/badge/github.com/guionardo/go.svg)](https://pkg.go.dev/github.com/guionardo/go)
[![Go tests and checking](https://github.com/guionardo/go/actions/workflows/go.yml/badge.svg)](https://github.com/guionardo/go/actions/workflows/go.yml)
![coverage](https://raw.githubusercontent.com/guionardo/go/badges/.badges/main/coverage.svg)
[![CodeQL](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql)
[![Quality Report](https://img.shields.io/badge/Quality-make%20quality--report-blue)](/quality-report.md)

## Package Index

| Package | Import | Description |
|---------|--------|-------------|
| [brdocs](#package-brdocs) | `br_docs` | Brazilian document validation (CPF, CNPJ) |
| [cache](#package-cache) | `cache` | Generic key-value cache with pluggable backends |
| [mem](#package-cache) | `cache/mem` | In-memory cache backend |
| [redis](#package-cache) | `cache/redis` | Redis cache backend |
| [valkey](#package-cache) | `cache/valkey` | Valkey cache backend |
| [memcache](#package-cache) | `cache/memcache` | Memcache cache backend |
| [postgres](#package-cache) | `cache/postgres` | PostgreSQL cache backend |
| [config](#package-config) | `config` | Typed configuration provider (YAML + env + validation) |
| [environment](#package-config) | `config/environment` | Environment variable parsing |
| [profile](#package-config) | `config/profile` | YAML profile loading and merging |
| [merger](#package-config) | `config/merger` | Recursive deep-merge of maps |
| [validation](#package-config) | `config/validation` | Struct validation |
| [flow](#package-flow) | `flow` | Generic control flow utilities (ternary, defaults) |
| [fraction](#package-fraction) | `fraction` | Immutable fraction arithmetic |
| [httptestmock](#package-httptest_mock) | `httptest_mock` | HTTP mock server framework for tests |
| [mid](#package-mid) | `mid` | Cross-platform machine ID retrieval |
| [pathtools](#package-path_tools) | `path_tools` | File and directory path utilities |
| [reflecttools](#package-reflect_tools) | `reflect_tools` | Reflection utilities (zero-value check) |
| [release](#package-release) | `release` | Self-update mechanism via GitHub Releases |
| [set](#package-set) | `set` | Generic set with algebra, JSON, SQL support |
| [shelltools](#package-shell_tools) | `shell_tools` | Shell argument parsing and env lookup |
| [timetools](#package-time_tools) | `time_tools` | Adaptive time format parsing |

## Development

Don't forget to install pre-commit and setup the commit hook.

## Packages

### Package brdocs

Import `github.com/guionardo/go/br_docs`

Validation for CPF and CNPJ

```go
func IsCPF(doc string) bool
func IsCNPJ(doc string) bool
func RemoveNonDigitAndLetters(value *string)
```

### Package cache

Import `github.com/guionardo/go/cache`

Generic key-value cache abstraction with pluggable backend providers.

```go
import "github.com/guionardo/go/cache"
```

The `Cache[K, V]` interface exposes `Get`, `Set`, `Delete`, `GetOrSet`, and `Close` — all accepting `context.Context`.

#### Providers

Each provider lives in its own sub-package and is independently importable:

| Package | Backend | Driver | Connection |
|---------|---------|--------|------------|
| `cache/mem` | In-memory | None (stdlib) | None — zero dependency |
| `cache/redis` | Redis | go-redis/v9 | Lazy — dials on first query |
| `cache/valkey` | Valkey | valkey-go | Eager — dials at construction |
| `cache/memcache` | Memcache | gomemcache | Lazy — goroutine ctx wrapper |
| `cache/postgres` | Postgres | pgx/v5 | Eager — pgxpool at construction |

#### Interface

```go
type Cache[K comparable, V any] interface {
    Get(ctx context.Context, key K) (V, error)
    Set(ctx context.Context, key K, value V, ttl ...time.Duration) error
    Delete(ctx context.Context, key K) error
    GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error)
    Close() error
}
```

#### Basic Usage

Consumer code never imports a provider directly — swap backends by changing the constructor:

```go
// In tests — zero-dependency in-memory cache
c := mem.New[string, string]()
c.Set(ctx, "mykey", "myvalue")

// In production — Redis
c := redis.New[string, string](redis.WithAddr("localhost:6379"))

// Same interface, different backend
v, err := c.Get(ctx, "mykey")
```

#### Sentinel Errors

```go
var ErrMiss   = errors.New("cache: key not found")
var ErrClosed = errors.New("cache: cache is closed")
```

Errors are wrapped with the provider prefix (`cache/redis:`, `cache/postgres:`, etc.) so callers can use `errors.Is()`.

### Package config

Import `github.com/guionardo/go/config`

Generic typed configuration provider with YAML profile loading, environment variable overrides, and struct validation.

```go
import "github.com/guionardo/go/config"

type AppConfig struct {
	Port   int    `env:"APP_PORT" default:"8080"`
	Host   string `env:"APP_HOST" default:"localhost"`
	DBPath string `env:"DB_PATH"`
}

func main() {
	provider := config.NewProvider[AppConfig]()
	cfg, err := provider.GetConfiguration()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Listening on %s:%d\n", cfg.Host, cfg.Port)
}
```

#### Provider

`Provider[T]` loads configuration from YAML profiles (with scope-based layering) and environment variables. Supports thread-safe `GetConfiguration()` and `UpdateConfiguration()`.

#### Options

- `WithProfilesPath(path)` — set base directory for YAML profile files
- `WithScope(scope)` — set active scope name (e.g. "production", "development")
- `WithDefaultScope(scope)` — set fallback scope name
- `WithLogger(logger)` — inject a custom Logger
- `WithDebugLogger()` — enable debug logging (not for production)

#### Sub-packages

- `environment` — reads configuration from environment variables into struct fields via `env` and `default` struct tags
- `profile` — loads and merges YAML profile files by scope (default + scope-specific)
- `merger` — recursive deep-merge of `map[string]any` maps
- `validation` — struct validation via `go-playground/validator` and the `Validator` interface

### Package flow

Import `github.com/guionardo/go/flow`

Simplify logic flows

```go
// Default returns the second argument (valueIfZero) when the value has the default (zero)
func Default[T comparable](value T, valueIfZero T) T

// If is a generic ternary operator
func If[T any](condition bool, valueIfTrue T, valueIfFalse T) T
```

### Package fraction

Import `github.com/guionardo/go/fraction`

This package is originally a work of Miguel Dorta [go-fraction](https://github.com/nethruster/go-fraction).

```go
type Fraction struct  // immutable, always simplified

func New[T, K integer](numerator T, denominator K) (Fraction, error)
func FromFloat64(f float64) (Fraction, error)

func (Fraction) Add(Fraction) Fraction
func (Fraction) Subtract(Fraction) Fraction
func (Fraction) Multiply(Fraction) Fraction
func (Fraction) Divide(Fraction) (Fraction, error)
func (Fraction) Float64() float64
func (Fraction) Equal(Fraction) bool
func (Fraction) Numerator() int64
func (Fraction) Denominator() int64
```

### Package httptest_mock

Import `github.com/guionardo/go/httptest_mock`

Full HTTP mock server framework for tests. Define mocks programmatically or via JSON/YAML files.

More [documentation](httptest_mock/README.md).

```go
mock := httptestmock.NewMock("GET", "/api/resource").
    WithResponseStatus(200).
    WithResponseBody(`{"key": "value"}`)

handler := httptestmock.MockHandler{}
handler.AddMocks(mock)

server := httptest.NewServer(&handler)
defer server.Close()
```

### Package mid

Import `github.com/guionardo/go/mid`

Machine identification using OS-specific sources:

- Linux: hostnamectl, /var/lib/dbus/machine-id, or /etc/machine-id
- Windows: MachineID from registry SQMClient
- macOS: "{model number}|{serial number}|{hardware uuid}" from system_profiler

```go
func MachineID() string
```

### Package path_tools

Import `github.com/guionardo/go/path_tools`

```go
func DirExists(pathName string) bool
func CreatePath(path string) error
func FileExists(fileName string) bool
func FindFileInPath(filename string) (string, error)
func GetRootFolder(base string) (string, error)
```

### Package reflect_tools

Import `github.com/guionardo/go/reflect_tools`

```go
// IsZeroValue checks if the provided value is considered a zero value.
// Handles numeric types, strings, booleans, time.Time, time.Duration,
// slices, arrays, maps, and pointers.
func IsZeroValue(value any) bool
```

### Package release

Import `github.com/guionardo/go/release`

Complete self-update mechanism for Go CLI tools distributed via GitHub Releases. Includes version detection, update checking, SHA256-verified downloads, atomic binary replacement via an embedded swapper process, and automatic relaunch.

Full documentation: [release/README.md](release/README.md)

```go
result := release.PerformSelfUpdate(context.Background())
if result.Updated {
    fmt.Println("Updated to", result.Release.TagName)
    os.Exit(0)
}
if result.Err != nil {
    log.Fatal(result.Err)
}
```

Configure with functional options:

```go
release.PerformSelfUpdate(
    context.Background(),
    release.WithOwner("myorg"),
    release.WithRepo("mycli"),
    release.WithGitHubToken(os.Getenv("GITHUB_TOKEN")),
)
```

#### Architecture

1. Reads current version from `debug.ReadBuildInfo()`
2. Checks GitHub Releases for newer version
3. Downloads the platform-specific asset matching `runtime.GOOS`/`runtime.GOARCH`
4. Verifies SHA256 digest (go-digest format)
5. Extracts the embedded swapper binary
6. Spawns swapper and exits
7. Swapper performs atomic backup-rename-replace with checksum verification
8. Swapper relaunches the new binary with original CLI args

See [release/README.md](release/README.md) for GitHub workflow examples in Go, Python, and .NET.

### Package set

Import `github.com/guionardo/go/set`

Generic set struct backed by a Go map.

```go
type Set[T comparable] map[T]struct{}

s := set.New(1, 2, 3)
s.Add(4)
s.Has(2)           // true
union := s.Union(other)
```

`Set[T]` supports JSON marshal/unmarshal and database/sql Scanner/Valuer.

### Package shell_tools

Import `github.com/guionardo/go/shell_tools`

Utilities to parse and reconstruct simple shell-like argument lists.

```go
args := shelltools.NewQuotedShellArgs(`one "two three" 'four five'`)
// args[0] = "one", args[1] = "two three", args[2] = "four five"
```

### Package time_tools

Import `github.com/guionardo/go/time_tools`

Flexible time parsing that tries multiple common layouts automatically, promoting successful templates for faster subsequent parses.

```go
t, err := timetools.Parse("2024-03-15T10:20:30Z")
// t = 2024-03-15 10:20:30 +0000 UTC

timetools.SetLayouts([]string{"2006-01-02", time.RFC3339})
t, err = timetools.Parse("2024-12-25")
```

## 🤝 Contributing

Bugs or contributions on new features can be made in the [issues page](https://github.com/guionardo/go/issues).

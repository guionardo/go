# go
Golang tools, examples, and packages

[![Go Reference](https://pkg.go.dev/badge/github.com/guionardo/go.svg)](https://pkg.go.dev/github.com/guionardo/go)
[![Go Tests](https://github.com/guionardo/go/actions/workflows/go_tests.yml/badge.svg)](https://github.com/guionardo/go/actions/workflows/go_tests.yml)
![coverage](https://raw.githubusercontent.com/guionardo/go/badges/.badges/main/coverage.svg)
[![CodeQL](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql)
[![Go Report Card](https://goreportcard.com/badge/github.com/guionardo/go)](https://goreportcard.com/report/github.com/guionardo/go)

## Table of Contents

- [go](#go)
	- [Table of Contents](#table-of-contents)
	- [Development](#development)
	- [Package brdocs](#package-brdocs)
	- [Package flow](#package-flow)
	- [Package fraction](#package-fraction)
	- [Package mid](#package-mid)
	- [Package path\_tools](#package-path_tools)
	- [Package shell\_tools](#package-shell_tools)
	- [Package set](#package-set)
	- [Package httptest\_mock](#package-httptest_mock)
	- [Package time\_tools](#package-time_tools)
	- [Package reflect\_tools](#package-reflect_tools)
	- [Package config](#package-config)
		- [Provider](#provider)
		- [Options](#options)
		- [Sub-packages](#sub-packages)
	- [🤝 Contributing](#-contributing)

## Development

Don't forget to install pre-commit and setup the commit hook.

## Package brdocs

Validation for CPF and CNPJ

```go
// IsCPF verifies if the given string is a valid CPF document.
// Punctuation will be automatically removed
func IsCPF(doc string) bool

// IsCNPJ verifies if the given string is a valid CNPJ document.
// Punctuation will be automatically removed. Rules for new alfanumeric format.
func IsCNPJ(doc string) bool

// RemoveNonDigitAndLetters updates the value, keeping only 0-9, A-Z characters
func RemoveNonDigitAndLetters(value *string)
```

## Package flow

Simplify logic flows

```go
// Default returns the second argument (valueIfZero) when the value has the default (zero)
func Default[T comparable](value T, valueIfZero T) T

// If is a generic ternary operator
func If[T any](condition bool, valueIfTrue T, valueIfFalse T) T
```

## Package fraction

This package is originally a work of Miguel Dorta [go-fraction](https://github.com/nethruster/go-fraction).

I needed to encapsulate in this repository due to release name restrictions.

```go
// Fraction represents a fraction. It is an immutable type.
//
// It is always a valid fraction (never x/0) and it is always simplified.
type Fraction struct
```

## Package mid

Machine Identification using data from operational system trying to detect unique machine

* Linux: hostnamectl, /var/lib/dbus/machine-id, or /etc/machine-id
* Windows: MachineID from registry SQMClient
* MacOS: "{model number}|{serial number}|{hardware uuid}" from system_profiler (under validation)

```go
// Machine ID
func MachineID() string
```

## Package path_tools

```go
// DirExists simply returns true if the pathName is a existing directory
func DirExists(pathName string) bool

// CreatePath Create full path, with permissions updated from parent folder.
func CreatePath(path string) error

// FileExists symply returns true if the fileName is a existing file
func FileExists(fileName string) bool

// FindFileInPath searches for a file in the paths from the PATH environment variable
// returns the first occurrence or error
// Handles OS-specific path separators
func FindFileInPath(filename string) (string, error)
```

## Package shell_tools

Utilities to parse and reconstruct simple shell-like argument lists.

- Type: `QuotedShellArgs` — a []string wrapper that holds parsed arguments.
- Constructor: `NewQuotedShellArgs(s string) QuotedShellArgs` — parse input string into `QuotedShellArgs`.
- Method: `QuotedShellArgs.String() string` — join arguments back into a shell-safe string (quotes added as needed).

Behavior:
- Supports single `'` and double `"` quotes. Quotes are removed from parsed arguments.
- Splits on whitespace.
- Reconstructs a shell-like string adding quotes only when needed.

Example:

```go
package main

import (
    "fmt"

    "github.com/guionardo/go/shell_tools"
)

func main() {
    input := `one "two three" 'four five' six\ seven`
    args := shell_tools.NewQuotedShellArgs(input)

    // args is a QuotedShellArgs (slice of strings)
    fmt.Printf("%q\n", []string(args)) // ["one" "two three" "four five" "six seven"]
    fmt.Println(args.String())         // one "two three" "four five" "six seven"
}
```

## Package set

Generic set struct

```go
// Set values methods
type Set[T comparable] map[T]struct{}
```

`Set[T]` can be [un]marshaled and respects the Scanner and Valuer interfaces

```go
type Scanner = database/sql.Scanner
type Valuer = database/sql/driver.Valuer
```

## Package httptest_mock

Utilities for mocking HTTP servers in tests.

- Easily create mock HTTP servers with custom handlers.
- Record requests and responses for assertions.
- Supports setting up expected responses and verifying received requests.
- More [documentation](httptest_mock/README.md)


## Package time_tools

Flexible time parsing utility that tries multiple common layouts automatically, promoting successful templates for faster subsequent parses.

```go
// Parse attempts to parse a time string using multiple common layouts
// (RFC3339, DateTime, DateOnly, Kitchen, ANSIC, etc.).
// The matched template is promoted to the front for future calls.
func Parse(s string) (time.Time, error)

// SetLayouts replaces the global layouts list with a custom set of
// time format strings, in priority order.
func SetLayouts(newLayouts []string)
```

Example:

```go
t, err := timetools.Parse("2024-03-15T10:20:30Z")
// t = 2024-03-15 10:20:30 +0000 UTC

timetools.SetLayouts([]string{"2006-01-02", time.RFC3339})
t, err = timetools.Parse("2024-12-25")
// t = 2024-12-25 00:00:00 +0000 UTC
```

## Package reflect_tools

Utilities for working with Go's reflection, including zero value checks.

```go
// IsZeroValue checks if the provided value is considered a zero value.
// Handles numeric types, strings, booleans, time.Time, time.Duration, slices, arrays, maps, and pointers.
// Returns true if the value is zero, nil or empty, false otherwise.
func IsZeroValue(value any) bool
```

## Package config

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

### Provider

`Provider[T]` loads configuration from YAML profiles (with scope-based layering) and environment variables. Supports thread-safe `GetConfiguration()` and `UpdateConfiguration()`.

### Options

- `WithProfilesPath(path)` — set base directory for YAML profile files
- `WithScope(scope)` — set active scope name (e.g. "production", "development")
- `WithDefaultScope(scope)` — set fallback scope name
- `WithLogger(logger)` — inject a custom Logger
- `WithDebugLogger()` — enable debug logging (not for production)

### Sub-packages

- `environment` — reads configuration from environment variables into struct fields via `env` and `default` struct tags
- `profile` — loads and merges YAML profile files by scope (default + scope-specific)
- `merger` — recursive deep-merge of `map[string]any` maps
- `validation` — struct validation via `go-playground/validator` and the `Validator` interface

## 🤝 Contributing

Bugs or contributions on new features can be made in the [issues page](https://github.com/guionardo/go/issues).

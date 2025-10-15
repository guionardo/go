# go
Golang tools, examples, and packages

[![Go Reference](https://pkg.go.dev/badge/github.com/guionardo/go.svg)](https://pkg.go.dev/github.com/guionardo/go)
[![Go Tests](https://github.com/guionardo/go/actions/workflows/go_tests.yml/badge.svg)](https://github.com/guionardo/go/actions/workflows/go_tests.yml)
![coverage](https://raw.githubusercontent.com/guionardo/go/badges/.badges/main/coverage.svg)
[![CodeQL](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql)

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


## Package set

Generic set struct

```go
// Set values methods
type Set[T comparable] map[T]struct{}
```

`Set[T]` can be [un]marshaled and respects the Scanner and Valuer interfaces

```go
type Scanner interface {
	Scan(value interface{}) error
}
type Valuer interface {
	Value() (driver.Value, error)
}
```

## 🤝 Contributing

Bugs or contributions on new features can be made in the [issues page](https://github.com/guionardo/go/issues).

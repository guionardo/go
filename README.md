# go
Golang tools, examples, and packages

[![Go Tests](https://github.com/guionardo/go/actions/workflows/go_tests.yml/badge.svg)](https://github.com/guionardo/go/actions/workflows/go_tests.yml)
![coverage](https://raw.githubusercontent.com/guionardo/go/badges/.badges/main/coverage.svg)
[![CodeQL](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql)

## Development

Don't forget to install pre-commit and setup the commit hook.

## Package flow

Simplify logic flows

```go
// If is a generic ternary operator
func If[T any](condition bool, valueIfTrue T, valueIfFalse T) T
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

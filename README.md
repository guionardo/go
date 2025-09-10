# go
Golang tools, examples, and packages

[![Go Tests](https://github.com/guionardo/go/actions/workflows/go_tests.yml/badge.svg)](https://github.com/guionardo/go/actions/workflows/go_tests.yml)
![coverage](https://raw.githubusercontent.com/guionardo/go/badges/.badges/main/coverage.svg)
[![CodeQL](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql)

## Development

Don't forget to install pre-commit and setup the commit hook.

## Package path_tools

```go
// DirExists simply returns true if the pathName is a existing directory
func DirExists(pathName string) bool

// CreatePath Create full path, with permissions updated from parent folder.
func CreatePath(path string) error

// FileExists symply returns true if the fileName is a existing file
func FileExists(fileName string) bool

// Set values methods
type Set[T comparable] map[T]struct{}
```

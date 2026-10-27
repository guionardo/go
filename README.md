# go

Golang tools, examples, and packages.

[![Go Reference](https://pkg.go.dev/badge/github.com/guionardo/go.svg)](https://pkg.go.dev/github.com/guionardo/go)
[![Go tests and checking](https://github.com/guionardo/go/actions/workflows/go.yml/badge.svg)](https://github.com/guionardo/go/actions/workflows/go.yml)
![coverage](https://raw.githubusercontent.com/guionardo/go/badges/.badges/main/coverage.svg)
[![CodeQL](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/guionardo/go/actions/workflows/github-code-scanning/codeql)
[![Quality Report](https://img.shields.io/badge/Quality-make%20quality--report-blue)](/quality-report.md)

A collection of reusable Go libraries — configuration, data structures, validators,
caching, project detection, CLI self-update. Every package is independently importable
and documented in its own README; the API reference lives on
[pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go).

## Installation

Requires **Go 1.26.4 or newer**. Add the module to your `go.mod`:

```bash
go get github.com/guionardo/go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/guionardo/go/cache/mem"
	"github.com/guionardo/go/flow"
)

func main() {
	ctx := context.Background()
	c := mem.New[string, string](ctx) // in-memory cache: default TTL, sweep, expired-entry cleanup

	_ = c.Set(ctx, "greeting", "hello")

	v, _ := c.Get(ctx, "greeting")
	fmt.Println(flow.If(len(v) > 0, v, "default"))
}
```

## Packages

| Package | What it does | Documentation |
|---------|--------------|---------------|
| `br_docs` | Brazilian document validation — CPF and CNPJ | [README](./br_docs/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/br_docs) |
| `cache` | Generic key-value cache with 6 pluggable backends: mem, Redis, Valkey, Memcache, Postgres, SQLite | [README](./cache/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/cache) |
| `config` | Typed configuration provider — YAML profiles, environment overrides, validation | [README](./config/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/config) |
| `flow` | Control-flow helpers — ternary `If`, zero-value `Default`, insertion-ordered `OrderedMap` | [README](./flow/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/flow) |
| `fraction` | Immutable fraction arithmetic over `int64` | [README](./fraction/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/fraction) |
| `httptest_mock` | HTTP mock server for tests, with requests declared in code or in fixture files | [README](./httptest_mock/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/httptest_mock) |
| `mid` | Cross-platform machine identifier (Linux, macOS, Windows) | [README](./mid/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/mid) |
| `path_tools` | File and directory path helpers — existence, creation, Go-root detection | [README](./path_tools/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/path_tools) |
| `project_probe` | Never-fail project detection across Go, Python, JavaScript, Rust, Java, PHP and C#/.NET | [README](./project_probe/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/project_probe) |
| `reflect_tools` | Zero-value checking through reflection | [README](./reflect_tools/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/reflect_tools) |
| `release` | Self-update through GitHub Releases — version check, digest verification, atomic swap, relaunch | [README](./release/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/release) |
| `set` | Generic set with algebra (union, intersect, diff), JSON, and SQL scan/valuer support | [README](./set/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/set) |
| `shell_tools` | Quoted shell-argument parsing and environment lookup | [README](./shell_tools/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/shell_tools) |
| `time_tools` | Time parsing that tries a prioritized list of layouts | [README](./time_tools/README.md) · [API](https://pkg.go.dev/github.com/guionardo/go/time_tools) |

## Development

Install the tooling and the commit hook once:

```bash
make deps
pre-commit install
```

| Target | Purpose |
|--------|---------|
| `make test` | Run the full test suite |
| `make coverage-quick` | Unit-only coverage gate — packages ≥80%, files ≥70%, total ≥75% |
| `make coverage` | Unit + Docker E2E coverage, merged and gated |
| `make lint` / `make lint-fix` | `golangci-lint` |
| `make benchmark` / `make benchmark-quick` | Benchmark suite (mem always; Docker-backed providers when available) |
| `make quality-report` | Regenerate `quality-report.md` (lint, vulns, coverage, metrics) |
| `make swapper` | Regenerate the self-update binaries embedded by `release/` |

**Clean checkout:** `release/embed_swapper.go` embeds the generated
`release/swapper/swapper_*` binaries, so run `make swapper` before `go build ./...`
or `go test ./...` on a fresh clone.

Commits follow [Conventional Commits](https://www.conventionalcommits.org/); every
package documents itself through `doc.go`, a runnable `example_test.go`, and its own
`README.md` — see [AGENTS.md](./AGENTS.md#documentation) for the rules.

## Contributing

Bugs and feature requests belong in the [issue tracker](https://github.com/guionardo/go/issues).
See [CONTRIBUTING.md](./CONTRIBUTING.md) for the workflow.

## License

Released under the [MIT License](./LICENSE). Copyright (c) 2025 Guionardo Furlan.

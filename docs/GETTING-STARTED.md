<!-- generated-by: gsd-doc-writer -->
# Getting Started

This guide covers the prerequisites, installation steps, and first-run instructions for
`github.com/guionardo/go` — a Go module of reusable utility packages (config, cache, set, flow,
and more). It is a **library module**, not a standalone CLI: there is nothing to build or run at the
top level, so most of this guide is about setting up the development environment to work on and test
the packages.

## Prerequisites

### Go toolchain

The module requires **Go 1.26.4 or newer** (declared in [`go.mod`](../go.mod); there is no
`toolchain` directive, and `golang.org/x/tools v0.48.0` is only an indirect dependency). The module's
`tool` block lists the helper binaries `gosec` and `govulncheck`. In practice the project targets the
latest toolchain; install the required version via your system package manager or the official
installers.

```bash
go version  # should report go1.26.4 or newer
```

### Build / lint / test tools

The accompanying [`Makefile`](../Makefile) manages most developer tooling. `make deps` installs the
following, reusing your shell's package manager (`brew` on macOS, `apt` on Debian/Ubuntu):

| Tool | Purpose | Installed by |
|------|---------|--------------|
| [pre-commit](https://pre-commit.com) | Git hooks (commit conventions, fmt, lint, test, vuln) | `make install-pre-commit` |
| [golangci-lint](https://golangci-lint.run) | Linting configured in [`.golangci.yml`](.golangci.yml) | `make install-golangci` |
| [commitlint](https://github.com/conventionalcommit/commitlint) | Enforces Conventional Commits | `make install-commitlint` |
| [govulncheck](https://golang.org/x/vuln) | Vulnerability scanning (also a module `tool`) | `make install-govulncheck` |
| [go-test-coverage](https://github.com/vladopajic/go-test-coverage) | Enforces coverage thresholds | `make install-go-test-coverage` |
| gocovmerge | Merges E2E + unit coverage profiles (only needed for `make coverage`) | `make check-gocovmerge` |

The go.mod `tool` block installs only `gosec` and `govulncheck`. The others are installed by the
Makefile rather than `tool` directives: `golangci-lint` via `curl` of its install script,
`go-test-coverage` and `commitlint` via `go install`.

### PATH requirement

The `deps` and `test-e2e` targets verify that the Go binary folder is on your `PATH`.
If `make` reports `GO BIN folder is not in PATH`, add it to your shell profile:

```bash
# macOS / Linux
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Docker (optional)

Running the E2E tests (`make test-e2e`) and the full coverage report (`make coverage`) requires a
running Docker daemon because the cache providers (Redis, Valkey, Memcache, Postgres) are tested in
containers. The Makefile defaults `DOCKER_HOST` to an OrbStack socket; override it on the command
line if you use Docker Desktop or a remote host:

```bash
make test-e2e DOCKER_HOST=unix:///var/run/docker.sock
```

## Installation steps

Clone the repository and install developer dependencies:

```bash
# 1. Clone (SSH)
git clone git@github.com:guionardo/go.git
cd go

# 2. Install all development dependencies (Go toolchain + hooks + linters)
make deps
```

If you prefer HTTPS instead of SSH:

```bash
git clone https://github.com/guionardo/go.git
```

### Consuming the library (as a dependency)

As an open-source Go module, consumers add it with `go get` rather than cloning:

```bash
go get github.com/guionardo/go
```

All packages are independently importable from the single module, for example:

```go
import "github.com/guionardo/go/flow"
import "github.com/guionardo/go/cache"
import "github.com/guionardo/go/cache/mem"
import "github.com/guionardo/go/config"
```

## First run

Because this is a library, the fastest "working output" is a passing test run of the whole module.
From the repository root:

```bash
make deps        # once
make test        # or: go test ./... -v
```

Alternatively, write a tiny program that imports one of the packages. The in-memory cache
(`cache/mem`) and the control-flow helpers (`flow`) have no external runtime dependencies, so they
are a good first smoke test:

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
	c := mem.New[string, string](ctx) // in-memory cache with default TTL, sweep + expired-entry cleanup
	_ = c.Set(ctx, "greeting", "hello")
	v, _ := c.Get(ctx, "greeting")
	fmt.Println(flow.If(v != "", "default", "fallback"))
}
```

Run it:

```bash
go run .
```

For a config-based example using `config` and `config/profile`, see the README's `Package config`
section. All package usage examples live in [README.md](README.md#packages).

## Common setup issues

### pre-commit hooks are not active

The quickest source of friction is forgetting to install the Git hooks. Run:

```bash
make install-pre-commit
```

This runs `pre-commit autoupdate` and installs the hooks for both `commit-msg` and `pre-commit`
stages. You can also run hooks manually (headlessly) at any time:

```bash
pre-commit run --all-files
```

The pre-commit config (see [`.pre-commit-config.yaml`](.pre-commit-config.yaml)) enforces, among
others, `commitlint` on commit messages, `go mod tidy`, `gofmt -s`, `go test ./...`,
`golangci-lint run`, and `govulncheck`.

### Go binary folder not on PATH

`make` validates that `$(go env GOPATH)/bin` is on your `PATH` (`gocheck`/`check-golangci`/etc.).
If it isn't, `make` exits with `GO BIN folder is not in PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
# optionally make it permanent in ~/.zshrc / ~/.bashrc
```

### Wrong or missing Go version

If `go build` or `make test` fails with a toolchain or directive error, your installed Go is older
than 1.26.4. Upgrade the toolchain (e.g., `brew upgrade go` on macOS) and re-run.

### E2E / coverage tasks require a running Docker daemon

`make test-e2e` and `make coverage` spin up containers for the Redis, Postgres, Memcache, and
Valkey backends. Without Docker those targets fail; pass a valid `DOCKER_HOST` or start Docker
first. If your container host is not the default, override it (see the example above).

## Next steps

- See [DEVELOPMENT.md](DEVELOPMENT.md) for the local development workflow, build commands, code
  style, branch conventions, and the PR process.
- See [TESTING.md](TESTING.md) for test commands, coverage thresholds, and how tests run in CI.
- See [ARCHITECTURE.md](ARCHITECTURE.md) for a component overview of the module.
- See [CONFIGURATION.md](CONFIGURATION.md) for the `config` package's typed configuration details.

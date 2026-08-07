<!-- generated-by: gsd-doc-writer -->
# TESTING.md

Testing guide for the `github.com/guionardo/go` module — a collection of Go utility packages
(config, data structures, validators, cache, CLI self-update).

## Test Framework and Setup

The project uses the standard Go testing toolchain (`go test`) with the
[testify](https://github.com/stretchr/testify) assertion library (`v1.11.1`), which provides
`assert` and `require` packages for assertions in tests.

Additional testing dependencies (from `go.mod`):

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/stretchr/testify` | v1.11.1 | Assertions (`assert`) and fatal assertions (`require`) |
| `github.com/testcontainers/testcontainers-go` | v0.43.0 | E2E integration tests against real servers via Docker |
| `github.com/testcontainers/testcontainers-go/modules/redis` | v0.43.0 | Redis container for cache E2E tests |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.43.0 | Postgres container for cache E2E tests |

There is no separate test framework configuration file (no `jest.config`, `.nycrc`, etc.) —
tests are discovered by `go test` using the `_test.go` suffix convention.

### Required setup

- **Go toolchain** — the version pinned in `go.mod` (`go 1.26.4`) is used by CI via
  `go-version-file: go.mod`.
- **Dependencies** — run `make deps` (or `go mod download`) to install the module dependencies.
- **Docker (E2E only)** — the cache E2E tests require a running Docker daemon to spin up
  Redis, Valkey, Memcache, and Postgres containers. The Makefile defaults
  `DOCKER_HOST` to a local OrbStack socket (`unix:///Users/guionardo/.orbstack/run/docker.sock`).

## Running Tests

All test commands are exposed as Makefile targets under the `##@ Testing` section.

### Full test suite (unit + example tests)

```bash
make test
```

Runs `go test ./... -v` — the verbose output of every package test.

### Single package, file, or test

```bash
# A single package
go test ./cache/mem -v

# A single test function in a package
go test ./cache/mem -run TestMem -v

# With the race detector
go test ./... -race
```

### E2E integration tests (requires Docker)

```bash
make test-e2e
```

Runs the cache E2E suite against real containers (Redis, Valkey, Memcache, Postgres):

```bash
DOCKER_HOST=unix:///Users/guionardo/.orbstack/run/docker.sock \
  go test ./cache/ -tags=e2e -run 'TestCacheE2E' -v -count=1 -timeout=300s
```

The E2E tests live in `cache/cache_e2e_test.go` behind a `//go:build e2e` build tag, so they are
excluded from a plain `go test ./...` run.

### Coverage

```bash
# Full coverage check (E2E + unit, merged) — requires Docker and gocovmerge
make coverage

# Quick coverage check (unit only, no Docker) — used before every commit
make coverage-quick
```

`make coverage` runs E2E and unit coverage separately and merges the profiles with `gocovmerge`
before validating against `.testcoverage.yml`. `make coverage-quick` runs only unit tests and
validates against `.testcoverage-quick.yml`.

## Writing New Tests

### File naming convention

- Unit tests live in `*_test.go` files next to the source files they test
  (e.g., `reflect_tools/reflect_tools_test.go`, `time_tools/parser_test.go`).
- White-box tests that reach into unexported identifiers use the `_internal_test.go` suffix
  (e.g., `cache/redis/redis_internal_test.go`, `cache/redis/options_internal_test.go`).
- Runnable documentation examples use `example_test.go` and are executed by `go test`
  (e.g., `cache/example_test.go`, `time_tools/example_test.go`).
- Tests use external test packages (`package foo_test`) for black-box API testing, and the
  same package (`package foo`) for internal tests.

### Test structure

Tests are **table-driven** — a `tests := []struct{...}` slice of cases, iterated with
`t.Run(name, ...)` for sub-tests. Use `testify` for assertions:

```go
func TestParse(t *testing.T) {
	t.Parallel()

	timetools.SetLayouts(defaultLayouts)

	tests := []struct {
		name    string
		s       string
		want    time.Time
		wantErr bool
	}{
		{name: "parse RFC3339", s: "2024-03-15T10:20:30Z", want: ..., wantErr: false},
		// ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := timetools.Parse(tt.s)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

### Conventions (see `AGENTS.md`)

- **`t.Parallel()`** — call it at the top of independent tests and sub-tests to run them
  concurrently. Exception: tests that modify global state (e.g., `collectFuncs` in `mid/`)
  must **not** use `t.Parallel()` to avoid races between sub-tests.
- **`testify`** — use `require` for fatal assertions (stop the test on failure) and `assert`
  for non-fatal ones. Use `t.Helper()` on shared test helper functions.
- **Platform differences** — CI runs on Linux, macOS, and Windows. Windows quirks to keep in
  mind: HTTP header keys are not always canonicalized (`req.Header.Get` may see `Api_key`
  instead of `Api-Key` — normalize headers instead); `os.LookupEnv` is case-insensitive;
  `os.WriteFile` with 0755 mode yields 0666 on Windows (skip permission assertions); test
  suites that mutate global state must not run sub-tests in parallel.
- **E2E tests** — guarded by a `//go:build e2e` build tag and exercised through the
  `cache.Cache` interface against testcontainers-managed servers.

## Coverage Requirements

Coverage is enforced by [`go-test-coverage`](https://github.com/vladopajic/go-test-coverage)
(`v2`) using the profiles configured in `.testcoverage.yml` (full, E2E + unit) and
`.testcoverage-quick.yml` (unit only). Both files define the same thresholds:

| Type | Threshold |
|------|-----------|
| File | >= 70% |
| Package | >= 80% |
| Total | >= 75% |

### Per-package overrides

The following packages are exempted or relaxed because they require real servers or are hard to
unit test (from `.testcoverage-quick.yml` / `.testcoverage.yml`):

| Package | Override threshold | Reason |
|---------|--------------------|--------|
| `cache/memcache` | 0 | Requires a real Memcache server (covered via E2E with Docker) |
| `cache/postgres` | 0 | Requires a real Postgres server (covered via E2E with Docker) |
| `cache/redis` | 0 | Requires a real Redis server (covered via E2E with Docker) |
| `cache/valkey` | 0 | Requires a real Valkey server (covered via E2E with Docker) |
| `cmd/example-updater` | 0 | CLI main package using `os.Exit()` |
| `release/swapper` | 0 | CLI main package using `os.Exit()` |
| `mid` | 50 | Platform-specific machine ID |
| `release` | 70 | Close to the default threshold |

A commit that drops below these thresholds will fail the coverage gate — run
`make coverage-quick` before committing (per `AGENTS.md`) to verify it passes.

## CI Integration

Tests run in the **`Go tests and checking`** workflow (`.github/workflows/go.yml`).

### `test` job — cross-platform tests

- **Triggers:** push and pull_request on `main` and `develop`.
- **Matrix:** `ubuntu-latest`, `macos-latest`, `windows-latest`.
- **Steps:** checkout → set up Go from `go-version-file: go.mod` → `make swapper` (builds the
  self-update swapper binaries for all platforms) → `go test -v ./...`.
- **Note:** E2E tests are excluded from CI because they require Docker; CI runs the plain unit
  test suite.

### `coverage` job — coverage gate

- **Runs on:** `ubuntu-latest`.
- **Steps:** checkout → set up Go → `make swapper` → `go test ./... -coverprofile=./cover.out
  -covermode=atomic -coverpkg=./...` → `vladopajic/go-test-coverage@v2` with
  `config: ./.testcoverage.yml` and `threshold-total: 75`.
- **Badges:** the action commits a coverage badge to the `badges` branch when the workflow runs
  on `main` (`git-token` is set only for the main branch).

### Local equivalents

Run the same checks locally with:

```bash
make test
make coverage-quick   # unit coverage gate (also required before commits)
```
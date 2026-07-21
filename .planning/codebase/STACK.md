# Technology Stack

**Analysis Date:** 2026-07-21

## Languages

**Primary:**
- Go 1.26.4 - All source code, defined in `go.mod`

**Secondary:**
- YAML - Configuration profiles (tested via `config/profile/profile.go`), CI workflows (`.github/workflows/go.yml`), pre-commit config (`.pre-commit-config.yaml`), commitlint config (`.commitlint.yaml`), golangci-lint config (`.golangci.yml`)
- JSON - Mock definitions for HTTP tests (`httptest_mock/`), release API responses (`release/release.go`)
- Markdown - Documentation (`README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, package READMEs)

## Runtime

**Environment:**
- Go 1.26.4 (compiled, direct native binaries for Linux, macOS, Windows)

**Package Manager:**
- Go modules (`go mod`)
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- Standard library - All functionality uses Go stdlib: `net/http`, `encoding/json`, `os/exec`, `reflect`, `database/sql/driver`, `log/slog`
- No web framework or application framework is used

**Testing:**
- `github.com/stretchr/testify` v1.11.1 - Assertions (`assert`, `require`) in all test files
- `testing` (stdlib) - Test runner
- `net/http/httptest` (stdlib) - HTTP test server infrastructure via `httptest_mock/` package
- `go-test-coverage` v2 (`github.com/vladopajic/go-test-coverage/v2`) - Coverage enforcement in CI

**Build/Dev:**
- `golangci-lint` - Comprehensive linting with ~40+ linters enabled
- `gofmt` / `goimports` / `golines` - Code formatting (configured in `.golangci.yml`)
- `pre-commit` - Git hooks framework (`.pre-commit-config.yaml`)
- `commitlint` (`github.com/conventionalcommit/commitlint`) - Conventional commits enforcement
- `govulncheck` (`golang.org/x/vuln/cmd/govulncheck`) - Vulnerability scanning
- `goreportcard` - Public code quality reporting (triggered via GitHub Actions)

## Key Dependencies

**Critical:**
- `github.com/go-playground/validator/v10` v10.30.3 - Struct validation in `config/validation/validator.go`, `httptest_mock/mock.go`
- `gopkg.in/yaml.v3` v3.0.1 - YAML marshaling/unmarshaling for config profiles (`config/profile/profile.go`, `config/provider.go`)
- `github.com/stretchr/testify` v1.11.1 - Test assertions across all packages
- `github.com/opencontainers/go-digest` v1.0.0 - Content digest verification in `release/release.go` (asset download integrity)

**Infrastructure:**
- `golang.org/x/sync` v0.21.0 - Sync primitives (available for use across packages)
- `golang.org/x/crypto` v0.53.0 - Indirect dependency (via validator)
- `golang.org/x/text` v0.38.0 - Indirect dependency (via validator)
- `golang.org/x/sys` v0.46.0 - Indirect dependency (via validator)

## Configuration

**Environment:**
- Config package (`config/`) uses struct tags `env:"VAR_NAME"` and `default:"value"` for environment variable binding
- Environment variable overrides for configuration scope: `SCOPE`, `DEFAULT_SCOPE`, `PROFILES_PATH`, `CONFIGURATION_LOG`
- `.env` file is gitignored (noted in `.gitignore`)

**Build:**
- `Makefile` - Common targets: `test`, `lint`, `lint-fix`, `coverage`, `deps`
- `.golangci.yml` - Linter configuration (Go 1.26 target, 120 line length, ~40 linters)
- `.testcoverage.yml` - Coverage thresholds: file 70%, package 80%, total 95%
- `.pre-commit-config.yaml` - Pre-commit hooks: conventional commits, gofmt, go mod tidy, go test, golangci-lint, govulncheck
- `.commitlint.yaml` - Conventional commits (v0.10.1): feat/fix/docs/style/refactor/perf/test/build/ci/chore/revert

## Platform Requirements

**Development:**
- Go 1.26+
- `make` for task automation
- `pre-commit`, `golangci-lint`, `commitlint`, `govulncheck` (installable via `make deps`)
- Cross-platform: Linux, macOS, Windows (tested in CI matrix)

**Production:**
- This is a library/utility module with no deployable application — consumption via `go get github.com/guionardo/go`
- CI pipeline validates across ubuntu-latest, macos-latest, windows-latest

---

*Stack analysis: 2026-07-21*

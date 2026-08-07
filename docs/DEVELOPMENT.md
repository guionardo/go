<!-- generated-by: gsd-doc-writer -->
# Development

This guide covers how to set up the repo locally for contributing to the Go utility packages at `github.com/guionardo/go`, the commands available via the `Makefile`, the code-style and quality gates enforced by CI, and the branch/PR workflow.

## Local setup

The project is a Go module (`github.com/guionardo/go`) requiring **Go 1.26+** (see `go.mod`). No external services are required for basic development — Redis, Valkey, Memcache, and Postgres are only needed for the E2E coverage runs of the `cache` package providers.

1. **Fork and clone** the repository:

   ```bash
   git clone git@github.com:guionardo/go.git
   cd go
   ```

2. **Install the development tooling** (go-test-coverage, golangci-lint, commitlint, govulncheck, pre-commit, and the git hooks):

   ```bash
   make deps
   ```

   `make deps` runs `gocheck` (verifies Go is installed and on `PATH`), installs the individual tools, then installs pre-commit and its `commit-msg`/`pre-commit` hooks. It also runs `pre-commit autoupdate` to keep hook revisions current.

3. **Verify the setup** by running the lint and quick coverage gates:

   ```bash
   make lint
   make coverage-quick
   ```

   Both must pass cleanly before you start committing.

## Build commands

The `Makefile` provides the following targets (see `make help` for a self-describing list):

| Command | Description |
|---------|-------------|
| `make help` | Print the help text listing all targets |
| `make gocheck` | Verify the Go toolchain and `$GOBIN` are on `PATH` |
| `make deps` | Install/update tooling and pre-commit hooks |
| `make test` | Run all unit tests (`go test ./... -v`) |
| `make test-e2e` | Run the `cache` E2E integration tests (requires a Docker host via `DOCKER_HOST`) |
| `make coverage` | Comprehensive coverage check: runs E2E + unit coverage, merges profiles with `gocovmerge`, and enforces `.testcoverage.yml` thresholds |
| `make coverage-quick` | Quick coverage gate: unit tests only, enforces `.testcoverage-quick.yml` thresholds; run before every commit |
| `make lint` | Run all linters (`golangci-lint run ./...`) |
| `make lint-fix` | Run all linters with auto-fix (`golangci-lint run --fix ./...`) |
| `make quality-report` | Generate a quality report (lint, security, coverage, metrics) into `quality-report.md` |
| `make check-go-test-coverage` | Verify `go-test-coverage` is installed (run `make deps` if not) |
| `make swapper` / `swapper-linux` / `swapper-darwin` / `swapper-windows` | Cross-compile the `release/swapper` self-update binaries |
| `make swapper-clean` | Remove the swapper binaries |

The environment variables `GOBIN` (default `$(go env GOPATH)/bin`) and `DOCKER_HOST` (default `unix:///Users/guionardo/.orbstack/run/docker.sock`) configure tool paths and the Docker endpoint used by the E2E coverage runs.

## Code style

Code style is enforced through `gofmt`, `go vet`, and `golangci-lint`, layered with pre-commit hooks and a Conventional Commits gate.

- **Formatting**: `gofmt`, `goimports`, and `golines` (target max length 120) are enabled as formatters in `.golangci.yml` (the gofmt `-s` simplification is disabled there; it is applied only by the pre-commit hook).
- **Linting**: `golangci-lint` runs a broad linter set (including `staticcheck`, `gosec`, `govet`, `errcheck`, `godoclint`, `misspell`, and many more) configured in `.golangci.yml` (`version: "2"`, `go: "1.26"`, new-issue-only diffing against `main`). Run it with `make lint` (auto-fix with `make lint-fix`).
- **Pre-commit hooks** (`.pre-commit-config.yaml`) run on every staged commit: `gofmt`, `go test ./...`, `golangci-lint run`, `go mod tidy`, `commitlint`, and a vulnerability check via the `govulncheck` hook. `no-commit-to-branch` blocks direct commits to `main`.
- **Commit message format**: Conventional Commits, enforced by `commitlint` (`.commitlint.yaml`) — header length 10–72 chars, body lines ≤72 chars, and a fixed type enum (`feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`).

Write Go doc comments for all exported symbols and use `testify` for test assertions; keep functions focused and single-purpose, per project conventions.

## Coverage thresholds

Coverage is enforced with [`go-test-coverage`](https://github.com/vladopajic/go-test-coverage) via two config files: `.testcoverage.yml` (full, includes E2E) and `.testcoverage-quick.yml` (unit-only, run before each commit).

| Type | Threshold |
|------|-----------|
| Total project coverage | ≥ 75% |
| Per-package coverage | ≥ 80% |
| Per-file coverage | ≥ 70% |

The same thresholds are configured in both files. File/package overrides apply in `cache/memcache`, `cache/postgres`, `cache/redis`, and `cache/valkey` (threshold 0 — providers are tested via E2E with Docker), `cmd/example-updater` and `release/swapper` (threshold 0 — CLI `main` packages with `os.Exit`), `mid` (threshold 50% — platform-specific machine ID), and `release` (threshold 70%).

## Branch conventions

The default branch is `main`. Work on a dedicated branch for each change set, and open a pull request to `main` when complete:

- **Milestone/feature branches** follow the convention `gsd/v{VERSION}-{slug}` (e.g., `gsd/v1.6-cache-dedup`). Other descriptive prefixes such as `feature/...` and `copilot/sub-pr-...` are also used in the repo.
- Material changes should go through the PR flow described below; direct pushes to `main` are blocked by the `no-commit-to-branch` pre-commit hook.

## PR process

CI runs on GitHub Actions via the workflows in `.github/workflows/`: `go.yml` runs cross-platform tests (ubuntu/macos/windows) plus the coverage check, `opencode.yml` runs agent checks, and `release.yml` publishes releases. Linting is not part of the CI workflows — it is enforced locally via `make lint` and the pre-commit hooks. This runs alongside `CONTRIBUTING.md` and project conventions:

1. Run `make test` and `make lint` and confirm both pass before submitting.
2. Run `make coverage-quick` and confirm it passes; it enforces the thresholds above. Fix uncovered code or add tests if it fails (cache providers are exempt — they need Docker E2E coverage, enforced separately by `make coverage`).
3. Add tests for any new code or functionality; keep PRs focused on a single concern and avoid mixing unrelated changes.
4. Commit with a Conventional Commits message (enforced by the `commitlint` hook; e.g., `feat:`, `fix:`, `docs:`, `refactor:`, `test:` — see `.commitlint.yaml`).
5. Before pushing a release tag, regenerate and stage the quality report: `make quality-report && git add quality-report.md`.
6. Open the PR against `main` and ensure all CI checks pass.
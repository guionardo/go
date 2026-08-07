# Contributing

Welcome! We appreciate your interest in contributing to this project.

## Prerequisites

- Go 1.26+
- pre-commit
- golangci-lint
- commitlint
- govulncheck
- go-test-coverage

## Development Setup

Install all tooling dependencies:

```bash
make deps
```

Common targets:

- `make test` — run all tests
- `make lint` — run golangci-lint
- `make coverage` — run tests with coverage and enforce thresholds
- `make lint-fix` — run linters with auto-fix

## Pull Request Process

1. Run `make test` and `make lint` before submitting your PR.
2. Add tests for any new code or functionality.
3. Keep PRs focused on a single concern — avoid mixing unrelated changes.
4. Ensure all CI checks pass.

## Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation changes
- `refactor:` — code refactoring
- `test:` — adding or updating tests
- `chore:` — maintenance tasks
- `ci:` — CI/CD changes

## Code Style

Code follows standard Go conventions (`gofmt`, `go vet`, etc.) plus project-specific rules defined in [.golangci.yml](.golangci.yml). Run `make lint` to verify.

## Branch Conventions

The default branch is `main`; all changes are merged into `main` via pull request.

- Work on a dedicated branch rather than committing directly to `main`.
- Feature branches describe the change being made (e.g. `feature/cache`).
- Milestone branches follow the pattern `gsd/v{VERSION}-{slug}` (e.g. `gsd/v1.6-cache-dedup`) and are opened as a PR to `main` when complete.
- Rebase or merge your branch on the latest `main` before opening the PR to keep the diff clean.

## Issue Reporting

Report bugs and request features through [GitHub Issues](https://github.com/guionardo/go/issues). This repository does not ship pre-built issue templates, so please include:

- A clear, descriptive title and a concise description of the problem or request.
- Steps to reproduce the issue, ideally with a minimal code sample.
- Expected behavior versus what actually happened.
- Your environment: Go version, operating system, and the affected package.

## Documentation and Further Reading

See GETTING-STARTED.md for prerequisites and first-run instructions, and DEVELOPMENT.md for local development setup. Additional references include [ARCHITECTURE.md](docs/ARCHITECTURE.md) and [CONFIGURATION.md](docs/CONFIGURATION.md).

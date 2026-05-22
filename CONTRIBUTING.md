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

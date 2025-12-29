# GitHub Copilot Instructions for guionardo/go

This repository contains Golang tools, examples, and packages. It's a collection of reusable Go utilities and libraries.

## Repository Structure

- `br_docs/` - Brazilian document validation (CPF, CNPJ)
- `flow/` - Logic flow utilities (ternary operator, default values)
- `fraction/` - Fraction type and operations
- `httptest_mock/` - Mocking helper for http requests
- `mid/` - Machine identification utilities
- `path_tools/` - File and directory path utilities
- `reflect_tools/` - Reflection and typing tools
- `set/` - Generic set implementation
- `shell_tools/` - Shell argument parsing utilities

## Development Workflow

### Prerequisites

- Go 1.25 or later
- Make sure `GOPATH/bin` is in your PATH

### Installing Dependencies

```bash
make deps
```

This will install:
- pre-commit hooks
- golangci-lint
- commitlint
- govulncheck
- go-test-coverage
- swag (swagger)

### Building

This is a library repository with no main build target. Individual packages can be imported and tested independently.

### Testing

Run all tests:
```bash
make test
```

Or directly with go:
```bash
go test ./... -v
```

For test coverage:
```bash
make coverage
```

### Linting

Run linters:
```bash
make lint
```

Auto-fix linting issues:
```bash
make lint-fix
```

The repository uses golangci-lint with extensive linter configuration in `.golangci.yml`.

## Code Style and Conventions

### General Guidelines

- Follow standard Go conventions and idioms
- Use meaningful variable and function names
- Keep functions focused and single-purpose
- Write tests for all new functionality
- Maintain high test coverage (see `.testcoverage.yml` for thresholds)

### Testing

- Place test files alongside the code they test (e.g., `file.go` and `file_test.go`)
- Use the `testify` package for assertions (`github.com/stretchr/testify`)
- Write both unit tests and edge case tests
- Test coverage is enforced via go-test-coverage

### Commits

- Follow Conventional Commits format (enforced by commitlint)
- Pre-commit hooks will run automatically for validation
- Configuration is in `.commitlint.yaml`

### Documentation

- Use Go doc comments for all exported functions, types, and packages
- Keep the main README.md at repository root updated with package usage examples
- Document complex logic with inline comments when necessary

### Package-Specific Notes

- **br_docs**: Handles Brazilian document validation with punctuation removal
- **flow**: Provides generic ternary operators and default value helpers
- **fraction**: Immutable fraction type, always simplified
- **mid**: Platform-specific machine ID detection (Linux, Windows, macOS)
- **path_tools**: OS-aware path operations
- **set**: Generic set with database/sql Scanner and Valuer interfaces
- **shell_tools**: Shell argument parsing with quote handling

## Adding New Packages

1. Create a new directory under `pkg/`
2. Follow the existing package structure
3. Include comprehensive tests
4. Add package documentation to the main README.md at repository root
5. Ensure linting passes
6. Verify test coverage meets thresholds

## Common Tasks

### Adding a new utility function
1. Identify the appropriate package or create a new one
2. Implement the function with proper documentation
3. Add comprehensive tests
4. Update the main README.md at repository root with usage example if it's a major feature
5. Run `make lint` and `make test`

### Fixing a bug
1. Add a test that reproduces the bug
2. Fix the implementation
3. Verify all tests pass
4. Ensure no linting issues

### Updating dependencies
1. Modify `go.mod` as needed
2. Run `go mod tidy`
3. Verify tests still pass
4. Check for any breaking changes

## Security

- Run `govulncheck` to check for known vulnerabilities
- No secrets should be committed to the repository
- Follow security best practices for all code

## Pre-commit Hooks

The repository uses pre-commit hooks for:
- Commit message validation (commitlint)
- Code quality checks
- Configuration is in `.pre-commit-config.yaml`

Install with: `make install-pre-commit`

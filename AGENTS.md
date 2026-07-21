# Project Instructions

## Repository

Go utility packages at `github.com/guionardo/go`. A collection of reusable Go libraries —
config, data structures, validators, cache, CLI self-update.

## Before Every Commit

Run the coverage check and verify it passes:

```bash
make coverage-quick
```

This enforces the thresholds in `.testcoverage-quick.yml`: packages ≥80%, files ≥70%, total ≥75%. Do not commit if it fails. Fix uncovered code or add tests first. (Note: cache providers tested via E2E with Docker — threshold overrides apply.)

## Code Style

- Follow standard Go conventions and idioms
- Keep functions focused and single-purpose
- Use `testify` for assertions in tests
- Write both unit tests and edge case tests
- Go doc comments for all exported symbols
- Follow Conventional Commits format
- Minimal external dependencies — prefer stdlib solutions

## Packages

| Package | Description |
|---------|-------------|
| `br_docs` | Brazilian document validation (CPF, CNPJ) |
| `cache` | Generic key-value cache with 5 backends (mem, Redis, Valkey, Memcache, Postgres) |
| `config` | Typed configuration provider (YAML + env + validation) |
| `flow` | Generic control flow (ternary, defaults) |
| `fraction` | Immutable fraction arithmetic |
| `httptest_mock` | HTTP mock server framework for tests |
| `mid` | Cross-platform machine ID |
| `path_tools` | File/directory path utilities |
| `reflect_tools` | Reflection utilities |
| `release` | Self-update mechanism via GitHub Releases |
| `set` | Generic set with algebra, JSON, SQL support |
| `shell_tools` | Shell argument parsing, env lookup |
| `time_tools` | Adaptive time format parsing |

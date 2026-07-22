# Project Instructions

## Repository

Go utility packages at `github.com/guionardo/go`. A collection of reusable Go libraries —
config, data structures, validators, cache, CLI self-update.

## Milestone Branches

Every new milestone must have its own git branch for the pull request flow.
Work on the milestone branch, then open a PR to `main` when complete.
Branch name: `gsd/v{VERSION}-{slug}` (e.g., `gsd/v1.6-retry-package`).

## Before Every Commit

- **Spike findings for go** (implementation patterns, constraints, gotchas) → `Skill("spike-findings-go")`

Run the coverage check and verify it passes:

```bash
make coverage-quick
```

This enforces the thresholds in `.testcoverage-quick.yml`: packages ≥80%, files ≥70%, total ≥75%. Do not commit if it fails. Fix uncovered code or add tests first. (Note: cache providers tested via E2E with Docker — threshold overrides apply.)

Then regenerate and stage the quality report before pushing a tag (not before every commit — too time consuming):

```bash
make quality-report
git add quality-report.md
```

```bash
make coverage-quick
```

This enforces the thresholds in `.testcoverage-quick.yml`: packages ≥80%, files ≥70%, total ≥75%. Do not commit if it fails. Fix uncovered code or add tests first. (Note: cache providers tested via E2E with Docker — threshold overrides apply.)

## Cross-Platform Testing

The CI runs tests on Linux, macOS, and Windows. Known platform differences:

- **Windows HTTP transport**: `http.DefaultClient` does not always canonicalize header keys before sending. A header set as `Api-Key` may arrive at the server as `Api_key`. Use normalization (lowercase + underscore→hyphen) instead of `req.Header.Get()` when matching headers.
- **Windows env vars**: `os.LookupEnv` is case-insensitive on Windows — `GetEnv("PATH")` and `GetEnv("path")` return the same value.
- **Windows file permissions**: `os.WriteFile` with 0755 mode produces 0666 on Windows. Skip permission checks when running on Windows.
- **Parallel test isolation**: Tests modifying global state (e.g., `collectFuncs` in `mid/`) must not use `t.Parallel()` to avoid races between sub-tests.

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

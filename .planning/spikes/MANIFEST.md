# Spike Manifest

## Idea

Replace the deprecated goreportcard.com with a local `make quality-report` target that aggregates golangci-lint results, govulncheck vulnerabilities, test coverage, lines of code, and dependency information into a single markdown report.

## Requirements

- Report must be in markdown format
- Must include lint results from golangci-lint
- Must include security vulnerability scan from govulncheck
- Must include test coverage per function
- Must include lines of code and file counts
- Must include dependency list
- Must be runnable via a single Makefile target

## Spikes

| # | Name | Type | Validates | Verdict | Tags |
|---|------|------|-----------|---------|------|
| 001 | golangci-lint-report | standard | Given a Go project, when `make quality-report` runs, then it produces a comprehensive markdown report | ✓ VALIDATED | golangci-lint, govulncheck, quality, makefile |

---
spike: 001
name: golangci-lint-report
type: standard
validates: "Given a Go project with golangci-lint and govulncheck installed, when 'make quality-report' runs, then it produces a comprehensive markdown report covering lint results, security vulnerabilities, test coverage, LOC, and dependencies"
verdict: VALIDATED
related: []
tags: [golangci-lint, govulncheck, quality, makefile, reporting]
---

# Spike 001: golangci-lint Quality Report

## What This Validates

That `golangci-lint` (v2.12.2) combined with `govulncheck`, `go test -cover`, and `go list` can produce a comprehensive local quality report in markdown format — replacing goreportcard.com.

## Research

goreportcard.com is deprecated. The official recommendation is to use `golangci-lint` for code quality analysis. `golangci-lint` v2 supports JSON and HTML output via `--output.json.path` / `--output.html.path`. For security scanning, `govulncheck` is the Go official tool.

### Approach Comparison

| Approach | Tool | Pros | Cons |
|----------|------|------|------|
| goreportcard.com | Web service | No setup | Deprecated, external dependency |
| golangci-lint HTML | Built-in output | Zero config | No coverage/security/metrics |
| Custom markdown script | quality-report.sh | All metrics in one file, extensible | Custom script to maintain |

**Chosen approach:** Custom markdown script that aggregates multiple tools.

## How to Run

```bash
make quality-report
```

Or directly:

```bash
bash .planning/spikes/001-golangci-lint-report/quality-report.sh quality-report.md .
```

## What to Expect

A `quality-report.md` file with these sections:
- **Lint Results:** 48 enabled linters, issue breakdown by linter
- **Security Vulnerabilities:** govulncheck output (or "No known vulnerabilities")
- **Test Coverage:** per-function coverage from `go tool cover`
- **Lines of Code:** breakdown by file extension + totals
- **Dependencies:** direct and transitive module list

## Investigation Trail

1. Initial script had JSON parsing issues with govulncheck's multi-object output format
2. Fixed to iterate line-by-line and find the `Vulnerabilities` object
3. golangci-lint v2 JSON output uses `{"Issues":[], "Report":{"Linters":[...]}}` structure
4. Makefile rule uses an order-only prerequisite to track the generated file

## Results

- **Verdict: VALIDATED ✓**
- All data sources produce correct output
- ~500-line comprehensive report generated in ~15 seconds
- Script is ~120 lines, easy to extend with additional metrics
- Govulncheck requires Go ≥1.22 for some vulnerability checks

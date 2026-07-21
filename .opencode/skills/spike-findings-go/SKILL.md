---
name: spike-findings-go
description: Implementation blueprint from spike experiments. Requirements, proven patterns, and verified knowledge for building go. Auto-loaded during implementation work.
---

<context>
## Project: go

Replace the deprecated goreportcard.com with a local `make quality-report` target that aggregates golangci-lint results, govulncheck vulnerabilities, test coverage, lines of code, and dependency information into a single markdown report.

Spike sessions wrapped: 2026-07-21
</context>

<requirements>
## Requirements

- Report must be in markdown format
- Must include lint results from golangci-lint
- Must include security vulnerability scan from govulncheck
- Must include test coverage per function
- Must include lines of code and file counts
- Must include dependency list
- Must be runnable via a single Makefile target
</requirements>

<findings_index>
## Feature Areas

| Area | Reference | Key Finding |
|------|-----------|-------------|
| Quality Reporting | references/quality-reporting.md | Custom markdown script aggregating lint + security + coverage + metrics works in ~15s |

## Source Files

Original spike source files are preserved in `sources/` for complete reference.
</findings_index>

<metadata>
## Processed Spikes

- 001-golangci-lint-report
</metadata>

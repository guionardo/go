# Spike Conventions

Patterns and stack choices established across spike sessions. New spikes follow these unless the question requires otherwise.

## Stack

- Go (CLI tooling, scripting via bash)
- Python for JSON processing and report assembly
- golangci-lint v2 for static analysis
- govulncheck for security vulnerability scanning
- `golang.org/x/sync/singleflight` v0.22.0 for dedup of concurrent GetOrSet misses

## Structure

- Spike scripts stored alongside their README in `.planning/spikes/NNN-name/`
- Makefile targets reference spike scripts by their spike path
- Reports use markdown format with consistent section headers
- Go concurrency spikes: standalone `package main` + `go run .`, verified with `go run -race .`

## Patterns

- JSON output from tools piped to Python for parsing and formatting
- Line-delimited JSON requires per-line iteration (not single `json.load`)
- All metrics aggregated into one markdown file, not separate files
- For concurrency spikes: wrap ONLY the setter in `singleflight.Group.Do`, not
  the whole Get/Set dance
- singleflight fn must double-check Get before computing (clobber guard)
- Panicking setter replays panics to all waiters — recover inside the wrapper
- Blocking `Do()` is context-blind; use `DoChan` + select if per-caller cancel matters

## Tools & Libraries

- golangci-lint v2.12.2 (JSON output via `--output.json.path`)
- govulncheck v1.3.0 (line-delimited JSON output)
- Go stdlib `go test -cover` for coverage

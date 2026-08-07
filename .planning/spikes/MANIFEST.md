# Spike Manifest

## Idea

Replace the deprecated goreportcard.com with a local `make quality-report` target that aggregates golangci-lint results, govulncheck vulnerabilities, test coverage, lines of code, and dependency information into a single markdown report.

## Spike Session 2026-08-06: singleflight in cache GetOrSet (ARCHIVED)

All findings integrated into production code. Spikes 002–009 cleaned up.

## Requirements (original quality-report)

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
| 002 | singleflight-dedup | standard | Given N concurrent misses on the same key, when setter wrapped in singleflight, then it runs exactly once and all callers share the value | ✓ ARCHIVED | singleflight, concurrency, cache, GetOrSet |
| 003 | singleflight-failure | standard | Given a failing setter or canceled context, when wrapped in singleflight, then the error is shared to all waiters and nothing is cached | ✓ ARCHIVED | singleflight, concurrency, cache, error |
| 004 | singleflight-ttl | standard | Given per-key TTL + expiring values, when singleflight group reused across calls, then TTL is respected and stale values are not served | ✓ ARCHIVED | singleflight, concurrency, cache, ttl |
| 005 | singleflight-placement | standard | Given 5 similar GetOrSet impls, then one shared cache-package helper serves all providers race-free | ✓ ARCHIVED | singleflight, cache, design |
| 006 | real-provider-integration | standard | Given the shared helper, when all 5 providers delegate GetOrSet, then error semantics (prefixes + valkey initErr guard) are preserved | ✓ ARCHIVED | singleflight, cache, integration, error |
| 007 | delete-vs-inflight | standard | Given Delete during an in-flight setter, when the leader finishes, then the key is not resurrected (and Forget is sufficient) | ⚠ ARCHIVED | singleflight, cache, delete, forget |
| 008 | dochan-context-aware | standard | Given a slow setter under cancel-heavy load, when the wrapper uses DoChan + select, then canceled waiters release promptly | ✓ ARCHIVED | singleflight, context, DoChan, goroutines |
| 009 | thundering-herd-benchmark | standard | Given N concurrent misses, when singleflight wraps the setter vs naive, then setter runs 1 vs N times with measurable win | ✓ ARCHIVED | singleflight, cache, benchmark |

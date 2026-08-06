# Spike Manifest

## Idea

Replace the deprecated goreportcard.com with a local `make quality-report` target that aggregates golangci-lint results, govulncheck vulnerabilities, test coverage, lines of code, and dependency information into a single markdown report.

## Spike Session 2026-08-06: singleflight in cache GetOrSet

Explore wrapping the setter inside `GetOrSet` with `golang.org/x/sync/singleflight`
to avoid simultaneous rework when many callers miss the same key concurrently.
Applied to all 5 provider GetOrSet implementations (mem, redis, valkey, memcache, postgres).

Requirements emerging:

- `singleflight.Group` must wrap only the setter, not the whole Get/Set dance
- The `shared` return is a global signal (leader sees true too) — never use it as a "I'm the follower" check
- Error from a failing setter is shared with all waiters (verify in 003)
- Placement must be shared code, not duplicated per provider (see 005)

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
| 002 | singleflight-dedup | standard | Given N concurrent misses on the same key, when setter wrapped in singleflight, then it runs exactly once and all callers share the value | ✓ VALIDATED | singleflight, concurrency, cache, GetOrSet |
| 003 | singleflight-failure | standard | Given a failing setter or canceled context, when wrapped in singleflight, then the error is shared to all waiters and nothing is cached | PENDING | singleflight, concurrency, cache, error |
| 004 | singleflight-ttl | standard | Given per-key TTL + expiring values, when singleflight group reused across calls, then TTL is respected and stale values are not served | PENDING | singleflight, concurrency, cache, ttl |
| 005 | singleflight-placement | design | Given 5 similar GetOrSet impls, when singleflight is introduced, then it lives in one shared place with no duplication | PENDING | singleflight, cache, design |

---
name: spike-findings-go
description: Implementation blueprint from spike experiments. Requirements, proven patterns, and verified knowledge for building go. Auto-loaded during implementation work.
---

<context>
## Project: go

Replace the deprecated goreportcard.com with a local `make quality-report` target that aggregates golangci-lint results, govulncheck vulnerabilities, test coverage, lines of code, and dependency information into a single markdown report.

A second line of work explores wrapping the setter inside the `cache` package's
`GetOrSet` with `golang.org/x/sync/singleflight` to avoid simultaneous rework when
many callers miss the same key concurrently, applied across all 5 provider
implementations (mem, redis, valkey, memcache, postgres).

Spike sessions wrapped: 2026-07-21 (quality reporting), 2026-08-06 (singleflight)
</context>

<requirements>
## Requirements

**Quality reporting:**
- Report must be in markdown format
- Must include lint results from golangci-lint
- Must include security vulnerability scan from govulncheck
- Must include test coverage per function
- Must include lines of code and file counts
- Must include dependency list
- Must be runnable via a single Makefile target

**Singleflight in cache GetOrSet:**
- `singleflight.Group` must wrap only the setter, not the whole Get/Set dance
- The `shared` return is a global signal (leader sees true too) — never use it as an "I'm the follower" check
- Error from a failing setter is shared with all waiters
- Placement must be shared code, not duplicated per provider
- Do (blocking) is context-blind; use DoChan + select only if per-caller cancel is required
- Panicking setters fan out panics to all members — wrapper must recover and return error
- singleflight does not cache completed results — TTL stays fully owned by the provider
- The singleflight fn must double-check Get before computing to avoid clobbering a concurrent direct Set
- Prefer DoChan + select for HTTP-facing GetOrSet so canceled waiters release immediately (validated in spike 008, ~15 ms vs ~300 ms blocking)
- redis/valkey keep their historical error prefixes by wrapping the shared helper's error in their thin GetOrSet; mem/memcache/postgres return it raw; valkey's initErr guard stays provider-side
- group.Forget does NOT stop an in-flight leader — guard Delete-during-flight with a deletion-generation tombstone re-checked inside the Do fn before Set
- Benchmark dedup impact with `go test -bench=. -benchmem -benchtime=2x` (validated in spike 009)
</requirements>

<findings_index>
## Feature Areas

| Area | Reference | Key Finding |
|------|-----------|-------------|
| Quality Reporting | references/quality-reporting.md | Custom markdown script aggregating lint + security + coverage + metrics works in ~15s |
| Singleflight in Cache GetOrSet | references/singleflight-cache.md | One shared cache-package helper dedups concurrent misses (setter runs once, race-free, TTL-safe); DoChan variant for cancel, per-provider error glue, and generation-tombstone delete guard |

## Source Files

Original spike source files are preserved in `sources/` for complete reference.
</findings_index>

<metadata>
## Processed Spikes

- 001-golangci-lint-report
- 002-singleflight-dedup
- 003-singleflight-failure
- 004-singleflight-ttl
- 005-singleflight-placement
- 006-singleflight-real-providers
- 007-singleflight-delete
- 008-singleflight-context
- 009-singleflight-benchmark
</metadata>
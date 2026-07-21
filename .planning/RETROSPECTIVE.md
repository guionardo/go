# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.4 — Core Packages

**Shipped:** 2026-07-21
**Phases:** 1 | **Plans:** 3 | **Commits:** 24

### What Was Built
- Generic `Cache[K, V]` interface with 5 backends (in-memory, Redis, Valkey, Memcache, Postgres)
- In-memory provider with TTL sweep and concurrent-safe access
- Redis + Valkey providers with JSON serialization and sub-second TTL support
- Memcache provider with goroutine-per-call context wrapping
- Postgres provider with UNLOGGED table, pg_prewarm, background TTL sweep
- 50 E2E tests using testcontainers-go across all 5 providers
- Build-tag separation (e2e) for Docker-dependent tests

### What Worked
- Generic `Cache[K, V]` interface made testing easy — swap providers with one line
- Functional options pattern consistent across all providers
- Design-first (interface) → provider implementation order was effective
- Build tags kept `go test ./...` working without Docker
- UAT caught real issues (Valkey readiness check race, missing build tags)

### What Was Inefficient
- `ExampleNew` tests required real servers — needed build tag retrofitting
- Valkey container readiness check was unreliable — required fix during UAT
- No STATE.md meant gsd-tools couldn't track progress automatically

### Patterns Established
- Each provider in own sub-package, importable independently
- E2E tests in testcontainers with `//go:build e2e` tag
- VERIFICATION.md + UAT.md as dual verification gates
- Per-provider `Options` type with functional options

### Key Lessons
1. Example tests (`ExampleXxx`) should check env vars or use build tags — they can't skip like regular tests
2. Container readiness checks need `wait.ForListeningPort` alongside log matching for reliability
3. Build tags are essential for separating unit/E2E tests in a package

### Cost Observations
- Model mix: 100% adaptive (no explicit model selection)
- Sessions: 1 session (4 hours)
- Notable: Cache package from zero to shipped in a single session

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Commits | Phases | Key Change |
|-----------|---------|--------|------------|
| v1.0 | 24 | 1 | Initial GSD workflow setup with plan→execute→verify→UAT cycle |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.0 | 50 E2E + unit tests | 95%+ target | 4 (gomemcache, pgx, go-redis, valkey-go) |

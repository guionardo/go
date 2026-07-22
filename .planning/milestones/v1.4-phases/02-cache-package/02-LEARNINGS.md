---
phase: 02
phase_name: Cache Package
project: "go - Golang tools, examples, and packages"
generated: "2026-07-21"
counts:
  decisions: 6
  lessons: 4
  patterns: 5
  surprises: 2
missing_artifacts: []
---

# Phase 2 Learnings: Cache Package

## Decisions

### Generic Cache Interface Design
Cache[K,V] interface with context.Context on every method and optional per-key TTL.

**Rationale:** Go 1.18+ generics enable type-safe cache operations without casting. Context on every method supports cancellation and tracing. Optional TTL variadic on Set/GetOrSet gives callers per-key control with provider-level default fallback.

**Source:** 02-01-SUMMARY.md

### Exported Option Interface for Cross-Package Functional Options
Changed from unexported `apply(*config)` to exported `Apply(*Config)` with exported `Config` struct.

**Rationale:** The plan's original unexported types prevented the `mem` package from calling `opt.apply()` because Go does not allow calling unexported interface methods from outside the defining package. Exporting the struct and method is necessary for the functional options pattern to work across packages.

**Source:** 02-01-SUMMARY.md

### Valkey Provider Uses initErr Pattern Instead of Panicking
Constructor stores connection errors in `initErr` field instead of panicking on `NewClient` failure.

**Rationale:** valkey-go eagerly dials at construction time (unlike go-redis which uses lazy connection). Panicking when no server is running would break tests that must work without a Valkey server. The `initErr` pattern provides clean error surfacing without panics.

**Source:** 02-02-SUMMARY.md

### Memcache Goroutine-Per-Call Context Wrapping
Uses goroutine with select on ctx.Done() to cancel blocking gomemcache calls.

**Rationale:** gomemcache does not natively support context.Context. Wrapping each call in a goroutine with context cancellation is the idiomatic Go approach to add timeout/cancellation support to libraries that don't support it.

**Source:** 02-03-SUMMARY.md

### Postgres Constructor Returns Error
`New[K,V]` returns `(*Cache, error)` instead of just `*Cache`.

**Rationale:** pgxpool.New validates the connection string and can fail (invalid credentials, unreachable host). Returning an error is the correct Go idiom for fallible constructors.

**Source:** 02-03-SUMMARY.md

### Close Idempotency via Channel Guard
Uses `select { case <-c.stop: default: close(c.stop) }` instead of `sync.Once`.

**Rationale:** This pattern prevents panic on double-close of a stop channel while being simpler and more idiomatic than `sync.Once` for channel operations.

**Source:** 02-01-SUMMARY.md

## Lessons

### Cross-Package Option Interface Requires Exported Types
Unexported interface methods cannot be called from outside the defining package. The functional options pattern must use exported types when the option is applied in a different package.

**Context:** The cache/mem package could not call `opt.apply()` because both the method and config struct were unexported. This caused a compile error during task 3.

**Source:** 02-01-SUMMARY.md

### E2E Tests Need Build Tags
Docker-dependent tests require `//go:build e2e` to prevent `go test ./...` from failing in CI environments without Docker.

**Context:** Cache provider integration tests use testcontainers-go for Redis, Valkey, Memcache, and Postgres. Without build tags, `go test ./...` fails in CI.

**Source:** 02-UAT.md

### Container Readiness Checks Need Multiple Strategies
Valkey container readiness check required both log matching and `wait.ForListeningPort` for reliability.

**Context:** Valkey was marked ready by testcontainers before the Valkey server actually accepted connections. The `wait.ForListeningPort` addition fixed the race.

**Source:** 02-UAT.md

### Example Tests Must Handle Backend Unavailability
Example tests for Redis, Valkey, Memcache, and Postgres silently fail when backends are not running.

**Context:** `_ = c.Set(...)` swallowed errors, producing empty output that failed `// Output` assertions without explanation. Fixed by explicitly checking errors.

**Source:** 02-03-SUMMARY.md

## Patterns

### Provider Per Sub-Package
Each cache backend is in its own sub-package (`cache/mem/`, `cache/redis/`, etc.) and is independently importable.

**When to use:** When building a family of related implementations behind a shared interface. Each sub-package can be tested and versioned independently.

**Source:** 02-VERIFICATION.md

### Background Sweep Goroutine
A goroutine with a ticker sweeps expired entries on a configurable interval, stopped via a channel.

**When to use:** For any cached data that needs periodic cleanup. The ticker + channel pattern provides clean shutdown via Close(). Used by both in-memory and Postgres providers.

**Source:** 02-01-SUMMARY.md

### Functional Options with Exported Config
Each provider defines its own `Option` type and `Config` struct, following the same pattern.

**When to use:** When a constructor has many optional parameters. The functional options pattern gives clear names at call sites and is extensible without breaking backward compatibility.

**Source:** 02-02-SUMMARY.md

### JSON Serialization for Cache Values
All providers serialize values via `encoding/json` for storage and deserialize on retrieval.

**When to use:** When the cache needs to store arbitrary Go types. JSON is universally available in Go and works across all backends.

**Source:** 02-VERIFICATION.md

### Error Wrapping with Provider Prefix
Every error is wrapped with `"cache/provider-name: %w"` format.

**When to use:** When callers need to distinguish which backend produced an error. Using `errors.Is` on the base error still works through the wrapping.

**Source:** 02-01-SUMMARY.md

## Surprises

### Valkey Files Were Already Committed
The Valkey provider files existed in git before plan 02-02 executed them.

**Impact:** Zero — writes were idempotent. But it revealed a prior committed omission.

**Source:** 02-02-SUMMARY.md

### Cross-Platform Option Interface Visibility
Go's visibility rules for interface methods surprised the plan — unexported interface methods cannot be called from outside the defining package, even through the interface.

**Impact:** Required a fix commit to export the method and config struct. Added ~5 minutes to execution.

**Source:** 02-01-SUMMARY.md

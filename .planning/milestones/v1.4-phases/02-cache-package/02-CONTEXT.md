# Phase 2: Cache Package - Context

**Gathered:** 2026-07-21
**Status:** Ready for planning

<domain>
## Phase Boundary

A generic `Cache[K, V]` abstraction over multiple backends (in-memory, Redis, Memcache, Postgres, Valkey) that lets consumers swap providers without code changes. Each provider in its own sub-package — independently importable. Includes a Close method for cleanup and configurable default TTL.

</domain>

<decisions>
## Implementation Decisions

### Interface Design
- **D-01:** Generic `Cache[K, V]` interface with `Get`, `Set`, `Delete`, `GetOrSet` methods
- **D-02:** Every method accepts `context.Context` for cancellation and tracing
- **D-03:** `Set` accepts optional per-key TTL with provider-level default fallback
- **D-04:** `Close() error` included in the interface for provider cleanup (connection pools, goroutines)

### Concurrency Safety
- **D-05:** In-memory provider uses `sync.RWMutex` — `RLock` for reads, `Lock` for writes. Consistent with `config/provider.go` pattern.

### Eviction Strategy
- **D-06:** Periodic background goroutine sweeps expired entries (configurable interval)
- **D-07:** Passive TTL check on `Get` as correctness backstop

### Serialization
- **D-08:** `encoding/json` for serializing generic values in external providers (Redis, Memcache, Postgres). Callers implement `json.Marshaler`/`json.Unmarshaler` for custom types if needed.

### Connection Configuration
- **D-09:** Functional options pattern for all providers — `NewRedisCache(WithAddr(...), WithPoolSize(...))`. Consistent with `config/options.go`.

### Postgres Provider
- **D-10:** Simple TTL table with `cache_key`, `value`, `expires_at` columns. Background sweep deletes expired rows.

### Default TTL
- **D-11:** Configurable at construction via functional option — `NewCache(WithDefaultTTL(5*time.Minute))`. No hardcoded default.

### Error Handling
- **D-12:** All provider errors are wrapped with `fmt.Errorf("provider: %w", err)` and returned — never swallowed.

### Provider Architecture
- **D-13:** Each provider in its own sub-package: `cache/mem`, `cache/redis`, `cache/memcache`, `cache/postgres`, `cache/valkey`
- **D-14:** Base package exposes only the `Cache[K, V]` interface — no required provider dependency

### the agent's Discretion
- Metrics/observability — add if needed, not part of v1 interface
- Key namespacing — leave to caller
- Provider-specific encoding — `encoding/json` for all external providers

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase requirements
- `.planning/REQUIREMENTS.md` § Cache Package — CACHE-01 through CACHE-11

### Codebase conventions
- `.planning/codebase/CONVENTIONS.md` — naming, error handling, functional options pattern
- `.planning/codebase/STRUCTURE.md` — where new packages go, file naming
- `.planning/codebase/TESTING.md` — 95% coverage, testify, t.Parallel(), example_test.go

### Existing patterns
- `config/options.go` — functional options pattern reference
- `config/provider.go` — sync.RWMutex pattern reference
- `set/set.go` — generic type conventions

### Exploration artifacts
- `.planning/notes/cache-design-decisions.md` — initial design decisions from exploration

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Functional options pattern:** `config/options.go` — provider configuration uses the same `type option func(*provider)` pattern. New providers should follow this.
- **Error handling:** `fmt.Errorf("context: %w", err)` with sentinel errors pattern from `fraction/` and `set/`.

### Established Patterns
- **Generic types:** `Set[T comparable]` pattern from `set/set.go` — use `[K comparable, V any]` for the cache interface
- **Package structure:** Top-level directory per feature (e.g., `cache/`), sub-packages per provider (e.g., `cache/mem/`)
- **Testing:** `t.Parallel()` in all subtests, testify assertions, table-driven tests, `example_test.go` per package

### Integration Points
- **Go module:** Package lives at `github.com/guionardo/go/cache/...` alongside other utility packages
- **CI:** `.testcoverage.yml` enforces 95% total coverage — cache package must maintain this

</code_context>

<specifics>
## Specific Ideas

- Memory cache serves as both a standalone provider and the zero-dependency testing replacement — `cache.NewInMemory()` for tests, swap to `cache.NewRedis(...)` in production
- Behavior should align with Go stdlib conventions where applicable (nil/zero values handled gracefully)

</specifics>

<deferred>
## Deferred Ideas

### Batch Operations
- MGet, MSet, MDel — deferred to future phase. Tracked in `.planning/seeds/batch-operations.md`.
- Trigger: when at least 2 downstream projects need batch cache ops.

</deferred>

---

*Phase: 2-Cache Package*
*Context gathered: 2026-07-21*

# Phase 7: Batch operations - Context

**Gathered:** 2026-08-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Add `MGet`, `MSet`, and `MDel` batch operations to all 5 cache providers (mem,
redis, valkey, memcache, postgres) with provider-appropriate batching and no
errors on empty or partial key sets.
</domain>

<decisions>
## Implementation Decisions

### Interface Architecture
- **D-01:** Batch operations live on the unexported `cacher[K,V]` interface
  in `cache/concrete_cache.go`. `concreteCache` exposes them on the
  `Cache[K,V]` interface (or as direct methods on the concrete type).
  Providers implement via `cacher`; `concreteCache` provides default
  per-key-loop fallbacks.
- **D-02:** This keeps backward compatibility — the `Cache[K,V]` interface
  does not grow new methods; external implementors are unaffected.

### MGet Return Type
- **D-03:** `MGet(ctx, keys ...K) map[K]V` — returns only found keys.
  Missing keys are simply absent from the map. No per-key error
  propagation. Matches BATCH-01.

### MSet TTL Policy
- **D-04:** `MSet(ctx, items map[K]V, ttl ...time.Duration)` — single
  optional TTL applies to all keys in the batch. Follows the existing
  `Set` variadic TTL pattern. Matches BATCH-02.

### MDel Signature
- **D-05:** `MDel(ctx, keys ...K)` — standard multi-key delete.
  Deleting already-missing keys is idempotent (no error). Matches BATCH-02.

### Error Handling
- **D-06:** Accumulate errors — continue processing remaining keys when
  one fails. Return an aggregate error (`errors.Join`) listing all
  failures at the end. Best-effort: process as many keys as possible.

### Provider-Specific Strategies
- **D-07:** mem — iterate the underlying `flow.OrderedMap` store under a
  single lock acquisition (BATCH-03).
- **D-08:** memcache — use native `GetMulti` for MGet (BATCH-04).
- **D-09:** redis — use `go-redis` pipeline for MGet/MSet/MDel (BATCH-05).
- **D-10:** valkey — use valkey-go pipeline for MGet/MSet/MDel (BATCH-05).
- **D-11:** postgres — batch operations in a single query batch /
  transaction (BATCH-06).

### Edge Cases
- **D-12:** Empty key sets — all batch ops complete without error.
- **D-13:** Partial key sets — process available keys, skip missing
  (MGet returns subset; MDel no-ops on already-deleted).
- **D-14:** All batch ops must pass the race detector (BATCH-07).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Cache architecture
- `cache/cache.go` — `Cache[K,V]` interface (public contract)
- `cache/concrete_cache.go` — `cacher` unexported interface and
  `concreteCache` implementation (batch ops will live here)
- `cache/singleflight.go` — `SingleflightGetOrSet` (existing shared helper)
- `cache/mem/store.go` — `OrderedMap`-backed store (batch iteration pattern)

### Provider implementations (reference for per-provider batching)
- `cache/mem/mem.go` — `memoryCache` (implements `cacher`)
- `cache/redis/redis.go` — `redisCache` (implements `cacher`)
- `cache/valkey/valkey.go` — `valkeyCache` (implements `cacher`)
- `cache/memcache/memcache.go` — `memcacheCache` (implements `cacher`)
- `cache/postgres/postgres.go` — `postgresCache` (implements `cacher`)

### Requirements
- `.planning/REQUIREMENTS.md` — BATCH-01 through BATCH-07

### Testing
- `cache/cache_e2e_test.go` — E2E test pattern (testcontainers-go)
- `.testcoverage-quick.yml` — coverage thresholds and provider overrides

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `flow.OrderedMap` — mem store already uses this; `Keys()` and `Range()`
  methods support efficient batch iteration.
- `SingleflightGetOrSet[K,V]` — cacher pattern already established;
  batch ops follow the same provider-interface pattern.

### Established Patterns
- Each provider implements the unexported `cacher` interface with
  `GetFunc`/`SetFunc`/`DeleteFunc`/`CloseFunc`. Batch ops add
  `MGetFunc`/`MSetFunc`/`MDelFunc` to the same interface.
- `concreteCache` delegates to `cacher`; default fallback implementations
  iterate per-key if a provider doesn't override (mem, redis, valkey,
  memcache, postgres all expected to override).

### Integration Points
- `cache/concrete_cache.go` — add batch methods + `cacher` interface methods
- Each provider file — implement provider-specific batch methods
- `cache/cache.go` — no changes (backward-compatible); `Cache[K,V]` stays
  unchanged unless decided otherwise
- `cache/doc.go` — update to document batch ops

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the requirements doc — open to standard
batch-operation approaches per provider.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 7-Batch operations*
*Context gathered: 2026-08-07*

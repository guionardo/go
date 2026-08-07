# Phase 6 — Provider Integration (retroactive)

## Summary

All 5 cache providers (mem, redis, valkey, memcache, postgres) were migrated from
per-provider hand-rolled `GetOrSet` implementations to a shared
`concreteCache[K,V]` / `cacher[K,V]` architecture in `cache/concrete_cache.go`.

**What changed per provider:**
- Exported `Cache[K,V]` struct → unexported provider type implementing the `cacher`
  interface (`GetFunc`/`SetFunc`/`DeleteFunc`/`CloseFunc`)
- `New()` returns `cache.Cache[K,V]` via `cache.NewConcreteCache(...)`, providing
  shared singleflight dedup for `GetOrSet` (setter runs once on concurrent misses)
- Per-provider `GetOrSet` methods removed (delegated to shared `SingleflightGetOrSet.Do`)
- compile-time `cache.Cache` interface assertions removed (no longer directly satisfied)

**Preserved:**
- Historical error prefixes per provider (`cache/redis:`, `cache/valkey:`, etc.)
- valkey's `initErr` guard (pre-connection validation)
- TTL resolution logic unchanged per provider
- Lazy/eager connection semantics per provider
- All tests pass (unit + E2E against real containers: 493 total, 60 E2E)

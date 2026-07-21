# Phase 2: Cache Package - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-21
**Phase:** 2-Cache Package
**Areas discussed:** Concurrency safety, Eviction strategy, Serialization, Connection config, Postgres approach, Close/Shutdown, Default TTL value

---

## Concurrency Safety

| Option | Description | Selected |
|--------|-------------|----------|
| sync.RWMutex | Standard Go pattern — RLock for reads, Lock for writes. Consistent with config/provider.go in this codebase. | ✓ |
| sync.Map | Go stdlib concurrent map — simpler but less control over eviction. No locking needed by caller. | |
| Channel-based | Single goroutine owns the map, requests go through channels. More complex but avoids mutex contention. | |

**User's choice:** sync.RWMutex
**Notes:** Consistent with existing codebase pattern

---

## Eviction Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Passive cleanup on Get | Check TTL on each Get — expired items removed on access. Simple, no background goroutines. | |
| Periodic background sweep | Background goroutine with ticker sweeps expired entries every N seconds. Combined with passive cleanup for correctness. | ✓ |
| LRU with max size | Evict least-recently-used items when cache exceeds configured max entries. Adds complexity but bounds memory. | |

**User's choice:** Periodic background sweep
**Notes:** Combined with passive cleanup on Get for correctness

---

## Serialization

| Option | Description | Selected |
|--------|-------------|----------|
| encoding/json | Stdlib JSON — simple, debuggable, works everywhere. | ✓ |
| encoding/gob | Go-native binary format — more compact but only works in Go-to-Go communication. | |
| Custom Marshal/Unmarshal interface | Require callers to provide serialize/deserialize functions. Most flexible but adds API surface. | |

**User's choice:** encoding/json
**Notes:** Simplicity wins — callers can implement json interfaces for custom types

---

## Connection Config

| Option | Description | Selected |
|--------|-------------|----------|
| Functional options | NewRedisCache(WithAddr(...), WithPoolSize(...)). Standard in this codebase, type-safe. | ✓ |
| Connection string | Format: redis://user:pass@host:6379/0?pool=10. Parsed internally. Portable. | |
| Simple address param | NewRedisCache("localhost:6379"). Simpler but less flexible. | |

**User's choice:** Functional options
**Notes:** Consistent with config/options.go pattern

---

## Postgres Approach

| Option | Description | Selected |
|--------|-------------|----------|
| TTL table with sweep | Simple polling table (cache_key, value, expires_at). Background sweep deletes expired rows. | ✓ |
| LISTEN/NOTIFY | Use Postgres LISTEN/NOTIFY for invalidation events. More complex. | |
| Hybrid with advisory locks | Two-table + advisory locks. Most complex. | |

**User's choice:** TTL table with sweep
**Notes:** Keep it simple — reliable and portable

---

## Close/Shutdown

| Option | Description | Selected |
|--------|-------------|----------|
| Include Close in interface | Close() error on the interface. Required for Redis/Memcache/Postgres to release connections. | ✓ |
| Keep interface clean | Require callers to handle provider lifecycle via type assertion to io.Closer. | |
| Context-based shutdown | Provider goroutines stop when passed context is cancelled. No explicit Close needed. | |

**User's choice:** Include Close in interface
**Notes:** Clean shutdown semantics expected by consumers

---

## Default TTL Value

| Option | Description | Selected |
|--------|-------------|----------|
| 5 minutes | Reasonable general-purpose default for most cache use cases. | |
| Configurable at construction | User provides default when creating the cache — NewCache(WithDefaultTTL(5*time.Minute)). | ✓ |
| 30 minutes | Better for long-lived config and reference data. | |
| No default — explicit only | Set requires explicit TTL always. | |

**User's choice:** Configurable at construction
**Notes:** Most flexible — defaults provided at cache creation time

---

## the agent's Discretion

- Metrics/observability — v1 scope, add if needed
- Key namespacing — leave to caller
- Provider-specific encoding — json for all

## Deferred Ideas

- Batch operations (MGet, MSet, MDel) — tracked in `.planning/seeds/batch-operations.md`

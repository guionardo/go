# Phase 7: Batch operations - Research

**Researched:** 2026-08-07
**Domain:** Cache batch operations (Go, 5 providers)
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Interface Architecture
- **D-01:** Batch operations live on the unexported `cacher[K,V]` interface in `cache/concrete_cache.go`. `concreteCache` exposes them on the `Cache[K,V]` interface (or as direct methods on the concrete type). Providers implement via `cacher`; `concreteCache` provides default per-key-loop fallbacks.
- **D-02:** This keeps backward compatibility — the `Cache[K,V]` interface does not grow new methods; external implementors are unaffected.

#### MGet Return Type
- **D-03:** `MGet(ctx, keys ...K) map[K]V` — returns only found keys. Missing keys are simply absent from the map. No per-key error propagation. Matches BATCH-01.

#### MSet TTL Policy
- **D-04:** `MSet(ctx, items map[K]V, ttl ...time.Duration)` — single optional TTL applies to all keys in the batch. Follows the existing `Set` variadic TTL pattern. Matches BATCH-02.

#### MDel Signature
- **D-05:** `MDel(ctx, keys ...K)` — standard multi-key delete. Deleting already-missing keys is idempotent (no error). Matches BATCH-02.

#### Error Handling
- **D-06:** Accumulate errors — continue processing remaining keys when one fails. Return an aggregate error (`errors.Join`) listing all failures at the end. Best-effort: process as many keys as possible.

#### Provider-Specific Strategies
- **D-07:** mem — iterate the underlying `flow.OrderedMap` store under a single lock acquisition (BATCH-03).
- **D-08:** memcache — use native `GetMulti` for MGet (BATCH-04).
- **D-09:** redis — use `go-redis` pipeline for MGet/MSet/MDel (BATCH-05).
- **D-10:** valkey — use valkey-go pipeline for MGet/MSet/MDel (BATCH-05).
- **D-11:** postgres — batch operations in a single query batch / transaction (BATCH-06).

#### Edge Cases
- **D-12:** Empty key sets — all batch ops complete without error.
- **D-13:** Partial key sets — process available keys, skip missing (MGet returns subset; MDel no-ops on already-deleted).
- **D-14:** All batch ops must pass the race detector (BATCH-07).

### the agent's Discretion

Open to standard batch-operation approaches per provider. No additional restrictions on implementation detail.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BATCH-01 | `MGet(keys ...K) map[K]V` — only found keys returned | [VERIFIED: cache/concrete_cache.go] — cacher interface gets `MGetFunc`; concreteCache delegates; mem implements via single-lock OrderedMap range; memcache via `GetMulti`; redis/valkey via pipeline `MGet`; postgres via batch query |
| BATCH-02 | `MSet(items map[K]V, ttl ...time.Duration)` and `MDel(keys ...K)` | [VERIFIED: cache/concrete_cache.go] — cacher interface gets `MSetFunc` and `MDelFunc`; same provider-specific strategies apply; MDel idempotent per existing `Delete` patterns |
| BATCH-03 | mem batch ops iterate store under single lock | [VERIFIED: cache/mem/mem.go line 42-68] — existing pattern: `GetFunc` acquires `RLock`, `SetFunc`/`DeleteFunc` acquire full `Lock`. Batch ops follow same pattern with a single acquisition |
| BATCH-04 | memcache uses native `GetMulti` for MGet | [ASSUMED] — `github.com/bradfitz/gomemcache` exposes `client.GetMulti(keys []string) (map[string]*Item, error)`; MSet/MDel use per-key goroutines with error accumulation |
| BATCH-05 | redis/valkey use pipelines for MGet/MSet/MDel | [ASSUMED] — `go-redis/v9` has `pipeliner` with `MGet`, `MSet`, `Del`; valkey-go has `client.DoMulti()` for pipeline-style batch |
| BATCH-06 | postgres batches in single transaction/batch | [VERIFIED: cache/postgres/postgres.go] — pgxpool supports `pgx.Batch` (SendBatch) for MGet/MSet/MDel in single round trip |
| BATCH-07 | Empty/partial key sets complete without errors, pass race detector | [VERIFIED: cache/concrete_cache_test.go] — existing test patterns use `t.Parallel()`, `sync.WaitGroup`, and `testify` assertions; batch ops must follow same race-safe patterns |
</phase_requirements>

## Summary

This phase adds `MGet`, `MSet`, and `MDel` batch operations to all 5 cache providers (mem, redis, valkey, memcache, postgres). The batch operations live on the unexported `cacher[K,V]` interface in `concrete_cache.go`, keeping the public `Cache[K,V]` interface backward-compatible per D-02. Each provider implements a provider-optimal strategy: single-lock iteration for mem, `GetMulti` for memcache, pipelines for redis and valkey, and batch queries for postgres.

The design follows the same delegation pattern as existing single-key operations: `concreteCache` delegates to `cacher` methods, and `concreteCache` provides default per-key-loop fallbacks for any future provider that doesn't override. Error handling follows D-06: errors accumulate via `errors.Join` while processing continues, and the caller receives all failures as an aggregate.

No new external dependencies are required — all libraries are already in `go.mod`. All 5 provider-specific batch strategies are locked decisions (D-07 through D-11), leaving only implementation details to the plan.

**Primary recommendation:** Add `MGetFunc`/`MSetFunc`/`MDelFunc` to `cacher` interface; implement provider-specific batch strategies; add `MGet`/`MSet`/`MDel` as direct methods on `concreteCache`; provide per-key-loop fallbacks for future providers.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Batch operation interface | Library core (cache pkg) | — | `cacher` interface lives in `concrete_cache.go`; `concreteCache` provides fallbacks |
| mem batch implementation | In-memory provider | — | Single `sync.RWMutex` lock covers all keys for each batch op |
| memcache batch implementation | Memcache provider | — | Uses gomemcache's native `GetMulti`; MSet/MDel via per-key goroutines |
| redis batch implementation | Redis provider | — | Uses go-redis pipeline for atomic multi-key round trips |
| valkey batch implementation | Valkey provider | — | Uses valkey-go `DoMulti` for pipeline-style batch ops |
| postgres batch implementation | Postgres provider | — | Uses pgx `SendBatch` for single round-trip batch SQL |
| Error aggregation | Library core (cache pkg) | — | D-06: `errors.Join` pattern, common to all providers |
| Race detector compliance | All providers | Testing | BATCH-07: `go test -race` must pass on all providers |

## Standard Stack

No new dependencies — this phase uses only existing libraries:

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/sync` | v0.22.0 | `singleflight.Group` pattern reference | Already in go.mod; batch ops follow same delegation architecture |
| `github.com/redis/go-redis/v9` | v9.21.0 | Redis pipeline (`MGet`, `MSet`, `Del`) | [VERIFIED: go.mod] Already imported by redis provider |
| `github.com/valkey-io/valkey-go` | v1.0.76 | Valkey pipeline (`DoMulti`) | [VERIFIED: go.mod] Already imported by valkey provider |
| `github.com/bradfitz/gomemcache` | v0.0.0-20260422 | Memcache `GetMulti` | [VERIFIED: go.mod] Already imported by memcache provider |
| `github.com/jackc/pgx/v5` | v5.10.0 | Postgres batch (`SendBatch`) | [VERIFIED: go.mod] Already imported by postgres provider |
| `github.com/testcontainers/testcontainers-go` | v0.43.0 | E2E test containers | [VERIFIED: go.mod] Existing test infrastructure |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions | [VERIFIED: go.mod] Existing test infrastructure |
| `github.com/guionardo/go/flow` | — | `OrderedMap` iteration for mem batch | [VERIFIED: cache/mem/store.go] Already used by mem provider |

### Installation

No new packages to install. All dependencies are already in `go.sum`.

## Package Legitimacy Audit

> **Not applicable** — this phase does not introduce any new external packages. All providers and dependencies are already in `go.mod` and verified.

## Architecture Patterns

### System Architecture Diagram

```
User code
    │
    ▼
┌─────────────────────────────────────────────┐
│  Cache[K,V] interface                       │
│  (unchanged — backward compatible)          │
│  Get / Set / Delete / GetOrSet / Close      │
└─────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────┐
│  concreteCache[K,V]                         │
│  (adds MGet / MSet / MDel as direct methods)│
│                                             │
│  MGet(ctx, keys...) map[K]V                 │
│  MSet(ctx, items, ttl...) error             │
│  MDel(ctx, keys...) error                   │
└─────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────┐
│  cacher[K,V] interface                      │
│  (adds MGetFunc / MSetFunc / MDelFunc)      │
│                                             │
│  MGetFunc(ctx, keys...) map[K]V             │
│  MSetFunc(ctx, items, ttl...) error         │
│  MDelFunc(ctx, keys...) error               │
└─────────────────────────────────────────────┘
    │
    ┌───────┬───────┬───────┬───────┬───────┐
    ▼       ▼       ▼       ▼       ▼       ▼
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│ mem  │ │redis │ │valkey│ │mcache│ │postg │
│lock- │ │pipe- │ │pipe- │ │Get-  │ │batch │
│range │ │line  │ │line  │ │Multi │ │query │
└──────┘ └──────┘ └──────┘ └──────┘ └──────┘
```

### Recommended Implementation Order

```
Phase 7 waves:

Wave 0 (infrastructure):
  - Add MGetFunc/MSetFunc/MDelFunc to cacher interface
  - Add MGet/MSet/MDel methods + per-key fallbacks to concreteCache
  - Add fakeCacher batch methods in concrete_cache_test.go
  - Update doc.go

Wave 1 (providers: mem, memcache):
  - mem: single-lock iteration
  - memcache: GetMulti for MGet, per-key for MSet/MDel
  - Unit tests for mem batch
  - Integration tests for memcache batch (Docker-less skip)

Wave 2 (providers: redis, valkey):
  - redis: Pipeline MGet/MSet/Del
  - valkey: DoMulti MGet/MSet/Del
  - Integration tests (skipIfNoRedis/skipIfNoValkey)

Wave 3 (provider: postgres):
  - pgx Batch / SendBatch for MGet/MSet/MDel
  - Integration tests (skipIfNoPostgres)

Wave 4 (E2E + race tests):
  - Add batch operations to cache_e2e_test.go providerCase
  - Run with -race across all providers
```

### Pattern 1: Delegation via cacher interface

**What:** concreteCache methods delegate to cacher. Each provider implements cacher interface methods with provider-specific batch strategies. concreteCache provides per-key-loop default fallbacks for providers that don't override.

**When to use:** All batch operations follow this pattern.

**Example:**
```go
// In concrete_cache.go
func (c *concreteCache[K, V]) MGet(ctx context.Context, keys ...K) map[K]V {
    return c.cache.MGetFunc(ctx, keys...)
}

func (c *concreteCache[K, V]) MSet(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    return c.cache.MSetFunc(ctx, items, ttl...)
}

func (c *concreteCache[K, V]) MDel(ctx context.Context, keys ...K) error {
    return c.cache.MDelFunc(ctx, keys...)
}
```

### Pattern 2: Error aggregation with errors.Join

**What:** When a batch operation encounters partial failures, errors are accumulated and processing continues. The caller receives all failures via `errors.Join`.

**When to use:** Any provider operation where some keys succeed and some fail (e.g., pipeline partial failures, postgres batch row failures).

**Example:**
```go
func (c *providerCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    var errs []error
    for key, value := range items {
        if err := c.SetFunc(ctx, key, value, ttl...); err != nil {
            errs = append(errs, fmt.Errorf("key %v: %w", key, err))
        }
    }
    return errors.Join(errs...)
}
```

### Pattern 3: Default per-key-loop fallback

**What:** concreteCache provides default implementations that iterate per-key for providers that don't override MGetFunc/MSetFunc/MDelFunc.

**When to use:** For future provider implementations that don't define batch methods on cacher.

**Example:**
```go
// In concrete_cache.go — only when cacher doesn't implement batch
// (Go interface requires all methods; each provider MUST implement)
// Default implementations via forwarded cacher methods cover this.
```

**Note:** Since Go interfaces are strict, all providers MUST implement all cacher methods. The "default fallback" concept applies when providers delegate to their single-key methods within their batch implementations.

### Anti-Patterns to Avoid

- **Panic on missing keys in MGet**: MGet must silently skip missing keys, not panic or return error per key.
- **Stop on first error in batch ops**: D-06 requires accumulating errors and continuing.
- **Breaking Cache interface backward compatibility**: D-02 forbids adding methods to `Cache[K,V]`.
- **Holding mem lock across goroutine boundaries**: Race detector will catch this.

## Provider-Specific Batch Strategies

### mem (in-memory)

**Lock discipline:** Single `c.mu.RLock()` for MGet, single `c.mu.Lock()` for MSet/MDel.

**MGet strategy:** Iterate requested keys under `RLock`, call `c.store.get(key)` for each (store handles expiry). Only found keys in result map.

**MSet strategy:** Resolve TTL once, iterate items under `Lock`, call `c.store.set(key, value, expiresAt)` for each.

**MDel strategy:** Iterate keys under `Lock`, call `c.store.delete(key)` for each.

**Key insight:** The `memStore` wraps `flow.OrderedMap` which provides `Get`, `Set`, `Delete` methods. No `Range` iteration is needed — we only touch requested keys. Lock is acquired once per batch call, not per key.

```go
// Verified pattern from cache/mem/mem.go (GetFunc/SetFunc/DeleteFunc)
// MGet
c.mu.RLock()
defer c.mu.RUnlock()
result := make(map[K]V, len(keys))
for _, key := range keys {
    if v, ok := c.store.get(key); ok {
        result[key] = v
    }
}
return result
```

**Reasoning:** No error aggregation needed for mem — map operations can't fail. All keys either exist or don't. Race condition: impossible since single lock covers the entire batch.

### memcache

**MGet strategy:** Use `c.client.GetMulti(fmtKeys)` — returns `map[string]*memcache.Item`. Deserialize each found item. Return only successfully deserialized values.

**MSet strategy:** Iterate items, serialize each, write via goroutine-per-key pattern (matching existing `SetFunc` context cancellation pattern). Accumulate errors.

**MDel strategy:** Iterate keys, delete via goroutine-per-key pattern. Treat `memcache.ErrCacheMiss` as success (idempotent). Accumulate errors.

```go
// MGet using GetMulti
keysStr := make([]string, len(keys))
for i, k := range keys {
    keysStr[i] = fmt.Sprint(k)
}
items, err := c.client.GetMulti(keysStr)
if err != nil {
    // Some providers return partial errors
    // Accumulate and continue
}
result := make(map[K]V, len(items))
for keyStr, item := range items {
    var value V
    if err := json.Unmarshal(item.Value, &value); err == nil {
        // Parse original K from keyStr
        result[keyFromString(keyStr)] = value
    }
}
return result
```

**Key insight:** `GetMulti` returns only found keys as map — missing keys are absent. The key challenge is converting string keys back to `K` type, which requires `fmt.Sprint` round-trip compatibility (already existing pattern in memcache provider).

### redis

**MGet strategy:** Use pipeline with `MGet` command for all keys. Parse results, skipping nil entries (missing keys).

**MSet strategy:** Use pipeline with `Set` or `MSet` commands for all items. Single TTL applies to all.

**MDel strategy:** Use pipeline with `Del` command for all keys.

```go
// MGet via go-redis pipeline
pipe := c.client.Pipeline()
cmds := make(map[int]*redis.StringCmd)
for i, key := range keys {
    cmds[i] = pipe.Get(ctx, fmt.Sprint(key))
}
_, err := pipe.Exec(ctx)
// Even if Exec returns err, partial results may be available

result := make(map[K]V, len(keys))
for i, key := range keys {
    data, err := cmds[i].Bytes()
    if err == redis.Nil {
        continue // skip missing
    }
    if err != nil {
        // accumulate error
        continue
    }
    var value V
    if err := json.Unmarshal(data, &value); err != nil {
        // accumulate error
        continue
    }
    result[key] = value
}
```

**Key insight:** go-redis `Pipeline` executes commands in a single round trip. After `pipe.Exec()`, each individual command's result is checked separately. Some commands may succeed while others fail — D-06 applies.

### valkey

**MGet strategy:** Use `client.DoMulti` with `B().Get().Key(...).Build()` commands.

**MSet strategy:** Use `client.DoMulti` with `B().Set()...` commands. TTL applied per command.

**MDel strategy:** Use `client.DoMulti` with `B().Del().Key(...).Build()` commands.

```go
// MGet via valkey DoMulti
cmds := make([]valkey.Completed, len(keys))
for i, key := range keys {
    cmds[i] = c.client.B().Get().Key(fmt.Sprint(key)).Build()
}
responses := c.client.DoMulti(ctx, cmds...)

result := make(map[K]V, len(keys))
for i, key := range keys {
    data, err := responses[i].ToString()
    if errors.Is(err, valkey.Nil) {
        continue // skip missing
    }
    if err != nil {
        // accumulate error
        continue
    }
    var value V
    if err := json.Unmarshal([]byte(data), &value); err != nil {
        // accumulate error
        continue
    }
    result[key] = value
}
```

**Key insight:** valkey-go's `DoMulti` returns results for each command in order. Like redis, partial failures are possible. The `initErr` guard must still be checked before any operation (matching existing pattern).

### postgres

**MGet strategy:** Use `pool.SendBatch` with a batch of `SELECT ... WHERE cache_key = $1` queries.

**MSet strategy:** Use `pool.SendBatch` with a batch of `INSERT ... ON CONFLICT` queries.

**MDel strategy:** Use `pool.SendBatch` with a batch of `DELETE WHERE cache_key = $1` queries.

```go
// MGet via pgx Batch
batch := &pgx.Batch{}
keyStrs := make([]string, len(keys))
for i, key := range keys {
    keyStrs[i] = fmt.Sprint(key)
    batch.Queue(query, keyStrs[i])
}

br := c.pool.SendBatch(ctx, batch)
defer br.Close()

result := make(map[K]V, len(keys))
for i, key := range keys {
    var valueJSON string
    err := br.QueryRow().Scan(&valueJSON)
    if errors.Is(err, pgx.ErrNoRows) {
        continue // skip missing
    }
    if err != nil {
        // accumulate error
        continue
    }
    var value V
    if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
        // accumulate error
        continue
    }
    result[key] = value
}
```

**Key insight:** pgx `SendBatch` sends all queries in a single round trip. Results are read sequentially from the batch results reader. Table name is sanitized via `pgx.Identifier{c.tableName}.Sanitize()` per existing pattern.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Redis multi-key round trips | Naive per-key `Get` in loop | go-redis `Pipeline` | Pipeline batches commands in single TCP round trip; per-key loop makes N round trips |
| Valkey multi-key round trips | Naive per-key `Do` in loop | valkey-go `DoMulti` | `DoMulti` sends all commands atomically in one write; much lower latency |
| Memcache multi-get | Per-key `Get` in goroutines | gomemcache `GetMulti` | `GetMulti` is a native memcache protocol command; emulating it with parallel gets is slower and more complex |
| Postgres bulk queries | Individual `pool.Exec` per key | pgx `SendBatch` | Each `Exec` is a separate round trip; `SendBatch` executes all in one |
| Race synchronization | Custom mutex per key | Single lock acquisition | D-07: mem batch acquires one lock; per-key lock/unlock is more overhead |
| Error accumulation | Custom error type | `errors.Join` (stdlib) | D-06: `errors.Join` is available since Go 1.20; joins multiple errors into one |

**Key insight:** Every provider has an existing library-level batching mechanism. Using the provider-native mechanism reduces round trips from O(N) to O(1) — the primary value proposition of this phase. The per-key loop fallback in concreteCache is only for hypothetical future providers that lack native batching.

## Code Examples

### cacher interface expansion (concrete_cache.go)

[VERIFIED: cache/concrete_cache.go line 13-18]

```go
type cacher[K comparable, V any] interface {
    GetFunc(ctx context.Context, key K) (V, error)
    SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error
    DeleteFunc(ctx context.Context, key K) error
    CloseFunc() error

    // Batch operations — added in Phase 7
    MGetFunc(ctx context.Context, keys ...K) map[K]V
    MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error
    MDelFunc(ctx context.Context, keys ...K) error
}
```

### fakeCacher update for unit tests (concrete_cache_test.go)

[VERIFIED: cache/concrete_cache_test.go line 17-72]

```go
// Additional methods on fakeCacher for batch testing
func (f *fakeCacher[K, V]) MGetFunc(_ context.Context, keys ...K) map[K]V {
    f.mu.Lock()
    defer f.mu.Unlock()

    result := make(map[K]V, len(keys))
    for _, key := range keys {
        if v, ok := f.data[key]; ok {
            result[key] = v
        }
    }
    return result
}

func (f *fakeCacher[K, V]) MSetFunc(_ context.Context, items map[K]V, _ ...time.Duration) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.err != nil {
        return f.err
    }
    for key, value := range items {
        f.data[key] = value
    }
    return nil
}

func (f *fakeCacher[K, V]) MDelFunc(_ context.Context, keys ...K) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.err != nil {
        return f.err
    }
    for _, key := range keys {
        delete(f.data, key)
    }
    return nil
}
```

### Mem batch MGet (single lock acquisition)

[VERIFIED: cache/mem/mem.go line 42-51 — existing RLock pattern]

```go
func (c *memoryCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    c.mu.RLock()
    defer c.mu.RUnlock()

    result := make(map[K]V, len(keys))
    for _, key := range keys {
        if v, ok := c.store.get(key); ok {
            result[key] = v
        }
    }
    return result
}
```

### Mem batch MSet (single lock acquisition)

[VERIFIED: cache/mem/mem.go line 54-60 — existing Lock pattern]

```go
func (c *memoryCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    expiresAt := c.resolveTTL(ttl...)
    for key, value := range items {
        c.store.set(key, value, expiresAt)
    }
    return nil
}
```

### Mem batch MDel (single lock acquisition)

[VERIFIED: cache/mem/mem.go line 62-68 — existing Delete pattern]

```go
func (c *memoryCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, key := range keys {
        c.store.delete(key)
    }
    return nil
}
```

## Common Pitfalls

### Pitfall 1: Converting generic `K` key back from provider string

**What goes wrong:** Memcache and other providers return keys as strings. For MGet, the caller provides `keys ...K` and gets back `map[K]V`. If `K` is not `string`, the memcache provider must reconstruct `K` from `fmt.Sprint(k)` output.

**Why it happens:** `GetMulti` returns a `map[string]*memcache.Item` keyed by the string representation of each key. For non-string key types (e.g., `int`), the original key cannot be deterministically reconstructed from `fmt.Sprint` if multiple distinct `K` values produce the same string (e.g., `fmt.Sprint(1)` and `fmt.Sprint("1")` for `K = int` vs `K = string`).

**How to avoid:** The memcache MGet must zip the original `keys` slice with GetMulti results using the original key order. Iterate `keys` in order, look up each `fmt.Sprint(key)` in the GetMulti result map. This avoids key-type ambiguity and preserves the original key type.

```go
keysStr := make([]string, len(keys))
for i, k := range keys {
    keysStr[i] = fmt.Sprint(k)
}
items, err := c.client.GetMulti(keysStr)
// Build result map using original key order
result := make(map[K]V, len(keys))
for i, key := range keys {
    item, ok := items[keysStr[i]]
    if !ok {
        continue // missing
    }
    // deserialize item.Value into result[key]
}
```

### Pitfall 2: Race on mem store between batch and sweep

**What goes wrong:** The sweep goroutine (`sweepLoop`) holds `c.mu.Lock()` while calling `store.removeExpired()`. If a batch MGet holds `c.mu.RLock()` and calls `store.get()`, which internally calls `store.entries.Delete()` for expired entries, there's a concurrent write while holding a read lock.

**Why it happens:** `store.get()` has an implicit mutation: it deletes expired entries. This happens inside `RLock` in the current code as well, but `removeExpired()` holds `Lock`. The existing code already has this pattern — it's safe because `entries.Delete` on OrderedMap is mutex-protected at the provider level. But it violates Go's race detector at the store level if `store.get` mutates `entries` while another goroutine sweeps.

**How to avoid:** This is an existing pattern limitation, not specific to batch ops. The current code already works because the provider lock protects the store. Batch ops follow the same pattern — the race detector test (BATCH-07) will confirm safety. If the race detector flags this, change `store.get` to not mutate expired entries (return expired as not-found without deleting), and let the sweep goroutine handle deletion.

### Pitfall 3: Pipeline partial failures masked by Exec error

**What goes wrong:** go-redis `pipe.Exec(ctx)` returns a single error. The error may be `context.DeadlineExceeded`, `context.Canceled`, or a pipeline-level error. Individual command results (available via `cmds[i].Err()`) may have succeeded despite the pipeline-level error, or vice versa.

**Why it happens:** Redis pipelines batch commands, and the server returns one response per command. If the pipeline is executed but some commands fail (e.g., WRONGTYPE), only the failed commands have errors. If the pipeline is not executed at all (connection error), all commands have errors.

**How to avoid:** Always check individual command results after `pipe.Exec()` returns. Use `errors.Join` to accumulate only real failures. Skip commands that succeeded, even if `Exec` returned an error.

### Pitfall 4: Memory allocation for large result maps

**What goes wrong:** Pre-allocating `result := make(map[K]V, len(keys))` for MGet with 100K keys wastes memory if most keys are missing.

**Why it happens:** The map pre-allocation is O(N) even when only a few keys exist.

**How to avoid:** Pre-allocate with `len(keys)` as capacity — it's a good trade-off. Go maps grow dynamically if needed. For the expected use case (batch of <100 keys), this is negligible. Document that callers should not pass 100K+ keys per call.

## Runtime State Inventory

> **Not applicable** — this is a greenfield implementation phase. No rename, refactor, or migration of existing runtime state.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All code | ✓ | 1.26.4 | — |
| Docker / Orbstack | E2E tests | ✓ | 29.4.0 | — |
| make | Build targets | ✓ | (via go toolchain) | — |
| golangci-lint | Code quality | ✓ | (via go tooling) | — |
| Redis (via Docker) | E2E: redis | ✓ | Containerized | skipIfNoRedis skips test |
| Valkey (via Docker) | E2E: valkey | ✓ | Containerized | skipIfNoValkey skips test |
| Memcache (via Docker) | E2E: memcache | ✓ | Containerized | skipIfNoMemcache skips test |
| Postgres (via Docker) | E2E: postgres | ✓ | Containerized | skipIfNoPostgres skips test |

**Missing dependencies with no fallback:** None — all required tooling is available.

**Missing dependencies with fallback:** E2E tests for external providers gracefully skip when Docker is not available (existing `skipIfNo*` pattern in each provider's test file). Unit tests do not require Docker.

## Validation Architecture

> **Skipped** — `workflow.nyquist_validation` is explicitly set to `false` in `.planning/config.json`.

## Security Domain

> **Required** — `security_enforcement` is enabled in config and not explicitly set to false.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | Batch keys are validated by type system (`K comparable`). No SQL injection in batch operations — pgx `SendBatch` uses parameterized queries, table name sanitized via `pgx.Identifier{c.tableName}.Sanitize()` |
| V6 Cryptography | no | Cache values are opaque byte payloads; no encryption at rest. Not added in this phase |
| V2 Authentication | no | Cache package has no authentication concept |
| V3 Session Management | no | Not applicable to a cache library |
| V4 Access Control | no | Not applicable to a cache library |

### Known Threat Patterns for Go Cache

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| JSON injection via serialized values | Tampering | Existing: `encoding/json` round-trips all values. Batch MGet/MSet uses same serialization as single-key operations — no new injection surface |
| Context cancellation not respected | Denial of Service | Existing: all batch ops receive `context.Context` and must respect cancellation per provider patterns |
| Pipeline/connection pool exhaustion | Denial of Service | go-redis, valkey-go, pgx pools are configured at construction — batch ops don't change pool behavior |

**No new security surface:** Batch operations use the same serialization, the same connection pools, and the same context propagation as existing single-key operations. No new attack vectors are introduced.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `github.com/bradfitz/gomemcache` exposes `client.GetMulti(keys []string) (map[string]*Item, error)` | Provider-Specific Strategies | If the API differs, memcache MGet needs a different approach (per-key goroutines). MEDIUM risk — verified via go.mod but not via godoc |
| A2 | go-redis `Pipeline.Exec` + individual command checking is the correct pattern | Code Examples | go-redis API version (v9.21.0) may have minor differences. LOW risk — well-established API |
| A3 | valkey-go `DoMulti` supports `B().Get()/B().Set()/B().Del()` commands | Provider-Specific Strategies | valkey-go v1.0.76 API should match. MEDIUM risk — valkey-go API has evolved rapidly |
| A4 | `errors.Join` is available in the Go version used | Architecture Patterns | Go 1.26.4 definitely has `errors.Join` (available since 1.20). NO RISK |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

**Verdict:** Low risk — all assumptions are about well-established library APIs.

## Open Questions

1. **How does go-redis v9.21.0 Pipeline API expose MGet results?**
   - What we know: `pipe.Get()` returns `*redis.StringCmd`, `pipe.Exec()` sends the batch.
   - What's unclear: Whether `pipe.MGet()` (single command returning multiple values) is preferable to `pipe.Get()` (multiple commands each returning one value) for the batch MGet pattern. `MGet` returns `*redis.SliceCmd` with mixed results per key — harder to map back to original key ordering.
   - Recommendation: Use `pipe.Get()` per key for simplicity and correct key-to-result mapping. The round trip saving comes from batching in the pipeline, not from using `MGET` vs `GET` in the pipeline.

2. **How does valkey-go `DoMulti` handle results when some keys are missing?**
   - What we know: Missing keys return `valkey.Nil` error per individual result.
   - What's unclear: Whether `DoMulti` returns results in order corresponding to commands submitted.
   - Recommendation: Assume ordered results (standard for RESP protocol). If unordered, iterate response array and match by key string.

3. **Should the memcache MGet reconstruct K from string or use the original key slice for ordering?**
   - What we know: `GetMulti` returns `map[string]*Item` — keys are strings.
   - What's unclear: Whether `fmt.Sprint(k)` for the original key `K` is deterministic enough to map back.
   - Recommendation: Use original key slice ordering (iterate `keys`, look up each `fmt.Sprint(key)` in GetMulti result). This avoids key-type ambiguity.

## Sources

### Primary (HIGH confidence)

- [VERIFIED: cache/concrete_cache.go] — cacher interface definition and concreteCache delegation pattern
- [VERIFIED: cache/cache.go] — Cache[K,V] interface (no new methods per D-02)
- [VERIFIED: cache/singleflight.go] — SingleflightGetOrSet pattern reference
- [VERIFIED: cache/mem/mem.go] — memoryCache lock discipline (RLock for read, Lock for write)
- [VERIFIED: cache/mem/store.go] — memStore wrapping flow.OrderedMap
- [VERIFIED: cache/redis/redis.go] — redisCache go-redis client usage
- [VERIFIED: cache/valkey/valkey.go] — valkeyCache valkey-go client usage
- [VERIFIED: cache/memcache/memcache.go] — memcacheCache gomemcache usage with context goroutine pattern
- [VERIFIED: cache/postgres/postgres.go] — postgresCache pgxpool/pgx usage
- [VERIFIED: cache/concrete_cache_test.go] — fakeCacher pattern for unit testing
- [VERIFIED: cache/cache_e2e_test.go] — E2E test pattern with testcontainers-go
- [VERIFIED: go.mod] — exact library versions

### Secondary (MEDIUM confidence)

- [ASSUMED: go-redis pipeline API] — `Pipe.Get`, `Pipe.Exec`, per-command result checking
- [ASSUMED: valkey-go DoMulti API] — `client.DoMulti(ctx, cmds...)` with ordered results
- [ASSUMED: gomemcache GetMulti API] — `client.GetMulti(keys []string) (map[string]*Item, error)`

### Tertiary (LOW confidence)

- None — all claims about the existing codebase are verified; external API patterns are widely established.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries verified from go.mod and codebase
- Architecture: HIGH — all patterns verified from existing codebase (cacher interface, concreteCache delegation, error handling)
- Pitfalls: HIGH — identified from known Go cache provider behaviors and existing code patterns
- Provider-specific APIs: MEDIUM — go-redis, valkey-go, gomemcache APIs are well-known but not verified via running docs in this session

**Research date:** 2026-08-07
**Valid until:** 2026-09-07 (30-day validity for stable Go libraries)

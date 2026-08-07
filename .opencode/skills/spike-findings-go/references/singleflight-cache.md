# Singleflight in Cache GetOrSet

## Requirements

These are non-negotiable for the real build (from MANIFEST.md session requirements
"Spike Session 2026-08-06: singleflight in cache GetOrSet").

- `singleflight.Group` must wrap only the setter, not the whole Get/Set dance.
- The `shared` return is a **global** signal (leader sees `true` too when a
  follower arrived) — never use it as an "I am the follower" check.
- Error from a failing setter is shared with **all** waiters (same error object).
- Placement must be **shared code**, not duplicated per provider.
- For HTTP-facing GetOrSet prefer **`DoChan` + select on the caller's ctx**
  (spike 008): canceled waiters release at cancel latency instead of blocking
  for the setter's full duration. Blocking `Do()` is acceptable only for short,
  guaranteed-fast setters. A single shared wrapper should offer both variants.
- Panicking setters fan out panics to every waiter — the wrapper must recover
  and return an error instead.
- singleflight does **not** cache completed results (the group key is deleted on
  completion) — per-key TTL stays fully owned by the provider.
- The singleflight fn must **double-check Get** before computing to avoid a slow
  in-flight computation clobbering a concurrent direct `Set`.
- **Error semantics differ per provider** (spike 006): redis/valkey wrap setter
  errors with a provider prefix; mem/memcache/postgres return them raw; valkey
  short-circuits on `initErr`. Never hardcode a prefix inside the shared helper.
- **`group.Forget(key)` does NOT stop an in-flight leader** (spike 007): it only
  unregisters the key for future callers. To prevent a Delete-during-flight from
  being resurrected by the leader's late Set, re-check a deletion-generation
  tombstone inside the `Do` fn right before calling Set.

## How to Build It

A single shared helper in the `cache` package. Each provider embeds it and
delegates, passing its own `Get`/`Set` closures. This eliminates the 5× duplicated
GetOrSet bodies (mem, redis, valkey, memcache, postgres all hand-roll the same
miss→setter→Set today).

`golang.org/x/sync` is already a dependency (v0.22.0) — no new module needed.

```go
// in cache package
import "golang.org/x/sync/singleflight"

type singleflightGetOrSet[K comparable, V any] struct {
    group singleflight.Group
}

func (s *singleflightGetOrSet[K, V]) do(
    ctx context.Context,
    key K,
    get func(context.Context, K) (V, error),
    set func(context.Context, K, V, ...time.Duration) error,
    setter func() (V, error),
    ttl ...time.Duration,
) (V, error) {
    var zero V
    if v, err := get(ctx, key); err == nil {
        return v, nil
    }

    v, err, _ := s.group.Do(fmt.Sprint(key), func() (any, error) {
        // double-check: another leader may have already Set while we queued
        if v, err := get(ctx, key); err == nil {
            return v, nil
        }
        computed, err := setter()
        if err != nil {
            return zero, err
        }
        if err := set(ctx, key, computed, ttl...); err != nil {
            return zero, err
        }
        return computed, nil
    })
    if err != nil {
        return zero, err
    }
    return v.(V), nil
}
```

**Context-aware variant (spike 008)** — use for HTTP-facing GetOrSet. Waiters
release on caller cancel instead of blocking for the setter's full duration:

```go
func (s *singleflightGetOrSet[K, V]) doChan(
    ctx context.Context,
    key K,
    get func(context.Context, K) (V, error),
    set func(context.Context, K, V, ...time.Duration) error,
    setter func() (V, error),
    ttl ...time.Duration,
) (V, error) {
    var zero V
    if v, err := get(ctx, key); err == nil {
        return v, nil
    }

    ch := s.group.DoChan(fmt.Sprint(key), func() (any, error) {
        if v, err := get(ctx, key); err == nil {
            return v, nil
        }
        computed, err := setter()
        if err != nil {
            return zero, err
        }
        if err := set(ctx, key, computed, ttl...); err != nil {
            return zero, err
        }
        return computed, nil
    })
    select {
    case <-ctx.Done():
        return zero, ctx.Err()
    case res := <-ch:
        if res.Err != nil {
            return zero, res.Err
        }
        return res.Val.(V), nil
    }
}
```

Provider side (e.g. mem):

```go
type Cache[K comparable, V any] struct {
    ...
    sf singleflightGetOrSet[K, V]   // zero value is fine
}

func (c *Cache[K, V]) GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error) {
    return c.sf.do(ctx, key, c.Get, c.Set, setter, ttl...)
}
```

**Per-provider error glue (spike 006)** — the shared helper must return the
setter error verbatim. Prefixing belongs to the providers that historically do
it (redis → `cache/redis:`, valkey → `cache/valkey:`), so those thin GetOrSet
bodies wrap the helper's error themselves:

```go
// redis GetOrSet (thin): keeps its historical error prefix
v, err := c.sf.do(ctx, key, c.Get, c.Set, setter, ttl...)
if err != nil {
    return zero, fmt.Errorf("cache/redis: %w", err)
}
return v, nil
```

mem/memcache/postgres return the helper's error raw. Also note valkey carries an
`initErr` guard (returns early if the client failed to initialize) — that guard
must stay provider-side, it cannot move into the shared helper.

**Delete-during-flight guard (spike 007)** — `group.Forget(key)` does NOT stop
an in-flight leader from later calling Set, so a Delete that lands mid-flight
would be resurrected. The provider keeps a deletion generation; the shared fn
re-checks it right before Set:

```go
// provider supplies: hasDeleted func(key K) bool
computed, err := setter()
if err != nil {
    return zero, err
}
if hasDeleted(key) {        // tombstone re-checked at the last moment
    return zero, cache.ErrKeyDeleted
}
if err := set(ctx, key, computed, ttl...); err != nil {
    return zero, err
}
return computed, nil
```

Verification (all proven in spikes):

- 50–40 concurrent misses → setter runs **exactly once**, all callers share the value.
- Subsequent GetOrSet returns the cached value without re-running the setter.
- Run with `go run -race .` — clean, no data race.
- **Per-provider parity (spike 006):** against real mem/redis/valkey/memcache/
  postgres, error strings match the historical format exactly (redis/valkey keep
  prefixes; mem/memcache/postgres keep raw setter errors).
- **Delete vs in-flight (spike 007):** a Delete issued while the leader computes
  is not resurrected — the tombstone re-check aborts the late Set. `Forget` alone
  is NOT sufficient (verified failure).
- **Cancellation (spike 008):** waiters with a canceled ctx return in ~15 ms via
  DoChan+select; blocking Do holds them ~300 ms until the setter finishes.
- **Herd benchmark (spike 009):** 64 concurrent callers on a ~100 ms setter —
  naive runs the setter 64×; singleflight runs it **once**. Wall-time ~2.31 ms
  vs ~1.17 ms (worst case), allocs 15,340 B/op vs 8,716 B/op, 145 vs 87
  allocs/op. Dedup alone yields ~2× speedup and ~half the allocations.

## What to Avoid

- **Treating `shared` as a per-caller flag.** `Group.Do` returns `c.dups > 0`
  computed at *return time*. The leader returns `shared=true` whenever any
  follower was waiting. Asserting `shared == callers-1` fails — use `== callers`.
- **Putting the `Set` outside the singleflight fn.** Followers would each re-Set,
  and you cannot identify the leader from `shared` alone. Put Set inside the fn
  (runs once) and rely on the double-check Get for the clobber guard.
- **Not double-checking Get inside the fn.** A concurrent direct `Set(key, fresh)`
  landing while the leader computes gets overwritten by the older in-flight result.
  Always re-read inside the fn before computing.
- **Expecting `Do()` to honor the caller's context.** It is context-blind. A dead
  caller stays blocked until the leader completes. Use `DoChan` + select if
  per-caller cancellation is required — never assume abort works.
- **Depending on `Forget` to stop an in-flight write (spike 007).** `Forget`
  only removes the key from future registration; the already-running leader still
  completes its `Set`. Prove any delete-cancellation with a generation tombstone,
  not `Forget`.
- **Hardcoding provider error prefixes in the shared helper (spike 006).** The
  helper is provider-agnostic; redis/valkey wrap its error in their thin GetOrSet.
- **Letting a setter panic escape.** singleflight captures the panic and replays it
  to every waiting caller (a `*panicError`), and to extra goroutines when channels
  are involved. Recover inside the wrapper and return an error.
- **Wrapping the whole Get/Set in the group.** Only the setter + Set should be
  inside; the fast-path Get stays outside to avoid lock contention.

## Constraints

- `golang.org/x/sync/singleflight` v0.22.0. API: `Do`, `DoChan`, `Forget`,
  `Result{Val, Err, Shared}`. Verified via `go doc -src`.
- `Do(key, fn)` returns `(v any, err error, shared bool)`. Error value identity is
  preserved to all waiters — treat as read-only (do not mutate or the shared
  pointer breaks equality checks).
- `doCall` deletes the group key on completion — singleflight never caches
  completed results, so freshness is controlled by the provider's TTL alone.
- Staleness window: a caller joining an in-flight computation gets the result "as
  of when the leader started". Bounded by the setter's duration. Acceptable for
  cache semantics; document it.
- Thread-safety: `Group` is safe for concurrent use; same key dedups, different
  keys run in parallel.

## Origin

Synthesized from spikes: 002, 003, 004, 005, 006, 007, 008, 009
Source files available in: sources/002-singleflight-dedup/, sources/003-singleflight-failure/, sources/004-singleflight-ttl/, sources/005-singleflight-placement/, sources/006-singleflight-real-providers/, sources/007-singleflight-delete/, sources/008-singleflight-context/, sources/009-singleflight-benchmark/
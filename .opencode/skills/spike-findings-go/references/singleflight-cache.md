# Singleflight in Cache GetOrSet

## Requirements

These are non-negotiable for the real build (from MANIFEST.md session requirements
"Spike Session 2026-08-06: singleflight in cache GetOrSet").

- `singleflight.Group` must wrap only the setter, not the whole Get/Set dance.
- The `shared` return is a **global** signal (leader sees `true` too when a
  follower arrived) — never use it as an "I am the follower" check.
- Error from a failing setter is shared with **all** waiters (same error object).
- Placement must be **shared code**, not duplicated per provider.
- Use blocking `Do()` — it is context-blind; a canceled waiter can only abandon
  via `DoChan` + select. Accept this tradeoff for a cache `GetOrSet`.
- Panicking setters fan out panics to every waiter — the wrapper must recover
  and return an error instead.
- singleflight does **not** cache completed results (the group key is deleted on
  completion) — per-key TTL stays fully owned by the provider.
- The singleflight fn must **double-check Get** before computing to avoid a slow
  in-flight computation clobbering a concurrent direct `Set`.

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

Verification (all proven in spikes):

- 50–40 concurrent misses → setter runs **exactly once**, all callers share the value.
- Subsequent GetOrSet returns the cached value without re-running the setter.
- Run with `go run -race .` — clean, no data race.

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

Synthesized from spikes: 002, 003, 004, 005
Source files available in: sources/002-singleflight-dedup/, sources/003-singleflight-failure/, sources/004-singleflight-ttl/, sources/005-singleflight-placement/
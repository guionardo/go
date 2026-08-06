---
spike: 004
name: singleflight-ttl
type: standard
validates: "Given per-key TTL and a long-lived singleflight group, when values expire mid-flight, then expiry forces recompute, direct Set is not clobbered, and staleness is bounded by setter duration"
verdict: VALIDATED
related: [002, 003, 005]
tags: [singleflight, concurrency, cache, ttl]
---

# Spike 004: singleflight-ttl

## What This Validates

Whether a long-lived `singleflight.Group` violates cache TTL semantics: does it
serve stale results after expiry? Does a slow setter clobber a fresh direct
`Set`? What is the staleness window?

## How to Run

```bash
cd .planning/spikes/004-singleflight-ttl && go run .
```

## What to Expect

```
[A] values across calls: 1, 2, 3 — each call recomputes, no stale cache
CASE A: PASS — group forgets after completion; expiry naturally forces recompute
[B] follower at 101ms got: "computed@101ms" ... total computations: 1
CASE B: PASS — mid-flight result shared; staleness window = setter duration
[C] follower direct-set value observed: "fresh-direct-set"
CASE C: PASS — re-checking the store inside the singleflight fn prevents clobbering
```

## Investigation Trail

- Read `singleflight.doCall` source: on completion it does `delete(g.m, key)`.
  **singleflight never caches completed results** — it only dedups in-flight
  calls. Therefore a long-lived group cannot itself serve stale data after TTL
  expiry: the next `GetOrSet` miss starts a fresh computation.
- Case B reveals the real staleness caveat: a caller arriving after logical
  expiry but while a computation is in-flight will still receive that in-flight
  (older) result rather than triggering a new one. The staleness window equals
  the setter's duration.
- Case C: if a direct `Set(key, fresh)` lands while the group leader is
  computing, the leader's slower result would normally clobber it. The fix is
  the classic double-check: re-read the backing store *inside* the singleflight
  fn before computing.

## Results

VALIDATED. Design implications for the real build:

1. **TTL is safe:** singleflight does not cache, so per-key TTL in the provider's
   own `Set`/Get expiry logic fully controls freshness. No per-key group
   lifecycle needed.
2. **Mid-flight staleness:** a caller joining an in-flight computation gets the
   result "as of when the leader started". If the setter is slow, short-TTL keys
   can be served slightly stale once. Acceptable for cache semantics; document
   it.
3. **Clobbering guard (mandatory):** the singleflight fn MUST re-check the cache
   (Get) before invoking the setter. Otherwise a concurrent direct `Set` or a
   cache refresh is overwritten by the older in-flight computation. This is the
   same double-check used inside the current mem/redis GetOrSet paths — it must
   move INSIDE the singleflight fn.
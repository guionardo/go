# Spike Wrap-Up Summary

**Date:** 2026-08-06
**Spikes processed:** 8
**Feature areas:** Singleflight in Cache GetOrSet
**Skill output:** `./.opencode/skills/spike-findings-go/`

## Processed Spikes
| # | Name | Type | Verdict | Feature Area |
|---|------|------|---------|--------------|
| 002 | singleflight-dedup | standard | VALIDATED | Singleflight in Cache GetOrSet |
| 003 | singleflight-failure | standard | VALIDATED | Singleflight in Cache GetOrSet |
| 004 | singleflight-ttl | standard | VALIDATED | Singleflight in Cache GetOrSet |
| 005 | singleflight-placement | standard | VALIDATED | Singleflight in Cache GetOrSet |
| 006 | singleflight-real-providers | frontier | VALIDATED | Singleflight in Cache GetOrSet |
| 007 | singleflight-delete | frontier | PARTIAL | Singleflight in Cache GetOrSet |
| 008 | singleflight-context | frontier | VALIDATED | Singleflight in Cache GetOrSet |
| 009 | singleflight-benchmark | frontier | VALIDATED | Singleflight in Cache GetOrSet |

## Key Findings

1. **Dedup is perfect.** 50 concurrent misses on one key → setter runs exactly
   once; all callers receive the same value. Different keys run in parallel.
2. **`shared` is a global flag.** `Group.Do` returns `c.dups > 0` at return time,
   so the leader also reports `shared=true` when any follower waited. It can
   never be used to identify "the leader".
3. **`Do()` is context-blind.** A canceled waiter can only abandon its wait via
   `DoChan` + select on its own ctx; the leader always runs to completion. No
   abort mechanism exists from a follower's cancel.
4. **Panics fan out.** A panicking setter is captured as `*panicError` and the
   panic is replayed to every waiting caller. Wrappers must recover and return
   an error.
5. **TTL is safe.** `doCall` deletes the group key on completion, so singleflight
   never caches completed results — the provider's own TTL fully controls
   freshness. The only staleness window is a caller joining an in-flight
   computation (bounded by setter duration).
6. **Clobber guard required.** The singleflight fn must re-read the cache (Get)
   before computing, or a slow in-flight computation overwrites a concurrent
   fresh direct `Set`.
7. **One shared helper, not 5 copies.** A `cache`-package helper taking the
   provider's `Get`/`Set` closures serves all 5 providers race-free (verified
   `go run -race` against the real `cache/mem` provider). `golang.org/x/sync`
   v0.22.0 is already a dependency.
8. **Per-provider error glue (006).** The shared helper must return the setter
   error verbatim. redis/valkey keep their historical `cache/redis:` /
   `cache/valkey:` prefixes by wrapping the helper's error in their thin
   GetOrSet; mem/memcache/postgres return it raw. valkey's `initErr` guard
   cannot move into the shared helper.
9. **`Forget` is not a delete-cancel (007).** `group.Forget(key)` only removes
   the key from future registration — an in-flight leader still completes and
   would resurrect a concurrently-deleted key. Prevent resurrection with a
   deletion-generation tombstone re-checked inside the `Do` fn before Set.
10. **DoChan frees waiters (008).** Waiters with a canceled ctx return in ~15 ms
    via DoChan+select vs ~300 ms held by blocking Do. Prefer the DoChan variant
    for HTTP-facing GetOrSet.
11. **Dedup is measurably cheaper (009).** Benchmark on 64 concurrent callers /
    ~100 ms setter: setter runs 64× (naive) vs 1×; ~2.31 ms vs ~1.17 ms worst
    case, 15,340 B/op vs 8,716 B/op, 145 vs 87 allocs/op — ~2× faster, ~half the
    allocations. Benchmark via `go test -bench=. -benchmem -benchtime=2x`.

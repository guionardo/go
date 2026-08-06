---
spike: 002
name: singleflight-dedup
type: standard
validates: "Given N concurrent GetOrSet misses on the same missing key, when wrapped in singleflight, then the setter runs exactly once and all callers get the same value"
verdict: VALIDATED
related: [003, 004, 005]
tags: [singleflight, concurrency, cache, GetOrSet]
---

# Spike 002: singleflight-dedup

## What This Validates

Given 50 concurrent GetOrSet calls on the same missing key, when the setter is
wrapped in `singleflight.Group.Do`, then the setter runs exactly once and all
callers receive the same computed value.

## Research

Wrapped the setter (not the whole GetOrSet) in `singleflight.Group.Do(key, fn)`.
`group.Do` guarantees only one execution in-flight per key; duplicates wait on a
`sync.WaitGroup` and receive the original's results.

Key API facts from `go doc -src golang.org/x/sync/singleflight.Group.Do`:

- `Do(key, fn)` returns `(v any, err error, shared bool)`.
- Duplicate callers call `c.wg.Wait()` then return `c.val, c.err, true`.
- If the leader panics (`*panicError`) or calls `runtime.Goexit()`, the panic /
  goexit is **replayed** to every waiting caller.

## How to Run

```bash
cd .planning/spikes/002-singleflight-dedup && go run .
```

## What to Expect

```
callers:      50
setter runs:  1
all same val: true
SPIKE 002: VALIDATED
```

## Investigation Trail

- 1st attempt: asserted `shared == callers-1` (only followers shared). FAILED —
  leader also reported `shared=true` because a follower arrived during the
  5ms computation.
- Root cause: `Do` returns `c.dups > 0` computed at **return time**, not "was
  this caller a follower". Leaders report `shared=true` whenever any peer waited.
- Fix: assert `shared == callers` and document the real semantic.

## Results

VALIDATED. `singleflight` collapses N concurrent misses to a single setter call
with no code duplication and linear speedup of the expensive step (50 callers,
5ms setter → ~6ms total). Dedup behavior is solid.

**Critical nuance for the real build:** `shared` is a *global* signal (any peer
was waiting), not a *per-caller* one. If `shared` is used to decide whether to
re-check / re-cache (e.g. only the leader posts to the backing store), the guard
must account for the leader also seeing `shared=true`. See Spike 005 for
placement guidance.
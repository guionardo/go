---
spike: 008
name: dochan-context-aware
type: standard
validates: "Given a slow setter under cancel-heavy concurrent load, when the wrapper uses DoChan + select on the caller's ctx (vs blocking Do), then canceled waiters release promptly instead of piling up"
verdict: VALIDATED
related: [003]
tags: [singleflight, context, goroutines, DoChan]
---

# Spike 008: dochan-context-aware

## What This Validates

Spike 003 established `Do()` is context-blind — a waiter whose ctx is canceled
stays blocked until the leader finishes. This spike quantifies that cost under
cancel-heavy load and verifies `DoChan` + select as the mitigation.

## How to Run

```bash
cd .planning/spikes/008-singleflight-context && go run .
```

## What to Expect

```
  Do(blocking)      peak-goroutines=80   avg-waiter=~300.456601ms
  DoChan(ctx-aware) peak-goroutines=68   avg-waiter=~15.300778ms
```

## What This Validates

- 5 waves × 64 callers each cancel their ctx at 15ms; the setter takes 300ms.
- **Blocking `Do`:** waiters hold their stack + ctx for the full 300ms average.
- **`DoChan` + select:** waiters abandon their own wait at ~15ms (when the ctx
  fires) and return; the leader keeps computing to completion invisibly.

## Investigation Trail

- Peak goroutine counts (80 vs 68) are **not** the discriminator — the harness's
  `sync.WaitGroup` keeps all goroutines of a wave alive regardless of whether
  they are waiting. The real cost is waiter wall-time and that waiters pin
  their stack + request ctx (retaining memory, timers, and their goroutine
  slot) for the setter's full duration.
- DoChan keeps the dedup benefit (single setter execution) while freeing
  waiters immediately on cancel. A canceled waiter simply stops waiting; the
  leader still runs and its result is discarded (or cached by the DoubleCheck on
  a later `Do`).

## Results

VALIDATED. For request-scoped contexts, prefer `DoChan` + select over blocking
`Do`:

- Bounded waiter hold time (≈ cancel latency vs setter duration).
- No goroutine per caller parked across the whole slow computation.

Real-build impact: the shared helper should offer the DoChan variant (or take the
caller's `ctx` and internally select) for the cache's HTTP-facing method, keeping
the blocking `Do` only where short TTL/lazy setter is guaranteed. This is a small,
self-contained addition on top of the 005 helper (swap `g.Do` for `g.DoChan`).

Tradeoff: with DoChan, a canceled waiter's `ctx.Err()` is returned in its place
(waiter sees cancel, not the setter's eventual result) — callers must tolerate
either the value or `context.Canceled`. Documented, not a bug.
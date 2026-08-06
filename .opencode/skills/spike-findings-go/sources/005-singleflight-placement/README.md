---
spike: 005
name: singleflight-placement
type: standard
validates: "Given 5 near-identical provider GetOrSet implementations, when singleflight is factored into the shared cache package, then one copy serves all providers with dedup intact"
verdict: VALIDATED
related: [002, 003, 004]
tags: [singleflight, cache, design, GetOrSet]
---

# Spike 005: singleflight-placement

## What This Validates

The five providers (mem, redis, valkey, memcache, postgres) each hand-roll the
same GetOrSet: miss → `setter()` → Set. Adding singleflight to each would
duplicate the logic 5×. This spike validates a shared helper in the `cache`
package that providers delegate to, with all behaviors proven in spikes
002–004 baked in.

## How to Run

```bash
cd .planning/spikes/005-singleflight-placement && go run .          # dedup + cache hit
cd .planning/spikes/005-singleflight-placement && go run -race .   # race-clean
```

## What to Expect

```
callers=40 setterRuns=1 elapsed=3.7ms
cached read: "computed" err=<nil> setterRuns=1
SPIKE 005: VALIDATED — shared singleflightGetOrSet works against real mem provider
```

Clean under `-race`.

## Investigation Trail

- All 5 providers share the identical GetOrSet body (verified via grep: mem,
  redis, valkey, postgres, memcache all do Get-miss → setter → Set).
- Prototyped `singleflightGetOrSet[K, V]` in the spike as the shape for a
  `cache` package helper. Provider embeds the struct, calls `do(ctx, key,
  c.Get, c.Set, setter, ttl...)`.
- Tested against the REAL `cache/mem` provider (imported the actual package),
  not a mock — 40 concurrent callers, setter ran once, subsequent read was a
  cache hit.
- `-race` clean confirms no data race between concurrent Get/Set closures.

## Results

VALIDATED. The singleflight logic can live in ONE shared place (the `cache`
package) and every provider delegates, eliminating 5× duplication. The helper
must:

1. Accept the provider's `Get`/`Set` closures so it stays backend-agnostic.
2. Run the setter inside `group.Do` (spike 002 dedup).
3. Double-check `Get` inside the fn before computing (spike 004 clobber guard).
4. Propagate `setter` errors unchanged (spike 003).

**Tradeoff confirmed:** `Do` is context-blind (spike 003) — the shared helper
blocks waiters until the leader completes; there is no per-caller cancel unless
the helper switches to `DoChan` + select. For a cache `GetOrSet` this is an
acceptable, documented tradeoff.

**Alternative rejected:** a per-provider singleflight field duplicates the
pattern 5× and invites drift. The shared-helper approach is strictly better.
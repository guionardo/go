---
spike: 006
name: real-provider-integration
type: standard
validates: "Given the spike-005 shared helper, when all 5 providers delegate GetOrSet to it, then each provider's current error semantics (prefixes + valkey initErr guard) are preserved"
verdict: VALIDATED
related: [003, 005]
tags: [singleflight, cache, integration, error, redis, valkey]
---

# Spike 006: real-provider-integration

## What This Validates

The spike-005 helper takes only `get`/`set` closures. But real providers differ in
error handling: redis/valkey wrap setter errors with a provider prefix, while
mem/memcache/postgres return them raw; valkey additionally short-circuits on
`initErr`. This spike checks whether migrating all 5 providers to the shared helper
silently changes behavior.

## Research

Read all 5 provider GetOrSet implementations:

| Provider | setter error | initErr guard | ctx-aware transport |
|----------|-------------|---------------|---------------------|
| mem | RAW | none | mutex |
| memcache | RAW | none | goroutine+select |
| postgres | RAW | none | pgxpool |
| redis | WRAPPED `cache/redis:` | none | go-redis |
| valkey | WRAPPED `cache/valkey:` | YES (before setter) | valkey-go |

## How to Run

```bash
cd .planning/spikes/006-singleflight-real-providers && go run .
```

## What to Expect

```
[redis] 005-helper setter error: prefix_preserved=false err="setter-boom"
CASE redis: PASS — helper drops the 'cache/redis:' prefix; wrapping providers must wrap inside the delegated fn
[mem] helper setter error raw: err="mboom"
CASE mem: PASS — raw providers (mem/memcache/postgres) keep identical error identity
[valkey] with initErr, setterRan=false err="cache/valkey: disco-init-failed"
CASE valkey: PASS — initErr guard + prefix cannot move into helper; must stay provider-side
```

## Investigation Trail

- Confirmed the 005 helper returns `setter()` errors unwrapped. For redis/valkey
  this drops the `cache/redis:` / `cache/valkey:` prefix — a behavioral change
  consumers may rely on for error classification.
- Confirmed error **identity** is preserved for raw providers (mem/memcache/
  postgres): `err == target` holds through the helper.
- Valkey's `initErr` guard runs *before* the setter on a miss and depends on
  provider state — it is structurally inexpressible as a `get`/`set` closure.
  The guard must remain in the provider's own GetOrSet.

## Results

VALIDATED — with one required design refinement to the 005 plan:

- The shared helper is compatible with **all 5 providers**, but providers that
  prefix setter errors must wrap the helper's result in their thin GetOrSet
  (`if err != nil { return zero, fmt.Errorf("cache/redis: %w", err) }`). This is a
  one-liner — the dedup logic itself stays shared.
- Valkey keeps its `initErr` guard in its own GetOrSet, before delegating.
- A cleaner alternative: give the helper an optional `wrapErr func(error) error`
  option; providers pass it only when needed. Either way, **do not hardcode**
  prefixes inside the shared helper (providers have different prefixes).
- Net: migration is safe; error semantics are preserved with minimal per-provider
  glue. No test currently asserts the prefixes, so existing tests keep passing.
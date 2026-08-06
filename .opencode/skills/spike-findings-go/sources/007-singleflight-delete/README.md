---
spike: 007
name: delete-vs-inflight
type: standard
validates: "Given a Delete on a key whose setter is still in-flight under singleflight, when the leader finishes, then the deleted key is not resurrected — and group.Forget is sufficient"
verdict: PARTIAL
related: [002, 004]
tags: [singleflight, cache, delete, forget, resurrection]
---

# Spike 007: delete-vs-inflight

## What This Validates

In the `miss -> sf.Do(key, compute+Set)` pattern, a concurrent `Delete(key)`
during the setter's flight is intended to remove the key. Does the in-flight
leader's late `Set` resurrect it? Is `group.Forget(key)` enough to prevent that?

## How to Run

```bash
cd .planning/spikes/007-singleflight-delete && go run .
```

## What to Expect

```
[A-bare-Delete]      stored_after="computed" resurrected=true  setterRuns=1
  => RESURRECTED: leader's late Set() overwrote the Delete.
[B-Delete+Forget]    stored_after="computed" resurrected=true  setterRuns=1
  => RESURRECTED: leader's late Set() overwrote the Delete. (Forget did NOT help)
[C-Delete+gen-tombstone] stored_after="" resurrected=false     setterRuns=2
  => PREVENTED reliably: tombstone checked inside Do fn suppresses the late Set
```

## Investigation Trail

- Case A confirmed the basic hazard: Delete lands at 8ms, leader's Set lands at
  40ms → the key is re-populated after being deleted.
- Case B was the surprise: **`group.Forget(key)` does NOT cancel or prevent the
  in-flight leader from writing.** Forget only removes the key from the group's
  internal map so future `Do` calls start a new computation. The running `fn`
  is never interrupted and completes its Set. Misread of the singleflight docs.
- Case C: the reliable fix is a deletion-generation tombstone (a monotonic
  counter bumped by Delete) that the `Do` fn re-checks *immediately before* Set.
  If the generation changed since the computation started, skip the Set and
  return `cache.ErrMiss`.
- Note: the "not resurrected" outcome for A/B is timing luck (Delete landing
  after the leader's Set) — never guaranteed under contention.

## Results

PARTIAL. The proposed `Delete + group.Forget` pairing is **insufficient** and
must not be assumed. The verified prevention is the tombstone/generation check
inside the singleflight fn before Set.

Design implication for the real build:

- The shared helper gains an optional `isInvalidated func() bool` (or a
  `Set`-guard closure) that providers may wire to their Delete/expiry logic.
- Providers with a natural version/epoch (e.g. a cache-wide generation counter,
  or postgres `updated_at`) can wire it for free. Others can omit the guard —
  resurrection then stays possible, so the tradeoff must be documented.
- `group.Forget` is still worth calling from `Delete` (frees the key for the
  next miss), but it is a **necessary-not-sufficient** part of the fix.
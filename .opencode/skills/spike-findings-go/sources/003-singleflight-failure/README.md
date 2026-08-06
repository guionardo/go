---
spike: 003
name: singleflight-failure
type: standard
validates: "Given a failing setter or a caller whose context is canceled, when wrapped in singleflight, then the error is shared to all waiters, nothing is cached, and cancellation is honored"
verdict: VALIDATED
related: [002, 004, 005]
tags: [singleflight, concurrency, cache, error, context]
---

# Spike 003: singleflight-failure

## What This Validates

If the setter returns an error, every concurrent waiter must receive that same
error (so nothing is cached). A panicking setter must not corrupt the group. A
caller whose context is canceled must be able to abandon its wait without
blocking the leader.

## How to Run

```bash
cd .planning/spikes/003-singleflight-failure && go run .
```

## What to Expect

```
[A] setter error shared to all 10 callers: true (err identity preserved: true)
CASE A: PASS — error propagated to all waiters as the SAME error value
CASE B: PASS — panic replayed to first caller: setter-panic
[C] canceled follower abandoned at 5ms (leader still running)
[C] leader completed independently at 90ms — not aborted by follower cancel
CASE C: PASS — singleflight is context-blind
```

## Investigation Trail

- Case A confirmed out of the box: all waiters receive `c.val, c.err` — pointer
  identity preserved, so errors must be treated as read-only (don't wrap/mutate).
- Case B confirmed: a panicking setter is captured as `*panicError` and the panic
  is **replayed to every caller**, not just the leader. The group does NOT leak
  or deadlock.
- Case C initially had my logic backwards (panic "context should NOT abort").
  Reality: `Group` never sees the caller's `context`. A blocked `Do()` cannot be
  canceled. The only way for a canceled follower to leave early is to select on
  its own ctx against `DoChan`.

## Results

VALIDATED. Key behavior for the real build:

- **Blocker:** `Do()` is context-blind — a caller whose ctx dies can sit blocked
  until the slow setter finishes. For a cache, this can pile up goroutines on a
  hanging setter. If per-caller cancellation is desired, the wrapper must use
  `DoChan` + `select` on the caller's ctx. There is no built-in abort of the
  leader from a follower's cancel.
- **Panic replay:** if a setter panics, each in-flight caller panics too. A robust
  wrapper should recover inside the setter (return error) rather than letting
  singleflight fan out panics.
- **Error identity:** the exact error object is shared with all waiters. Callers
  must not assume ownership (no mutation, e.g. wrapping with context changes
  identity will make equality checks against a sentinel fail if done naively).
# Phase 5: Shared singleflight helper - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-06
**Phase:** 5-shared-singleflight-helper
**Areas discussed:** Helper visibility & placement, Setter ctx propagation, Canceled-waiter semantics, Helper API shape, Panic policy, TTL ownership boundary, Public helper testing

---

## Helper Visibility & Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Export from cache | Export SingleflightGetOrSet[K,V] from the cache package — providers import it directly; becomes public API users could adopt | ✓ |
| cache/internal package | Put it in cache/internal/singleflight — kept out of public API surface | |
| Unexported, per-provider | Keep unexported and duplicate per provider — violates shared-helper goal | |

**User's choice:** Export from cache
**Notes:** Providers live in subpackages so unexported helpers are unreachable; public API was accepted.

---

## Setter ctx Propagation

| Option | Description | Selected |
|--------|-------------|----------|
| No ctx on setter | Keep setter as func() (V,error) — matches existing interface, zero churn | |
| ctx-aware setter | Change to func(context.Context) (V,error) — breaks public Cache interface | ✓ |

**User's choice:** ctx-aware setter (breaking change)
**Notes:** Follow-ups locked: setter receives leader's ctx; waiters' ctx never reach the setter; leader cancel propagates to setter via ctx.Err(); single explicit ctx for waiter-select + setter; breaking change accepted in v1.6.

---

## Canceled-waiter Semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Value if ready, else ctx.Err | Return shared value if leader finished (non-blocking select), else cancellation | ✓ |
| Always ctx.Err on cancel | Strict cancel semantics even if value arrived simultaneously | |

**User's choice:** Value if ready, else ctx.Err
**Notes:** Sentinel `cache.ErrCanceled` wraps the underlying `ctx.Err()` so errors.Is matches both.

---

## Helper API Shape

| Question | Decision | Selected |
|----------|----------|----------|
| Structure | Embedded `SingleflightGetOrSet[K,V]` struct + Do method (persists Group for cross-call dedup) | ✓ |
| Methods | Both `Do` (blocking) and `DoChan` (cancel-aware), identical signatures returning (V, error) | ✓ |
| Group key | `fmt.Sprint(key)` (validated in spike 005) | ✓ |
| Error handling | Helper returns raw errors; provider prefix decoration deferred to Phase 6 | ✓ |

**User's choice:** Embedded struct + Do/DoChan + fmt.Sprint + raw errors

---

## Panic Policy

| Option | Description | Selected |
|--------|-------------|----------|
| Always return error | Recover in wrapper, return Go error to every waiter — deterministic | ✓ |
| Re-panic to leader | Return error to waiters but re-panic leader — inconsistent, callers can't identify leader | |

**User's choice:** Always return error
**Notes:** Typed `cache.Panic` error wrapping recovered value + stack, inspectable via errors.As.

---

## TTL Ownership Boundary

| Option | Description | Selected |
|--------|-------------|----------|
| TTL passthrough | Helper passes ttl... to set closure; provider Set resolves defaults | ✓ |
| Helper resolves TTL | Helper duplicates provider-default logic into shared code | |

**User's choice:** TTL passthrough
**Notes:** Leader's TTL wins when waiters pass different ttls (only leader calls Set) — document.

---

## Public

### Helper Testing

| Option | Description | Selected |
|--------|-------------|----------|
| Full public test suite | All 9 validated spike behaviors + race detector + godoc Examples | ✓ |
| Minimal | Only what providers exercise in Phase 6 | |
| In-package | In-package stub get/set closures — no coupling to providers | ✓ |

**User's choice:** Full public test suite with in-package stub

---

## the agent's Discretion

- None — every surfaced gray area was decided by the user. Implementation-level details (internal fn naming, test structure) left to planner/executor.

## Deferred Ideas

- Todos for config HTTP endpoint, httptest_mock ServeMux routing, and Windows header-matching were reviewed and NOT folded (non-cache scope). Remain pending in `.planning/todos/pending/`.
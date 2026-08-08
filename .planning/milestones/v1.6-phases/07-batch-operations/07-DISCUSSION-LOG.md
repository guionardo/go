# Phase 7: Batch operations - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-07
**Phase:** 7-Batch operations
**Areas discussed:** Interface architecture, MGet return type, MSet TTL policy, Error handling

---

## Interface Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Add to Cache interface | Breaking for external implementors; simplest API | |
| Add to cacher + expose via concreteCache | Backward-compatible; internal interface with provider overrides | ✓ |
| Separate Batch adapter type | Non-breaking; extra import/discovery friction | |

**User's choice:** Approach B — cacher + concreteCache
**Notes:** Follows the same pattern established in Phase 5/6 for singleflight. Backward compatibility prioritized.

---

## MGet Return Type

| Option | Description | Selected |
|--------|-------------|----------|
| map[K]V only | Simple: only found keys. Missing = absent. | ✓ |
| Result struct per key | map[K]BatchResult[V] with Value + Error | |
| Split return (map + missing list) | (map[K]V, []K) — found + missing | |

**User's choice:** map[K]V only
**Notes:** Matches BATCH-01. Missing keys are simply absent from the result map. No per-key error propagation.

---

## MSet TTL Policy

| Option | Description | Selected |
|--------|-------------|----------|
| Single TTL for all keys | Follows existing Set variadic TTL pattern | ✓ |
| Per-key TTL via wrapper | More flexible but verbose for callers | |

**User's choice:** Single TTL for all keys
**Notes:** Matches BATCH-02 proposed signature. Simpler API for the common case.

---

## Error Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Accumulate errors — continue processing | Best-effort: process all keys, join errors at end | ✓ |
| Stop on first error | All-or-nothing: return immediately on first failure | |

**User's choice:** Accumulate errors
**Notes:** Uses errors.Join to aggregate. No data loss from early termination. Consistent with best-effort batch semantics.

---

## Agent's Discretion

None — all areas were discussed and decided.

## Deferred Ideas

None — discussion stayed within phase scope.

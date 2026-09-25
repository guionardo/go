---
phase: 05-shared-singleflight-helper
plan: 01
subsystem: cache
tags: [singleflight, dedup, cache, error-handling, concurrency]

# Dependency graph
requires: []
provides:
  - "cache.SingleflightGetOrSet[K,V] blocking dedup helper (Do + callSetter)"
  - "cache.Panic typed error (Value, Stack, Error(), Unwrap())"
affects: [05-02, 05-03, 05-04, provider_phase6]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "singleflight.Group wraps only the setter + Set; fast-path Get stays outside the group"
    - "deferred-recover inside the group fn converts setter panics into *cache.Panic (never re-panics)"
    - "double-check Get inside the group fn prevents a stale computation from clobbering a concurrent direct Set"
    - "typed panic error mirrors x/sync panicError: Error() + Unwrap() for errors.Is/As traversal, stack line trimmed"

key-files:
  created: [cache/singleflight.go, cache/singleflight_test.go]
  modified: [cache/errors.go, cache/example_test.go]

key-decisions:
  - "D-01/D-02: exported SingleflightGetOrSet[K,V] struct embedding singleflight.Group; providers embed it (state persists per provider)"
  - "D-04/D-06: single explicit ctx; setter receives the leader's ctx"
  - "D-12: singleflight's shared return is discarded — dedup asserted via setter-run count and result identity, never shared==callers-1"
  - "D-13: group key is fmt.Sprint(key)"
  - "D-16/D-17: setter panics recovered inside the group fn into *cache.Panic, delivered to every waiter; never re-panicked"
  - "D-18/D-19: ttl passed verbatim to set closure; leader's ttl wins for concurrent waiters with different TTLs"

patterns-established:
  - "Panic containment: recover inside the singleflight fn, return typed error to all waiters (T-05-01 mitigate)"
  - "Clobber guard: double-check Get inside the fn prevents stale in-flight overwrite of a concurrent direct Set (SF-08)"
  - "Fast-path Get outside the group avoids singleflight lock contention on hits (T-05-03)"

requirements-completed: [SF-02, SF-05, SF-08]

coverage:
  - id: D1
    description: "SingleflightGetOrSet.Do deduplicates concurrent misses — setter runs exactly once, all callers share the value, cached reads don't re-run the setter"
    requirement: SF-02
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_SetterRunsOnce"
        status: pass
    human_judgment: false
  - id: D2
    description: "A panicking setter yields *cache.Panic to every waiter (string + sentinel-error panic values); errors.Is traverses via Unwrap; no panic escapes the Do call"
    requirement: SF-05
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_PanicRecovery"
        status: pass
    human_judgment: false
  - id: D3
    description: "A concurrent direct Set is never clobbered by a stale in-flight computation (double-check Get picks up the fresh value)"
    requirement: SF-08
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_ClobberGuard"
        status: pass
    human_judgment: false
  - id: D4
    description: "Zero-value setter stored and shared across waiters; leader's TTL passed verbatim and leader TTL wins"
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_ZeroValueSetter"
        status: pass
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_TTLPassthrough"
        status: pass
    human_judgment: false
  - id: D5
    description: "Canceled leader's context surfaces the raw context error; helper is race-detector clean (SF-09 core)"
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_LeaderCancel"
        status: pass
      - kind: unit
        ref: "go test -race ./cache/ -count=1"
        status: pass
    human_judgment: false
  - id: D6
    description: "Godoc example ExampleSingleflightGetOrSet_Do runs with deterministic output"
    verification:
      - kind: unit
        ref: "cache/example_test.go#ExampleSingleflightGetOrSet_Do"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 1: SingleflightGetOrSet.Do Shared Helper Summary

**Exported cache.SingleflightGetOrSet[K,V] blocking dedup helper with panic recovery (cache.Panic) and a double-check Get clobber guard, validated by a 6-behavior TDD suite and a godoc example.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-08-06
- **Completed:** 2026-08-06
- **Tasks:** 3 (TDD: RED / GREEN / REFACTOR)
- **Files modified:** 4

## Accomplishments

- `cache.SingleflightGetOrSet[K, V]` with `Do(ctx, key, get, set, setter, ttl...)`: fast-path Get **outside** the singleflight group (cache hits never contend), and inside the group fn a double-check Get (SF-08), `callSetter` panic-recovery boundary (SF-05), and a single in-group `Set` with TTL passthrough (D-18). Setter runs exactly once for N concurrent misses; leader's TTL wins (D-19); raw errors only (D-14).
- `cache.Panic` typed error in `cache/errors.go` — `Value any`, `Stack []byte`, `Error()` formats value + stack (mirroring x/sync `panicError`), `Unwrap()` returns the value when it is an error so `errors.Is`/`errors.As` traverse the recovered value (D-17). Stack's first "goroutine N" line trimmed via the `newPanic` helper.
- `cache/singleflight_test.go` — six Do test functions + subtests (setter-once, panic recovery for string and error panic values, zero-value setter, clobber guard, TTL passthrough with leader-wins, leader cancel with raw-equality assertion — **no** reference to `cache.ErrCanceled`, a 05-02-owned symbol).
- `ExampleSingleflightGetOrSet_Do` godoc example over an in-test stub store with deterministic `// Output:` (D-21/D-22).

- **Gate runs:** `go test -race ./cache/` (clean), `make coverage-quick` (cache pkg 93.5% coverage; file ≥70%, package ≥80%, total ≥75% all satisfied), `go vet ./cache/` (clean), `go build ./...` (clean).

## Task Commits

1. **Task 1: RED — failing tests** - `80bbdae` (test)
2. **Task 2: GREEN — SingleflightGetOrSet.Do + callSetter + Panic** - `0d2a7b7` (feat)
3. **Task 3: REFACTOR — godoc example, race + coverage + vet** - `268e1b8` (refactor)

**Plan metadata:** Plan executes TDD RED/GREEN/REFACTOR sequence with three atomic commits. `docs(05-01)` final metadata commit captures this SUMMARY.

## Files Created/Modified

- `cache/singleflight.go` (created) - `SingleflightGetOrSet[K,V]` struct + `Do` + `callSetter`; the shared helper providers embed and delegate to.
- `cache/errors.go` (modified) - added `type Panic struct` with `Error()`, `Unwrap()`, and the `newPanic` helper (stack trim).
- `cache/singleflight_test.go` (created) - full Do test suite (`TestSingleflightGetOrSet_*`, six functions + subtests).
- `cache/example_test.go` (modified) - added `ExampleSingleflightGetOrSet_Do` godoc example.

## Decisions Made

- Followed the validated spike-005 blueprint (`.opencode/skills/spike-findings-go/references/singleflight-cache.md`) exactly; setter adapted to take `context.Context` per D-04.
- The `shared` return of `group.Do` is intentionally discarded (D-12); dedup is proven via setter-run count and result identity in tests — never `shared == callers-1`.
- Replaced `&Panic{Value: r, Stack: debug.Stack()}` inline with a dedicated `newPanic` helper that trims the first "goroutine N [status]:" line of the stack, matching x/sync's `newPanicError` (plan's "trim the first goroutine line of the stack" note).
- Kept `cache.LeaderCancel` assertion as raw-error equality with `ctx.Err()` and never referenced `cache.ErrCanceled` (05-02-owned symbol — referencing it would break the 05-01 build; defining it early would redeclare in 05-02).

## Deviations from Plan

None - plan executed exactly as written. The plan's prohibition on asserting `shared == callers-1`, on letting setter panics escape, on `Set` outside the group fn, and on hardcoding provider error prefixes were all honored.

### Auto-fixed Issues

_None_ — the implementation followed the validated spike-005 blueprint with no deviation-rule triggers.

---

**Total deviations:** 0 auto-fixed
**Impact on plan:** None. No scope creep.

## Issues Encountered

None - all three TDD gates passed first try on this plan; tests were written, failed compile (RED confirmed), then implemented (GREEN), then raced/vetted (REFACTOR).

## User Setup Required

None - no external service configuration required. Uses only the already-pinned `golang.org/x/sync` v0.22.0 dependency (D-15).

## Next Phase Readiness

- The `SingleflightGetOrSet` span is the foundation for 05-02 (DoChan cancel-aware variant uses the same group and double-check fn; `cache.ErrCanceled` is defined in 05-02) and for provider delegation in Phase 6 (each provider embeds the helper and delegates `GetOrSet`).
- 05-03 already landed its ctx-aware setter line of `Cache` interface and provider signatures; 05-01's helper shares the module build.

## Self-Check: PASSED

- `cache/singleflight.go`, `cache/singleflight_test.go`, `cache/errors.go`, `cache/example_test.go` all present.
- Commits `80bbdae` (RED), `0d2a7b7` (GREEN), `268e1b8` (REFACTOR) all present in git history.
- Gates: `go test -race ./cache/` (0), `go vet ./cache/` (0), `go build ./...` (0), `make coverage-quick` (0).

---

*Phase: 05-shared-singleflight-helper*
*Completed: 2026-08-06*
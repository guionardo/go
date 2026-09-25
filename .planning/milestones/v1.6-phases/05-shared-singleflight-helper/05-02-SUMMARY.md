---
phase: 05-shared-singleflight-helper
plan: 02
subsystem: cache
tags: [singleflight, cache, cancellation, context, error-handling, concurrency]

# Dependency graph
requires:
  - phase: 05-01
    provides: "cache.SingleflightGetOrSet[K,V] Do + callSetter + cache.Panic; the fn body DoChan reuses verbatim (double-check Get → callSetter → Set)"
provides:
  - "cache.SingleflightGetOrSet[K,V].DoChan — cancel-aware variant, identical signature to Do, returns (V, error) only, no Result channel leak (D-12)"
  - "cache.ErrCanceled sentinel wrapping waiter ctx.Err() via double %w so errors.Is matches both ErrCanceled and context.Canceled (D-09/D-10)"
affects: [05-04, provider_phase6, HTTP-facing GetOrSet callers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "DoChan waiter select: non-blocking result-channel read with default FIRST (value-if-ready wins, D-08), then blocking select on ctx.Done() vs result channel"
    - "leaderCh closed by the leader's fn (only the leader's fn executes) discriminates leader from waiter inside the ctx.Done() case; leader blocks on ch and returns the raw setter ctx.Err(), never ErrCanceled (D-04/D-11)"
    - "double %w wrap fmt.Errorf(\"%w: %w\", ErrCanceled, ctx.Err()) so both errors.Is checks hold (D-10)"
    - "waiter contexts never reach the setter — cancellation only gates the waiter's own select (D-04)"

key-files:
  created: []
  modified: [cache/singleflight.go, cache/errors.go, cache/singleflight_test.go, cache/cache_test.go, cache/example_test.go]

key-decisions:
  - "D-08: value-if-ready — non-blocking ch read precedes honoring ctx.Done(); a canceled waiter whose value is already ready gets the value, not ErrCanceled"
  - "D-09/D-10: ErrCanceled wraps the waiter's ctx.Err() with two %w verbs; errors.Is matches BOTH ErrCanceled and context.Canceled"
  - "D-11: leader path never wraps — inside ctx.Done(), first re-read ch non-blocking, then consult leaderCh; a leader blocks on ch and returns the raw fn result (setter ctx.Err())"
  - "D-12: DoChan returns (V, error) only; the singleflight.Result channel never leaks to callers"
  - "D-04: canceling a waiter never cancels the shared computation — only the waiter's own select is gated by its ctx"

patterns-established:
  - "Cancel-aware waiter abandonment: select on ctx.Done() releases waiters at cancel latency instead of blocking for the setter duration (SF-06, validated ~15 ms vs ~300 ms in spike 008)"
  - "Leader vs waiter discrimination via a fn-local closed channel (leaderCh) — the fn runs once so only the leader's signal fires"
  - "Ordered priority in the ctx.Done() case (re-read ch → leaderCh → ErrCanceled wrap) avoids the unordered-select hazard of wrapping a leader's own cancellation (T-05-13 mitigate)"

requirements-completed: [SF-06]

# Coverage metadata — one entry per shipped deliverable (drives UAT routing)
coverage:
  - id: D1
    description: "DoChan canceled-waiter path: a waiter whose ctx is canceled while the leader's setter is blocked abandons its wait promptly and receives an error matching BOTH cache.ErrCanceled and context.Canceled; the leader still completes, stores, and a later read hits it (SF-06, D-04, D-10)"
    requirement: SF-06
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_DoChan_CanceledWaiter"
        status: pass
    human_judgment: false
  - id: D2
    description: "Value-if-ready (D-08): a waiter whose ctx is canceled AFTER the leader's value is stored receives the shared value, not ErrCanceled"
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_DoChan_ValueIfReady"
        status: pass
    human_judgment: false
  - id: D3
    description: "DoChan dedup: N concurrent DoChan callers on the same key run the setter exactly once and all receive the identical value (D-20)"
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_DoChan_SharedValue"
        status: pass
    human_judgment: false
  - id: D4
    description: "Setter error shared with DoChan waiters as the identical error object (spike 003)"
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_DoChan_SetterError"
        status: pass
    human_judgment: false
  - id: D5
    description: "Leader-cancel path never delivers ErrCanceled: the DoChan leader that cancels its own ctx receives the raw context.Canceled from the setter's ctx.Err(), never the waiter wrap (D-11, T-05-13)"
    verification:
      - kind: unit
        ref: "cache/singleflight_test.go#TestSingleflightGetOrSet_DoChan_LeaderCancel"
        status: pass
    human_judgment: false
  - id: D6
    description: "cache.ErrCanceled sentinel identity: Error() contains 'canceled' (err_canceled subtest in TestCacheSentinelErrors)"
    verification:
      - kind: unit
        ref: "cache/cache_test.go#TestCacheSentinelErrors/err_canceled"
        status: pass
    human_judgment: false
  - id: D7
    description: "ExampleSingleflightGetOrSet_DoChan godoc example runs with deterministic // Output (D-22)"
    verification:
      - kind: unit
        ref: "cache/example_test.go#ExampleSingleflightGetOrSet_DoChan"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-08-06
status: complete
---

# Phase 05 Plan 02: SingleflightGetOrSet.DoChan — cancel-aware waiter + ErrCanceled Summary

**DoChan cancel-aware variant of SingleflightGetOrSet[K,V] with the cache.ErrCanceled sentinel: canceled waiters abandon at cancel latency (SF-06) returning an error matching both ErrCanceled and context.Canceled, value-if-ready wins (D-08), and the leader path never wraps its own cancellation (D-11)**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-08-06T14:27:06Z
- **Completed:** 2026-08-06T14:39:00Z (approx)
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- `DoChan` method with identical signature to `Do` (D-12), returning `(V, error)` only — the `singleflight.Result` channel never leaks to callers
- `cache.ErrCanceled` sentinel with a double `%w` wrap so `errors.Is` matches BOTH `cache.ErrCanceled` AND `context.Canceled` (D-09/D-10)
- Ordered priority inside the `ctx.Done()` case: non-blocking ch re-read (value-if-ready, D-08) → `leaderCh` consult (leader returns raw setter ctx.Err(), D-11) → `ErrCanceled` wrap (waiter abandon, D-10) — mitigates the unordered-select hazard (T-05-13)
- Waiter cancellation never disturbs the shared computation: the leader's fn completes and Set lands (D-04), asserted post-release in the canceled-waiter test
- Full test suite: 5 DoChan behavior tests + `err_canceled` sentinel subtest + godoc example; race-detector clean, `go vet` clean, coverage-quick thresholds pass

## Task Commits

Each task was committed atomically following the TDD gate sequence:

1. **Task 1: RED - write failing tests for SingleflightGetOrSet.DoChan** - `513edf4` (test)
2. **Task 2: GREEN - implement DoChan and cache.ErrCanceled** - `3474acc` (feat)
3. **Task 3: REFACTOR - godoc example, race check, coverage gate** - `cf3f387` (refactor)

**Plan metadata:** (docs commit made by orchestrator per plan 05-02 workflow)

## Files Created/Modified
- `cache/singleflight.go` - Added `DoChan` method: fast-path Get outside the group, fn-local `leaderCh` closed by the leader's fn, non-blocking result read (D-08), blocking select on ctx vs result, ordered ctx.Done() case (ch re-read → leaderCh → ErrCanceled wrap)
- `cache/errors.go` - Added `cache.ErrCanceled` sentinel with godoc documenting the double errors.Is contract (D-09/D-10)
- `cache/singleflight_test.go` - Added 5 DoChan tests: CanceledWaiter, ValueIfReady, SharedValue, SetterError, LeaderCancel
- `cache/cache_test.go` - Added `err_canceled` subtest to `TestCacheSentinelErrors`
- `cache/example_test.go` - Added `ExampleSingleflightGetOrSet_DoChan` with deterministic `// Output:`

## Decisions Made
- Followed the plan's ordered-priority ctx.Done() case exactly (ch re-read → leaderCh → ErrCanceled) rather than a plain two-way select — this is the T-05-13 mitigation that keeps the leader's own cancellation raw
- Used a fn-local `leaderCh` closed at the top of the group fn; since only the leader's fn executes, this deterministically discriminates leader from waiter without exposing any channel internals (D-12)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `DoChan` + `ErrCanceled` ready for plan 05-04 (remaining helper work) and Phase 6 (provider wiring: HTTP-facing providers will prefer `DoChan` for their GetOrSet bodies)
- All 05-02 verification gates pass: `go test -race ./cache/ -count=1` (31 tests), `make coverage-quick` (file ≥70%, package ≥80%, total 78.3%), `go vet ./cache/`

---
*Phase: 05-shared-singleflight-helper*
*Completed: 2026-08-06*

## Self-Check: PASSED

- All 5 modified files + SUMMARY.md exist on disk
- Commit hashes verified in git log: `513edf4` (test), `3474acc` (feat), `cf3f387` (refactor)
- `go test -race ./cache/ -count=1`: 31 passed (race clean)
- `go vet ./cache/`: no issues
- `make coverage-quick`: file ≥70% / package ≥80% / total 78.3% — PASS
- No stub patterns found in modified source files
- No new threat surface beyond the plan's threat register (T-05-06/07/08/09/13)


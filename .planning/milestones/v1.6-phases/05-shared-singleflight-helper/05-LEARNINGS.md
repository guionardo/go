---
phase: 05
phase_name: "Shared singleflight helper"
project: "github.com/guionardo/go"
generated: "2026-10-08"
counts:
  decisions: 8
  lessons: 5
  patterns: 7
  surprises: 4
missing_artifacts:
  - "05-UAT.md"
---

# Phase 05 Learnings: Shared singleflight helper

## Decisions

### Exported SingleflightGetOrSet embedded per-provider (D-01/D-02)
`cache.SingleflightGetOrSet[K, V]` is exported from the root cache package as a struct embedding `singleflight.Group`; providers embed it and delegate via its `Do`/`DoChan` methods.

**Rationale:** An unexported helper is unreachable from provider subpackages, and a free-function form would create a fresh `singleflight.Group` per call, losing dedup — embedding persists group state per provider.
**Source:** 05-01-SUMMARY.md

### Breaking ctx-aware setter across the Cache contract (D-03/D-06)
`GetOrSet`'s setter changed from `func() (V, error)` to `func(context.Context) (V, error)` in the `Cache` interface plus all 5 providers, provider tests, and the E2E suite.

**Rationale:** Accepted public break in v1.6 so the setter receives a ctx — the caller's ctx verbatim in providers (D-06), the leader's ctx inside the helper (D-04); one explicit ctx, no separate setterCtx.
**Source:** 05-03-SUMMARY.md

### Group wraps only setter + Set; fast-path Get stays outside (SF-02)
`Do`/`DoChan` call `get` before entering the singleflight group; only the double-check Get, `callSetter`, and `Set` run inside the fn.

**Rationale:** Cache hits must never contend on the singleflight lock (T-05-03) — the group exists to dedup concurrent misses only.
**Source:** 05-01-SUMMARY.md

### Panicking setter → typed `*cache.Panic`, never re-panicked (D-16/D-17)
`callSetter` recovers inside the group fn into `&Panic{Value, Stack}` (stack first line trimmed by `newPanic` in cache/errors.go), delivered identically to every waiter.

**Rationale:** No caller can identify the leader, so every waiter gets an error; `Panic.Error()`/`Unwrap()` mirror x/sync's `panicError` so `errors.Is`/`errors.As` traverse a sentinel panic value.
**Source:** 05-01-SUMMARY.md

### Double-check Get inside the fn — clobber guard (D-07/SF-08)
The group fn re-runs `get(ctx, key)` before invoking the setter; a hit returns immediately and skips the setter.

**Rationale:** A direct `Set` landing while the computation is in-flight must win over the stale computed value; pinned by `TestSingleflightGetOrSet_ClobberGuard` (setter runs zero times, store holds "fresh").
**Source:** 05-01-SUMMARY.md

### DoChan cancel-aware variant with value-if-ready (D-08/D-12)
`DoChan` has the identical signature to `Do`, uses `group.DoChan`, and selects non-blocking on the result channel first, then blocking on ctx vs result; returns `(V, error)` only.

**Rationale:** HTTP-facing callers release at cancel latency (~15 ms vs ~300 ms blocking, spike 008); a canceled waiter whose value is already ready receives the value, not an error; the `singleflight.Result` channel never leaks.
**Source:** 05-02-SUMMARY.md

### ErrCanceled double-%w wrap; leader path never wrapped (D-09/D-10/D-11)
Canceled waiters get `fmt.Errorf("%w: %w", ErrCanceled, ctx.Err())`; a DoChan leader that cancels its own ctx receives the raw setter `ctx.Err()` instead.

**Rationale:** `errors.Is` must match BOTH `cache.ErrCanceled` and `context.Canceled` for waiters, while the leader's own cancellation is the setter's ctx error, not an abandoned wait — pinned by `TestSingleflightGetOrSet_DoChan_LeaderCancel`.
**Source:** 05-02-SUMMARY.md

### TTL passthrough — leader's ttl wins (D-18/D-19)
`ttl...` is passed verbatim to the set closure; the set closure records the ttl it receives and the test asserts the leader's 5s arrives untouched even with mixed waiter TTLs.

**Rationale:** Provider `Set` owns default resolution — no TTL logic in shared code; only the leader calls `Set`, so the leader's ttl is what lands (documented on Do's godoc).
**Source:** 05-01-SUMMARY.md

---

## Lessons

### Wave-1 parallel plans share a module build — transient failures are expected
`go vet -tags e2e ./cache/` failed once in 05-03 with `undefined: cache.SingleflightGetOrSet` because plan 05-01's test file had landed before its implementation; the re-run after 05-01 landed passed.

**Context:** The plan's context note predicted this exact cross-plan condition; wave-1 plans (05-01 helper, 05-03 interface ripple) compile together, so gate failures referencing a sibling plan's in-flight files are re-run, not debugged.
**Source:** 05-03-SUMMARY.md

### Cross-plan symbol ownership can break a build from either direction
05-01 tests could not reference `cache.ErrCanceled` (05-02-owned): referencing it fails 05-01's build with `undefined`, and defining it early would redeclare in 05-02.

**Context:** The leader-cancel assertion used raw-error equality with `ctx.Err()` only — no `errors.Is(err, cache.ErrCanceled)` anywhere in 05-01's files.
**Source:** 05-01-SUMMARY.md

### The git index is untrusted at task start — pre-staged files can be swept into commits
`.github/workflows/opencode.yml`, staged before execution began, was captured by 05-03 Task 1's first commit (`96399da`) alongside the 6 planned files.

**Context:** Fixed with `git reset --soft HEAD~1` + `git restore --staged`, recommitting only plan files as `6ade60f`; the workflow file stayed untracked on disk. Verify commit scope with `git show --stat` after parallel activity.
**Source:** 05-03-SUMMARY.md

### E2E-tagged test files need `-tags e2e` and a package-wide compile check
`cache/cache_e2e_test.go` sits behind the `e2e` build tag, so its closure edits are only compile-verified by `go vet -tags e2e ./cache/`.

**Context:** That vet compiles the whole cache test package, so it cannot typecheck E2E closure edits in isolation while a parallel plan is mid-flight — the failure seen was the shared-build condition, confirmed only by the post-05-01 re-run.
**Source:** 05-03-SUMMARY.md

### A spike-validated blueprint plus locked decisions yields first-try green
All three 05-01 TDD gates passed first try; 05-01 and 05-02 both report "None - plan executed exactly as written" with zero deviations.

**Context:** Implementation followed `.opencode/skills/spike-findings-go/references/singleflight-cache.md` and D-01..D-22; the phase's only auto-fix (05-03) was a commit-scope bug, not a design change.
**Source:** 05-01-SUMMARY.md

---

## Patterns

### Fast-path Get outside the singleflight group
Call `get` before `group.Do`/`group.DoChan` and return immediately on a hit; the group fn then re-checks and computes.

**When to use:** Dedup wrappers over a cache where hits vastly outnumber misses — keeps hits off the singleflight lock (T-05-03), so lock contention never degrades the cached-read path.
**Source:** 05-01-SUMMARY.md

### Panic containment: recover inside the group fn, return a typed error to all waiters
Wrap the setter in `callSetter` with a deferred recover that converts the panic into a typed error; recovery lives inside the singleflight fn so no waiter ever sees a panic.

**When to use:** Running user callbacks inside `group.Do`/`DoChan` (T-05-01) — because x/sync replays a recovered panic to every waiter, conversion must happen before the value crosses the fn boundary.
**Source:** 05-01-SUMMARY.md

### Double-check Get as a clobber guard
Inside the group fn, re-run `get(ctx, key)` right before calling the setter; on a hit, return the fresh value and skip the setter entirely.

**When to use:** Cache GetOrSet helpers where a concurrent direct `Set` can land while a stale computation is in-flight (SF-08); pinned by the ClobberGuard test (setter never runs, store keeps the fresh value).
**Source:** 05-01-SUMMARY.md

### Leader/waiter discrimination via fn-local leaderCh + ordered ctx.Done() priority
The DoChan fn closes a DoChan-local `leaderCh` first; the ctx.Done() case runs in order: non-blocking ch re-read → leaderCh consult → ErrCanceled wrap, so a leader blocks on ch and returns the raw result.

**When to use:** Cancel-aware dedup where a plain two-way select on ctx.Done()/result could wrap the leader's own cancellation; the ordered case is the T-05-13 mitigation and keeps x/sync channel internals unexported (D-12).
**Source:** 05-02-SUMMARY.md

### Double %w wrap for dual errors.Is identity
`fmt.Errorf("%w: %w", ErrCanceled, ctx.Err())` makes one error match both the package sentinel and the context sentinel.

**When to use:** Abandonment/derived errors callers may probe with either a domain sentinel or a stdlib sentinel (D-10); assert both `errors.Is` checks in tests so a wrapping regression can't slip through.
**Source:** 05-02-SUMMARY.md

### Concurrency test harness: WaitGroup fan-out + atomic counter + mutex-guarded stub store
N goroutines call the helper over an in-test map store with `sync.WaitGroup`, an `atomic.Int64` setter-run counter, and a shared results slice; assertions run after `wg.Wait()`.

**When to use:** Proving setter-once dedup and race-clean behavior without provider coupling (D-21) — assert `setterRuns.Load()==1` and result identity, never the singleflight `shared` flag.
**Source:** 05-PATTERNS.md

### Compile-time interface assertions enforce a breaking signature ripple
`var _ cache.Cache[string, any] = (*Cache[string, any])(nil)` in each provider fails the build if its `GetOrSet` signature drifts from the interface.

**When to use:** Interface signature breaks (D-03) — the assertions force the ripple through providers and their tests in the same phase rather than a later one (T-05-11); verified by `go build ./cache/...`.
**Source:** 05-03-SUMMARY.md

---

## Surprises

### x/sync replays recovered panics to every waiter
A setter panic is not only seen by the leader's goroutine — `group.Do`/`DoChan` re-panic the recovered `*panicError` to each waiter, so the wrapper's recovery must sit INSIDE the fn.

**Impact:** Recovery wrapped around `group.Do` would be too late; `callSetter` converts the panic into `*cache.Panic` before any waiter receives it (D-16/D-17).
**Source:** 05-PATTERNS.md

### singleflight's `shared` return is a global flag, not per-caller
The leader itself sees `shared == true` once a follower arrives, so `shared == callers-1` is never a valid dedup assertion.

**Impact:** The wrapper discards `shared` (D-12); dedup is proven by setter-run count and result identity in tests instead.
**Source:** 05-01-SUMMARY.md

### A plain ctx.Done()/result select can wrap the leader's own cancellation
An unordered two-way select on `ctx.Done()` vs the result channel can surface the leader's own cancel as `ErrCanceled` instead of the raw `ctx.Err()` (T-05-13).

**Impact:** DoChan's ctx.Done() case needed ordered handling (ch re-read → leaderCh → wrap) so the leader path never returns `ErrCanceled`, pinned by `TestSingleflightGetOrSet_DoChan_LeaderCancel`.
**Source:** 05-02-SUMMARY.md

### The singleflight fn's once-execution doubles as a leader signal
Because only the leader's fn body runs, closing a DoChan-local `leaderCh` at fn entry deterministically identifies the leader — no need for `Result.Shared` or x/sync internals.

**Impact:** The leader/waiter split inside ctx.Done() is resolved with a private channel, preserving the no-channel-leak contract (D-12).
**Source:** 05-02-SUMMARY.md

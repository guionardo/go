---
phase: 15-provider-foundation-core-cache-semantics
plan: 02
subsystem: cache/database
tags: [go, sqlite, cache, ttl, sweeper, lifecycle, wal, sidecar, reopen]

requires:
  - phase: 15-provider-foundation-core-cache-semantics
    provides: "sqlite provider foundation: open/bootstrap/check/primitives, expiry-filtered SelectSQL, resolveTTL groundwork (15-01)"
provides:
  - "Postgres-parity resolveTTL matrix locked by tests (per-call > default > none; negative/multiple-ttl inputs follow the >0 predicate)"
  - "Absolute UnixNano expiry stored in expires_at; filter-only reads never delete rows; expired reads return ErrMiss"
  - "SweepSQL with the partial-index predicate + best-effort sweep(ctx) that logs and never propagates (D-09)"
  - "Synchronous sweep-on-open before New returns (D-10) reclaiming only expired rows while unexpired rows survive"
  - "WithSweepInterval opt-in ticker (default none, D-08) with startSweeper/sweepLoop, Close cancellation (close(stop) + <-done before db.Close), idempotent Close"
  - "Proven STOR-05 lifecycle: final checkpoint on last-close, -wal/-shm removal, db file + unexpired entries preserved across reopen, expiry across restart"
affects: [15-03, phase-16, phase-17]

actuals:
  tokens: 5193
  tasks: 3
  commits: 3
  plan_head_before: ac8d9f9510977058d0c745537319518bae6a67fe
  plan_head_after: ad1b877b1e6f985894156c7c89ff2b709b0d57ed

tech-stack:
  added: []
  patterns:
    - "open-sequence extension: bootstrap -> journal_mode read-back -> synchronous sweep -> conditional sweeper start"
    - "opt-in background goroutine with stop/done channels; Close cancels and joins before closing the DB"
    - "test-only raw sql.DB observer handle to count rows after New returns (no internals leak in black-box tests)"
    - "require.Eventually on raw row counts via a t-free helper (testify runs conditions on a separate goroutine)"

key-files:
  created:
    - cache/sqlite/sweep.go
  modified:
    - cache/sqlite/schema.go
    - cache/sqlite/sqlite.go
    - cache/sqlite/options.go
    - cache/sqlite/sqlite_test.go
    - cache/sqlite/sqlite_internal_test.go

key-decisions:
  - "Sweeper lifecycle: startSweeper() extracts the channel creation + go sweepLoop() launch so no ctx-carrying function spawns a ctx-less goroutine (contextcheck); channels exist only when interval > 0, so stop/done stay nil for d <= 0 (D-08)"
  - "CloseFunc order is CAS guard -> close(stop) + <-done -> db.Close(); the engine's last connection performs the final checkpoint and removes -wal/-shm, provider code never deletes sidecar files"
  - "keyRowCount (no *testing.T) exists because testify's require.Eventually condition runs on a separate goroutine where t.FailNow is illegal"
  - "TestResolveTTL kept as a single function (15-01 ownership) but restructured onto a named ttlCase/ttlMatrix table to satisfy funlen while extending the matrix in place"
  - "TestDoubleClose gained a concurrent 8-way close subtest to back the must_haves claim that the CAS guard serializes concurrent Close"

patterns-established:
  - "TTL correctness is split by responsibility: writes bind absolute UnixNano; reads filter (never delete); reclamation is open-sweep + opt-in ticker only"
  - "Open-time sweep is best-effort by construction: sweep(ctx) returns nothing, so no caller can couple cache operations to sweep errors"
  - "Sidecar assertions use require.Eventually (Windows-tolerant) and never assert Unix permissions"

requirements-completed: [STOR-05, TTL-01, TTL-02, TTL-03, TTL-04]

coverage:
  - id: D1
    description: "resolveTTL matrix matches the postgres contract: per-call ttl > 0 wins, else defaultTTL > 0, else no expiry; negative and multiple-ttl inputs follow the same predicate with expiry-window assertions"
    requirement: TTL-01
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestResolveTTL (7-row ttlMatrix)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Per-call, default, and no-expiry TTL behavior on live handles: immediate value, then ErrMiss via require.Eventually once the deadline passes; no-TTL entries persist"
    requirement: TTL-01
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestTTLExpiry, #TestDefaultTTLExpiry, #TestNoExpiry"
        status: pass
    human_judgment: false
  - id: D3
    description: "Expired Get returns ErrMiss while a raw COUNT(*) still reports the seeded row — the read path is filter-only (TTL-02)"
    requirement: TTL-02
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestReadsNeverDeleteRows"
        status: pass
    human_judgment: false
  - id: D4
    description: "Synchronous sweep-on-open reclaims expired rows before New returns while unexpired rows survive; sweep failures are logged and swallowed (D-09)"
    requirement: TTL-03
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestSweepOnOpen (raw observer COUNT 0/1), #TestSweepFailureSwallowed"
        status: pass
    human_judgment: false
  - id: D5
    description: "Opt-in periodic sweeper (default none, no goroutine/channels for unset/zero/negative) reclaims post-open expired rows on ticks; option zero-value verified"
    requirement: TTL-04
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestPeriodicSweep, #TestSweeperDisabled, #TestOptions_WithSweepInterval"
        status: pass
    human_judgment: false
  - id: D6
    description: "Close cancels and waits for the sweeper, is idempotent (sequential and 8-way concurrent), and post-Close ops carry the cache/sqlite: prefix with errors.Is(ErrClosed)"
    requirement: STOR-05
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestDoubleClose, #TestClosedErrorPrefix; #TestPeriodicSweep close legs"
        status: pass
    human_judgment: false
  - id: D7
    description: "Durability lifecycle: -wal/-shm gone and db file present after Close; unexpired entries survive reopen with a working roundtrip; TTL elapsed across restart is ErrMiss; Close nil on deferred-error handles"
    requirement: STOR-05
    verification:
      - kind: integration
        ref: "cache/sqlite/sqlite_test.go#TestSidecarRemoval, #TestReopenPersistence, #TestExpiryAcrossRestart, #TestCloseAfterDeferredError"
        status: pass
    human_judgment: false

duration: 18min
completed: 2026-10-08
status: complete
---

# Phase 15 Plan 02: TTL Semantics, Sweeps, and Durability Lifecycle Summary

**Postgres-parity TTL resolution with absolute UnixNano expiry, filter-only reads that never delete rows, synchronous sweep-on-open plus an opt-in ticker with clean Close cancellation, and the proven STOR-05 lifecycle (final checkpoint, sidecar removal, reopen persistence)**

## Performance

- **Duration:** 18 min
- **Started:** 2026-10-08T14:11:59Z
- **Completed:** 2026-10-08T14:30:00Z
- **Tasks:** 3 (1 tracer + 2 TDD)
- **Files modified:** 6 (1 new: `cache/sqlite/sweep.go`; 5 extended)

## Accomplishments

- TTL resolution is locked to the postgres contract by a 7-row matrix (`per-call > default > none`, plus negative and multiple-ttl inputs) and live expiry tests: per-call `TestTTLExpiry`, `TestDefaultTTLExpiry`, and `TestNoExpiry` all prove the boundary behavior without fixed sleeps.
- Reads are filter-only (TTL-02): `TestReadsNeverDeleteRows` seeds an expired row, gets `ErrMiss`, and asserts the raw `COUNT(*)` is still 1 — no delete-on-read anywhere.
- The open sequence now ends with a synchronous best-effort sweep (D-10): `TestSweepOnOpen` seeds expired and unexpired rows, reopens via the public `New`, and proves (through a raw observer handle) that only the expired row is gone before `New` returns. `TestSweepFailureSwallowed` proves sweep errors never propagate (D-09).
- The opt-in periodic sweeper is real and cancel-safe: `WithSweepInterval(10ms)` reclaims a post-open expired row (`require.Eventually`), `d <= 0` leaves `stop`/`done` nil with no goroutine, and `Close` cancels via `close(stop)`, waits on `done`, then closes the DB — idempotent sequentially and under an 8-way concurrent close (`-race` clean).
- STOR-05 lifecycle is proven end-to-end: `TestSidecarRemoval` (db file present, `-wal`/`-shm` gone via `require.Eventually`), `TestReopenPersistence` (no-TTL and 1h-TTL entries survive Close/reopen plus a post-reopen roundtrip), `TestExpiryAcrossRestart` (elapsed TTL is `ErrMiss` after reopen), and `TestCloseAfterDeferredError` (nil cleanup on poisoned handles).
- Every prohibited path is clean: no `os.Remove`/`Shutdown` on sidecars, `sweep()` returns nothing and no caller checks a sweep error, and only `cache/sqlite/*` changed (6 files, frozen root intact).

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): TTL end-to-end — expiry filter, reads-never-delete, synchronous sweep-on-open** — `8474d1e` (feat)
2. **Task 2 (TDD): opt-in periodic sweep with clean Close cancellation** — `696058f` (feat)
3. **Task 3 (TDD): close lifecycle, sidecar removal, and reopen persistence** — `ad1b877` (test)

**Plan metadata:** `plan_head_before: ac8d9f9` → `plan_head_after: ad1b877` (3 commits measured via the plan ledger)

## Files Created/Modified

- `cache/sqlite/sweep.go` — new: `sweep(ctx)` (best-effort, D-09), `startSweeper()`, `sweepLoop()` (ticker + stop/done contract)
- `cache/sqlite/schema.go` — added `SweepSQL` (`expires_at IS NOT NULL AND expires_at <= ?`, partial-index shape)
- `cache/sqlite/sqlite.go` — open() calls `c.sweep(ctx)` after the read-back (D-10) and conditionally `c.startSweeper()` (D-08); `CloseFunc` cancels/waits the sweeper before `db.Close()`; struct gains `sweepInterval`, `stop`, `done`
- `cache/sqlite/options.go` — `Config.SweepInterval` + `WithSweepInterval(d)` with the documented no-sweeper default (deliberate mem/postgres divergence)
- `cache/sqlite/sqlite_internal_test.go` — extended `TestResolveTTL` matrix (single ownership preserved), `TestReadsNeverDeleteRows`, `TestSweepOnOpen`, `TestSweepFailureSwallowed`, `TestPeriodicSweep`, `TestSweeperDisabled`, `TestOptions_WithSweepInterval`, `keyRowCount` helper
- `cache/sqlite/sqlite_test.go` — `TestTTLExpiry`, `TestDefaultTTLExpiry`, `TestNoExpiry`, `TestSidecarRemoval`, `TestReopenPersistence`, `TestExpiryAcrossRestart`, `TestCloseAfterDeferredError`, `TestDoubleClose`, `TestClosedErrorPrefix`

## Decisions Made

- `startSweeper()` exists as its own method: `contextcheck` forbids a ctx-carrying function (`open`) from spawning a ctx-less goroutine; the extraction keeps the `sweepLoop()` shape (background ticks, lifetime owned solely by Close) while staying lint-clean.
- Channels are created only when the interval is positive; `stop`/`done` stay nil otherwise, and `Close` branches on `c.stop != nil` — matching the "no hidden maintenance" D-08 semantics.
- `keyRowCount` deliberately does not take `*testing.T`: testify's `require.Eventually` evaluates its condition on a separate goroutine, where `t.FailNow` (used by `require`) is illegal.
- `TestResolveTTL` was restructured onto a named `ttlCase`/`ttlMatrix` table instead of spawning a second test function — extends 15-01's single-owned test in place while clearing the `funlen` gate.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `TestSweepFailureSwallowed` to clear the file-coverage gate**
- **Found during:** Task 1 (after the first `make coverage-quick`)
- **Issue:** `sweep.go` measured 50% (1/2 statements) — the `logger().Warn` error branch was uncovered, below the 70% file threshold. AGENTS.md blocks commits below threshold and no `cache/sqlite` override may be added.
- **Fix:** Added `TestSweepFailureSwallowed`, which closes the raw DB handle and calls `sweep()` directly: no panic, no error, branch covered.
- **Files modified:** `cache/sqlite/sqlite_internal_test.go`
- **Verification:** `make coverage-quick` PASS (package 89.9% at that commit; 91.6% final)
- **Committed in:** `8474d1e` (Task 1 commit)

**2. [Rule 3 - Blocking] Lint-gate refactors for the new code**
- **Found during:** Tasks 1–3 (`golangci-lint run ./cache/sqlite/...` was exit 1)
- **Issue:** new-code linters flagged `funlen` on the extended `TestResolveTTL`, `wsl_v5` cuddling around `t.Cleanup` calls, and `contextcheck` on `go c.sweepLoop()` inside ctx-carrying `open()`.
- **Fix:** table-driven `ttlMatrix`; blank lines before `t.Cleanup` per repo style; `startSweeper()` extraction.
- **Files modified:** `cache/sqlite/sqlite_internal_test.go`, `cache/sqlite/sqlite_test.go`, `cache/sqlite/sweep.go`, `cache/sqlite/sqlite.go`
- **Verification:** `golangci-lint run ./cache/sqlite/...` → `0 issues.`
- **Committed in:** `8474d1e`, `696058f`, `ad1b877`

**3. [Rule 1 - Bug] Test helper didn't propagate the sweep interval**
- **Found during:** Task 2 (assertion-level RED)
- **Issue:** 15-01's `newInternalProvider` helper only copied `cfg.DefaultTTL`, so the sweeper never started in internal tests and `TestPeriodicSweep` failed on its `stop != nil` assertion even with the correct option.
- **Fix:** the helper now mirrors `New`'s construction (`defaultTTL` + `sweepInterval`).
- **Files modified:** `cache/sqlite/sqlite_internal_test.go`
- **Verification:** `TestPeriodicSweep` + full suite green
- **Committed in:** `696058f` (Task 2 commit)

### Scope-tightening (documented, in-plan)

- `TestDoubleClose` gained a `concurrent_close_is_serialized_by_the_cas_guard` subtest — the plan flagged concurrent-Close serialization as a must-haves assumption to "review during execution"; the subtest (plus `-race`) is the review evidence.

---

**Total deviations:** 3 auto-fixed (1 Rule 1, 2 Rule 3)
**Impact on plan:** All fixes were gate-mandated (coverage, lint) or test-helper corrections; no source-contract change, no scope creep, frozen root untouched.

## Issues Encountered

- **Pre-existing concurrent first-open flake (deferred, DI-15-01):** a `go test -cover` run failed `TestBootstrapIdempotence/concurrent_first_open_all_usable` (15-01's STOR-04 test) with a deferred `SQLITE_BUSY (5)`. Investigation proved it is pre-existing (reproduces 3/25 at `3d92e70` in a throwaway worktree, and with this plan's sweep disabled), and root-caused it by DSN isolation: the per-connection `PRAGMA journal_mode=WAL` can return immediate `SQLITE_BUSY` when several connections race the DELETE→WAL conversion of a fresh file (busy handler not consulted for that lock path; 4/160 pings with the production DSN vs 0/160 without `_journal_mode`). Fixing it means adding retry, which 15-RESEARCH explicitly assigns to Phase 16 (CONC-01/02); per the executor scope boundary it was **logged, not fixed** — full evidence and a recommended Phase 16 action are in `.planning/phases/15-provider-foundation-core-cache-semantics/deferred-items.md`. No delivered code was changed for this; all this plan's gates re-ran green after documentation.
- No other problems; the tracer feedback gate passed its end-to-end `<verify>` re-run before expansion.

## TDD Gate Compliance

Repo constraints (no empty commits; per-commit `make coverage-quick` green; RED ships with its implementation) override the standalone-RED-commit pattern, as in 15-01. In-session RED evidence recorded:

- **Task 1 (tracer):** tests written first; RED observed — `TestSweepOnOpen` failed with `expected: 0, actual: 1` (expired row not reclaimed); 12 other named tests passed (already-working filter behavior). GREEN after `SweepSQL` + `sweep.go` + the open() call. Commit `feat(15-02)`.
- **Task 2:** tests written first; first RED was compile-level (`undefined: WithSweepInterval`, `c.stop`/`c.done` absent); after minimal compile scaffolding the intentional assertion-level RED appeared — `TestPeriodicSweep` failed on `assert.NotNil(c.stop)`/`c.done` and `Condition never satisfied` (row not reclaimed). GREEN after `startSweeper`/`sweepLoop`/Close wiring. Commit `feat(15-02)`.
- **Task 3:** tests written first; they passed immediately because the durability behavior was delivered by tasks 1–2 (the engine's last-close checkpoint) — the plan prescribes a test-only proven-commit (`test(15-02)`), mirroring 15-01's layered design. Commit `test(15-02)`.
- Gate-scope check: all three commits match `^(feat|test)\((0*15)-(0*2)\):` and each was preceded by test-touching work in the same scope.

### Probe-fallback assumptions review (must_haves)

| Assumption | Outcome |
|---|---|
| Concurrent Close / Close racing ops serialized by CAS; no panic or hang | Reviewed: `atomic.Bool.CompareAndSwap` linearizes; `TestDoubleClose` sequential + 8-way concurrent, full suite `-race` clean |
| Negative-vs-zero durations and multiple-ttl inputs beyond the matrix | Resolved: matrix extended in place (negative→default/none, first-arg-wins); `> 0` predicate = postgres parity |
| Repeated expired reads are idempotent | Resolved: filter-only SELECT is side-effect free; `TestReadsNeverDeleteRows` + repeated `Eventually` Gets |
| Expiry uses a per-call bound now; concurrent reads/writes cannot resurrect | Resolved: predicate re-evaluated in SQL per read with `time.Now().UnixNano()`; `TestTTLExpiry` + `-race` suite |
| Sweep timing beyond open is unspecified; open sweep + opt-in ticker are the only reclamation points | Resolved and documented: nothing else calls `sweep()` |
| Interval exactly on a tick boundary is ordinary ticker behavior | Resolved: no special-casing; `d <= 0` disables only |
| Unset (zero) interval means no goroutine at all | Verified: `TestSweeperDisabled` (nil `stop`/`done`), `TestOptions_WithSweepInterval` (zero default) |
| Sweep ticks cannot overlap (single pinned connection) | Structural: one ticker, each `sweep` completes before the next `select` iteration |

## Known Stubs

None introduced by this plan. (The three batch loop placeholders from 15-01 remain intentionally, tracked for Phase 16; this plan did not touch them.)

## Threat Flags

None — no new network endpoints, auth paths, file-access patterns, or schema changes beyond the planned `SweepSQL` predicate. T-15-06 (bound-now expiry), T-15-07 (sweeper lifecycle), and T-15-08 (engine-only sidecar cleanup) are enforced by construction and the tests above; T-15-SC holds (no dependency changes in this plan).

## Next Phase Readiness

- TTL-01..04 and STOR-05 are demonstrably satisfied: package coverage 91.6%, `-race` clean, `golangci-lint` 0 issues, all six plan verification commands green.
- `cache/sqlite` now has the full semantic surface for 15-03: `optimize.go` (Optimizable adapter + `WithAutoCheckpoint`) and docs/example remain, building on the stable `open`/Close ordering delivered here.
- **Watch item for Phase 16:** DI-15-01 (concurrent first-open WAL conversion race) — nondeterministically fails 15-01's `TestBootstrapIdempotence/concurrent_first_open_all_usable` (~5–12% per full-suite run) until the retry policy lands; see `deferred-items.md`.

---

## Self-Check: PASSED

- All 6 `cache/sqlite/*` files present on disk (verified)
- All 3 task commits present in history: `8474d1e`, `696058f`, `ad1b877` (verified)
- All plan verification commands re-run green: `go test ./cache/sqlite/...` (64 tests), `-race` clean, `-cover` 91.6% ≥ 80%, `golangci-lint` 0 issues
- Prohibition checks: no `os.Remove`/`os.Delete`/`Shutdown` in `cache/sqlite/`; `sweep()` returns nothing and no caller checks it
- Frozen-root check: only `cache/sqlite/*` changed since `plan_head_before` (6 files)
- Acceptance greps: `SweepSQL` predicate, `WithSweepInterval`, `defer close(c.done)`, `c.sweep(ctx)`, `c.sweepInterval > 0`, `close(c.stop)` all confirmed

---

*Phase: 15-provider-foundation-core-cache-semantics*
*Completed: 2026-10-08*

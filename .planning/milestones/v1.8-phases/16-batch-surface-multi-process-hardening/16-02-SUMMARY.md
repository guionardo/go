---
phase: 16-batch-surface-multi-process-hardening
plan: 02
subsystem: database
tags: [sqlite, wal, retry, busy-timeout, journal-size-limit, modernc, concurrency]

# Dependency graph
requires:
  - phase: 16-batch-surface-multi-process-hardening
    provides: 16-01 spike evidence (D-11) — confirmed D-06 policy triple (5 ms backoff / 5 s budget / busy-only Code()&0xff==5 at PingContext) and the 30/30 policy-arm proof
  - phase: 15-provider-foundation-core-cache-semantics
    provides: pinned modernc.org/sqlite driver, DSN pragma set, pinned 1-connection pool, deferred initErr taxonomy, DI-15-01 root-cause record
provides:
  - cache/sqlite/retry.go — interface-based busy classifier + bounded retry loop (busyErr, isBusyError, busyTimeout, busyBackoff, retryBusy, retryBusyWithin)
  - open() retry wiring: exactly one retryBusy(ctx, c.db.PingContext) call between pool pinning and bootstrap (DI-15-01 closed)
  - D-08 journal_size_limit: buildDSN 4-arg signature + journalSizeLimitBytes constant riding the file-mode DSN
  - deterministic-first-open regression + behavioral BEGIN-contention boundedness evidence (CONC-02)
affects: [16-03 (batch surface — MGet/MSet/MDel), 16-04 (provider-driven E2E children + crash arm + CI)]

# Actuals (#2632) — pairs with the plan's estimate (55000 tokens / 3 tasks, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 4306
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []            # no new dependencies (go.mod untouched)
  patterns:
  - "Interface-based busy classifier: errors.As against interface{ Code() int } + Code()&0xff, never the concrete driver type (keeps the blank import) and never string matching"
  - "Absolute-deadline retry loop: time.Now().Add(budget) checked after each attempt — total wait bounded regardless of retry count; ctx cancellation via select on ctx.Done()"
  - "DSN/budget mechanical sync: TestBusyTimeoutDSNSync parses busy_timeout=NNNN out of dsnSuffixFile and pins it to busyTimeout.Milliseconds()"
  - "DSN-primed contention sandwich: raw second handle with the production DSN holds BEGIN IMMEDIATE (via _txlock) to block a provider write, then rollback proves recovery"

key-files:
  created:
  - cache/sqlite/retry.go
  - cache/sqlite/retry_internal_test.go
  modified:
  - cache/sqlite/sqlite.go
  - cache/sqlite/dsn.go
  - cache/sqlite/dsn_internal_test.go
  - cache/sqlite/sqlite_internal_test.go

key-decisions:
  - "TDD commit pattern follows the repo constraint carried since v1.7 (no empty commits; per-commit make coverage-quick green) — RED evidence is recorded in-session (assertion-level failures on the target tests) and ships with its implementation in the feat commit; the test-only hardening lands as a test commit"
  - "journal_size_limit ships as a buildDSN parameter + compile-time constant (64 << 20), no new exported option (research OQ1 resolution) — non-positive values append nothing, no sentinel semantics"
  - "The retry covers only connection establishment (PingContext): the single wrapped call sits between pool pinning and bootstrap; bootstrap DDL is never retried (0/240 probe evidence), BEGIN contention is never retried (D-07)"
  - "The classifier's mask comparison uses named constants sqliteBusyCode/sqliteBusyMask over the plan's literal &0xff==5 shape — identical semantics, mnd-clean"

patterns-established:
  - "Seam-locked compile coordination: buildDSN signature change, its call site, and its equality-test owner (TestDSNBuildDSN) land in one commit — the package never sits uncompiled"
  - "Stub-then-real TDD RED inside the repo's constrained pattern: a minimal stub (same identifiers, wrong behavior) turns the RED phase from compile-level into assertion-level (#3770-grade) evidence before the real implementation replaces it"
  - "Lint accommodation precedent: //nolint:funlen on a behavior-matrix test (mem_test.go precedent) and decorder-compliant const ordering (single const block)"

requirements-completed: [CONC-01, CONC-02]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Bounded busy-only open retry (D-06): interface classifier covering primary + extended busy codes, 5 s absolute deadline loop with 5 ms backoff, exactly one call site in open() between pool pinning and bootstrap; non-busy errors fail fast; New stays never-failing via deferred initErr"
    requirement: "CONC-01"
    verification:
      - kind: unit
        ref: "cache/sqlite/retry_internal_test.go#TestIsBusyError + TestRetryBusyWithin + TestBusyTimeoutDSNSync (go test ./cache/sqlite/ -run 'TestIsBusyError|TestRetryBusy|TestBusyTimeoutDSNSync' -v — 14 passed)"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestBootstrapIdempotence/concurrent_first_open_all_usable (go test -count=20 — 40/40 passed, deterministic; probe had ~10% flake raw)"
        status: pass
      - kind: other
        ref: "grep gate: [ \"$(sed 's|//.*||' cache/sqlite/sqlite.go | grep -c 'retryBusy(')\" = \"1\" ] (comment-stripped call-site count)"
        status: pass
    human_judgment: false
  - id: D2
    description: "journal_size_limit 64 MB bound via DSN pragma (D-08): buildDSN 4-arg appends wal_autocheckpoint then journal_size_limit in order (positive-only, strconv-formatted); memory mode omits both; read-back proof — file 67108864, memory -1, 200/67108864 coexisting on one pinned connection"
    requirement: "CONC-02"
    verification:
      - kind: unit
        ref: "cache/sqlite/dsn_internal_test.go#TestDSNBuildDSN + cache/sqlite/sqlite_internal_test.go#TestJournalSizeLimitReadBack (9 passed)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Behavioral CONC-02 proof: BEGIN contention surfaces as a busy-classified cache/sqlite:-wrapped error in under two busy_timeout periods (no provider-side retry of BEGIN), and the identical write succeeds after the blocking transaction rolls back"
    requirement: "CONC-02"
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestBeginContentionBounded (go test -run TestBeginContentionBounded -v -timeout=60s — passed)"
        status: pass
    human_judgment: false

# Metrics
duration: 23min
completed: 2026-10-08
status: complete
# Commit ledger (#3968) — measured, not narrated.
plan_head_before: 14c384df480274283ecdea4a3fdae9645b0244eb
plan_head_after: add32ba69d41973e990a1c7a45116433d2a4a0f4
commits: 3
---

# Phase 16 Plan 02: Bounded Open Retry + journal_size_limit Summary

**DI-15-01 closed in the provider: the confirmed D-06 policy (5 ms backoff / 5 s busy-only budget at the PingContext seam) makes the concurrent first-open regression deterministic (40/40 at -count=20), D-08 bounds WAL growth at 64 MB via the DSN pragma with exact read-back proof, and BEGIN contention is proven bounded and typed with no provider-side retry.**

## Performance

- **Duration:** ~23 min
- **Started:** 2026-10-08T22:12:51Z
- **Completed:** 2026-10-08T22:35:00Z (approx)
- **Tasks:** 3/3
- **Files:** 2 created, 4 modified

## Accomplishments

- `retry.go` (new, pure helpers, no driver import): `busyErr` interface + `isBusyError` classifier (`errors.As`, `Code()&0xff == 5` — primary-code masking covers extended variants SQLITE_BUSY_RECOVERY 261 and SQLITE_BUSY_SNAPSHOT 517) + `busyTimeout`/`busyBackoff` constants + `retryBusy`/`retryBusyWithin` absolute-deadline loop (ctx-cancel short-circuit, exhaustion returns the last busy error).
- `open()` in sqlite.go carries exactly one `retryBusy(ctx, c.db.PingContext)` call between pool pinning and bootstrap; exhaustion closes the handle, nils `c.db`, and rides New's deferred `initErr` — New stays never-failing, non-busy errors fail fast.
- `TestBootstrapIdempotence/concurrent_first_open_all_usable` (`workers=4`, shape unchanged) is now deterministic: 40/40 passes at `-count=20` — DI-15-01's ~5–12% flake eliminated.
- `testBusyTimeoutDSNSync` mechanically pins the retry budget to the DSN: `busy_timeout=5000` parsed out of `dsnSuffixFile` == `busyTimeout.Milliseconds()` == 5000 — no silent drift between retry.go and dsn.go.
- D-08 shipped per the locked decision: `dsnJournalSizeLimitPrefix` + `journalSizeLimitBytes = 64 << 20` constants, `buildDSN` 4-arg signature appending `wal_autocheckpoint` then `journal_size_limit` (positive-only, `strconv.FormatInt` — A6 hygiene), memory mode omitting both; file read-back 67108864, memory -1, coexistence 200/67108864 on one pinned connection; no new option, options.go untouched.
- CONC-02 behavioral evidence: `TestBeginContentionBounded` — a second raw handle (production DSN, `_txlock=immediate`) holding BEGIN IMMEDIATE makes a provider write return a busy-classified `cache/sqlite:`-wrapped error in under two `busy_timeout` periods (no provider retry of BEGIN, D-07); after rollback the identical write succeeds.

## Task Commits

Each task was committed atomically:

1. **Task 1: Bounded open retry in open() — retry.go + wiring (DI-15-01, D-06)** - `b79c572` (feat)
2. **Task 2: journal_size_limit via DSN pragma + read-back (D-08)** - `bc22528` (feat)
3. **Task 3: CONC-02 bounded BEGIN contention + open-only retry structural proof** - `add32ba` (test)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update — docs commit)

## Files Created/Modified

- `cache/sqlite/retry.go` (NEW) — `busyErr`, `isBusyError`, `busyTimeout`, `busyBackoff`, `retryBusy`, `retryBusyWithin`; `sqliteBusyCode`/`sqliteBusyMask` named constants; no modernc.org/sqlite import.
- `cache/sqlite/retry_internal_test.go` (NEW) — `fakeError` classifier matrix (5/261/517/1/plain/wrapped), millisecond-budget loop table (busy-then-nil, 3-busy-then-nil, non-busy immediate, always-busy exhaustion ≥2 calls, ctx-cancel), `TestBusyTimeoutDSNSync`.
- `cache/sqlite/sqlite.go` (MOD) — `open()` retry block (13 lines, single call site, mirrors the bootstrap error block); `buildDSN` call site gains `journalSizeLimitBytes`.
- `cache/sqlite/dsn.go` (MOD) — `dsnJournalSizeLimitPrefix` + `journalSizeLimitBytes` constants (single decorder-compliant const block); 4-arg `buildDSN`.
- `cache/sqlite/dsn_internal_test.go` (MOD) — `TestDSNBuildDSN` extended in place (single owner): journal-only, both-pragmas-in-order, non-positive no-ops, memory omission of both `_pragma` keys.
- `cache/sqlite/sqlite_internal_test.go` (MOD) — `TestJournalSizeLimitReadBack` (file 67108864 / memory -1 / coexistence) + `readJournalSizeLimit` helper; `TestBeginContentionBounded`.

## Decisions Made

- TDD commit pattern under the repo constraint (no empty commits; per-commit coverage gate): RED evidence recorded in-session at assertion level; RED ships with its implementation in the feat commits; the test-only hardening task commits as `test(16-02)`.
- `journal_size_limit` is a `buildDSN` parameter + compile-time constant (no `WithJournalSizeLimit` option) per research Open Question 1 resolution.
- The classifier uses named constants (`sqliteBusyCode = 5`, `sqliteBusyMask = 0xff`) with identical semantics to the plan's literal `&0xff == 5` — keeps golangci-lint at zero issues without an inline nolint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Lint accommodations in the new test file**
- **Found during:** Task 1 (golangci-lint run before commit)
- **Issue:** `errname` flagged `fakeErr` (must be `XxxError`) and `funlen` flagged `TestRetryBusyWithin` (93 lines).
- **Fix:** Renamed the fake type to `fakeError` (13 occurrences); added `//nolint:funlen` on the behavior-matrix test with the repo's mem_test.go precedent justification.
- **Files modified:** cache/sqlite/retry_internal_test.go
- **Verification:** golangci-lint run ./cache/sqlite/... → no issues; all target tests still green.
- **Committed in:** `b79c572` (Task 1 commit)

**2. [Rule 3 - Blocking] decorder: second const declaration rejected**
- **Found during:** Task 2 (golangci-lint run before commit)
- **Issue:** `const journalSizeLimitBytes int64 = 64 << 20` as a separate declaration violates the decorder single-const-block rule.
- **Fix:** Merged it into the existing DSN const block with its doc comment.
- **Files modified:** cache/sqlite/dsn.go
- **Verification:** golangci-lint clean; dsn tests unchanged and green.
- **Committed in:** `bc22528` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 lint accommodations — zero behavioral effect)
**Impact on plan:** Confined to style/test naming; no scope creep, no source-contract change.

## TDD Gate Compliance

Repo constraints (no empty commits; per-commit `make coverage-quick` green — a standalone RED commit cannot pass the AGENTS.md gate) override the standalone-RED-commit pattern, as documented in 15-01/15-02/15-03. In-session RED evidence recorded for both `tdd="true"` tasks:

- **Task 1 (tracer):** `retry_internal_test.go` written first; a transient stub `retry.go` (same identifiers, wrong behavior — `Code() == 0` classifier, single-attempt loop) produced assertion-level RED: `TestIsBusyError` failed 4 cases (`expected: true, actual: false` for 5/261/517/wrapped), `TestRetryBusyWithin` failed 4 cases (busy-then-nil returned the busy error, `context canceled` missing from chain, `isBusyError` false on exhaustion). GREEN replaced the stub with the real deadline loop + `open()` wiring; 14/14 target tests + 40/40 concurrent first-open passed. Commit `feat(16-02)`.
- **Task 2:** `TestDSNBuildDSN` extended to the 4-arg shape and `TestJournalSizeLimitReadBack` added first; a stub 4-arg `buildDSN` ignoring `journalSizeLimit` produced assertion-level RED: `TestJournalSizeLimitReadBack` failed 2 cases (`expected: 67108864, actual: -1`) and the DSN equality case failed. GREEN replaced the append; 9/9 target tests + full package green. Commit `feat(16-02)`.
- **Task 3:** test-only task (no `tdd="true"`): `test(16-02)` commit as the repo pattern prescribes.
- Gate-scope check: commits match `^(feat|test)\((0*16)-(0*2)\):`; the only `test(16-02)` commit lands after the feats by repo design (the plan's File-3 task is inherently test-only).

## Issues Encountered

- None beyond the recorded lint accommodations; the tracer feedback gate re-ran the full `<verify>` end-to-end before expansion (14/14 + 40/40 green).
- Pre-existing, out of scope: `internal_probe/two_process_test.go` (untracked scratch from the spike) fails LSP with `undefined: crashHelper`; not tracked in git, not shipped, not touched by this plan.

## Known Stubs

None introduced by this plan. (The three batch loop placeholders from 15-01 at sqlite.go remain intentionally — they are Plan 16-03's scope, not touched here.)

## Threat Flags

None — the plan's threat-model entries are all enforced: T-16-03 (journal_size_limit value is a compile-time constant / `strconv.FormatInt` of int64 — never a caller string; memory mode omits), T-16-04 (busy-only classifier, absolute 5 s deadline, ctx cancellation, single call site, non-busy fails fast, New still never fails), T-16-05 (primary-code masking + unit matrix over 5/261/517/1/plain/wrapped), T-16-SC (no package installs, go.mod untouched).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 16-03 (batch surface) builds on a deterministic-first-open provider: `cache.NewConcreteCache` batch seams can be exercised without the DI-15-01 flake interfering.
- CONC-01 gate satisfied in-process (40/40) with the cross-process evidence carried by 16-01 (30/30 policy arm); Plan 16-04 rewires the E2E children to the shipped provider, whose `open()` now owns the retry.
- CONC-02 satisfied: contention is bounded and typed at BEGIN with no auto-retry beyond open; Checkpoint/Vacuum untouched (`optimize.go` unmodified).
- No blockers.

---
*Phase: 16-batch-surface-multi-process-hardening*
*Completed: 2026-10-08*

## Self-Check: PASSED

- Files: retry.go, retry_internal_test.go, dsn.go, dsn_internal_test.go, sqlite.go, sqlite_internal_test.go, 16-02-SUMMARY.md all exist on disk.
- Commits: b79c572 (Task 1), bc22528 (Task 2), add32ba (Task 3) all present in git history.
- Plan-range change set (14c384d..HEAD) lists exactly the six planned paths; no production files outside cache/sqlite touched, no coverage override added, go.mod untouched.

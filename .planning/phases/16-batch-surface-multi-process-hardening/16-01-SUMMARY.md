---
phase: 16-batch-surface-multi-process-hardening
plan: 01
subsystem: testing
tags: [sqlite, wal, two-process, retry, e2e, modernc, busy-timeout]

# Dependency graph
requires:
  - phase: 15-provider-foundation-core-cache-semantics
    provides: pinned modernc.org/sqlite driver, DSN pragma set (_busy_timeout=5000/_journal_mode=WAL/_txlock=immediate), schema constants, DI-15-01 root-cause record
provides:
  - e2e-tagged two-process WAL spike harness (TestMain helper-process re-exec, raw + policy arms)
  - primary in-repo evidence that the DI-15-01 first-open race is cross-process real and that the bounded busy-only retry eliminates it
  - confirmed D-06 retry policy triple (5 ms backoff / 5 s budget / busy-only Code()&0xff==5 at the PingContext seam)
  - 16-SPIKE-FINDINGS.md evidence record (D-11)
affects: [16-02 (open() retry implementation), 16-04 (provider-driven children + crash arm + CI)]

# Actuals (#2632) — pairs with the plan's estimate (35000 tokens / 2 tasks, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 4470
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []            # no new dependencies (go.mod untouched; stdlib + pinned driver only)
  patterns:
  - "Helper-process re-exec: TestMain env intercept (SQLITE_E2E_HELPER=1) -> exec.Command(os.Args[0]) children; e2e build tag keeps the harness out of the unit lane"
  - "Bounded busy-only retry at the connection-establishment seam (errors.As + interface{ Code() int }, Code()&0xff == 5, 5 ms backoff, 5 s absolute deadline)"
  - "Raw-vs-policy dual arm: the raw arm reproduces the failure as observational counter-evidence; only the policy arm gates"

key-files:
  created:
  - cache/sqlite/twoprocess_e2e_test.go
  - .planning/phases/16-batch-surface-multi-process-hardening/16-SPIKE-FINDINGS.md
  modified: []

key-decisions:
  - "Policy evidence: 30/30 clean parent runs (5 + 20 + 5) with journal_mode=wal, integrity_check=ok, 301 rows per role — retry is load-bearing, not test-cosmetic (raw arm: 8/10 runs failed with immediate SQLITE_BUSY at the ping)"
  - "Raw-arm failures are observational only — the test never gate-keeps on probabilistic raw failures (D-11); raw arm is selected via SQLITE_E2E_RETRY=0"
  - "Harness children run raw database/sql handles with a DSN/schema mirror; the mirror is deliberately spike-local and eliminated in Plan 16-04 when children rewire to the shipped provider"
  - "No cache/sqlite coverage override and no production-code changes — the e2e build tag keeps coverage metrics byte-identical"

patterns-established:
  - "E2E harness isolation: //go:build e2e + TestMain child intercept; untagged go test -list shows zero TestTwoProcess symbols"
  - "Child diagnostics: stdout/stderr captured in bytes.Buffer; Kill+Wait cleanup registered for every spawned child"

requirements-completed: [CONC-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Two-process WAL spike harness: raw arm reproduces DI-15-01 cross-process; policy arm proves the bounded busy-only retry makes two children share one fresh cache file deterministically clean"
    requirement: "CONC-01"
    verification:
      - kind: e2e
        ref: "cache/sqlite/twoprocess_e2e_test.go#TestTwoProcessSpikeContention (go test -tags=e2e -run TestTwoProcessSpikeContention ./cache/sqlite/ -count=5 / -count=20)"
        status: pass
      - kind: unit
        ref: "go test -list 'TestTwoProcess' ./cache/sqlite/ (untagged: zero results) + go test ./cache/sqlite/ green"
        status: pass
    human_judgment: false
  - id: D2
    description: "16-SPIKE-FINDINGS.md D-11 evidence record: harness shape, raw-vs-policy counts, integrity/row-count evidence, confirmed policy triple, disposition"
    requirement: "CONC-01"
    verification:
      - kind: other
        ref: "grep gate (Raw arm / Policy arm / integrity / 5 ms / D-11) + acceptance extras (PingContext, DI-15-01, 16/30, 30/30, no absolute paths)"
        status: pass
    human_judgment: false

# Metrics
duration: 4min
completed: 2026-10-08
status: complete
# Commit ledger (#3968) — measured, not narrated.
plan_head_before: 3ec60ceb3772c32ccd0f8585cbf7c7968c0135f0
plan_head_after: c311bb803e2915998076107bbe5d4d47014e8e97
commits: 2
---

# Phase 16 Plan 01: Two-Process WAL Spike Summary

**Two-process WAL spike: raw arm reproduces DI-15-01 at 8/10 runs (immediate `SQLITE_BUSY` at PingContext), while the confirmed 5 ms / 5 s busy-only retry policy runs 30/30 clean with `journal_mode=wal`, `integrity_check=ok`, and exactly 301 rows per role.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-10-08T22:05:49Z
- **Completed:** 2026-10-08T22:10:00Z (approx)
- **Tasks:** 2/2
- **Files:** 2 created, 0 modified

## Accomplishments

- `cache/sqlite/twoprocess_e2e_test.go` (e2e build tag): `TestMain` re-exec harness — parent spawns two OS processes against one shared fresh file; each child runs 100 point upsert/read-backs, ten 20-key prepared-upsert batch transactions, and a final role marker; parent asserts exit 0, `journal_mode=wal`, `integrity_check=ok`, and exactly 301 rows per role.
- Raw arm (`SQLITE_E2E_RETRY=0`) reproduces the DI-15-01 first-open race cross-process: 8/10 runs failed, each with exactly one child exiting on `ping: database is locked (5) (SQLITE_BUSY)` — the busy handler bypassed at connection establishment (research probe prior: 16/30).
- Policy arm (default) is deterministically green: 30/30 clean runs, 0 failures; every run passed all four parent assertions (research probe prior: 30/30).
- `16-SPIKE-FINDINGS.md` records the D-11 evidence: harness shape, raw-vs-policy numbers, integrity/row-count evidence, the confirmed policy triple (5 ms backoff / 5 s absolute budget / busy-only `Code()&0xff == 5` at `PingContext`, never at bootstrap DDL), and the disposition to Plan 16-02 (implementation) and Plan 16-04 (provider-driven children + crash arm + CI).
- Unit-lane isolation verified: untagged `go test -list 'TestTwoProcess'` lists zero results; `go test ./cache/sqlite/` green; `make coverage-quick` green (sqlite 92.5%, total 81.5%, all thresholds PASS, no `cache/sqlite` override).

## Task Commits

Each task was committed atomically:

1. **Task 1: Two-process contention spike — raw arm vs bounded-retry policy arm** - `b7ad3fc` (test)
2. **Task 2: 16-SPIKE-FINDINGS.md — D-11 evidence record** - `c311bb8` (docs)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update — docs commit)

## Files Created/Modified

- `cache/sqlite/twoprocess_e2e_test.go` — `//go:build e2e` helper-process harness: `TestMain` child intercept, `helperMain()` raw-handle child (ping + policy retry, mirrored DDL/DSN/SQL, iteration loop), `TestTwoProcessSpikeContention` parent, env constants `SQLITE_E2E_HELPER`/`_DB`/`_ROLE`/`_RETRY`, `iterations = 100`
- `.planning/phases/16-batch-surface-multi-process-hardening/16-SPIKE-FINDINGS.md` — D-11 evidence record (harness shape, raw arm 8/10, policy arm 30/30, confirmed policy, disposition)

## Decisions Made

- The retry policy is confirmed in the exact shape Plan 16-02 will implement: busy-only classifier via `errors.As` against `interface{ Code() int }` with `Code()&0xff == 5`, 5 ms backoff, 5 s absolute deadline, wrapping exactly one `PingContext` before bootstrap.
- The raw arm is observational and never a gate (the test does not pass deterministically without the policy); its failures are captured for the findings record via `SQLITE_E2E_RETRY=0` runs.
- The harness keeps spike-local DSN/schema/SQL mirrors (including the idempotent bootstrap DDL both children race), explicitly eliminated in Plan 16-04 when children rewire to the shipped provider.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added the mirrored schema bootstrap to the harness child**
- **Found during:** Task 1 (first policy-arm verification run)
- **Issue:** The plan's harness steps specified ping + point upserts + batch transactions, but the raw `database/sql` child never created `cache_entries`; every child failed with `point upsert 0: SQL logic error: no such table: cache_entries (1)`, making both arms unusable.
- **Fix:** Added `e2eCreateTableSQL` / `e2eCreateIndexSQL` constants mirroring `schema.go` and an `e2eBootstrap()` helper that runs them inside one `BEGIN IMMEDIATE` transaction (mirroring the provider's `bootstrap()`), called after the ping and before the iteration loop. Both children race the idempotent DDL exactly as the research probe did.
- **Files modified:** `cache/sqlite/twoprocess_e2e_test.go`
- **Verification:** Policy arm 30/30 clean; raw arm 8/10 failures now all reproduce the intended `SQLITE_BUSY` at ping (not a schema error); `gofmt`/`go vet -tags=e2e` clean.
- **Committed in:** `b7ad3fc` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The fix was required for the harness to function at all; it stays inside the planned file and scope (test-only, mirror eliminated in Plan 16-04). No production code touched.

## TDD Gate Compliance

Not applicable: the plan's frontmatter `type` is `execute`, and neither task carried `tdd="true"` (Task 1 is a test-only tracer harness; Task 2 is a docs-only record). No RED/GREEN gate sequence applies; the phase's `TDD_MODE=true` gate fires only for behavior-adding `tdd="true"` tasks with non-test source files.

## Issues Encountered

- First policy-arm verification failed (5/5) due to the missing schema bootstrap — resolved by the Rule 3 fix above before any commit; the recorded policy evidence (30/30) is all post-fix.

## Known Stubs

None — no placeholder values, TODOs, or unwired data sources were introduced.

## Threat Flags

None — no new security-relevant surface beyond the plan's `<threat_model>`: the child intercept activates only on the exact env value `SQLITE_E2E_HELPER=1`, the harness is e2e-build-tagged (absent from production binaries and the unit lane), and every child is Kill+Wait-cleaned with captured stdout/stderr.

## Next Phase Readiness

- Plan 16-02 can implement `retryBusy`/`isBusyError` in `open()` with the confirmed triple; `TestBootstrapIdempotence/concurrent_first_open_all_usable` re-strengthening has primary evidence behind it.
- Plan 16-04 extends this same harness file with provider-driven children, the kill-mid-write crash arm, and the 3-OS CI wiring; the raw DSN/schema/SQL mirrors are the designated seam for that rewiring.
- CONC-01 precondition satisfied; no blockers.

---
*Phase: 16-batch-surface-multi-process-hardening*
*Completed: 2026-10-08*

## Self-Check: PASSED

- Files: `cache/sqlite/twoprocess_e2e_test.go`, `16-SPIKE-FINDINGS.md`, `16-01-SUMMARY.md` all exist on disk.
- Commits: `b7ad3fc` (Task 1), `c311bb8` (Task 2) both present in git history.
- Plan-range change set (`3ec60ce..HEAD`) lists exactly the two planned paths; no production files touched, no coverage override added.

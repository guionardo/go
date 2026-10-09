---
phase: 16-batch-surface-multi-process-hardening
plan: 04
subsystem: testing
tags: [sqlite, wal, two-process, e2e, crash-recovery, ci, concurrency, modernc]

# Dependency graph
requires:
  - phase: 16-batch-surface-multi-process-hardening
    provides: 16-01 two-process spike harness (TestMain re-exec, raw arms) + 16-SPIKE-FINDINGS (D-11 confirmed retry policy)
  - phase: 16-batch-surface-multi-process-hardening
    provides: 16-02 provider-owned bounded busy retry in open() + journal_size_limit DSN (children ride this production path)
  - phase: 16-batch-surface-multi-process-hardening
    provides: 16-03 shipped batch surface (MGet/MSet/MDel) the E2E children exercise
  - phase: 15-provider-foundation-core-cache-semantics
    provides: pinned modernc.org/sqlite driver, WAL DSN pragmas (_busy_timeout=5000/_txlock=immediate), pinned pool, schema constants
provides:
  - provider-driven two-process contention E2E (TestTwoProcessContention) with exact per-role 287-row, integrity_check, WAL, deleted-key and marker assertions
  - kill-mid-write crash-recovery E2E (TestTwoProcessCrashRecovery) — uncommitted row rolled back, survivor exits 0, file reopens clean
  - CI wiring — 'SQLite two-process E2E' on all three OSes plus 'SQLite race detector' gated to non-Windows runners
  - CONC-04 multi-process contract in cache/sqlite/doc.go (same-host sharing, bounded BEGIN contention, bounded open retry, no network/cross-host, canceled-MGet partial results)
affects: [phase-16 verification, gsd-ship, future multi-process phases, SQLite provider consumers needing the multi-process contract]

# Actuals (#2632) — pairs with the plan's estimate (55000 tokens / 3 tasks, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 6548
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []            # no new dependencies (go.mod untouched); stdlib + pinned driver only
  patterns:
  - "Provider-driven E2E: children construct the shipped provider (sqlite.New + WithPath); no raw handle, no retry logic, and no env toggle on the child path — the production open() is what keeps runs deterministic"
  - "Crash-arm synchronization: raw BEGIN IMMEDIATE + uncommitted row + marker file; parent Kill+Wait; reopen proves WAL recovery (uncommitted rolled back, integrity ok)"
  - "Bounded child lifecycle: waitChild/killChild carry explicit timeouts (60 s) and cleanupChildren Kill+Waits every spawned process — no unbounded waits (T-16-10)"
  - "CI harness isolation: the e2e build tag keeps both arms invisible to the untagged unit lane and to coverage metrics (no cache/sqlite override)"

key-files:
  created: []
  modified:
  - cache/sqlite/twoprocess_e2e_test.go
  - .github/workflows/go.yml
  - cache/sqlite/doc.go

key-decisions:
  - "Parent raw verification decodes the provider's JSON-encoded value column (json.Unmarshal) instead of comparing raw TEXT — the provider stores JSON while the old raw-handle spike stored plain strings"
  - "The crash child blocks on a timer loop rather than a bare select {} so the Go runtime's deadlock detector never terminates it on its own before the parent kills it"
  - "Row-count contract pinned to 287 per role: 100 point keys − 14 distinct deletions (MDel targets 0,7,15,22,30,37,45; point Delete targets 0,10,20,30,40,50,60,70,80; 0 and 30 overlap) + 10×20 batch keys + 1 final marker; spot checks 0000/0007 (MDel) and 0050 (point Delete)"
  - "The CI race step is gated to runner.os != 'Windows' (a C toolchain is required; Linux+macOS satisfy CONC-03) with the Windows -race broadening recorded as a reviewer-flagged follow-up in the workflow comment"

patterns-established:
  - "Raw-handle mirror discipline: e2eDSN/DDL/SQL mirrors survive only for the raw verification handle and the crash child, each marked as the test-only mirror of dsnSuffixFile/schema.go"
  - "Timeout-bounded child interactions: every parent→child wait (waitChild, killChild, require.Eventually marker check) has an explicit bound; cleanup Kill+Wait runs unconditionally"

requirements-completed: [CONC-01, CONC-03, CONC-04, CONC-05]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Provider-driven contention arm (D-10A): two OS processes construct sqlite.New+WithPath on one fresh file and run 100 iterations of Set/Get + point Delete-with-absence-check (from the 10th), periodic 20-key MSet+MGet, and chunked MDel, ending with a role marker; both exit 0 and a reopen asserts journal_mode=wal, integrity_check=ok, deleted keys absent and exactly 287 rows per role"
    requirement: "CONC-01"
    verification:
      - kind: e2e
        ref: "cache/sqlite/twoprocess_e2e_test.go#TestTwoProcessContention (go test -tags=e2e -run TestTwoProcessContention ./cache/sqlite/ -v -count=3 -timeout=180s — 3/3 PASS; combined arms -count=5 — 10/10 PASS)"
        status: pass
      - kind: unit
        ref: "go test -list 'TestTwoProcess' ./cache/sqlite/ (untagged: zero TestTwoProcess results — e2e build tag effective)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Kill-mid-write crash-recovery arm (D-10B): crasher holds an open BEGIN IMMEDIATE with an uncommitted crash:uncommitted row and signals via marker file; parent kills it, the provider-driven survivor completes with exit 0, and the reopen asserts integrity_check=ok, the uncommitted row absent, and the survivor's final marker present"
    requirement: "CONC-05"
    verification:
      - kind: e2e
        ref: "cache/sqlite/twoprocess_e2e_test.go#TestTwoProcessCrashRecovery (go test -tags=e2e -run TestTwoProcessCrashRecovery ./cache/sqlite/ -v -count=3 -timeout=180s — 3/3 PASS)"
        status: pass
      - kind: e2e
        ref: "go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s — both arms PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "CI wiring (D-09): 'SQLite two-process E2E' step runs the tagged harness on ubuntu/macos/windows with no OS gate; 'SQLite race detector' step runs go test -race on the non-Windows runners with the C-toolchain rationale comment; the untagged 'Run tests' step is unchanged"
    requirement: "CONC-05"
    verification:
      - kind: other
        ref: "grep gate: exactly one 'SQLite two-process E2E' + one 'SQLite race detector' + runner.os != 'Windows' in .github/workflows/go.yml"
        status: pass
      - kind: unit
        ref: "go test -race ./cache/sqlite/ -count=1 -timeout=180s (local run of the gated step: exit 0, no DATA RACE, no FAIL)"
        status: pass
    human_judgment: false
  - id: D4
    description: "CONC-04 documentation: doc.go Data-and-state paragraph states same-host multi-process sharing is supported (WAL + busy_timeout bound contention at BEGIN), the bounded busy-only open retry, network/cross-host unsupported, and the canceled-MGet partial-results contract (A5)"
    requirement: "CONC-04"
    verification:
      - kind: other
        ref: "grep gate: 'same-host' and 'partial results' present in cache/sqlite/doc.go"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-10-08
status: complete
# Commit ledger (#3968) — measured, not narrated.
plan_head_before: ad1b304c523e1d388f2e35609672ae5d75b6c062
plan_head_after: a717aac65655aa410af8b350a2bbc969513c396e
commits: 3
---

# Phase 16 Plan 04: Provider-Driven Two-Process E2E + CI + CONC-04 Summary

**The milestone headline is now a permanent CI regression: two OS processes drive the shipped `cache/sqlite` provider on one WAL file (contention arm with exact 287-row/integrity assertions), the kill-mid-write crash arm proves WAL recovery rolls back the uncommitted row while the survivor completes, both arms run on all three CI OSes with a non-Windows race step, and doc.go states the CONC-04 multi-process contract.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-10-08T22:57:11Z
- **Completed:** 2026-10-08T23:06:00Z (approx)
- **Tasks:** 3/3
- **Files:** 0 created, 3 modified (363 insertions, 166 deletions)
- **Estimate vs actuals:** plan estimated 55000 tokens (low confidence); realized diff measures 6548 estimate-tokens (chars/4) — the spike (16-01) and research probe had already de-risked the harness shape, so execution stayed lean.

## Accomplishments

- **Contention arm rewired to the shipped provider (D-10A):** `helperMain` constructs `sqlite.New[string, string](sqlite.WithPath(...))` and drives only public API calls; the spike's `SQLITE_E2E_RETRY` toggle, local retry loop, `isBusyError`/`retryPing` helpers, and `e2eBatchUpsert` are gone. The production `open()` retry (16-02) is what keeps the first-open race deterministic — proven by 10/10 consecutive combined-arm runs.
- **Mixed workload per D-10A:** 100 iterations of Set + Get-read-back; point Delete with an absence check (`errors.Is(err, cache.ErrMiss)`) from the 10th iteration on; a 20-key MSet + MGet every 10th; a chunked MDel every 15th; final `role:final = "done"` marker.
- **`TestTwoProcessContention` (renamed) assertions:** both children exit 0; reopen asserts `journal_mode=wal`, `integrity_check=ok`, each role's marker (JSON-decoded), deleted-key spot checks `0000`/`0007` (MDel) + `0050` (point Delete), and exactly **287 rows per role**.
- **Crash arm (D-10B):** `crashMain` opens a raw test-only handle, bootstraps the mirrored schema, holds an open `BEGIN IMMEDIATE` (via `_txlock=immediate`) with an uncommitted `crash:uncommitted` row, writes the `SQLITE_E2E_MARKER` file, then blocks (timer loop — never exits on its own). `TestTwoProcessCrashRecovery` waits the marker (2 s bound), starts the provider-driven survivor, kills the crasher, asserts the killed wait reports a non-zero exit (never a specific code/signal), waits the survivor with exit 0, and reopens to assert `integrity_check=ok`, `crash:uncommitted` absent, `survivor:final` present.
- **CI (D-09):** the test matrix gains `SQLite two-process E2E` (no OS gate — Windows handle semantics get the real proof) and `SQLite race detector` (`if: runner.os != 'Windows'`, C-toolchain rationale comment; Windows -race left as a reviewer-flagged follow-up). The untagged `Run tests` step is untouched — the harness stays invisible to the unit lane.
- **CONC-04 docs:** doc.go's Data-and-state paragraph now states the same-host multi-process support, WAL + 5 s `busy_timeout` contention bounded at BEGIN, the busy-only bounded open retry, network/cross-host unsupported, and the canceled-MGet partial-results note (A5).
- **Phase-closing gates green locally:** both e2e arms (10/10 stress), `go test -race ./cache/sqlite/` clean, `go test ./...` green, `make coverage-quick` PASS at 81.9% (no cache/sqlite override), `golangci-lint run` 0 issues, `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/` OK.

## Task Commits

Each task was committed atomically:

1. **Task 1: Provider-driven contention arm — children exercise the shipped provider** - `ef56c77` (test)
2. **Task 2: Crash-recovery arm — kill mid-write, survivor completes (D-10B)** - `77cda4c` (test)
3. **Task 3: CI steps + CONC-04 docs + phase-closing gates** - `a717aac` (docs)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update — docs commit)

## Files Created/Modified

- `cache/sqlite/twoprocess_e2e_test.go` (MOD) — provider-driven `helperMain`; `runContentionWorkload` + `batchRoundTrip` + `pointKey`/`batchKey`; `crashMain` + `crashEnv`/`markerEnv`/`markerTimeout`; `startChild` (variadic extra env) / `waitChild` / `killChild` / `cleanupChildren` (all timeout-bounded); assertion helpers `assertJournalWAL`/`assertIntegrityOK`/`assertKeyAbsent`; `TestTwoProcessContention` (287-row contract) + `TestTwoProcessCrashRecovery`.
- `.github/workflows/go.yml` (MOD) — `SQLite two-process E2E` (3-OS, `-tags=e2e -count=1`) and `SQLite race detector` (`if: runner.os != 'Windows'`, rationale comment) after the unchanged `Run tests` step.
- `cache/sqlite/doc.go` (MOD) — Data-and-state paragraph extended with the CONC-04 multi-process contract and the canceled-MGet partial-results note.

## Decisions Made

- Raw parent assertions decode the provider's JSON-encoded value column (`json.Unmarshal`) instead of comparing raw TEXT — the old spike stored plain strings via raw handles; the provider stores JSON.
- The crash child blocks on a timer loop (`for { time.Sleep(time.Minute) }`), not a bare `select {}` — the runtime's deadlock detector would kill a bare-select process on its own, violating "never returns on its own".
- `perRoleRows = 287` is a named constant with the full deletion-overlap derivation in its doc comment (MDel ∪ point Delete = 14 distinct, 0 and 30 overlap).
- The race step's Windows gate is documented in a workflow comment as a reviewer-flagged follow-up (research OQ3 resolution), keeping the matrix green where the C toolchain exists.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Parent marker assertion compared raw JSON to a plain string**
- **Found during:** Task 1 (first tagged verification run — all 3 runs failed with `expected: "done", actual: "\"done\""`)
- **Issue:** The old raw-handle spike wrote plain strings, so the parent compared the `value` column directly to `"done"`. The provider JSON-encodes values (`json.Marshal("done")` → `"\"done\""`), so the reopened-file assertion could never match.
- **Fix:** Decode the raw column with `json.Unmarshal` before comparing (added `encoding/json` import); the same pattern is used for the survivor's marker in Task 2.
- **Files modified:** `cache/sqlite/twoprocess_e2e_test.go`
- **Verification:** 3/3 tagged runs PASS after the fix; task verify green before commit.
- **Committed in:** `ef56c77` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug in new test assertions; fixed in-session before any commit)
**Impact on plan:** None — all acceptance criteria met as written; no production code touched.

## TDD Gate Compliance

Not applicable as a gate: the plan frontmatter `type` is `execute`, and none of the three tasks carries `tdd="true"`; `task.is-behavior-adding` reports `false` for the plan (Task 1/2 touch only the e2e test file; Task 3 touches workflow config and package docs). The runtime gate therefore never fires.

In-session RED evidence was still recorded in the repo's constrained pattern: before Task 1's rewrite, `go test -tags=e2e -run 'TestTwoProcessContention'` produced `testing: warning: no tests to run` (no PASS line — the task's verify fails accordingly); the crash arm's target test did not exist before Task 2. GREEN shipped with each task's `test(16-04)` commit; the tracer feedback gate re-ran the full Task-1 verify end-to-end (3/3 PASS + unit-lane isolation) before expanding to Task 2.

## Issues Encountered

- The first doc.go wording used sentence-initial "Same-host", which the plan's case-sensitive verify grep (`grep -q 'same-host'`) rejected; rephrased to "File mode supports same-host multi-process sharing…" so the contract sentence begins with the subject and the grep matches. Caught and fixed before the Task 3 commit.
- One stale LSP diagnostic referenced a removed untracked `internal_probe/` scratch file; it does not exist on disk and nothing in the repo references it (same note as 16-02/16-03).

## Known Stubs

None — no placeholder values, TODOs, or unwired data sources were introduced. All assertions are live and backed by the tagged runs.

## Threat Flags

None beyond the plan's `<threat_model>`. Enforced: T-16-10 (every child killed+waited with explicit timeouts; bounded `require.Eventually` marker check; 180 s per-step timeouts; no permission assertions), T-16-11 (children print operational diagnostics only — no values beyond test fixtures, buffers surfaced only on failure), T-16-SC (no package installs; go.mod untouched).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- CONC-01/03/04/05 evidence is complete: the two-process contention + crash arms are permanent, e2e-tagged regressions; the race step and the 3-OS E2E now run in CI; doc.go carries the CONC-04 contract. Phase 16's four plans are all complete.
- The remaining phase-level step is `/gsd-verify-work` (human verification mode is `end-of-phase`) plus `/gsd-ship` once verified — the milestone branch `gsd/v1.8-sqlite-cache-backend` is ready for PR.
- No blockers. CI-side confirmation of the new matrix steps happens on the first push to the workflow.

---

*Phase: 16-batch-surface-multi-process-hardening*
*Completed: 2026-10-08*

## Self-Check: PASSED

- Files: `cache/sqlite/twoprocess_e2e_test.go`, `.github/workflows/go.yml`, `cache/sqlite/doc.go`, `16-04-SUMMARY.md` all exist on disk.
- Commits: `ef56c77` (Task 1), `77cda4c` (Task 2), `a717aac` (Task 3) all present in git history; plan range `ad1b304..a717aac` lists exactly the three planned paths.
- Gates: e2e arms (10/10 combined stress), race clean, `go test ./...` green, coverage-quick 81.9% PASS (no cache/sqlite override), golangci-lint 0 issues, Windows CGO-free build OK.

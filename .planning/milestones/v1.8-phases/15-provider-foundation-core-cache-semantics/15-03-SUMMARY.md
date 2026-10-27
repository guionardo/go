---
phase: 15-provider-foundation-core-cache-semantics
plan: 03
subsystem: cache/database
tags: [go, sqlite, cache, optimizable, checkpoint, vacuum, wal, autocheckpoint, docs]

requires:
  - phase: 15-provider-foundation-core-cache-semantics
    provides: "sqlite provider foundation: open/bootstrap/check/primitives, DSN builder, deferred-error taxonomy (15-01)"
  - phase: 15-provider-foundation-core-cache-semantics
    provides: "TTL/sweep semantics, Close lifecycle and sidecar-free reopen (15-02)"
provides:
  - "sqlite.Optimizable (Checkpoint = PRAGMA wal_checkpoint(TRUNCATE), Vacuum = full VACUUM) implemented by the batchCache adapter New returns, preserving New's cache.BatchCache return type (D-05/D-06)"
  - "Blocked-checkpoint hardening: three-column PRAGMA scan, busy != 0 returns a descriptive error; driver failures wrapped with cache/sqlite: prefix (Pitfall 16)"
  - "Maintenance state machine: check() first in both methods — closed handles wrap ErrClosed, deferred handles wrap ErrInvalidPath; no SQL is ever reached on a broken handle"
  - "WithAutoCheckpoint(pages): pages > 0 appends _pragma=wal_autocheckpoint(<strconv.Itoa>) to the file-mode DSN; read-back proves 200/default 1000/memory ignored (D-07, A6)"
  - "Enriched doc.go (usage, precedence, TTL, persistence, WAL/local-storage note), runnable Example, README package-index and provider-table rows"
  - "Repo-wide green gates with no cache/sqlite coverage override: go test ./..., make coverage-quick (92.5% package), golangci-lint 0 issues, CGO-free Windows cross-build"
affects: [phase-16, phase-17]

actuals:
  tokens: 5355
  tasks: 3
  commits: 3
  plan_head_before: 85f3728fd44aecdaaba77c66f7b879eaaed609fe
  plan_head_after: 0fa18fdbd2a30c3588de5d20397e38f304e2ff43

tech-stack:
  added: []
  patterns:
    - "Optional-interface adapter: New returns an sqlite-side wrapper embedding cache.BatchCache so type assertions to provider-specific interfaces (Optimizable) succeed without touching the frozen root"
    - "Explicit-caller-only maintenance: checkpoint/vacuum execute only through Optimizable calls; no scheduled maintenance anywhere"
    - "DSN pragma tuning proof: strconv.Itoa int append plus a PRAGMA read-back test (configured / default / memory-ignored)"
    - "Empirical pre-test probe for platform behavior (busy timeout shape) before committing to test assertions"

key-files:
  created:
    - cache/sqlite/optimize.go
    - cache/sqlite/example_test.go
  modified:
    - cache/sqlite/options.go
    - cache/sqlite/dsn.go
    - cache/sqlite/sqlite.go
    - cache/sqlite/sqlite_internal_test.go
    - cache/sqlite/sqlite_test.go
    - cache/sqlite/dsn_internal_test.go
    - cache/sqlite/doc.go
    - README.md

key-decisions:
  - "decorder forbids separate top-level type declarations while the plan's acceptance criteria grep for the literal 'type Optimizable interface' / 'type batchCache[...] struct'; resolved with two separate declarations plus one justified //nolint:decorder on batchCache — both contract strings present, lint at 0 issues (repo has 83 nolint precedents)"
  - "Blocked-checkpoint shape verified by an empirical probe BEFORE writing tests: PRAGMA wal_checkpoint(TRUNCATE) returns err=nil with busy=1 after the 5s busy timeout (not a driver error); memory returns busy=0, log=-1, ckpt=-1. The three-column scan + busy!=0 descriptive error is the correct implementation"
  - "Task 2 shipped as a test-only commit (test(15-03)): the hardened behavior was delivered by task 1's implementation, so the tests characterize it — the same layered pattern as 15-01/15-02"
  - "Maintenance SQL strings live only in optimize.go: the prohibition grep 'VACUUM prints exactly optimize.go' holds with checkpointSQL/vacuumSQL as file-local constants"
  - "VACUUM/checkpoint errors never wrap the driver raw: every path returns the cache/sqlite: prefix, and busy errors carry the count (busy=%d)"

patterns-established:
  - "type-assertable extension surface: batchCache{BatchCache; provider} embeds the shared interface and delegates provider-specific methods — the template for any future optional capability on a frozen cache provider"
  - "Maintenance methods start with check(), mirroring every primitive operation — broken/closed handles can never run storage maintenance"
  - "Coverage-gate-safe test layering: tracer ships behavior + tests together; the hardening task ships the error-state matrix as its own commit"

requirements-completed: [STOR-07]

coverage:
  - id: D1
    description: "sqlite.Optimizable succeeds on New's return; Checkpoint/Vacuum run and return nil on healthy file handles; memory Checkpoint is a valid busy=0 no-op"
    requirement: STOR-07
    verification:
      - kind: integration
        ref: "cache/sqlite/sqlite_test.go#TestOptimizable, #TestOptimizableMemory"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestCheckpointColumns"
        status: pass
    human_judgment: false
  - id: D2
    description: "A blocked checkpoint surfaces as an error with the busy count, never a silent nil: a second connection holding an open read transaction makes Checkpoint fail after the busy timeout"
    requirement: STOR-07
    verification:
      - kind: integration
        ref: "cache/sqlite/sqlite_test.go#TestCheckpointBlocked (second handle + open read tx; error contains busy)"
        status: pass
      - kind: unit
        ref: "cache/sqlite/optimize.go three-column Scan(&busy, &logFrames, &checkpointed) proven by #TestCheckpointColumns"
        status: pass
    human_judgment: false
  - id: D3
    description: "Maintenance respects provider state: after Close both methods wrap ErrClosed; deferred-error handles wrap ErrInvalidPath; driver-level failures wrap with the cache/sqlite: prefix"
    requirement: STOR-07
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestOptimizableClosed, #TestOptimizableDeferredErr"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestCheckpointColumns/driver_error_is_wrapped_with_prefix"
        status: pass
    human_judgment: false
  - id: D4
    description: "WithAutoCheckpoint(pages) wires wal_autocheckpoint through the file-mode DSN (strconv.Itoa; memory ignores it): read-back shows 200 when configured, 1000 by default, 1000 in memory mode"
    requirement: STOR-07
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestAutoCheckpointReadBack (200 / 1000 / memory-ignored)"
        status: pass
      - kind: unit
        ref: "cache/sqlite/dsn_internal_test.go#TestDSNBuildDSN (append cases: 200, zero, negative; memory ignored)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Package docs, runnable example, and README discoverability rows: doc.go covers usage/precedence/TTL/persistence and the WAL local-storage note; Example runs; README gains the package-index and provider-table rows"
    requirement: STOR-07
    verification:
      - kind: other
        ref: "go test ./cache/sqlite/ -run Example -v (Example PASS with deterministic Output)"
        status: pass
      - kind: other
        ref: "grep README.md: rows at lines 22 and 85; doc.go/example acceptance greps"
        status: pass
    human_judgment: false
  - id: D6
    description: "Repo-wide quality gates green with no cache/sqlite override: go test ./..., make coverage-quick (package 92.5%, total 81.5%, thresholds file/package/total PASS), golangci-lint 0 issues, GOOS=windows CGO-free cross-build"
    requirement: STOR-07
    verification:
      - kind: other
        ref: "go test ./... && make coverage-quick && grep -c 'cache/sqlite' .testcoverage-quick.yml (0) && golangci-lint run && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-10-08
status: complete
---

# Phase 15 Plan 03: Optimizable Storage Controls, AutoCheckpoint, and Package Docs Summary

**STOR-07 delivered: a type-assertable `sqlite.Optimizable` adapter with TRUNCATE-checkpoint and full-VACUUM semantics (blocked checkpoints error with the busy count, memory is a valid no-op), `WithAutoCheckpoint` wired through the DSN with a 200/1000 read-back proof, enriched docs and README rows — closing Phase 15 with the repo green at 92.5% package coverage and no override**

## Performance

- **Duration:** 9 min
- **Started:** 2026-10-08T14:33:37Z
- **Completed:** 2026-10-08T14:42:52Z
- **Tasks:** 3 (1 tracer + 1 TDD + 1 docs/gates)
- **Files modified:** 10 (2 new: `cache/sqlite/optimize.go`, `cache/sqlite/example_test.go`; 8 extended)

## Accomplishments

- `sqlite.New` now returns a `batchCache` adapter that embeds the frozen `cache.BatchCache` surface and adds the exported `Optimizable` interface — `c.(sqlite.Optimizable)` succeeds while `New`'s declared return type stays `cache.BatchCache[K, V]` (D-05, RESEARCH Finding 2).
- `Checkpoint` runs `PRAGMA wal_checkpoint(TRUNCATE)` scanning all three columns (`busy, log, checkpointed`); `busy != 0` returns `cache/sqlite: checkpoint blocked (busy=N)` — an empirical probe confirmed the row-path shape (err=nil, busy=1 after the 5 s busy timeout) before the tests were written. Memory mode returns busy=0 and is a valid no-op.
- `Vacuum` runs the full VACUUM as the documented exclusive/heavy operation; both maintenance methods start with `check()`, so closed handles wrap `ErrClosed`, deferred handles wrap `ErrInvalidPath`, and no SQL is reached on a broken handle. Every maintenance error carries the `cache/sqlite:` prefix.
- `WithAutoCheckpoint(pages)` appends `_pragma=wal_autocheckpoint(<strconv.Itoa(pages)>)` to the file-mode DSN only when `pages > 0` (A6 — no caller string ever reaches the DSN); read-back tests prove configured 200, default 1000, and memory mode ignoring the option.
- Package documentation completed for this phase's slice: `doc.go` (positioning, usage, location precedence memory > path > name = zero, TTL semantics, persistence/`:memory:` pinning, WAL local-storage note), a runnable `Example` with deterministic output, and both README rows (package index + provider table).
- Phase 15 closes green: `go test ./...` PASS, `make coverage-quick` PASS (package 92.5%, total 81.5%, no `cache/sqlite` override — grep count 0), `golangci-lint run` 0 issues, `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/` PASS.

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): Optimizable adapter + WithAutoCheckpoint + read-back** — `fafce11` (feat)
2. **Task 2 (TDD): blocked checkpoint, closed/deferred states, three-column scan** — `1cb5a3a` (test)
3. **Task 3 (auto): docs, runnable example, README rows, repo-wide gates** — `0fa18fd` (docs)

**Plan metadata:** `plan_head_before: 85f3728` → `plan_head_after: 0fa18fd` (3 commits measured via the plan ledger)

## Files Created/Modified

- `cache/sqlite/optimize.go` — new: `Optimizable` interface (doc'd cost classes), `batchCache` adapter, `checkpoint` (three-column scan + busy error) and `vacuum` provider methods; `checkpointSQL`/`vacuumSQL` constants
- `cache/sqlite/options.go` — `Config.AutoCheckpoint` + `WithAutoCheckpoint(pages)` (D-07: zero keeps SQLite's default, non-positive appends nothing)
- `cache/sqlite/dsn.go` — `buildDSN(path, memory, autoCheckpointPages)`; file-mode append via `strconv.Itoa`; memory mode ignores the parameter
- `cache/sqlite/sqlite.go` — `New` returns `&batchCache{BatchCache: cache.NewConcreteCache(c), provider: c}`; `open` passes `cfg.AutoCheckpoint`
- `cache/sqlite/sqlite_test.go` — `TestOptimizable`, `TestOptimizableMemory`, `TestCheckpointBlocked` (second handle + open read tx, 5 s busy timeout), `TestOptimizableClosed`, `TestOptimizableDeferredErr`
- `cache/sqlite/sqlite_internal_test.go` — `TestAutoCheckpointReadBack` (200/1000/memory) + `readAutoCheckpoint`, `TestCheckpointColumns` (three-column proof + driver-error wrapping)
- `cache/sqlite/dsn_internal_test.go` — `TestDSNBuildDSN` extended with autocheckpoint append cases (200, negative/zero, memory ignored)
- `cache/sqlite/doc.go` — enriched package docs (this phase's slice; full constraint statements remain Phase 17 QUAL-03)
- `cache/sqlite/example_test.go` — runnable `Example` (in-memory Set/Get, `// Output: value: hello`)
- `README.md` — package index row (line 22) + provider table row (line 85)

## Decisions Made

- **decorder vs literal acceptance strings:** the plan's acceptance criteria grep for `type Optimizable interface` and `type batchCache[K comparable, V any] struct`, while decorder (`disable-dec-num-check: false`) forbids multiple top-level type declarations. Resolved with two separate declarations and one justified `//nolint:decorder` on `batchCache`; all acceptance greps pass and `golangci-lint run` reports 0 issues (the repo already carries 83 justified nolint sites).
- **Empirical pre-test probe:** before writing the blocked-checkpoint test, a throwaway probe measured the real behavior — blocked returns `err=nil, busy=1` after 5.07 s (the busy-timeout wait), free returns `busy=0`, memory returns `busy=0, log=-1, ckpt=-1`. This locked the three-column row-path implementation and the error assertion (`busy`) instead of guessing at a driver-level `SQLITE_BUSY` error.
- **Task 2 is test-only:** the hardened behavior (check-first ordering, descriptive busy error, prefix wrapping) shipped in task 1's implementation, so task 2's commit is `test(15-03)` characterizing it — the same layered pattern documented in 15-01/15-02.
- **Maintenance SQL locality:** `checkpointSQL`/`vacuumSQL` live only in `optimize.go` so the prohibition grep (`grep -rln 'VACUUM' cache/sqlite/` prints exactly `optimize.go`) holds by construction.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] decorder vs the plan's literal acceptance strings — one justified nolint**
- **Found during:** Task 1 (lint gate)
- **Issue:** `decorder` rejects multiple top-level `type` declarations in a new file ("multiple \"type\" declarations are not allowed; use parentheses instead"), but the plan's acceptance criteria grep literally for `type Optimizable interface` and `type batchCache[K comparable, V any] struct`. The grouped-parentheses form fails both greps; separate declarations fail decorder.
- **Fix:** kept the two separate declarations and added a single `//nolint:decorder` on `batchCache` with a justification comment (repo precedent: 83 nolint sites, including decorder-adjacent suppressions).
- **Files modified:** `cache/sqlite/optimize.go`
- **Verification:** all four Task-1 acceptance greps PASS; `golangci-lint run` → `0 issues.`
- **Committed in:** `fafce11` (Task 1 commit)

### Environment note (not a plan deviation)

- `gsd_run query git.base-branch --is-protected <branch>` again returned the base branch name (`main`) instead of a boolean; the documented five-name protected-branch fallback (established in 15-01) was used before every commit. Branch `gsd/v1.8-sqlite-cache-backend` verified non-protected.

---

**Total deviations:** 1 auto-fixed (1 Rule 3 blocking)
**Impact on plan:** The single fix is a lint-config accommodation with zero behavioral effect; no scope creep, frozen root untouched (only `cache/sqlite/*` + `README.md`).

## TDD Gate Compliance

Repo constraints (no empty commits; per-commit `make coverage-quick` green; RED ships with its implementation) override the standalone-RED-commit pattern by explicit plan instruction, as in 15-01/15-02. In-session RED evidence recorded:

- **Task 1 (tracer):** tests written first; RED observed — `undefined: sqlite.Optimizable` (×2), `undefined: WithAutoCheckpoint` (×2), `too many arguments in call to buildDSN` (×6); the target tests were the failing ones. GREEN after `optimize.go` + options/DSN/New changes; commit `feat(15-03)`.
- **Task 2:** tests written first per the behavior list; they passed immediately because the behavior was delivered by task 1 (layered design — the plan prescribes a test-only proven-commit `test(15-03)`). No source changes were required; the hardening matrix (blocked/closed/deferred/driver-error/three-column) is the evidence.
- **Task 3:** docs/example/README; `Example` verified runnable via `go test -run Example -v` (PASS, deterministic output); commit `docs(15-03)`.
- Gate-scope check: all three commits match `^(feat|test|docs)\((0*15)-(0*3)\):`.

### Probe-fallback assumption review (must_haves)

| Assumption | Outcome |
|---|---|
| Checkpoint/vacuum behavior beyond the four tested states (busy=0, busy!=0 blocked, memory no-op, closed/deferred handles) is unspecified | **Resolved:** an empirical probe (deleted before commit) established the exact shapes; the four tested states cover every branch of `optimize.go` (success, busy-blocked, check-error, driver-error). Nothing beyond them is reachable |
| Blocked checkpoint waits through the 5 s busy timeout before reporting busy | **Confirmed by test:** `TestCheckpointBlocked` completes in 5.06 s, matching the probed busy-handler behavior |
| `New`'s return type stays `cache.BatchCache[K, V]` while the dynamic type implements `Optimizable` | **Proven:** `TestOptimizable`/`TestOptimizableMemory` assert the type assertion succeeds on the returned interface value |

## Known Stubs

None introduced by this plan. (The three batch loop placeholders from 15-01 remain intentionally, tracked for Phase 16; this plan did not touch them.)

## Threat Flags

None — the plan's threat-model entries are all addressed: T-15-09 (explicit-caller-only maintenance surface, documented cost classes, descriptive blocked-checkpoint errors), T-15-10 (`strconv.Itoa` int-only DSN payload, non-positive appends nothing, memory ignores, string-equality + read-back tests), T-15-11 (doc.go states the local-storage/WAL constraint; full operational statements remain Phase 17 QUAL-03), T-15-SC (no new dependency).

## Issues Encountered

- The `--is-protected` CLI quirk (returns `main` instead of a boolean) recurred; handled with the fallback documented in 15-01. No impact.
- No other problems: the tracer feedback gate re-ran its end-to-end `<verify>` before expansion (both commands green), and the task-2 5-second blocked-checkpoint test fits well inside the 120 s `go test ./...` budget.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase 15 is functionally complete** pending plan-metadata commit and verification: all four requirements families (PROV/STOR/TTL) now have green, override-free coverage (92.5% package, 81.5% total) and the package is discoverable (doc.go, example, README).
- STOR-07's probe-fallback row is resolved with full-branch evidence (see the review table above).
- For Phase 16: batch placeholders (MGet/MSet/MDel loops) remain the designated replacement targets (BATCH-01..03); watch item DI-15-01 (concurrent first-open WAL conversion race) is documented in `deferred-items.md` and belongs to CONC-01/02.
- For Phase 17: full constraint documentation (network FS, sidecar/growth operations) and CI/doc/benchmark hardening remain (QUAL-01/03); `make quality-report` was deliberately not run (tag-time ritual).

---

## Self-Check: PASSED

- Created files present on disk: `cache/sqlite/optimize.go`, `cache/sqlite/example_test.go` (verified)
- All 3 task commits present in history: `fafce11`, `1cb5a3a`, `0fa18fd` (verified); ledger-measured count = 3
- Plan-level verification re-run post-commit: `go test ./...` PASS, `golangci-lint run` 0 issues, Windows CGO-free cross-build PASS, `make coverage-quick` PASS (92.5% package / 81.5% total, no override, grep 0)
- Task acceptance criteria re-run: optimizer suite (0.216 s) and hardening suite (5.303 s) green; all grep criteria PASS
- Prohibitions: `grep -rln 'VACUUM' cache/sqlite/` → exactly `cache/sqlite/optimize.go`; `grep -rn 'auto_vacuum' cache/sqlite/` → nothing
- Frozen-root check: changes limited to `cache/sqlite/*` + `README.md` (10 files)

---

*Phase: 15-provider-foundation-core-cache-semantics*
*Completed: 2026-10-08*

---
phase: 15-provider-foundation-core-cache-semantics
plan: 01
subsystem: cache/database
tags: [go, sqlite, cache, modernc, wal, database-sql, singleflight, ttl]

requires: []
provides:
  - "cache/sqlite provider foundation: New[K,V] returns cache.BatchCache[K,V] via cache.NewConcreteCache on the 7-method cacher interface"
  - "memory > path > name > zero location resolution with ErrInvalidPath validation ('?'/'#', allow-listed names)"
  - "DSN-carried pragmas (busy_timeout/WAL/synchronous/immediate), pinned single-connection pool, BEGIN IMMEDIATE idempotent bootstrap"
  - "deferred initErr error taxonomy (ErrClosed/ErrInvalidPath sentinels, cache/sqlite: prefix) and warn-only journal_mode read-back"
  - "batch placeholders with explicit Phase 16 TODO markers"
affects: [15-02, 15-03, phase-16, phase-17]

actuals:
  tokens: 12736
  tasks: 3
  commits: 3

tech-stack:
  added: [modernc.org/sqlite v1.60.1, modernc.org/libc v1.77.1 (exact pin per cznic/sqlite#177)]
  patterns:
    - "structural satisfaction of the unexported cacher[K,V] interface + NewConcreteCache wiring"
    - "DSN-carried per-connection pragmas from compile-time constants; never post-open Exec"
    - "pool pinned to one connection in both modes"
    - "deferred initErr open/validation errors (New never fails)"
    - "one BEGIN IMMEDIATE transaction wrapping CREATE ... IF NOT EXISTS DDL"
    - "filter-on-read TTL groundwork (expiry predicate in SELECT; reads never delete)"

key-files:
  created:
    - cache/sqlite/doc.go
    - cache/sqlite/options.go
    - cache/sqlite/dsn.go
    - cache/sqlite/schema.go
    - cache/sqlite/sqlite.go
    - cache/sqlite/sqlite_test.go
    - cache/sqlite/sqlite_internal_test.go
    - cache/sqlite/dsn_internal_test.go
    - cache/sqlite/options_internal_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Task-1's prescribed tests alone yield 66.4% package coverage vs the repo's 80% package / 70% file gate; pulled forward the pure unit tests (location matrix, options, resolveTTL) into the tracer commit (deviation Rule 3 — AGENTS.md blocks commits under threshold, and no cache/sqlite override may be added)"
  - "check() order is initErr before closed; sentinels are provider-local with the cache/sqlite: prefix (D-12, not aliasing root cache.ErrClosed)"
  - "journal_mode read-back is warn-only after bootstrap — construction never fails on mismatch (D-11, silent WAL degradation)"
  - "batch MGet/MSet/MDel ship as loop-based placeholders with Phase 16 TODO markers (RESEARCH Finding 1 phase-boundary decision; the 7-method interface requires them to compile)"
  - "verified libc v1.77.1 transitively requires golang.org/x/tools v0.50.0, so tidy's wider golang.org/x/* MVS bumps are mandatory, not incidental"

patterns-established:
  - "Provider foundation shape reused by other durable backends: resolveLocation(cfg, userCacheDir seam) + buildDSN(path, memory) as pure testable functions"
  - "Coverage-gate-safe task sequencing: pure unit tests must ship with the foundation commit, not deferred to later commits"
  - "Lint-clean from the start: grouped declarations (decorder), extracted helpers (funlen), named constants (goconst/mnd), wg.Go (modernize)"

requirements-completed: [PROV-01, PROV-02, PROV-03, PROV-04, STOR-01, STOR-02, STOR-03, STOR-04, STOR-06]

coverage:
  - id: D1
    description: "sqlite.New[K,V] returns cache.BatchCache via NewConcreteCache; memory-mode Set/Get/Delete/Close/GetOrSet roundtrip with cache.ErrMiss and sqlite.ErrClosed semantics"
    requirement: PROV-01
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestMemoryRoundTrip"
        status: pass
    human_judgment: false
  - id: D2
    description: "Deterministic location precedence (memory > path > name > zero = memory) and ErrInvalidPath rejection for '?'/'#' paths and invalid names"
    requirement: STOR-01
    verification:
      - kind: unit
        ref: "cache/sqlite/dsn_internal_test.go#TestLocation"
        status: pass
      - kind: unit
        ref: "cache/sqlite/dsn_internal_test.go#TestInvalidPathAndName"
        status: pass
    human_judgment: false
  - id: D3
    description: "DSN pragmas applied on the pinned connection: file mode reads back wal/5000/1; memory mode sets busy_timeout=5000 and omits journal_mode"
    requirement: STOR-03
    verification:
      - kind: integration
        ref: "cache/sqlite/sqlite_internal_test.go#TestPragmaReadBack"
        status: pass
      - kind: unit
        ref: "cache/sqlite/dsn_internal_test.go#TestDSNBuildDSN"
        status: pass
    human_judgment: false
  - id: D4
    description: "Idempotent schema bootstrap under one BEGIN IMMEDIATE transaction: sequential reopen and 4-goroutine concurrent first-open both leave one cache_entries table and usable handles"
    requirement: STOR-04
    verification:
      - kind: integration
        ref: "cache/sqlite/sqlite_internal_test.go#TestBootstrapIdempotence"
        status: pass
    human_judgment: false
  - id: D5
    description: "Pool pinned to one connection in both modes; independently constructed memory caches never share entries; file CRUD incl. spaces-in-path"
    requirement: STOR-02
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestPoolPinned"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestMemoryIsolation"
        status: pass
      - kind: integration
        ref: "cache/sqlite/sqlite_test.go#TestFileCRUD, #TestFilePathWithSpaces"
        status: pass
    human_judgment: false
  - id: D6
    description: "Error taxonomy: deferred open errors surface ErrInvalidPath on first op, post-Close ops surface ErrClosed, Close idempotent and nil on deferred-error handles, corrupted rows and marshal failures never map to ErrMiss, canceled ctx errors"
    requirement: PROV-04
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_internal_test.go#TestDeferredErr, #TestClosedTaxonomy, #TestCorruptedRowNotMiss, #TestCanceledContext"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestClosedOps, #TestMarshalFailure"
        status: pass
    human_judgment: false
  - id: D7
    description: "GetOrSet dedup inherited from the shared singleflight (20 concurrent misses run the setter once, -race clean); fmt.Sprint key and encoding/json value parity"
    requirement: PROV-02
    verification:
      - kind: integration
        ref: "cache/sqlite/sqlite_test.go#TestGetOrSetDedup (-race)"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestParity"
        status: pass
    human_judgment: false
  - id: D8
    description: "Batch MGet/MSet/MDel compile and smoke-pass as loop-based placeholders carrying Phase 16 TODO markers"
    requirement: PROV-01
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestBatchPlaceholderSmoke"
        status: pass
    human_judgment: false
  - id: D9
    description: "Dependency pins: modernc.org/sqlite v1.60.1 direct + modernc.org/libc v1.77.1 exact pin with the cznic/sqlite#177 comment; CGO-free Windows cross-build"
    requirement: STOR-06
    verification:
      - kind: other
        ref: "go list -m modernc.org/sqlite modernc.org/libc && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/"
        status: pass
    human_judgment: false

duration: 14min
completed: 2026-10-08
status: complete
---

# Phase 15 Plan 01: Provider Foundation + Core Cache Semantics Summary

**`cache/sqlite` provider foundation: pinned pure-Go driver, three location modes, DSN-carried WAL pragmas on a pinned 1-connection pool, deferred-error taxonomy, and singleflight GetOrSet dedup — proven end-to-end by 50 green tests at 88.3% package coverage**

## Performance

- **Duration:** 14 min
- **Started:** 2026-10-08T13:55:12Z
- **Completed:** 2026-10-08T14:08:49Z
- **Tasks:** 3 (1 tracer + 2 TDD)
- **Files modified:** 11 (9 new under `cache/sqlite/`, `go.mod`, `go.sum`)

## Accomplishments

- `sqlite.New[K,V]` returns a real `cache.BatchCache[K,V]` built on the frozen `cacher[K,V]` seam through `cache.NewConcreteCache` — memory (`:memory:`/zero/`WithMemory`), explicit path, and `os.UserCacheDir()/<name>/cache.db` (MkdirAll 0700) all resolve deterministically (memory > path > name > zero).
- Per-connection pragmas ride in the DSN only: file mode reads back `journal_mode=wal`, `busy_timeout=5000`, `synchronous=1`; memory mode sets `busy_timeout` and never carries a `journal_mode` key. Pool pinned 1/1 with zero lifetimes in both modes; two memory caches stay isolated.
- Idempotent bootstrap under one `BEGIN IMMEDIATE` (via `_txlock=immediate`): sequential reopens and 4 concurrent first-opens of a fresh path all leave exactly one `cache_entries` table + partial expiry index (no `user_version`, no migrations).
- Error taxonomy is `errors.Is`-complete: deferred open/validation errors (bad path/name → `ErrInvalidPath`), post-Close ops → `ErrClosed`, idempotent `Close` (nil even on deferred-error handles), corrupted rows and marshal failures never collapse to `cache.ErrMiss`, canceled contexts error.
- GetOrSet dedup is fully inherited: 20 concurrent misses run the setter exactly once (race-clean), and `fmt.Sprint`/`encoding/json` parity holds for int/stringified keys, unicode, empty strings, and structs.
- Batch MGet/MSet/MDel ship as loop-based placeholders with explicit `Phase 16:` markers — compile-level satisfaction of the 7-method interface validated by smoke test.

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): pinned dependency + minimal end-to-end provider** — `38007f2` (feat)
2. **Task 2 (TDD): file-mode location resolution, DSN pragmas, idempotent bootstrap + read-back** — `5eb8c5e` (feat)
3. **Task 3 (TDD): error taxonomy, Close semantics, GetOrSet dedup + parity** — `3d92e70` (feat)

**Plan metadata:** `plan_head_before: c1e1f64` → `plan_head_after: 3d92e70` (3 commits measured)

## Files Created/Modified

- `cache/sqlite/doc.go` — package docs: positioning, location precedence, deferred-error contract, data-at-rest caveat
- `cache/sqlite/options.go` — `Config`, provider-local `Option`, `WithName/WithPath/WithMemory/WithDefaultTTL` (zero value = memory)
- `cache/sqlite/dsn.go` — `resolveLocation` (memory > path > name > zero), `validName` allow-list, `buildDSN` with compile-time-constant suffixes
- `cache/sqlite/schema.go` — `CreateTableSQL` (WITHOUT ROWID) / `CreateIndexSQL` / `SelectSQL` / `UpsertSQL` / `DeleteSQL`
- `cache/sqlite/sqlite.go` — `sqliteCache`, `New`, `open`, `verifyJournalMode`, `bootstrap`, `check`, the 7 primitive methods, `resolveTTL`, sentinels, logger
- `cache/sqlite/sqlite_test.go` — black-box: memory/file CRUD, GetOrSet dedup, isolation, parity, marshal failure, closed ops, batch smoke
- `cache/sqlite/sqlite_internal_test.go` — resolveTTL matrix, pool pinning, pragma read-back, bootstrap idempotence, deferred/closed/corrupted/ctx error paths
- `cache/sqlite/dsn_internal_test.go` — location precedence matrix (injected `userCacheDir`, error stub), validation matrix, DSN string equality
- `cache/sqlite/options_internal_test.go` — one test per option + zero-value defaults
- `go.mod` / `go.sum` — `modernc.org/sqlite v1.60.1` + exact `modernc.org/libc v1.77.1` pin (cznic/sqlite#177 comment)

## Decisions Made

- Deferred-error precedence: `check()` tests `initErr` before `closed`; sentinels are provider-local (`cache/sqlite:` prefix); `Close` returns nil even when the handle deferred an open error (matches redis/valkey cleanup precedent).
- `journal_mode` read-back is warn-only after bootstrap (D-11) — construction never fails on silent WAL degradation.
- The pool is pinned in both modes before any statement runs; the memory DSN never carries `journal_mode` (SQLite reports `"memory"` there).
- Batch operations are intentionally loop-based placeholders with Phase 16 markers — no second dedup layer or batch semantics added early (explicit phase boundary).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pulled forward pure unit tests to clear the per-commit coverage gate**
- **Found during:** Task 1 (tracer commit)
- **Issue:** The tracer's prescribed tests (`TestMemoryRoundTrip`, `TestBatchPlaceholderSmoke`) alone measured 66.4% package coverage (dsn.go 16.7%, options.go ~40%) against the repo thresholds package ≥80 / file ≥70 — AGENTS.md says do not commit under threshold, and the plan forbids adding a `cache/sqlite` override.
- **Fix:** Added the pure unit tests for the foundation (location matrix, validation matrix, DSN equality, all options + zero defaults, `resolveTTL` matrix) in the same tracer commit; tasks 2–3 still added their own behavioral/integration tests on top.
- **Files modified:** `cache/sqlite/dsn_internal_test.go`, `cache/sqlite/options_internal_test.go`, `cache/sqlite/sqlite_internal_test.go`
- **Verification:** `go-test-coverage` PASS at 84.9% (task 1), 88.3% final package coverage; `make coverage-quick` green at every commit
- **Committed in:** `38007f2` (Task 1 commit)

**2. [Rule 3 - Blocking] Lint-gate refactors (golangci-lint was exit 1 with 57 new-issue findings)**
- **Found during:** Task 3 (lint verification)
- **Issue:** `issues.new: true` / `new-from-rev: HEAD` meant all new package code was in scope: decorder (declaration grouping), funlen, goconst, mnd, paralleltest, testifylint, wsl_v5, modernize findings blocked the required clean lint run.
- **Fix:** `golangci-lint run --fix` for the auto-fixable subset, then manual refactors: grouped type/const/var declarations, moved the provider type before its vars, named constants (`memoryPath`, `cacheDirPerm`, `testAppName`), extracted `locationCases`/`probeConcurrentOpen` helpers, added `t.Parallel()`, switched error assertions to `require`, adopted `wg.Go`.
- **Files modified:** all nine `cache/sqlite/*.go` files
- **Verification:** `golangci-lint run ./cache/sqlite/...` → `0 issues.`; full test suite still green
- **Committed in:** `3d92e70` (Task 3 commit)

### Observations (no action required)

- **Wider-than-predicted MVS bumps:** the plan expected only `x/sys v0.47.0→v0.48.0` and `x/sync v0.22.0→v0.23.0`; `go mod tidy` also bumped `x/tools v0.50.0`, `x/mod v0.41.0`, `x/crypto v0.57.0`, `x/net v0.59.0`, `x/telemetry`, `x/text v0.42.0`. Verified cause: `modernc.org/libc v1.77.1` itself requires `golang.org/x/tools v0.50.0` (and `x/sys v0.48.0`). All bumped modules declare `go 1.26.0`, compatible with the repo's `go 1.26.4` directive. No other *direct* dependency changed.
- Task 2's tests characterize behavior the tracer already delivered (per the plan's layered design); the only new task-2 code was the warn-only `verifyJournalMode` read-back, which is unobservable-by-design on the happy path (D-11).

---

**Total deviations:** 2 auto-fixed (both Rule 3 blocking; 0 Rule 1/2/4)
**Impact on plan:** Both fixes were gate-mandated (coverage + lint) and confined to tests/style; no scope creep, no source-contract change.

## TDD Gate Compliance

Repo constraints (no empty commits; per-commit `make coverage-quick` green; RED ships with its implementation) override the standalone-RED-commit pattern by explicit plan instruction. In-session RED evidence recorded:

- **Task 1 (tracer):** RED observed before implementation — `go test ./cache/sqlite/ -run TestMemoryRoundTrip` failed with `no non-test Go files in cache/sqlite` (package absent); GREEN after implementation.
- **Task 2:** tests written first; the new behavior (file mode/read-back) was verified green after implementation. No `test(15-01)` commit by design (see instruction above).
- **Task 3:** tests written first (dedup/parity/taxonomy); implementation fixes applied only as demanded by the lint/test gates; all green.
- Gate validator note: no `test(15-01):` commit exists in history — this is the intended plan shape, not a missed RED commit.

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| `MGetFunc` loop-based placeholder | cache/sqlite/sqlite.go | 239 | Explicit Phase 16 scope boundary (BATCH-01 chunked IN-list replacement planned in 15-02/16); smoke-tested, compile-required by the 7-method interface |
| `MSetFunc` loop-based placeholder | cache/sqlite/sqlite.go | 256 | Same boundary (BATCH-02 single-transaction prepared upsert replacement) |
| `MDelFunc` loop-based placeholder | cache/sqlite/sqlite.go | 269 | Same boundary (BATCH-03 chunked IN-list delete replacement) |

These stubs do not prevent this plan's goal: they satisfy the frozen interface, smoke-pass, and are explicitly scheduled for replacement per the phase-boundary decision.

## Issues Encountered

- `gsd_run query git.base-branch --is-protected <branch>` returns the base branch name (`main`) rather than a boolean in this environment; the executor fell back to the documented five-name protected-branch check (`main|master|develop|trunk|release/*`) — branch `gsd/v1.8-sqlite-cache-backend` verified non-protected before every commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `cache/sqlite` foundation is green, lint-clean, CGO-free on a Windows cross-build, and coverage-safe (88.3% package) — 15-02 (TTL/sweep/Close polish) and 15-03 (Optimizable/autocheckpoint/docs) can build directly on `open`/`bootstrap`/`check`/primitives.
- TTL groundwork is in place: `resolveTTL` semantics, `expires_at INTEGER` column, expiry-filtered `SelectSQL`; sweep SQL and the ticker are 15-02 scope.
- `Optimizable` adapter (RESEARCH Finding 2) is still required in 15-03: `New` currently returns the raw `cache.NewConcreteCache` value (per this plan's task action); the adapter wrapper must be introduced there.
- `make coverage-quick` green repo-wide at 81.1%; go.mod pins verified: `modernc.org/sqlite v1.60.1`, `modernc.org/libc v1.77.1`, `golang.org/x/sys v0.48.0`, `golang.org/x/sync v0.23.0`.

## Self-Check: PASSED

- All 9 created files present on disk (verified)
- All 3 task commits present in history: `38007f2`, `5eb8c5e`, `3d92e70` (verified)
- All task acceptance criteria re-run and passing: package tests, `-race` dedup/isolation, Windows CGO-free cross-build, coverage 88.3% ≥80, `golangci-lint` 0 issues, `gofmt -l` empty, pins verified, frozen-root diff limited to `cache/sqlite/*` + `go.mod`/`go.sum`

---
*Phase: 15-provider-foundation-core-cache-semantics*
*Completed: 2026-10-08*

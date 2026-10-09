---
phase: 16-batch-surface-multi-process-hardening
plan: 03
subsystem: database
tags: [sqlite, batch, chunked-in, prepared-upsert, race, tdd, concurrency]

# Dependency graph
requires:
  - phase: 16-batch-surface-multi-process-hardening
    provides: 16-02 hardened open()/DSN behavior (bounded busy retry, journal_size_limit) — the pool the batch code and race test run on
  - phase: 15-provider-foundation-core-cache-semantics
    provides: pinned 1-connection pool, DSN pragma set (_txlock=immediate), frozen cacher[K,V] 7-method primitive, deferred initErr taxonomy
provides:
  - cache/sqlite/batch.go — chunkSize=100, uniqueKeys, chunksOf, inPlaceholders, MGetFunc/MSetFunc/MDelFunc real implementations (all three placeholders gone)
  - cache/sqlite/batch_internal_test.go — unit matrix (dedup, chunk boundaries, placeholder text, decode skip, empty no-ops, one-TTL, no-partial-writes, MDel fail-fast/idempotency)
  - cache/sqlite/schema.go — MGetSelectSQL (cache_key + value, expiry predicate, %s marker list) and MDelSQL
  - cache/sqlite/sqlite_test.go — TestBatchMGet, TestBatchMSetOneTTL, TestBatchMDel, TestBatchSemantics, TestConcurrentBatchOpsRace
affects: [16-04 (provider-driven E2E children + crash arm + CI) — children now consume the shipped batch surface]

# Actuals (#2632) — pairs with the plan's estimate (60000 tokens / 3 tasks, confidence low).
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 7621
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []            # no new dependencies (go.mod untouched, no package installs)
  patterns:
  - "Per-chunk helper extraction: mgetChunk/msetTx keep each function's cyclomatic complexity under cyclop's 10 while the row-handling defer lives in the helper's own scope (sqlclosecheck-compatible close-before-next-chunk on the pinned pool)"
  - "msetPair package-level type + decorder-compliant declaration order (type before const) — extracted from the inline struct when the transaction body moved to msetTx"

key-files:
  created:
  - cache/sqlite/batch.go
  - cache/sqlite/batch_internal_test.go
  modified:
  - cache/sqlite/schema.go
  - cache/sqlite/sqlite.go
  - cache/sqlite/sqlite_test.go

key-decisions:
  - "TDD commit pattern per the repo constraint (no empty commits; per-commit make coverage-quick green): RED evidence recorded in-session at assertion level via transient stubs (16-03-RED-EVIDENCE-task1.json / task2.json, RED_EVIDENCE_OK for both); RED ships with its implementation in the feat commits; the test-only hardening commits as test(16-03)"
  - "TestBatchPlaceholderSmoke removed in Task 1 (its MSet/MDel legs depended on the then-stub batch methods and broke the per-commit suite gate); its MGet contract is covered by TestBatchMGet in Task 1 and the full replacement TestBatchSemantics lands in Task 3 — the acceptance criterion 'placeholder smoke test no longer exists' holds"
  - "The cross-chunk MGet internal test seeds point-wise via SetFunc, not MSetFunc, keeping Task 1 self-contained before the MSet implementation lands"
  - "MSetFunc's transaction body extracted to msetTx purely for cyclop compliance — the single BeginTx/Prepare/Commit seam is unchanged (grep-verified 1/1/1); no per-pair autocommit path exists (zero SetFunc references in batch.go)"

patterns-established:
  - "Stub-then-real RED inside the repo's constrained pattern (16-02 precedent): transient stubs with identical identifiers but wrong behavior turn RED into assertion-level evidence before the real implementation replaces them in the same session"
  - "gomod-independent lint accommodation: //nolint:funlen on behavior-matrix tests (mem_test.go precedent) and //nolint:funlen,gocognit,cyclop on the CONC-03 harness with a justification comment"
  - "TAP-bridged RED evidence: the classifier's parser consumes node-test TAP summaries, so Go RED output is bridged into the record (real failing test names from the actual run; verdict RED_EVIDENCE_OK)"

requirements-completed: [BATCH-01, BATCH-02, BATCH-03, CONC-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Chunked MGet (D-01/D-02/D-03, BATCH-01): dedup-before-chunk order, chunkSize=100 fixed, empty input issues no query, expired rows absent by SQL predicate with the raw row retained (reads never delete), undecodable rows skipped at debug, chunk failures warn+skip with the wrapped error, per-chunk map[string]K re-keying, cross-chunk retrieval at 150 keys"
    requirement: "BATCH-01"
    verification:
      - kind: unit
        ref: "cache/sqlite/batch_internal_test.go#TestMGetFunc (9 subtests) + TestUniqueKeys/TestChunksOf/TestInPlaceholders (go test -run 'TestUniqueKeys|TestChunksOf|TestInPlaceholders|TestMGetFunc|TestBatchMGet' — 21 passed)"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestBatchMGet (black-box: found subset, empty no-op, TTL expiry via require.Eventually)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Atomic MSet (D-04, BATCH-02): pre-marshal before BeginTx (marshal error writes nothing), resolveTTL once — identical raw expires_at across rows, one BEGIN IMMEDIATE transaction with one prepared UpsertSQL, defer Rollback as the atomicity backstop, 0-or-all invariant under ~1ms context cancellation (2000 pairs x 5 iterations)"
    requirement: "BATCH-02"
    verification:
      - kind: unit
        ref: "cache/sqlite/batch_internal_test.go#TestMSetFunc (5 subtests) + TestMSetNoPartialWrites (go test -run 'TestMSet' — 7 passed)"
        status: pass
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestBatchMSetOneTTL (black-box: every key of the batch expires together)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Chunked MDel (D-05, BATCH-03): deduped chunked IN deletes, missing and duplicate keys no-ops, 250 keys in three chunks, idempotent second call, fail-fast wrapped error on the first chunk failure"
    requirement: "BATCH-03"
    verification:
      - kind: unit
        ref: "cache/sqlite/batch_internal_test.go#TestMDelFunc (6 subtests) + sqlite_test.go#TestBatchMDel (black-box at 150 keys)"
        status: pass
    human_judgment: false
  - id: D4
    description: "CONC-03 race evidence: 8 goroutines x 50 iterations mixing Set/Get/Delete/MGet(3)/MSet(5)/MDel/GetOrSet on per-worker prefixed keys, file and :memory: subtests — go test -race runs clean with both subtests PASSing and no DATA RACE output"
    requirement: "CONC-03"
    verification:
      - kind: unit
        ref: "cache/sqlite/sqlite_test.go#TestConcurrentBatchOpsRace (go test -race ./cache/sqlite/ -run 'TestConcurrentBatchOpsRace' -v -count=1 -timeout=180s — PASS file_mode 0.74s + memory_mode 0.67s)"
        status: pass
    human_judgment: false

# Metrics
duration: 20min
completed: 2026-10-08
status: complete
# Commit ledger (#3968) — measured, not narrated.
plan_head_before: 2f5a7881e3f69f24b742940c4740f9c5adf1a145
plan_head_after: 7a0151a5058c88b1a4f1e31128d2da242e2a707c
commits: 3
---

# Phase 16 Plan 03: Batch Surface + Multi-Process Hardening — MGet/MSet/MDel Summary

**The Phase 15 loop placeholders are gone: batch.go carries all three real batch methods — chunked, deduplicated, warn-and-skip MGet (BATCH-01), a single-transaction prepared-upsert MSet with pre-marshal atomicity and exactly one TTL (BATCH-02), and chunked, idempotent, fail-fast MDel (BATCH-03) — proven race-clean on both file and :memory: modes under 8 goroutines of mixed batch + point operations (CONC-03).**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-10-08T22:31:44Z
- **Completed:** 2026-10-08T22:52:00Z (approx)
- **Tasks:** 3/3
- **Files:** 2 created, 3 modified (984 insertions, 53 deletions)
- **Estimate vs actuals:** plan estimated 60000 tokens (low confidence); realized diff measures 7621 estimate-tokens (chars/4) — the batch mechanics were already probe-verified in 16-RESEARCH, so execution stayed lean.

## Accomplishments

- **batch.go (NEW):** `chunkSize = 100` (D-01), `uniqueKeys` (stringified-form dedup preserving first-seen order; form collisions collapse to the first occurrence — D-02), `chunksOf` (order-preserving bounded splits), `inPlaceholders` (only `?` markers generated; keys/values stay bound parameters — never caller data in SQL).
- **MGetFunc:** per-chunk `map[string]K` re-keying (fmt.Sprint is not invertible — Finding 5), expiry predicate folded into `MGetSelectSQL` so expired rows are absent by SQL; the raw row count stays 1 after MGet (reads never delete — TTL-02). Chunk query failure → `logger().Warn` with the key count + wrapped error, remaining chunks still run (D-03); undecodable rows → `logger().Debug` skip. Rows are fully drained and closed via `defer` inside the per-chunk helper before the next chunk issues (pinned 1-conn pool — Pitfall 3).
- **MSetFunc:** every value pre-marshals into `msetPair` before `BeginTx` — a marshal error (proven with a chan value amid valid pairs) writes zero rows. `resolveTTL` called exactly once; every row carries the same absolute expiry (raw `expires_at` equality asserted). One `BEGIN IMMEDIATE` transaction (+`_txlock=immediate` DSN), one `tx.PrepareContext(UpsertSQL)`, one `ExecContext` per pair, `Commit`; `defer tx.Rollback()` is the atomicity backstop on every error class. `TestMSetNoPartialWrites` proves 0-or-2000 under ~1 ms cancellation across 5 iterations.
- **MDelFunc:** chunked IN deletes over deduplicated keys; missing/duplicate keys are no-ops; the first chunk error returns the wrapped `cache/sqlite:` error (fail-fast, idempotent caller retry — D-05); closed-handle errors carry the prefix.
- **schema.go:** `MGetSelectSQL` (selects `cache_key` AND `value` — WITHOUT ROWID has no rowid shortcut, Finding 5) + `MDelSQL` templates, both with the `%s`-is-only-`inPlaceholders` marker contract.
- **sqlite.go:** all three Phase 15 loop placeholders removed — the batch seam now lives exclusively in batch.go (grep-verified: 0 declarations in sqlite.go, 1 each in batch.go).
- **Race evidence (CONC-03):** `TestConcurrentBatchOpsRace` — file + `:memory:` subtests, 8 goroutines x 50 iterations mixing Set/Get/Delete/MGet(3 keys)/MSet(5 keys)/MDel/GetOrSet on per-worker prefixed keys with a start-channel release; passes `go test -race` clean (file 0.74 s, memory 0.67 s), no DATA RACE output, no goroutine outlives the test (t.Context()).
- **Gates:** full package suite green (140 tests), `make coverage-quick` green (81.9% total, sqlite 91.2% package — **no** cache/sqlite coverage override, verified by the plan's grep gate), golangci-lint at 0 issues.

## Task Commits

Each task was committed atomically:

1. **Task 1: Chunked MGet — batch.go helpers + MGetSelectSQL + placeholder removal** - `05acd7c` (feat)
2. **Task 2: MSet atomic prepared-upsert transaction + chunked MDel** - `6b1ff03` (feat)
3. **Task 3: Black-box batch semantics + CONC-03 race test + package gates** - `7a0151a` (test)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update — docs commit)

## Files Created/Modified

- `cache/sqlite/batch.go` (NEW) — `msetPair` type, `chunkSize` const, `uniqueKeys`/`chunksOf`/`inPlaceholders` helpers, `MGetFunc` + `mgetChunk`, `MSetFunc` + `msetTx`, `MDelFunc`.
- `cache/sqlite/batch_internal_test.go` (NEW) — `TestUniqueKeys` (3 subtests), `TestChunksOf` (5 boundary cases), `TestInPlaceholders`, `TestMGetFunc` (9 subtests), `TestMSetFunc` (5 subtests), `TestMSetNoPartialWrites` (5 iterations), `TestMDelFunc` (6 subtests), `countTableRows` + `readRawExpiry` helpers (declared once — single-ownership with the existing `countKeyRows`/`keyRowCount`).
- `cache/sqlite/schema.go` (MOD) — `MGetSelectSQL`, `MDelSQL`.
- `cache/sqlite/sqlite.go` (MOD) — the three placeholder methods removed (44 lines deleted).
- `cache/sqlite/sqlite_test.go` (MOD) — `TestBatchMGet`, `TestBatchMSetOneTTL`, `TestBatchMDel`, `TestBatchSemantics` (placeholder replacement), `TestConcurrentBatchOpsRace` + `runBatchWorkers`.

## Decisions Made

- TDD commit pattern under the repo constraint (documented in 15-01..16-02): in-session assertion-level RED evidence (stub-then-real), verified via `gsd_run check tdd-red-evidence` (RED_EVIDENCE_OK for both tdd tasks), shipping with the implementation in feat commits; the test-only task commits as `test(16-03)`.
- The placeholder smoke test's removal precedes Task 3's full replacement because its MSet/MDel legs depended on the then-stubbed methods (Rule 3 — suite gate); `TestBatchMGet` covers the MGet contract in Task 1 exactly as the plan's Task-1 verify regex expects; `TestBatchSemantics` owns the complete black-box contract per Task 3.
- `msetTx` extraction keeps MSetFunc under cyclop's complexity budget while preserving the plan's single-transaction letter (BeginTx/Prepare/Commit counts of 1/1/1 verified by grep; zero SetFunc references in batch.go — the no-per-pair-autocommit acceptance is provable from the source).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] TestBatchPlaceholderSmoke broken by the placeholder removal order**
- **Found during:** Task 1 (full-suite gate before commit)
- **Issue:** The smoke test exercised MSet/MDel, which were still stubs until Task 2, so the Task-1 suite gate failed.
- **Fix:** Removed it in Task 1 (its MGet leg is covered by the new black-box `TestBatchMGet`, exactly the name Task 1's verify regex runs); the full replacement `TestBatchSemantics` landed in Task 3 as the plan intends. The acceptance criterion "placeholder smoke test no longer exists" holds.
- **Files modified:** cache/sqlite/sqlite_test.go (+ TestBatchMGet, − placeholder)
- **Committed in:** `05acd7c` (removal + TestBatchMGet), `7a0151a` (TestBatchSemantics)

**2. [Rule 3 - Blocking] Lint accommodations (cyclop/funlen/gocognit/sqlclosecheck)**
- **Found during:** Tasks 1-2 (golangci-lint runs before commits)
- **Issue:** `MSetFunc` cyclomatic complexity 11 > 10 (cyclop); `_ = rows.Close()` in the chunk loop flagged (sqlclosecheck); behavior-matrix and harness tests exceeded funlen; `runBatchWorkers` also tripped gocognit.
- **Fix:** Extracted `msetTx` and `mgetChunk` helpers (structural, behavior-neutral — single-tx seam unchanged); scoped `//nolint` directives on test functions with justification comments (repo precedent: mem_test.go).
- **Files modified:** cache/sqlite/batch.go, cache/sqlite/batch_internal_test.go, cache/sqlite/sqlite_test.go
- **Verification:** golangci-lint run ./cache/sqlite/... → 0 issues; all tests green.
- **Committed in:** `05acd7c`, `6b1ff03`, `7a0151a`

**3. [Rule 3 - Blocking] testifylint and perfsprint/wsl_v5 in the new tests**
- **Found during:** Task 2-3 lint runs
- **Issue:** `assert.Len(..., 0)` (should be `assert.Empty`), `fmt.Sprintf` of a constant pattern (perfsprint), and two "missing whitespace above decl" warnings (wsl_v5).
- **Fix:** Idiomatic replacements; no behavior change.
- **Files modified:** cache/sqlite/sqlite_test.go, cache/sqlite/batch_internal_test.go
- **Committed in:** `6b1ff03`, `7a0151a`

---

**Total deviations:** 3 auto-fixed (all Rule 3 — commit ordering and lint accommodations; zero behavioral effect, no scope creep).
**Impact on plan:** None — all acceptance criteria met as written.

## TDD Gate Compliance

Repo constraints (no empty commits; per-commit `make coverage-quick` green — a standalone RED commit cannot pass the AGENTS.md gate) override the standalone-RED-commit pattern, as documented in 15-01/15-02/16-02. In-session RED evidence was recorded for both `tdd="true"` tasks and machine-verified:

- **Task 1 (tracer):** transient stubs (identity helpers + empty-map MGetFunc) produced assertion-level RED: 14 failing targets across TestUniqueKeys/TestChunksOf/TestInPlaceholders/TestMGetFunc — verified `RED_EVIDENCE_OK` via `gsd_run check tdd-red-evidence` on `16-03-RED-EVIDENCE-task1.json` (record: exit 1, target TestUniqueKeys, expected/actual documented). GREEN replaced the stubs with the real helpers + chunked MGet; 21/21 target tests + full suite passed. Commit `feat(16-03)` `05acd7c`. Tracer gate: full `<verify>` re-run green before expansion.
- **Task 2:** `TestMSetFunc`/`TestMSetNoPartialWrites`/`TestMDelFunc` written first; the no-op stubs failed 11 targets — verified `RED_EVIDENCE_OK` on `16-03-RED-EVIDENCE-task2.json`. GREEN implemented the prepared-upsert transaction + chunked MDel; 16/16 target + black-box tests and the full suite passed. Commit `feat(16-03)` `6b1ff03`.
- **Task 3:** test-only (no `tdd="true"`): `test(16-03)` commit per the repo pattern.
- **Gate-scope check:** commits match `^(feat|test)\((0*16)-(0*3)\):`; three commits total for the plan — the single `test(16-03)` lands last by repo design (Plan File-3 task is inherently test-only).

## Issues Encountered

- None beyond the recorded lint accommodations; the tracer feedback gate re-ran the complete `<verify>` end-to-end before expansion.
- Pre-existing, out of scope: `internal_probe/two_process_test.go` (untracked scratch from the 16-01 spike) still fails LSP with `undefined: crashHelper`; not tracked, not shipped, not touched by this plan (same note as 16-02).

## Known Stubs

None introduced by this plan — all three Phase 15 batch placeholders are replaced by the real implementations in batch.go. (The RED-phase transient stubs were replaced in-session and never shipped.)

## Threat Flags

None — the plan's threat register is fully enforced:
- T-16-06: only `inPlaceholders` markers enter generated SQL (helper unit-pinned); keys/values/now bound; dedup-then-chunk order fixed.
- T-16-07: warn logs carry key counts + wrapped errors only; key strings appear only at debug for decode skips; no new logging surface.
- T-16-08: rows drained + closed per chunk (defer in `mgetChunk`'s scope) before the next chunk query; `rows.Err()` checked.
- T-16-09: pre-marshal before BeginTx, single transaction with defer Rollback, one prepared upsert, 0-or-all invariant repeated across 5 cancellation runs.
- T-16-SC: no package installs (go.mod untouched).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 16-04 (provider-driven E2E children + crash arm + CI) consumes the shipped batch surface: children can now drive `MSet`/`MGet`/`MDel` through the real provider instead of raw SQL mirrors.
- CONC-03 satisfied with race-clean evidence on both pool modes; the CI race step (linux/macos per research OQ3) will run this test in the matrix.
- The BatchCache surface is complete across all v1.6 semantics on cache/sqlite; interfaces untouched (frozen `cacher` compiled unchanged through all three commits).
- No blockers.

---
*Phase: 16-batch-surface-multi-process-hardening*
*Completed: 2026-10-08*

## Self-Check: PASSED

- Files: batch.go, batch_internal_test.go, schema.go, sqlite.go, sqlite_test.go, 16-03-SUMMARY.md all exist on disk.
- Commits: 05acd7c (Task 1), 6b1ff03 (Task 2), 7a0151a (Task 3) all present in git history.
- Plan-range change set (2f5a788..HEAD) touches exactly the five planned `cache/sqlite/` paths; no production files elsewhere, no coverage override added, go.mod untouched.
- TDD gate: both tdd tasks hold in-session RED evidence records verified RED_EVIDENCE_OK (`16-03-RED-EVIDENCE-task1.json`, `16-03-RED-EVIDENCE-task2.json`, phase dir; not tracked — evidence artifacts).
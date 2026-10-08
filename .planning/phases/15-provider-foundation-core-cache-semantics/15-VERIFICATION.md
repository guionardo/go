---
phase: 15-provider-foundation-core-cache-semantics
verified: 2026-10-08T15:05:38Z
status: passed
score: 18/19
covered_files:
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-01-PLAN.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-02-PLAN.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-03-PLAN.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-01-SUMMARY.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-02-SUMMARY.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-03-SUMMARY.md
  - cache/sqlite/doc.go
  - cache/sqlite/dsn.go
  - cache/sqlite/dsn_internal_test.go
  - cache/sqlite/example_test.go
  - cache/sqlite/optimize.go
  - cache/sqlite/options.go
  - cache/sqlite/options_internal_test.go
  - cache/sqlite/schema.go
  - cache/sqlite/sqlite.go
  - cache/sqlite/sqlite_internal_test.go
  - cache/sqlite/sqlite_test.go
  - cache/sqlite/sweep.go
  - README.md
  - go.mod
  - go.sum
covered_digest: "v2:sha256:3d7396461293dd00c148f2b5834f4b456f9c29a8825a40686455134300575720"
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "Schema bootstrap: a 4-goroutine concurrent first-open of a fresh path leaves exactly one cache_entries table and all handles usable (STOR-04 plan stress; 15-REVIEW WR-01 = deferred-items.md DI-15-01)"
    addressed_in: "Phase 16"
    evidence: "Phase 16 goal: 'two OS processes share one cache file under WAL without corruption, with contention bounded by busy_timeout'; CONC-01/CONC-02 (REQUIREMENTS.md traceability); 15-RESEARCH assigns the bounded retry backstop to Phase 16; deferred-items.md DI-15-01 recommends the retry policy plus re-strengthening TestBootstrapIdempotence/concurrent_first_open_all_usable."
---
<!-- covered_digest regenerated with the deterministic verification.fingerprint verb on 2026-10-08 (the verifier's hand-rolled digest did not match the verb's computation; sanity-checked: all covered files git-clean at regeneration time). -->

# Phase 15: Provider Foundation + Core Cache Semantics Verification Report

**Phase Goal:** A working, durable `cache/sqlite` provider — `sqlite.New[K, V]` constructs a `cache.BatchCache[K, V]` in all three location modes; Get/Set/Delete/GetOrSet/Close match the five existing providers' contracts; TTL expiry and sweeps are enforced; WAL pragmas are verified on every pooled connection; and long-running storage controls are available.
**Verified:** 2026-10-08T15:05:38Z
**Status:** passed (1 item deferred to Phase 16 — see Deferred Items)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | `sqlite.New[K, V]` returns a real `cache.BatchCache[K, V]` built on the 7-method provider seam through `cache.NewConcreteCache` (roadmap SC1, PROV-01) | ✓ VERIFIED | `cache/sqlite/sqlite.go:62-78` (`New` → `&batchCache{BatchCache: cache.NewConcreteCache(c), provider: c}`); `cache/concrete_cache.go:14-26` consumes `GetFunc/SetFunc/DeleteFunc/CloseFunc/MGetFunc/MSetFunc/MDelFunc` structurally (compile-enforced); `TestMemoryRoundTrip`, `TestFileCRUD` pass |
| 2 | Three location modes are deterministic — memory > path > name > zero(=memory); `WithName` resolves to `os.UserCacheDir()/<name>/cache.db` with the directory created 0700; explicit path accepted as given (SC1, STOR-01) | ✓ VERIFIED | `dsn.go:32-56`; `TestLocation` runs an 8-row precedence matrix incl. injected `userCacheDir` error propagation; e2e scratch run (`New(WithName("verify-phase15"))` with temp HOME): `ROUNDTRIP=v DIR_OK=true PERM=700` |
| 3 | Paths/names containing `'?'`/`'#'` and invalid names are rejected via the deferred error path: `New` returns a non-nil handle; first operation wraps `ErrInvalidPath` (STOR-06, D-11/D-12) | ✓ VERIFIED | `dsn.go:37-44`, `sqlite.go:164-174`; `TestInvalidPathAndName` (3 bad paths, 7 bad names, valid-name matrix), `TestDeferredErr` (Get/Set/Delete all `errors.Is(ErrInvalidPath)`; `Close` nil) |
| 4 | File mode read-back proves `journal_mode=wal`, `busy_timeout=5000`, `synchronous=1` on the pinned connection; memory DSN omits the `journal_mode` key (SC5, STOR-03) | ✓ VERIFIED | `dsn.go:14-16` (compile-time constant suffixes); `TestPragmaReadBack` (file + memory subtests); `TestDSNBuildDSN` asserts exact string equality and `NotContains("journal_mode")` for memory |
| 5 | Pool pinned to exactly one connection in both modes; independently constructed memory caches never share entries (STOR-02) | ✓ VERIFIED | `sqlite.go:100-103` (`SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, zero lifetimes); `TestPoolPinned` (both modes `Stats().MaxOpenConnections == 1`); `TestMemoryIsolation` |
| 6 | Schema bootstrap idempotent under one `BEGIN IMMEDIATE` (`_txlock=immediate`); sequential reopen leaves one table; 4-goroutine concurrent first-open leaves one table and usable handles (STOR-04) | ✗ FAILED — **deferred** | Sequential leg VERIFIED (`TestBootstrapIdempotence/sequential_reopen_leaves_one_table`). Concurrent leg nondeterministically fails: measured **29/40 isolated subtest runs** failing with `cache/sqlite: database is locked (5) (SQLITE_BUSY)` (WAL-conversion race, pre-existing at `3d92e70`). Explicitly deferred to Phase 16 — see Deferred Items. |
| 7 | `GetOrSet` dedups through the shared `SingleflightGetOrSet` (fast-path Get, double-check Get, setter-once, panic recovery); the provider adds no second dedup layer (PROV-02, SC2) | ✓ VERIFIED | `concrete_cache.go:54-61` → `cache/singleflight.go:35-71`; provider itself has no dedup code; `TestGetOrSetDedup` (20 concurrent misses, atomic setter count == 1, identical results) passes under `-race` |
| 8 | Key/value parity: `fmt.Sprint` keys, `encoding/json` values; int↔stringified, unicode, empty-string, struct roundtrips; marshal failure errors with nothing stored; corrupted row is not `ErrMiss` (PROV-03) | ✓ VERIFIED | `sqlite.go:187-199` (only `sql.ErrNoRows` → `ErrMiss`; unmarshal failure returned as error), `sqlite.go:212-226`; `TestParity` (3 subtests), `TestMarshalFailure`, `TestCorruptedRowNotMiss` |
| 9 | Error taxonomy: `cache/sqlite:` prefix on provider errors; post-Close ops wrap `ErrClosed`; `Close` idempotent (second call nil); canceled context errors (PROV-04, SC2) | ✓ VERIFIED | `sqlite.go:164-174`, `248-261`; `TestClosedOps`, `TestClosedTaxonomy`, `TestClosedErrorPrefix`, `TestCanceledContext`, `TestDoubleClose` (sequential + 8-way concurrent, race-clean). Cosmetic double-prefix noted as IN-01 (Info, non-blocking). |
| 10 | Batch surface compiles and smoke-passes as loop-based placeholders with `Phase 16:` replacement markers; 7-method interface structurally satisfied (PROV-01 phase boundary) | ✓ VERIFIED | `sqlite.go:263-305` (`MGetFunc/MSetFunc/MDelFunc` with `Phase 16:` comments); `TestBatchPlaceholderSmoke` |
| 11 | TTL resolution matches the postgres contract: per-call > 0 wins; else default > 0; else none; negative/multiple inputs follow the `> 0` predicate (TTL-01) | ✓ VERIFIED | `sqlite.go:309-319`; `TestResolveTTL` 7-row `ttlMatrix` (incl. expiry-window assertions) |
| 12 | Expiry stored as absolute UnixNano (`expires_at INTEGER`, NULL = never); survives restarts (TTL-01) | ✓ VERIFIED | `sqlite.go:217-224`; `TestReopenPersistence` (no-TTL and 1h-TTL entries returned after Close + reopen) |
| 13 | Expired entries are never returned (`Get` → `ErrMiss`); reads never delete rows (TTL-02) | ✓ VERIFIED | `schema.go:26-27` (expiry predicate folded into SELECT), `sqlite.go:187-190`; `TestTTLExpiry`, `TestDefaultTTLExpiry`, `TestNoExpiry`, `TestReadsNeverDeleteRows` (raw `COUNT(*)` stays 1 after expired Get) |
| 14 | Best-effort sweep runs synchronously on open; expired rows reclaimed; sweep errors logged, never surfaced (TTL-03, D-09/D-10) | ✓ VERIFIED | `sqlite.go:118-119`, `sweep.go:11-15`, `schema.go:40` (`expires_at IS NOT NULL AND expires_at <= ?`, bound `time.Now().UnixNano()`, never SQL `datetime('now')`); `TestSweepOnOpen` (raw observer: stale COUNT 0, fresh COUNT 1), `TestSweepFailureSwallowed` |
| 15 | Opt-in periodic sweep (default: no sweeper at all) reclaims on ticks; `Close` cancels and waits before DB close; second Close nil (TTL-04, D-08) | ✓ VERIFIED | `sqlite.go:121-125`, `248-261` (CAS → `close(stop)` → `<-done` → `db.Close()`); `sweep.go:19-43`; `TestPeriodicSweep` (Eventually COUNT 0, double Close), `TestSweeperDisabled` (nil stop/done, no goroutine), `TestOptions_WithSweepInterval`; `-race` subset green |
| 16 | Clean shutdown: db file remains, `-wal`/`-shm` removed by the engine's last connection; reopen preserves unexpired entries and stays writable (STOR-05, SC3) | ✓ VERIFIED | `sqlite.go:255-257`; `TestSidecarRemoval` (`require.Eventually` both sidecars absent + db present), `TestReopenPersistence`, `TestExpiryAcrossRestart`, `TestCloseAfterDeferredError` (nil on poisoned handle) |
| 17 | `Optimizable` type-assertable on `New`'s return: `Checkpoint` = `PRAGMA wal_checkpoint(TRUNCATE)` with three-column scan, busy≠0 → descriptive error (never silent nil); `Vacuum` = full VACUUM; memory checkpoint valid no-op; closed/deferred states wrap sentinels (STOR-07, SC5) | ✓ VERIFIED | `optimize.go:20-96`, `sqlite.go:74-77`; `TestOptimizable`, `TestOptimizableMemory`, `TestCheckpointBlocked` (second connection holds read tx; error contains `busy`), `TestOptimizableClosed`, `TestOptimizableDeferredErr`, `TestCheckpointColumns` |
| 18 | `WithAutoCheckpoint(pages)`: >0 appends `_pragma=wal_autocheckpoint(<strconv.Itoa>)` to the file-mode DSN only; read-back shows configured value; zero keeps driver default 1000; memory ignores it (STOR-07, A6) | ✓ VERIFIED | `dsn.go:82-92`, `options.go:72-79`, `sqlite.go:95`; `TestAutoCheckpointReadBack` (200 / default 1000 / memory-ignored), `TestDSNBuildDSN` append cases (200, ≤0, memory) |
| 19 | Package slice ships: `doc.go` (usage, precedence, TTL, persistence, WAL/local-storage note), runnable Example, README rows; repo-wide gates green with no coverage override (plan 15-03) | ✓ VERIFIED | `doc.go` (49 lines), `example_test.go` (Example PASS in suite), `README.md:22,85`; `go test ./...` → 905 passed / 26 packages; `make coverage-quick` → sqlite **92.5%**, total **81.5%**, thresholds PASS, `grep -c 'cache/sqlite' .testcoverage-quick.yml` = 0; `golangci-lint run` → 0 issues; Windows CGO-free cross-build OK |

**Score:** 18/19 truths verified (0 behavior-unverified; 1 deferred to Phase 16 — deferred items do not open gaps per Step 9b)

### Deferred Items

Items not achieved in this phase but explicitly addressed in a later milestone phase.

| # | Item | Addressed In | Evidence |
| - | ---- | ------------ | -------- |
| 1 | Concurrent first-open of a fresh DB can defer `SQLITE_BUSY` on the per-connection `PRAGMA journal_mode=WAL` (WAL conversion race); `New` returns a handle whose every operation then fails until discarded. Covering test `TestBootstrapIdempotence/concurrent_first_open_all_usable` is nondeterministically red. | Phase 16 | Phase 16 goal: "two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`"; CONC-01 (sharing without corruption/hard failure under contention) and CONC-02 (contention surfaces at BEGIN; no unbounded auto-retry) in `REQUIREMENTS.md`; `15-RESEARCH` assigns the bounded retry backstop to Phase 16 ("do not build retry here"); `deferred-items.md` DI-15-01 carries the root cause (isolation table) and recommends the retry + regression-test re-strengthening. Review WR-01 is the same issue (`15-REVIEW-DISPOSITION.md`, open). |

Measured during this verification (strengthening the Phase 16 case): **29/40 isolated runs of the subtest fail** (~72%) — notably higher than the ~5–12% per-full-suite estimate documented at deferral time; the full suite (85/85), the race subset, `make coverage-quick`, and lint were all green on this pass.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `cache/sqlite/sqlite.go` | provider type, `New`, open/bootstrap, `check`, 7 primitive methods, `resolveTTL`, sentinels, logger | ✓ VERIFIED | 319 lines; exports `New`/`ErrClosed`/`ErrInvalidPath`; all 9 SQL call sites context-aware; wired via `New` |
| `cache/sqlite/options.go` | `Config` + `WithName/WithPath/WithMemory/WithDefaultTTL/WithSweepInterval/WithAutoCheckpoint` | ✓ VERIFIED | 80 lines; zero-value config = memory mode |
| `cache/sqlite/dsn.go` | `resolveLocation`, `validName`, `buildDSN` (+ constant suffixes) | ✓ VERIFIED | 93 lines; pure functions fully unit-tested |
| `cache/sqlite/schema.go` | 6 SQL constants (schema + statements); fixed three-column `WITHOUT ROWID` + partial index; no `user_version` | ✓ VERIFIED | 41 lines |
| `cache/sqlite/sweep.go` | `sweep(ctx)` (no error return), `startSweeper`, `sweepLoop` | ✓ VERIFIED | 43 lines; stop/done contract |
| `cache/sqlite/optimize.go` | `Optimizable` interface + `batchCache` adapter + `checkpoint`/`vacuum` | ✓ VERIFIED | 96 lines; VACUUM confined to this file (prohibition grep) |
| `cache/sqlite/doc.go` | package documentation | ✓ VERIFIED | 49 lines; usage/precedence/TTL/persistence/WAL note |
| `cache/sqlite/example_test.go` | runnable example | ✓ VERIFIED | deterministic `// Output: value: hello`, PASS in suite |
| `cache/sqlite/{sqlite,sqlite_internal,dsn_internal,options_internal}_test.go` | behavioral/internal test suite | ✓ VERIFIED | 47 test functions + 1 Example / 85 tests run, all green; package coverage 92.5% |
| `go.mod` / `go.sum` | `modernc.org/sqlite v1.60.1` + exact `modernc.org/libc v1.77.1` pin | ✓ VERIFIED | lines 19/120 with cznic/sqlite#177 comment; `x/sys v0.48.0`, `x/sync v0.23.0` present |
| `README.md` | package index row + provider table row | ✓ VERIFIED | lines 22 and 85 contain `cache/sqlite` |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `New` | `resolveLocation` → `buildDSN` → `sql.Open("sqlite", dsn)` | open sequence | WIRED | `sqlite.go:83-98`; blank modernc import at `sqlite.go:16` |
| `open` | pool pin → `bootstrap` (`BeginTx` = BEGIN IMMEDIATE) | `_txlock=immediate` in DSN | WIRED | `sqlite.go:100-112`; driver honors `_txlock` (vendored `modernc.org/sqlite@v1.60.1/sqlite.go:395`) |
| `New` | `cache.NewConcreteCache(c)` | structural 7-method interface | WIRED | `sqlite.go:74-77`; `concrete_cache.go:14-26`; compile-enforced |
| `GetOrSet` | `concreteCache.sf.Do` → `GetFunc`/`SetFunc` | shared singleflight | WIRED | `concrete_cache.go:54-61` → `singleflight.go:35-71`; no provider-side dedup |
| `GetFunc` | `SelectSQL` → `sql.ErrNoRows` → wrapped `cache.ErrMiss` | miss mapping | WIRED | `sqlite.go:187-190`; corollary: all other errors wrapped as themselves |
| `open` | `c.sweep(ctx)` (synchronous, best-effort) | before `New` returns | WIRED | `sqlite.go:118-119` → `sweep.go:11-15` |
| `WithSweepInterval > 0` | `startSweeper` → `sweepLoop` → ticker → `sweep` | opt-in goroutine | WIRED | `sqlite.go:121-125`; `sweep.go:19-43` |
| `Close` | `close(stop)` → `<-done` → `db.Close()` | cancel-and-join before DB close | WIRED | `sqlite.go:248-261`; last-connection close is the only sidecar-removal channel (no `os.Remove` in package) |
| `New` | `&batchCache{BatchCache, provider}` | `Optimizable` assertion target | WIRED | `sqlite.go:74-77` → `optimize.go:31-60` (frozen root wrapper cannot forward provider methods) |
| `Checkpoint` | `check()` → `PRAGMA wal_checkpoint(TRUNCATE)` → 3-col scan → busy≠0 error | maintenance state machine | WIRED | `optimize.go:66-82`; `TestCheckpointColumns`/`TestCheckpointBlocked` |
| `WithAutoCheckpoint` | `Config.AutoCheckpoint` → `buildDSN` → `_pragma=wal_autocheckpoint(<int>)` | DSN append | WIRED | `options.go:76-79` → `dsn.go:88-90` → `sqlite.go:95`; read-back test |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `sqlite.go:GetFunc` | `data` | `SelectSQL` (`QueryRowContext`) against `cache_entries`, expiry-filtered | Yes — real SQLite query, no fallback | ✓ FLOWING |
| `sqlite.go:SetFunc` | `data`, `exp` | `json.Marshal` + `UpsertSQL` positional binds | Yes — persisted JSON; NULL when no TTL | ✓ FLOWING |
| `sweep.go:sweep` | bound now | `SweepSQL` (`ExecContext`) with `time.Now().UnixNano()` | Yes — real DELETE pass | ✓ FLOWING |
| `optimize.go:checkpoint` | `busy/logFrames/checkpointed` | `PRAGMA wal_checkpoint(TRUNCATE)` three-column result | Yes — engine result, busy surfaced | ✓ FLOWING |
| `dsn.go:buildDSN` | DSN string | `resolveLocation` output + `strconv.Itoa(pages)` | Yes — read back via `PRAGMA wal_autocheckpoint` | ✓ FLOWING |

No static returns, hardcoded fallbacks, or mock data sources found in production code.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full package suite (contracts, TTL, sweeps, lifecycle, Optimizable) | `go test ./cache/sqlite/...` | 85 passed | ✓ PASS |
| Dedup/isolation/close/sweeper under race detector | `go test ./cache/sqlite/... -race -run 'TestGetOrSetDedup\|TestMemoryIsolation\|TestDoubleClose\|TestPeriodicSweep'` | 6 passed, no data race | ✓ PASS |
| Repo-wide regression (frozen root and five providers intact) | `go test ./...` | 905 passed / 26 packages | ✓ PASS |
| Coverage gate, no override (AGENTS.md pre-commit) | `make coverage-quick` | sqlite 92.5%, total 81.5%, all thresholds PASS; `.testcoverage-quick.yml` has no sqlite entry | ✓ PASS |
| Lint | `golangci-lint run ./cache/sqlite/...` | 0 issues | ✓ PASS |
| CGO-free Windows build | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/` | exit 0, no output | ✓ PASS |
| `WithName` e2e (directory creation + roundtrip; no dedicated test exists) | scratch `go run` with temp HOME: `New(WithName("verify-phase15"))` → Set/Get | `ROUNDTRIP=v DIR_OK=true PERM=700` | ✓ PASS |
| Known deferred race (DI-15-01 / WR-01) | `go test ./cache/sqlite/ -run 'TestBootstrapIdempotence/concurrent_first_open_all_usable' -count=1` ×40 | 29/40 failed with `database is locked (5) (SQLITE_BUSY)` | ✗ FAIL — known, deferred to Phase 16 |
| Prohibition greps | `cache=shared` / `auto_vacuum` / `os.Remove\|Shutdown` / post-open `Exec(...PRAGMA)` | all empty; `VACUUM` only in `optimize.go` | ✓ PASS |

### Probe Execution

Skipped — no probes declared (no `scripts/*/tests/probe-*.sh`; the plans mention only throwaway empirical probes, not committed probe scripts).

🚫 SKIP REASON: no documented probes in plans or repository.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| PROV-01 | 15-01 | `New` returns `cache.BatchCache` on the `cacher` seam via `NewConcreteCache`, matching the five providers | ✓ SATISFIED | Truths 1, 10; `sqlite.go:62-78`; suite green |
| PROV-02 | 15-01 | `GetOrSet` uses shared `SingleflightGetOrSet`; no second dedup layer | ✓ SATISFIED | Truth 7; `TestGetOrSetDedup` (race) |
| PROV-03 | 15-01 | `fmt.Sprint` keys, JSON values, misses wrap `cache.ErrMiss` | ✓ SATISFIED | Truth 8; `TestParity`, `TestMarshalFailure` |
| PROV-04 | 15-01 | Context-aware SQL; Delete-missing no-op; idempotent Close; `cache/sqlite:` prefix | ✓ SATISFIED | Truths 3, 9; all 9 SQL sites context-aware; `TestClosedOps`, `TestCanceledContext` |
| STOR-01 | 15-01 | Default `os.UserCacheDir()/<name>/` (MkdirAll 0700); explicit path as given | ✓ SATISFIED | Truth 2; e2e scratch run `PERM=700`; `TestFilePathWithSpaces` (nested dir creation) |
| STOR-02 | 15-01 | Empty path = `:memory:`; pool pinned 1/1 both modes | ✓ SATISFIED | Truths 2, 5; `TestPoolPinned`, `TestMemoryIsolation` |
| STOR-03 | 15-01 | DSN-carried pragmas (WAL/busy_timeout/synchronous/immediate); read-back asserts WAL | ✓ SATISFIED | Truth 4; `TestPragmaReadBack`, `TestDSNBuildDSN` |
| STOR-04 | 15-01 | Idempotent `BEGIN IMMEDIATE` bootstrap; fixed three-column schema | ✓ SATISFIED | Truth 6 — sequential idempotence verified; concurrent first-open stress deferred to Phase 16 (Deferred Items #1) |
| STOR-05 | 15-02 | Clean Close (final checkpoint; sidecars removed); reopen preserves entries | ✓ SATISFIED | Truth 16; `TestSidecarRemoval`, `TestReopenPersistence` |
| STOR-06 | 15-01 | `'?'`/`'#'` rejected so paths cannot inject DSN parameters | ✓ SATISFIED | Truth 3; `TestInvalidPathAndName`; grep: no caller strings in DSN |
| STOR-07 | 15-03 | Tune `wal_autocheckpoint`; explicit optimize (checkpoint and/or VACUUM) | ✓ SATISFIED | Truths 17, 18; `TestAutoCheckpointReadBack`, `TestOptimizable*` |
| TTL-01 | 15-02 | TTL resolution parity; absolute UnixNano expiry survives restarts | ✓ SATISFIED | Truths 11, 12; `TestResolveTTL`, `TestReopenPersistence` |
| TTL-02 | 15-02 | Expired never returned (`ErrMiss`); reads never delete | ✓ SATISFIED | Truth 13; `TestReadsNeverDeleteRows` |
| TTL-03 | 15-02 | Best-effort open sweep; partial index on `expires_at` | ✓ SATISFIED | Truth 14; `TestSweepOnOpen`; `CreateIndexSQL` partial index |
| TTL-04 | 15-02 | Optional periodic sweep interval; unset → open sweep + read filtering only | ✓ SATISFIED | Truth 15; `TestPeriodicSweep`, `TestSweeperDisabled` |

**Orphaned requirements:** none — PLAN frontmatter covers exactly the 15 IDs ROADMAP/REQUIREMENTS map to Phase 15 (PROV-01..04, STOR-01..07, TTL-01..04); no Phase 15 requirement is unclaimed.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers anywhere in `cache/sqlite/` | — | Debt-marker gate clean |
| `cache/sqlite/sqlite.go` | 266, 283, 296 | `Phase 16:` replacement markers on `MGetFunc`/`MSetFunc`/`MDelFunc` loop implementations | ℹ️ Info | Intentional phase-boundary decision: the methods do real work and smoke-pass; chunked/transactional semantics are BATCH-01..03 in Phase 16 (REQUIREMENTS.md traceability) |
| `cache/sqlite/sqlite.go` | 166, 170 | `check()` wraps sentinels that already carry the `cache/sqlite:` prefix → double prefix (review IN-01) | ℹ️ Info | Cosmetic; `errors.Is` intact; contract "errors carry the prefix" still holds; recorded `open` in `15-REVIEW-DISPOSITION.md` |
| `cache/sqlite/dsn.go` | 44 | Invalid cache names use `ErrInvalidPath` with "invalid path" wording (review IN-02) | ℹ️ Info | Matches the plan's prescribed contract exactly; suggestion for a dedicated sentinel is recorded `open` in `15-REVIEW-DISPOSITION.md` |

No 🛑 Blockers: all prohibition greps clean (`cache=shared`, post-open PRAGMA exec, sidecar deletion, `auto_vacuum`, SQL `datetime()`, `VACUUM` locality).

### Human Verification Required

None. The phase has no visual, real-time, or external-service surface; all 19 truth areas are machine-verified by 85 green tests plus targeted commands re-run during this verification. The one robustness failure is a measured, automated, deferred item (Phase 16), not a human-testing question.

### Gaps Summary

No blocking gaps. The phase goal is achieved on disk and reproduced by execution: the constructor builds a real `cache.BatchCache` through the frozen `NewConcreteCache` seam in all three location modes (including an end-to-end `WithName` check with 0700 directory creation); the primitive CRUD/GetOrSet/Close contracts match the existing providers (parity, misses, error taxonomy, singleflight dedup); TTL resolution, absolute-expiry storage, filter-only reads, open sweep, and the opt-in periodic sweep are enforced; WAL pragmas are verified by read-back on the pinned connection; and the long-running controls (`Optimizable` checkpoint/vacuum, `WithAutoCheckpoint`) are type-assertable and read-back-proven. The frozen `cache/` root and five providers are untouched (git diff scope: `cache/sqlite/*`, `go.mod`, `go.sum`, `README.md`), the security register shows `threats_open: 0`, and the repo is green repo-wide with no coverage override.

One carried-forward item: the plan-added 4-goroutine concurrent first-open stress fails nondeterministically (`SQLITE_BUSY` on WAL conversion; 29/40 isolated failures measured today, ~5–12% per full suite). It is a pre-existing condition with root-cause evidence, explicitly assigned to Phase 16 (CONC-01/02) by `15-RESEARCH`, `deferred-items.md` (DI-15-01), and review WR-01 — recorded under Deferred Items, not counted as a Phase 15 gap. Phase 16 should treat the elevated isolated failure rate as its regression evidence. Two Info-level review findings (IN-01 double prefix, IN-02 name-error wording) remain open in `15-REVIEW-DISPOSITION.md` as non-blocking quality follow-ups.

---

_Verified: 2026-10-08T15:05:38Z_
_Verifier: the agent (gsd-verifier)_

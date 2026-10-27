---
phase: 15-provider-foundation-core-cache-semantics
verified: 2026-10-27T11:47:23Z
status: passed
score: 18/18 must-haves verified
covered_files:
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-01-PLAN.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-01-SUMMARY.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-02-PLAN.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-02-SUMMARY.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-03-PLAN.md
  - .planning/phases/15-provider-foundation-core-cache-semantics/15-03-SUMMARY.md
  - README.md
  - cache/README.md
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
  - go.mod
  - go.sum
covered_digest: "v3:sha256:c83357b366bcd269b7b1e91b18095375e7a11507c81414dd1478aea99309cdaf"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 18/19
  gaps_closed:
    - "Concurrent first-open of a fresh database no longer leaves a permanently broken handle — the Phase 16 bounded busy-only retry (cache/sqlite/retry.go) closes DI-15-01 / WR-01; 0/30 isolated runs failed."
  gaps_remaining: []
  regressions: []
---

# Phase 15: Provider Foundation + Core Cache Semantics Verification Report

**Phase Goal:** A working, durable `cache/sqlite` provider — `sqlite.New[K, V]` constructs a `cache.BatchCache[K, V]` in all three location modes; Get/Set/Delete/GetOrSet/Close match the five existing providers' contracts; TTL expiry and sweeps are enforced; WAL pragmas are verified on every pooled connection; and long-running storage controls are available.
**Verified:** 2026-10-27T11:47:23Z
**Status:** passed
**Re-verification:** Yes — regenerated because downstream phases (16, 17, 17.1, v1.8 closeout) changed files this report covered. Verdict re-derived from the codebase; the previous verdict was not carried forward.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | `sqlite.New[K, V]` returns a real `cache.BatchCache[K, V]` built through `cache.NewConcreteCache` on the 7-method provider seam, in all three location modes (roadmap SC1, PROV-01) | ✓ VERIFIED | `cache/sqlite/sqlite.go:67-83` (`New` → `&batchCache{BatchCache: cache.NewConcreteCache(c), provider: c}`); `cache/concrete_cache.go:14-26` consumes `GetFunc/SetFunc/DeleteFunc/CloseFunc/MGetFunc/MSetFunc/MDelFunc` structurally (compile-enforced); `TestMemoryRoundTrip`, `TestFileCRUD`, `TestNewWithNameDurability` pass |
| 2 | Deterministic location precedence (memory > path > name > zero=memory); `WithName` resolves to `os.UserCacheDir()/<name>/cache.db` with the directory created 0700; explicit path accepted as given (SC1, STOR-01) | ✓ VERIFIED | `cache/sqlite/dsn.go:43-67`; `TestLocation` 8-row matrix (incl. injected `userCacheDir` error); `TestNewWithNameDurability` asserts file existence, 0700 (non-Windows) and cross-reopen data; `TestFilePathWithSpaces` (nested dir creation) |
| 3 | Pool pinned to exactly one connection in both modes; independently constructed memory caches never share entries (STOR-02) | ✓ VERIFIED | `cache/sqlite/sqlite.go:105-108` (`SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, zero lifetimes); `TestPoolPinned` (memory + file `Stats().MaxOpenConnections == 1`); `TestMemoryIsolation` |
| 4 | File-mode read-back proves `journal_mode=wal`, `busy_timeout=5000`, `synchronous=1`; memory DSN omits `journal_mode`; pragmas ride the DSN so every physical connection receives them (SC5, STOR-03) | ✓ VERIFIED | `cache/sqlite/dsn.go:10-35` (constant suffixes), `:94-109`; `TestPragmaReadBack` (file + memory); `TestDSNBuildDSN` exact-string equality and `NotContains("journal_mode")` for memory; `TestBusyTimeoutDSNSync` pins retry budget to the DSN literal |
| 5 | Schema bootstrap idempotent under `BEGIN IMMEDIATE`; fixed three-column schema; sequential reopen and a 4-goroutine concurrent first-open both leave exactly one table with all handles usable (SC3, STOR-04) | ✓ VERIFIED | `cache/sqlite/sqlite.go:162-178`; `cache/sqlite/schema.go:10-22`; `TestBootstrapIdempotence` both legs pass; isolated stress re-run **0/30 failures** (`TestBootstrapIdempotence/concurrent_first_open_all_usable`). The previously deferred WAL-conversion race is fixed by the Phase 16 bounded busy-only retry (`cache/sqlite/retry.go:43-75`, invoked at `sqlite.go:118`) |
| 6 | Clean shutdown: `Close` closes the DB (last connection checkpoints WAL and removes `-wal`/`-shm`); the database file remains; reopen preserves unexpired entries and stays writable (STOR-05) | ✓ VERIFIED | `cache/sqlite/sqlite.go:276-289`; `TestSidecarRemoval` (`require.Eventually` both sidecars absent + db present); `TestReopenPersistence`; `TestExpiryAcrossRestart`; `TestCloseAfterDeferredError` (nil on poisoned handle) |
| 7 | Paths/names containing `'?'`/`'#'` and invalid names are rejected via `ErrInvalidPath` through the deferred error path (STOR-06) | ✓ VERIFIED | `cache/sqlite/dsn.go:48-56`, `:72-87`; `TestInvalidPathAndName` (3 bad paths, 7 bad names, valid-name matrix); `TestDeferredErr` (`Get/Set/Delete` all `errors.Is(ErrInvalidPath)`, `Close` nil); `TestDeferredErrorPrefix` |
| 8 | `Optimizable` type-assertable on `New`'s return; `Checkpoint` = `PRAGMA wal_checkpoint(TRUNCATE)` with 3-column scan and busy≠0 error; `Vacuum` = full VACUUM; memory checkpoint valid no-op; closed/deferred states wrap sentinels (SC5, STOR-07) | ✓ VERIFIED | `cache/sqlite/optimize.go:20-96`, `sqlite.go:79-82`; `TestOptimizable`, `TestOptimizableMemory`, `TestCheckpointBlocked` (busy error), `TestOptimizableClosed`, `TestOptimizableDeferredErr`, `TestCheckpointColumns` |
| 9 | `WithAutoCheckpoint(pages)`: >0 appends `_pragma=wal_autocheckpoint(<int>)` to the file-mode DSN only; read-back shows the value; zero keeps driver default 1000; memory ignores it (STOR-07) | ✓ VERIFIED | `cache/sqlite/options.go:72-79`, `dsn.go:100-102`, `sqlite.go:100`; `TestAutoCheckpointReadBack` (200 / default 1000 / memory-ignored); `TestDSNBuildDSN` append cases. The shipped DSN also bounds passive WAL growth with `journal_size_limit` (`dsn.go:25,34,104-106`; `TestJournalSizeLimitReadBack`) — an additive storage control, not a conflict |
| 10 | TTL resolution matches the postgres contract: per-call > 0 wins, else default > 0, else no expiry; negative/multiple inputs follow the `> 0` predicate (TTL-01) | ✓ VERIFIED | `cache/sqlite/sqlite.go:293-303`; `TestResolveTTL` 7-row `ttlMatrix` incl. expiry-window assertions |
| 11 | Expiry stored as absolute UnixNano (`expires_at INTEGER`, NULL = never) and survives restarts (TTL-01) | ✓ VERIFIED | `cache/sqlite/sqlite.go:235-257`; `TestReopenPersistence` (no-TTL and 1h-TTL entries returned after Close + reopen); `TestExpiryAcrossRestart` |
| 12 | Expired entries are never returned (`Get` → error wrapping `cache.ErrMiss`); reads never delete rows (TTL-02) | ✓ VERIFIED | `cache/sqlite/schema.go:24-27` (expiry predicate in SELECT), `sqlite.go:215-218`; `TestTTLExpiry`, `TestDefaultTTLExpiry`, `TestNoExpiry`, `TestReadsNeverDeleteRows` (raw `COUNT(*)` stays 1 after an expired Get) |
| 13 | Best-effort sweep runs synchronously on open; expired rows reclaimed; sweep errors logged and never surfaced; partial index on `expires_at` (TTL-03) | ✓ VERIFIED | `sqlite.go:136-137`, `sweep.go:11-15`, `schema.go:19-22,51-54` (bound `time.Now().UnixNano()`, never SQL `datetime('now')`); `TestSweepOnOpen` (raw observer: stale COUNT 0, fresh COUNT 1); `TestSweepFailureSwallowed` |
| 14 | Opt-in periodic sweep (default: no sweeper at all) reclaims on ticks; `Close` cancels and waits before DB close; second Close nil (TTL-04) | ✓ VERIFIED | `options.go:62-70`, `sqlite.go:139-143,276-286`, `sweep.go:19-43` (CAS → `close(stop)` → `<-done` → `db.Close()`); `TestPeriodicSweep` (Eventually COUNT 0 with an active sweeper, double Close), `TestSweeperDisabled` (nil stop/done, no goroutine), `TestOptions_WithSweepInterval`, `TestDoubleClose` (sequential + 8-way concurrent) under `-race` |
| 15 | `GetOrSet` dedups through the shared `SingleflightGetOrSet` (fast-path Get, double-check Get, setter-once, panic recovery); the provider adds no second dedup layer (PROV-02, SC2) | ✓ VERIFIED | `cache/concrete_cache.go:60-72` → `cache/singleflight.go:35-88`; `batchCache` only embeds the shared surface (`optimize.go:37-42`); `TestGetOrSetDedup` (20 concurrent misses, setter count == 1, identical results) passes under `-race` |
| 16 | Key/value parity: `fmt.Sprint` keys, `encoding/json` values; int↔stringified, unicode, empty-string, struct roundtrips; marshal failure errors with nothing stored; corrupted row is not `ErrMiss` (PROV-03) | ✓ VERIFIED | `sqlite.go:207-230` (only `sql.ErrNoRows` → `ErrMiss`; unmarshal failure returned as itself), `:235-257`; `TestParity` (3 subtests), `TestMarshalFailure`, `TestCorruptedRowNotMiss` |
| 17 | Error taxonomy: every provider error carries the `cache/sqlite:` prefix; post-Close ops match `sqlite.ErrClosed` and `cache.ErrClosed`; `Close` idempotent; all SQL is context-aware; `Delete` of a missing key is a no-op (PROV-04, SC2) | ✓ VERIFIED | `sqlite.go:188-202` (single-wrap `check`, sentinel pass-through), `:44-51` (`ErrClosed = fmt.Errorf("cache/sqlite: %w", cache.ErrClosed)`), `:260-270` (`DeleteSQL` no-op on absent key), `:276-289`; grep confirms zero non-context SQL calls in production; `TestClosedOps`, `TestClosedErrorPrefix` (exactly one prefix, `cache/sqlite: cache: cache is closed`), `TestMissErrorPrefix`, `TestDeferredErrorPrefix`, `TestCanceledContext`, `TestDoubleClose` |
| 18 | Package slice ships: `doc.go`, runnable examples, package documentation in the package README index, pinned dependencies; repo-wide gates green with no coverage override | ✓ VERIFIED | `doc.go` (68 lines), `example_test.go` (2 Examples with `// Output:`, PASS in suite); `README.md:53` cache index row names SQLite, `cache/README.md:86` provider table row (Phase 17.1 documentation restructure — same intent, new location); `go.mod:19` `modernc.org/sqlite v1.60.1`, `go.mod:120` `modernc.org/libc v1.77.1`; `go test ./...` all packages pass; `make coverage-quick` sqlite **92.7%**, total **82.4%**, thresholds PASS with no sqlite entry in `.testcoverage-quick.yml`; lint 0 issues; CGO-free Windows build OK |

**Score:** 18/18 truths verified (0 behavior-unverified; 0 overrides)

### Deferred Items

None remaining. The single deferred item carried by the previous report (DI-15-01 / review WR-01 — concurrent first-open `SQLITE_BUSY` leaving a broken handle) is now resolved by Phase 16: `open()` retries only busy-classified connection establishment within the `busy_timeout` budget (`cache/sqlite/retry.go:43-75`, called at `sqlite.go:118`), and the regression test `TestBootstrapIdempotence/concurrent_first_open_all_usable` passed **30/30** isolated runs during this verification. It is therefore no longer a deferred item and no longer a gap.

### Advisory (New Scope, Unevidenced)

None. No new-scope anti-pattern findings were raised on this pass.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `cache/sqlite/sqlite.go` | provider type, `New`, open/bootstrap, `check`, primitive methods, `resolveTTL`, sentinels, logger | ✓ VERIFIED | 303 lines; `New` returns `cache.BatchCache`; `ErrClosed` wraps `cache.ErrClosed`; all SQL call sites context-aware; wired via `New` |
| `cache/sqlite/options.go` | `Config` + `WithName/WithPath/WithMemory/WithDefaultTTL/WithSweepInterval/WithAutoCheckpoint` | ✓ VERIFIED | 80 lines; zero-value config = memory mode |
| `cache/sqlite/dsn.go` | `resolveLocation`, `validName`, `buildDSN` (+ constant suffixes) | ✓ VERIFIED | 109 lines; pure functions fully unit-tested; DSN constant-only |
| `cache/sqlite/schema.go` | schema + statement constants; fixed three-column `WITHOUT ROWID` + partial expiry index; no `user_version` | ✓ VERIFIED | 55 lines; expiry predicate folded into SELECT; sweep predicate index-shaped |
| `cache/sqlite/sweep.go` | `sweep(ctx)` (no error return), `startSweeper`, `sweepLoop` | ✓ VERIFIED | 43 lines; stop/done contract |
| `cache/sqlite/optimize.go` | `Optimizable` interface + `batchCache` adapter + `checkpoint`/`vacuum` | ✓ VERIFIED | 96 lines; `VACUUM` confined to this file (prohibition grep) |
| `cache/sqlite/doc.go` | package documentation | ✓ VERIFIED | 68 lines; usage/precedence/TTL/persistence/WAL/multi-process/batch notes |
| `cache/sqlite/example_test.go` | runnable examples | ✓ VERIFIED | 2 Examples with deterministic `// Output:` blocks, PASS in suite |
| `cache/sqlite/{sqlite,sqlite_internal,dsn_internal,options_internal}_test.go` | behavioral/internal test suite | ✓ VERIFIED | 70 test functions + 2 Examples; package suite green; coverage 92.7% |
| `go.mod` / `go.sum` | `modernc.org/sqlite v1.60.1` + exact `modernc.org/libc v1.77.1` pin | ✓ VERIFIED | `go.mod:19/120` with cznic/sqlite#177 comment |
| `README.md` / `cache/README.md` | package index row + provider table row | ✓ VERIFIED | `README.md:53` (cache row names SQLite); `cache/README.md:86` provider row. Phase 17.1 moved provider detail from the root README to `cache/README.md` per `AGENTS.md`; intent preserved |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `New` | `resolveLocation` → `buildDSN` → `sql.Open("sqlite", dsn)` | open sequence | WIRED | `sqlite.go:88-103`; blank modernc import at `sqlite.go:16` |
| `open` | pool pin → `retryBusy(PingContext)` → `bootstrap` (`BeginTx` = BEGIN IMMEDIATE) | `_txlock=immediate` in DSN | WIRED | `sqlite.go:105-130`; `retry.go:43-75`; Phase 16 retry closes DI-15-01 |
| `New` | `cache.NewConcreteCache(c)` | structural 7-method interface | WIRED | `sqlite.go:79-82`; `concrete_cache.go:14-26`; compile-enforced |
| `GetOrSet` | `concreteCache.sf.Do` → `GetFunc`/`SetFunc` | shared singleflight | WIRED | `concrete_cache.go:60-72` → `singleflight.go:35-88`; no provider-side dedup |
| `GetFunc` | `SelectSQL` → `sql.ErrNoRows` → wrapped `cache.ErrMiss` | miss mapping | WIRED | `sqlite.go:215-218` |
| `open` | `c.sweep(ctx)` (synchronous, best-effort) | before `New` returns | WIRED | `sqlite.go:136-137` → `sweep.go:11-15` |
| `WithSweepInterval > 0` | `startSweeper` → `sweepLoop` → ticker → `sweep` | opt-in goroutine | WIRED | `sqlite.go:139-143`; `sweep.go:19-43` |
| `Close` | `close(stop)` → `<-done` → `db.Close()` | cancel-and-join before DB close | WIRED | `sqlite.go:276-289`; no `os.Remove` in production code |
| `New` | `&batchCache{BatchCache, provider}` | `Optimizable` assertion target | WIRED | `sqlite.go:79-82` → `optimize.go:37-60` |
| `Checkpoint` | `check()` → `PRAGMA wal_checkpoint(TRUNCATE)` → 3-col scan → busy≠0 error | maintenance state machine | WIRED | `optimize.go:66-82`; `TestCheckpointColumns`/`TestCheckpointBlocked` |
| `WithAutoCheckpoint` | `Config.AutoCheckpoint` → `buildDSN` → `_pragma=wal_autocheckpoint(<int>)` | DSN append | WIRED | `options.go:76-79` → `dsn.go:100-102` → `sqlite.go:100`; read-back test |
| 7-method seam | `MGetFunc/MSetFunc/MDelFunc` | `batch.go` (Phase 16) | WIRED | `batch.go:78-241` replaced the Phase 15 loop placeholders; `cache/concrete_cache.go:129-147` delegates; `TestBatchMGet/MSetOneTTL/MDel/Semantics` pass |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `sqlite.go:GetFunc` | `data` | `SelectSQL` (`QueryRowContext`) against `cache_entries`, expiry-filtered | Yes — real SQLite query, no fallback | ✓ FLOWING |
| `sqlite.go:SetFunc` | `data`, `exp` | `json.Marshal` + `UpsertSQL` positional binds | Yes — persisted JSON; NULL when no TTL | ✓ FLOWING |
| `sweep.go:sweep` | bound now | `SweepSQL` (`ExecContext`) with `time.Now().UnixNano()` | Yes — real DELETE pass | ✓ FLOWING |
| `optimize.go:checkpoint` | `busy/logFrames/checkpointed` | `PRAGMA wal_checkpoint(TRUNCATE)` three-column result | Yes — engine result, busy surfaced | ✓ FLOWING |
| `dsn.go:buildDSN` | DSN string | `resolveLocation` output + `strconv.Itoa`/`FormatInt` | Yes — read back via PRAGMAs | ✓ FLOWING |

No static returns, hardcoded fallbacks, or mock data sources found in production code.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full package suite (contracts, TTL, sweeps, lifecycle, Optimizable, batch) | `go test ./cache/sqlite/... -count=1` | `ok` (5.3s) | ✓ PASS |
| Dedup/isolation/close/sweeper/concurrency under race detector | `go test ./cache/sqlite/... -race -run 'TestGetOrSetDedup\|TestMemoryIsolation\|TestDoubleClose\|TestPeriodicSweep\|TestConcurrentBatchOpsRace\|TestTwoProcess' -count=1` | `ok` (2.3s), no data race | ✓ PASS |
| Repo-wide regression | `go test ./... -count=1` | all 26 packages ok | ✓ PASS |
| Previously deferred concurrent first-open race | `go test ./cache/sqlite/ -run 'TestBootstrapIdempotence/concurrent_first_open_all_usable' -count=1` ×30 | **0/30 failures** (was 29/40 failing at prior verification) | ✓ PASS |
| Coverage gate, no override (AGENTS.md pre-commit) | `make coverage-quick` | sqlite **92.7%**, total **82.4%**, all thresholds PASS; `.testcoverage-quick.yml` has no sqlite entry | ✓ PASS |
| Lint | `golangci-lint run ./cache/sqlite/...` | 0 issues | ✓ PASS |
| CGO-free Windows build | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cache/sqlite/` | exit 0, no output | ✓ PASS |
| Prohibition greps | `cache=shared` / `auto_vacuum` / post-open `Exec(...PRAGMA)` / `os.Remove` in production | all empty; `VACUUM` appears only in `optimize.go` | ✓ PASS |

### Probe Execution

Skipped — no probes declared for this phase (no `scripts/*/tests/probe-*.sh`; no probe script referenced by the PLANs or SUMMARYs). Verification was performed with the package test suite and targeted commands above.

🚫 SKIP REASON: no documented probes in plans or repository.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `dsn_internal_test.go` | PROV-01, STOR-01, STOR-03, STOR-06 | 3 (8 subcases + 11 name cases) | 0 | No — expected DSNs are literal strings, location paths derived independently | Value (exact equality) | OK |
| `sqlite_internal_test.go` | STOR-02/03/04/05, TTL-01..04 | 18 | 0 | No — raw SQL observers read the database independently of the provider | Value + Behavioral (raw `COUNT(*)`, PRAGMA read-back) | OK |
| `sqlite_test.go` | PROV-01..04, STOR-01/05/06/07, TTL-01..04 | 26 (many subcases) | 0 | No — black-box through the public API; expectations are literal values | Value + Behavioral (reopen, sidecar removal, contention, dedup) | OK |
| `options_internal_test.go` | STOR-01/02, STOR-07, TTL-04 | 5 | 0 | No | Value | OK |
| `example_test.go` | QUAL slice (plan 15-03) | 2 Examples | 0 | No — `// Output:` asserted by `go test` | Behavioral | OK |

**Disabled tests on requirements:** 0
**Circular patterns detected:** 0
**Insufficient assertions:** 0 — every requirement-linked test asserts values or multi-step behavior, not mere existence.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (12/12; `check.decision-coverage-verify` — `not_honored: []`.)

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| PROV-01 | 15-01 | `New` returns `cache.BatchCache` on the `cacher` seam via `NewConcreteCache`, matching the five providers | ✓ SATISFIED | Truths 1, 18; `sqlite.go:67-83`; suite green |
| PROV-02 | 15-01 | `GetOrSet` uses shared `SingleflightGetOrSet`; no second dedup layer | ✓ SATISFIED | Truth 15; `TestGetOrSetDedup` (race) |
| PROV-03 | 15-01 | `fmt.Sprint` keys, JSON values, misses wrap `cache.ErrMiss` | ✓ SATISFIED | Truth 16; `TestParity`, `TestMarshalFailure`, `TestCorruptedRowNotMiss` |
| PROV-04 | 15-01 | Context-aware SQL; Delete-missing no-op; idempotent Close; `cache/sqlite:` prefix | ✓ SATISFIED | Truth 17; zero non-context SQL sites; `TestClosedOps`, `TestClosedErrorPrefix`, `TestCanceledContext` |
| STOR-01 | 15-01 | Default `os.UserCacheDir()/<name>/` (MkdirAll 0700); explicit path as given | ✓ SATISFIED | Truth 2; `TestNewWithNameDurability`; `TestFilePathWithSpaces` |
| STOR-02 | 15-01 | Empty path = `:memory:`; pool pinned 1/1 both modes | ✓ SATISFIED | Truths 2, 3; `TestPoolPinned`, `TestMemoryIsolation` |
| STOR-03 | 15-01 | DSN-carried pragmas (WAL/busy_timeout/synchronous/immediate); read-back asserts WAL | ✓ SATISFIED | Truth 4; `TestPragmaReadBack`, `TestDSNBuildDSN` |
| STOR-04 | 15-01 | Idempotent `BEGIN IMMEDIATE` bootstrap; fixed three-column schema | ✓ SATISFIED | Truth 5; both `TestBootstrapIdempotence` legs pass, concurrent leg 0/30 failures |
| STOR-05 | 15-02 | Clean Close (final checkpoint; sidecars removed); reopen preserves entries | ✓ SATISFIED | Truth 6; `TestSidecarRemoval`, `TestReopenPersistence` |
| STOR-06 | 15-01 | `'?'`/`'#'` rejected so paths cannot inject DSN parameters | ✓ SATISFIED | Truth 7; `TestInvalidPathAndName`; DSN built only from validated path + constant suffixes |
| STOR-07 | 15-03 | Tune `wal_autocheckpoint`; explicit optimize (checkpoint and/or VACUUM) | ✓ SATISFIED | Truths 8, 9; `TestOptimizable*`, `TestAutoCheckpointReadBack`, `TestCheckpointBlocked`, `TestJournalSizeLimitReadBack` |
| TTL-01 | 15-02 | TTL resolution parity; absolute UnixNano expiry survives restarts | ✓ SATISFIED | Truths 10, 11; `TestResolveTTL`, `TestReopenPersistence` |
| TTL-02 | 15-02 | Expired never returned (`ErrMiss`); reads never delete | ✓ SATISFIED | Truth 12; `TestReadsNeverDeleteRows` |
| TTL-03 | 15-02 | Best-effort open sweep; partial index on `expires_at` | ✓ SATISFIED | Truth 13; `TestSweepOnOpen`; `CreateIndexSQL` partial index |
| TTL-04 | 15-02 | Optional periodic sweep interval; unset → open sweep + read filtering only | ✓ SATISFIED | Truth 14; `TestPeriodicSweep`, `TestSweeperDisabled` |

**Orphaned requirements:** none — the PLAN frontmatter covers exactly the 15 IDs ROADMAP/REQUIREMENTS map to Phase 15 (PROV-01..04, STOR-01..07, TTL-01..04); no Phase 15 requirement is unclaimed.

### Downstream Supersessions (not contradictions)

Downstream phases legitimately changed details this phase originally decided. The requirements remain satisfied by the new arrangement:

| Phase 15 decision / artifact | Shipped arrangement | Requirement still satisfied |
| --- | --- | --- |
| 15-D-12: exported `ErrClosed`/`ErrInvalidPath` each carry a `cache/sqlite:` prefix, with `check()` wrapping bare errors | Phase 17.1 changed `ErrClosed` to wrap `cache.ErrClosed` and made `check()` stop double-wrapping. Message is `cache/sqlite: cache: cache is closed`; `errors.Is` matches both `sqlite.ErrClosed` and `cache.ErrClosed`; exactly one `cache/sqlite: ` delimiter (`TestClosedErrorPrefix`, `TestMissErrorPrefix`, `TestDeferredErrorPrefix`) | **Yes** — PROV-04's "provider errors carry a `cache/sqlite:` prefix" and "post-Close ops match `ErrClosed`" both hold; single-prefix is stricter than the original |
| Batch methods were loop-based placeholders with `Phase 16:` markers | Phase 16 shipped chunked `MGet`/`MDel` and transactional `MSet` in `cache/sqlite/batch.go` | **Yes** — PROV-01 only requires the 7-method seam; `BatchCache` is fully implemented and tested |
| Root `README.md` carried the package row and provider details | Phase 17.1 per `AGENTS.md` kept the root README as an index (`README.md:53` names SQLite) and moved provider detail to `cache/README.md:86` | **Yes** — documentation intent preserved in the canonical location |
| The concurrent first-open stress was deferred to Phase 16 (DI-15-01) | Phase 16 added a bounded busy-only connection-establishment retry | **Yes** — now verified in-phase (0/30 failures); no deferred item remains |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers anywhere in `cache/sqlite/` | — | Debt-marker gate clean |
| `cache/sqlite/optimize.go` | 78 | Blocked-checkpoint error is a bare `fmt.Errorf("cache/sqlite: checkpoint blocked (busy=%d)")` without a wrapped sentinel | ℹ️ Info | Matches the STOR-07 contract (descriptive error, never silent nil); no `errors.Is` target was required |
| `cache/sqlite/dsn.go` | 55 | Invalid cache names use `ErrInvalidPath` with "invalid cache name" wording (review IN-02) | ℹ️ Info | Contract-compliant; a dedicated sentinel was only a suggestion, never a requirement |
| `.planning/.../15-REVIEW-DISPOSITION.md` | — | Review dispositions still list WR-01 and IN-01 as `open` although the code now fixes both (Phase 16 retry; Phase 17.1 single prefix) | ℹ️ Info | Documentation staleness only — the code and tests prove resolution; no runtime impact |

No 🛑 Blockers: all prohibition greps clean (`cache=shared`, post-open PRAGMA exec, sidecar deletion, `auto_vacuum`, SQL `datetime()`, `VACUUM` locality).

### Human Verification Required

N/A — infrastructure/foundation phase with no user-facing elements. All 18 truth areas are machine-verified by the package suite (70 tests + 2 Examples) plus targeted commands re-run during this verification. No truth was left behavior-unverified, so no human-verification item is emitted.

### Gaps Summary

No gaps. The phase goal is achieved in the shipped code and reproduced by execution: the constructor builds a real `cache.BatchCache` through `NewConcreteCache` in all three location modes (memory, explicit path, and `WithName` with 0700 directory creation); the primitive CRUD/GetOrSet/Close contracts match the existing providers (parity, misses, single-prefix error taxonomy, shared singleflight dedup); TTL resolution, absolute-expiry storage, filter-only reads, open sweep, and the opt-in periodic sweep are enforced; DSN-carried WAL pragmas are read back on the pinned connection; and the long-running controls (`Optimizable` checkpoint/vacuum, `WithAutoCheckpoint`, plus the additive `journal_size_limit` bound) are type-assertable and read-back-proven.

The only previously deferred robustness item — the concurrent first-open `SQLITE_BUSY` race (DI-15-01 / WR-01) — was closed by Phase 16 and re-measured here at 0/30 failures, so it no longer appears as a gap or a deferral. Downstream phases 16, 17, 17.1 and the v1.8 closeout changed covered files without breaking any Phase 15 requirement; where they superseded a Phase 15 implementation decision (notably the 17.1 `ErrClosed` arrangement), the requirement is still satisfied and is recorded under Downstream Supersessions rather than as a contradiction. Repo-wide tests, coverage gates (sqlite 92.7%, total 82.4%), lint, and the CGO-free Windows build are all green.

---

_Verified: 2026-10-27T11:47:23Z_
_Verifier: the agent (gsd-verifier)_

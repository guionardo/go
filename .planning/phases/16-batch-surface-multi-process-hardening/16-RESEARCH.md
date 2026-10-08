# Phase 16: Batch Surface + Multi-Process Hardening - Research

**Researched:** 2026-10-08
**Domain:** `cache/sqlite` batch surface (chunked IN queries, transactional prepared upsert) + multi-process WAL hardening (bounded open retry for the DI-15-01 WAL-conversion race, two-process proof via helper-process E2E) on the pinned pure-Go `modernc.org/sqlite` v1.60.1 driver
**Confidence:** HIGH for driver/batch mechanics — every mechanical claim this phase depends on was verified this session with a live probe against the pinned driver (including a two-OS-process probe); HIGH for integration seams (direct code reads with line ranges); MEDIUM for cross-OS transfer of the probe results (darwin/arm64 → linux/windows CI is D-09's job, already locked).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Batch Mechanical Details
- **D-01:** MGet/MDel chunk size is a **fixed internal constant of 100 keys per chunk** — no `WithChunkSize` option, no new API surface. Values far below SQLite's default variable limit; adjustable later without breaking callers. Replaces the open "chunk size" item from STATE.md (v1.8 planning).
- **D-02:** MGet/MDel inputs are **deduplicated before chunking**; an empty key list is a **no-op** (MGet returns an empty map without issuing any query; MDel returns nil) — **Reversibility:** reversible.
- **D-03:** MGet error policy — `cache.BatchCache.MGet` returns `map[K]V` with no error channel, so a failed chunk is **logged (slog warn) and skipped**; remaining chunks still run; found-so-far is returned. Undecodable entries stay silently skipped per BATCH-01.
- **D-04:** MSet runs as **one `BEGIN IMMEDIATE`…`COMMIT` transaction** with the single prepared upsert; all values are **marshaled up-front before the first write**, and ANY error (marshal, SQL, context) rolls back the whole batch (BATCH-02 atomicity). One TTL applies to the batch (per `BatchCache.MSet` signature).
- **D-05:** MDel chunk failures return the wrapped error (partial delete is acceptable — MDel is idempotent; caller may retry).

#### Contention, Retry & DI-15-01
- **D-06:** DI-15-01 fix: **bounded open retry** — `New` retries the fresh-file open when the per-connection `PRAGMA journal_mode=WAL` conversion hits immediate `SQLITE_BUSY` (busy handler bypassed), bounded by `busy_timeout` with a tiny backoff. `New` stays never-failing: ops defer the wrapped error if retries exhaust. The concurrent-first-open test stays and must be deterministic green (~72% repro evidence from Phase 15 verification).
- **D-07:** Beyond the open window, **no provider-side retries**: ordinary BEGIN contention surfaces as the bounded `busy_timeout` error (`_txlock=immediate`) — milestone's no-retry-middleware decision holds; Checkpoint/Vacuum get no extra retry either.
- **D-08:** **`journal_size_limit` set to 64 MB via DSN pragma** (`_pragma=journal_size_limit(67108864)` in file mode) to bound passive WAL growth between checkpoints in long-running processes — **Reversibility:** costly — silently changes behavior for existing cache files if later adjusted; zero-value/disabled path keeps SQLite's unlimited default, no sentinel semantics (mirrors `WithAutoCheckpoint`).

#### Two-Process Proof & CI
- **D-09:** The CONC-05 two-process E2E (helper-process pattern: test binary re-executes itself via env flag) runs on **all three CI OSes** (ubuntu/macos/windows) — Windows WAL sidecar/handle semantics get the real proof.
- **D-10:** E2E covers **contention + crash recovery**: (A) both processes concurrently Set/Get/Delete (+ a batch op) asserting no corruption and correct values — the CONC-01 exit criterion — and (B) kill-mid-write: one process dies while the other holds a write lock; the survivor continues and the file reopens cleanly.
- **D-11:** The two-process spike (folded todo below) runs first as the phase exit criterion; its results confirm the retry policy before the E2E hardens it.

### the agent's Discretion
- MGet result ordering (map return — order follows query rows, no guarantee needed).
- Spike test fixture details (scenario timing, iteration counts, helper-process plumbing) and how the flake-quarantine is disposed once D-06 lands.
- Whether `WithAutoCheckpoint`/`journal_size_limit` interact (both DSN pragmas; keep additive).

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. (Reviewed Todos not folded: none — the single matching todo, the two-process WAL spike, was folded as D-11.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BATCH-01 | MGet via chunked IN queries; missing/expired/undecodable silently skipped; bounded chunk size | Chunked-IN SQL shape (Finding 2, Pattern 2); 100+1 bound params probe-verified; decoded-error log level (debug) per CONTEXT specifics; dedup/no-op (D-02) |
| BATCH-02 | MSet single transaction, one prepared upsert, batch atomic, one TTL | Prepared-upsert transaction seam (Pattern 3, probe-verified 100 execs/tx); up-front marshal (D-04) makes marshal errors atomic before the tx opens; no-partial-writes invariant test (Pitfall 4) |
| BATCH-03 | MDel via chunked IN deletes; idempotent for missing | Pattern 2; fail-fast wrapped error per D-05 |
| CONC-01 | Two processes share one file without corruption/hard failure; spike = exit criterion | **Probe: two OS processes × 300 iterations incl. batch txs: integrity=ok, correct counts**; raw first-open race 16/30 runs cross-process, 0/30 with bounded retry (Finding 1, PROBE8); spike design (Pattern 5) |
| CONC-02 | Immediate lock mode; contention surfaces at BEGIN; bounded by busy_timeout, no unbounded auto-retry | Probe: BEGIN IMMEDIATE blocked → SQLITE_BUSY (5) after exactly busy_timeout (5.05s / 309ms), typed `*sqlite.Error`, Code()&0xff==5; retry only in the open window (D-06/D-07, Finding 1) |
| CONC-03 | `go test -race` clean under concurrent goroutines on the pinned pool | Race strategy (Pattern 6); CI race step on ubuntu recommended; existing 15 race subset is green precedent |
| CONC-04 | Package docs: WAL local-storage only, same-host sharing | **Already substantially shipped in doc.go:37-38** ("WAL requires local storage: network filesystems and cross-host sharing are unsupported"); phase adds the multi-process contention bullet |
| CONC-05 | CI E2E: two OS processes, one temp DB, no corruption, correct behavior | Full harness design (Pattern 5, Code Example 5); CI integration (Pattern 7); crash arm probe-verified 10/10 (Finding 1, PROBE9) |
</phase_requirements>

## Summary

Phase 16 finishes the `BatchCache` surface on `cache/sqlite` (replacing the Phase 15 loop placeholders at `sqlite.go:263-305`) and proves the milestone headline: two OS processes sharing one cache file under WAL with contention bounded by `busy_timeout`. This phase's research resolved the milestone's **only remaining LOW-confidence item** — two-process `SQLITE_BUSY` behavior under the locked pragma set — with primary evidence: a throwaway probe against the pinned driver (in-process mechanics) and a **helper-process two-process probe** (real OS processes, re-exec pattern) that reproduce both the DI-15-01 first-open race cross-process and its fix.

**Primary finding — The DI-15-01 fix is a hard requirement for CONC-01, not a test-cosmetic.** The probe's two-process runs on a fresh file failed **16 of 30 runs** (~53%: one of the two helper processes hit `database is locked (5) (SQLITE_BUSY)` at first use — the WAL-conversion race, busy handler bypassed). With a bounded connect retry (5 ms backoff, 5 s budget, busy-only classification), the same harness ran **30/30 clean** (300 point writes + 6×20 batch transactions per process, `PRAGMA integrity_check` = `ok`, 420 rows per namespace). The crash arm (kill-mid-write while holding `BEGIN IMMEDIATE`) ran **10/10**: the survivor completed its loop, the killed process's uncommitted row was rolled back, and the file reopened with `integrity_check = ok`. These results are the strongest possible input to the planner: D-06's retry is not an optional resilience nicety — without it CONC-01/CONC-05 fail.

The batch mechanics themselves are standard and fully verified: chunked `IN` SELECT/DELETE with 100 keys + the `now` bound = 101 params (driver limit 32766 — verified in the bundled lib), a single prepared upsert inside one `BEGIN IMMEDIATE` transaction (100-exec tx probe-verified; contention at BEGIN probe-verified at exactly `busy_timeout`), and `_pragma=journal_size_limit(67108864)` co-existing with `_pragma=wal_autocheckpoint(500)` and reading back exactly (D-08's DSN append is a string-shape mirror of the existing autocheckpoint append — no new mechanism, no driver change).

**Primary recommendation:** Plan in four commits in dependency order — (1) **spike** (D-11): a scratch two-process harness proving the retry policy on the shipped provider and producing the 16-SPIKE-FINDINGS evidence record; (2) **provider hardening**: D-06 bounded connect retry (`retryBusy` + `isBusyError` pure helpers) + D-08 `journal_size_limit` DSN append, re-strengthening `TestBootstrapIdempotence/concurrent_first_open_all_usable` to deterministic green; (3) **batch surface**: `batch.go` with real `MGetFunc`/`MSetFunc`/`MDelFunc` + `batch_internal_test.go` and black-box tests; (4) **two-process E2E + CI + race + docs**: the hardened helper-process harness (contention + crash arms) wired into the 3-OS CI matrix, a race step, CONC-04 doc bullet. Every commit must pass `make coverage-quick` (no `cache/sqlite` override — file ≥70%, package ≥80%); the e2e-tagged harness stays out of the unit lane per Phase 15 Pitfall 13.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Batch SQL + chunking + dedup | `cache/sqlite` (provider) | SQLite engine | Provider owns the chunked-IN shape, `map[string]K` re-keying, and D-02 dedup; engine executes the parameterized statements |
| In-process serialization | `database/sql` pool config (1 conn, pinned in Phase 15) | SQLite engine | Same as Phase 15: `MaxOpenConns(1)` is the whole in-process model; batch ops must not change it |
| Cross-process serialization | SQLite engine (WAL + busy handler) | `cache/sqlite` (DSN pragmas, bounded open retry) | WAL + `busy_timeout` + `_txlock=immediate` are the cross-process contract; the provider only sets them via DSN and retries the open window (D-06) |
| First-open WAL conversion race | `cache/sqlite` (`open()`) | SQLite engine | D-06 bounded connect retry is provider-owned; engine's busy handler is bypassed for the conversion (probe + DI-15-01) |
| Transaction atomicity (MSet) | `cache/sqlite` (`sql.Tx` via pinned conn) | SQLite engine | One `BeginTx` + prepared stmt + `Commit`/`Rollback`; pre-marshal makes the marshal-error class atomic before the tx opens |
| Error/log policy (MGet warn+skip, decode debug) | `cache/sqlite` | — | Frozen `MGet` signature has no error channel (D-03); logging is the designed surface |
| Concurrency proof (CONC-01/03/05) | Tests: helper-process E2E + `-race` | CI matrix (3 OSes, D-09) | Evidence lives in the e2e-tagged harness and the race run; provider supplies hooks, tests supply the proof |
| WAL constraints documentation (CONC-04) | `cache/sqlite` `doc.go` | — | Substantially shipped (doc.go:37-38); phase adds the multi-process bullet |

## Critical Planning Findings

### Finding 1 — Two-process probe: DI-15-01 is a real cross-process failure; bounded connect retry fixes it 100% (PRIMARY EVIDENCE)

Throwaway probe, pinned `modernc.org/sqlite` v1.60.1, darwin/arm64, Go 1.27.0; helper-process re-exec harness; fresh temp DB per run; two children × 300 iterations each (point upserts + reads + 6× 20-key prepared-upsert transactions). **Raw** (production DSN, no retry): **16/30 runs failed** — exactly one child per failure hit `database is locked (5) (SQLITE_BUSY)` at its first statement (DDL on a fresh file), the same busy-handler-bypass as Phase 15's in-process evidence (`sqlite_internal_test.go:442`). **With a busy-only connect retry (5 ms backoff, 5 s budget)** before the first statement: **30/30 clean** — `integrity_check=ok`, `journal_mode=wal`, per-namespace row counts exact (420 each), no busy surfaced. Crash arm (child killed via `Process.Kill()` while holding an open `BEGIN IMMEDIATE` write tx with an uncommitted row, marker-file synchronized): **10/10** — survivor (blocked on the busy handler at kill time) completed its loop, uncommitted row absent after reopen, `integrity_check=ok`.

**Planning consequence:** D-06's retry is the gate for CONC-01/CONC-05, not a test-only fix. The retry boundary is precise: the failure surfaces at **connection establishment (`PingContext`)**, never at bootstrap DDL that follows a successful ping (0/240 DDL failures across both arms of the in-process probe). So the retry loop wraps the first `PingContext` in `open()`; the bootstrap's `BEGIN IMMEDIATE` needs no retry because the busy handler covers it (Finding 3). Same-process concurrent first-open measured 14/60 iterations failing raw (4 workers each) vs **0 non-busy failures with retry (8 retries used across 240 attempts)** — the regression test at `sqlite_internal_test.go:442` becomes deterministic green for free.

### Finding 2 — The concrete chunked-IN SQL shape (SchemaSQL style, probe-verified at 101 params)

The chunked `MGet` query is a *dynamic-placeholder* `IN` list whose structure is a constant template; only `?` markers vary by chunk size (values stay bound). Verified live at 100 keys + 1 `now` bound = 101 params → 100 rows returned (`SQLITE_MAX_VARIABLE_NUMBER = 32766` in the bundled lib `[VERIFIED: modernc.org/sqlite v1.60.1 lib/sqlite.go:4014]` — chunk 100 is 0.3% of headroom). Full SQL shapes in Pattern 2. The `IN` template already carries the expiry predicate, so expired rows are absent by SQL — no code-level skip needed for them; `cache_key` is selected so rows can be re-keyed to `K` via a per-chunk `map[string]K` (keys are `fmt.Sprint`ed, which is not invertible — the original `K` must be recovered from the chunk's input list).

### Finding 3 — BEGIN IMMEDIATE contention is bounded and typed — the `isBusyError` classifier shape

`_txlock=immediate` → the driver emits `begin immediate` (`[VERIFIED: modernc.org/sqlite v1.60.1 tx.go:20-24]`). Probe: holding a write tx on one connection, a second `BeginTx` on another connection (same DSN) returns `database is locked (5) (SQLITE_BUSY)` after **exactly busy_timeout** (5.05 s at 5000; 309 ms at 300) — the busy handler IS consulted for BEGIN. The error is `*modernc/sqlite.Error` with `Code() == 5` (`[VERIFIED: error.go:12-21]`), and the driver enables extended result codes per connection (`[VERIFIED: conn.go:113]`), so extended busy variants (`SQLITE_BUSY_SNAPSHOT` = 517, etc.) exist; the classifier must compare the **primary code**: `errors.As(err, &coded)` where `coded interface{ Code() int }`, then `c().Code()&0xff == 5`. Using the interface (not the concrete `*sqlite.Error`) keeps the classifier unit-testable without importing the driver — a test fake with `Code() int` drives the pure helper, and a behavioral subtest reproduces real contention. NOTE the import collision gotcha in Code Example 2b (`package sqlite` importing `"modernc.org/sqlite"` — use the interface approach and keep the blank import).

Also probe-verified for CONC-02's test design: while a writer holds an open write tx, a second connection reads committed rows immediately (196 µs) and never sees the uncommitted row — WAL writers don't block readers, the milestone's "reads never delete rows / no lock churn" story holds under contention.

### Finding 4 — `journal_size_limit` via `_pragma` works and co-exists with `wal_autocheckpoint` (D-08)

Probe: DSN `...&_pragma=journal_size_limit(67108864)&_pragma=wal_autocheckpoint(500)` reads back **67108864 / 500** on the same connection — two `_pragma` keys coexist; the driver executes them verbatim, lexicographically sorted (busy_timeout first) `[VERIFIED: modernc.org/sqlite v1.60.1 sqlite.go:434-460]`. SQLite semantics (sqlite.org/pragma.html): "Each time a transaction is committed or a WAL file resets, SQLite compares the size of the WAL file left in the file-system to the size limit... if larger it is truncated to the limit" — i.e. it bounds post-checkpoint WAL file size, which neither autocheckpoint frequency nor TRUNCATE checkpoints change by default. The implementation is a string-shape mirror of `dsnAutoCheckpointPrefix` (`dsn.go:20`): a new `_pragma=journal_size_limit(<strconv.Itoa>)` append in `buildDSN` (`dsn.go:82-93`); the value is a compile-time constant, never caller input (existing A6 hygiene). D-08's "zero-value/disabled path" = `buildDSN` appends only when the param is positive (exactly the `WithAutoCheckpoint` param contract, `dsn.go:87-91`); see Open Question 1 for the option-vs-constant reading, though CONTEXT's "none planned per D-01/D-08 defaults" (`16-CONTEXT.md` code_context) fixes the shipped value as a hard default.

### Finding 5 — `WITHOUT ROWID` + TEXT KEY: the chunked MGet needs `SELECT cache_key, value` (both columns)

Because `cache_key TEXT PRIMARY KEY` on a `WITHOUT ROWID` table is the row identity, the chunked SELECT must return `cache_key` alongside `value` to map rows back to `K`; there is no rowid shortcut. This is the one deviation from the single-key `SelectSQL` (`schema.go:26-27`) which selects `value` only.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `modernc.org/sqlite` | **v1.60.1 (already pinned)** | `database/sql` driver; provides `*sqlite.Error` with `Code()` | Milestone-locked in Phase 15; **no new dependency** this phase — the classifier uses `errors.As` against an interface, so even the import stays blank |
| `database/sql` (stdlib) | Go 1.26 | `sql.Tx` prepared upsert, context query/exec | The batch seam is `tx.PrepareContext` + `tx.ExecContext` + `Commit`/`Rollback`; pool stays pinned (Phase 15) |
| `encoding/json` (stdlib) | Go 1.26 | Up-front marshaling (D-04) | Same JSON parity as `SetFunc` (`sqlite.go:212-226`) |
| `fmt.Sprint` (stdlib) | Go 1.26 | Key stringification + chunk de-keying | Invertible only via the chunk's `map[string]K` — the re-keying design depends on it |
| `strings` (stdlib) | Go 1.26 | `strings.Repeat` for `IN` placeholders | No sprintf of user data; only `?` markers are repeated (`strings.Repeat("?,", n-1)+"?"`) |
| `os/exec` (stdlib) | Go 1.26 | Helper-process re-exec (`exec.Command(os.Args[0])`) | The CONC-05 harness; same pattern Go's own stdlib tests use (`os/exec` `TestMain`, github.com/golang/go `src/os/exec/exec_test.go`) |
| `testing` `TestMain` + env flag | Go 1.26 | Child-process entry point | Standard helper-process pattern; child never enters `m.Run()` when the env flag is set |
| `log/slog` (stdlib) | Go 1.26 | MGet chunk warn / decode debug | Existing `logger()` (`sqlite.go:51-53`); no new logging surface |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.11.1 (existing) | Assertions in batch/retry/E2E tests | Unchanged |
| `golangci-lint` | installed | `contextcheck` on new helpers; existing gates | The retry loop must not introduce goroutine/context lint issues |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Chunked `IN` per 100 keys | One `IN` for the whole batch | 32766-var limit caps total; a 10k-key MGet would exceed it — chunking is the locked contract (D-01) |
| `tx.PrepareContext` + loop | `tx.ExecContext` per pair | Both work (probe); `PrepareContext` is the letter of BATCH-02 ("one prepared upsert") and avoids per-call prepare cost |
| Interface-based busy classifier | `errors.As` to `*modernc.sqlite.Error` | Concrete type forces a named import inside `package sqlite` (name-collision with the driver package); interface keeps the blank import and enables fake-error unit tests |
| Ping-retry at `open()` | Retry around `bootstrap()` / whole `open` | Probe proved failures surface at connection establishment only; pinging before bootstrap is the minimal, bounded seam |
| `retryBusy` loop with `time.After` | `golang.org/x/sync` retry lib | No new dependency; the loop is 12 lines; a clock-free test injects a fake fn |
| e2e build tag for the harness | Always-on test | `coverage-quick` (unit lane) would count the harness and break package thresholds; Phase 15 Pitfall 13 precedent (`cache/cache_e2e_test.go` uses `//go:build e2e`) |

**Installation:** none — no new packages this phase (`go.mod` untouched).

## Package Legitimacy Audit

> This phase installs **no external packages**: the driver is pinned from Phase 15, `database/sql`/`os/exec`/`testing` are stdlib, and the E2E harness needs no Docker. The `package-legitimacy check` seam covers npm/pypi/crates only and is not applicable to a Go phase with zero new modules. No `[ASSUMED]` package names appear anywhere in this research.

| Package | Registry | Verdict | Disposition |
|---------|----------|---------|-------------|
| (none — no new dependencies) | — | — | Approved |

**Packages removed due to [SLOP] verdict:** none. **Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
Consumer code (Process A)                     Consumer code (Process B)
  c := sqlite.New[...] (same file path)         c := sqlite.New[...] (same file path)
  MGet / MSet / MDel / Get/Set/Delete/GetOrSet  MGet / MSet / MDel / Get/Set/Delete/GetOrSet
        │                                             │
        ▼                                             ▼
cache/sqlite provider (both processes, per-process state)
  open(): resolveLocation → buildDSN (D-08: +journal_size_limit)
          → sql.Open → pin pool 1/1 → retryBusy(PingContext)  [D-06: NEW]
          → bootstrap (BEGIN IMMEDIATE DDL) → verifyJournalMode → sweep
  batch.go [NEW]: MGetFunc  = dedup → chunk(100) → SELECT ... IN (...) AND expiry  → warn+skip per chunk (D-03)
                  MSetFunc  = pre-marshal all → BeginTx (BEGIN IMMEDIATE) → Prepare(UpsertSQL)
                              → Exec per pair → Commit; any error → Rollback (D-04)
                  MDelFunc  = dedup → chunk(100) → DELETE ... IN (...); first error wrapped & returned (D-05)
        │                                             │
        ▼                                             ▼
   SQLite engine — one shared file, WAL mode, busy_timeout=5000, synchronous=NORMAL
   - write lock acquired at BEGIN (immediate mode) → contention = bounded SQLITE_BUSY at BEGIN
   - readers never block on writers (WAL snapshot isolation; probe-verified)
   - first-open WAL conversion: exclusive → busy handler bypassed → D-06 retry covers it
        │
        ▼
   File: <cache>.db   +  <cache>.db-wal  (bounded by journal_size_limit=64MB)  +  <cache>.db-shm (same host only)

CI (3-OS matrix, D-09): go test -tags=e2e -run TestTwoProcess ./cache/sqlite/  (helper-process harness)
```

### Recommended Project Structure

```
cache/sqlite/
├── sqlite.go                 # unchanged seam minus the three placeholders at :263-305 (they MOVE to batch.go)
├── batch.go                  # [NEW] MGetFunc/MSetFunc/MDelFunc + chunkedKeys/dedupe helpers + chunkSize const
├── batch_internal_test.go    # [NEW] unit tests: dedup, chunking, placeholders, decode-skip, no-partial-writes
├── sqlite_test.go            # extend: black-box batch semantics (happy path, expiry, parity with mem)
├── dsn.go                    # extend: journalSizeLimit param on buildDSN + prefix const + busyTimeout const
├── dsn_internal_test.go      # extend: buildDSN journal_size_limit cases (single-ownership — do not re-declare)
├── schema.go                 # extend: MGetSelectSQL / MDelSQL templates (SchemaSQL style)
├── retry.go                  # [NEW] isBusyError + retryBusy helpers (pure, unit-testable) OR in sqlite.go
├── retry_internal_test.go    # [NEW] fake-Error classifier + loop-bound/backoff unit tests
├── sqlite_internal_test.go   # extend: deterministic concurrent_first_open_all_usable (re-strengthen, :442)
├── doc.go                    # extend: same-host/multi-process contention bullet (CONC-04 remainder)
├── twoprocess_e2e_test.go    # [NEW] //go:build e2e — TestMain + helper child + contention & crash arms
└── (spike artifacts)         # 16-SPIKE-FINDINGS.md or folded into E2E commit message (D-11; discretion)
```

**Structure rationale:** `batch.go` + `batch_internal_test.go` mirror the Phase 15 `dsn.go`/`sweep.go` split (per-file coverage ≥70% with co-located tests); `retry.go` mirrors the pure-function seam pattern of `resolveLocation` — the retry loop and classifier are deterministic without a DB, and the heavy concurrent test is behavioral. `twoprocess_e2e_test.go` is e2e-tagged so `make coverage-quick` (untagged lane) never measures the harness (Phase 15 Pitfall 13).

### Pattern 1: Dedup + chunking helpers (pure, unit-testable)

**What:** D-02 input normalization: dedupe keys by their `fmt.Sprint`ed form, no-op on empty, then slice into 100-key chunks. For `MGet`, per-chunk `map[string]K` recovers the original key type from SQL rows (Finding 5).
**When to use:** First line of every `MGetFunc`/`MDelFunc`.
**Example:**
```go
// Source: this session's design; mirrors mem.go:112-118 (per-lock batch loop) in SQL form.
const chunkSize = 100 // D-01: fixed internal constant

// uniqueKeys dedups by stringified form, preserving input order.
func uniqueKeys[K comparable](keys []K) []K { /* map[string]struct{} over fmt.Sprint */ }

// chunksOf splits into bounded slices.
func chunksOf[K comparable](keys []K, size int) [][]K

// inPlaceholders returns "?,?,...,?" with n markers (n >= 1).
func inPlaceholders(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }
```

### Pattern 2: Chunked-IN SQL shapes (SchemaSQL style)

**What:** Constant templates with a `%s` placeholder for the marker list; values always bound. The expiry predicate is folded into the chunk SELECT (BATCH-01's "expired skipped" is SQL, not code).
**When to use:** MGet/MDel only — single-key ops keep the existing constants (`schema.go:26-35`).
**Example:**
```go
// Source: schema.go style (probe-verified: 100 keys + now = 101 params, 100 rows).
const (
    // MGetSelectSQL fetches one 100-key chunk. %s is ONLY the "?,?,..." marker
    // list produced by inPlaceholders — never caller data.
    MGetSelectSQL = `SELECT cache_key, value FROM cache_entries
WHERE cache_key IN (%s) AND (expires_at IS NULL OR expires_at > ?)`

    // MDelSQL deletes one 100-key chunk; missing keys are no-ops (BATCH-03).
    MDelSQL = `DELETE FROM cache_entries WHERE cache_key IN (%s)`
)
```
MGet per chunk: `QueryContext(ctx, fmt.Sprintf(MGetSelectSQL, inPlaceholders(len(chunk))), strKeys..., now)`; iterate rows, `Scan(&cacheKey, &value)`, `result[keyOf[cacheKey]] = value`. Decode failure → `logger().Debug(...)` + skip (CONTEXT specifics: decode = data hygiene at debug; DB errors = warn per D-03). Non-`ErrNoRows` chunk errors → `logger().Warn(...)` + continue to next chunk (D-03); a canceled context is also a chunk error here — the frozen signature has no error channel, so log-and-skip is the contracted behavior (flag for reviewers, do not invent an error return).
MDel per chunk: `ExecContext(ctx, fmt.Sprintf(MDelSQL, inPlaceholders(len(chunk))), strKeys...)`; first error → `fmt.Errorf("cache/sqlite: %w", err)` and return (D-05 fail-fast; idempotent so caller retry is safe).

### Pattern 3: MSet — one `BEGIN IMMEDIATE` transaction with one prepared upsert

**What:** D-04 seam: marshal everything up-front (before the tx opens, so a marshal error writes nothing), then one `BeginTx` (immediate via `_txlock`), one `tx.PrepareContext(ctx, UpsertSQL)`, one `ExecContext` per pair, `stmt.Close()`, `Commit`; any error rolls back via `defer tx.Rollback()` (no-op after Commit). `resolveTTL` is called once and one expiry binds the whole batch (mem parity: `mem.go:128-137` resolves once).
**When to use:** MSet only. Empty map → early `nil` (D-02 no-op discipline).
**Example:**
```go
func (c *sqliteCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    if err := c.check(); err != nil { return err }
    if len(items) == 0 { return nil }

    type pair struct{ key, data string }
    pairs := make([]pair, 0, len(items))
    for k, v := range items { // D-04: pre-marshal, nothing written on failure
        data, err := json.Marshal(v)
        if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
        pairs = append(pairs, pair{fmt.Sprint(k), string(data)})
    }
    expiresAt, hasTTL := c.resolveTTL(ttl...)
    var exp any
    if hasTTL { exp = expiresAt }

    tx, err := c.db.BeginTx(ctx, nil) // BEGIN IMMEDIATE via _txlock
    if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    defer func() { _ = tx.Rollback() }() // no-op after Commit — atomicity backstop
    stmt, err := tx.PrepareContext(ctx, UpsertSQL)
    if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    defer func() { _ = stmt.Close() }()
    for _, p := range pairs {
        if _, err := stmt.ExecContext(ctx, p.key, p.data, exp); err != nil {
            return fmt.Errorf("cache/sqlite: %w", err) // defer rolls back
        }
    }
    if err := tx.Commit(); err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    return nil
}
```
`//nolint:contextcheck`-safe: context flows through every `*Context` call; no goroutines introduced.

### Pattern 4: D-06 bounded connect retry (pure helpers + one call site)

**What:** `open()` pings the pinned pool before bootstrap, retrying only busy-class errors within the `busy_timeout` budget (5 s) with a tiny backoff (5 ms probe-proven). Deadline is `time.Now().Add(budget)` — absolute, so total wait is bounded even with worst-case backoff; `ctx` checking keeps cancellation prompt. `New` stays never-failing: exhausted retries fall into the existing `initErr` deferred path (`sqlite.go:72`).
**When to use:** Exactly one place — `open()` between pool pinning and `bootstrap` (`sqlite.go:100-107`). `isBusyError` is also wired into the deterministic regression test's contract.
**Example:**
```go
// retry.go — pure helpers, no driver import (Finding 3).
// busyErr is the minimal interface modernc's *sqlite.Error satisfies (Code() int).
type busyErr interface{ Code() int }

// isBusyError classifies SQLITE_BUSY by primary code. The driver enables
// extended result codes (conn.go:113), so compare code & 0xff.
func isBusyError(err error) bool {
    var be busyErr
    return errors.As(err, &be) && be.Code()&0xff == 5
}

const busyTimeout = 5 * time.Second // single source; mirrors _busy_timeout=5000 in dsn.go:15

func retryBusy(ctx context.Context, fn func(context.Context) error) error {
    deadline := time.Now().Add(busyTimeout)
    for {
        err := fn(ctx)
        if err == nil || !isBusyError(err) || time.Now().After(deadline) {
            return err
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(5 * time.Millisecond): // tiny backoff (D-06)
        }
    }
}
```
Call site: `if err := retryBusy(ctx, c.db.PingContext); err != nil { ... return err }` — a failed `database/sql` connection is discarded by the pool, so each Ping re-dials (verified: 8 retries across 240 raw-fail attempts all recovered in-process; 30/30 two-process runs cross-process).
**Determinism for tests:** unit-test `retryBusy` with a fake `fn` that yields N busy errors then nil (assert loop count and deadline); unit-test `isBusyError` with a fake `type fakeErr struct{ code int }` + `Code()`. The behavioral test is `TestBootstrapIdempotence/concurrent_first_open_all_usable` (`sqlite_internal_test.go:442`) — 4 providers opening one fresh path — which must now pass `-count=40` deterministically (probe: with retry 0/240 failures). Keep `t.Parallel()` and the existing worker shape; if the schedule allows, raise workers to 8 to strengthen the race window.

### Pattern 5: Helper-process two-process E2E (TestMain + env-flag re-exec)

**What:** In `twoprocess_e2e_test.go` (`//go:build e2e`), `TestMain` intercepts the child role before `m.Run()`; the parent spawns `exec.Command(os.Args[0])` with role env vars into a shared temp DB. `os.Args[0]` is the test binary and works on Windows. Go's own stdlib uses this pattern (os/exec `TestMain`, `helperCommand`).
**When to use:** CONC-01/CONC-05 only; e2e-tagged so the unit lane never executes it.
**Example (skeleton):**
```go
//go:build e2e
package sqlite_test

const (
    helperEnv  = "SQLITE_E2E_HELPER"
    dbEnv      = "SQLITE_E2E_DB"
    roleEnv    = "SQLITE_E2E_ROLE"
    crashEnv   = "SQLITE_E2E_CRASH"
    markerEnv  = "SQLITE_E2E_MARKER"
    iterations = 100          // per-process op count (discretion: tune for CI budget)
)

func TestMain(m *testing.M) {
    if os.Getenv(helperEnv) == "1" {
        os.Exit(helperMain()) // child path; never runs tests
    }
    os.Exit(m.Run())
}

func TestTwoProcessContention(t *testing.T) { /* parent: spawn 2 children, wait, assert exit 0,
    then reopen and assert integrity_check == "ok", journal_mode == "wal",
    per-role final rows present, deleted keys absent */ }

func TestTwoProcessCrashRecovery(t *testing.T) { /* parent: crasher holds BEGIN IMMEDIATE
    (marker file), survivor loops writes, kill crasher, assert survivor exit 0,
    reopen cleanly, uncommitted row absent */ }
```
Child `helperMain()` sketch (probe-verified logic): `sql.Open("sqlite", db+dsnFile)` → pin 1/1 → `retryBusy(PingContext)` (this is the *provider's own* open path in production; the E2E exercises the shipped provider via `sqlite.New`, so the retry lives in `open()` and the child just calls provider ops) → loop: `Set`/`Get`/`Delete` on `role:`-prefixed keys, plus periodic `MSet` batch (20 keys, beginning at 10% of iterations) and `MGet`/`MDel`; write a final marker key `role:final`. Crash child: provider or raw handle opens, `BeginTx`, inserts `uncommitted` row, writes marker file, blocks (`select {}`); parent asserts `require.Eventually` the marker, `Process.Kill()`, `Wait()`.
**Assertion strategy** (from the probe): exit codes all 0 in the contention arm (contention below `busy_timeout` — 5 s is generous for µs-scale autocommit writes); `integrity_check` after both arms; per-role key final values; the crash arm additionally asserts the uncommitted row count is 0 after reopen and that the survivor's data survived. Keep child stdout/stderr captured (`bytes.Buffer`) for failure diagnosis on CI.

### Pattern 6: Race strategy on the pinned pool

**What:** CONC-03 evidence = `go test -race` with concurrent goroutines hammering one provider: mixed `Get/Set/Delete/MGet/MSet/MDel/GetOrSet` (MGet/MSet/MDel are new code — the old race subset covered only point ops). On the pinned pool `database/sql` serializes real SQL, so the detector validates provider state (closed flag, channels) and the new batch code under caller-side races.
**When to use:** New black-box test `TestConcurrentBatchOpsRace` in `sqlite_test.go` (file mode + `:memory:` subtests), ~8 goroutines × bounded op loop with `t.Context()`; runs in the unit lane under `-race` in CI.
**Example command:** `go test -race ./cache/sqlite/ -count=1 -timeout=180s` (CI step, Pattern 7). Windows caveat: `-race` needs a C toolchain — gate the CI step to linux/macos (`runner.os != 'Windows'`) or run locally on all three; ubuntu-only is sufficient evidence for CONC-03 plus the local matrix.

### Pattern 7: CI integration points (`go.yml`)

**What:** The existing matrix `test` job (`ubuntu/macos/windows`) runs `go test -v ./...` — untagged, so the e2e harness is not yet executed anywhere. Two additions:
**When to use:** This phase, once the E2E and race tests exist.
**Example:**
```yaml
# .github/workflows/go.yml — extend the matrix job's steps (after "Run tests")
- name: SQLite two-process E2E
  run: go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s
- name: SQLite race detector
  if: runner.os != 'Windows'   # -race needs a C toolchain; windows gating keeps the matrix green
  run: go test -race ./cache/sqlite/ -count=1 -timeout=180s
```
Precedents already in the repo: the `coverage` job runs `go test ./...` with the badge upload; `Makefile:90-93` (`test-e2e`) runs Docker-backed e2e with `-run 'TestCacheE2E'` — the sqlite E2E is intentionally separate (`TestTwoProcess*` names) so it never drags Docker requirements into the matrix; `make coverage` (`Makefile:106-116`) runs `-tags=e2e -coverpkg=... ./cache/` — the sqlite e2e file is in `./cache/sqlite`, a subpackage already inside `./cache/`, so `make coverage` will compile and run it too (harmless without Docker; it needs none).

### Anti-Patterns to Avoid

- **Preparing the chunk statement per call** (`db.PrepareContext` per chunk): dynamic placeholder counts make the statement text vary; plain `QueryContext`/`ExecContext` (driver-side prepare) is what the probe verified. Reserve `tx.PrepareContext` for the single fixed MSet upsert (BATCH-02's letter).
- **Retrying any non-busy open error:** bad path, corrupt file, permissions → deferred `initErr` immediately; only `Code()&0xff==5` enters the loop (D-06/D-07).
- **Retrying BEGIN contention:** D-07 is explicit — beyond the open window, no provider retries; the CONC-02 test asserts the bounded error, not a retry.
- **Logging keys/values in MGet decode warnings:** decode skips log at debug with the key *count*/chunk index, never the key string (V7 logging hygiene; SECURITY table in Phase 15).
- **Chunking before dedup:** duplicate keys would re-query/re-delete the same stringified key (waste + wrong map re-key); D-02 order is fixed: dedup → chunk.
- **`strings.Repeat` of caller data into SQL:** only the `?` marker list is generated; keys/values stay bound parameters (V5).
- **Interleaving new rows mid-chunk-read:** not applicable — read is snapshot-consistent in WAL (probe: reader sees committed state at statement start).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Retry/busy classification | String-matching `"SQLITE_BUSY"` | `errors.As` + `interface{ Code() int }` + `Code()&0xff == 5` | Strings are locale/version-fragile; the typed code is stable across driver versions (Finding 3, probe) |
| Cross-process child processes | A second helper binary / `go run` | Test-binary re-exec via `exec.Command(os.Args[0])` + `TestMain` env flag | Standard Go pattern (stdlib-proven), no build plumbing, works on Windows (D-09) |
| In-process write serialization | A mutex/queue around batch ops | The pinned 1-connection pool | Already the Phase 15 model; `database/sql` serializes; adding locks would be the "background write-serializer" out-of-scope anti-pattern |
| Batch key map reconstruction | Parsing `fmt.Sprint` output | Per-chunk `map[string]K` from the input list | `Sprint` is not invertible; the chunk's own input is the only correct reverse map (Finding 5) |
| WAL growth control | Periodic `VACUUM`/file deletion | `PRAGMA journal_size_limit` via DSN (D-08) | SQLite-native, bounded, zero maintenance; VACUUM stays opt-in (out-of-scope table) |
| First-open WAL conversion race | Serializing opens with a lock file / retrying the whole open | Bounded ping-retry inside `open()` | Probe-proven minimal seam; lock files add cross-platform failure modes |
| E2E assertions | Implementing own DB consistency checker | `PRAGMA integrity_check` + row-count/final-value assertions | SQLite's own verifier; zero code |

**Key insight:** every hard problem this phase touches (concurrency, process coordination, WAL lifecycle) is already solved by the engine or the stdlib — the provider's job is DSN configuration, one bounded retry, and passing parameters through.

## Runtime State Inventory

> Not applicable: this phase is greenfield additions (batch methods, retry, tests) over shipped Phase 15 code; no rename/refactor/migration of runtime state. The only "state" change is D-08's `journal_size_limit` — a DSN pragma effective per connection, no stored schema/data mutation (flagged in D-08's Reversibility note as a behavior change for existing files, not a data migration).

## Common Pitfalls

### Pitfall 1: The first-open race still bites if the retry is wired to the wrong call
**What goes wrong:** Retrying `bootstrap()` (which retries the DDL transaction) instead of connection establishment — or retrying `sql.Open` (which never connects).
**Why it happens:** `sql.Open` is lazy; the DSN pragmas apply at first physical connection; the busy bypass belongs to the WAL conversion, and the probe shows it surfaces at `PingContext` (0/240 DDL failures after a successful ping).
**How to avoid:** `retryBusy(ctx, c.db.PingContext)` immediately after pool pinning, before `bootstrap`; keep the loop busy-classified and deadline-bounded.
**Warning signs:** test flakes moving from `Ping`-time to `begin`/DDL-time; retries counted but failures persisting.

### Pitfall 2: Extended result codes make `Code() == 5` comparisons wrong
**What goes wrong:** `SQLITE_BUSY_SNAPSHOT` (517), `SQLITE_BUSY_RECOVERY` (261) etc. fail a raw `== 5` check; the driver enables extended codes (`conn.go:113`).
**Why it happens:** `sqlite3_step` returns extended codes when enabled; `Error.Code()` carries the raw rc (`conn.go:433-440, 966`).
**How to avoid:** `Code()&0xff == 5`; test the classifier with an extended code value (e.g. 517) via the fake-error seam.
**Warning signs:** busy errors slipping through the classifier in corner contention paths.

### Pitfall 3: Unconsumed rows deadlock the pinned pool (Phase 15 Pitfall 11, now on the batch path)
**What goes wrong:** MGet's chunk `QueryContext` without `defer rows.Close()` holds the single connection; the next chunk (or any op) blocks until ctx deadline.
**How to avoid:** Every chunk: `rows, err := ...; if err != nil {...}; defer rows.Close()` inside the chunk scope (close before the next chunk), iterate fully, check `rows.Err()`. Review point for the executor.
**Warning signs:** context-deadline errors on the second chunk; `-race` hangs.

### Pitfall 4: Atomicity tested only by "happy path" passes — the rollback class needs the invariant test
**What goes wrong:** Pre-marshal makes marshal errors fire before the tx opens (good), so a naive test "marshal failure stores nothing" never exercises `tx.Rollback`; a mid-batch SQL failure is hard to force deterministically with bound params.
**How to avoid:** (a) deterministic: pre-marshal failure → assert zero rows; ctx-canceled-before-call → zero rows; (b) invariant: `TestMSetNoPartialWrites` — a large MSet (e.g. 2000 pairs) with the ctx canceled ~1 ms after start; assert the stored count is `0` **or** `2000`, never partial (both outcomes legal; rollback branch gets executed in practice; repeat 3-5 iterations for coverage).
**Warning signs:** no `mset no-partial` test; timestamp-based sleeps asserting a specific outcome.

### Pitfall 5: MGet chunk failure swallowed including context cancellation
**What goes wrong:** D-03's log-warn-and-skip applies to *any* chunk error — including `context.Canceled` mid-MGet — because the frozen signature has no error channel (`cache/cache.go:39`).
**Why it happens:** The contract is unavoidable; the root decision is D-03.
**How to avoid:** Log the wrapped error at warn (never pretend success); document in `doc.go` that a canceled MGet returns partial results; do NOT attempt to smuggle errors into the map.
**Warning signs:** tests asserting MGet "returns errors"; reviewers suggesting signature changes (frozen).

### Pitfall 6: `_pragma` value hygiene for journal_size_limit
**What goes wrong:** A caller-tunable value interpolated into `_pragma=...` is verbatim SQL execution in the driver (M6 in Phase 15 security) — the worst injection class in this package.
**How to avoid:** Same as `wal_autocheckpoint`: the value is a compile-time constant (`67108864` from D-08) or `strconv.Itoa` of an int64 field — never a string; append only in file mode (memory DSN must not carry `_pragma=journal_size_limit`; mirror `dsn.go:87-91` behavior).
**Warning signs:** `_pragma` building from an option string; memory-mode DSN containing journal_size_limit.

### Pitfall 7: E2E harness leaks into the unit lane / fails locally
**What goes wrong:** `make coverage-quick` computes thresholds with the harness counted (file/package drops below 70/80); or the harness runs without `-tags=e2e` and hangs (children waiting).
**How to avoid:** `//go:build e2e` on the harness file; `TestMain` intercepts child roles only when the env flag is set; parent always `Kill()`s children in `defer`/cleanup; assert-with-`require.Eventually` for the marker file.
**Warning signs:** CI unit lane slowing; `coverage-quick` red on batch commit; zombie test processes on macOS after failures.

### Pitfall 8: Windows file semantics in the E2E (Phase 15 Pitfall 6, now cross-process)
**What goes wrong:** Unclosed handles in children block `t.TempDir` cleanup on windows-latest; killed children may leave `-wal`/`-shm` momentarily; AV scanners can hold the file.
**How to avoid:** Every child `defer db.Close()`; parent asserts sidecar absence with `require.Eventually` (not immediately); children exit promptly; `Process.Kill()` + `Wait()` in `defer`; never assert Unix permissions.
**Warning signs:** CI red only on windows-latest; `-wal` files in temp dirs after the E2E.

### Pitfall 9: `t.Parallel()` inside the concurrent-first-open regression test
**What goes wrong:** The re-strengthened `concurrent_first_open_all_usable` (deterministic-green with retry) must still not race the sequential sibling; the sibling's `require.NoError(t, c1...` on shared temp paths is fine, but parallel top-level tests share the package binary's module cache only — keep the existing `t.Parallel()` structure and the fresh `t.TempDir()` pattern per subtest.
**Why it happens:** Good isolation is already the Phase 15 shape (`sqlite_internal_test.go:424-469`).
**How to avoid:** Change only the retry behavior (add `retryBusy` to `open()`), keep the test shape; raise workers or add `-count` loop only if the planner wants extra strength.
**Warning signs:** new helper functions leaking global state across subtests (Phase 15 parallel-isolation note).

## Code Examples

### 1. Bounded open-retry wiring in `open()` (D-06)
```go
// sqlite.go — inside open(), replacing the current first-touch ordering:
db.SetMaxOpenConns(1)   // existing pool pinning (sqlite.go:100-103) stays
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
db.SetConnMaxIdleTime(0)
c.db = db

// D-06: fresh-file WAL conversion races another process's first open and can
// return immediate SQLITE_BUSY with the busy handler bypassed (DI-15-01).
// Retry only connection establishment, only busy errors, within busy_timeout.
if err := retryBusy(ctx, c.db.PingContext); err != nil {
    _ = db.Close()
    c.db = nil
    return err // deferred via New's initErr (sqlite.go:72) — New never fails
}

if err := c.bootstrap(ctx); err != nil { /* existing: close + return */ }
```

### 2. `isBusyError` + fake-error unit test
```go
// retry.go
type busyErr interface{ Code() int }

func isBusyError(err error) bool {
    var be busyErr
    return errors.As(err, &be) && be.Code()&0xff == 5
}

// retry_internal_test.go
type fakeErr struct{ code int }

func (f fakeErr) Error() string { return fmt.Sprintf("fake %d", f.code) }
func (f fakeErr) Code() int     { return f.code }

assert.True(t, isBusyError(fakeErr{code: 5}))
assert.True(t, isBusyError(fakeErr{code: 517})) // extended busy snapshot
assert.False(t, isBusyError(fakeErr{code: 1}))  // SQLITE_ERROR
assert.False(t, isBusyError(errors.New("plain")))
```

### 2b. Why the import stays blank (name collision)
```go
// package sqlite imports "modernc.org/sqlite" — the DRIVER package is ALSO
// named sqlite. A named import would read `sqlite.Error` inside `package sqlite`,
// legal but confusing; the interface-based classifier (Code Example 2) avoids
// the import entirely. Keep: import _ "modernc.org/sqlite" (sqlite.go:16).
```

### 3. Chunked MGet (BATCH-01, D-02, D-03)
```go
func (c *sqliteCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    result := make(map[K]V) // empty map, not nil — D-02 no-op returns empty map
    if err := c.check(); err != nil { // initErr/closed: log once, return empty
        logger().Warn("cache/sqlite: batch get skipped", "error", err)
        return result
    }
    now := time.Now().UnixNano()
    for _, chunk := range chunksOf(uniqueKeys(keys), chunkSize) {
        byKey := make(map[string]K, len(chunk))
        strKeys := make([]any, 0, len(chunk))
        for _, k := range chunk {
            sk := fmt.Sprint(k)
            byKey[sk] = k
            strKeys = append(strKeys, sk)
        }
        args := append(strKeys, now)
        rows, err := c.db.QueryContext(ctx, fmt.Sprintf(MGetSelectSQL, inPlaceholders(len(chunk))), args...)
        if err != nil {
            logger().Warn("cache/sqlite: batch get chunk failed", "keys", len(chunk), "error", err) // D-03
            continue
        }
        for rows.Next() {
            var cacheKey, data string
            if err := rows.Scan(&cacheKey, &data); err != nil {
                logger().Warn("cache/sqlite: batch get row failed", "error", err)
                continue
            }
            var value V
            if err := json.Unmarshal([]byte(data), &value); err != nil {
                logger().Debug("cache/sqlite: batch get undecodable entry skipped", "key", cacheKey) // CONTEXT specifics
                continue
            }
            result[byKey[cacheKey]] = value
        }
        if err := rows.Err(); err != nil { // close before the next chunk (Pitfall 3)
            logger().Warn("cache/sqlite: batch get chunk iteration failed", "error", err)
        }
        _ = rows.Close()
    }
    return result
}
```

### 4. Chunked MDel (BATCH-03, D-05)
```go
func (c *sqliteCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    if err := c.check(); err != nil {
        return err
    }
    for _, chunk := range chunksOf(uniqueKeys(keys), chunkSize) {
        strKeys := make([]any, 0, len(chunk))
        for _, k := range chunk {
            strKeys = append(strKeys, fmt.Sprint(k))
        }
        if _, err := c.db.ExecContext(ctx, fmt.Sprintf(MDelSQL, inPlaceholders(len(chunk))), strKeys...); err != nil {
            return fmt.Errorf("cache/sqlite: %w", err) // D-05: partial delete acceptable, wrapped error
        }
    }
    return nil // empty/deduped-to-empty input naturally returns nil here (D-02)
}
```

### 5. Helper-process harness (CONC-05; probe-verified shape)
```go
// twoprocess_e2e_test.go — //go:build e2e
func helperMain() int { // child: drive the SHIPPED provider via sqlite.New
    c := sqlite.New[string, string](sqlite.WithPath(os.Getenv(dbEnv)))
    defer func() { _ = c.Close() }()
    ctx := context.Background()
    role := os.Getenv(roleEnv)
    for i := range iterations {
        key := fmt.Sprintf("%s:%04d", role, i)
        if err := c.Set(ctx, key, fmt.Sprintf("v-%d", i)); err != nil { return 1 }
        if _, err := c.Get(ctx, key); err != nil { return 1 }
        if i%10 == 0 {
            batch := map[string]string{}
            for j := range 20 { batch[fmt.Sprintf("%s:batch:%d:%d", role, i, j)] = "b" }
            if err := c.MSet(ctx, batch); err != nil { return 1 }
            if len(c.MGet(ctx, fmt.Sprintf("%s:batch:%d:0", role, i))) != 1 { return 1 }
        }
        if i%15 == 0 {
            if err := c.MDel(ctx, fmt.Sprintf("%s:%04d", role, i/2)); err != nil { return 1 }
        }
    }
    if err := c.Set(ctx, role+":final", "done"); err != nil { return 1 }
    return 0
}
```
Crash arm: crasher child (env `crashEnv=1`) opens a raw `*sql.DB` with the production DSN, `BeginTx`, `INSERT ... 'uncommitted'`, writes the marker env path, `select {}`; parent `require.Eventually` the marker, `Process.Kill()`, `Wait()` (exit != 0 is expected and asserted), then survivor continues and the parent reopens and asserts `PRAGMA integrity_check` = `ok` + `uncommitted` count = 0.

### 6. DSN journal_size_limit append (D-08)
```go
// dsn.go — mirror dsnAutoCheckpointPrefix (dsn.go:20):
const (
    dsnSuffixFile           = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"
    dsnAutoCheckpointPrefix = "&_pragma=wal_autocheckpoint("
    dsnJournalSizeLimitPfx  = "&_pragma=journal_size_limit("

    // journalSizeLimitBytes bounds WAL file size between checkpoints (D-08:
    // 64 MB). Zero disables the pragma (SQLite's unlimited default).
    journalSizeLimitBytes = 64 << 20
)

func buildDSN(path string, memory bool, autoCheckpointPages int, journalSizeLimit int64) string {
    if memory {
        return path + dsnSuffixMemory
    }
    dsn := path + dsnSuffixFile
    if autoCheckpointPages > 0 {
        dsn += dsnAutoCheckpointPrefix + strconv.Itoa(autoCheckpointPages) + ")"
    }
    if journalSizeLimit > 0 {
        dsn += dsnJournalSizeLimitPfx + strconv.FormatInt(journalSizeLimit, 10) + ")"
    }
    return dsn
}
```
`open()` passes `journalSizeLimitBytes`; `dsn_internal_test.go` extends with the D-08 equality cases (two `_pragma` params both appended, memory mode omits both — single-ownership rule: extend `TestDSNBuildDSN`, don't re-declare).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Loop-based batch placeholders (`sqlite.go:263-305`, Phase 15) | Chunked-IN queries + one-tx prepared upsert | this phase (BATCH-01..03) | SQL round-trips drop from O(n) to O(ceil(n/100)); MSet atomicity is real |
| Unbounded retry absence — first-open race surfaced as deferred busy (DI-15-01) | Bounded connect retry (5 s budget, 5 ms backoff; busy-only) | this phase (D-06) | CONC-01/05 pass deterministically; probe: 16/30 two-process runs → 0/30 |
| WAL grows without bound between checkpoints (default unlimited) | `journal_size_limit=64 MB` via DSN (file mode) | this phase (D-08) | Long-running processes get a hard WAL ceiling; passive autocheckpoint behavior otherwise unchanged |
| Contention at any statement (deferred tx upgrades) | Contention at BEGIN only (`_txlock=immediate`, Phase 15) + **probe-verified** bounded SURFACE | Phase 15 design, proven this phase | CONC-02's observable contract: `busy_timeout`-bounded error at BEGIN |
| Single-process proof only | Two-process helper-process E2E on 3 OSes (D-09/D-10) | this phase | Milestone headline proven in CI, incl. Windows handle semantics |

**Deprecated/outdated:**
- The Phase 15 "batch placeholders" pattern (`sqlite.go` loop stubs) — replaced in place by `batch.go` (the `// Phase 16:` comments at `sqlite.go:263-305` are removed with the stubs).
- Milestone research's "two-process behavior must be desk-researched" LOW-confidence flag (STATE.md blockers) — **resolved by this phase's probes**: behavior is now primary-source verified on the pinned driver.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Probe results (darwin/arm64, Go 1.27.0, pinned driver) transfer to Linux/Windows CI: WAL locking, busy handler, and the race/recovery semantics are engine-level; the 3-OS E2E (D-09) is the confirmation loop | Findings 1/3; Environment | A CI-only failure on one OS → test/CI adjustment, not design change; D-09 already locks the matrix |
| A2 | D-08 reading: 64 MB is the shipped hard default (constant in `buildDSN`); "zero-value/disabled path" = the `buildDSN` param contract (positive appends, zero skips), matching `WithAutoCheckpoint`'s existing shape — **no new option** (CONTEXT code_context: "options.go — none planned per D-01/D-08 defaults") | Pattern 4 / Code Example 6 | If the planner reads D-08 as requiring an exported `WithJournalSizeLimit`, add it in options.go (small) — see Open Question 1 |
| A3 | MGet/MDel dedup keyed on the stringified form (SQL operates on strings); two distinct `K` values with equal `Sprint` (e.g. `any(1)` vs `any("1")`) collapse to the first occurrence | Pattern 1 | Documented edge; matches SQL-level reality; mem/postgres do not define it either |
| A4 | MDel fails fast on the first chunk error (D-05's "partial delete acceptable" permits stop-and-return; retry re-runs remaining keys) | Pattern 2 | If reviewers prefer continue-all-then-error, iteration continues after the first error — trivial change either way |
| A5 | MGet chunk-failure logging includes context cancellation (D-03 verbatim: warn + skip) — callers see partial maps on cancellation | Pattern 2 / Pitfall 5 | Contract consequence of the frozen signature; flagged for the planner as a documented behavior |
| A6 | Race detector evidence: CI on linux + macos (`runner.os != 'Windows'` gating) plus local runs on all OSes satisfies CONC-03 | Pattern 6/7 | If Windows `-race` is demanded, mingw toolchain setup on windows-latest is extra CI work — flag for human |
| A7 | The spike (D-11) can reuse the E2E harness file rather than a separate scratch binary; "spike first" = the harness's contention arm is proven before the batch code lands, and binary/harness artifacts stay in the e2e-tagged file | Pattern 5 | If the spike is expected as a throwaway, the executor deletes/re-creates it — cost is small either way (discretion) |
| A8 | MSet's `defer tx.Rollback()` (defensive) plus pre-marshal makes all practical error classes atomic; the only reachable SQL-time failure is engine-level (busy/disk/ctx) | Pattern 3 / Pitfall 4 | The invariant test (`0-or-all`) is the safety net, not the rollback line itself |

**If this table is empty:** n/a — the table above lists open items; none blocks planning, and A2/A4/A6/A7 are planner-visible choices with defaults.

## Open Questions

1. **D-08 surface: constant vs option**
   - What we know: CONTEXT code_context says no option plumbing is planned ("none planned per D-01/D-08 defaults"); D-08's text mentions a "zero-value/disabled path" mirroring `WithAutoCheckpoint`.
   - What's unclear: whether a caller-facing zero-value path must exist (option) or the param-level zero contract suffices (constant default, no option).
   - Recommendation: implement as a `buildDSN` param gated on `> 0`, shipped value = constant `64 << 20`, **no new option** (smaller API surface; option is additive later without breaking callers). If the planner disagrees, adding `WithJournalSizeLimit(int64)` mirrors `WithAutoCheckpoint` exactly and costs one option + tests.

2. **Spike artifact disposition**
   - What we know: D-11 folds the pending todo; its results "confirm the retry policy before the E2E hardens it".
   - What's unclear: whether the spike is a separate scratch (throwaway) or the E2E harness's first iteration.
   - Recommendation: build the E2E harness as the spike vehicle (contention arm only), record findings in the task description/commit message (or an optional `16-SPIKE-FINDINGS.md`), then extend the same file with the crash arm and CI wiring. Marked as the agent's discretion in CONTEXT.

3. **Race-detector CI scope**
   - What we know: CONC-03 requires `go test -race` green; the repo's CI has no `-race` anywhere today; `-race` needs cgo (works on linux/macos runners; Windows needs mingw).
   - What's unclear: how much OS coverage the plan should buy.
   - Recommendation: `if: runner.os != 'Windows'` in the matrix job (linux + macos), plus the existing local runs; a follow-up phase can broaden if demanded.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/test | ✓ | go1.27.0 darwin/arm64 (repo `go 1.26.4`; driver requires ≥1.26.0) | — |
| `modernc.org/sqlite` v1.60.1 + `modernc.org/libc` v1.77.1 | all phase code | ✓ | pinned in `go.mod` (Phase 15) | — |
| `go-test-coverage` + `golangci-lint` | per-commit gates | ✓ | installed | — |
| Docker | **not needed** — the E2E is helper-process (no containers); existing `make coverage` e2e flows unchanged | ✓ (for the unchanged cache root tests) | — | — |
| Linux/Windows runtimes | 3-OS E2E + race + cross-OS verification | ✗ local | — | GitHub Actions matrix (D-09) |
| SQLite CLI / other DB tools | none | — | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** Linux/Windows local execution — covered by the existing CI matrix (D-09 adds the E2E there; A6 covers race gating).

## Validation Architecture

> **Skipped.** `.planning/config.json` sets `workflow.nyquist_validation: false`, so the Validation Architecture section is omitted per the research contract. Test strategy for every requirement is documented in the Phase Requirements table and Common Pitfalls (deterministic seeks: fake-error classifier unit tests, `-count=40` bootstrapping regression, no-partial-writes invariant, `require.Eventually` crash-marker sync, `integrity_check` assertions).

## Security Domain

`security_enforcement: true` (ASVS L1). This phase adds no auth/session/crypto surface — the applicable categories are input validation, error handling/logging, and filesystem resource handling, all extensions of Phase 15's posture. `sqlite`-specific control for every pattern below is the same one that Phase 15 established: **all SQL statements bind parameters; the only generated SQL text is the `?` marker list; the only generated pragma values are compile-time constants.**

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes | `inPlaceholders`/`fmt.Sprintf` templates produce only `?` markers (keys/values bound); `journal_size_limit` value is a constant/`strconv.FormatInt` of an int64; dedup before chunking; no caller string reaches SQL |
| V6 Cryptography | no | — |
| V7 Error Handling & Logging | yes | Wrapped `%w` errors with `cache/sqlite:` prefix; warn = DB/system errors (MGet chunk), debug = undecodable rows; **never log key strings or values** (log chunk index/count); D-03's skip policy must not swallow the wrapped error silently |
| V12 Files & Resources | yes | Cross-process file sharing is the new trust boundary: document same-host-only (CONC-04); idempotent Close in children; no `os.Remove` of sidecars (engine-owned) |

### Known Threat Patterns for {Go + SQLite + database/sql}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL injection via batch keys/values | Tampering | Bound params only; generated text is the `?` list (`inPlaceholders`); MGetSelectSQL/MDelSQL templates never embed data |
| DSN injection via `_pragma` (verbatim SQL execution in the driver) | Tampering | Only compile-time constants / `strconv.Itoa`/`FormatInt`; rejection of `?`/`#` in paths already shipped (dsn.go:37-44); memory mode omits file pragmas |
| Cross-process cache poisoning / unauthorized sharing | Spoofing/Tampering | Same-host + filesystem permissions are the trust boundary; WAL requires shared memory (local storage only — doc.go:37-38 + CONC-04 bullet); `integrity_check`-based E2E assertions detect corruption |
| Secret leakage through batch logging | Information Disclosure | MGet logs counts/chunk index and the wrapped error only; decode skips log at debug with the key name? **no** — log the key at debug is acceptable (data hygiene event), but never values; Phase 15's V7 rule stands ("never log keys/values" applies to values; keys at debug for the skip event are per CONTEXT specifics and are caller-chosen cache keys, not secrets — planner may downgrade to a key-count log if strict) |
| Unbounded resource growth (WAL) | DoS | `journal_size_limit=64 MB` (D-08) bounds the WAL; no periodic VACUUM (out of scope); checkpoint stays opt-in |

## Sources

### Primary (HIGH confidence)
- **Throwaway probe (this session, darwin/arm64, Go 1.27.0, pinned v1.60.1 — inside the repo, removed after use)** — verbatim outputs:
  - `PROBE1 journal_size_limit=67108864 wal_autocheckpoint=500` — two `_pragma` DSN params coexist and read back.
  - `PROBE2 begin2 err=database is locked (5) (SQLITE_BUSY) elapsed=5.052995625s busy=true code=5 (primary=5)`; `PROBE2b err=... elapsed=309.05025ms` at busy_timeout=300 — BEGIN IMMEDIATE contention is busy-handler-bounded and typed.
  - `PROBE3 rows=100` — chunked IN with 101 bound params.
  - `PROBE4 rows=100` — prepared upsert, 100 execs in one tx.
  - `PROBE5 reader-during-write err=nil v="1" elapsed=196.917µs; reader count=1 (expect 1)` then 2 after commit — WAL readers don't block on writers, uncommitted invisible.
  - `PROBE6 raw: ping_fail=14 ddl_fail=0 / 60 iters; retry: nonbusy_fail=0 retries_used=8` — in-process first-open race surfaces at connection establishment only; bounded retry fixes all.
  - `PROBE8 ... total=840 alpha=420 beta=420; journal_mode=wal integrity=ok` — two OS processes × 300 iters incl. batch txs; raw arm: **16/30 two-process runs failed** (one child busy at first use); retry arm: **0/30**.
  - `PROBE9 integrity=ok uncommitted_rows=0 survivor_rows=420 ... 10/10 runs` — kill-mid-write recovery: survivor continues, uncommitted rolled back, file reopens cleanly.
- **`modernc.org/sqlite` v1.60.1 module source (read this session)**: `error.go:12-21` (`Error` with `Code()`), `conn.go:113` (`extendedResultCodes(true)`), `conn.go:433-440` (step returns rc + errstr), `conn.go:966-981` (`Error{... code: int(rc)}`), `tx.go:20-24` (`begin immediate`), `sqlite.go:285-460` (applyQueryParams: validation-first, busy_timeout first, `_pragma` verbatim sorted), `lib/sqlite.go:4014` (`SQLITE_MAX_VARIABLE_NUMBER = 32766`).
- **Repo code reads (this session, with line ranges)**: `cache/cache.go:34-47` (frozen `BatchCache`), `cache/concrete_cache.go:13-23` (cacher) and `:69-83` (MGet/MSet/MDel passthrough), `cache/sqlite/sqlite.go:263-305` (placeholders), `cache/sqlite/schema.go:26-40` (SQL constants), `cache/sqlite/dsn.go:14-24,82-93` (DSN suffixes + buildDSN), `cache/sqlite/sqlite_internal_test.go:424-468` (bootstrap test, concurrent leg), `cache/sqlite/doc.go:34-41` (WAL constraint already documented at :37-38), `cache/mem/mem.go:112-139` (batch precedent, single-lock resolveTTL-once semantics).

### Secondary (MEDIUM confidence)
- [sqlite.org/pragma.html](https://www.sqlite.org/pragma.html) — `journal_size_limit` semantics (truncation on commit/WAL reset; negative = unlimited; zero = truncate always), retrieved via websearch this session.
- [sqlite.org/wal.html](https://sqlite.org/wal.html) — autocheckpoint default (1000 pages), "checkpoint does not normally truncate the WAL file (unless the journal_size_limit pragma is set)", WAL local-storage/shared-memory constraint.
- [golang.org/go src/os/exec/exec_test.go](https://github.com/golang/go/blob/master/src/os/exec/exec_test.go) — TestMain helper-command re-exec pattern; [pkg.go.dev/testing](https://pkg.go.dev/testing) — TestMain contract (runs before flag.Parse; os.Exit semantics).

### Tertiary (LOW confidence)
- None — every design-relevant claim was verified against the pinned driver, the repo, or the primary SQLite docs. (Websearch-only hits were used for pattern confirmation, not as claim sources; the two-process claim is probe-backed, not web-backed.)

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — no new dependencies; batch/retry/harness libraries are stdlib + the pinned driver, all probe-verified this session.
- Architecture: **HIGH** — batch SQL shapes, tx seam, retry boundary, and E2E harness were executed against the real driver (30/30 + 10/10 probe runs); integration seams read directly with line ranges.
- Pitfalls: **HIGH** for the race/retry/atomicity clusters (probe-reproduced); MEDIUM for Windows-specific timing (AV/handle nuances carried from Phase 15 evidence; D-09 CI is the confirmation).

**Research date:** 2026-10-08
**Valid until:** 2026-11-07 (30 days — driver pinned v1.60.1; SQLite pragma semantics stable). The two-process probe results are tied to the pinned driver build; a driver bump invalidates them.
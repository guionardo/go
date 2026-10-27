# Phase 16 — Two-Process WAL Spike Findings (D-11)

**Recorded:** 2026-10-08
**Harness:** `cache/sqlite/twoprocess_e2e_test.go` (first line `//go:build e2e`)
**Purpose:** CONC-01 precondition — prove two OS processes share one fresh cache
file under WAL without corruption, and confirm the D-06 bounded-retry policy in
its exact shape before Plan 16-02 implements it in `open()`. This is the folded
two-process WAL spike (D-11) and the disposition record for DI-15-01.

## 1. Harness shape

Helper-process re-exec via `TestMain` + env flag (the standard Go pattern; no
second binary, works on Windows):

| Env var | Role |
|---------|------|
| `SQLITE_E2E_HELPER=1` | Child intercept — the re-executed test binary runs `helperMain()` instead of `m.Run()` |
| `SQLITE_E2E_DB=<abs path>` | Shared database file (under the parent's `t.TempDir()`) |
| `SQLITE_E2E_ROLE=alpha\|beta` | Per-child key namespace and final-marker name |
| `SQLITE_E2E_RETRY=1\|0` | `1` (default) = policy arm; `0` = raw arm (single ping, no retry) |

- The parent (`TestTwoProcessSpikeContention`) spawns two children with
  `exec.Command(os.Args[0])`, the parent environment plus the four variables
  above; child stdout/stderr are captured in `bytes.Buffer` for failure
  diagnostics, and Kill+Wait cleanup is registered for both children.
- Each child opens a **raw `database/sql` handle** with
  `dbPath + "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"`,
  pins the pool at one connection, establishes the connection (policy or raw
  ping), idempotently bootstraps the mirrored schema inside `BEGIN IMMEDIATE`,
  then runs `iterations = 100` point upsert/read-back pairs, a 20-key
  prepared-upsert transaction every 10th iteration (10 transactions × 20 rows),
  and finishes with the marker key `role:final` = `"done"`.
- Parent assertions after both children exit: exit code 0 each; raw reopen
  reports `PRAGMA journal_mode` = `wal`; `PRAGMA integrity_check` = `ok`;
  each role's row count is exactly `iterations + (iterations/10)*20 + 1 = 301`;
  each role's final marker is `done`.
- **DSN/SQL mirrors:** `e2eDSN` mirrors the `dsnSuffixFile` literal and the
  `e2eCreateTableSQL`/`e2eCreateIndexSQL`/`e2eUpsertSQL`/`e2eSelectSQL`
  constants mirror `schema.go`. This mirror is spike-local and is eliminated in
  Plan 16-04, when the children rewire to the shipped provider (whose `open()`
  owns the real DSN).
- **Unit-lane isolation:** the e2e build tag keeps the harness invisible to the
  untagged lane — `go test -list 'TestTwoProcess' ./cache/sqlite/` lists zero
  results, the untagged suite is green, and `make coverage-quick` stays green
  with no `cache/sqlite` override in `.testcoverage-quick.yml`.

## 2. Raw arm observation

Ten parent runs with `SQLITE_E2E_RETRY=0` (no retry; one ping):

- **8/10 runs failed** (80%). In every failing run exactly one child exited 1
  with `sqlite e2e helper <role>: ping: database is locked (5) (SQLITE_BUSY)` —
  the DI-15-01 first-open WAL-conversion race, reproduced cross-process with
  the busy handler bypassed at connection establishment. Failing role split:
  alpha 5, beta 3 (each failing run has exactly one loser; the winner — often
  the process that performed the conversion — proceeds normally).
- 2/10 runs passed (runs 4 and 9) — the race is probabilistic, as expected.

Research probe prior (`16-RESEARCH.md`, Finding 1/PROBE8): **16/30 raw
two-process runs failed** (~53%) with the same signature (one child hits
immediate `SQLITE_BUSY` at first use, busy handler bypassed); Phase 15 measured
29/40 (~72%) in-process. The observed 8/10 is the same failure class at the
high end of the earlier range.

## 3. Policy arm result

Same harness with `SQLITE_E2E_RETRY=1` (the parent default). Post-fix evidence:

- **30/30 clean runs** (invocations `-count=5`, `-count=20`, and the tracer-gate
  `-count=5`), 0 failures; both children exited 0 in every run.
- Every run's raw reopen reported `journal_mode` = `wal` and
  `integrity_check` = `ok`.
- Every run's per-role row counts were exactly 301 (100 point + 200 batch + 1
  final marker), with both final markers `done`.

Research probe prior: **30/30 clean with the policy** (vs 16/30 failed raw).
With an ~80% per-run raw failure rate on this machine, 30 consecutive clean
policy runs is ~0.2^30 by chance alone — the retry, not scheduling luck, is
doing the work.

## 4. Confirmed policy

The D-06 retry policy is confirmed in the exact triple Plan 16-02 implements:

| Parameter | Value |
|-----------|-------|
| Backoff | **5 ms** wait between attempts (`time.After(5 * time.Millisecond)`) |
| Budget | **5 s** absolute deadline (`time.Now().Add(5*time.Second)` — same value `_busy_timeout=5000` sets in the DSN) |
| Classification | **Busy-only** via `errors.As` against `interface{ Code() int }` and `Code()&0xff == 5` (covers extended codes such as `SQLITE_BUSY_SNAPSHOT` 517) |
| Seam | **Connection establishment only** — exactly one `PingContext` wrapped; never at bootstrap DDL (DDL after a successful ping was never observed to fail) and never around `BeginTx` (ordinary contention stays bounded by `busy_timeout`, D-07) |

A failed ping discards the pooled connection, so each retry re-dials and
re-applies the DSN pragmas. Non-busy errors (bad path, corrupt file,
permissions) never enter the loop.

## 5. Disposition

- **Plan 16-02 (D-06):** implement this policy in `open()` —
  `retryBusy(ctx, c.db.PingContext)` between pool pinning and `bootstrap`, with
  the pure `isBusyError` classifier; re-strengthen
  `TestBootstrapIdempotence/concurrent_first_open_all_usable` to deterministic
  green as the in-process regression.
- **Plan 16-04 (D-09/D-10):** extend this same harness file with
  provider-driven children, the kill-mid-write crash arm, and the 3-OS CI
  wiring; the raw DSN/SQL mirrors in the harness are eliminated at that point.
- **DI-15-01:** closed from the evidence side — the race is reproduced
  cross-process (raw arm) and eliminated by the confirmed policy (policy arm).
  The fix lands with Plan 16-02.
- **CONC-01:** precondition satisfied — two OS processes share one fresh file
  under WAL with `integrity_check = ok`, correct row counts, and deterministic
  clean exits under the policy.

*Evidence artifact for D-11; harness committed in `16-01` as
`cache/sqlite/twoprocess_e2e_test.go`.*

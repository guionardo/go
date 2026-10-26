# QUAL-02 Coverage Audit — `cache/sqlite`

**Audit date:** 2026-10-26
**Command:** `make coverage-quick` (`go test ./... -coverprofile=./cover.out -covermode=atomic -count=1`
then `go-test-coverage --config=./.testcoverage-quick.yml`)
**Tree:** branch `gsd/phase-17-delivery-hardening`, after 17-01 (CI) and 17-02 (benchmarks) landed and
after this plan's doc/example commits.

This record closes **QUAL-02 by verification, not by churn** (D-12): every one of the five named test
classes already has a covering test in the shipped suite, so this phase adds **no** test for a class
that already had one. The evidence below is a mapping plus a quoted gate run — reproducible by
re-running the one command in the header.

## Five named classes -> existing tests

| Named class | Covering test(s) and `file.go:LINE` | What it asserts |
|-------------|-------------------------------------|-----------------|
| **unit** | `TestFileCRUD` `cache/sqlite/sqlite_test.go:319`; `TestParity` `cache/sqlite/sqlite_test.go:472`; `TestMarshalFailure` `cache/sqlite/sqlite_test.go:530`; `TestClosedOps` `cache/sqlite/sqlite_test.go:546`; `TestDoubleClose` `cache/sqlite/sqlite_test.go:657`; `TestOptimizable` `cache/sqlite/sqlite_test.go:705`; `TestResolveTTL` `cache/sqlite/sqlite_internal_test.go:73`; `TestBootstrapIdempotence` `cache/sqlite/sqlite_internal_test.go:509`; `TestCorruptedRowNotMiss` `cache/sqlite/sqlite_internal_test.go:598`; `TestCanceledContext` `cache/sqlite/sqlite_internal_test.go:614`; batch units `TestUniqueKeys` `cache/sqlite/batch_internal_test.go:19`, `TestChunksOf` `cache/sqlite/batch_internal_test.go:47`, `TestInPlaceholders` `cache/sqlite/batch_internal_test.go:92`, `TestMGetFunc` `cache/sqlite/batch_internal_test.go:106`, `TestMSetFunc` `cache/sqlite/batch_internal_test.go:247`, `TestMSetNoPartialWrites` `cache/sqlite/batch_internal_test.go:364`, `TestMDelFunc` `cache/sqlite/batch_internal_test.go:404` | The provider contract at function granularity: file-mode CRUD, key-type parity through the stringified SQL keyspace, marshal failure, closed-cache taxonomy, idempotent Close, Optimizable surface, TTL resolution, idempotent bootstrap, a corrupted row reporting a decode error rather than `ErrMiss`, and canceled-context handling — plus the chunk/dedup/placeholder helpers and MGet/MSet/MDel units. |
| **:memory:** | `TestMemoryRoundTrip` `cache/sqlite/sqlite_test.go:23`; `TestMemoryIsolation` `cache/sqlite/sqlite_test.go:454`; `TestTTLExpiry` `cache/sqlite/sqlite_test.go:353`; `TestPoolPinned` `cache/sqlite/sqlite_internal_test.go:231` | Set/Get/Delete round-trip in `:memory:` mode and `ErrClosed` after Close; two separate `WithMemory()` instances do not see each other's keys; per-call TTL expiry works in memory mode; `MaxOpenConnections == 1` in both memory and file mode, which is what makes `:memory:` correct per pooled connection. |
| **reopen/persistence** | `TestReopenPersistence` `cache/sqlite/sqlite_test.go:587`; `TestExpiryAcrossRestart` `cache/sqlite/sqlite_test.go:621`; `TestSidecarRemoval` `cache/sqlite/sqlite_test.go:564`; `TestSweepOnOpen` `cache/sqlite/sqlite_internal_test.go:116` | Set → Close → reopen → Get returns both the no-TTL and the TTL value, and a further write/read after reopen proves the WAL-derived state is consistent; expired entries stay hidden across a restart (reclaimed by the open sweep or filtered by the read); `-wal`/`-shm` are gone after the last Close while the database file remains; rows seeded before `New` are swept synchronously on open while unexpired rows survive. |
| **concurrent** | `TestConcurrentBatchOpsRace` `cache/sqlite/sqlite_test.go:203`; `TestBeginContentionBounded` `cache/sqlite/sqlite_internal_test.go:386`; `TestTwoProcessContention` `cache/sqlite/twoprocess_e2e_test.go:413`; `TestTwoProcessCrashRecovery` `cache/sqlite/twoprocess_e2e_test.go:475` | 8 goroutines mixing Set/Get/Delete/MGet/MSet/MDel/GetOrSet on one provider in both file and `:memory:` mode complete without a data race on the pinned single-connection pool (its doc comment names it the CONC-03 evidence; CI runs the package under `-race`); a second write lock surfaces as a busy-classified error at BEGIN bounded by `busy_timeout`, with no provider-side retry beyond the open window, and recovery once the lock is released; two separate processes contend and recover from a crash on the same database file (the `e2e`-tagged lane). |
| **sweep** | `TestSweepOnOpen` `cache/sqlite/sqlite_internal_test.go:116`; `TestSweepFailureSwallowed` `cache/sqlite/sqlite_internal_test.go:156`; `TestPeriodicSweep` `cache/sqlite/sqlite_internal_test.go:170`; `TestSweeperDisabled` `cache/sqlite/sqlite_internal_test.go:195`; `TestReadsNeverDeleteRows` `cache/sqlite/sqlite_internal_test.go:98` | The open-time sweep is synchronous and removes only expired rows; a sweep against a broken handle logs and returns without propagating anything; the opt-in ticker reclaims rows that arrived after open and Close is idempotent; an unset or non-positive interval creates no sweeper, ticker or goroutine at all; the read path is filter-only — an expired Get reports `ErrMiss` and never deletes the row. |

Every path and line above was re-read on this tree before writing it down; no row was added for a
class that lacked coverage, because none lacked it.

## Gate output

Verbatim tail of the `make coverage-quick` run performed in this task (exit 0, numbered so nothing is
paraphrased):

```text
ok  	github.com/guionardo/go/cache/sqlite	5.232s	coverage: 92.6% of statements
File coverage threshold (70%) satisfied:	PASS
Package coverage threshold (80%) satisfied:	PASS
Total coverage threshold (75%) satisfied:	PASS
Total test coverage: 82.1% (2628/3201)
```

`cache/sqlite` sits at **92.6% of statements** — 12.6 points above the 80% package threshold — with no
override. The repo total for this profile is **82.1% (2628/3201)**, 7.1 points above the 75% total
threshold. Note on reproduction: two consecutive `make coverage-quick` runs in this task reported the
total denominator as 3201 with 2628 and 2629 covered statements — a one-statement jitter in an
unrelated package, still 82.1% and still PASS on all three thresholds. `cache/sqlite`'s own 92.6% was
identical in both runs. Do not treat the ±1 statement as drift in the sqlite provider. The E2E-merged profile (`make coverage`, `./.testcoverage.yml`) needs Docker and is not
reachable on this machine; it is not the AGENTS.md pre-commit gate, and the numbers from the two
profiles are different measurements of the same passing gate — do not try to reconcile them.

## No override

Both coverage configs were grepped after stripping comments:

```bash
$ grep -v '^#' .testcoverage-quick.yml | grep -c 'cache/sqlite'
0
$ grep -v '^#' .testcoverage.yml | grep -c 'cache/sqlite'
0
```

`grep -n 'cache/sqlite' .testcoverage.yml .testcoverage-quick.yml` produces no matches at all (grep
exit 1). The override lists are exactly the shipped eight: `cache/memcache`, `cache/postgres`,
`cache/redis`, `cache/valkey`, `cmd/example-updater`, `release/swapper`, `mid` (50), `release` (70).
The sqlite provider appears in neither, so it is held to the full file 70 / package 80 / total 75
thresholds. Per D-14 no guard test or script enforces this — a future override would land as a
reviewable diff in a config file, which is the intended tripwire.

## Why the coverage gate inputs stay unchanged (D-13)

`make coverage` merges E2E coverage from `./cache/` **only** (`go test -tags=e2e ... ./cache/`), so
`cache/sqlite/twoprocess_e2e_test.go` (`//go:build e2e`) never enters that profile. Widening the merge
root to `./cache/...` looks like an obvious improvement and is deliberately **not** done here:

- the unit gate already passes with no override (`cache/sqlite` 92.6%, total 82.1%), so a wider merge
  buys no threshold headroom;
- E2E correctness is proven by its own dedicated three-OS CI step (the `SQLite two-process E2E`
  step), which is a stronger and more direct proof than folding E2E coverage into a percentage.

`.testcoverage.yml`, `.testcoverage-quick.yml` and the Makefile's coverage targets are therefore
byte-identical to what shipped. **Do not "fix" the merge root** — this paragraph exists so a later
agent does not reopen it.

## Not added in this phase

- **No test** for a named class that already had coverage — all five classes were already covered
  (D-12: audit, not churn).
- **No coverage-margin hardening** — no test was written to push a percentage higher.
- **No guard test or script** for the "no override" promise (D-14); the passing gate plus the
  reviewable config diff is the mechanism.
- **Shared provider-contract E2E registration** stays a recorded follow-up (RESEARCH Open Question 1)
  with no requirement behind it — it is not a QUAL-02 gap and nothing here asserts it.

*Phase: 17-Delivery Hardening · Plan 17-03 · requirement QUAL-02*

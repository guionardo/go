# Phase 15 Deferred Items

Items discovered during 15-02 execution that are pre-existing (not caused by this
plan's changes) and are deliberately not fixed here per the executor scope
boundary. Each carries enough evidence for Phase 16 (CONC-01/02) to act.

## DI-15-01: Concurrent first-open of a fresh file can defer SQLITE_BUSY (WAL conversion race)

**Discovered during:** 15-02 Task 3 (a `go test -cover` run failed
`TestBootstrapIdempotence/concurrent_first_open_all_usable`).

**Status:** Pre-existing at 15-01 (`3d92e70`) — reproduced 3/25 iterations in a
temporary worktree checked out at that commit, with zero 15-02 changes present.
Reproduced identically with 15-02's open-time sweep disabled, so the sweep is not
a contributing cause.

**Root cause (proven by isolation):** `PRAGMA journal_mode=WAL`, applied per
connection from the DSN, returns an **immediate** `SQLITE_BUSY (5)` when several
connections race the DELETE→WAL conversion of a brand-new database file. The
busy handler installed by `_busy_timeout=5000` is not consulted for this lock
path, so the failure surfaces in ~2.5% of concurrent connection establishments
and ~10% of 4-way concurrent first-open test iterations.

**Isolation evidence** (scratch diagnostic, removed after use; 40 iterations ×
4 workers = 160 connection pings per variant, all on one fresh path):

| DSN variant | ping SQLITE_BUSY |
|-------------|------------------|
| `_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate` (production DSN) | 4/160 |
| `_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate` (no journal_mode) | 0/160 |
| `_busy_timeout=5000&_journal_mode=WAL` | 3/160 |
| `_busy_timeout=5000` | 0/160 |

The driver (modernc.org/sqlite v1.60.1, `sqlite.go` `applyQueryParams`) applies
`busy_timeout` before `journal_mode`, so ordering is not the issue; SQLite
itself bypasses the busy handler for the exclusive lock taken during the WAL
mode conversion. Once the file is already in WAL mode, subsequent connections'
`journal_mode=WAL` pragmas are no-ops and do not fail.

**Impact:** `New` never fails (D-11), so the racing provider's handle carries a
deferred `database is locked (5) (SQLITE_BUSY)` and every operation on it errors
until discarded. The affected 15-01 test therefore fails nondeterministically
(~5–12% per full-suite run), which can also randomly fail `make coverage-quick`.

**Why not fixed here:**
- Pre-existing (scope boundary: log out-of-scope discoveries, do not fix them).
- The remedy is a bounded retry of connection establishment / the WAL conversion
  race, and `.planning/.../15-RESEARCH.md` explicitly assigns the retry policy
  to Phase 16: "The bounded retry backstop and the two-process spike are
  **Phase 16** (CONC-01/02) — do not build retry here." The same race exists
  cross-process, so it belongs in the CONC design.

**Recommended Phase 16 action:** Let the two-process spike (CONC-01) confirm
cross-process behavior, then design the retry policy to cover first-open WAL
conversion races (e.g., bounded retry around physical-connection establishment
in `open()` on `SQLITE_BUSY`), and re-strengthen
`TestBootstrapIdempotence/concurrent_first_open_all_usable` as the regression
test.

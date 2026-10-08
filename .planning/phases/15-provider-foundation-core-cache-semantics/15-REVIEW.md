---
phase: 15-provider-foundation-core-cache-semantics
reviewed: 2026-10-08T14:57:18Z
depth: standard
files_reviewed: 15
files_reviewed_list:
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
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 15: Code Review Report

**Reviewed:** 2026-10-08T14:57:18Z
**Depth:** standard
**Files Reviewed:** 15
**Status:** issues_found

## Summary

The phase delivers a new `cache/sqlite` provider: DSN/location resolution (`dsn.go`), functional options (`options.go`), fixed schema (`schema.go`), CRUD + TTL + lifecycle (`sqlite.go`), best-effort and opt-in sweeps (`sweep.go`), and a type-assertable `Optimizable` maintenance surface (`optimize.go`), plus docs, an example, README rows, and the `modernc.org/sqlite v1.60.1` / `modernc.org/libc v1.77.1` pins.

Verification performed during review (not just reading):
- `go vet ./cache/sqlite/...` and `go build ./cache/...` — clean;
- `go test ./cache/sqlite/... -race` — 85 tests pass; package coverage 92.4% (file-level all ≥66.7%, gate 70% applies to production files);
- `go mod verify` — all modules verified; the modernc pins are present and consistent;
- Driver-source cross-check of every DSN key used (`_busy_timeout`, `_journal_mode`, `_synchronous`, `_txlock`, `_pragma`) against vendored `modernc.org/sqlite@v1.60.1` — all are recognized and validated; `_txlock=immediate` is honored by `BeginTx`; memory-mode `PRAGMA wal_checkpoint(TRUNCATE)` returns a zero-busy row, so the documented no-op holds.

Security posture is solid: every SQL statement binds positional parameters (no string-built SQL, keys/values are never identifiers), paths are rejected on `?`/`#` before reaching the DSN, cache names are allow-listed to `[A-Za-z0-9._-]` with `.`/`..`/separators rejected, `WithAutoCheckpoint`'s only DSN variable is `strconv.Itoa` of an int, the pool is pinned to one connection, and the sweeper lifecycle is cancel-safe (single ticker, `Close` cancels then waits before closing the DB). No Critical findings.

The one material defect is the concurrent first-open race documented by the project itself as DI-15-01: I independently reproduced it at a higher rate than the deferral note (10/60 runs and 4/20 runs), with the exact error `cache/sqlite: database is locked (5) (SQLITE_BUSY)`. It is reported below as a Warning because the project has consciously deferred the fix to Phase 16 (CONC-01/02) with root-cause evidence — but it is a real robustness and CI-gate defect in shipped, tested behavior, not merely a quality nit.

## Narrative Findings (AI reviewer)

### Warnings

#### WR-01: Concurrent first-open of a fresh database can leave the provider permanently broken, and the covering test is nondeterministically red

**File:** `cache/sqlite/sqlite.go:95` (open/bootstrap path), `cache/sqlite/dsn.go:15` (`_journal_mode=WAL` in the file DSN), `cache/sqlite/sqlite_internal_test.go:462` (flaky assertion)

**Issue:** When several providers first-open the same brand-new database concurrently, the per-connection `PRAGMA journal_mode=WAL` applied from the DSN can return an immediate `SQLITE_BUSY` during the DELETE→WAL conversion; SQLite does not consult the busy handler for that lock path, so `_busy_timeout=5000` does not help. `open()` records the failure as `initErr` and `New` still returns a handle; every subsequent operation then fails with the deferred `cache/sqlite: database is locked (5) (SQLITE_BUSY)` until the handle is discarded (the returned provider is dead, with no in-provider retry). This is a supported scenario: same-host process/goroutine sharing is implied by the docs (`doc.go:36-38` only excludes network/cross-host sharing), and the phase's own test asserts that concurrent first-open must work.

Independent reproduction during this review:

```
go test ./cache/sqlite/... -run 'TestBootstrapIdempotence/concurrent_first_open_all_usable' -count=60 -v
→ 10 failures, all:  cache/sqlite: database is locked (5) (SQLITE_BUSY)
a second run: -count=20 → 4 failures
```

Because the failure surfaces only when tests are repeated, the phase's blocking gate (`make coverage-quick`, required by `AGENTS.md` before commits) is randomly red (~5–20% per run depending on scheduling and machine load). The project already found and root-caused this as **DI-15-01** in `deferred-items.md`, explicitly assigning the retry policy to Phase 16 (CONC-01/02). This review confirms the evidence and flags that shipping with the race means the phase's CI guarantee is nondeterministic until then.

**Fix:** Add the bounded retry around connection establishment/first use that `deferred-items.md` recommends for CONC-01/02 — e.g. in `open()`, retry the failing physical-connection step (bootstrap/verify) a small number of times with backoff when the error is SQLITE_BUSY, discarding the failed attempt each time:

```go
// sketch — Phase 16 owns the final policy
const maxOpenAttempts = 5
var err error
for attempt := 0; attempt < maxOpenAttempts; attempt++ {
    if err = c.bootstrap(ctx); err == nil {
        break
    }
    if !isBusyError(err) {
        break
    }
    time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
}
```

Until the retry lands, the concurrent-first-open regression test should not be left able to fail the blocking coverage gate nondeterministically (quarantine/skip with a DI-15-01 reference, or gate it behind an env var). The test itself is valuable and should be re-strengthened, not deleted, per the deferred-items recommendation.

### Info

#### IN-01: Every closed/deferred error carries the `cache/sqlite:` prefix twice

**File:** `cache/sqlite/sqlite.go:164-174`; sources at `cache/sqlite/dsn.go:38` and `cache/sqlite/dsn.go:44`

**Issue:** `ErrClosed` (`cache/sqlite: cache is closed`) and `ErrInvalidPath` (`cache/sqlite: invalid path`) already embed the provider prefix, then `check()` wraps them again with `fmt.Errorf("cache/sqlite: %w", ...)`. The result is user-visible double prefixes such as `cache/sqlite: cache/sqlite: cache is closed` and `cache/sqlite: cache/sqlite: invalid path: path must not contain '?' or '#'`. `errors.Is` still works and tests only assert `Contains`, so this is cosmetic, but it degrades the error contract documented as "all provider errors carry the `cache/sqlite:` prefix" (D-12).

**Fix:** Keep exactly one wrapping site. Either drop the prefix from the sentinel definitions (`ErrClosed = errors.New("cache is closed")`, `ErrInvalidPath = errors.New("invalid path")`) and let `check()`/operations add it once, or return `c.initErr` unwrapped from `check()`. Existing tests asserting `ErrorContains(err, "cache/sqlite:")` continue to pass either way.

#### IN-02: Invalid cache names are reported as `ErrInvalidPath` with an "invalid path" message

**File:** `cache/sqlite/dsn.go:43-45`; docs gap at `cache/sqlite/options.go:31-36`

**Issue:** `resolveLocation` rejects an unsafe `WithName` value with `fmt.Errorf("%w: invalid cache name %q", ErrInvalidPath, cfg.Name)`, producing `cache/sqlite: invalid path: invalid cache name "..."`. A cache name is not a path, the sentinel is documented as the `?`/`#` path-rejection error (D-12/STOR-06), and `WithName`'s doc comment never mentions what an invalid name returns. Callers doing `errors.Is(err, sqlite.ErrInvalidPath)` cannot distinguish a bad name from a bad path (relevant for telemetry/messages).

**Fix:** Introduce a dedicated sentinel and document it on `WithName`:

```go
// dsn.go
var ErrInvalidName = errors.New("cache/sqlite: invalid name")
// ...
return "", false, fmt.Errorf("%w: %q", ErrInvalidName, cfg.Name)
```

(If a single sentinel is a deliberate API decision, reword the message to `invalid path or name` and note it in `WithName`'s doc comment.)

---

_Reviewed: 2026-10-08T14:57:18Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_

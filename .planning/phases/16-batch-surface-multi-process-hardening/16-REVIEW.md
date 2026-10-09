---
phase: 16-batch-surface-multi-process-hardening
reviewed: 2026-10-08T23:17:33Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - .github/workflows/go.yml
  - cache/sqlite/batch.go
  - cache/sqlite/batch_internal_test.go
  - cache/sqlite/doc.go
  - cache/sqlite/dsn.go
  - cache/sqlite/dsn_internal_test.go
  - cache/sqlite/retry.go
  - cache/sqlite/retry_internal_test.go
  - cache/sqlite/schema.go
  - cache/sqlite/sqlite.go
  - cache/sqlite/sqlite_internal_test.go
  - cache/sqlite/sqlite_test.go
  - cache/sqlite/twoprocess_e2e_test.go
findings:
  critical: 0
  warning: 2
  info: 5
  total: 7
status: issues_found
---

# Phase 16: Code Review Report

**Reviewed:** 2026-10-08T23:17:33Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed the completed batch surface (MGet/MSet/MDel), the bounded open-retry
fix for DI-15-01, the DSN `journal_size_limit` pragma, the two-process E2E,
and the CI wiring. Cross-checked against the frozen `cache.BatchCache`
contract, the mem provider parity precedent, and decisions D-01..D-10 in
16-CONTEXT.md.

Verification performed (all green): `go test ./cache/sqlite/` (148 tests,
including the DI-15-01 regression `TestBootstrapIdempotence` and the
`TestBeginContentionBounded` D-07 no-retry proof), `-race` run, `go vet`,
`golangci-lint` (full enabled linter set), the `-tags=e2e` two-process E2E
(both arms), and coverage (package 92.1%, worst file 85% — the
`.testcoverage-quick.yml` gates hold). The implementation faithfully matches
the documented design: pre-marshal-then-transaction MSet atomicity is real
(no partial writes possible through the single upsert tx), MGet chunk failure
policy follows D-03, MDel fail-fast follows D-05, chunk math and the
`perRoleRows = 287` E2E arithmetic check out exactly.

No critical issues found. Two warnings on latent robustness of the two new
loop constructs (retry budget semantics, chunking guard), plus five
informational items. The retry/deadline semantics of `retryBusyWithin` are
the only place where the documented bound ("bounded by the same timeout",
doc.go / D-06) can be exceeded in a reachable path.

## Warnings

### WR-01: `retryBusyWithin` can overrun its budget by a full extra `fn` call

**File:** `cache/sqlite/retry.go:51-65`
**Issue:** The absolute deadline is only consulted *after* `fn` returns and
only once per iteration. The sequence: `fn` returns busy at T < deadline, the
`select` sleeps `backoff` and crosses the deadline, and the loop then invokes
`fn` one more time past the deadline. If `fn` (a `PingContext`) blocks in the
SQLite busy handler for the full `busy_timeout` — exactly the case D-06/doc.go
("bounded by the same timeout") claims to bound — the total open-window wait
becomes budget + backoff + one full `fn` duration, i.e. up to ~10 s for the
nominal 5 s budget (2×). `New` in that window rides a deferred error into
every operation, so a slow-but-recoverable first-open contention scenario is
turned into a false permanent failure at double the documented bound. The
unit test only exercises the fast-`fn` path (fake busy errors return
immediately), so the overrun is untested.
**Fix:** Hoist the deadline check to the top of the loop and return the last
busy error without calling `fn` again once the deadline has passed:

```go
func retryBusyWithin(ctx context.Context, budget, backoff time.Duration, fn func(context.Context) error) error {
	deadline := time.Now().Add(budget)
	var lastErr error

	for {
		if time.Now().After(deadline) && lastErr != nil {
			return lastErr // budget exhausted: surface the last busy error
		}

		err := fn(ctx)
		if err == nil || !isBusyError(err) || time.Now().After(deadline) {
			return err
		}

		lastErr = err

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}
```

Alternatively, select on a deadline timer in the `select` so the sleep cannot
cross it:

```go
timer := time.NewTimer(backoff)
defer timer.Stop()
// and in the select: case <-time.After(time.Until(deadline)): return lastErr
```

### WR-02: `chunksOf` hangs forever (infinite loop, unbounded allocation) on `size <= 0`

**File:** `cache/sqlite/batch.go:45-58`
**Issue:** With `size <= 0`, `start += size` never advances, so the loop
appends `keys[0:0]` chunks forever — an infinite loop that grows a slice
until OOM. Both production call sites pass the fixed `chunkSize` constant
(100), so this is latent today, but `chunksOf` is a standalone exported-shape
helper whose `size` parameter is already exercised by `TestChunksOf` with
arbitrary sizes, and D-01 explicitly contemplates the chunk size being
"adjustable later" — the first refactor that reads it from config (with a
zero-value default) turns a hang into a production outage. The edge case is
also untested (`TestChunksOf` covers only positive sizes).
**Fix:** Guard the parameter:

```go
func chunksOf[K comparable](keys []K, size int) [][]K {
	if len(keys) == 0 || size <= 0 {
		return nil
	}
	// ...
}
```

## Info

### IN-01: MSet marshal failure error does not identify the failing key

**File:** `cache/sqlite/batch.go:159-165`
**Issue:** When one value of a batch fails `json.Marshal`, the returned error
is `cache/sqlite: json: unsupported type: ...` with no indication of which key
failed. For a large batch (the atomicity test uses 2000 items) the caller
cannot tell which entry to fix without bisecting.
**Fix:** Include the key in the wrapping error:
`return fmt.Errorf("cache/sqlite: marshal key %q: %w", fmt.Sprint(key), err)`.

### IN-02: MGet logs a Warn for every call against a closed provider

**File:** `cache/sqlite/batch.go:76-79`
**Issue:** `MGetFunc` routes *both* the deferred open error and `ErrClosed`
through `logger().Warn("… batch get skipped …")`. A closed provider is a
normal shutdown condition (the batch API also has no error channel), and
during shutdown races every concurrent `MGet` can emit a warning — log spam
at the wrong level. (The sibling test name "closed_provider_returns_empty_map"
also implies a noise-free path.) DB/init failures legitimately deserve Warn;
`ErrClosed` does not.
**Fix:** Special-case the level:

```go
if err := c.check(); err != nil {
	if errors.Is(err, ErrClosed) {
		logger().Debug("cache/sqlite: batch get skipped", "error", err)
	} else {
		logger().Warn("cache/sqlite: batch get skipped", "error", err)
	}
	return result
}
```

### IN-03: Double `cache/sqlite:` prefix persists on the batch surface (Phase 15 IN-01, still open)

**File:** `cache/sqlite/sqlite.go:177-187` (callers: `batch.go:148-151`, `214-217`)
**Issue:** `check()` wraps `ErrClosed`/`ErrInvalidPath` (which already carry
the `cache/sqlite:` prefix) with `fmt.Errorf("cache/sqlite: %w", …)`, so the
new batch methods surface errors like
`cache/sqlite: cache/sqlite: cache is closed`. This is the Phase 15 IN-01
ledger entry; it was not addressed in this phase and the new `MSetFunc`/
`MDelFunc` error paths are affected just like the point API. Cosmetic but
confusing when matching error text downstream.
**Fix:** Drop the prefix when wrapping sentinel errors (`ErrClosed`,
`ErrInvalidPath`), i.e. `fmt.Errorf("%w", c.initErr)` / `fmt.Errorf("%w",
ErrClosed)` — the sentinels are already prefixed.

### IN-04: MSet does not dedup colliding stringified keys, so the stored value is nondeterministic

**File:** `cache/sqlite/batch.go:157-166`
**Issue:** `MGet`/`MDel` dedup inputs by their `fmt.Sprint` form (D-02,
`uniqueKeys`), but `MSetFunc` iterates the input map directly without
deduping. With `K = any` (or any custom `fmt.Stringer`) two distinct map keys
like `42` and `"42"` share the SQL keyspace row, and both upserts run inside
the same transaction — the surviving value depends on Go's randomized map
iteration order and is therefore nondeterministic across calls. (The mem
provider keeps both keys distinct, so the parity story is also inconsistent
here.)
**Fix:** Either dedup pairs by string form (same predicate as `uniqueKeys`)
before the transaction, or document that colliding forms are last-writer-wins
with unspecified order. Add one test pinning the chosen behavior.

### IN-05: `coverage` job pins unpinned third-party actions at old majors (pre-existing)

**File:** `.github/workflows/go.yml:48-49`
**Issue:** The coverage job uses `actions/checkout@v3` and
`actions/setup-go@v3` (no SHA pinning, and v3 is EOL relative to the `test`
job's `@v4`/`@v5`). This predates the phase — only the two new E2E/race steps
were added — but the file is in scope and unpinned action supply-chain
pinning is a standard hardening item.
**Fix:** Align with the test job: `actions/checkout@v4`, `actions/setup-go@v5`
with `go-version-file: go.mod`, and consider SHA-pinning both.

---

_Reviewed: 2026-10-08T23:17:33Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
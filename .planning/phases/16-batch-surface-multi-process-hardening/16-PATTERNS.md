# Phase 16: Batch Surface + Multi-Process Hardening - Pattern Map

**Mapped:** 2026-10-08
**Files analyzed:** 15 (5 new core, 8 modified core, 2 conditional/discretionary)
**Analogs found:** 12 / 15 (3 with no close analog — `retry.go`, `twoprocess_e2e_test.go`, and the optional spike artifact; use 16-RESEARCH.md patterns there)

> **Tracked-source gate:** every analog path below was verified with `git ls-files` on 2026-10-08 — all tracked source. No gitignored install/runtime mirrors are named. The only new paths are the five `cache/sqlite/*` files marked NEW (they do not exist yet).
>
> **Phase shape:** this phase EXTENDS the existing `cache/sqlite/` provider (no new package). Batch implementations move from the Phase 15 loop placeholders (`sqlite.go:263-305`) into a new `batch.go`; `retry.go` carries the D-06 bounded open retry; `twoprocess_e2e_test.go` carries the CONC-05 helper-process harness. `cache/cache.go`, `cache/concrete_cache.go`, and the other five providers are **frozen** — do not modify (only read as precedence).
>
> **Compile-coordination seams the planner must encode (atomic per commit):**
>
> | Seam | Why it must land together |
> |------|---------------------------|
> | `buildDSN(path, memory, autoCheckpointPages, journalSizeLimit)` signature change (`dsn.go:82`) | Callers `sqlite.go:95` and `dsn_internal_test.go:170-189` must be updated in the same commit or the package fails to build |
> | Placeholder removal (`sqlite.go:263-305`) + `batch.go` addition | Removing stubs before the real methods exist breaks the `cache.cacher` interface contract (`concrete_cache.go:13-23`) |
> | `retryBusy` call insertion in `open()` (`sqlite.go:100-107`) + `retry.go` | `sqlite_internal_test.go:442` (`concurrent_first_open_all_usable`) is red until the retry lands (DI-15-01) |

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cache/sqlite/batch.go` (NEW) | provider service | batch CRUD | `cache/mem/mem.go` (batch loop semantics) + `cache/postgres/postgres.go` (marshal/tx shape) | role+flow match |
| `cache/sqlite/retry.go` (NEW) | utility | request-response (bounded retry loop) | **none** — nearest pure-helper style is `cache/sqlite/dsn.go` (`resolveLocation`/`validName`) | none |
| `cache/sqlite/batch_internal_test.go` (NEW) | test (internal) | n/a | `cache/sqlite/sqlite_internal_test.go` | exact |
| `cache/sqlite/retry_internal_test.go` (NEW) | test (internal) | n/a | `cache/sqlite/dsn_internal_test.go` (pure-function table style) | role-match |
| `cache/sqlite/twoprocess_e2e_test.go` (NEW) | test (E2E harness) | event-driven / process orchestration | **none** — build-tag precedent `cache/cache_e2e_test.go:1-3` only | none |
| `cache/sqlite/sqlite.go` (mod) | provider service | CRUD + request-response | itself (`open`/`bootstrap`/`check`) | exact |
| `cache/sqlite/dsn.go` (mod) | utility | transform | itself (`dsnAutoCheckpointPrefix` append) | exact |
| `cache/sqlite/schema.go` (mod) | model (SQL constants) | n/a | itself (`SelectSQL`/`DeleteSQL`) | exact |
| `cache/sqlite/doc.go` (mod) | docs | n/a | itself (WAL bullet `:34-42`) | exact |
| `cache/sqlite/dsn_internal_test.go` (mod) | test (internal) | n/a | itself (`TestDSNBuildDSN`) | exact |
| `cache/sqlite/sqlite_internal_test.go` (mod) | test (internal) | n/a | itself (concurrent first-open `:400-468`) | exact |
| `cache/sqlite/sqlite_test.go` (mod) | test (black-box) | n/a | itself (batch smoke + race precedent `TestGetOrSetDedup`) | exact |
| `.github/workflows/go.yml` (mod) | config (CI) | n/a | itself (`test` matrix job `:10-29`) | exact |
| `cache/sqlite/options.go` (CONDITIONAL mod) | config | n/a | itself (`WithAutoCheckpoint` `:72-80`) | exact |
| `.planning/phases/16-.../16-SPIKE-FINDINGS.md` (CONDITIONAL NEW) | docs | n/a | `15-provider-foundation.../deferred-items.md` (evidence record style) | role-match |

---

## Pattern Assignments

### `cache/sqlite/batch.go` (provider service, batch CRUD) — NEW

**Analogs:** `cache/mem/mem.go` (batch loop semantics, resolveTTL-once) + `cache/postgres/postgres.go` (marshal + upsert shape) + `cache/sqlite/sqlite.go` (provider internals: `check`, error prefix, logger)

**Imports pattern** — provider files use one stdlib block, then the driver blank import in `sqlite.go` only, then the project import (`sqlite.go:3-19`). `batch.go` needs the subset: `context`, `encoding/json`, `fmt`, `log/slog`, `strings`, `time` (no `database/sql` — `sql.Tx` is returned by `c.db.BeginTx`, no type name needed). The driver blank import stays ONLY in `sqlite.go:16` — do not re-import `modernc.org/sqlite` (Pitfall: package-name collision, RESEARCH CE2b).

**Batch semantics precedent** — `cache/mem/mem.go:86-120`:
```go
// MGetFunc retrieves values for multiple keys under a single read lock.
func (c *memoryCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[K]V, len(keys))
	for _, key := range keys {
		if v, ok := c.store.get(key); ok {
			result[key] = v
		}
	}
	return result
}

// MSetFunc stores multiple key-value pairs under a single write lock.
func (c *memoryCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := c.resolveTTL(ttl...)   // <- resolve ONCE for the batch
	for key, value := range items {
		c.store.set(key, value, expiresAt)
	}
	return nil
}
```
Also `cache/mem/mem.go:112-120` (`MDelFunc` loop) — the semantics the SQLite batch must preserve: call it once, never per-key.

**Core pattern (RESEARCH Pattern 2/3 + CE3/CE4, probe-verified):** chunked `IN` queries per 100 keys (constants from `schema.go` additions), dedup BEFORE chunking (D-02), per-chunk `map[string]K` re-keying (Finding 5: `fmt.Sprint` is not invertible), MSet pre-marshal + one `BeginTx` + `tx.PrepareContext(UpsertSQL)` + `Commit` (D-04). Full verified code: `16-RESEARCH.md` CE3 (MGet), CE4 (MDel), Pattern 3 (MSet). `open()` already proves the tx shape: `sqlite.go:144-160` (`BeginTx` → `defer tx.Rollback()` → `ExecContext` → `Commit`).

**check() guard** — first statement of every op, `sqlite.go:164-174`:
```go
func (c *sqliteCache[K, V]) check() error {
	if c.initErr != nil {
		return fmt.Errorf("cache/sqlite: %w", c.initErr)
	}
	if c.closed.Load() {
		return fmt.Errorf("cache/sqlite: %w", ErrClosed)
	}
	return nil
}
```
- MGet: `check()` failure → `logger().Warn(...)` + return empty map (frozen signature has no error channel — D-03/Pitfall 5).
- MSet/MDel: `check()` failure → `return err`.

**Error handling pattern** — every DB error wraps with the package prefix, `sqlite.go:188-194, 224-226, 237-239`:
```go
if err != nil {
	return fmt.Errorf("cache/sqlite: %w", err)
}
```
- MSet: ANY error (marshal, BeginTx, Prepare, Exec, Commit) rolls back and returns wrapped (D-04); pre-marshal loop before `BeginTx` (nothing written on marshal failure).
- MDel: first chunk error → wrapped return (D-05 fail-fast; idempotent caller retry).
- MGet: chunk error → `logger().Warn("cache/sqlite: batch get chunk failed", "keys", len(chunk), "error", err)` + `continue`; row decode failure → `logger().Debug(...)` + `continue` (CONTEXT specifics: decode = data hygiene at debug; DB = warn). Never log values; key at debug is acceptable per Phase 15 V7 note.

**Logger pattern** — `sqlite.go:51-53` + `sweep.go:11-15` (`logger().Warn("cache/sqlite: sweep failed", "error", err)`).

**SQL constants to consume** — `schema.go:26-40` (existing `SelectSQL`/`UpsertSQL`/`DeleteSQL`; new `MGetSelectSQL`/`MDelSQL` added there, see below).

**Chunking helpers (RESEARCH Pattern 1)** — `chunkSize = 100` (D-01), `uniqueKeys`, `chunksOf`, `inPlaceholders` (only `?` markers generated — never caller data, V5). Probe: 100 keys + `now` bound = 101 params, driver limit 32766.

---

### `cache/sqlite/retry.go` (utility, bounded retry loop) — NEW — NO ANALOG

**Analog:** none — the repo contains no retry/backoff helper (`grep` for `retry|backoff|time.After` in non-test Go: zero hits). Nearest *style* analog for a small pure-function seam: `cache/sqlite/dsn.go:32-56` (`resolveLocation`) and `:61-76` (`validName`) — pure, deterministic, unit-testable without a DB.

**Use RESEARCH Pattern 4 / Code Example 2 verbatim:**
```go
type busyErr interface{ Code() int }

// isBusyError classifies SQLITE_BUSY by primary code (extended codes: &0xff).
func isBusyError(err error) bool {
	var be busyErr
	return errors.As(err, &be) && be.Code()&0xff == 5
}

const busyTimeout = 5 * time.Second // mirrors _busy_timeout=5000 in dsn.go:15

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

**Conventions to respect:**
- `busyTimeout = 5 * time.Second` must stay in sync with the literal `_busy_timeout=5000` in `dsn.go:15` — the DSN is a const string, so document the pairing (planner may derive the DSN from a shared ms constant if preferred).
- Magic numbers: the repo marks literals `//nolint:mnd` (precedent `cache/mem/mem.go:22-24`) or hoists them into named constants; `5 * time.Millisecond` likely needs a named const or the nolint.
- No driver import: `errors.As` against the interface keeps `import _ "modernc.org/sqlite"` blank and the classifier unit-testable with a fake error (Finding 3, CE2/2b).
- `contextcheck`-safe (no goroutines; context flows into `fn`).
- Alternative allowed by RESEARCH structure: the helpers may live in `sqlite.go` instead of a new file — if so, keep the same names and the matching test file moves accordingly.

**Call site** (modifies `sqlite.go`, see below): `retryBusy(ctx, c.db.PingContext)` between pool pinning (`sqlite.go:100-103`) and `bootstrap` (`sqlite.go:107`). RESEARCH CE1.

---

### `cache/sqlite/batch_internal_test.go` (test, internal) — NEW

**Analog:** `cache/sqlite/sqlite_internal_test.go` (exact — same `package sqlite`, same harness).

**Harness pattern** — `sqlite_internal_test.go:54-71`:
```go
func newInternalProvider(t *testing.T, opts ...Option) *sqliteCache[string, string] {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	c := &sqliteCache[string, string]{
		defaultTTL:    cfg.DefaultTTL,
		sweepInterval: cfg.SweepInterval,
	}
	c.initErr = c.open(t.Context(), cfg)

	t.Cleanup(func() { _ = c.CloseFunc() })

	return c
}
```

**Existing helpers to reuse (do not re-declare — single-ownership):**
- `keyRowCount(db, key)` / `countKeyRows(t, db, key)` — `sqlite_internal_test.go:381-398` (raw row count bypassing expiry filter; the no-partial-writes invariant test needs exactly this).
- `readAutoCheckpoint(t, c)` — `:324-331` (shape for a `readJournalSizeLimit` read-back helper).
- Table-driven style with `t.Parallel()` per subtest — `TestResolveTTL` `:73-94`, `TestSweepOnOpen` `:116-152`.
- Seeding rows straight through the handle with raw SQL — `:104`, `:127-131` (`c.db.ExecContext(t.Context(), UpsertSQL, ...)`).

**Tests to write (RESEARCH Pitfalls 4 + Pattern 1):** dedup (duplicate keys query once), chunk boundary at 100/101/200 keys, `inPlaceholders` output, decode-skip (seed `{not-json` like `:520` and assert chunk result omits only that row), empty-input no-ops (D-02), and `TestMSetNoPartialWrites` (2000 pairs, ctx canceled ~1 ms in, assert stored count is 0 **or** 2000, repeat 3-5×).

---

### `cache/sqlite/retry_internal_test.go` (test, internal) — NEW

**Analog:** `cache/sqlite/dsn_internal_test.go` (pure-function table style) + `sqlite_internal_test.go:73-94`.

**Fake-error classifier pattern (RESEARCH CE2):**
```go
type fakeErr struct{ code int }

func (f fakeErr) Error() string { return fmt.Sprintf("fake %d", f.code) }
func (f fakeErr) Code() int     { return f.code }

assert.True(t, isBusyError(fakeErr{code: 5}))
assert.True(t, isBusyError(fakeErr{code: 517})) // SQLITE_BUSY_SNAPSHOT (extended)
assert.False(t, isBusyError(fakeErr{code: 1}))  // SQLITE_ERROR
assert.False(t, isBusyError(errors.New("plain")))
```
Plus `retryBusy` loop-bound tests with a fake `fn` yielding N busy errors then nil (assert loop count / deadline / ctx-cancel short-circuit). Deterministic, no DB — the behavioral proof stays in `sqlite_internal_test.go` (see below).

---

### `cache/sqlite/twoprocess_e2e_test.go` (test, E2E harness) — NEW — NO ANALOG

**Analog:** **none** — verified by grep: no `func TestMain`, no `os.Args[0]` re-exec, no helper-process pattern anywhere in the repo. Only partial precedents:
- Build-tag + external-package shape: `cache/cache_e2e_test.go:1-3` (`//go:build e2e`, `package cache_test`).
- `os/exec` usage: `mid/machineid_darwin.go:11`, `mid/machineid_linux.go:43`, `mid/machineid_windows.go:24` (plain command execution — not re-exec, not useful as a pattern).
- E2E run wiring: `Makefile:90-93` (`test-e2e`, `-tags=e2e -run 'TestCacheE2E'`; note it targets `./cache/` only, so the new sqlite E2E is CI-step-owned, added below).

**Use RESEARCH Pattern 5 / Code Example 5 verbatim (probe-verified shape):**
- `//go:build e2e`, `package sqlite_test`.
- `TestMain` intercepts the child when `os.Getenv(helperEnv) == "1"` before `m.Run()`; parent spawns `exec.Command(os.Args[0])` with role/db env vars.
- Env constants: `SQLITE_E2E_HELPER`, `SQLITE_E2E_DB`, `SQLITE_E2E_ROLE`, `SQLITE_E2E_CRASH`, `SQLITE_E2E_MARKER`.
- Child drives the SHIPPED provider via `sqlite.New[string, string](sqlite.WithPath(...))` (the retry lives in the provider, not the harness).
- Contention arm: exit codes 0, then reopen and assert `PRAGMA integrity_check` == `ok` + `journal_mode` == `wal` + per-role final keys present / deleted keys absent.
- Crash arm: crasher holds `BEGIN IMMEDIATE` with an uncommitted row, writes marker file, blocks; parent `require.Eventually` marker → `Process.Kill()` → `Wait()`; assert survivor finishes, uncommitted row count 0, integrity ok.
- Capture child stdout/stderr in `bytes.Buffer` for CI diagnosis; kill children in `defer`/cleanup; every child `defer db.Close()` (Pitfall 8, Windows handles).

**Windows/cleanup discipline (Pitfall 8):** assert sidecar/lock cleanup with `require.Eventually`, never immediately; never assert Unix permissions.

---

### `cache/sqlite/sqlite.go` (provider, CRUD) — MODIFIED

**Analog:** itself.

**Change 1 — remove placeholders `:263-305`** (`MGetFunc`/`MSetFunc`/`MDelFunc` loop stubs with `// Phase 16:` comments). The real methods move to `batch.go` on the same receiver `*sqliteCache[K, V]`. Must land in the same commit as `batch.go` (interface contract `concrete_cache.go:13-23`).

**Change 2 — D-06 retry wiring in `open()`**, between pool pinning (`:100-103`) and `bootstrap` (`:107-112`). RESEARCH CE1:
```go
db.SetMaxOpenConns(1)   // existing (sqlite.go:100-103) stays
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
db.SetConnMaxIdleTime(0)
c.db = db

// D-06: fresh-file WAL conversion races another process's first open and can
// return immediate SQLITE_BUSY with the busy handler bypassed (DI-15-01).
if err := retryBusy(ctx, c.db.PingContext); err != nil {
	_ = db.Close()
	c.db = nil
	return err // deferred via New's initErr (sqlite.go:72) — New never fails
}

if err := c.bootstrap(ctx); err != nil { /* existing: close + return */ }
```

**Change 3 — `buildDSN` call site `:95`** gains the 4th arg (`journalSizeLimitBytes`), same commit as `dsn.go`.

**Do not touch:** `New` never-failing contract (`:62-78`), `check()` (`:164-174`), point-op CRUD (`:179-242`), `CloseFunc` (`:248-261`), `resolveTTL` (`:309-319`) — batch parity depends on them as-is.

---

### `cache/sqlite/dsn.go` (utility, transform) — MODIFIED

**Analog:** itself — direct string-shape mirror of the existing autocheckpoint append.

**Constants + buildDSN pattern (`dsn.go:14-24, 82-93`):**
```go
const (
	dsnSuffixFile   = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"
	dsnSuffixMemory = "?_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate"

	dsnAutoCheckpointPrefix = "&_pragma=wal_autocheckpoint("
	// ...
)

func buildDSN(path string, memory bool, autoCheckpointPages int) string {
	if memory {
		return path + dsnSuffixMemory
	}
	dsn := path + dsnSuffixFile
	if autoCheckpointPages > 0 {
		dsn += dsnAutoCheckpointPrefix + strconv.Itoa(autoCheckpointPages) + ")"
	}
	return dsn
}
```

**D-08 addition (RESEARCH CE6):** new `dsnJournalSizeLimitPfx = "&_pragma=journal_size_limit("` + package const `journalSizeLimitBytes = 64 << 20`; `buildDSN` gains `journalSizeLimit int64` and appends `dsnJournalSizeLimitPfx + strconv.FormatInt(journalSizeLimit, 10) + ")"` when `> 0`. Memory mode appends neither (Pitfall 6: `_pragma` values are compile-time constants only — never caller strings). Probe: both `_pragma` params coexist and read back `67108864` / `500`.

**Recommendation (RESEARCH OQ1 / A2):** ship as a `buildDSN` param with the constant default, **no new exported option** — unless the planner reads D-08 as requiring a zero-value caller path, in which case `WithJournalSizeLimit` mirrors `WithAutoCheckpoint` exactly (see `options.go` below).

---

### `cache/sqlite/schema.go` (model, SQL constants) — MODIFIED

**Analog:** itself (`schema.go:26-40` style: constant templates, bound positional params, expiry filtered in SQL).

**Existing style to mirror:**
```go
// SelectSQL fetches a non-expired value by key. Expiration is filtered in SQL;
// reads never delete rows.
SelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`

// DeleteSQL removes a key. Deleting a missing key is a no-op.
DeleteSQL = `DELETE FROM cache_entries WHERE cache_key = ?`
```

**New constants (RESEARCH Pattern 2, probe-verified at 101 params):**
```go
// MGetSelectSQL fetches one 100-key chunk. %s is ONLY the "?,?,..." marker
// list produced by inPlaceholders — never caller data.
MGetSelectSQL = `SELECT cache_key, value FROM cache_entries
WHERE cache_key IN (%s) AND (expires_at IS NULL OR expires_at > ?)`

// MDelSQL deletes one 100-key chunk; missing keys are no-ops (BATCH-03).
MDelSQL = `DELETE FROM cache_entries WHERE cache_key IN (%s)`
```
**Finding 5 deviation to encode:** the chunk SELECT must return **both** `cache_key` and `value` (WITHOUT ROWID + TEXT PRIMARY KEY has no rowid shortcut) — this is intentional and differs from single-key `SelectSQL` which selects `value` only. Expiry predicate is folded into SQL — expired rows are absent by query, no code-level skip needed.

---

### `cache/sqlite/doc.go` (docs) — MODIFIED

**Analog:** itself — extend the existing WAL/local-storage bullet (`doc.go:34-42`):
```go
// Data and state: file-backed entries persist across process runs. Close
// closes the database, which checkpoints the WAL and removes the -wal/-shm
// sidecars on the last connection. File mode verifies journal_mode (WAL) at
// open; a mismatch only logs a warning. WAL requires local storage: network
// filesystems and cross-host sharing are unsupported.
```
**CONC-04 addition:** same-host multi-process sharing is supported and serialized by WAL + `busy_timeout` (contention surfaces as a bounded error at BEGIN); network filesystems / cross-host remain unsupported. Also note (Pitfall 5 / A5): a canceled MGet returns partial results — the frozen signature has no error channel. Keep prose style of the surrounding paragraph (no new sections needed).

---

### `cache/sqlite/dsn_internal_test.go` (test, internal) — MODIFIED

**Analog:** itself — `TestDSNBuildDSN` (`:164-190`) is the single owner of buildDSN equality cases; **extend, do not re-declare** (CONTEXT single-ownership rule).

**Existing shape (`:164-190`):**
```go
func TestDSNBuildDSN(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "cache.db")
	fileSuffix := "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"

	assert.Equal(t, filePath+fileSuffix, buildDSN(filePath, false, 0))
	assert.Equal(t, filePath+fileSuffix+"&_pragma=wal_autocheckpoint(200)", buildDSN(filePath, false, 200))
	// ... memory-mode cases
}
```
Every call site gains the 4th arg (0 for no-limit) and new cases assert: both pragmas appended when both positive (order: autocheckpoint then journal_size_limit — append order in `buildDSN`), journal-only when autocheckpoint zero, none when journal ≤ 0, and memory mode omits it.

---

### `cache/sqlite/sqlite_internal_test.go` (test, internal) — MODIFIED

**Analog:** itself — the concurrent first-open regression (`:400-468`). DI-15-01 record (`deferred-items.md:53-58`) explicitly assigns this test as the regression.

**Keep the shape (Pitfall 9) — change only what the retry makes deterministic:**
```go
func probeConcurrentOpen(t *testing.T, path string) error { ... } // :400-422

t.Run("concurrent_first_open_all_usable", func(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "concurrent.db")
	const workers = 4
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := range errs {
		wg.Go(func() { errs[i] = probeConcurrentOpen(t, path) })
	}
	wg.Wait()
	for i, err := range errs {
		require.NoErrorf(t, err, "worker %d", i) // must now be deterministic green
	}
	...
})
```
**Optional strengthening (RESEARCH Pattern 4):** raise workers to 8 and/or verify locally with `-count=40` (probe: 0/240 failures with retry); if adding an internal assertion that the retry actually engaged, keep it in a separate unit test with the fake seam — do not weaken the behavioral test. Also add a `journal_size_limit` read-back subtest next to `TestAutoCheckpointReadBack` (`:291-331`) using the same `readAutoCheckpoint` helper shape.

---

### `cache/sqlite/sqlite_test.go` (test, black-box) — MODIFIED

**Analog:** itself.

**Replace/extend the batch smoke test** — `TestBatchPlaceholderSmoke` (`:49-67`) becomes real black-box batch semantics (happy path, missing keys absent, expiry respected via `MGet`, parity with `mem` semantics, dedup/no-op inputs).

**Existing black-box conventions to reuse:**
- Construction + cleanup: `sqlite.New[string, string](sqlite.WithMemory())` / `WithPath(filepath.Join(t.TempDir(), ...))` + `t.Cleanup(func() { _ = c.Close() })` (`:24-26`, `:69-73`).
- `require.Eventually` for TTL effects (`:116-120`, `:136-140`) — do not sleep-assert.
- Marshal failure stored nothing (`:280-294`) — the MSet pre-marshal atomicity analog.
- **Race test precedent (CONC-03, RESEARCH Pattern 6):** `TestGetOrSetDedup` (`:161-202`) is the goroutine-count + `sync.WaitGroup` + shared `start` channel pattern; add `TestConcurrentBatchOpsRace` (~8 goroutines × bounded mixed `Get/Set/Delete/MGet/MSet/MDel` loop, file mode + `:memory:` subtests, `t.Context()`).
- Wrapped-error assertions: `require.ErrorContains(t, err, "cache/sqlite:")` (`:289`, `:452`).

---

### `.github/workflows/go.yml` (config, CI) — MODIFIED

**Analog:** itself — the `test` matrix job (`:10-29`), `runs-on: ${{ matrix.os }}`, steps end at `:28-29`:
```yaml
      - name: Run tests
        run: go test -v ./...
```
**Add after "Run tests" (RESEARCH Pattern 7):**
```yaml
      - name: SQLite two-process E2E
        run: go test -tags=e2e -run 'TestTwoProcess' ./cache/sqlite/ -v -count=1 -timeout=180s
      - name: SQLite race detector
        if: runner.os != 'Windows'   # -race needs a C toolchain; keeps the matrix green
        run: go test -race ./cache/sqlite/ -count=1 -timeout=180s
```
Runs on all three OSes per D-09; race gated to linux+macos per RESEARCH A6/OQ3. The untagged `go test -v ./...` stays as-is (the e2e-tagged harness is invisible to it).

---

### `cache/sqlite/options.go` (config) — CONDITIONAL, likely unchanged

**Analog:** itself — `WithAutoCheckpoint` (`:72-80`) is the exact template if the planner decides D-08 needs a caller-facing zero-value option (`WithJournalSizeLimit(int64)`):
```go
// WithAutoCheckpoint tunes WAL automatic checkpointing for file-mode caches:
// pages > 0 sets wal_autocheckpoint to that page count; the zero value keeps
// SQLite's default (1000 pages) — no sentinel semantics (D-07). Non-positive
// values append nothing, and memory mode ignores the option.
func WithAutoCheckpoint(pages int) Option {
	return func(cfg *Config) {
		cfg.AutoCheckpoint = pages
	}
}
```
RESEARCH recommendation (OQ1/A2): **no new option** — ship `journalSizeLimitBytes = 64 << 20` as the default constant and keep the param-level zero contract. If implemented, mirror the docs wording ("no sentinel semantics", memory mode ignores) and add cases to `options_internal_test.go`.

---

### `.planning/phases/16-batch-surface-multi-process-hardening/16-SPIKE-FINDINGS.md` — CONDITIONAL NEW

**Analog (style):** `15-provider-foundation-core-semantics/deferred-items.md` evidence-record format (root cause, isolation table, impact, recommended action) and `15-VERIFICATION.md` deferred-evidence blocks. D-11 discretion: the spike may instead be folded into the E2E task description / commit message. If written, record: harness shape, raw vs retry failure counts, integrity/row-count evidence (RESEARCH Finding 1 numbers).

---

## Shared Patterns

### Deferred initErr + never-failing New
**Source:** `cache/sqlite/sqlite.go:62-78` (`New` records `c.open(...)` into `initErr`) and `:164-174` (`check`).
**Apply to:** `batch.go` (MGet: log+empty map; MSet/MDel: return the wrapped error).
**Behavior test precedent:** `sqlite_internal_test.go:471-492` (`TestDeferredErr` — every surface returns the deferred error; `Close` still nil).

### Error wrapping with package prefix
**Source:** `cache/sqlite/sqlite.go:188-194, 224-226, 237-239`; `optimize.go:74-79`.
**Apply to:** every new DB error in `batch.go`, `retry.go` call site.
```go
return fmt.Errorf("cache/sqlite: %w", err)
```

### Logger (sync.OnceValue + warn/debug discipline)
**Source:** `cache/sqlite/sqlite.go:51-53`; usage `sweep.go:11-15`.
**Apply to:** MGet chunk failures (warn, include key count — never key strings/values), decode skips (debug). No new logging surface.

### resolveTTL once per batch
**Source:** `cache/mem/mem.go:104` (`expiresAt := c.resolveTTL(ttl...)` outside the loop); `postgres.go:191`.
**Apply to:** `MSetFunc` — call `c.resolveTTL(ttl...)` once; one expiry binds the whole batch (D-04); missing TTL stores SQL NULL via `var exp any` (`sqlite.go:219-222`).

### Transaction discipline (BEGIN IMMEDIATE via DSN)
**Source:** `sqlite.go:144-160` (bootstrap tx: `BeginTx` + `defer tx.Rollback()` + `Commit`); `_txlock=immediate` rides the DSN (`dsn.go:15`).
**Apply to:** `MSetFunc` only. Never add retries around BEGIN (D-07); contention surfaces as the bounded busy error.

### TDD + co-located tests + single-ownership
**Source:** Phase 15 pattern (`dsn_internal_test.go`, `sqlite_internal_test.go`); repo style: `package sqlite` for internals, `package sqlite_test` for black-box, testify assertions, `t.Parallel()` subtests.
**Apply to:** all five test files. Extend existing matrices; never re-declare a test/helper another file owns.

### Coverage gate
**Source:** `.testcoverage-quick.yml` (`file: 70`, `package: 80`, `total: 75`; **no `cache/sqlite` override**) + `Makefile:119-122` (`coverage-quick`).
**Apply to:** every commit. The E2E harness must stay `//go:build e2e` so it never enters the unit lane (Pitfall 7; precedent `cache/cache_e2e_test.go:1-3`). `make coverage` (`Makefile:105-117`) runs its e2e lane against `./cache/` only — the sqlite two-process E2E is owned by the new CI step.

### Function options: zero value = SQLite default, no sentinel
**Source:** `options.go:72-80` (`WithAutoCheckpoint`) + `dsn.go:87-91` (positive appends, zero skips).
**Apply to:** any `journal_size_limit` surface — additive, local-only, compile-time constant values.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `cache/sqlite/retry.go` | utility | request-response (retry loop) | No retry/backoff helper exists anywhere in the repo (grep-verified). Use 16-RESEARCH Pattern 4 / CE2; style-borrow from `dsn.go` pure helpers. |
| `cache/sqlite/twoprocess_e2e_test.go` | test (E2E harness) | event-driven / process orchestration | No `TestMain` helper-process/re-exec pattern in the repo (grep-verified). Use 16-RESEARCH Pattern 5 / CE5; only the `//go:build e2e` convention has a precedent (`cache/cache_e2e_test.go:1-3`). |
| `.planning/.../16-SPIKE-FINDINGS.md` (conditional) | docs | n/a | New artifact type for this repo; borrow evidence-record format from `15-.../deferred-items.md` / `15-VERIFICATION.md`. |

---

## Metadata

**Analog search scope:** `cache/sqlite/`, `cache/mem/`, `cache/postgres/`, `cache/` root (`cache.go`, `concrete_cache.go`, `errors.go`, `cache_e2e_test.go`), `.github/workflows/go.yml`, `Makefile`, `.testcoverage-quick.yml`, `.opencode/skills/spike-findings-go/`; repo-wide greps for `func TestMain`, `GO_WANT_HELPER_PROCESS`, `os.Args[0]`, `exec.Command`, `retry|backoff|time.After`, `errors.As|Code()`.
**Files scanned:** 12 `cache/sqlite` files + 8 analogs/configs + 4 repo-wide pattern searches.
**Tracked-source verification:** all named analog paths confirmed with `git ls-files` (2026-10-08). No mirror/gitignored paths used.
**Pattern extraction date:** 2026-10-08
**Primary pattern source beyond the codebase:** `16-RESEARCH.md` Patterns 1–7 and Code Examples 1–6 (probe-verified against `modernc.org/sqlite` v1.60.1 — no new dependencies; `go.mod` untouched).

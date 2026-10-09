# Architecture Research

**Domain:** Embedded SQLite-backed cache provider (`cache/sqlite`) for the `github.com/guionardo/go` cache subsystem
**Researched:** 2026-10-08
**Confidence:** MEDIUM overall — codebase wiring verified locally (HIGH); SQLite/driver mechanics cross-checked against official docs plus independent implementations (MEDIUM); multi-process `SQLITE_BUSY` behavior under contention remains an open spike (LOW, flagged below).

## Scope and Constraints

This research covers **only the new `cache/sqlite` provider**. The existing cache architecture is frozen (v1.6 D-02) and was not re-researched:

- `cache.Cache[K, V]` / `cache.BatchCache[K, V]` + `cache.NewConcreteCache` (`cache/cache.go`, `cache/concrete_cache.go`)
- The unexported `cacher[K, V]` primitive (`GetFunc/SetFunc/DeleteFunc/CloseFunc` + `MGetFunc/MSetFunc/MDelFunc`) — the provider satisfies it **structurally** (Go implicit interfaces), exactly like `cache/postgres`
- Singleflight dedup inside `concreteCache.GetOrSet`, `cache.ErrMiss`, `fmt.Sprint` key encoding, `encoding/json` value encoding
- `cache/postgres` is the closest SQL-backed analog and the behavioral parity target

**Hard constraint:** new package only. No changes to `cache/` root files or existing providers. The only non-new file changes are `go.mod` / `go.sum` and package-index docs (`README.md`, `AGENTS.md` table).

## Standard Architecture

### System Overview

```
┌────────────────────────────────────────────────────────────────────────┐
│  Consumer code                                                         │
│  cache.BatchCache[K, V]  =  Cache[K, V] + MGet / MSet / MDel           │
├────────────────────────────────────────────────────────────────────────┤
│  cache/concrete_cache.go  (EXISTING — untouched)                       │
│  Get / Set / Delete / GetOrSet(singleflight) / Close / MGet/MSet/MDel  │
│  delegates every call through the unexported cacher[K, V] interface    │
├────────────────────────────────────────────────────────────────────────┤
│  cache/sqlite  (NEW)                                                   │
│  sqliteCache[K, V] implements cacher[K, V] structurally                │
│  GetFunc · SetFunc · DeleteFunc · MGetFunc · MSetFunc · MDelFunc ·     │
│  CloseFunc        +  TTL resolution  +  lazy expiry on read            │
│  options.go (Config) │ schema.go (DDL) │ sweep.go (sweep-on-open)      │
├────────────────────────────────────────────────────────────────────────┤
│  database/sql  (*sql.DB pool pinned to MaxOpenConns(1), MaxIdleConns(1))│
│  DSN pragmas applied by the driver on EVERY new connection             │
├────────────────────────────────────────────────────────────────────────┤
│  modernc.org/sqlite  ("sqlite" driver, pure Go, SQLite 3.53.4)         │
├────────────────────────────────────────────────────────────────────────┤
│  SQLite file + sidecars (-wal, -shm)        or        :memory:         │
│  WAL journal · busy_timeout(N) · synchronous=NORMAL · _txlock=immediate│
└────────────────────────────────────────────────────────────────────────┘
```

Key flow insight: `New` returns `cache.NewConcreteCache[K, V](c)` — so the provider only implements the 7-method `cacher` primitive and inherits `GetOrSet` singleflight dedup, `Cache`, and `BatchCache` for free. Zero integration code in the root package.

### Component Responsibilities

| Component | Responsibility | Implementation |
|-----------|----------------|----------------|
| `cache/sqlite.New[K,V]` | Config → location resolution → DSN build → `sql.Open` → pool pinning → schema DDL → sweep-on-open → wrap | `New[K comparable, V any](opts ...Option) (cache.BatchCache[K, V], error)` — mirrors `postgres.New` signature (no `ctx` param) |
| `sqliteCache[K,V]` | 7 primitive methods; TTL resolution; lazy delete on read | `struct { db *sql.DB; defaultTTL time.Duration }` — no mutex needed (`sql.DB` is concurrency-safe; all access serialized by the 1-connection pool) |
| `options.go` | `Config` + functional options | Mirror `cache/postgres/options.go` shape: `Config` struct, `Option func(*Config)`, `defaultConfig()`, `WithXxx` |
| `schema.go` | DDL constants (`CreateTableSQL`, statements) | `CREATE TABLE IF NOT EXISTS` + partial expiry index |
| `sweep.go` | One-shot expired-row DELETE on open | `DELETE ... WHERE expires_at IS NOT NULL AND expires_at <= ?`, best-effort with `slog` warn (postgres sweeper precedent) |
| `database/sql` pool | Serialize all DB access in-process; enforce one SQLite connection | `SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, `SetConnMaxLifetime(0)`, `SetConnMaxIdleTime(0)` |
| `modernc.org/sqlite` | Pure-Go SQLite driver; applies DSN pragmas per connection | blank import `_ "modernc.org/sqlite"`; driver name `"sqlite"` |
| `concreteCache` (existing) | Singleflight `GetOrSet`; `Cache`/`BatchCache` surface | untouched |
| `SingleflightGetOrSet` (existing) | Dedup concurrent misses; fast-path Get outside group | untouched; requires `GetFunc` to return an error on miss so the setter path engages |

## Recommended Project Structure

```
cache/sqlite/
├── doc.go                    # package docs + usage example (postgres doc.go parity)
├── options.go                # Config, Option, WithName/WithPath/WithMemory/WithDefaultTTL/WithBusyTimeout
├── sqlite.go                 # sqliteCache[K,V], New, DSN builder, Get/Set/Delete/MGet/MSet/MDel/Close
├── schema.go                 # CreateTableSQL, upsert/select/delete/sweep SQL constants
├── sweep.go                  # sweep-on-open (one-shot, best-effort)
├── options_internal_test.go  # defaults + option application (postgres parity)
├── sqlite_test.go            # black-box: :memory: + file modes, TTL, batch, persistence, concurrency
├── sqlite_internal_test.go   # DSN builder, resolveTTL, chunking, error paths
└── example_test.go           # runnable Example for godoc
```

### Structure Rationale

- **One provider file (`sqlite.go`), not a split like redis/valkey.** The SQLite provider is small (single `*sql.DB`, no background goroutine); postgres splits `postgres.go` + `sweeper.go` + `schema.go` — this layout is the postgres layout minus the loop.
- **`sweep.go` exists even though it is one DELETE.** It keeps the "sweep-on-open" contract explicit and testable, and leaves a natural seam if a future milestone adds an interval sweep.
- **No `config.go`/`doc.go` mixture divergence.** `mem` calls its options file `config.go`; `postgres`/`redis`/`memcache`/`valkey` call it `options.go`. New provider follows the majority + closest analog (`options.go`).
- **No background goroutine anywhere.** No `stop` channel, no `sweepLoop`. CLI processes are short-lived; per-process sweep loops would multiply across concurrent CLI invocations, and multi-process writes are already serialized by SQLite + `busy_timeout`.

## Design Decisions (SQLite-specific)

### D1 — Location resolution and `:memory:` mode

Three modes, resolved in `New` before `sql.Open`:

| Priority | Condition | Resolved database |
|----------|-----------|-------------------|
| 1 | `WithMemory()` **or** `WithPath(":memory:")` | `:memory:` (private in-memory DB, one connection) |
| 2 | `WithPath(p)`, `p != ""` | file at `p` (parent dir created with `os.MkdirAll`) |
| 3 | default (`Name`, no path) | `filepath.Join(os.UserCacheDir(), "guionardo-go", name + ".db")` |

Decisions and rationale:

- **`os.UserCacheDir()` is the root — never CWD, never a hardcoded `~`.** `go doc os.UserCacheDir` (verified locally): Unix `$XDG_CACHE_HOME` else `$HOME/.cache`; Darwin `$HOME/Library/Caches`; Windows `%LocalAppData%`; it **errors** when the location cannot be determined (e.g. `$HOME` unset or relative `XDG_CACHE_HOME`). `New` must propagate that error — never silently fall back to a different directory.
- **The stdlib doc explicitly says "Users should create their own application-specific subdirectory"** — hence the `guionardo-go` segment. Recommend `WithName(name)` defaulting to `"cache"`.
- **Reject path separators in `name`** (`filepath.Base(name) != name` → error). Cheap defense; prevents `WithName("../../x")` escaping the cache root.
- **`os.MkdirAll(filepath.Dir(path), 0o755)` before open.** The modernc driver creates the database file but *not* its parent directory; this is a standard gotcha. On Windows the mode is effectively ignored (AGENTS.md note: no permission assertions on Windows).
- **Plain `:memory:` + single-connection pool** (see D3), not `file::memory:?cache=shared`. Rationale in anti-pattern 2: each `:memory:` pool therefore owns an isolated private DB (parallel-safe tests, no cross-instance name collisions); the shared-cache URI exists only to let *multiple* connections reach one memory DB, which a 1-connection pool does not need.
- **`:memory:` is inherently per-process.** The multi-process WAL story applies only to file mode; in-memory data cannot be shared across processes at all (SQLite in-memory DBs never share across processes). Document this in `doc.go`.
- **Option API note for the roadmap:** the milestone phrase "`:memory:` when path empty" cannot be taken literally if an empty `Path` also has to mean "unset" (needed for the default UserCacheDir branch). The recommended explicit `WithMemory()` option (plus the `:memory:` sentinel through `WithPath`) resolves the ambiguity. If phase discussion insists empty ⇒ memory, then branch 3 needs a different trigger and the default-location path becomes unreachable — do not do that.

### D2 — DSN and pragmas (applied per connection, via DSN — NOT via `db.Exec`)

The canonical DSNs the provider builds:

```
file mode:
  <path>?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_txlock=immediate

:memory: mode:
  :memory:?_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_txlock=immediate
```

Decisions and rationale:

- **Pragmas go in the DSN, not `db.Exec` after opening.** modernc applies DSN driver keys on *every new connection*; `db.Exec("PRAGMA ...")` sets state on whichever pooled connection happened to serve the call, and pooled state is not reset between borrowers — the classic "PRAGMA journal_mode=WAL silently didn't stick on connection 2" failure. This is documented behavior of the driver ("For state every connection should have, use DSN parameters or a connection hook").
- **Use the `_pragma=name(value)` form** (repeatable) for all three pragmas. The driver also accepts validated shorthand keys (`_journal_mode`, `_busy_timeout`, `_synchronous` — migratable from mattn); `_pragma` is the most explicit and universally documented form. Our values are compile-time constants, so the `StrictPragmas` multi-statement concern does not apply.
- **`journal_mode=WAL` for file mode only.** WAL is a persistent property of the database file: once set, every process picks it up. For `:memory:` the journal mode cannot be changed (stays memory/off; change attempts are ignored), so the memory DSN omits it rather than shipping a misleading no-op.
- **`busy_timeout(N)` is per-connection and must be in the DSN.** modernc defaults the busy timeout to **0** (mattn defaulted to 5000 ms), so without it a concurrent writer fails instantly with `SQLITE_BUSY`. Default 5000 ms, configurable via `WithBusyTimeout`.
- **`synchronous=NORMAL`** — safe with WAL, one fewer fsync per commit; the standard pairing.
- **`_txlock=immediate`** — every `database/sql` transaction begins with `BEGIN IMMEDIATE`, acquiring the write lock at BEGIN instead of on first write. This removes the deferred-read→write upgrade race where two connections both BEGIN-then-write and one gets `SQLITE_BUSY` despite the timeout. Consequence to document: all transactions on this pool are write transactions; the provider never opens read-only transactions (single-statement reads need none), so this is safe.
- **No `foreign_keys` pragma.** The schema has no foreign keys; enabling it buys nothing (anti-feature discipline).
- **No `auto_vacuum` by default.** `auto_vacuum` must be set before the first table/write and the driver applies `_auto_vacuum` first precisely for that ordering; but FULL moves pages on every commit and INCREMENTAL only reclaims when `PRAGMA incremental_vacuum` is run. For a cache that overwrites/reuses a bounded key space, freelist reuse stabilizes the file (see D6); enabling either mode adds write amplification for no functional gain. Document growth behavior instead. If a future milestone wants it, INCREMENTAL + an explicit reclaim is the defensible option.
- **No `wal_autocheckpoint` / `journal_size_limit` tuning by default.** SQLite's defaults (autocheckpoint ~1000 pages, last-close checkpoint deletes the WAL) fit a local CLI cache. Both are documented tuning knobs if metrics later demand them (questions.md item 3).
- **Best-effort WAL verification on open:** after schema creation, `PRAGMA journal_mode` read via `QueryRow` and compared to `"wal"` in file mode; mismatch logs a warning (read-only filesystem fallback), never fails construction. `db.Exec` cannot observe the pragma's returned value; `QueryRow` can. This restates as a runtime check only — the DSN still carries the pragma for new connections.

### D3 — Connection pool: `MaxOpenConns(1)` for BOTH modes

```go
db, err := sql.Open("sqlite", dsn)   // blanket import _ "modernc.org/sqlite"
if err != nil { return nil, fmt.Errorf("cache/sqlite: %w", err) }

db.SetMaxOpenConns(1)    // one physical SQLite connection per provider
db.SetMaxIdleConns(1)    // keep THE connection alive (critical for :memory:)
db.SetConnMaxLifetime(0) // no age-based recycling
db.SetConnMaxIdleTime(0) // no idle eviction
```

Rationale (this is the single most important pool decision):

- **Why 1 for file mode:** SQLite allows one writer at a time. With a >1 pool, two goroutines can begin write transactions on two connections and hit `SQLITE_BUSY` even with `busy_timeout` (timeout bounds waiting, it does not remove contention); `MaxOpenConns(1)` converts in-process contention into goroutines waiting on the pool mutex — deterministic, zero spurious BUSY. WAL's payoff is preserved *across processes* (other processes read while this one writes), which is the milestone's stated multi-process target. modernc's own docs also recommend bounding the pool because each connection carries its own page cache and libc runtime state.
- **Why 1 for `:memory:`:** each new connection with the bare `:memory:` name gets **its own private database**. With an unpinned pool, a write on one connection and a read on another lands on different DBs — the classic "no such table" symptom documented by SQLite and reproduced across Go issue threads. `MaxOpenConns(1)` is the canonical fix.
- **Why `MaxIdleConns(1)` + zero lifetimes:** with `:memory:`, the database is deleted when the last connection closes. `SetConnMaxIdleTime`/`SetConnMaxLifetime` > 0 (or an idle pool that drops its only conn) would evict the single connection and silently destroy the cache. Keep both at 0 and force one idle connection.
- **Trade-off accepted:** in-process reads serialize behind writes. For a utility cache with short statements this is a rounding error; a reader/writer two-pool split (`mode=ro` reader pool + writer pool) is explicitly deferred — it doubles handles, needs both pools pointed at the same file with all pragmas on both, and complicates Close. Revisit only if benchmarks show read starvation.
- **No provider-level mutex needed.** `sql.DB` is concurrency-safe and the pool serializes; `sqliteCache` holds only `db` + `defaultTTL`, both immutable after `New`.

### D4 — Schema and key/value encoding

```sql
CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key  TEXT    PRIMARY KEY,   -- fmt.Sprint(key), same as every provider
    value      TEXT    NOT NULL,      -- encoding/json, same as every provider
    expires_at INTEGER                -- UnixNano; NULL = no expiry
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;     -- partial index matches the sweep predicate
```

Decisions and rationale:

- **Key/value encoding parity is a hard requirement:** keys are `fmt.Sprint(key)` bound as `string`; values are `json.Marshal` → `string` (TEXT). Verified in `postgres.go`, `redis.go`, `memcache.go`. Parity keeps `GetOrSet` and cross-provider behavior identical, and keeps DB contents debuggable with any SQLite client.
- **`expires_at INTEGER` = UnixNano, `NULL` = never.** Chosen over the postgres `TIMESTAMPTZ` analog because SQLite has no first-class timestamp; integer epoch compares correctly, indexes compactly, and avoids SQLite date-function/format parsing entirely. Sub-second TTLs stay exact. Compare against a **bound `time.Now().UnixNano()` parameter** (constant per query) rather than SQL `unixepoch()`/`strftime()` so the expiry index is a range scan and no row-local function is evaluated. (UnixNano overflows in 2262 — document in the schema comment, not a practical concern.)
- **Omit `created_at`** (postgres has it; nothing reads it). Minimal schema, less churn. Deliberate divergence, recorded here.
- **`WITHOUT ROWID`:** for a `TEXT PRIMARY KEY` key-value table this removes the redundant rowid B-tree — the table *is* the PK index (SQLite doc: recommended for tables with non-integer PKs where PK lookups dominate). Supported since SQLite 3.8.2; bundled version is 3.53.4. Small, safe win; if the planner prefers strict postgres-convention parity, a plain rowid table is the fallback and nothing else changes.
- **Partial expiry index** mirrors the postgres partial index (`WHERE expires_at IS NOT NULL`), keeps NULL-expiry entries out of the index, and matches both the sweep and the "is expired" predicate.
- **Table name is a constant** (`cache_entries`, postgres parity). No `WithTableName` in v1.8: a configurable identifier would require SQL identifier validation (postgres uses `pgx.Identifier.Sanitize()`; database/sql has no equivalent) and multiple logical caches should use multiple DB files (`WithName`) instead. Deferred anti-feature.
- **No migrations framework.** `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` idempotent DDL, exactly like postgres (`schema.go`). Executed on every `New`; new statements must be appended compatibly or handled by a future versioned migration.

### D5 — TTL semantics and lazy expiry on read

`resolveTTL` mirrors postgres exactly:

```go
func (c *sqliteCache[K, V]) resolveTTL(ttl ...time.Duration) (int64, bool) {
    if len(ttl) > 0 && ttl[0] > 0 {          // per-key TTL wins
        return time.Now().Add(ttl[0]).UnixNano(), true
    }
    if c.defaultTTL > 0 {                    // provider default second
        return time.Now().Add(c.defaultTTL).UnixNano(), true
    }
    return 0, false                          // no expiry -> NULL
}
```

- **Default TTL = 0 (no expiry).** Postgres default is also 0; `mem`'s 5-minute default is the outlier. A persistent cache that silently expires entries is surprising; callers opt in via `WithDefaultTTL` or per-`Set`/`GetOrSet` TTL.
- **Read query:**
  ```sql
  SELECT value, expires_at FROM cache_entries WHERE cache_key = ?
  ```
  - `sql.ErrNoRows` → `fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)` (exact postgres wrapping; `GetOrSet` relies on this error to trigger the setter path).
  - Row found and `expires_at` non-NULL and `<= now` → **lazy delete**: best-effort `DELETE FROM cache_entries WHERE cache_key = ? AND expires_at IS NOT NULL AND expires_at <= ?`, then return `cache.ErrMiss`. The `expires_at <= ?` predicate makes the delete safe if a concurrent Set refreshed the key between SELECT and DELETE.
  - Otherwise unmarshal JSON; unmarshal failure returns a wrapped error (postgres parity — corruption is an error, not a miss).
- **Why SELECT-then-check instead of postgres' `WHERE ... AND (expires_at IS NULL OR expires_at > NOW())`:** to satisfy the milestone's "lazy delete on read" the provider must distinguish "row absent" from "row expired" anyway; the two-step form does both in one SELECT plus one conditional DELETE.
- **`MGet` does NOT lazy-delete.** Batch read filters expired rows in SQL (`AND (expires_at IS NULL OR expires_at > ?)`) and skips them silently (D-06 best-effort parity); issuing per-key deletes inside a batch read would multiply statements. The open-time sweep and `Get` clean up. Document the asymmetry.

### D6 — Sweep-on-open (one-shot, no background goroutine)

```go
// called once from New, after schema DDL, before wrapping
func (c *sqliteCache[K, V]) sweep(ctx context.Context) {
    if _, err := c.db.ExecContext(ctx, sweepSQL, time.Now().UnixNano()); err != nil {
        logger().Warn("cache/sqlite: sweep failed", "error", err)
    }
}
```

- **One `DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at <= ?`** — uses the partial index; cheap.
- **Best-effort:** warn-and-continue (postgres sweeper logs failures, does not fail operation). A cache that fails to construct because cleanup failed would be strictly worse.
- **Why no interval/loop:** every concurrent CLI process would run its own ticker against the same file (redundant write contention); short-lived processes would never fire it usefully; and the milestone scope says lazy-delete + sweep-on-open. A future `WithSweepInterval` that starts a per-instance goroutine is a deliberate non-goal for v1.8 (anti-feature).
- **Growth semantics to document (questions.md item 3):** DELETE never shrinks the SQLite file; pages go to the freelist and are reused. A bounded working key set stabilizes near its high-water mark. In WAL mode the last connection close checkpoints and removes `-wal`; a long-lived process accumulates `-wal` until autocheckpoint (default ~1000 pages) or close. No auto_vacuum (D2). `doc.go` should say: file size is a high-water bound, not a live-row bound; DB files are safe to delete to reclaim space.
- **File-per-cache-name isolation** means operators can delete `<UserCacheDir>/guionardo-go/<name>.db*` freely (documented as "safe to delete when the process is not running").

### D7 — Batch operations and transaction strategy

**MSet — one transaction, one prepared statement, one commit** (the single most impactful SQLite write optimization; autocommit would pay a commit/fsync per key — benchmark lore: 5000 inserts 46.9 s individually vs 0.047 s in one transaction):

```go
func (c *sqliteCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    if len(items) == 0 { return nil }
    expiresAt, hasTTL := c.resolveTTL(ttl...)

    tx, err := c.db.BeginTx(ctx, nil)   // BEGIN is rewritten to BEGIN IMMEDIATE by _txlock
    if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    defer tx.Rollback()                 // no-op after Commit

    stmt, err := tx.PrepareContext(ctx, upsertSQL)
    if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    defer stmt.Close()

    for key, value := range items {
        data, err := json.Marshal(value)          // hard failure, before any write
        if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
        var exp any
        if hasTTL { exp = expiresAt }             // bind NULL when no expiry
        if _, err := stmt.ExecContext(ctx, fmt.Sprint(key), string(data), exp); err != nil {
            return fmt.Errorf("cache/sqlite: %w", err)   // fail-fast + deferred rollback
        }
    }
    if err := tx.Commit(); err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    return nil
}
```

- **Marshal before opening the transaction** where practical (postgres does this per item inside its batch; for SQLite, pre-validating all items avoids opening a tx that will be rolled back).
- **Fail-fast + rollback, not postgres' accumulate-and-continue.** A `database/sql` transaction is the atomic unit; `errors.Join` accumulation only makes sense for pgx's SendBatch rows. The batch is all-or-nothing; document it.
- **One transaction regardless of item count** — the prepared statement avoids per-item SQL parsing; no chunking needed because each execution binds a fixed 3 parameters.
- **`upsertSQL`:**
  ```sql
  INSERT INTO cache_entries (cache_key, value, expires_at) VALUES (?, ?, ?)
  ON CONFLICT (cache_key) DO UPDATE
      SET value = excluded.value, expires_at = excluded.expires_at
  ```

**MGet — chunked `IN` query, read-only, no transaction:**

```sql
SELECT cache_key, value FROM cache_entries
WHERE cache_key IN (<n placeholders>) AND (expires_at IS NULL OR expires_at > ?)
```

- One query beats N point lookups (postgres uses `SendBatch` because of network RTT; SQLite is in-process, so a single set-oriented statement is optimal).
- **Chunk at 500 keys per statement.** SQLite's `SQLITE_MAX_VARIABLE_NUMBER` is 32766 in modern builds but was 999 before 3.32; 500 stays safely under both and bounds statement text. n+1 bound values per chunk (keys + now).
- Scan rows into `map[K]V` keyed by the original `K`; skip rows that fail to scan/unmarshal (D-06 best-effort, postgres parity). Duplicate keys in the variadic input collapse naturally in the map.
- `len(keys) == 0` → return empty map.

**MDel — chunked `DELETE ... IN (...)`, all chunks in one transaction:**

```sql
DELETE FROM cache_entries WHERE cache_key IN (<n placeholders>)
```

- Idempotent by construction (D-05: deleting missing keys is not an error).
- Wrap the chunk loop in one `BeginTx` so the whole batch is atomic (same `_txlock=immediate` semantics). Fail-fast + rollback.

### D8 — Open/close lifecycle and error contract

Open sequence in `New` (all with `context.Background()` — postgres precedent; `New` takes no `ctx`):

1. Apply options over `defaultConfig()`.
2. Resolve location (D1) — may return `os.UserCacheDir` error.
3. `os.MkdirAll` for file mode.
4. Build DSN (D2).
5. `sql.Open("sqlite", dsn)` (lazy; no connection yet).
6. Pin pool (D3).
7. Exec schema DDL — this creates the first connection, applies DSN pragmas, and creates the file.
8. File mode: read `PRAGMA journal_mode`, warn if not `"wal"`.
9. `sweep(context.Background())` best-effort.
10. `return cache.NewConcreteCache(c), nil`.

Close: `return c.db.Close()` — idempotent enough for the interface; `database/sql.Close` waits for in-flight queries, prevents new ones, and the last SQLite connection close checkpoints/deletes the `-wal` sidecar. No goroutine to stop, no stop channel (postgres `CloseFunc` has one only because of its sweeper).

Error contract:

| Condition | Behavior |
|-----------|----------|
| Key missing or expired | `fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)` — required for `GetOrSet` |
| Invalid/corrupt JSON value on `Get` | wrapped error (not a miss) |
| JSON value in `MGet` | skipped (best-effort, D-06) |
| `MSet` marshal error | hard failure, no writes |
| `SQLITE_BUSY` after `busy_timeout` | wrapped error surfaced to caller (spike item: verify observable behavior) |
| Sweep failure | `slog.Warn`, construction succeeds |
| `Close` called twice | no error |
| Ops after `Close` | `database/sql` error ("sql: database is closed"), wrapped |

All provider errors prefixed `"cache/sqlite: "` (postgres convention). Logger via `var logger = sync.OnceValue[*slog.Logger]` with `slog.String("module", "cache/sqlite")` (postgres pattern).

## Architectural Patterns

### Pattern 1: Per-connection DSN pragmas

**What:** Every pragma needed for correct multi-connection behavior is a DSN query parameter (`_pragma=...`, `_txlock=...`), never a post-open `db.Exec`.
**When to use:** Always, for SQLite providers with connection pools.
**Trade-offs:** DSN string construction is slightly more code; gains correctness across every pooled connection and across reconnections.

### Pattern 2: Single-connection pool as the concurrency model

**What:** `SetMaxOpenConns(1)` (plus `MaxIdleConns(1)`, zero conn lifetimes) is the provider's entire in-process locking strategy.
**When to use:** Embedded caches with short statements; mandatory for `:memory:`.
**Trade-offs:** In-process reads serialize. Accepted; WAL still provides cross-process read/write concurrency.

### Pattern 3: One transaction + one prepared statement per batch write

**What:** MSet/MDel open a single transaction, prepare once, loop, commit once.
**When to use:** Any multi-row write.
**Trade-offs:** A mid-batch failure rolls the whole batch back (documented atomicity); negligible downside for cache semantics.

### Pattern 4: Lazy expiry on read + one-shot sweep on open

**What:** Get deletes the expired row it encounters; New deletes all expired rows once.
**When to use:** TTL caches with many short-lived processes.
**Trade-offs:** Expired rows can linger until the next `New`/`Get`; bounded by process start frequency and read traffic. No goroutine lifecycle, no duplicate sweepers in multi-process use.

### Pattern 5: Structural satisfaction of the frozen `cacher` interface

**What:** `sqliteCache[K,V]` implements the seven methods; `New` passes it to `cache.NewConcreteCache` — Go implicit interfaces make the unexported interface satisfiable from another package exactly as `postgres` does.
**When to use:** Every new provider; this is the integration contract.
**Trade-offs:** The compiler only verifies the method set at the `NewConcreteCache` call site — keep a compile-time assertion test or rely on `New`'s return.

## Data Flow

### Request Flow (Get / GetOrSet)

```
Get(ctx, key) ──► concreteCache.Get ──► GetFunc
                                        │
                    SELECT value, expires_at WHERE cache_key = ?
                                        │
             ┌──────────────────────────┼───────────────────────────┐
             ▼                          ▼                           ▼
        ErrNoRows               expired (<= now)               valid row
             │                          │                           │
      wrap ErrMiss        best-effort DELETE, wrap ErrMiss     json.Unmarshal → value
```

```
GetOrSet(ctx, key, setter, ttl)
   └─ fast-path Get OUTSIDE singleflight (misses return ErrMiss)
        └─ singleflight group Do(key)
             ├─ double-check Get
             ├─ setter(ctx)            ← runs once for N concurrent misses
             └─ SetFunc → upsert (json.Marshal, resolveTTL)
```

### Batch Flow (MSet)

```
MSet(ctx, items, ttl) ──► BeginTx (BEGIN IMMEDIATE via _txlock)
                            └─ Prepare upsert ON CONFLICT DO UPDATE
                                 └─ for each item: marshal → Exec(bound params)
                                      └─ Commit (single fsync)
   any error ──► deferred Rollback ──► wrapped error
```

### Key Data Flows

1. **Value encoding:** `V → json.Marshal → string → TEXT column`; reverse on read. Identical to all other providers.
2. **Key encoding:** `K → fmt.Sprint → string → TEXT PRIMARY KEY`; also used by singleflight's internal group key.
3. **Expiry:** `ttl/defaultTTL → time.Now().Add(...).UnixNano() → INTEGER`; `NULL` = no expiry; all comparisons against a bound `now` parameter.
4. **Multi-process:** process A write commits to `-wal`; process B reads the latest committed snapshot via WAL; `busy_timeout` queues competing writers; last close checkpoints.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single process, short-lived CLI | Default config; `MaxOpenConns(1)`; sweep-on-open prunes; nothing else needed |
| Many sequential CLI invocations (multi-process) | WAL + `busy_timeout(5000)` + `_txlock=immediate`; writes queue instead of failing; keep transactions short |
| Concurrent long-running processes | Same; consider raising `busy_timeout` (e.g. 10–30 s) if write bursts are long; batch MSet to one commit |
| High in-process read concurrency | Split reader pool (`mode=ro`, N conns) + writer pool — deferred; benchmark first |
| Large key counts / churn | Partial expiry index + chunked MGet; file grows to a high-water mark and stabilizes; delete the DB file to reclaim fully |

### Scaling Priorities

1. **First bottleneck (in-process):** serialization on the single connection for read-heavy workloads. Fix: reader-pool split (deferred) or accept it — statements are sub-millisecond locally.
2. **Second bottleneck (multi-process):** writer contention under `busy_timeout`; fix by shortening write transactions (already one tx per batch) and increasing `busy_timeout`.
3. **Third (disk):** file/WAL growth; fix by bounded key spaces, `journal_size_limit`, or manual file deletion — not auto_vacuum (D2).

## Anti-Patterns

### Anti-Pattern 1: Setting pragmas once with `db.Exec` after `sql.Open`

**What people do:** `db, _ := sql.Open(...); db.Exec("PRAGMA journal_mode=WAL")`.
**Why it's wrong:** PRAGMA state is per-connection; the pool may open more connections later (and recreates connections after failures) that never receive it. WAL still persists on the file once set, but `busy_timeout` and other per-connection pragmas silently don't apply.
**Do this instead:** put every pragma in the DSN (D2).

### Anti-Pattern 2: Bare `:memory:` with the default pool

**What people do:** `sql.Open("sqlite", ":memory:")` and expect pooled connections to share data.
**Why it's wrong:** each pooled connection gets its own private database; writes "vanish" (symptom: `no such table`).
**Do this instead:** `SetMaxOpenConns(1)` + `SetMaxIdleConns(1)` + zero conn lifetimes. Only reach for `file::memory:?cache=shared` when multiple connections genuinely must share one memory DB (and then use a unique name per instance).

### Anti-Pattern 3: Autocommit per key in MSet/MDel

**What people do:** loop `db.Exec(upsert, ...)` for each item.
**Why it's wrong:** one implicit transaction + commit/fsync per key; SQLite write throughput collapses by orders of magnitude.
**Do this instead:** one transaction, one prepared statement, one commit (D7).

### Anti-Pattern 4: Default deferred transactions for batch writes

**What people do:** `BeginTx` with driver-default `BEGIN DEFERRED` and write inside.
**Why it's wrong:** the write lock is only taken at the first write statement; concurrent writers race on upgrade and can fail with `SQLITE_BUSY` despite `busy_timeout`.
**Do this instead:** `_txlock=immediate` in the DSN (D2).

### Anti-Pattern 5: WAL on a network filesystem / shared folder

**What people do:** point the cache at NFS/SMB/a synced folder.
**Why it's wrong:** WAL requires shared memory and locking semantics that network filesystems do not reliably provide; corruption risk.
**Do this instead:** local disk only; document that `os.UserCacheDir` is local by definition. `doc.go` should carry this warning.

### Anti-Pattern 6: Treating DELETEs as disk reclamation

**What people do:** expect the file to shrink after TTL expiry sweeps.
**Why it's wrong:** SQLite moves freed pages to the freelist; the file stays at its high-water mark.
**Do this instead:** document growth behavior; optionally provide "delete the DB file to reclaim" guidance (D6).

### Anti-Pattern 7: Background sweep goroutines per process

**What people do:** copy the postgres `sweepLoop` pattern into the CLI-oriented provider.
**Why it's wrong:** every concurrent process sweeps the same file — duplicated write contention for no benefit in short-lived processes; adds lifecycle complexity (stop channel) that the interface does not need.
**Do this instead:** lazy delete on read + one-shot sweep on open (D6); revisit only with benchmark/UAT evidence.

### Anti-Pattern 8: `auto_vacuum=FULL`

**What people do:** enable full auto-vacuum to keep the cache file small.
**Why it's wrong:** page movement on every commit — write amplification for a cache that already reuses pages.
**Do this instead:** leave NONE; document; consider INCREMENTAL + explicit reclaim only if a real metric demands it.

### Anti-Pattern 9: Unbounded SQLite pool

**What people do:** rely on `database/sql` defaults (unlimited connections).
**Why it's wrong:** each connection carries its own page cache and modernc libc state; writers still serialize at the database level and now fight through SQLITE_BUSY.
**Do this instead:** `SetMaxOpenConns(1)` (D3); modernc's docs say the same.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| SQLite via `modernc.org/sqlite` | blank import; `sql.Open("sqlite", dsn)` | pure-Go; CGO_ENABLED=0; darwin/linux/windows amd64+arm64; v1.60.1 bundles SQLite 3.53.4 |
| `modernc.org/libc` (transitive) | direct version pin | driver docs: import the exact libc version pinned by the driver's go.mod (v1.77.1 for v1.60.1) — "fragile dependency" warning; `go.mod` should carry it explicitly |
| Go module proxy / CI | `go get modernc.org/sqlite@v1.60.1`; CI matrix unchanged | no Docker needed for this provider's tests (unlike redis/valkey/postgres/memcache E2E) |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cache/sqlite` ↔ `cache` (root) | `cache.NewConcreteCache[K,V](c)` returns `cache.BatchCache[K,V]` | Structural satisfaction of unexported `cacher`; no root changes allowed |
| `cache/sqlite` ↔ `cache/sqlite` (methods) | `context.Context` on every op; `ErrMiss` wrapping on misses | `GetOrSet`'s singleflight depends on the miss error |
| `New` ↔ `database/sql` | DSN + pool pinning + DDL Exec | first Exec materializes the connection and applies DSN pragmas |
| `CloseFunc` ↔ file | `db.Close()` | last connection checkpoints/deletes `-wal`; no goroutine teardown |

## New vs Modified Components (explicit)

| Artifact | New / Modified | Detail |
|----------|----------------|--------|
| `cache/sqlite/doc.go` | **NEW** | package docs, usage example, WAL/`:memory:`/network-FS caveats |
| `cache/sqlite/options.go` | **NEW** | `Config{Name, Path, Memory, DefaultTTL, BusyTimeout}`, `Option`, `WithName`, `WithPath`, `WithMemory`, `WithDefaultTTL`, `WithBusyTimeout`, `defaultConfig` |
| `cache/sqlite/sqlite.go` | **NEW** | `sqliteCache[K,V]`, `New`, DSN builder, 7 primitive methods, `resolveTTL`, logger |
| `cache/sqlite/schema.go` | **NEW** | DDL + statement constants |
| `cache/sqlite/sweep.go` | **NEW** | `sweep(ctx)` |
| `cache/sqlite/*_test.go`, `example_test.go` | **NEW** | unit + integration + example tests, incl. helper-process multi-process test |
| `go.mod`, `go.sum` | **MODIFIED** | add `modernc.org/sqlite` (+ explicit `modernc.org/libc` pin per driver docs) |
| `README.md` | **MODIFIED** | package index row for `cache/sqlite` |
| `AGENTS.md` | **MODIFIED** | packages table row (`cache/sqlite`) |
| `quality-report.md` | **MODIFIED** | regenerated before tag (`make quality-report`) |
| `cache/cache.go`, `concrete_cache.go`, `singleflight.go`, `errors.go` | **UNTOUCHED** | frozen interfaces (v1.6 D-02) |
| `cache/{mem,redis,valkey,memcache,postgres}/**` | **UNTOUCHED** | no cross-provider changes |
| `Makefile`, `.testcoverage-quick.yml` | **UNTOUCHED** | SQLite tests run in the normal unit lane; no threshold overrides needed |

## Suggested Build Order

Dependency-ordered steps (each step compiles and is testable):

1. **Foundation — options, location, DSN, open/close, schema** (`options.go`, `schema.go`, `sqlite.go` skeleton)
   - `Config` + options + `defaultConfig`; location resolution (D1) incl. `:memory:`; DSN builder (D2); `sql.Open` + pool pinning (D3); DDL exec; `CloseFunc`; `doc.go`.
   - Unblocks everything; smoke test: `New` in `:memory:` mode, then `Close`.
2. **Primitives + TTL** (`sqlite.go`)
   - `SetFunc` (upsert + `resolveTTL`), `GetFunc` (SELECT, miss wrapping, lazy delete), `DeleteFunc` (idempotent DELETE).
   - This alone gives a working `cache.Cache[K,V]` (GetOrSet comes free through `NewConcreteCache`).
3. **Sweep-on-open** (`sweep.go` + call site in `New`)
   - One-shot best-effort DELETE; test with pre-seeded expired rows.
4. **Batch surface** (`sqlite.go`)
   - `MGetFunc` (chunked `IN`), `MSetFunc` (single tx + prepared upsert), `MDelFunc` (tx + chunked `IN`).
   - Depends on step 2's SQL constants and TTL handling.
5. **Hardening, tests, docs, gates**
   - `:memory:` + temp-file tests; TTL/expiry/sweep; reopen-persistence; concurrency (`-race`); helper-process multi-process busytimeout test (spike item 4); `example_test.go`; `doc.go`; coverage thresholds; README/AGENTS rows.

Natural phase grouping if the roadmap wants two phases: (1–3) "core provider (primitives, TTL, sweep)" and (4–5) "batch + multi-process hardening". Step 1 is a hard prerequisite for all others; steps 2 and 3 are independent; step 4 depends on 2; step 5 depends on everything.

## Open Questions / Spikes

Per `.planning/research/questions.md` (tier-floor; unresolved):

1. **Two-process `SQLITE_BUSY` behavior (questions.md #4).** Required spike: run two OS processes writing through modernc to one file with WAL + `busy_timeout`, observe whether/when errors surface and whether retries are needed beyond `busy_timeout`. Note for the spike design: POSIX advisory locks are per-process, so two `sql.DB` handles in one test process do **not** reproduce cross-process lock behavior — use the helper-process pattern (`os/exec` re-running the test binary with an env flag) or a small two-binary harness. This is the only LOW-confidence area of this document.
2. **Exact driver version + dependency-tree acceptance (questions.md #1).** Latest at research time: `modernc.org/sqlite v1.60.1` (Go module proxy, 2026-09-29). Transitive deps include `modernc.org/libc`, `memory`, `mathutil`, `fileutil`, `golang.org/x/sys`, `google/uuid`, `mattn/go-isatty`, `ncruces/go-strftime`, `dustin/go-humanize`, `remyoudompheng/bigfft`. STACK research owns the final pin; the architecture requires the explicit libc pin.
3. **Growth guidance (questions.md #3).** Design answer: freelist reuse + high-water behavior, no auto_vacuum, documented file-deletion escape hatch. Validate empirically in UAT (write/expire cycles → observe file stability).
4. **`WithPath("")` resolution ambiguity (D1).** Confirm the recommended `WithMemory()` shape with the user in phase discussion before locking the option API.
5. **Chunk constant (D7).** 500 is a conservative default; if benchmarking shows MGet chunking dominates, raise toward the modern 32766 variable limit — but do not rely on it without checking the bundled build's `SQLITE_MAX_VARIABLE_NUMBER`.

## Sources

| Source | Used for | Confidence |
|--------|----------|------------|
| `pkg.go.dev/modernc.org/sqlite` (Driver.Open, README, platform table, connecting docs) — fetched directly | DSN keys/order, `_txlock`, `:memory:`/URI behavior, platform support, FULLMUTEX, pool warning, libc pin, performance notes | MEDIUM (official package docs; cross-checked with independent implementations) |
| `sqlite.org/pragma.html` | WAL persistence, in-memory journal restriction, busy_timeout semantics, auto_vacuum | MEDIUM |
| `sqlite.org/wal.html` + SQLite forum threads | multi-process WAL semantics, checkpoint/last-close, network FS caveat, BEGIN IMMEDIATE/upgrade races | MEDIUM |
| `sqlite.org/inmemorydb.html` + mattn/go-sqlite3 issues #204/#274 | `:memory:` per-connection isolation, shared-cache URI, `SetMaxOpenConns(1)` | MEDIUM |
| `go.dev/doc/database/manage-connections` | pool semantics (`SetMaxOpenConns` lock analogy) | MEDIUM |
| `pkg.go.dev/github.com/hollis-labs/go-sqlite` + tessl sqlite-go-best-practices + theitsolutions.io | single-writer pool + `_txlock=immediate` convergence, DSN-per-connection rationale | MEDIUM (independent implementations converging) |
| Single-thread/system.data.sqlite benchmarks | autocommit vs single-transaction write cost | LOW-MEDIUM (classic benchmark lore, directionally reliable) |
| `proxy.golang.org` @latest + `gitlab.com/cznic/sqlite` go.mod | v1.60.1, libc v1.77.1 requirement | MEDIUM |
| Local code inspection: `cache/concrete_cache.go`, `cache/cache.go`, `cache/singleflight.go`, `cache/errors.go`, `cache/postgres/*`, `cache/mem/*`; `go doc os.UserCacheDir` | integration contract, parity conventions, UserCacheDir semantics | HIGH |

## Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Integration points / wiring (cacher, NewConcreteCache, ErrMiss, singleflight) | HIGH | Directly inspected code; no ambiguity |
| Schema, encoding, TTL, sweep design | HIGH for design parity; MEDIUM for SQLite mechanics | Parity verified against postgres/mem/redis/memcache code; mechanics corroborated by official docs |
| Pragmas, pool sizing, `:memory:` handling, batch transactions | MEDIUM | Cross-checked official docs + multiple independent Go implementations; consistent convergence |
| Multi-process `SQLITE_BUSY` behavior under contention | LOW | Tier-floor in questions.md; requires the spike (helper-process pattern) before implementation relies on it |
| Driver version / dependency tree | MEDIUM | Primary sources (proxy, go.mod) but version drift is expected; STACK owns the final pin |

---

*Architecture research for: `cache/sqlite` embedded provider — v1.8 SQLite Cache Backend*
*Researched: 2026-10-08*

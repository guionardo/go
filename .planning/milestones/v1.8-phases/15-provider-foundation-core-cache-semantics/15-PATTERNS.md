# Phase 15: Provider Foundation + Core Cache Semantics - Pattern Map

**Mapped:** 2026-10-08
**Files analyzed:** 14 (12 new in `cache/sqlite/`, 2 modified at repo root)
**Analogs found:** 12 / 14 (2 have no close analog — use RESEARCH patterns)

> **Tracked-source gate:** every analog path below was verified with `git ls-files` on 2026-10-08 (all tracked). `cache/sqlite/` does not exist in the repo — every path under it is new.
>
> **Frozen-root rule (CONTEXT):** `cache/`, `cache/doc.go`, and the five existing providers must NOT be modified. The only non-`cache/sqlite` edits are `go.mod`/`go.sum` and the README package index.
>
> **Critical compile seams the planner must encode (RESEARCH Findings 1–5):**
>
> | Finding | File it forces |
> |---------|----------------|
> | 1. `cacher[K,V]` requires 7 methods → batch placeholders mandatory | `sqlite.go` (`MGetFunc`/`MSetFunc`/`MDelFunc` loop stubs + Phase 16 TODO) |
> | 2. `NewConcreteCache` does not forward `Optimizable` → adapter required | `optimize.go` (`batchCache` wrapper `New` returns) |
> | 3. Reads never delete rows (filter-only SELECT) | `schema.go` (`SelectSQL`), `sqlite.go` (`GetFunc`) |
> | 4. No `user_version`; DDL-only bootstrap under `BEGIN IMMEDIATE` | `schema.go`, `sqlite.go` (`bootstrap`) |
> | 5. `New` never fails; deferred `initErr` + `ErrClosed`/`ErrInvalidPath` | `sqlite.go`, `dsn.go`, `options.go` |

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cache/sqlite/doc.go` | docs | n/a | `cache/postgres/doc.go` | exact |
| `cache/sqlite/options.go` | config | n/a | `cache/postgres/options.go` | exact |
| `cache/sqlite/dsn.go` | utility | transform | `cache/postgres/options.go` (config shape) + `cache/valkey/valkey.go` (deferred validation err) | partial |
| `cache/sqlite/schema.go` | model (SQL constants) | n/a | `cache/postgres/schema.go` | role-match |
| `cache/sqlite/sqlite.go` | provider (service) | CRUD + request-response | `cache/postgres/postgres.go` + `cache/valkey/valkey.go` | role-match |
| `cache/sqlite/sweep.go` | worker/service | batch (scheduled) | `cache/postgres/sweeper.go` + `cache/mem/sweeper.go` | role-match |
| `cache/sqlite/optimize.go` | provider service (adapter) | request-response | **none** — RESEARCH §5 / Finding 2 | none |
| `cache/sqlite/dsn_internal_test.go` | test | n/a | `cache/postgres/options_internal_test.go` + `cache/postgres/postgres_internal_test.go` | role-match |
| `cache/sqlite/options_internal_test.go` | test | n/a | `cache/postgres/options_internal_test.go` | exact |
| `cache/sqlite/sqlite_internal_test.go` | test | n/a | `cache/postgres/postgres_internal_test.go` + `cache/memcache/memcache_internal_test.go` | role-match |
| `cache/sqlite/sqlite_test.go` | test | n/a | `cache/postgres/postgres_test.go` + `cache/mem/sweeper_test.go` | role-match |
| `cache/sqlite/example_test.go` | test (example) | n/a | `cache/postgres/example_test.go` | exact (drop the skip helper) |
| `go.mod` / `go.sum` | config (deps) | n/a | existing `go.mod` require blocks | exact |
| `README.md` | docs | n/a | README package index row + provider table | exact (edit existing) |

---

## Pattern Assignments

### `cache/sqlite/doc.go` (docs)

**Analog:** `cache/postgres/doc.go`

**Package doc pattern** (`cache/postgres/doc.go:1-16`):
```go
// Package postgres provides a PostgreSQL backend for cache.Cache.
//
// Uses pgx/v5 with pgxpool for connection pooling. Creates an UNLOGGED
// table (blazing-fast writes, no WAL) and optionally calls pg_prewarm
// on startup. A background goroutine sweeps expired entries.
//
// Usage:
//
//	c, err := postgres.New[string, string](
//	    postgres.WithConnString("postgres://user:pass@localhost/cache"),
//	    ...
//	)
package postgres
```

Also `cache/redis/doc.go:1-14` (shorter provider doc; "Connection is lazy — dials on the first query" style).

**Phase-15 content:** package comment + basic usage. Wording from CONTEXT specifics: "mem but survives restarts; redis/postgres without the infra". Document location precedence (memory > path > name > zero = memory) per D-02, `os.UserCacheDir` default per D-03, and `?`/`#` rejection per STOR-06. Full constraint prose (network FS unsupported, sidecars, growth) is Phase 17 (QUAL-03) — minimal now.

**Do NOT** add sqlite to the provider list in `cache/doc.go:18-24` — root is frozen; README carries discoverability.

---

### `cache/sqlite/options.go` (config)

**Analog:** `cache/postgres/options.go`

**Config + Option + defaults** (`cache/postgres/options.go:6-23`):
```go
// Config holds configuration for the Postgres cache provider.
type Config struct {
	ConnString    string
	TableName     string
	PoolSize      int
	SweepInterval time.Duration
	DefaultTTL    time.Duration
}

// Option is a functional option for configuring the Postgres cache provider.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		TableName:     "cache_entries",
		PoolSize:      5,
		SweepInterval: 1 * time.Minute,
	}
}
```

**With-function shape** (`cache/postgres/options.go:48-53`):
```go
// WithSweepInterval sets the interval for the background sweep goroutine.
func WithSweepInterval(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.SweepInterval = d
	}
}
```

**sqlite Config fields (from RESEARCH structure + D-04):** `Name string`, `Path string`, `Memory bool`, `DefaultTTL time.Duration`, `SweepInterval time.Duration`, `AutoCheckpoint int`. Exported options: `WithName`, `WithPath`, `WithMemory`, `WithDefaultTTL`, `WithSweepInterval`, `WithAutoCheckpoint`.

**Divergences (locked decisions):**
- Default `SweepInterval` is **0 / no sweeper** (D-08) — unlike postgres's `1 * time.Minute` and mem's default. Do not copy a nonzero default.
- Do **not** use the root `cache.Option` interface (`cache/options.go:13-16`) — no provider consumes it (RESEARCH seam table); use provider-local `type Option func(*Config)` like postgres/redis/valkey.

---

### `cache/sqlite/dsn.go` (utility, transform) — PARTIAL ANALOG

**No direct analog exists** (`os.UserCacheDir` and DSN building appear nowhere in the repo). Use RESEARCH Code Examples §1 and §2 (Pattern 3) as the pattern source; borrow structure/error style from these files:

**Config consumption shape** (`cache/postgres/postgres.go:38-41`):
```go
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
```

**Deferred validation-error style** (`cache/valkey/valkey.go:19-23` stores `initErr`; `:51-53` wraps it) — see Shared Patterns §3.

**Sentinel declaration style** (`cache/errors.go:38-44`):
```go
var (
	// ErrMiss is returned by Get when the key is not in the cache.
	ErrMiss = errors.New("cache: key not found")

	// ErrClosed is returned when operations are attempted on a closed cache.
	ErrClosed = errors.New("cache: cache is closed")
)
```
sqlite declares `ErrClosed = errors.New("cache/sqlite: cache is closed")` and `ErrInvalidPath = errors.New("cache/sqlite: invalid path")` (D-12, provider-local; do not alias root).

**Must-build (RESEARCH §1):** `resolveLocation(cfg *Config, userCacheDir func() (string, error)) (path string, memory bool, err error)` with precedence memory > path > name > zero (D-01/D-02), `?`/`#` rejection → `ErrInvalidPath`, name allow-list `[A-Za-z0-9._-]+` rejecting `.`/`..`/empty/separators, and `buildDSN(path, memory, autoCheckpointPages)` with compile-time-constant pragma strings only. Injectable `userCacheDir` is the test seam (Pitfall 10).

**Validation pattern:** no interpolation of caller input into the DSN; the only variable value is `strconv.Itoa(int)` for `_pragma=wal_autocheckpoint(N)` (A6). `MkdirAll(filepath.Dir(path), 0o700)` for file mode (`os.MkdirAll` precedent: `release/update.go:129`, `path_tools/path_tool_linux.go:10`).

---

### `cache/sqlite/schema.go` (model, SQL constants)

**Analog:** `cache/postgres/schema.go`

**Exact pattern** (`cache/postgres/schema.go:3-21`):
```go
// CreateTableSQL is the SQL statement for creating the cache entries table.
// The table stores JSON-serialized values with optional TTL expiration.
const CreateTableSQL = `
CREATE UNLOGGED TABLE IF NOT EXISTS cache_entries (
    cache_key   TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;
`

// PrewarmSQL preloads the cache table into PostgreSQL shared buffers.
const PrewarmSQL = `SELECT pg_prewarm('cache_entries')`
```

**sqlite version (locked schema, CONTEXT/RESEARCH):** exported consts with doc comments —
- `CreateTableSQL`: `cache_key TEXT PRIMARY KEY, value TEXT NOT NULL, expires_at INTEGER` (UnixNano, NULL = never). Split `CREATE TABLE` and `CREATE INDEX` into separate consts because the bootstrap Execs them individually (`tx.ExecContext` per statement). Partial index: `ON cache_entries (expires_at) WHERE expires_at IS NOT NULL` (same shape as postgres).
- `SelectSQL`, `UpsertSQL`, `DeleteSQL`, `SweepSQL` per RESEARCH §3/Pattern 4:
```go
const SelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`
const UpsertSQL = `INSERT INTO cache_entries (cache_key, value, expires_at) VALUES (?, ?, ?)
ON CONFLICT(cache_key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`
const DeleteSQL = `DELETE FROM cache_entries WHERE cache_key = ?`
const SweepSQL = `DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at <= ?`
```
- No `PrewarmSQL`; no `user_version` (Finding 4); table name is fixed (`cache_entries`) — no `WithTableName` equivalent in D-04.
- `WITHOUT ROWID` is an open question (RESEARCH Open Q1, recommendation: yes) — planner decides; either way the three columns and partial index stand.

---

### `cache/sqlite/sqlite.go` (provider, CRUD + request-response) — composite analog

**Analogs:** `cache/postgres/postgres.go` (core CRUD, `resolveTTL`, logger, constructor wiring) + `cache/valkey/valkey.go` (deferred `initErr`, `New` without error) + `cache/redis/redis.go` (`New` signature parity).

**Imports pattern** (`cache/postgres/postgres.go:3-16`):
```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guionardo/go/cache"
)
```
sqlite swaps the pgx imports for `database/sql`, `database/sql/driver` (no — just `database/sql`), and the blank driver import `_ "modernc.org/sqlite"`; keep `sync/atomic` for the closed flag.

**Logger pattern** (`cache/postgres/postgres.go:30-32`):
```go
var logger = sync.OnceValue[*slog.Logger](func() *slog.Logger {
	return slog.With(slog.String("module", "cache/postgres"))
})
```

**Constructor parity — `New` returns `cache.BatchCache`, no error** (`cache/redis/redis.go:23-42`):
```go
// New creates a new Redis cache provider with optional functional options.
// Returns a cache.BatchCache sharing the in-memory singleflight GetOrSet.
func New[K comparable, V any](opts ...Option) cache.BatchCache[K, V] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	...
	return cache.NewConcreteCache(&redisCache[K, V]{
		client:     client,
		defaultTTL: cfg.DefaultTTL,
	})
}
```

**Deferred open error — `initErr`** (`cache/valkey/valkey.go:19-23` and `:51-53`):
```go
type valkeyCache[K comparable, V any] struct {
	client     valkey.Client
	initErr    error
	defaultTTL time.Duration
}
...
	if c.initErr != nil {
		return zero, fmt.Errorf("cache/valkey: %w", c.initErr)
	}
```

**CRUD + error wrapping** (`cache/postgres/postgres.go:69-129`) — copy the shape, translate SQL:
```go
func (c *postgresCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
	query := fmt.Sprintf(...)
	var valueJSON string
	err := c.pool.QueryRow(ctx, query, fmt.Sprint(key)).Scan(&valueJSON)
	if err == pgx.ErrNoRows {
		var zero V
		return zero, fmt.Errorf("cache/postgres: %w", cache.ErrMiss)
	}
	if err != nil {
		var zero V
		return zero, fmt.Errorf("cache/postgres: %w", err)
	}
	var value V
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		var zero V
		return zero, fmt.Errorf("cache/postgres: %w", err)
	}
	return value, nil
}
```
sqlite differences (RESEARCH §3, Findings 3/5/8):
- `c.check()` first (`initErr` then `closed`) in every primitive method.
- `errors.Is(err, sql.ErrNoRows)` → wrapped `cache.ErrMiss`; **never** map other errors to `ErrMiss`.
- `SetFunc` binds `nil` for SQL NULL when there is no TTL; `DeleteFunc` is a plain Exec, idempotent.
- `db.QueryRowContext(...).Scan` for singleton SELECTs, `ExecContext` for writes (Pitfall 11 — unconsumed `Row`s hold the single connection).

**TTL parity — `resolveTTL`** (`cache/postgres/postgres.go:244-256`):
```go
// resolveTTL converts the optional TTL to an expiration timestamp.
// Returns nil for no expiry.
func (c *postgresCache[K, V]) resolveTTL(ttl ...time.Duration) *time.Time {
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().Add(ttl[0])
		return &t
	}
	if c.defaultTTL > 0 {
		t := time.Now().Add(c.defaultTTL)
		return &t
	}
	return nil
}
```
sqlite returns `(int64, bool)` — `time.Now().Add(...).UnixNano(), true`, else `0, false` (RESEARCH §3) — semantics identical: per-call `>0` wins, else default `>0`, else no expiry.

**Open sequence + pool pinning + bootstrap** (RESEARCH §2 / Patterns 1, 3, 5; locked at milestone level): `sql.Open("sqlite", dsn)` → `SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, zero lifetimes → `BeginTx` (`BEGIN IMMEDIATE` via `_txlock`) wrapping both DDL Execs → `Commit` → file-mode `PRAGMA journal_mode` read-back (warn only) → synchronous best-effort `sweep(ctx)` → optional `sweepLoop` goroutine.

**Close** (RESEARCH §4 + `cache/mem/mem.go:71-83` idempotency precedent):
```go
func (c *memoryCache[K, V]) CloseFunc() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = newMemStore[K, V](0)
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}
	return nil
}
```
sqlite: `closed.CompareAndSwap(false, true)` guard; if a sweeper exists, `close(c.stop); <-c.done`; then `c.db.Close()` (last close checkpoints + removes `-wal`/`-shm`). Return `nil` on repeat calls; `Close` returns `nil` even when `initErr` is set (A4).

**Batch placeholders (Critical — Finding 1, A2):** `MGetFunc` loops `GetFunc` skipping misses; `MSetFunc`/`MDelFunc` loop the point ops — each with `// Phase 16: replace with chunked IN / single-transaction implementations`. Full BATCH-01..03 semantics stay in Phase 16.

**Structure** (`cache/postgres/postgres.go:18-28` shows the struct + doc-comment style):
```go
// postgresCache is the PostgreSQL-backed provider. It implements the
// cache.cacher primitive interface (GetFunc/SetFunc/DeleteFunc/CloseFunc);
// New wraps it in a cache.NewConcreteCache, which supplies the shared Cache
// surface (singleflight GetOrSet dedup, Cache interface).
type postgresCache[K comparable, V any] struct { ... }
```

---

### `cache/sqlite/sweep.go` (worker, scheduled batch)

**Analogs:** `cache/postgres/sweeper.go` + `cache/mem/sweeper.go`

**Loop + best-effort sweep** (`cache/postgres/sweeper.go:12-37`):
```go
// sweepLoop runs periodically to delete expired cache entries.
func (c *postgresCache[K, V]) sweepLoop() {
	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.sweep()
		case <-c.stop:
			return
		}
	}
}

// sweep deletes all expired entries from the cache table.
// Sweep is best-effort maintenance — errors are logged but not returned.
func (c *postgresCache[K, V]) sweep() {
	query := fmt.Sprintf(...)
	if _, err := c.pool.Exec(context.Background(), query); err != nil {
		slog.Warn("cache/postgres: sweep failed", "error", err)
	}
}
```

**Stop-channel shape with ctx** (`cache/mem/sweeper.go:8-22`) shows the ticker + `select` idiom; sqlite adds a `done` channel so `Close` can wait (D-08, RESEARCH §4):
```go
func (c *sqliteCache[K, V]) sweepLoop() {
	defer close(c.done)
	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.sweep(context.Background())
		case <-c.stop:
			return
		}
	}
}
```

**sqlite differences (locked):**
- `sweep(ctx)` uses `ExecContext(ctx, SweepSQL, time.Now().UnixNano())`; failure is logged via `logger().Warn` (D-09) — a periodic error skips the tick; never returned to callers.
- The open-time sweep is called synchronously from `open()` before `New` returns (D-10) — same function, no goroutine.
- `sweepLoop` only starts when `WithSweepInterval(d) > 0` (`d <= 0` = no sweeper, Open Q3 recommendation).
- Use the package `logger()` (postgres's sweeper calls bare `slog.Warn` — an inconsistency to avoid; use `logger()` like `postgres.go:53` does).

---

### `cache/sqlite/optimize.go` (provider adapter) — NO ANALOG

**No existing provider extends the `BatchCache` surface**, and no wrapper decorates `cache.NewConcreteCache` (Finding 2). Use RESEARCH Code Examples §5 verbatim as the pattern:

```go
type Optimizable interface {
	// Checkpoint runs PRAGMA wal_checkpoint(TRUNCATE): cheap, reclaims WAL space.
	Checkpoint(ctx context.Context) error
	// Vacuum runs a full VACUUM: exclusive and heavy; document both costs.
	Vacuum(ctx context.Context) error
}

// batchCache is the value New returns. The frozen root concreteCache does not
// forward extra methods, so the interface assertion target must be built here.
type batchCache[K comparable, V any] struct {
	cache.BatchCache[K, V]
	provider *sqliteCache[K, V]
}

func (b *batchCache[K, V]) Checkpoint(ctx context.Context) error { return b.provider.checkpoint(ctx) }
func (b *batchCache[K, V]) Vacuum(ctx context.Context) error     { return b.provider.vacuum(ctx) }
```

- Exported interface declaration style: follow `cache/cache.go:11-29` (doc comment per method).
- `checkpoint`: `PRAGMA wal_checkpoint(TRUNCATE)` returns **three columns** — `Scan(&busy, &log, &checkpointed)`; `busy != 0` → descriptive error (Pitfall 16, Open Q2 recommendation). Memory mode returns `busy=0, log=-1, ckpt=-1` and is a valid no-op success.
- `vacuum`: `ExecContext(ctx, "VACUUM")` wrapped with `cache/sqlite:`.
- both start with `c.check()` (`initErr` then `closed`).

---

### Tests

All test files share the repo conventions: testify `assert`/`require`, `t.Parallel()` where safe, `t.Context()` for contexts, table/subtests named `lower_snake_case`, `t.Cleanup` to close providers. Coverage gate: `.testcoverage-quick.yml` (file 70 / package 80 / total 75) with **no `cache/sqlite` override** (contrast lines 8-24) and `make coverage-quick` (`Makefile:119-122`) runs per commit — tests ship with each commit.

#### `cache/sqlite/options_internal_test.go` (test) — EXACT analog

**Analog:** `cache/postgres/options_internal_test.go:10-65`
```go
func TestOptions_WithConnString(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithConnString("postgres://user:pass@localhost/mydb")(cfg)

	assert.Equal(t, "postgres://user:pass@localhost/mydb", cfg.ConnString)
}
...
func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	assert.Equal(t, "cache_entries", cfg.TableName)
	assert.Equal(t, 5, cfg.PoolSize)
	assert.Equal(t, 1*time.Minute, cfg.SweepInterval)
	assert.Empty(t, cfg.ConnString)
	assert.Equal(t, time.Duration(0), cfg.DefaultTTL)
}
```
One `TestOptions_WithX` per option + `TestDefaultConfig`. sqlite's defaults assertion must encode the deliberate divergence: `SweepInterval == 0` (no sweeper), `Memory == false`, `Name/Path == ""`, `AutoCheckpoint == 0`.

#### `cache/sqlite/dsn_internal_test.go` (test) — role-match

**Analog:** `cache/postgres/options_internal_test.go` (structure) + `cache/postgres/postgres_internal_test.go:16` (constructing provider value directly):
```go
c := &postgresCache[string, string]{defaultTTL: 10 * time.Second}
```
**Must cover (RESEARCH Test Strategy):** location precedence matrix (memory/path/name/zero, `:memory:` sentinel via `WithPath`), `?`/`#` rejection → `errors.Is(err, ErrInvalidPath)`, invalid-name rejection, `userCacheDir` error propagation, DSN string equality for file/memory suffixes and `_pragma=wal_autocheckpoint(N)` append. Inject `userCacheDir func() (string, error)` — deterministic, no env mutation (Pitfall 10).

#### `cache/sqlite/options_internal... / sqlite_internal_test.go` (test) — role-match

**Analogs:**
- `cache/postgres/postgres_internal_test.go:10-50` — `TestResolveTTL` subtest matrix (`per_key_ttl_overrides_default`, `zero_ttl_uses_default`, `no_ttl_uses_default`, `no_ttl_no_default_returns_nil`). sqlite mirrors with `assert.WithinDuration`/numeric `UnixNano` assertions; constructor `&sqliteCache[string, string]{defaultTTL: ...}`.
- `cache/memcache/memcache_internal_test.go:13-49` — internal error-path test against a broken handle:
```go
c := &memcacheCache[string, string]{
	client: memcache.New(), // client with no servers — all ops will fail
}
...
	t.Run("mset_returns_error", func(t *testing.T) {
		err := c.MSetFunc(context.Background(), map[string]string{"a": "1"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "cache/memcache")
	})
```
sqlite equivalent: build with a bad path / closed DB to exercise `initErr` deferral, `ErrClosed`, and "corrupted JSON ≠ `ErrMiss`".
**Also internal-only tests:** reads never delete rows (`SELECT COUNT(*)` still 1 after an expired `Get`), sweep deletes expired rows, periodic sweep reclaims + stops on `Close` (`WithSweepInterval(10ms)`).

#### `cache/sqlite/sqlite_test.go` (test, black-box) — role-match

**Analog:** `cache/postgres/postgres_test.go` (shape) + `cache/mem/sweeper_test.go:11-39` (timing).

**Helper shape** (`cache/postgres/postgres_test.go:32-43`):
```go
func newTestCache(t *testing.T, connString string) cache.BatchCache[string, string] {
	t.Helper()

	c, err := postgres.New[string, string](
		postgres.WithConnString(connString),
	)
	require.NoError(t, err)

	t.Cleanup(func() { _ = c.Close() })

	return c
}
```
sqlite variant: `newTestCache(t *testing.T, opts ...sqlite.Option) cache.BatchCache[string, string]` with `t.Cleanup` — **no skip helper** (unlike postgres `skipIfNoPostgres`, SQLite is always available). Add `t.Cleanup(func(){ _ = c.Close() })` immediately after every `New` (Pitfall 6, Windows handles).

**Subtest style** (`cache/postgres/postgres_test.go:45-91`): one test func per feature area (`SetGet`, `Close`, `Batch` placeholders) with `t.Run("lower_snake_case", ...)`; `require.ErrorIs(t, err, cache.ErrMiss)` for misses (see `:176-179`).

**Timing idiom** (`cache/mem/sweeper_test.go:21-26`) uses `time.Sleep` today; RESEARCH requires `require.Eventually` instead (no repo usage yet — this is new). Windows guards conditional skips: `if runtime.GOOS == "windows" { ... }` (`release/self_update_test.go:54,72`, `path_tools/root_directory_test.go:25`, `project_probe/errors_test.go:61`); AGENTS.md explicitly warns about Windows permission checks and sidecar timing — use `require.Eventually` for `-wal`/`-shm` absence and never assert Unix permissions on Windows.

**Must cover (RESEARCH Test Strategy table):** `:memory:` + file CRUD, GetOrSet concurrency `-race` (counting setter runs once), TTL matrix + expiry, pragma read-back file `wal/5000/1` and memory `busy_timeout=5000`, bootstrap idempotence, reopen persistence, close idempotence + sidecars, `Optimizable` assertion + `WithAutoCheckpoint` read-back, batch placeholder smoke.

#### `cache/sqlite/example_test.go` (test, example) — EXACT analog minus skip

**Analog:** `cache/postgres/example_test.go:29-44`
```go
func TestPostgresExample_SetGet(t *testing.T) {
	connString := skipIfNoExamplePostgres(t)

	c, err := postgres.New[string, string](postgres.WithConnString(connString))
	require.NoError(t, err)

	err = c.Set(t.Context(), "example", "pg-value")
	require.NoError(t, err)

	value, err := c.Get(t.Context(), "example")
	require.NoError(t, err)
	assert.Equal(t, "pg-value", value)

	err = c.Close()
	require.NoError(t, err)
}
```
sqlite drops the skip helper entirely and uses `t.TempDir()` or `WithMemory()`; constructor returns no error. "Runnable example shape" is the agent's discretion (CONTEXT).

---

### `go.mod` / `go.sum` (config, deps) — EXACT pattern, known delta

**Analog:** existing `go.mod` direct block (`go.mod:5-19`) and indirect block (`:21-114`).

**Locked changes (RESEARCH Standard Stack + Installation):** add direct `modernc.org/sqlite v1.60.1`; indirect adds include `modernc.org/libc v1.77.1` (**exact pin** — upstream mandate, add the pin comment), plus `github.com/dustin/go-humanize`, `github.com/google/uuid`, `github.com/mattn/go-isatty`, `github.com/ncruces/go-strftime`, `github.com/remyoudompheng/bigfft`, `modernc.org/mathutil`, `modernc.org/memory`; existing pins bump: `golang.org/x/sys v0.47.0 → v0.48.0` (line 104) and `golang.org/x/sync v0.22.0 → v0.23.0` (line 17). Expected sequence: `go get modernc.org/sqlite@v1.60.1 && go mod tidy` then verify with `go list -m modernc.org/sqlite modernc.org/libc golang.org/x/sys golang.org/x/sync`. Full pin/CI assertion is Phase 17 (QUAL-01).

---

### `README.md` (docs) — EXACT pattern, edit in place

**Analog:** package index (`README.md:13-37`) and provider table (`:77-83`).

**Index row to add after `postgres`** (`README.md:21`):
```markdown
| [sqlite](#package-cache) | `cache/sqlite` | SQLite cache backend (embedded, survives restarts) |
```

**Provider table row** (`README.md:77-83`):
```markdown
| Package | Backend | Driver | Connection |
|---------|---------|--------|------------|
| `cache/mem` | In-memory | None (stdlib) | None — zero dependency |
| `cache/redis` | Redis | go-redis/v9 | Lazy — dials on first query |
| `cache/valkey` | Valkey | valkey-go | Eager — dials at construction |
| `cache/memcache` | Memcache | gomemcache | Lazy — goroutine ctx wrapper |
| `cache/postgres` | Postgres | pgx/v5 | Eager — pgxpool at construction |
```
sqlite row: `| `cache/sqlite` | SQLite (embedded) | modernc.org/sqlite | Eager — local file, lazy errors |` (wording is discretion). Positioning line from CONTEXT: "mem but survives restarts; redis/postgres without the infra". Keep edits scoped to the index/provider table per CONTEXT ("README package index row") — no rewrite of the cache section prose.

---

## Shared Patterns

### 1. Provider-local functional options
**Source:** `cache/postgres/options.go:14-15`, `cache/redis/options.go:14-15`, `cache/valkey/options.go:14-15`
**Apply to:** `cache/sqlite/options.go` (+ consumed by `sqlite.go`, `dsn.go`)
```go
// Option is a functional option for configuring the Postgres cache provider.
type Option func(*Config)
...
func defaultConfig() *Config { ... }
```
The root `cache.Option` interface (`cache/options.go:13-16`) is **not** consumed by any provider — do not use it for sqlite.

### 2. Error wrapping: `%w` + `cache/sqlite:` prefix
**Source:** `cache/postgres/postgres.go:80,84,90,111`; `cache/redis/redis.go:50-58`
**Apply to:** every method in `sqlite.go`, `sweep.go`, `optimize.go`
```go
if err != nil {
	var zero V
	return zero, fmt.Errorf("cache/postgres: %w", err)
}
```
Miss is the only mapped sentinel: `fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)` — only for `sql.ErrNoRows` (Pitfall 8).

### 3. Deferred open errors (`initErr`) + idempotent `Close`
**Source:** `cache/valkey/valkey.go:19-23,51-53,74-76,97-99`; `cache/mem/mem.go:71-83`
**Apply to:** `sqlite.go` (every primitive method: `check()` = `initErr` then `closed`), `optimize.go`
```go
type valkeyCache[K comparable, V any] struct {
	client     valkey.Client
	initErr    error
	defaultTTL time.Duration
}
```
`New` never fails (D-11); post-Close ops wrap `ErrClosed` (D-12). Do not let the driver's raw `"sql: database is closed"` leak (probe-verified).

### 4. Constructor parity: `New` returns `cache.BatchCache` via `cache.NewConcreteCache`
**Source:** `cache/redis/redis.go:25-42`, `cache/valkey/valkey.go:28-45`, `cache/postgres/postgres.go:37-67`
**Apply to:** `sqlite.go` (`New` wraps `sqliteCache`), `optimize.go` (the returned value is the `batchCache` adapter)
```go
return cache.NewConcreteCache(&valkeyCache[K, V]{
	client:     client,
	initErr:    err,
	defaultTTL: cfg.DefaultTTL,
})
```

### 5. TTL resolution parity
**Source:** `cache/postgres/postgres.go:244-256`
**Apply to:** `sqlite.go` (`resolveTTL` copied semantics, `(int64, bool)` return)
Precedence: per-call `> 0` wins → default `> 0` → none. Store absolute `UnixNano` (TTL-01); never SQL `datetime('now')`.

### 6. Best-effort sweeps + module logger
**Source:** `cache/postgres/sweeper.go:12-37`, `cache/postgres/postgres.go:30-32,53`
**Apply to:** `sqlite.go` (`open` calls `sweep`), `sweep.go`
```go
var logger = sync.OnceValue[*slog.Logger](func() *slog.Logger {
	return slog.With(slog.String("module", "cache/postgres"))
})
```
Sweep failures log and continue (D-09); `Close` cancels the ticker and waits (D-08).

### 7. Test conventions
**Source:** `cache/postgres/postgres_test.go:32-43` (`t.Cleanup` close), `cache/postgres/options_internal_test.go:11` (`t.Parallel`), `cache/memcache/memcache_internal_test.go:13-49` (broken-handle internal tests), `cache/mem/sweeper_test.go:21-26` (timing)
**Apply to:** all four/five test files
- testify `assert` for values, `require` for preconditions.
- `t.Context()` for contexts; `t.TempDir()` for file DBs (`path_tools/path_tool_test.go:14` idiom).
- `t.Cleanup(func(){ _ = c.Close() })` immediately after `New` (Pitfall 6).
- `require.Eventually` for WAL sidecar removal and sweeper effects (new to repo; RESEARCH mandates it over fixed sleeps).
- `runtime.GOOS == "windows"` guards for permission/sidecar assertions (`release/self_update_test.go:54,72`).

### 8. Coverage gate — no sqlite override
**Source:** `.testcoverage-quick.yml:3-6` + `Makefile:119-122`
**Apply to:** every commit in the phase
```yaml
threshold:
  file: 70
  package: 80
  total: 75
```
The override list (`.testcoverage-quick.yml:8-24`) covers Docker-tested providers only; `cache/sqlite` must meet file 70 / package 80 / total 75 with unit tests alone — design for testability (pure `dsn.go` functions, injectable `userCacheDir`, error branches tested).

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `cache/sqlite/optimize.go` | provider service (adapter) | request-response | No provider extends the frozen `BatchCache` surface; no decorator/wrapper exists in the repo. Use RESEARCH Code Examples §5 (Finding 2). |
| `cache/sqlite/dsn.go` | utility | transform | No DSN builder, path validation, or `os.UserCacheDir` use anywhere in the repo. Partial structural guidance only (`cache/postgres/options.go` config shape, `cache/valkey` error deferral). Use RESEARCH §1 + Pattern 3. |

Additionally, two test techniques have **no repo precedent** and must follow RESEARCH directly: `require.Eventually` (sidecar/sweeper timing) and injecting `userCacheDir` as a function seam.

## Metadata

**Analog search scope:** `cache/` (root + mem, redis, valkey, memcache, postgres), `go.mod`, `README.md`, `Makefile`, `.testcoverage-quick.yml`, repo-wide greps for `UserCacheDir` / `MkdirAll` / `require.Eventually` / `t.TempDir` / `runtime.GOOS`
**Files scanned:** 24 (all read directly; all git-tracked, verified via `git ls-files`)
**Pattern extraction date:** 2026-10-08

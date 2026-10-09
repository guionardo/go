# Stack Research: cache/sqlite — Embedded SQLite Cache Provider

**Domain:** Persistence layer addition (SQLite-backed cache provider) to an existing stdlib-first Go utility monorepo
**Project:** `github.com/guionardo/go` — milestone v1.8 `cache/sqlite`
**Researched:** 2026-10-08
**Confidence:** HIGH (driver, pragma, and WAL facts verified against primary sources: Go module proxy, driver v1.60.1 source, sqlite.org)

> This document is the milestone-scoped stack research for v1.8. It supersedes the v1.7
> `project_probe` STACK.md (kept in git history), whose general guidance (Go 1.26 stdlib-first,
> minimal deps, testify, coverage gates) remains valid and is assumed here.

## Executive Summary

The SQLite provider needs **exactly one new direct dependency: `modernc.org/sqlite` v1.60.1** — the
CGO-free, pure-Go SQLite driver. It registers a standard `database/sql` driver named `"sqlite"`,
bundles SQLite **3.53.4** (past both the 3.51.3 WAL-reset corruption fix and the 3.53.3
journal-rollback fix), and supports every platform in the repo's CI matrix (darwin/linux/windows,
amd64 + arm64) with `CGO_ENABLED=0`. The version was verified current on 2026-10-08
(`proxy.golang.org` reports v1.60.1, tagged 2026-09-29); its `go.mod` requires Go 1.26.0 — the repo
is on go 1.26.4, a clean match.

The rest of the implementation is stdlib: `database/sql` for the connection pool and prepared-statement
cache, `encoding/json` for value parity with the existing five providers, `os.UserCacheDir` for the
default path. Multi-process safety comes from an exact pragma recipe applied through the driver's DSN
on every pooled connection: `journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL` — with
WAL's documented semantics (single writer, concurrent readers, same-host only, checkpoint starvation
only under always-active readers) verified against sqlite.org.

Two sharp edges must be designed for, both verified from driver docs/source rather than guessed:
(1) `:memory:` + `database/sql` is only correct with `SetMaxOpenConns(1)` — every new connection opens
a *separate* in-memory database; (2) the driver executes `_pragma` DSN values as verbatim SQL
(multi-statement capable), so the provider should build its default DSN from the validated shorthand
keys (`_busy_timeout`, `_journal_mode`, `_synchronous`) and must reject/escape `?` in user-supplied
paths (everything after the first `?` is parsed as driver DSN query parameters).

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `modernc.org/sqlite` | **v1.60.1** (tagged 2026-09-29) | `database/sql` driver for embedded SQLite | The only mature, actively maintained CGO-free SQLite driver. Registers driver name `"sqlite"`; supports `darwin`, `linux`, `windows` on amd64+arm64 (Windows also 386; Linux also arm/386/loong64/ppc64le/riscv64/s390x); works with `-race`; cross-compiles. No C toolchain on any CI OS. Requires Go ≥ 1.26.0. |
| `database/sql` | Go 1.26 stdlib | Connection pool, per-connection statement cache, `QueryContext`/`ExecContext` | `modernc.org/sqlite` is a `database/sql` driver — no wrapper library needed. The pool replaces the pgx-pool pattern used by `cache/postgres` with stdlib and satisfies the `cacher` primitive's context-carrying methods. |
| SQLite (embedded) | **3.53.4** (compiled into the driver) | Storage engine | Embedded, zero-infra, ACID. WAL provides the required single-writer/multi-reader cross-process model. Contains upstream fixes for both recent corruption bugs (WAL-reset fixed in 3.51.3; journal-rollback fix in 3.53.4) — a reason to pin a *recent* driver, not just any version. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `modernc.org/libc` | **v1.77.1** (indirect) | Runtime libc port used by the transpiled SQLite | Pulled automatically by `go get`. **Must remain exactly v1.77.1** — upstream rule (gitlab.com/cznic/sqlite#177): downstream `go.mod` files must pin the same libc version the driver pins, or mysterious corruption/crashes can follow. Add a comment in `go.mod`; optionally a CI assertion. |
| `golang.org/x/sys` | v0.48.0 (indirect) | OS primitives (locks, mmap) | Transitive; already in the module graph at an older version — `go mod tidy` will settle it. |
| `encoding/json` | stdlib | Value serialization | TEXT columns holding `json.Marshal` output — parity with all existing providers (`fmt.Sprint(key)` for keys). |
| `os` / `path/filepath` | stdlib | `os.UserCacheDir`, path joining | Default location: `filepath.Join(os.UserCacheDir(), <cache name>, "cache.db")` pattern. Verified stdlib semantics: `$XDG_CACHE_HOME`/`$HOME/.cache` (Unix), `$HOME/Library/Caches` (Darwin), `%LocalAppData%` (Windows); errors cleanly when undetermined or XDG path is relative. |

No other runtime dependencies. No testify change (already a direct test dep). No testcontainers for
this provider — see Development Tools.

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `CGO_ENABLED=0` | CI build mode | The whole point of the driver choice: GitHub Actions matrix (Linux/macOS/Windows) needs **no gcc/msys2/TDM-GCC**. `-race` still works (pure Go). Cross-compilation works. |
| Go module/build cache | CI performance | The module zip is ~23.2 MB (148 MB extracted on disk) and the generated SQLite core is a multi-MB Go file — cold builds are slow. Cache `GOMODCACHE`/`GOCACHE` in CI; do not be surprised by a slower first compile than other packages. |
| `go test -race` | Concurrency validation | Provider must be safe for concurrent `Get`/`Set`; singleflight `GetOrSet` is already shared. Pure-Go allows `-race` on every matrix OS. |
| `t.TempDir()` + `Close()` | Test isolation | Tests use real files in temp dirs (no Docker). Always `Close()` before test cleanup: last close checkpoints and deletes `-wal`/`-shm`; Windows CI can fail to remove open files. |
| Self-exec subprocess helper | Multi-process test | A `TestMain`/helper-binary pattern (`os.Exec` re-invoking the test binary) exercises two processes on one DB file — the one behavior unit tests can't cover, and the input to the v1.8 spike (questions.md #4). |
| `make coverage-quick` | Coverage gate | Package ≥80 / file ≥70 / total ≥75. A new `cache/sqlite` package has no Docker excuse for gaps; no threshold override should be needed. |
| `golangci-lint` / `govulncheck` | Lint + vuln gate | modernc.org/sqlite participates in the Go vulnerability database (its SECURITY.md commits to vulndb disclosure), so the existing `govulncheck` job covers it. |

## Installation

```bash
# One new direct dependency (plus indirection):
go get modernc.org/sqlite@v1.60.1

# Confirm the required pins landed (libc must be v1.77.1):
go list -m modernc.org/sqlite modernc.org/libc

# CI / local quality gates (unchanged targets):
CGO_ENABLED=0 go build ./...
make lint
make coverage-quick
```

```go
// cache/sqlite/sqlite.go
import (
    "database/sql"
    _ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

db, err := sql.Open("sqlite", dsn)
```

## The DSN & Pragma Recipe (explicit)

Build the DSN from a **plain absolute filename** (not a `file:` URI) plus the driver's validated
shorthand keys. Plain-filename queries are consumed by the driver itself before `sqlite3_open_v2`,
so the underscore keys work and Windows path escaping is a non-issue:

```go
// File mode (default path or explicit path). Apply on every pooled connection via DSN.
dsn := path +
    "?_busy_timeout=5000" +   // must be first in driver apply order; per-connection
    "&_journal_mode=WAL" +    // persistent on the file; same-host multi-process only
    "&_synchronous=NORMAL" +  // WAL-safe (no corruption), gives up power-loss durability only
    "&_txlock=immediate"      // explicit write transactions acquire the write lock up front
```

```go
// :memory: mode. WAL/synchronous pragmas are ignored by SQLite for in-memory DBs
// (journal stays MEMORY/OFF; no error) — keep them for a single code path.
db, _ := sql.Open("sqlite", ":memory:?_busy_timeout=5000&_txlock=immediate")
```

| Pragma | Value | Why this value | Evidence |
|--------|-------|----------------|----------|
| `journal_mode` | `WAL` | Single writer + concurrent readers across *processes*; persistent on the file; last close checkpoints and deletes `-wal`/`-shm`. Does **not** work on network filesystems (same-host only). | sqlite.org/wal.html (primary) |
| `busy_timeout` | `5000` ms | Single-statement DML uses a *blocking* lock with the busy handler — concurrent CLI processes wait instead of failing. Every connection needs it (DSN guarantees this). A read→write *upgrade* in a deferred transaction uses a non-blocking lock and can still return `SQLITE_BUSY` immediately — which is why explicit write transactions get `_txlock=immediate`. | sqlite.org wal-lock.md, pragma.html (primary) |
| `synchronous` | `NORMAL` | In WAL mode: "WAL mode is safe from corruption with synchronous=NORMAL… WAL mode does lose durability. A transaction committed in WAL mode with synchronous=NORMAL might roll back following a power loss." Correct trade for a cache; default FULL fsyncs every commit. | sqlite.org/pragma.html (primary) |
| `wal_autocheckpoint` | leave default (`1000` pages ≈ 4 MB) | Automatic PASSIVE checkpoints recycle the WAL. Only pathological always-active readers starve it. Do **not** set 0. | sqlite.org/wal.html (primary) |
| `foreign_keys` | omit | Cache table has no FKs; nothing to enforce. | schema design |
| `auto_vacuum` | omit (default NONE) for v1 | Only takes effect before the first table is written; requires `incremental_vacuum`/`VACUUM` to actually reclaim. Document `VACUUM` as the manual compaction path; revisit only if file size becomes a complaint. | driver source (applies auto_vacuum before first write for exactly this reason); sqlite.org pragma docs |

**Do not use `_pragma=...` for provider-constructed DSNs.** Each `_pragma` value is executed as
verbatim SQL text (multiple statements after `;` run too), so it carries injection authority; the
shorthand keys are validated against the mattn-compatible value set and fail the open on a typo. If
`_pragma` is ever needed, enable `sqlite.StrictPragmas(true)` (still a process-global opt-in).
Also reject or URL-safe-escape `?` and `#` in user-supplied paths: for plain filenames the driver
treats everything after `?` as its own DSN query — a raw `?` both truncates the path and can inject
driver parameters.

## database/sql Integration Points

| Concern | Recommendation | Rationale |
|---------|----------------|-----------|
| Driver import | `import _ "modernc.org/sqlite"`; `sql.Open("sqlite", dsn)` | The driver registers itself as `"sqlite"` in `init()`. |
| Pool size (file mode) | `db.SetMaxOpenConns(1)` as the provider default (option to raise later) | SQLite has one writer; one connection per process removes intra-process lock contention entirely while WAL + `busy_timeout` handles *inter-process* concurrency — which is the actual requirement. The driver's own performance docs recommend bounding the pool; unbounded pools let periodic queries pile up. |
| Pool size (`:memory:`) | **Must** be 1: `SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, no `ConnMaxLifetime`/`ConnMaxIdleTime` | Every new connection to `:memory:` opens a **separate** database; if the pool closes the last connection the DB is deleted. Do not use `file::memory:?cache=shared` — SQLite discourages shared cache and modernc strips queries from plain names anyway. |
| Statement strategy | `db.QueryRowContext` / `db.ExecContext` with parameters; `INSERT … ON CONFLICT(cache_key) DO UPDATE` for Set/upsert | Upsert is native since SQLite 3.24 (bundled 3.53.4). `database/sql` caches prepared statements per connection automatically. |
| Batch (MSet/MSet/MDel) | One `BEGIN IMMEDIATE` transaction (`_txlock=immediate`) wrapping N executions of one prepared statement; delete-by-key loop or single `IN` with dynamic placeholders | Amortizes commit/sync cost; `immediate` avoids upgrade-`SQLITE_BUSY`. Matches the `BatchCache` best-effort error-joining convention. |
| Schema | `CREATE TABLE IF NOT EXISTS cache_entries (cache_key TEXT PRIMARY KEY, value TEXT NOT NULL, expires_at INTEGER)` + partial index `ON cache_entries(expires_at) WHERE expires_at IS NOT NULL` | Mirrors the postgres provider's `schema.go` pattern; INTEGER unix-millis expiry avoids date/time parsing entirely (SQLite has no native timestamp type). |
| TTL | Lazy filter on read (`expires_at IS NULL OR expires_at > ?`) + sweep expired rows on open (`DELETE … WHERE expires_at <= ?`) | Per the milestone; matches the existing sweeper precedent. No background goroutine required. |
| Error mapping | Wrap failures with `cache.ErrMiss` on empty row (postgres precedent); treat `SQLITE_BUSY` after timeout as a normal error to surface/retry | Keep parity with `cache/postgres` error wrapping. |
| Context | Use the `…Context` variants throughout | The `cacher` primitive methods already take `context.Context`. |

## CGO / CI Implications (explicit)

- **No C toolchain anywhere.** `CGO_ENABLED=0` builds on linux/amd64, darwin/arm64, and
  windows/amd64 CI runners; no gcc, no msys2, no TDM-GCC, no `CC` env plumbing. This is the single
  biggest CI-simplicity win versus the CGO drivers.
- **Cross-compile + race detector both work** (pure Go), unlike CGO drivers.
- **Cold-build cost is real.** ~23 MB module download (148 MB extracted) and a multi-MB generated
  source file make the first build noticeably slower than other packages; cache `GOMODCACHE` and
  `GOCACHE` in CI.
- **Dependency graph stays small**: driver + `modernc.org/libc`, `mathutil`, `memory`, `fileutil`,
  `x/sys`, a handful of tiny utils. No C bindings, no build tags to manage.
- **libc pinning is a repo hygiene rule.** `go mod tidy` will resolve `modernc.org/libc` to
  v1.77.1 today (MVS). If another future dependency ever requests a *higher* libc, the driver breaks
  silently-ish; add a `go.mod` comment and consider a one-line CI check
  (`go list -m -f '{{.Version}}' modernc.org/libc` equals v1.77.1) while v1.60.1 is pinned.
- **Vuln scanning**: covered by `govulncheck` via the Go vuln DB.

## Alternatives Considered

| Recommended | Alternative | Version (checked 2026-10-08) | When to Use Alternative |
|-------------|-------------|------------------------------|--------------------------|
| modernc.org/sqlite | `github.com/mattn/go-sqlite3` | v1.14.52 | Only if the consuming binary already requires CGO. It is ~17–31% faster on simple CRUD writes (independent M-series Mac benchmark) but requires `CGO_ENABLED=1` + gcc **on all three CI OSes** (Windows needs a TDM-GCC toolchain), breaks easy cross-compilation, and contradicts this repo's CGO-free CI. Not worth it for a cache. |
| modernc.org/sqlite | `github.com/ncruces/go-sqlite3` | v0.35.6 | A serious pure-Go alternative (wasm2go; only Go + x/sys deps; active). Choose it when dependency minimalism or specific extensions (e.g. encryption-at-rest) dominate **and** you accept the project's own documented caveat: "memory usage will be higher than alternatives" per Wasm-sandboxed connection, plus v0.x API maturity. Rejected for v1.8: higher per-connection memory and a smaller ecosystem than modernc. |
| modernc.org/sqlite | `zombiezen.com/go/sqlite` | v1.4.2 | Wraps modernc but **deliberately provides no `database/sql` driver** (its own `Conn` API); last release May 2025. Only if you wanted its non-`database/sql` API + migration helpers. Adds an API layer this provider does not need. |
| SQLite WAL | `go.etcd.io/bbolt` | v1.5.0 | Never for this feature: bbolt takes an exclusive OS file lock — "multiple processes cannot open the same database at the same time"; read-only shared locks block writers. Fails the multi-process requirement. |
| SQLite WAL | `github.com/dgraph-io/badger/v4` | v4.9.6 | Never for this feature: designed for single-process access. |
| `database/sql` + JSON | `github.com/glebarez/go-sqlite` (+ GORM) | v1.23.0 | Only inside an app that already uses GORM; it is a GORM driver shim over modernc. Pulls the ORM stack for a 6-statement provider — rejected. |
| Plain driver DSN | `sql.OpenDB(sqlite.NewConnector(dsn))` | — | Use the `NewConnector` API (driver ≥ v1.56.0) only if interposing on physical connections (tracing/metrics) becomes a requirement; `sql.Open` is simpler and equivalent today. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Any CGO SQLite driver (mattn, cgo amalgamation, system libsqlite3) | Destroys the CGO-free CI guarantee; needs gcc on Windows/macOS/Linux runners; hurts cross-compile | `modernc.org/sqlite` |
| ORMs (GORM + glebarez, `gorm.io/driver/sqlite`) | A cache is 6 SQL statements; ORM scaffolding/model tags add dependency weight and indirection for zero value | Raw `database/sql` |
| `sqlx`, `squirrel`, query builders | Stdlib `database/sql` covers parameterized queries, scanning, and transactions here | stdlib |
| `_pragma=...` in runtime-built DSNs | Values execute as verbatim SQL (multi-statement capable) — injection surface | Validated shorthand keys (`_busy_timeout`, `_journal_mode`, `_synchronous`, `_txlock`); `StrictPragmas(true)` if `_pragma` is unavoidable |
| Raw user path passed into the DSN without `?`/`#` handling | Everything after the first `?` becomes driver query params — path truncation + param injection | Validate/reject `?` and `#`, or escape when constructing the DSN |
| `file::memory:?cache=shared` / `cache=shared` | SQLite explicitly discourages shared-cache mode (different transaction semantics; race-prone history) | `:memory:` + `SetMaxOpenConns(1)` |
| `synchronous=OFF`, `journal_mode=OFF` | Corruption risk on power loss; `journal_mode=OFF` is silently disabled under defensive mode and is meaningless for a cache | `synchronous=NORMAL` |
| Disabling autocheckpoint (`wal_autocheckpoint=0`) / long-lived read transactions | Documented paths to unbounded WAL growth (checkpoint starvation) | Keep the 1000-page default; keep read transactions short |
| Migration frameworks / `golang-migrate` etc. | Single idempotent `CREATE TABLE IF NOT EXISTS` at open (postgres provider precedent) | `schema.go` constant executed on `New` |
| Docker/testcontainers E2E for this provider | SQLite is in-process; Docker adds nothing and would break the local/CI parity promise | Real temp files + self-exec subprocess for the multi-process case |
| A second storage engine (bbolt/badger) "for speed" | Multi-process concurrent access is the milestone requirement and both fail it | SQLite WAL |
| Unbounded `database/sql` pool | Driver docs: unbounded connections each carry page cache + libc state; periodic queries can pile up | `SetMaxOpenConns(1)` default |

## Stack Patterns by Variant

**If the path is empty (default mode):**
- `os.UserCacheDir()` → `filepath.Join(dir, "<cache name>")` → `MkdirAll(0o755)` → `cache.db`; surface `UserCacheDir`'s error if the location is undetermined (it documents this behavior).

**If the path is `:memory:`:**
- `SetMaxOpenConns(1)` + `SetMaxIdleConns(1)`, no conn lifetime/idle timeouts; WAL pragmas harmless no-ops. Optionally skip WAL pragmas for clarity.

**If the path is explicit (file):**
- Same DSN recipe as default mode. Document that WAL does not work on network filesystems (NFS/SMB) — a cache directory on a network mount silently degrades.

**If concurrent processes on one file:**
- `busy_timeout=5000` + single-statement writes; expect rare `SQLITE_BUSY` on post-crash recovery / last-close cleanup windows (documented WAL-mode cases) — treat as retryable, but keep transactions short. MSet/MSet/MDel use `BEGIN IMMEDIATE`.

**If TTL sweep runs:**
- Sweep expired rows once on `New` (before serving reads), then lazy delete on read. No goroutine. `VACUUM` remains a documented manual remedy for file size, not an automatic behavior.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| `modernc.org/sqlite` v1.60.1 | Go **≥ 1.26.0** (its `go.mod`) | Repo `go 1.26.4` — satisfied. Local toolchain 1.27.0 also fine. |
| `modernc.org/sqlite` v1.60.1 | `modernc.org/libc` **v1.77.1 (exact)** | Upstream mandate (issue #177). Add a `go.mod` comment; guard against future MVS bumps. |
| `modernc.org/sqlite` v1.60.1 | SQLite **3.53.4** embedded | Includes UPSERT (≥3.24), WAL, `ON CONFLICT … excluded`, `journal_size_limit`; post-WAL-reset-bug (3.51.3) and post-journal-rollback-fix (3.53.4). |
| driver shorthand DSN keys | v1.55.0+ | `_busy_timeout`, `_journal_mode`, `_synchronous`, `_auto_vacuum`, `_foreign_keys`, `_query_only` added in v1.55.0 (2026-07-20); v1.60.1 has them. `_txlock` and `_pragma` are older. |
| `StrictPragmas` / `NewConnector` | v1.56.0 / v1.60.0+ | Available if needed; not required by the default design. |
| `github.com/stretchr/testify` v1.11.1 | unchanged | Existing test dependency; no change. |
| `go.etcd.io/bbolt` v1.5.0 / `dgraph-io/badger/v4` v4.9.6 | rejected | Version facts verified; exclusion is architectural (multi-process), not version-driven. |

## Disposition of Open Questions (from `.planning/research/questions.md`)

1. **Driver & version — RESOLVED.** `modernc.org/sqlite` v1.60.1, current as of 2026-10-08. The
   dependency tree fits the minimal-deps posture: 1 direct import, ~10 small transitive modules, no
   CGO. The unresolved "1.3–2.0x gap" claim is refined, not refuted: the maintainer's own Sept 2026
   measurements show **1.3x–2.0x CPU-time ratios only for CPU-bound queries** (2.0x unindexed
   ORDER BY, 1.9x GROUP BY, 1.3x correlated subquery); I/O-bound work is dominated by the OS, an
   independent CRUD benchmark shows 10–31% on simple ops, and under concurrency the SQLite
   single-writer lock dominates, not the driver. For a local cache this is a non-issue.
2. **Pragma set — RESOLVED.** `journal_mode=WAL` (persistent; same-host only), `busy_timeout=5000`
   on every connection, `synchronous=NORMAL` (WAL-safe; durability trade only), default
   `wal_autocheckpoint=1000`; `_txlock=immediate` for explicit write transactions. Exact strings in
   the DSN recipe above.
3. **DB growth — RESOLVED with guidance.** Main file grows to a high-water mark and does **not**
   shrink on DELETE (freelist reuse); WAL is recycled at ~1000 pages (~4 MB) and deleted by the last
   close; `auto_vacuum` cannot be set after the first table is written and only pays off with
   `VACUUM`/`incremental_vacuum`. Document "size stabilizes; run `VACUUM` manually if needed".
4. **Two-process spike — STILL REQUESTED (implementation-level).** Research narrows the expected
   behavior, it cannot replace the spike: single-statement writes should block-and-wait under
   `busy_timeout`; a deferred read→write upgrade may return `SQLITE_BUSY` despite the timeout;
   `_txlock=immediate` should remove that class. The spike should assert these and pin the retry
   policy. Flag for the v1.8 plan.

## Sources

- `proxy.golang.org` `@latest` / `.mod` for `modernc.org/sqlite` v1.60.1, `modernc.org/libc` v1.77.1, `github.com/mattn/go-sqlite3` v1.14.52, `github.com/ncruces/go-sqlite3` v0.35.6, `zombiezen.com/go/sqlite` v1.4.2, `go.etcd.io/bbolt` v1.5.0, `github.com/dgraph-io/badger/v4` v4.9.6, `github.com/glebarez/go-sqlite` v1.23.0 — **HIGH** (primary registry data, checked 2026-10-08)
- `modernc.org/sqlite` v1.60.1 module source, read from `$GOMODCACHE` after `go mod download`: `driver.go` (DSN docs, apply order, `StrictPragmas`), `sqlite.go` (DSN validation/apply), `doc.go` (platform table, Performance section), `CHANGELOG.md` (v1.55.0 shorthand keys, v1.58.0 SQLite 3.53.4, v1.59.0 libc pin + `SetMaxOpenConns` guidance, v1.60.0/v1.60.1), `go.mod` (go 1.26.0; libc v1.77.1) — **HIGH** (primary)
- `sqlite.org/wal.html` (WAL semantics, checkpoint starvation, SQLITE_BUSY cases, persistence, last-close cleanup), `sqlite.org/pragma.html` (`busy_timeout`, `wal_autocheckpoint`, `synchronous` WAL safety matrix, `journal_size_limit`), `sqlite.org/src/doc/trunk/doc/wal-lock.md` (blocking vs non-blocking lock cases) — **HIGH** (primary docs, fetched 2026-10-08)
- `github.com/mattn/go-sqlite3` README (CGO_ENABLED=1 + gcc requirement, Windows toolchain, DSN table), `github.com/ncruces/go-sqlite3` README (`:memory:` pooling warning, Wasm memory caveat, dependencies), `zombiezen.com/go/sqlite` README (no `database/sql` by design; wraps modernc), `go.etcd.io/bbolt` docs (exclusive file lock; multi-process limitation) — **MEDIUM** (official project docs)
- `go doc os.UserCacheDir` — local Go stdlib docs — **HIGH** (primary)
- Independent benchmark: `go-sqlite-benchmark-mattn-vs-modernc` (M4, Go 1.26.1) — 17–31% mattn advantage on simple CRUD, modernc wins bulk scans — **MEDIUM** (third-party, consistent with maintainer figures)
- Prior research cache: `DATA_yvs1wule` claims in `.planning/research/questions.md` — all five dispositioned above; none refuted

---
*Stack research for: v1.8 cache/sqlite milestone*
*Researched: 2026-10-08*

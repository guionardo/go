# Pitfalls Research

**Domain:** Adding an embedded SQLite cache provider (`cache/sqlite`) to the existing `github.com/guionardo/go` utility library — multi-process access, TTL cleanup, pure-Go driver, strict CI matrix (Linux/macOS/Windows), coverage gates (package ≥80%, file ≥70%, total ≥75%)
**Researched:** 2026-10-08
**Confidence:** HIGH for SQLite semantics (official sqlite.org documentation and forum statements); MEDIUM for `modernc.org/sqlite` version-specific behavior and Windows ecosystem reports (community bug reports, cross-checked across independent incidents)
**Scope:** Milestone v1.8. Complements `.planning/research/questions.md` — each of its four questions is resolved or refuted below, with the remaining spikes called out.

---

## How to Read This Document

Pitfalls are triaged by severity for **this** milestone:

- **Critical** — causes data loss, silent wrong behavior, corruption risk, CI red across the matrix, or a user-visible product bug. Must be designed against before code is written.
- **Moderate** — causes flaky behavior, hard-to-cover code, or maintenance problems. Must have a written decision.
- **Minor** — polish/parity issues; address during normal implementation.

"Phase to address" suggestions map to the natural build order for this milestone and are inputs to ROADMAP.md, not fixed phases:

- **P1 — Provider foundation:** driver pin, DSN construction, open/close lifecycle, location modes (`:memory:`, `UserCacheDir`, explicit), schema bootstrap + `user_version`, pragma verification.
- **P2 — Core semantics:** Get/Set/Delete, JSON/TTL parity, lazy expiry, sweep-on-open, error mapping (`ErrMiss`/`ErrClosed`).
- **P3 — Batch + concurrency:** MGet/MSet/MDel, `_txlock`/retry policy, multi-process spike, connection pool tuning.
- **P4 — Hardening + delivery:** Windows/CI green, coverage gates, docs (`doc.go`/example), e2e/benchmark registration, growth/checkpoint guidance.

---

## Critical Pitfalls

Mistakes that cause data loss, corruption risk, rewrite, or a red CI matrix.

### Pitfall 1: `:memory:` + `database/sql` connection pool = multiple separate databases

**What goes wrong:** The milestone requires `:memory:` when the path is empty. `sql.Open("sqlite", ":memory:")` looks fine and `New` succeeds. Then `Set` writes through pooled connection A, and a later `Get` is served by pooled connection B, which has its own *empty* database. The key appears to vanish (`cache.ErrMiss` right after a successful `Set`). Worse, the schema created by the first connection (`CREATE TABLE`) does not exist on connection B: `no such table: cache_entries`. Symptoms are flaky and load-dependent — green locally, red in CI or under `-race`.

**Why it happens:** `database/sql` is a pool, not a connection. Every SQLite connection opened with the plain `:memory:` filename creates a **brand-new private database** (sqlite.org/inmemorydb.html: "Every `:memory:` database is distinct from every other"). The pool only opens extra connections when existing ones are busy, so the bug needs concurrency (parallel subtests, `t.Parallel`, concurrent goroutines, `-count`) to surface.

**How to avoid:**
- For `:memory:` mode, call `db.SetMaxOpenConns(1)` **and** `db.SetMaxIdleConns(1)` so every operation goes through the one connection that owns the database.
- Do **not** set `SetConnMaxLifetime`/`SetConnMaxIdleTime` in memory mode — retiring the only connection destroys the database and all cached data.
- Do not silently use `file::memory:?cache=shared` as the fix. Shared cache is a different locking model (table-level locks, `SQLITE_LOCKED`, unlock-notify semantics), is officially discouraged, and the database dies when the last connection closes. If shared memory is ever needed, it must be a deliberate, documented option — not the default.
- Document in `doc.go` that memory mode serializes all operations through one connection.
- Add a test that pins two connections (`db.Conn`) and asserts they see the same rows — or simply assert `db.Stats().OpenConnections == 1` after concurrent load.

**Warning signs:** `no such table` or `cache.ErrMiss` immediately after `Set` in tests that run in parallel or with `-count=N`; `db.Stats().OpenConnections > 1` in memory mode; a test that passes alone and fails in the full package run.

**Phase to address:** P1 — constructor/open path. The pool configuration decision is part of the location-modes feature.

---

### Pitfall 2: Pragmas applied to one pooled connection only — or silently ignored

**What goes wrong:** The provider appears to configure `busy_timeout` and WAL, but under contention every write still fails instantly with `database is locked (5) (SQLITE_BUSY)` with zero wait. Two root causes compound:

1. **One-off `db.Exec("PRAGMA busy_timeout = 5000")` configures exactly one connection.** `database/sql` hands it to whichever pooled connection answers the Exec; every other connection — and every connection opened later as the pool grows — runs with `busy_timeout=0`. `synchronous` and `foreign_keys` have the same per-connection nature. (Multiple independent production post-mortems document exactly this: a startup log says "pragmas configured" while 24 of 25 connections run defaults.)
2. **DSN parameter dialects differ per driver and per version.** Many published examples use `mattn/go-sqlite3` syntax (`?_journal_mode=WAL&_busy_timeout=5000`) against `modernc.org/sqlite`. Older modernc versions silently ignored unknown keys, so the database ran in `delete` journal mode with no busy handler. **Refutation for the pinned version:** `modernc.org/sqlite` v1.60.1 now supports validated mattn-compatible shorthands — `_busy_timeout`/`_timeout`, `_journal_mode`/`_journal`, `_synchronous`/`_sync`, `_auto_vacuum`/`_vacuum`, `_foreign_keys`/`_fk`, `_query_only` — and applies them in a fixed order with `busy_timeout` and `auto_vacuum` first. But `_pragma` values remain **executed verbatim per connection** and unknown keys can still be silently dropped, so "it looks configured" is not evidence.

There is a real trade-off the naive fix misses: putting `_journal_mode(WAL)` in the DSN makes **every new pooled connection** re-run the journal-mode change. WAL conversion needs a brief exclusive lock the first time; when the pool grows while another process is writing, a new connection can fail to open with `SQLITE_BUSY`. The safer split, used by several production codebases: DSN carries the per-connection pragmas (`busy_timeout`, `synchronous`, `foreign_keys`); WAL is set **once** on open, verified by read-back, with a bounded retry.

**Why it happens:** Pragmas look global but are not; drivers differ in DSN syntax; `sql.Open` is lazy so misconfiguration is invisible until the first contention or the first new pooled connection.

**How to avoid:**
- Build the DSN with per-connection pragmas only, e.g. `file:<path>?_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)` (or the validated shorthand equivalents, which guarantee `busy_timeout` first).
- Set `journal_mode=WAL` exactly once during `New`, on a dedicated connection (`db.Conn`), read the returned mode back, and require `"wal"`:
  - `PRAGMA journal_mode = WAL` **returns a row**. `Exec` is the wrong verb to trust; use `QueryRow(...).Scan(&mode)` and compare case-insensitively.
  - If the returned mode is not `wal` (unsupported filesystem, in-memory DB), fail loudly or degrade with an explicit, documented warning — never assume.
- Retry the WAL switch a bounded number of times (the switch does not always invoke the busy handler; see Pitfall 4).
- **Verify, don't assume:** add a test that pins several pooled connections concurrently (`db.Conn`, or `SetMaxOpenConns(N)` with N parallel workers) and asserts `PRAGMA busy_timeout` and `PRAGMA journal_mode` on *each* connection. This test is cheap and catches the entire failure class.
- Force one connection eagerly at `New` (`PingContext` or the pragma verification query) so a bad DSN or bad path fails at construction, not at first `Get`.

**Warning signs:** `SQLITE_BUSY` returned in milliseconds while `busy_timeout` is set to seconds; `PRAGMA busy_timeout` returning 0 on a freshly opened connection; `PRAGMA journal_mode` returning `delete`/`truncate`; pragma values set in code but absent in `db.Stats()`-era tests; DSN copied from a mattn example.

**Phase to address:** P1 — DSN builder + open sequence. The per-connection verification test belongs in P1 and must stay green for the life of the milestone.

---

### Pitfall 3: WAL on a network filesystem — corruption risk, and `journal_mode` lies silently

**What goes wrong:** The user (or the default) points the cache at an NFS/SMB/UNC mount, a Docker volume backed by a network share, or a cloud-synced folder (OneDrive/Dropbox/Google Drive). WAL mode requires a shared-memory wal-index (`.db-shm`) that only works when all processes are on the same host that stores the file. On network filesystems, locking and `fsync` guarantees vary or are broken; the SQLite team states flatly that WAL **does not work** over a network filesystem and that using SQLite on network storage risks corruption. `PRAGMA journal_mode=WAL` does not error in this situation — it **silently returns the previous mode** when shared memory is unavailable. The provider believes WAL is on while the database is unprotected.

**Why it happens:** `os.UserCacheDir()` is local on most developer machines, so nobody tests the network case. In enterprise environments home directories (and thus cache directories) are redirected to network storage; macOS network homes and Windows folder redirection do the same. The `journal_mode` fallback is documented but easy to miss.

**How to avoid:**
- Default to `os.UserCacheDir()` (local per-user disk) and keep explicit paths an advanced, documented option.
- **Check the read-back** from `PRAGMA journal_mode=WAL` and return an error (or a loud, documented degradation) if it is not `wal`. This single check converts silent corruption risk into a clear failure.
- Document in `doc.go` and the package example: "Do not place the database on a network share, network-mounted home directory, or cloud-synced folder." Name the concrete offenders (NFS, SMB/CIFS, UNC paths, OneDrive/Dropbox/Google Drive, Docker bind mounts from Windows drives).
- When the check fails and the caller still wants a working cache, the only supported fallback is a rollback-journal mode (`DELETE`), which is single-writer but does not depend on shared memory — make it an explicit, documented decision, never an automatic one.
- Keep in mind `os.UserCacheDir()` can still be on network storage; a "where is the DB" helper/log line during `New` (at debug level) materially helps diagnosis.

**Warning signs:** `PRAGMA journal_mode` read-back differs from `wal`; `SQLITE_IOERR_SHM`, `SQLITE_IOERR_SHMSIZE`, or `disk I/O error` on UNC paths; `.db-shm`/`.db-wal` files left behind unexpectedly on a remote path; corruption reports from a synced folder.

**Phase to address:** P1 — location modes + open verification; P4 — docs.

---

### Pitfall 4: `busy_timeout` does not cover the failures you think it covers

**What goes wrong:** `busy_timeout` is treated as "concurrent writers queue for up to N ms and then succeed." In reality SQLite deliberately **does not invoke the busy handler** in several situations and returns `SQLITE_BUSY` immediately:

- **Deferred read→write upgrade** inside a transaction that started with a `SELECT`: if another connection wrote in the meantime, the upgrade fails at once — in WAL mode as `SQLITE_BUSY_SNAPSHOT` (extended code **517**). Waiting cannot un-stale the snapshot, so no timeout value helps. `BEGIN DEFERRED` (the `database/sql` default) is the trap; `BEGIN IMMEDIATE` takes the write lock up front, where the busy handler *does* apply and writers queue.
- **`COMMIT` contention** with open readers (SQLite docs: COMMIT can return `SQLITE_BUSY`).
- **WAL bootstrap / last-close cleanup:** opening a fresh WAL database or switching to WAL needs a short exclusive lock; the busy handler does not always cover the journal-mode switch (multiple production fixes add an application-level retry around exactly this).
- **Checkpoint locking** and shared-memory lock I/O errors.

Even with WAL + `_txlock=immediate` + a generous timeout, plain `SQLITE_BUSY` remains possible under load, so a bounded application-level retry is required for a library that promises a clean `Get`/`Set` API.

**Why it happens:** The `busy_timeout` mental model ("it retries lock contention") is true only for a subset of lock paths; the deadlock-avoidance exception and the snapshot case are documented but widely unknown. See `sqlite.org/c3ref/busy_timeout.html` and the `SQLITE_BUSY_SNAPSHOT` result-code docs.

**How to avoid:**
- Keep write operations **single-statement/autocommit** wherever possible (upsert, delete): autocommit writes start as writes, so the busy handler applies.
- If a multi-statement write transaction is ever needed (e.g., a future batch implementation that reads before writing), start it with `BEGIN IMMEDIATE` — via `_txlock=immediate` in the DSN, or by using a dedicated write connection. Never rely on deferred transactions that read first.
- Add a small bounded retry helper for the provider: 3–5 attempts, ~10–50 ms backoff with jitter, `ctx`-aware, retrying the **whole operation** (not a statement inside a stale snapshot). Detect retryable errors via `errors.As(err, new(*sqlite.Error))` and `Code() & 0xff` ∈ {5 SQLITE_BUSY, 6 SQLITE_LOCKED}; never substring-match only (version-dependent text).
- Treat `SQLITE_BUSY_SNAPSHOT` (517) as "restart the transaction," never "retry the statement."
- Sweep-on-open and the WAL switch must be **best-effort resilient**: a busy database during open should be retried briefly, and sweep failure should be logged and skipped (mirroring the postgres provider's best-effort sweep at `cache/postgres/sweeper.go`).
- **Spike (from questions.md #4):** spawn two processes writing through `modernc.org/sqlite` against one file; measure whether `SQLITE_BUSY` escapes after timeout, and confirm the retry helper absorbs it. Do this before claiming multi-process safety.

**Warning signs:** `database is locked (5)` in ~0 ms despite a 5 s timeout; `database is locked (517)` or `SQLITE_BUSY_SNAPSHOT`; flaky tests under `-race`/`-count`; failures only when two test binaries hit the same file.

**Phase to address:** P3 — batch/concurrency. The `_txlock`/retry policy and the two-process spike are P3 exit criteria. Single-statement CRUD in P2 should already avoid deferred-then-write shapes.

---

### Pitfall 5: Concurrent first-open schema bootstrap race

**What goes wrong:** Two processes start within milliseconds on a fresh database (parallel CLI invocations, two test binaries, a user double-clicking twice) and both bootstrap the schema. Both read `PRAGMA user_version = 0` outside a transaction, both decide to migrate, and the loser dies with `duplicate column name`, `UNIQUE constraint failed`, `table already exists` (for non-idempotent DDL), or `database is locked` while switching to WAL. The error appears only on a cold database, so it survives to production and looks random.

**Why it happens:** `CREATE TABLE IF NOT EXISTS` and `PRAGMA user_version` checks are not atomic across connections. The check-then-act sequence is the classic race. Deferred transactions make it worse: a read (`IF NOT EXISTS` check) followed by a write (DDL) fails with `SQLITE_BUSY_SNAPSHOT`, which `busy_timeout` never retries.

**How to avoid:**
- Wrap the **entire** bootstrap (version read, all DDL, version write) in one `BEGIN IMMEDIATE` transaction and **re-read `PRAGMA user_version` inside the lock**; if another process won the race, roll back as a clean no-op.
- Keep v1 schema bootstrap idempotent: `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` + a single `PRAGMA user_version = 1` write. Never split the DDL across separate transactions.
- Treat a concurrent "already exists" as success (re-check inside the lock).
- Add bounded retry around `New` because the WAL switch is a second, independent bootstrap hazard (Pitfall 4).
- Test it: 4 goroutines (and optionally 2 processes) gated on a shared start channel opening the same fresh path; all must succeed and the schema must converge. Many projects document this test reproducing the bug pre-fix and passing 20/20 post-fix.
- Design `user_version` migration now even though v1 has one version: it is the mechanism future milestones will use, and the correct pattern is much cheaper to install on day one.

**Warning signs:** cold-start failures only; `duplicate column`/`UNIQUE constraint`/`database is locked` on parallel CI; passes with `-p 1` but fails with package parallelism; users reporting "first run after clearing cache fails."

**Phase to address:** P1 — schema bootstrap and open sequence.

---

### Pitfall 6: Windows file semantics — leaked handles break `t.TempDir`, CI goes red

**What goes wrong:** On Windows a file cannot be deleted while any handle holds it open (`ERROR_SHARING_VIOLATION`, "The process cannot access the file because it is being used by another process"). Tests that open the SQLite cache in a `t.TempDir()` and never call `Close` fail during temp-dir cleanup — but **only on windows-latest**, so the failure appears after the PR is pushed. A pending transaction can also prevent `database/sql` from actually closing the underlying handle (documented in mattn issue #906), and `-wal`/`-shm` sidecar files can keep handles alive briefly after `Close`. Antivirus and Windows Search indexing can hold transient locks, producing intermittent `SQLITE_BUSY` unrelated to application logic.

**Why it happens:** Unix permits unlinking open files; Windows does not. The macOS/Linux runs are green, so the leak is invisible locally. The repo targets all three OSes in CI (`go.yml` matrix) and must be Windows-clean.

**How to avoid:**
- Make `Close()` idempotent, stop the sweeper exactly once (mirroring `cache/postgres/postgres.go` CloseFunc), and close the `*sql.DB`.
- In tests, `t.Cleanup(func() { _ = c.Close() })` immediately after `New`, so cleanup runs before `t.TempDir` removal (cleanups fire LIFO — register close after TempDir is created, which the pattern does naturally).
- Never leave transactions open; roll back before returning. If a helper starts a transaction, it must `defer tx.Rollback()`.
- Do not assert successful file deletion immediately after `Close` — the Go testing package retries sharing violations, but relying on that is fragile; the correct fix is closing handles deterministically.
- Keep each test's database in its own `t.TempDir()` (never a shared path), including WAL sidecars.
- Do not test Unix permission bits on Windows (`os.WriteFile` 0755 becomes 0666 there — already documented in the repo's AGENTS.md cross-platform notes).
- Expect antivirus-induced `SQLITE_BUSY` in the wild as a reason the retry helper (Pitfall 4) is not optional on Windows.

**Warning signs:** CI failing only in the windows-latest leg with `TempDir RemoveAll cleanup: ... used by another process`; `.db-wal`/`.db-shm` files surviving tests; intermittent `SQLITE_BUSY` on Windows runners only.

**Phase to address:** P4 — CI/test hardening, but the `Close` contract is P1 and the cleanup pattern must be used from the first test.

---

### Pitfall 7: The database file only grows — WAL growth and freelist growth are silent

**What goes wrong:** The milestone's TTL design (lazy delete on read + sweep of expired rows on open) deletes rows forever, but two growth mechanisms are invisible until they hurt: (a) with default `auto_vacuum=NONE`, deleted pages go to a freelist and the **file never shrinks** — it stabilizes at a high-water mark set by peak key churn (a real report: 337 MB file, 88% freelist); (b) under sustained writes with overlapping readers, WAL checkpoint starvation lets the `-wal` file **grow without bound** (measured growth of ~12 MB/s in a lab reproduction), and checkpointing does not truncate the WAL unless `journal_size_limit` is set. A cache that is "correct" can still fill a user's disk.

**Why it happens:** SQLite deletes are logical, not physical; `wal_autocheckpoint=1000` only *schedules* a passive checkpoint and caps nothing. `auto_vacuum` must be chosen **before any table exists** (in WAL mode it can only be changed with a full `VACUUM`), so the decision is a schema-creation-time decision, not something to retrofit. Full `VACUUM` needs an exclusive lock, up to 2× the database size in free disk, and cannot coexist with other processes — hostile to a multi-process cache.

**How to avoid (and document):**
- Decide now: set `auto_vacuum=INCREMENTAL` when creating the database (before the first table; the validated `_auto_vacuum=INCREMENTAL` shorthand applies before other pragmas). Rails took exactly this decision for its SQLite cache/queue databases. Then have the sweep occasionally issue a **bounded** `PRAGMA incremental_vacuum(N)` (e.g., a few hundred pages) so space is returned gradually without a second file copy and without an exclusive lock.
  - Trade-off: pointer-map pages and relocation writes add ~0.1% storage and some write cost. For a cache with TTL churn this is usually worth it; if not, keep `NONE` and document the high-water-mark behavior explicitly.
  - If `NONE` is chosen, never retrofit without an offline `VACUUM`; changing modes later is inert until a `VACUUM` runs.
- Set `PRAGMA journal_size_limit` (e.g., 4–8 MB) so the `-wal` file is truncated on the next rewind after a checkpoint. It does **not** prevent starvation; it caps the disk cost.
- Keep read transactions short: `database/sql` closes `Rows` when fully consumed, so always fully scan/close `Rows` (MGet must not leave a query half-read), and never hold a transaction across user code.
- Never run a full `VACUUM` automatically from the provider — in multi-process use it will fail or block everyone. If a maintenance hook is desired, expose it as an explicit, documented, caller-invoked operation and skip it when other connections exist.
- Document the expected steady state: "file size stabilizes near the peak working set; WAL stays under `journal_size_limit` once writes pause; deleting the cache directory resets it."

**Warning signs:** `PRAGMA freelist_count` high relative to `page_count`; database file ⟫ live rows; `-wal` larger than a few MB after writes stop; long-lived readers preventing checkpoints; CI temp dirs filling up.

**Phase to address:** P2 (sweep) and P4 (documented growth guidance). The `auto_vacuum` decision, however, is made in P1 because it must precede table creation.

---

## Moderate Pitfalls

### Pitfall 8: Error mapping — never turn a database error into `ErrMiss`, and decide `ErrClosed`

**What goes wrong:** `Get` uses `QueryRow(...).Scan(...)`; the implementation maps *any* error to `cache.ErrMiss`, so a corrupted database, a permission failure, or a `SQLITE_BUSY` surfaces to callers as "key not found" — silently defeating the cache. The reverse mistake: returning `sql.ErrNoRows` unwrapped, so `errors.Is(err, cache.ErrMiss)` fails and consumers' miss handling breaks. A third variant: after `Close`, operations return raw `sql.ErrConnDone` with no `cache.ErrClosed` mapping (the sentinel exists in `cache/errors.go` but no provider currently returns it).

**Why it happens:** `ErrMiss` is the common path and gets tested; the error taxonomy is an afterthought. `database/sql` returns `sql.ErrNoRows` from `Scan`, not a sentinel.

**How to avoid:**
- Map **only** `errors.Is(err, sql.ErrNoRows)` to a wrapped `cache.ErrMiss` (`fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)` — match the postgres provider's pattern; tests use `require.ErrorIs`).
- Lazy expiry must be distinguishable: fetch the row, compare `expires_at` in Go (or filter in SQL), delete the expired row best-effort, and return `ErrMiss`. Never conflate "expired" with "query failed".
- Default-TTL semantics must match `mem`/`postgres` exactly (see Pitfall 15).
- Decide closed-cache behavior explicitly: validate against existing provider behavior — `mem` does not return `ErrClosed` today, so returning it from sqlite would be a behavioral divergence. If the provider tracks a closed flag (recommended for the sweeper), document whether operations after `Close` return `ErrClosed` or the driver's error, and add tests either way. Do not let `sql.ErrConnDone` leak as "miss".
- Add a test asserting that a corrupted/unavailable DB does **not** produce `ErrMiss`.

**Warning signs:** `errors.Is(err, cache.ErrMiss)` true for non-miss failures; `sql.ErrNoRows` visible in caller-facing errors; provider error text missing the `cache/sqlite:` prefix; no test for a broken DB.

**Phase to address:** P2 — error taxonomy, with tests.

---

### Pitfall 9: DSN construction from a user path — escaping and SQL injection

**What goes wrong:** The DSN is assembled with string concatenation (`"file:" + path + "?_pragma=..."`). A path containing `?`, `&`, `#`, a space, or `%` is silently misparsed (Windows paths with backslashes and UNC paths are especially exposed). Worse, modernc's `_pragma` values are **executed verbatim as SQL text**, not validated: the driver docs explicitly warn that anything after a `;` also runs (`_pragma=foreign_keys(1);ATTACH 'x.db' AS x`), so any user-controlled fragment that reaches the DSN carries SQL authority and file-path authority. While this milestone controls its own pragma strings, the *cache name* feeds the default file path, and the explicit path is user input.

**Why it happens:** Every blog example builds DSNs by concatenation because it works for simple paths. The injection surface only appears when values are interpolated rather than fixed.

**How to avoid:**
- Construct the DSN with `net/url.Values` (or `url.URL`) and `filepath.ToSlash`; percent-encode the path component. Never concatenate `?`/`&`/`#`-bearing values.
- Treat all pragma values as compile-time constants. Never interpolate caller-provided values into `_pragma`.
- If the driver version supports it, enable `StrictPragmas(true)` as defense in depth (rejects multi-statement `_pragma` values).
- Sanitize the cache **name** before deriving the default file path: reject or replace path separators and `..` (path traversal — a name like `../../.ssh/authorized_keys` must not escape the cache directory); prefer an explicit allow-list (`[A-Za-z0-9._-]`) and append `.db` yourself.
- Validate the resolved path with `filepath.Clean` and, for the default mode, verify it is still under `os.UserCacheDir()` after joining.

**Warning signs:** DSN built with `+`; names with `/` producing nested paths; tests only using alphanumeric temp paths (this is exactly why such bugs ship); missing test for `?`/`#`/space in the path.

**Phase to address:** P1 — DSN builder + name sanitization, with adversarial-path tests.

---

### Pitfall 10: Default location failures — `os.UserCacheDir` errors, missing directories, silent path surprises

**What goes wrong:** `os.UserCacheDir()` fails when it cannot determine a directory (notably `HOME`/`XDG_CACHE_HOME` unset on Linux — containers, CI, some daemons). The returned directory may not exist yet, so opening the DB fails unless the provider creates it. Symlink/network redirection of the cache directory silently changes the storage class (ties into Pitfall 3). And callers cannot discover which file the cache actually uses.

**Why it happens:** The happy path (developer laptop) always works; the failure modes appear in containers and enterprise fleets.

**How to avoid:**
- Treat `os.UserCacheDir()` failure as a construction error (`New` returns `(cache.BatchCache, error)` like postgres), not a panic.
- `os.MkdirAll(dir, 0o700)` before opening the DB; handle the "path exists but is a file" error clearly.
- Keep `:memory:` (empty path), default (`UserCacheDir` + package/cache-name), and explicit path as three distinct, documented modes; the zero value must be deterministic.
- Consider exposing the resolved path (e.g., a `Path()` method or returned `Options`) so users and bug reports can answer "where is my cache?" without guessing.
- On macOS, `UserCacheDir` is `~/Library/Caches`; on Windows `%LocalAppData%`; on Linux `$XDG_CACHE_HOME` or `~/.cache` — document that OS cleanups may delete these files (cache semantics allow it).

**Warning signs:** `os.UserCacheDir: $HOME is not defined` in CI/containers; test failures when `HOME` is unset; DB file created in unexpected directories; no way to discover the path.

**Phase to address:** P1 — location modes.

---

### Pitfall 11: Batch operations — SQLite variable limits, transaction shape, and best-effort semantics

**What goes wrong:** `MGet(keys...)` with thousands of keys builds one `IN (?,?,...)` query and dies with `too many SQL variables` (`SQLITE_MAX_VARIABLE_NUMBER` is 32,766 in modern SQLite, 999 in older builds). `MSet`/`MDel` run in a single deferred transaction that reads before writing, reintroducing `SQLITE_BUSY_SNAPSHOT`. Error semantics diverge from the contract established in v1.6: `MGet` must return only found keys (silently absent), `MDel` must be idempotent, `MSet` must use one TTL for all items.

**Why it happens:** Existing providers (postgres `SendBatch`, redis pipelines) hide the parameter-count constraint; SQLite is the first provider where a naive `IN` list hits a hard limit. Batch code is also the first place a multi-statement transaction appears.

**How to avoid:**
- Chunk `MGet`/`MSet`/`MDel` at a conservative size (e.g., 500–900 keys per statement/transaction) — safe across SQLite versions and keeps transaction size bounded.
- Implement `MGet` with `SELECT key, value FROM cache_entries WHERE cache_key IN (...) AND (expires_at IS NULL OR expires_at > ?)`; map rows back to the original `K` values (remember `fmt.Sprint` key conversion — Pitfall 14), and include only found keys.
- For `MSet`, either loop upserts in autocommit (simplest, contention-resilient) or use a single `BEGIN IMMEDIATE` transaction per chunk; do not use a deferred transaction that reads first. For `MDel`, a single bounded `DELETE ... WHERE key IN (...)` per chunk is idempotent by nature.
- Match the provider error philosophy: postgres `MGet` skips per-key errors (best-effort, D-06) while `MSet` returns `errors.Join` of accumulated errors; choose and document the sqlite equivalents (single-statement batch means one error per chunk — wrap with the provider prefix).
- Always fully consume `*sql.Rows` (and close) before returning, or an open cursor pins the read snapshot and starves checkpoints (Pitfall 7).

**Warning signs:** `too many SQL variables` in batch tests; batch tests using three keys only; a batch transaction that starts with a `SELECT`; `MGet` results containing missing keys; partially-applied `MSet` without documentation.

**Phase to address:** P3 — batch operations.

---

### Pitfall 12: Driver dependency, toolchain, and CI fallout

**What goes wrong:** Choosing the CGO driver (`mattn/go-sqlite3`) requires `CGO_ENABLED=1` plus a working C compiler on every CI leg (Linux, macOS, Windows) and complicates cross-compilation; the repo's pure-Go posture and `go test ./...` matrix make that a poor fit. Modernc is pure Go, cross-compiles trivially, and works with `CGO_ENABLED=0`, but it pulls a large dependency tree (`modernc.org/libc`, `modernc.org/memory`, `modernc.org/mathutil`, `modernc.org/fileutil`, `github.com/google/pprof` as a direct dependency) — increasing `go mod tidy` churn, build time, and `govulncheck` surface. Version pinning matters: driver behavior changed across releases (DSN shorthand keys, prepared-statement re-preparation fixed around v1.46, context-interrupt fixes in 2025/2026).

**Why it happens:** Driver choice is made once at milestone start; its CI/dependency consequences surface later in `vulncheck`/coverage/report jobs.

**How to avoid:**
- Pin `modernc.org/sqlite` to a specific recent version (research-time current: **v1.60.1**, published 2026-09-29, requires Go ≥1.26.0 — compatible with the repo's Go 1.26.4). Record the pin as a milestone decision.
- Run `go mod tidy` and inspect the diff; the indirect dependency list will grow substantially. Confirm the repo's "minimal external dependencies" posture is consciously traded for a zero-CGO build (pure-Go wins for this library).
- Run `govulncheck` early (the repo has a `make quality-report` / tool dependency) rather than discovering a transitive advisory in the tag pipeline.
- If `-race` is ever added to CI, note that modernc supports it but the race detector itself requires CGO toolchains; keep that separate from the driver choice.
- Context-cancellation poisoning is a real modernc risk: issues (cznic/sqlite #198, #241) show that cancelling a query can leave a pooled connection returning `interrupted (9)` for later operations, occasionally even for unrelated queries. Pin a version that includes the interrupt-race fixes, keep the retry helper's error classification able to recognize `interrupted`, and consider treating `interrupted` as retryable rather than fatal.
- Avoid `ncruces/go-sqlite3` for this milestone unless a spike proves otherwise: it had a Windows-only WAL corruption bug with pooled concurrent writers (fixed later), and it omits shared cache — an extra risk on the exact OS that already causes trouble.

**Warning signs:** `go.mod` growing without review; `govulncheck` failing on a transitive advisory; flaky `interrupted (9)` or `SQLITE_PROTOCOL` errors under load; unexplained compile-time increases in the coverage job.

**Phase to address:** P1 — driver pin decision (record in PROJECT.md Key Decisions); P4 — quality-report check.

---

### Pitfall 13: Coverage gates — `cache/sqlite` gets no E2E override, but has hard-to-cover paths

**What goes wrong:** `.testcoverage-quick.yml` zeroes the thresholds for `cache/memcache`, `cache/postgres`, `cache/redis`, `cache/valkey`, and other packages because they are exercised through Docker E2E tests. A new `cache/sqlite` package will **not** get such an override — it is embedded and must meet the full thresholds (package ≥80%, file ≥70%, total ≥75%). Provider code traditionally contains un-coverable branches (driver open failures, OS-specific behavior, background sweep errors), and multi-process tests implemented as subprocess helpers **do not contribute coverage** to the parent `go test -cover` run unless GOCOVERDIR instrumentation is wired up. The result: `make coverage-quick` fails at commit time and the author starts deleting error handling or adding threshold overrides instead of tests.

**Why it happens:** Coverage gates are enforced locally per commit (AGENTS.md: "Do not commit if it fails"), while E2E providers were already excused. Embedded providers have no such excuse, and multi-process verification is naturally subprocess-based.

**How to avoid:**
- Design for testability on day one: inject the DSN/path (tests use `t.TempDir()`), keep filesystem error paths reachable (nonexistent parent that is actually a file, directory path passed as DB path, unwritable location skipped on Windows/root), and keep OS-conditional code out of the package entirely (use stdlib).
- Cover the error taxonomy with tests rather than `//nolint`/exclusions: bad DSN, open failure, marshal failure, `sql.ErrNoRows`, closed-cache behavior, sweep error (close the DB handle from under the sweeper or use a canceled context), context cancellation.
- Keep the **unit** suite (no build tags, no Docker) the coverage contributor. Put the two-process race/spike test in a clearly separated helper; do not rely on it for coverage. If multi-process testing becomes permanent, either run it under `go test -tags=e2e` (like `cache_e2e_test.go`) or use Go's coverage-in-child instrumentation deliberately.
- Re-check both configs: `make coverage-quick` (unit-only) and the merged `make coverage` (E2E + unit) must both pass. SQLite tests should pass in both.
- Watch total coverage: adding a dependency-heavy package with moderate coverage can drag the repo-wide total toward the 75% floor even if the package itself passes.
- Flaky `busy` tests are a coverage risk (retried CI jobs, `-count=1`); make concurrency tests deterministic (barriers, generous timeouts, no assertions that `SQLITE_BUSY` is observed).

**Warning signs:** considering a `cache/sqlite` override in `.testcoverage-quick.yml`; `//nolint:exhaustruct`-style suppressions around error returns; subprocess test helpers claimed as coverage; coverage job time ballooning from modernc compilation.

**Phase to address:** P4 — CI/coverage hardening, with testability review during P1 design.

---

## Minor Pitfalls

### Pitfall 14: Key/value parity drift (`fmt.Sprint` keys, JSON values)

**What goes wrong:** The established provider contract serializes keys with `fmt.Sprint(key)` and values with `encoding/json`. Drift breaks compatibility: storing raw keys, using a different key type coercion, or handling `[]byte` values differently means the same logical key maps to different rows across providers and breaks the shared e2e conformance expectations. Edge cases: empty-string key (`""`), `nil` pointer keys (`"<nil>"`), integer keys (`"1"`), and keys whose string forms collide across types (int `1` vs string `"1"`) — the latter is an accepted cross-provider property, but it must be deliberate and tested.

**How to avoid:** Reuse the exact `fmt.Sprint(key)` + `json.Marshal`/`Unmarshal` pattern from `cache/postgres/postgres.go`; test empty string, unicode, spaces, JSON-hostile values, and non-JSON-marshalable values (channels/functions must return an error, not store garbage). Store keys in a `TEXT PRIMARY KEY` column.

**Warning signs:** tests that only use `"key"`/`"value"`; a `Set` that succeeds for a channel value; different key normalization from postgres.

**Phase to address:** P2 — CRUD.

---

### Pitfall 15: TTL semantics drift (zero/negative TTL, default TTL, expiry encoding)

**What goes wrong:** `mem` and `postgres` implement a specific resolution order in `resolveTTL`: explicit `ttl > 0` wins; otherwise the provider default applies; a non-positive explicit TTL means **never expires**. A sqlite implementation that "obviously" treats `ttl=0` as "use default" or stores expiry in local time will diverge. Storage encoding is also a trap: storing RFC3339 text with timezone or SQLite `datetime()` strings invites comparison bugs; integers are unambiguous.

**How to avoid:** Copy the `resolveTTL` semantics exactly and test the matrix (`no ttl`, `ttl>0`, `ttl==0`, `ttl<0`, default set/unset). Store `expires_at` as an INTEGER Unix nanoseconds (or milliseconds) column, `NULL` = no expiry; compare against `time.Now().UnixNano()` passed as a parameter, not `datetime('now')`, so tests are deterministic and clock handling is one place. Add a partial index on `expires_at` (mirroring `cache/postgres/schema.go`). Delete expired rows lazily on `Get`; on `MGet`, filter them out and optionally delete best-effort.

**Warning signs:** an expiry helper named differently from `resolveTTL`; TEXT dates; tests using `time.Sleep` for expiry (prefer injecting a clock into the row or a tiny TTL with a generous margin — existing e2e uses `2s` TTL + `3s` sleep, acceptable at e2e level only).

**Phase to address:** P2 — TTL core.

---

### Pitfall 16: Not registering the provider in the shared conformance/benchmark surfaces

**What goes wrong:** `cache/cache_e2e_test.go` runs a parametrized provider contract (`providerCase`) over mem/redis/valkey/memcache/postgres, and `cache/bench_test.go` benchmarks all providers. If `cache/sqlite` is not added, the provider silently lacks the shared contract checks (`set_and_get`, `get_miss`, TTL expiry, batch semantics, close) and the benchmark comparison. The milestone's target explicitly requires the `c.BatchCache` surface; conformance is how that is proven.

**How to avoid:** Add a non-Docker `providerCase` for sqlite to the e2e suite (it needs no container; it can live alongside `TestCacheE2E_Mem`) and a benchmark entry. Run the e2e suite (`make test-e2e` needs Docker on this machine, but the sqlite case should also be runnable under `-tags=e2e` without Docker). Keep `Close()` in each case so temp files are released.

**Warning signs:** `cache/sqlite` absent from `cache_e2e_test.go` and `bench_test.go`; provider tested only by bespoke tests that duplicate the shared ones.

**Phase to address:** P4 — test integration (can be added as soon as the constructor is stable).

---

### Pitfall 17: WAL sidecar files surprise users and tools

**What goes wrong:** In WAL mode the database is three files while open (`db`, `db-wal`, `db-shm`). After the last connection closes cleanly, SQLite checkpoints and **deletes** the `-wal`/`-shm` files — so a user who inspects the directory before/after `Close` sees different files. Users may try to copy the DB while open (getting an inconsistent snapshot without the WAL), or "clean up" by deleting the sidecars (which can corrupt committed-but-not-checkpointed transactions). A crashed process leaves the sidecars behind; the next open recovers automatically — unless someone deleted them.

**How to avoid:** Document the open-file set and the last-close cleanup in `doc.go`; state explicitly that `-wal`/`-shm` must never be deleted manually while any process has the DB open (the correct recovery is opening the DB and letting SQLite checkpoint, or `PRAGMA wal_checkpoint(TRUNCATE)` with no other connections). Do not treat the presence of sidecars as an error. Do not add cleanup code that removes them — only `Close` (via SQLite) should.

**Warning signs:** users filing "extra files appeared next to my cache"; provider code attempting `os.Remove` on sidecars; backup logic copying only `*.db`.

**Phase to address:** P4 — docs.

---

## Technical Debt Patterns

Shortcuts that look reasonable now and cost later.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `SetMaxOpenConns(1)` for **all** modes (not just `:memory:`) to dodge `SQLITE_BUSY` entirely | Trivial correctness; no retry policy needed in-process; data can never be split across conns | Serializes all reads behind one connection; a slow op blocks everything; cross-process `SQLITE_BUSY` still possible; hides the need for the retry helper | Acceptable for v1.8 MVP **only if** documented as a deliberate limitation and a follow-up (separate read pool) is scheduled. Not acceptable to then claim WAL read concurrency |
| `_pragma` DSN assembled by string concatenation | Matches every blog example; works on simple paths | Escaping bugs for `?& #%` and spaces; potential SQL authority if a value is ever caller-supplied | Never for user-influenced values; acceptable only for fully constant DSNs with a hard-coded path |
| Skipping `PRAGMA journal_mode` read-back | One less query and error path | WAL silently off on unsupported filesystems; multi-process promises become false | Never |
| Deferring `auto_vacuum` "until growth becomes a problem" | Avoids a creation-time decision and pointer-map overhead | Mode change later is inert without a full offline `VACUUM` (exclusive lock, 2× disk); users discover multi-GB cache files first | Acceptable only with explicit docs stating the file stabilizes at the high-water mark and will not shrink |
| Relying on a retry loop as the only `SQLITE_BUSY` strategy | Absorbs transient contention | Does not fix deferred-upgrade (`_txlock`), long transactions, or checkpoint starvation; hides design problems; latency tails | Retry is a **backstop** on top of autocommit/`IMMEDIATE` + short transactions, never the primary mechanism |
| Using a deferred transaction that reads before writing | `database/sql` default; familiar | `SQLITE_BUSY_SNAPSHOT` (517) bypasses `busy_timeout`; flaky under any concurrency | Never for write transactions |
| Passing a caller's tainted path straight into the DSN | Fewer validation branches | Escaping/misparse bugs; path traversal via cache name; possible PRAGMA injection | Never |

---

## Integration Gotchas

Common mistakes when wiring SQLite into this specific codebase and toolchain.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `database/sql` pool | Assuming one connection; setting pragmas once with `db.Exec` | Per-connection pragmas via DSN (or connection hook); pin-pool verification test; `:memory:` requires `MaxOpenConns(1)` + `MaxIdleConns(1)` |
| `cache.BatchCache` (`NewConcreteCache`) | Implementing only `cacher` CRUD; forgetting `MGetFunc/`MSetFunc/MDelFunc` are part of the interface | Implement all seven `cacher` methods (see `cache/concrete_cache.go`); use a compile-time assertion `var _ cache.BatchCache[string, string] = ...` in tests |
| `cache.ErrMiss` contract | Returning raw `sql.ErrNoRows` or mapping every error to miss | Wrap `ErrMiss` with the `cache/sqlite:` prefix; `require.ErrorIs` tests copy the postgres provider pattern |
| TTL resolution | New `resolveTTL` that disagrees with mem/postgres | Copy `resolveTTL` semantics verbatim (explicit ttl>0 > default > never); test the full matrix |
| `Close()` lifecycle | Non-idempotent close; sweeper leak; pending transaction blocks close on Windows | Stop sweeper exactly once (postgres CloseFunc pattern), close `*sql.DB`, `t.Cleanup(close)` in every test |
| E2E/bench suites | Provider not added to `providerCase`/benchmark registries | Register a non-Docker provider case and benchmark entry |
| `go.yml` matrix | Code green on Linux/macOS, broken on Windows | Per-test temp dirs, deterministic close, no permission assertions on Windows, retry helper for AV-induced `BUSY` |
| Coverage config | Adding a `cache/sqlite` threshold override like the Docker providers | No override: unit-test the provider to ≥80% package / ≥70% file; keep subprocess tests out of the coverage calculation |
| `govulncheck`/`go mod tidy` | Vendoring a large pure-Go dependency without review | Pin v1.60.1+, run `go mod tidy` + `govulncheck` early, record the dependency decision in PROJECT.md |
| `os.UserCacheDir` | Assuming `HOME` is set and the directory exists | Handle the error, `MkdirAll(0o700)`, expose/inspect the resolved path |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Unbounded connection pool + `journal_mode(WAL)` in DSN | New pooled connections contend on the WAL switch under load; intermittent open-time `SQLITE_BUSY` | Cap the pool (small fixed number, or 1 for memory); set WAL once + verify; keep only per-conn pragmas in DSN | As soon as pool grows concurrently with a cross-process writer |
| `synchronous=FULL` (default pairing concerns) on every commit | Commit latency dominated by fsync; cache slower than expected | Set `synchronous=NORMAL` in WAL mode (documented-safe trade-off: may lose the last transactions on power loss, no corruption); acceptable for a cache | At any write rate; laptop/CI disk-bound |
| Re-preparing every statement per call | CPU-bound Get/Set slower than the other providers; profiles show prepare overhead | modernc fixed re-preparation around v1.46 — pin ≥1.46 (v1.60.1); keep pool stable (`MaxIdleConns`) so prepared statements can be reused; avoid opening/closing DB per operation | Benchmarks vs mem/postgres; batch-heavy workloads |
| SQLite JSON encode/decode per Get/Set | CPU floor higher than mem; acceptable but measurable | Expected cost; document it; consider `json.RawMessage` fast paths only if benchmarks justify | Large values, high QPS |
| MGet one query per key | N round trips per batch | Single `IN` query chunked below the variable limit (Pitfall 11) | Batches > ~100 keys |
| Full `VACUUM`/unbounded `incremental_vacuum` | Writers blocked for seconds; failures with other connections; 2× disk | Bounded `incremental_vacuum(N)` in the sweep; never auto-`VACUUM` in multi-process | Any multi-process deployment; large freelists |
| Long-lived read snapshot (unclosed `Rows`, open tx) | `-wal` grows without bound; `wal_checkpoint(PASSIVE)` shows `checkpointed < log` | Always fully consume/close `Rows`; no transactions across user code; `journal_size_limit` as a cap | Under sustained write load with an overlapping reader |
| `:memory:` with `MaxOpenConns(1)` under heavy parallel tests | Tests serialize; suite slows | Acceptable; use file-backed temp DBs for concurrency tests that need parallelism | Large test suites only |

---

## Security Mistakes

Beyond generic web security, specific to an embedded cache DB on user machines.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Interpolating caller input into `_pragma` DSN values | SQL injection via a DSN that is executed as SQL text (`; ATTACH '...'`); arbitrary file access | Pragma values are compile-time constants; `StrictPragmas(true)` when available; never build `_pragma` from user data |
| Cache name used as a raw path segment | Path traversal (`../../...`) writes a `.db` outside the cache directory; on multi-user systems files may be picked up by other tooling | Allow-list characters, reject `.`/`..`/separators, join with `filepath.Clean` and verify containment under `os.UserCacheDir()` |
| World-readable DB / directory | Cached values may contain secrets/tokens; other local users read them | `MkdirAll(dir, 0o700)`, create DB with restrictive mode where the OS supports it (skip assertions on Windows); document that the default location is per-user |
| Manually deleting `-wal`/`-shm` to "fix a lock" | Losing committed transactions → corruption, especially with other processes attached | Never delete sidecars while any process has the DB open; recover by opening the DB (checkpoint) or `wal_checkpoint(TRUNCATE)` when alone |
| DSN/user path on a network or synced folder treated as safe | Cross-host WAL is corruption-prone; sync engines rewrite files out from under locks | Read-back `journal_mode` check; documented prohibition; default to local `UserCacheDir` |
| Logging full DSNs/paths or values at info level | Path disclosure; cached values in logs | Log at debug only; never log values; error messages carry the path only where already caller-known |

---

## "Looks Done But Isn't" Checklist

- [ ] **WAL verified, not assumed:** `PRAGMA journal_mode` read-back equals `wal` on open, with a bounded retry for the switch. Test a fresh file and a pre-existing non-WAL file.
- [ ] **`busy_timeout` on every connection:** pin N pooled connections concurrently and assert `PRAGMA busy_timeout` and `synchronous` on each — not just the first.
- [ ] **`:memory:` mode pinned:** `MaxOpenConns(1)` + `MaxIdleConns(1)`; a test proves Set→Get works with concurrent goroutines and that `db.Stats().OpenConnections == 1`.
- [ ] **Cold-start race tested:** 4+ goroutines (ideally 2 processes) opening the same fresh path simultaneously; all succeed; schema/version converge; pre-fix code demonstrably fails the test.
- [ ] **Two-process `SQLITE_BUSY` behavior spiked (questions.md #4):** observable behavior documented; retry helper absorbs the residual; outcomes recorded in the phase notes.
- [ ] **TTL semantics match mem/postgres:** explicit ttl>0, default TTL, ttl=0/negative (never expire) all tested; expired `Get` returns wrapped `ErrMiss` and deletes the row best-effort.
- [ ] **Sweep-on-open bounded and safe:** expired rows removed; failure logged and skipped (not fatal); no full-table lock held.
- [ ] **Error taxonomy:** `sql.ErrNoRows` → `ErrMiss` only; DB failures are not misses; closed-cache behavior decided and tested; provider prefix `cache/sqlite:` on wrapped errors.
- [ ] **Batch limits:** `MGet/MSet/MDel` tested at >1,000 keys; chunking verified; only found keys returned; MDel idempotent; MSet single TTL.
- [ ] **Close is idempotent, sweeper stopped, no leaked handles:** tests pass on Windows temp-dir cleanup; `Close` twice returns without panic.
- [ ] **Growth guidance:** `auto_vacuum` decision recorded (INCREMENTAL vs NONE); `journal_size_limit` set; docs explain high-water-mark and sidecar lifecycle.
- [ ] **Windows matrix green:** full `go test ./...` passes on windows-latest (no permission assertions, unique temp dirs, deterministic close).
- [ ] **Coverage green with no override:** `make coverage-quick` passes with `cache/sqlite` meeting 80/70; no new exclusions in `.testcoverage-quick.yml`.
- [ ] **Quality gates:** `go mod tidy` reviewed; `govulncheck` clean; lint clean; benchmark/e2e registrations added.
- [ ] **Docs:** `doc.go`, package example, and the network-filesystem warning ship with the provider.
- [ ] **No committed artifacts:** test DBs/WAL files are only in temp dirs; nothing new needs `.gitignore`.

---

## Recovery Strategies

When a pitfall slips through despite prevention.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| `:memory:` split databases (P1) | LOW | Set `MaxOpenConns(1)`/`MaxIdleConns(1)`; add the pin test; no data migration needed (memory data is ephemeral by definition) |
| Pragma misconfiguration shipped (P2/3/4) | LOW–MEDIUM | Add read-back verification + per-connection test; hot fix release; for WAL-fallback cases check whether the DB ran in rollback mode (data itself is intact) |
| WAL on network share / corruption suspicion (P3) | HIGH | Stop all writers; copy the three files to local disk **while no process is attached**; run `PRAGMA integrity_check`; if ok, move the cache to local storage and document; if corrupt, delete the cache (it is disposable) and rebuild |
| Deferred-upgrade `SQLITE_BUSY_SNAPSHOT` flakes (P4) | MEDIUM | Audit every `BeginTx`: switch write transactions to `_txlock=immediate`; add retry backstop; add a contention regression test |
| Cold-start migration race (P5) | MEDIUM | Wrap bootstrap in `BEGIN IMMEDIATE` + re-read version; ship retry; add concurrent-open test; affected users' DBs converge on next open (no manual repair if DDL was idempotent) |
| Windows temp-dir CI failures (P6) | LOW | Add `t.Cleanup(close)`; ensure transactions rolled back; rerun; no production impact |
| Unbounded DB growth (P7) | MEDIUM–HIGH | If created without `auto_vacuum`: choose `NONE` + document, or schedule an offline `VACUUM` (exclusive, 2× disk) to switch to `INCREMENTAL`; if WAL growth: set `journal_size_limit`, remove long readers, add bounded `incremental_vacuum` to sweep |
| Error taxonomy leak (P8) | LOW–MEDIUM | Fix mapping; add contract tests; callers that consumed "miss" for errors need a patch release note |
| DSN injection/traversal (P9) | HIGH (security) | Validate/sanitize names, switch to `url.Values` + `filepath.ToSlash`, adopt `StrictPragmas(true)`; audit for any other interpolated DSN values; advisory if a released version accepted tainted names |
| Coverage gate failing at commit (P13) | LOW | Add the missing error-path tests; do not add a threshold override for an embedded provider |

---

## Pitfall-to-Phase Mapping

Suggested roadmap phases for this milestone; each critical pitfall must have an owning phase and a verification.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1 — `:memory:` pool split | P1 (constructor) | Pin test: concurrent Set/Get in memory mode; `db.Stats().OpenConnections == 1` |
| 2 — Per-connection pragmas / silent ignore | P1 (DSN + open) | Read-back of `journal_mode == wal`; multi-pin test asserting `busy_timeout`/`synchronous` on N connections |
| 3 — Network filesystem + silent WAL fallback | P1 (open check) + P4 (docs) | Test forcing a non-WAL return (e.g., memory/unsupported path) produces the documented behavior; docs mention network/synced folders |
| 4 — `busy_timeout` blind spots + deferred upgrades | P3 | `_txlock`/autocommit audit; retry helper unit tests (primary codes 5/6, 517); two-process spike per questions.md #4 |
| 5 — Cold-start schema race | P1 (schema) | Concurrent-open test: 4 goroutines (and 2 processes) on one fresh path; schema/version converge; test fails against a naive implementation |
| 6 — Windows handles / CI | P1 (Close contract) + P4 (tests) | windows-latest leg green; repeated temp-dir cleanup with no sharing violations; Close idempotence test |
| 7 — DB/WAL growth | P1 (`auto_vacuum` decision) + P2 (sweep) + P4 (docs) | Test: with `INCREMENTAL`, sweep issues bounded `incremental_vacuum`; `journal_size_limit` set; docs state high-water-mark behavior |
| 8 — Error mapping / `ErrClosed` | P2 | `require.ErrorIs(err, cache.ErrMiss)` for misses; non-miss DB failure not `ErrMiss`; closed-cache behavior tested |
| 9 — DSN escaping/injection/traversal | P1 | Adversarial path tests (`?`, `#`, space, `..`, separators in name); DSN built via `url.Values` |
| 10 — UserCacheDir/default location | P1 | Tests with `HOME`/`XDG_CACHE_HOME` unset (construction error, no panic); MkdirAll path created; resolved path discoverable |
| 11 — Batch limits/semantics | P3 | MGet/MSet/MDel at >1,000 keys; only-found-keys; idempotent MDel; chunks documented |
| 12 — Driver dependency/CI/toolchain | P1 (pin) + P4 (quality) | Pinned version in `go.mod`; `go mod tidy` diff reviewed; `govulncheck` clean; context-interrupt behavior accounted for in retry classification |
| 13 — Coverage gates | P4 | `make coverage-quick` passes with no `cache/sqlite` override; merged `make coverage` passes; no flaky retries |
| 14 — Key/value parity | P2 | Edge-key/value table test identical to other providers' expectations |
| 15 — TTL semantics | P2 | Full TTL matrix test; expired `Get` returns `ErrMiss`; row deleted |
| 16 — Conformance registration | P4 | `cache/sqlite` present in `cache_e2e_test.go` provider cases and `bench_test.go` |
| 17 — Sidecar file lifecycle | P4 | `doc.go` documents the three files and never-delete rule |

---

## Verification Gaps (Honest Unknowns)

These were not fully resolved by desk research and should be answered by the two-process spike or during implementation:

1. **Residual `SQLITE_BUSY` probability with the chosen pragma set under two-process contention** — desk research confirms it is possible even with `busy_timeout`; only the spike can quantify it for this provider. (questions.md #4)
2. **Context-cancellation interrupt behavior on the pinned modernc version** — issues #198/#241 show connection poisoning in older releases; confirm the pinned version's behavior under `-race` stress, or classify `interrupted` as retryable.
3. **WAL + `auto_vacuum=INCREMENTAL` interaction under real multi-process churn on Windows NTFS** — semantics are documented, but `incremental_vacuum` write-lock duration and sidecar behavior should be measured before shipping sweep-time vacuum.
4. **Exact modernc DSN shorthand behavior across future minor upgrades** — the shorthand-key set is a newer compatibility feature; the pin and the read-back verification test are the guard, not assumptions from blog posts.
5. **Whether the repo wants `SetMaxOpenConns(1)` for file-backed mode as well** — a design decision trading read concurrency for simplicity; recommendation is to decide consciously in P1 and document it.

---

## Sources

- SQLite official documentation: [Write-Ahead Logging](https://sqlite.org/wal.html) (checkpointing, checkpoint starvation, last-close cleanup, network filesystem constraint, `journal_mode` fallback); [In-Memory Databases](https://sqlite.org/inmemorydb.html) (`:memory:` semantics); [PRAGMA reference](https://sqlite.org/pragma.html) (`auto_vacuum`, `incremental_vacuum`, `journal_size_limit`, `busy_timeout`); [VACUUM](https://sqlite.org/lang_vacuum.html); [Transaction](https://sqlite.org/lang_transaction.html) (DEFERRED/IMMEDIATE, upgrade failures); [How To Corrupt An SQLite Database File](https://sqlite.org/howtocorrupt.html); [SQLite Over a Network, Caveats and Considerations](https://sqlite.org/useovernet.html); [Result codes](https://sqlite.org/rescode.html) and [busy_timeout C API](https://sqlite.org/c3ref/busy_timeout.html) (busy handler not invoked on deadlock risk).
- `modernc.org/sqlite` official package docs (v1.60.1, 2026-09-29): DSN parameters (`_pragma`, `_txlock`, `_time_format`, `_dqs`, `_defensive`, validated mattn-compatible shorthands and their fixed application order), `*sqlite.Error.Code()`, `NewConnector`, `RegisterConnectionHook`, `StrictPragmas`.
- Driver/concurrency field reports (cross-checked): opentalon PR #314 and MaorBril/clauder PR #24 (mattn-style DSN silently ignored by modernc; per-connection pragma traps); trip2g "blaming SQLite for my own bug" (517 `SQLITE_BUSY_SNAPSHOT`, `_txlock=immediate`, pooled-connection pragmas); SQLite forum threads on `BUSY_SNAPSHOT` and network storage; mattn/go-sqlite3 FAQ #204/#511/#1205 (`:memory:` pooling, shared cache); Go issue #50510 / #51442 (`t.TempDir` Windows sharing violations); openzro commit (leaked SQLite handle failing Windows temp cleanup); Rails PR #57076 (`auto_vacuum=incremental` for SQLite cache databases); NetBird discussion #6701 (monotonic file growth, freelist); "Why your SQLite WAL file never shrinks" (checkpoint starvation lab measurements); babs/claude-quota and korinfra commits (concurrent migration races, `BEGIN IMMEDIATE` + re-read, WAL bootstrap retries); cvilsmeier/go-sqlite-bench and lbe/sqlite-read-benchmark (driver performance); ncruces PR #405 (Windows WAL-index corruption in the WASM driver).
- Local repo context: `cache/cache.go`, `cache/errors.go`, `cache/concrete_cache.go`, `cache/mem/*`, `cache/postgres/*`, `cache/cache_e2e_test.go`, `cache/bench_test.go`, `.testcoverage-quick.yml`, `.github/workflows/go.yml`, `go.mod`, `AGENTS.md`.

---

*Pitfalls research for: adding an embedded SQLite cache provider (v1.8) to github.com/guionardo/go*
*Researched: 2026-10-08*


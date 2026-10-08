# Feature Research

**Domain:** Embedded SQLite-backed cache provider (`cache/sqlite`) for the generic Go cache library in `github.com/guionardo/go`
**Researched:** 2026-10-08
**Confidence:** HIGH for engine/driver behavior and interface-parity requirements (official SQLite and modernc.org/sqlite docs, cross-checked); MEDIUM for Go ecosystem TTL/sweep and pool idioms (consensus across 4+ independent implementations)

> Scope note: this milestone adds the **sixth backend** to an existing cache library (v1.6 shipped singleflight GetOrSet dedup and `BatchCache` MGet/MSet/MDel across mem, redis, valkey, memcache, postgres). Everything below is scoped to the NEW backend; the shared `NewConcreteCache` / `cacher[K,V]` / `SingleflightGetOrSet` machinery already exists and is a dependency, not new work.

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `cacher[K,V]` primitive conformance (GetFunc/SetFunc/DeleteFunc/CloseFunc) wrapped by `cache.NewConcreteCache` | Every existing provider plugs into the shared adapter; without it the backend cannot be constructed at all | LOW | Return `cache.BatchCache[K,V]` like the other constructors. GetOrSet/Close/interface plumbing comes free once the four funcs exist |
| `BatchCache` operations: MGet / MSet / MDel | `NewConcreteCache` returns `BatchCache[K,V]` since v1.6; users casting to it will call these on every backend | LOW–MEDIUM | MGet: one chunked `SELECT ... WHERE cache_key IN (?,...)` (or per-key prepared reads in one read tx). MSet: single tx with `INSERT ... ON CONFLICT DO UPDATE`. MDel: single chunked `DELETE ... IN (?,...)`. Chunk to `SQLITE_MAX_VARIABLE_NUMBER` (999 legacy / 32766 modern SQLite) |
| Singleflight `GetOrSet` dedup | All 5 providers delegate through the shared `SingleflightGetOrSet`; inconsistent behavior here would be a regression | LOW | Inherited. Provider only supplies GetFunc/SetFunc; the shared helper does fast-path Get, double-check Get, leader-wins TTL, panic recovery. Do NOT add a second dedup layer |
| Per-key TTL + provider default TTL, same resolution rules as existing providers | Users port code between backends; TTL semantics must not shift | LOW | Mirror `resolveTTL`: `ttl[0] > 0` wins; `ttl[0] == 0` → no expiry; absent → `DefaultTTL`; `DefaultTTL <= 0` → no expiry. Store `expires_at` as absolute unix epoch INTEGER (NULL = no TTL) so expiry survives restarts |
| Expired entries are never returned (treated as `ErrMiss`) | Core cache contract; a stale value that "sometimes" appears is worse than no cache | LOW | Filter in the WHERE clause: `AND (expires_at IS NULL OR expires_at > ?)`, return `fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)`. Research consensus: filter on read, do not delete on read (deleting turns reads into writes and causes lock contention) |
| Expired-row reclamation: sweep on open + optional periodic sweep | Expired rows otherwise accumulate forever on disk; milestone explicitly requires a sweep on open | MEDIUM | `DELETE FROM ... WHERE expires_at IS NOT NULL AND expires_at <= ?`. Index `expires_at` (partial index like postgres `WHERE expires_at IS NOT NULL` is ideal). Sweep on open; periodic is an option (mirror mem/postgres `SweepInterval`) — when absent, startup + read filtering is sufficient |
| Durability: values persist across `Close` and process restarts | This is the entire reason the backend exists; local durable caching for CLIs/small services with no server | LOW | File mode only; absolute expiry timestamps; `Close` does a final checkpoint and SQLite removes `-wal`/`-shm` on clean last-connection close |
| Multi-process safety: WAL journal mode + `busy_timeout` | Milestone requirement and the key differentiator vs bbolt; two CLIs/services sharing a cache file must not corrupt it or hard-fail on contention | MEDIUM | DSN: `?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)` (modernc-native syntax; mattn-style `_journal_mode=WAL` params are fixed-order shorthand and were silently ignored in a real incident). WAL is persistent once set; readers don't block the writer; one writer at a time; all processes must be on the same host (WAL does NOT work on network filesystems). Consider `_txlock=immediate` for write transactions so lock acquisition fails fast at BEGIN rather than mid-transaction |
| Location modes: default `os.UserCacheDir` path via cache name; explicit path override; `:memory:` when path empty | Milestone requirement; matches user expectations that "just works" locally without config | MEDIUM | Default: `os.UserCacheDir()` + app/cache-name subdirectory + `MkdirAll` (0700). Unix `$XDG_CACHE_HOME` or `~/.cache`; Darwin `~/Library/Caches`; Windows `%LocalAppData%`; errors if HOME/AppData is undeterminable. Explicit: accept as given. `:memory:`: pin `SetMaxOpenConns(1)` — a pooled second connection would otherwise see a brand-new empty DB ("no such table"); note WAL is impossible/ignored for in-memory DBs, so don't assert it engaged |
| JSON value serialization + `fmt.Sprint` key parity | Existing external providers all serialize via `encoding/json` and stringify keys via `fmt.Sprint`; swapping backends must not change wire semantics | LOW | `key TEXT PRIMARY KEY` (from `fmt.Sprint(key)`), `value TEXT NOT NULL` (JSON). Marshal errors are hard failures; MGet silently skips missing/undecodable entries (D-03/D-06 parity) |
| Error and context parity (`ErrMiss`, wrapped provider errors, idempotent Delete/Close) | Consumers use `errors.Is(err, cache.ErrMiss)` and call `Close` twice; ctx cancellation must propagate to SQL calls | LOW | All queries via `*Context` methods; wrap with `cache/sqlite: ` prefix; MDel/Delete of missing keys succeed; Close idempotent (guard with stop channel/once) |
| In-process concurrency safety under `-race` | Every other provider is race-tested; SQLite's pool makes this subtler than a mutex | MEDIUM | Bound the pool (`SetMaxOpenConns`); single-connection pool serializes writers and eliminates in-process `SQLITE_BUSY` upgrade races (the cost: reads serialize too, acceptable for a cache — mem uses one RW mutex anyway). Pure-Go driver means every CI OS can run the full suite |
| Idempotent schema setup | Opening an existing cache file must not fail or migrate unexpectedly; opening a fresh file must create everything | LOW | `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS`; fixed schema (no user-supplied table name — one DB file per cache). Keep column set stable for future compatibility |
| Pure-Go, CGO-free, cross-platform | Project constraint: Linux/macOS/Windows CI matrix; `modernc.org/sqlite` is the milestone-mandated driver | LOW | `CGO_ENABLED=0` builds. Pin the `modernc.org/libc` version from the driver's `go.mod` (documented fragile dependency). E2E tests use temp dirs/`:memory:` — no Docker (unlike testcontainers-based providers) |
| `doc.go`, runnable example, tests at project coverage gates | Every package has `doc.go`; gates enforce packages ≥80%, files ≥70%, total ≥75% | LOW | Include TTL, reopen-persistence, and `:memory:` examples in docs |

### Differentiators (Competitive Advantage)

Features that set this backend apart from the other five providers and from embedded alternatives. Not required by the interface, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Zero-infrastructure durable cache | The only provider that persists data with no server, no container, no credentials — ideal for CLIs, desktop tools, small services | LOW (falls out of design) | Position in docs as: "mem but survives restarts; redis/postgres without the infra" |
| `:memory:` mode as a drop-in fast/test double | Same `Cache[K,V]` API as mem but exercises the SQL code path; lets downstream tests avoid Docker entirely | LOW–MEDIUM | Must pin the pool to 1 connection (see table stakes). Also useful as a scratch cache when persistence is unwanted |
| Multi-process shared cache file (WAL) | Multiple CLI invocations or a service + its sidecar can share one cache file safely — bbolt and badger explicitly cannot do this | MEDIUM | Requires WAL + busy_timeout + local disk. Document the same-host restriction and the busy timeout as the contention behavior |
| TTL that survives restarts | Absolute `expires_at` means a cache entry set 5 minutes before exit is still correctly expired after restart; mem cannot express this | LOW | Depends on storing expiry as an absolute timestamp, never a relative duration |
| Transactional batch writes | MSet/MDel in one transaction: fewer fsyncs, faster than per-key loops (the v1.6 benchmark goal) and a batch either lands together or not | MEDIUM | Single tx, prepared statement reuse; `BEGIN IMMEDIATE` semantics avoid mid-batch lock upgrade |
| Storage-lifecycle controls (checkpoint/optimize) | Long-running processes can tune `wal_autocheckpoint`/`synchronous` or trigger a checkpoint/`VACUUM` to reclaim disk | MEDIUM | Optional options. Defaults (autocheckpoint 1000 pages, synchronous NORMAL paired with WAL) are fine for cache data; expose only if cheap |
| Max-entries cap | Bounded file growth for write-heavy/rarely-read caches (mem has `MaxEntries`; postgres does not) | MEDIUM | Enforce on write: delete expired, then trim oldest `expires_at` beyond cap (cameo/rust pattern). Defer unless wanted |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems. Explicitly NOT in this milestone.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Arbitrary SQL query API / ORM behavior | "It's SQLite, let me query it" | Turns a cache backend into a database wrapper; leaks schema, breaks provider-swap abstraction, multiplies test/API surface | Keep the fixed `Cache`/`BatchCache` surface; users needing SQL should use SQLite (or the `database/sql` handle) directly |
| LRU / size-based eviction policies in v1 | Users fear unbounded growth | Adds an access-recency column write on every read (read amplification) and policy complexity; TTL already bounds most caches | TTL + startup/periodic sweep; optional max-entries cap as a bounded fallback; revisit only with evidence |
| Encryption at rest (SQLCipher) | Sensitive cached data | Not available in pure-Go `modernc.org/sqlite`; breaks the CGO-free constraint | Store non-sensitive data, or use OS-level disk encryption; document the gap |
| Network filesystem / multi-host support | "Point it at a shared drive" | WAL requires shared memory (wal-index) — officially does not work over network filesystems and may corrupt; cross-host sharing is the redis/postgres use case | Local disk only; document that shared-network deployments should use redis/valkey/postgres |
| Cross-process invalidation pub/sub | Multi-process cache coherence | No such facility in SQLite; building polling/triggers adds hidden goroutines and semantics the interface cannot express | Expiry via TTL; consumers needing invalidation use Delete (writes are visible cross-process immediately in WAL) |
| Custom codec registry (gob/msgpack) | Smaller/faster serialization | Splits the "swap providers without code changes" guarantee; triples test matrix | `encoding/json` parity with existing providers |
| Delete-on-read of expired rows | "Free the space immediately" | Converts every miss on an expired key into a write (lock churn, WAL growth, `SQLITE_BUSY` risk under multi-process load) | Filter expired rows on read (miss), reclaim in the startup/periodic sweep |
| Separate read/write connection pools (max-1 writer + N readers) | Squeeze read concurrency out of WAL | Two pools double the config/close/locking story; the cache fast path is µs-scale SQL, and mem serializes under one mutex anyway | Single bounded pool first; benchmark, and only split if the v1.8 benchmark suite proves a real regression |
| Retry middleware / auto-retry every `SQLITE_BUSY` beyond `busy_timeout` | "It should never fail" | Retrying past the busy timeout hides pathological lock holders and multiplies latency; SQLite's documented `SQLITE_BUSY` cases (exclusive lock, last-connection cleanup, crash recovery) are not all retry-fixable | `busy_timeout` (e.g. 5000 ms) + `_txlock=immediate` for writes; surface true errors to the caller |
| `file::memory:?cache=shared` as the default `:memory:` DSN | One-liner "fix" for pool sharing | The unnamed shared cache is process-global: two independent caches in one process would silently share one store and collide | Pin `SetMaxOpenConns(1)` on `:memory:`; optionally allow a named `file:<name>?mode=memory&cache=shared` DSN for deliberate sharing |
| Schema migration framework / `user_version` versioning | Future-proofing | YAGNI for a fixed 3-column cache table; migration machinery is weight with no user | Keep schema v1 stable; handle any future change with a targeted, documented upgrade path if ever needed |
| Periodic `VACUUM` / `auto_vacuum` by default | Reclaim file space automatically | `VACUUM` takes an exclusive lock and rewrites the whole file (blocking reads, racing other processes); `auto_vacuum` must be set before first write and changes file layout | Rely on WAL autocheckpoint; offer an explicit `VACUUM`/optimize helper only as P3 if needed |
| Background write-serializer goroutine (serialwrite pattern) | Squeeze more write throughput | Hidden goroutine, queue, shutdown semantics; unnecessary when pool=1 serializes writes and `busy_timeout` handles cross-process contention | Pool pinning + busy timeout; revisit only with benchmark evidence |

## Feature Dependencies

```
Cache surface (Get/Set/Delete/GetOrSet/Close)
    └──requires──> cache.cacher[K,V] + cache.NewConcreteCache   (EXISTS, v1.6)
                        └──requires──> provider GetFunc/SetFunc/DeleteFunc/CloseFunc   (NEW)

BatchCache surface (MGet/MSet/MSet... MDel)
    └──requires──> cache.cacher batch methods                   (EXISTS interface, v1.6)
                        └──requires──> SQL: IN-list SELECT / tx UPSERT / IN-list DELETE   (NEW)

GetOrSet singleflight dedup
    └──requires──> cache.SingleflightGetOrSet                    (EXISTS, v1.6)
                        └──requires──> provider GetFunc/SetFunc exact ErrMiss semantics   (NEW, correctness)

TTL read-miss + persistence
    └──requires──> expires_at INTEGER (absolute) + index         (NEW schema)
                        └──requires──> clock injection for tests (time.Now or NOW())

Startup sweep / periodic sweep
    └──requires──> expires_at index                              (NEW)
                        └──requires──> DELETE ... WHERE expires_at <= now

Multi-process safety
    └──requires──> WAL + busy_timeout DSN pragmas                (NEW)
                        └──requires──> local (non-network) filesystem, same host
                        └──enhances──> single-connection pool choice (avoids in-process upgrade races)

Persistence
    └──requires──> file path mode
                        └──requires──> os.UserCacheDir + MkdirAll (default) OR explicit path

:memory: mode
    └──requires──> SetMaxOpenConns(1) (or named shared-cache DSN)
    └──conflicts──> Persistence (no file)
    └──conflicts──> Multi-process sharing (memory DBs are process-local)

Cross-platform CI
    └──requires──> pure-Go driver modernc.org/sqlite (CGO-free)

Close hygiene
    └──enhances──> Persistence (final checkpoint; -wal/-shm removed on clean close; crash recovery on next open)
```

### Dependency Notes

- **All interface plumbing requires only the four `cacher` funcs:** `GetOrSet`, dedup, panic handling, TTL passthrough, and even `MGet/MSet/MDel` dispatch already exist in `concrete_cache.go` (v1.6). The new code is: SQL schema, `GetFunc/SetFunc/DeleteFunc/CloseFunc`, three batch funcs, constructor + options.
- **`GetFunc` must return the wrapped `cache.ErrMiss`** because `SingleflightGetOrSet.Do` treats any error as a miss and re-computes; returning a raw `sql.ErrNoRows` would work by accident but breach error-parity with the other providers.
- **TTL semantics must reuse the existing `resolveTTL` logic shape** (per-key > default > none) — postgres and mem already implement it identically; SQLite stores the resulting absolute time, not a duration.
- **`:memory:` conflicts with both persistence and multi-process sharing** — it is a separate mode, not a configuration tweak. Its pool must be pinned to 1 connection or the database/sql pool will hand out connections to *different* empty databases (mattn FAQ / SQLite docs: "every :memory: database is distinct").
- **Multi-process safety depends on local storage:** WAL's wal-index is shared memory, so network filesystems are officially unsupported. Constraints section of the docs must say so.
- **`BEGIN IMMEDIATE` / `_txlock=immediate` enhances multi-process write paths:** default `BEGIN` is deferred, so a write lock upgrade can fail with `SQLITE_BUSY` mid-transaction; taking the lock at BEGIN makes contention fail fast/fairly.
- **modernc driver DSN dependency:** all per-connection pragmas must ride in the DSN (`?_pragma=...`) or a connection hook — pragmas set via a single `db.Exec` after open only stick to whichever pooled connection served that Exec.
- **Pin modernc.org/libc exact version** (driver's README calls the dependency fragile); `go.mod` hygiene is a review item.

## MVP Definition

### Launch With (v1.8)

Minimum viable product — the milestone's target feature set.

- [ ] `cacher[K,V]` conformance + `NewConcreteCache` return — the only way the backend is usable
- [ ] Batch `MGet`/`MSet`/`MDel` — `BatchCache` is the return type; missing batch ops would be a visible regression
- [ ] TTL parity + lazy read expiry + startup sweep — the cache semantics users already rely on
- [ ] WAL + `busy_timeout` DSN on every connection — multi-process safety is the milestone's core promise
- [ ] Location modes: default `os.UserCacheDir` via cache name, explicit path, `:memory:` (pool-pinned) — milestone requirement
- [ ] JSON/fmt.Sprint parity, `ErrMiss` wrapping, ctx propagation, idempotent Delete/Close — provider-swap contract
- [ ] Tests (unit + `:memory:` + reopen/persistence + concurrent, `-race`), coverage gates, `doc.go`, example — project Definition of Done

### Add After Validation (v1.x)

- [ ] Periodic sweep interval option (reclaim on long-running processes; mirrors mem/postgres `SweepInterval`) — trigger: users report file growth between restarts
- [ ] Benchmark entries for the SQLite backend in the v1.6 bench suite (compare vs mem and per-key loops) — trigger: performance questions
- [ ] Max-entries cap (bounded file) — trigger: write-heavy caches with rare reads
- [ ] Checkpoint/`wal_autocheckpoint` + explicit optimize/`VACUUM` helper — trigger: long-lived processes hitting WAL growth
- [ ] Multi-process contention E2E test (two OS processes sharing one temp DB) — trigger: CI capability to spawn concurrent processes cross-platform

### Future Consideration (v2+)

- [ ] LRU / size-based eviction — defer until TTL proves insufficient; high read-path cost
- [ ] Encryption at rest — blocked by pure-Go driver; only if CGO becomes acceptable
- [ ] Split read/write pools — defer until benchmarks prove a need
- [ ] Named shared in-memory instances / multi-cache-per-process semantics — defer; niche
- [ ] `PRAGMA integrity_check` recovery helper — defer; document "delete the cache file" as the recovery path

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| `cacher` + `BatchCache` conformance via `NewConcreteCache` | HIGH | LOW | P1 |
| TTL parity (per-key + default, absolute storage) | HIGH | LOW | P1 |
| Lazy read expiry + startup sweep | HIGH | MEDIUM | P1 |
| WAL + busy_timeout (multi-process) | HIGH | MEDIUM | P1 |
| Path modes (default `/ explicit / :memory:`) | HIGH | MEDIUM | P1 |
| Persistence across restarts | HIGH | LOW | P1 |
| JSON / `fmt.Sprint` / error / ctx parity | HIGH | LOW | P1 |
| Tests, coverage, doc.go, example | HIGH (project gates) | LOW | P1 |
| Periodic sweep option | MEDIUM | LOW | P2 |
| Benchmark entries | MEDIUM | LOW | P2 |
| Checkpoint/optimize controls | MEDIUM | MEDIUM | P2 |
| Max-entries cap | LOW–MEDIUM | MEDIUM | P3 |
| LRU / size eviction | LOW | HIGH | P3 |
| Encryption at rest | LOW | HIGH (blocked) | P3 |

**Priority key:**
- P1: Must have for launch (v1.8)
- P2: Should have, add when possible (v1.x)
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | cache/mem | redis / valkey / memcache | cache/postgres | bbolt | badger | **cache/sqlite (ours)** |
|---------|-----------|---------------------------|----------------|-------|--------|--------------------------|
| Persistence | No | Yes (server) | Yes (server) | Yes | Yes | **Yes (file)** |
| Multi-process access | No | Yes (network) | Yes (network) | No — exclusive file lock (hangs on second open) | No — directory lock | **Yes — WAL, same host only** |
| TTL | Yes (sweeper) | Yes (native) | Yes (`expires_at` + sweeper) | No built-in | Yes (native) | **Yes (`expires_at` + open/periodic sweep)** |
| Batch MGet/MSet/MDel | Single lock | Pipelines / GetMulti | SendBatch | n/a (KV API) | n/a (KV API) | **Single tx / IN-lists** |
| GetOrSet singleflight dedup | Yes (shared) | Yes (shared) | Yes (shared) | Would need building | Would need building | **Yes (shared, inherited)** |
| Infrastructure required | None | Server + credentials | Server + credentials | None | None | **None** |
| CGO-free | Yes | Yes | Yes | Yes | Yes | **Yes (modernc.org/sqlite)** |
| Docker for E2E tests | No | Yes (testcontainers) | Yes (testcontainers) | No | No | **No (temp dirs / :memory:)** |
| Query interface | API | Commands | SQL (server) | API | API | **Fixed cache surface (SQL internal only)** |
| Typical concurrency model | One RWMutex | Server-side | Pool | 1 writer / N readers, single process | SSI transactions, single process | **1 writer / N readers across processes; bounded pool in-process** |

**Takeaway:** SQLite is the only option that combines persistence, multi-process file sharing (same host), zero infrastructure, and CGO-free pure Go. Its costs are: ~2x slower than C SQLite on CPU-bound work (I/O-bound workloads unaffected — fine for a cache), WAL's local-disk-only constraint, and the additional PRAGMAs/pool discipline Go's `database/sql` requires.

## Sources

- SQLite official docs — Write-Ahead Logging (multi-process rules, checkpoints, `SQLITE_BUSY` cases, 2026 WAL-reset bug fixed in 3.51.3): https://sqlite.org/wal.html (HIGH; primary spec)
- SQLite official docs — In-Memory Databases (`:memory:` distinctness, `cache=shared`, named memory DBs): https://sqlite.org/inmemorydb.html (HIGH; primary spec)
- SQLite official docs — PRAGMA journal_mode (in-memory DB journal mode is only MEMORY/OFF; WAL persistence): https://sqlite.org/pragma.html (HIGH; primary spec)
- SQLite official docs — UPSERT (`ON CONFLICT` semantics, `excluded.` table): https://sqlite.org/lang_upsert.html (HIGH; primary spec)
- modernc.org/sqlite v1.60.1 package docs — DSN parameters (`_pragma=`, `_txlock`, `_defensive`, `StrictPragmas`), connection-pool warnings, libc pinning, OFD locking, native-vs-C performance: https://pkg.go.dev/modernc.org/sqlite (HIGH; official driver docs)
- Go standard library — `os.UserCacheDir` per-OS locations and app-subdirectory guidance: https://pkg.go.dev/os#UserCacheDir (HIGH; canonical stdlib contract)
- mattn/go-sqlite3 FAQ + issue #204 — `:memory:` per-connection databases, intermittent "no such table", `SetMaxOpenConns(1)`: https://github.com/mattn/go-sqlite3/wiki/FAQ (MEDIUM; corroborated with SQLite spec and multiple threads)
- MaorBril/clauder PR #24 — production incident: mattn-style DSN params silently ignored by modernc, `_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)` fix: https://github.com/MaorBril/clauder/pull/24 (MEDIUM; single but concrete reproduction)
- hollis-labs/go-sqlite README — "WAL ≠ concurrent writes", single-writer pool, `BEGIN IMMEDIATE` / `_txlock=immediate`, busy_timeout behavior in Go: https://github.com/hollis-labs/go-sqlite (MEDIUM; design corroborated by SQLite docs)
- TTL-on-SQLite implementations (same schema/pattern: lazy read filter + startup periodic purge): sqlite-kv (TS) codefloe.com/twiceahri/simple-llm-sharing, requests-cache SQLite backend (Python), cameo SqliteCache (Rust), sqlite-cache gc (Rust): https://github.com/requests-cache/requests-cache/blob/main/requests_cache/backends/sqlite.py (MEDIUM; pattern consensus across 4+ independent codebases)
- etcd-io/bbolt README — exclusive file lock, single-process limitation, no TTL: https://github.com/etcd-io/bbolt (HIGH for bbolt behavior; competitor analysis)
- Dgraph badger README — LSM, TTL support, directory locking: https://github.com/Dgraph-io/badger (HIGH for badger behavior; competitor analysis)
- SQLite driver benchmark (mattn vs modernc, 2026) — modernc ~17–54% slower on write paths, better tail latency: https://github.com/maulanashalihin/go-sqlite-benchmark-mattn-vs-modernc (MEDIUM; single benchmark, use as directional only)
- blinki-io/dingo PR #2499 (2026) — fixed-shape prepared SQL vs ORM: ~32% faster, 3.5–4.8x fewer allocations on SQLite writes: https://github.com/blinklabs-io/dingo/pull/2499 (MEDIUM; single but measured)

---

*Feature research for: `cache/sqlite` — embedded SQLite cache backend (milestone v1.8)*
*Researched: 2026-10-08*

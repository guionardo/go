# Requirements: go — v1.8 SQLite Cache Backend

**Defined:** 2026-10-08
**Core Value:** Provide reliable, well-tested utility packages that solve common Go development problems consistently — so downstream projects don't reinvent these wheels.
**Milestone:** v1.8 SQLite Cache Backend (`cache/sqlite`)

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Provider Integration

- [ ] **PROV-01**: `sqlite.New[K, V](opts...)` returns `cache.BatchCache[K, V]`, built on a sqlite provider implementing the `cacher[K, V]` primitive and wrapped by `cache.NewConcreteCache` — matching the shape of the five existing providers.
- [ ] **PROV-02**: `GetOrSet` uses the shared `SingleflightGetOrSet` (fast-path Get, double-check Get, panic recovery); the provider adds no second dedup layer.
- [ ] **PROV-03**: Keys are stringified with `fmt.Sprint` and values serialized with `encoding/json`, matching redis/postgres wire semantics; misses return an error wrapping `cache.ErrMiss`.
- [ ] **PROV-04**: All SQL calls use context-aware methods; `Delete` of a missing key is a no-op; `Close` is idempotent; provider errors carry a `cache/sqlite:` prefix.

### Storage & Location

- [ ] **STOR-01**: Default location is `os.UserCacheDir()/<cache-name>/` (created with `MkdirAll`, 0700) when constructed with a cache name; an explicit path override is accepted as given (parent directories created as needed).
- [ ] **STOR-02**: An empty path selects `:memory:` mode; both file and `:memory:` modes pin the pool to one connection (`SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`) — mandatory for `:memory:` correctness.
- [ ] **STOR-03**: Every pooled connection receives DSN-carried pragmas — `journal_mode=WAL` (file mode), `busy_timeout=5000`, `synchronous=NORMAL`, immediate transaction lock mode — using the pinned driver's validated DSN keys; a read-back test asserts `journal_mode` is WAL for file databases.
- [ ] **STOR-04**: Schema bootstrap is idempotent under `BEGIN IMMEDIATE` (`CREATE TABLE/INDEX IF NOT EXISTS`); schema is fixed: `cache_key TEXT PRIMARY KEY`, `value TEXT` (JSON), `expires_at INTEGER NULL`.
- [ ] **STOR-05**: `Close` performs a clean shutdown (final checkpoint; `-wal`/`-shm` removed on last connection close); reopening an existing cache file preserves unexpired entries.
- [ ] **STOR-06**: Path values containing DSN metacharacters (`?`, `#`) are rejected or escaped so user-supplied paths cannot inject DSN parameters.
- [ ] **STOR-07**: Long-running processes can tune WAL checkpointing (`wal_autocheckpoint` option) and trigger an explicit optimize (checkpoint and/or `VACUUM`) to reclaim disk space.

### Expiry & Reclamation

- [ ] **TTL-01**: TTL resolution matches existing providers — per-call TTL wins, zero/absent falls back to the provider default, no default means no expiry; expiry is stored as an absolute UnixNano timestamp and survives restarts.
- [ ] **TTL-02**: Expired entries are never returned: reads filter on `expires_at` and return `ErrMiss`; reads never delete rows.
- [ ] **TTL-03**: A best-effort sweep runs on open, deleting expired rows in one pass, backed by a partial index on `expires_at`.
- [ ] **TTL-04**: An optional periodic sweep interval (mirroring mem/postgres `SweepInterval`) can be configured; when unset, startup sweep + read filtering is the only reclamation.

### Batch Operations

- [ ] **BATCH-01**: `MGet` retrieves multiple keys via chunked `IN`-list queries; missing, expired, and undecodable entries are silently skipped (bounded chunk size).
- [ ] **BATCH-02**: `MSet` writes all pairs in a single transaction with a prepared upsert (`INSERT ... ON CONFLICT DO UPDATE`); the batch is atomic — any error rolls back the whole batch; one TTL applies to the batch.
- [ ] **BATCH-03**: `MDel` deletes multiple keys via chunked `IN`-list deletes; idempotent for missing keys.

### Multi-Process & Concurrency

- [ ] **CONC-01**: Two processes sharing one cache file can read and write without corruption or hard failure under contention; verified by a two-process spike (helper-process pattern) that is an exit criterion for the multi-process phase.
- [ ] **CONC-02**: Write transactions acquire locks immediately (immediate transaction lock mode) so contention surfaces at BEGIN rather than mid-transaction; contention beyond `busy_timeout` surfaces as an error — no unbounded auto-retry.
- [ ] **CONC-03**: The provider passes `go test -race` with concurrent goroutines on the pinned single-connection pool.
- [ ] **CONC-04**: Package docs state that WAL requires local storage (network filesystems unsupported) and that file sharing is same-host only.
- [ ] **CONC-05**: A CI E2E test spawns two OS processes sharing one temp database and asserts no corruption and correct behavior under concurrent writes.

### Delivery & Quality

- [ ] **QUAL-01**: The package builds and tests with `CGO_ENABLED=0` on Linux, macOS, and Windows; `modernc.org/sqlite` is pinned (v1.60.1) together with the exact `modernc.org/libc` pin required by its `go.mod`.
- [ ] **QUAL-02**: Tests cover unit, `:memory:`, reopen/persistence, concurrent, and sweep scenarios; coverage gates pass with no `.testcoverage-quick.yml` override (package ≥80%, file ≥70%, total ≥75%).
- [ ] **QUAL-03**: `doc.go` documents the provider (TTL, reopen persistence, `:memory:`, multi-process constraints) and a runnable example exists; the README package index is updated.
- [ ] **QUAL-04**: Benchmark entries measure the sqlite backend against mem and per-key loops, following the v1.6 benchmark suite conventions.

## v2 Requirements

Deferred to a future release. Tracked but not in the current roadmap.

### Storage Lifecycle

- **MAXE-01**: Max-entries cap that trims oldest entries beyond a configured bound (trigger: write-heavy caches with rare reads).
- **POOL-01**: Split read/write connection pools (trigger: benchmarks prove a real read-concurrency need).
- **RECV-01**: `PRAGMA integrity_check` recovery helper (current recovery path is "delete the cache file").

### Eviction & Security

- **LRU-01**: LRU / size-based eviction (trigger: TTL proves insufficient; high read-path cost).
- **ENCR-01**: Encryption at rest (blocked by the pure-Go driver; only if CGO becomes acceptable).

### In-Memory

- **MEMS-01**: Named shared in-memory instances for deliberate multi-cache sharing.

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Arbitrary SQL query API / ORM behavior | Turns a cache backend into a database wrapper; leaks schema and breaks the provider-swap abstraction |
| Delete-on-read of expired rows | Converts every expired-key read into a write (lock churn, WAL growth); filter-on-read + sweep instead |
| Retry middleware beyond `busy_timeout` | Hides pathological lock holders and multiplies latency; true errors must surface |
| `file::memory:?cache=shared` as the default `:memory:` DSN | Process-global shared cache would make independent caches silently share state; pool pinning instead |
| Schema migration framework / `user_version` versioning | YAGNI for a fixed three-column table; targeted documented upgrade path if ever needed |
| Periodic `VACUUM` / `auto_vacuum` by default | `VACUUM` takes an exclusive lock and rewrites the file; opt-in optimize helper instead |
| Background write-serializer goroutine | Hidden goroutine/queue semantics; pool pinning + `busy_timeout` suffice |
| Custom codec registry (gob/msgpack) | Splits the swap-providers guarantee; JSON parity maintained |
| Cross-process invalidation pub/sub | No SQLite facility; adds hidden polling and semantics the interface cannot express |
| Network filesystem / multi-host support | WAL requires shared memory — officially unsupported; use redis/valkey/postgres for shared infrastructure |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| *(populated by roadmap creation)* | | |

**Coverage:**
- v1 requirements: 27 total
- Mapped to phases: 0
- Unmapped: 27 ⚠️ (pending roadmap)

---
*Requirements defined: 2026-10-08*
*Last updated: 2026-10-08 after initial definition*

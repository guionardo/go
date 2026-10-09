# Phase 15: Provider Foundation + Core Cache Semantics - Context

**Gathered:** 2026-10-08
**Status:** Ready for planning

<domain>
## Phase Boundary

A working, durable `cache/sqlite` provider: `sqlite.New[K, V]` constructs a `cache.BatchCache[K, V]` in all three location modes; Get/Set/Delete/GetOrSet/Close match the five existing providers' contracts; TTL expiry and sweeps are enforced; WAL pragmas are verified on every pooled connection; and long-running storage controls are available (checkpoint/optimize + `wal_autocheckpoint` tuning).

Out of scope: batch operations (MGet/MSet/MDel — Phase 16) and CI/doc/benchmark hardening (Phase 17).

</domain>

<decisions>
## Implementation Decisions

### Constructor & Location API
- **D-01:** `:memory:` mode is selectable **both ways**: an explicit `WithMemory()` option *and* the empty-path zero value (STOR-02) — `New` with no location options is an in-memory cache. `WithMemory()` exists for discoverability in tests/services.
- **D-02:** Contradictory location options resolve by **documented precedence: memory > path > name > (zero value = memory)**. `New` keeps the provider-parity signature (returns `BatchCache`, no error), so conflicts are deterministic and documented in doc.go rather than reported as errors.
- **D-03:** Name-based default location is `os.UserCacheDir()/<name>/cache.db` (dir created with `MkdirAll`, 0700). — **Reversibility:** costly — once callers have cache files at the published default path, changing the default silently orphans existing caches (no migration path for a cache, but users lose persistence).
- **D-04:** Option set exported: `WithName`, `WithPath`, `WithMemory`, `WithDefaultTTL` (shared `cache.Option` pattern).

### Optimize / Checkpoint Surface (STOR-07)
- **D-05:** Explicit controls are exposed through an **optional exported interface** (`sqlite.Optimizable`) implemented by the provider; callers type-assert the returned `cache.BatchCache`. `New`'s return type stays `cache.BatchCache[K, V]` — parity with the five providers is preserved.
- **D-06:** The interface exposes two methods with distinct cost classes: `Checkpoint(ctx)` = `PRAGMA wal_checkpoint(TRUNCATE)` (cheap, reclaims WAL) and `Vacuum(ctx)` = full `VACUUM` (exclusive, heavy — documented as such).
- **D-07:** `WithAutoCheckpoint(pages int)`: `pages > 0` sets `wal_autocheckpoint`; the zero value keeps SQLite's default (1000 pages) — no sentinel/gotcha semantics.

### Periodic Sweep Mechanics (TTL-04)
- **D-08:** `WithSweepInterval(d)` starts an opt-in ticker goroutine (mem/postgres `SweepInterval` parity); when unset there is **no sweeper**. `Close` cancels the ticker and waits for it to exit (idempotent close).
- **D-09:** Sweeps are **best-effort**: an open-time sweep failure is swallowed (reclamation is opportunistic, not correctness); a periodic sweep that errors skips that tick and retries on the next interval. Cache operations are never coupled to sweep errors.
- **D-10:** The open-time sweep runs **synchronously inside `New`** before it returns — state is clean immediately; worst-case startup delay is bounded by `busy_timeout` (5s) under heavy cross-process contention.

### Error & Open-Time Semantics
- **D-11:** `New` never fails: open/validation failures (bad path, `?`/`#`, permissions, disk) are **recorded and deferred** — every operation returns the error wrapped (redis provider's lazy-connect precedent).
- **D-12:** Export sentinels `ErrClosed` (operation after `Close`) and `ErrInvalidPath` (STOR-06 `?`/`#` rejection); all provider errors carry the `cache/sqlite:` prefix; `errors.Is` works. `cache.ErrMiss` remains the miss sentinel (imported, not re-exported).

### Locked at milestone level (not re-discussed — see requirements/research)
- Pure-Go `modernc.org/sqlite` v1.60.1 + exact `modernc.org/libc` pin; CGO-free on Linux/macOS/Windows.
- DSN pragmas on every pooled connection via validated shorthand keys: `busy_timeout=5000`, `journal_mode=WAL` (file mode), `synchronous=NORMAL`, immediate transaction lock; **mandatory read-back verification test** asserting `journal_mode == wal` for file DBs.
- Pool pinned to **one connection** in both modes (`:memory:` correctness + in-process serialization).
- Schema: `cache_key TEXT PRIMARY KEY, value TEXT NOT NULL (JSON), expires_at INTEGER NULL (UnixNano)` + partial `expires_at` index; idempotent bootstrap under `BEGIN IMMEDIATE`.
- TTL resolution parity (per-call > default > none); expired = filtered on read → `ErrMiss`, never delete-on-read; sweep SQL `DELETE WHERE expires_at IS NOT NULL AND expires_at <= now`.
- `Close` performs a final checkpoint; `-wal`/`-shm` removed on last connection close.
- Values via `encoding/json`, keys via `fmt.Sprint`.
- WAL is local-storage only (network filesystems unsupported; same-host sharing).

### the agent's Discretion
- Context cancellation shape: use `*Context` SQL methods; a canceled ctx returns wrapped `ctx.Err()` (driver interrupt) — implementation detail within PROV-04 parity.
- `WithSweepInterval` clamping for tiny test-friendly intervals.
- doc.go structure, runnable example shape, README index row wording.
- Internal test seams (clock injection, helper-process pattern for multi-process tests in Phase 16).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase & requirements
- `.planning/REQUIREMENTS.md` — 27 v1 requirements; Phase 15 owns PROV-01..04, STOR-01..07, TTL-01..04
- `.planning/ROADMAP.md` — Phase 15 goal, dependencies, success criteria, execution order
- `.planning/PROJECT.md` — Current Milestone v1.8 + constraints (Go 1.26, minimal deps, 3-OS CI, coverage gates)
- `.planning/seeds/SEED-261008-2o5-sqlite-cache-provider.md` — agreed scope + breadcrumbs from the exploration

### Research (milestone-level, current 2026-10-08)
- `.planning/research/SUMMARY.md` — synthesis; DSN shorthand-key resolution (validated since driver v1.55.0, pinned v1.60.1); phase structure
- `.planning/research/STACK.md` — driver version/pin, DSN/pragma recipe, database/sql integration, CGO/CI implications
- `.planning/research/ARCHITECTURE.md` — schema, pool, TTL/sweep, batch transaction, `:memory:` handling, integration points
- `.planning/research/PITFALLS.md` — 17 triaged pitfalls (memory pool split, per-connection pragmas, WAL growth, Windows handles)
- `.planning/research/FEATURES.md` — parity table stakes, differentiators, anti-features
- `.planning/research/questions.md` — WAL/`SQLITE_BUSY` unresolved claims (Phase 16 spike; tier-floored)

### Code (integration contract)
- `cache/cache.go` — `Cache[K, V]` + `BatchCache[K, V]` interfaces (frozen)
- `cache/concrete_cache.go` — `cacher[K, V]` primitive + `NewConcreteCache` wrapper
- `cache/postgres/postgres.go` — closest provider analog (schema + TTL + sweep precedent)
- `cache/redis/redis.go` — constructor/error-wrapping/lazy-connect precedent
- `.opencode/skills/spike-findings-go/SKILL.md` — singleflight/TTL integration constraints for the shared `GetOrSet`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cache.NewConcreteCache` + `cacher[K, V]`: free `Cache`, `BatchCache`, and singleflight `GetOrSet` wiring — the provider only supplies the seven primitive funcs.
- `cache.ErrMiss` / `SingleflightGetOrSet`: the provider must return wrapped `ErrMiss` for the double-check seam to work.
- `cache/postgres`: template for TTL storage (`expires_at`), sweep-on-open, and error prefixing.

### Established Patterns
- Functional options: shared `cache.Option` (`WithDefaultTTL`) + provider-local options (see `cache/redis/options.go`).
- Sentinel errors + `%w` wrapping; constructors return `BatchCache`; no panics in production paths.
- `doc.go` per package, co-located tests, `example_test.go`, coverage gates (package ≥80 / file ≥70 / total ≥75).

### Integration Points
- New package `cache/sqlite`; `go.mod`/`go.sum` gain `modernc.org/sqlite` + pinned `modernc.org/libc`.
- README package index row; no changes to `cache/` root or the five existing providers.
- Phase 16 extends this provider with batch funcs; Phase 17 hardens CI/docs/benchmarks.

</code_context>

<specifics>
## Specific Ideas

- DSN (pinned driver v1.60.1, validated shorthand keys): `?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate` — with a read-back test as the mechanical guard.
- `:memory:` mode: pool pinned to 1 connection is mandatory; WAL is never asserted for memory databases.
- File mode also pins the pool to 1 connection (in-process serialization; cross-process concurrency still via WAL).
- Upsert shape for `Set`: `INSERT ... ON CONFLICT(cache_key) DO UPDATE`.
- README/doc wording positioning: "mem but survives restarts; redis/postgres without the infra".
- Document network-filesystem unsupported + same-host sharing in doc.go (CONC-04 text owned by this phase's docs decision; verification lands with Phase 16's CONC work).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

### Reviewed Todos (not folded)
- **Spike modernc SQLite WAL with two concurrent processes** (`.planning/todos/pending/2026-10-08-spike-modernc-sqlite-wal-with-two-concurrent-processes.md`) — reviewed during this discussion; kept for **Phase 16** where it is the two-process exit criterion (`resolves_phase: 16`), not folded into Phase 15.

</deferred>

---

*Phase: 15-Provider Foundation + Core Cache Semantics*
*Context gathered: 2026-10-08*

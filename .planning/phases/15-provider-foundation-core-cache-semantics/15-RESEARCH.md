# Phase 15: Provider Foundation + Core Cache Semantics - Research

**Researched:** 2026-10-08
**Domain:** Embedded SQLite cache provider (pure-Go `modernc.org/sqlite`) — constructor/location modes, primitive CRUD + TTL semantics, schema bootstrap, WAL pragma enforcement, storage lifecycle controls — inside the frozen `cache/` contract of `github.com/guionardo/go`
**Confidence:** HIGH for driver mechanics (verified from the pinned module's source and by a live probe in this session), HIGH for integration seams (direct code reads), MEDIUM for cross-OS pragma behavior until Phase 17 CI confirms. Multi-process contention behavior is explicitly Phase 16 spike territory — not researched here beyond what this phase needs.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Constructor & Location API
- **D-01:** `:memory:` mode is selectable **both ways**: an explicit `WithMemory()` option *and* the empty-path zero value (STOR-02) — `New` with no location options is an in-memory cache. `WithMemory()` exists for discoverability in tests/services.
- **D-02:** Contradictory location options resolve by **documented precedence: memory > path > name > (zero value = memory)**. `New` keeps the provider-parity signature (returns `BatchCache`, no error), so conflicts are deterministic and documented in doc.go rather than reported as errors.
- **D-03:** Name-based default location is `os.UserCacheDir()/<name>/cache.db` (dir created with `MkdirAll`, 0700). — **Reversibility:** costly — once callers have cache files at the published default path, changing the default silently orphans existing caches (no migration path for a cache, but users lose persistence).
- **D-04:** Option set exported: `WithName`, `WithPath`, `WithMemory`, `WithDefaultTTL` (shared `cache.Option` pattern).

#### Optimize / Checkpoint Surface (STOR-07)
- **D-05:** Explicit controls are exposed through an **optional exported interface** (`sqlite.Optimizable`) implemented by the provider; callers type-assert the returned `cache.BatchCache`. `New`'s return type stays `cache.BatchCache[K, V]` — parity with the five providers is preserved.
- **D-06:** The interface exposes two methods with distinct cost classes: `Checkpoint(ctx)` = `PRAGMA wal_checkpoint(TRUNCATE)` (cheap, reclaims WAL) and `Vacuum(ctx)` = full `VACUUM` (exclusive, heavy — documented as such).
- **D-07:** `WithAutoCheckpoint(pages int)`: `pages > 0` sets `wal_autocheckpoint`; the zero value keeps SQLite's default (1000 pages) — no sentinel/gotcha semantics.

#### Periodic Sweep Mechanics (TTL-04)
- **D-08:** `WithSweepInterval(d)` starts an opt-in ticker goroutine (mem/postgres `SweepInterval` parity); when unset there is **no sweeper**. `Close` cancels the ticker and waits for it to exit (idempotent close).
- **D-09:** Sweeps are **best-effort**: an open-time sweep failure is swallowed (reclamation is opportunistic, not correctness); a periodic sweep that errors skips that tick and retries on the next interval. Cache operations are never coupled to sweep errors.
- **D-10:** The open-time sweep runs **synchronously inside `New`** before it returns — state is clean immediately; worst-case startup delay is bounded by `busy_timeout` (5s) under heavy cross-process contention.

#### Error & Open-Time Semantics
- **D-11:** `New` never fails: open/validation failures (bad path, `?`/`#`, permissions, disk) are **recorded and deferred** — every operation returns the error wrapped (redis provider's lazy-connect precedent).
- **D-12:** Export sentinels `ErrClosed` (operation after `Close`) and `ErrInvalidPath` (STOR-06 `?`/`#` rejection); all provider errors carry the `cache/sqlite:` prefix; `errors.Is` works. `cache.ErrMiss` remains the miss sentinel (imported, not re-exported).

#### Locked at milestone level (not re-discussed — see requirements/research)
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

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.

### Reviewed Todos (not folded)
- **Spike modernc SQLite WAL with two concurrent processes** (`.planning/todos/pending/2026-10-08-spike-modernc-sqlite-wal-with-two-concurrent-processes.md`) — reviewed during this discussion; kept for **Phase 16** where it is the two-process exit criterion (`resolves_phase: 16`), not folded into Phase 15.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PROV-01 | `sqlite.New[K, V](opts...)` returns `cache.BatchCache[K, V]` built on a provider implementing the `cacher[K, V]` primitive, wrapped by `cache.NewConcreteCache`, matching the five existing providers | Integration Seams §1–2; New open sequence skeleton; deferred-error provider pattern |
| PROV-02 | `GetOrSet` uses the shared `SingleflightGetOrSet` (fast-path Get, double-check Get, panic recovery); provider adds no second dedup layer | Integration Seams §3 (root owns dedup; `GetFunc` must wrap `ErrMiss`); double-check requires miss error |
| PROV-03 | Keys `fmt.Sprint`, values `encoding/json`, misses wrap `cache.ErrMiss` | SQL/encoding parity section; Get/Set skeletons; parity pitfalls |
| PROV-04 | Context-aware SQL; `Delete` of a missing key is a no-op; idempotent `Close`; `cache/sqlite:` error prefix | `check()` + deferred errors; `*Context` methods; Delete SQL; Close skeleton |
| STOR-01 | Default `os.UserCacheDir()/<name>/` (MkdirAll 0700); explicit path override accepted as given (parent dirs created) | Location resolution algorithm; `os.UserCacheDir` semantics; deferred error handling |
| STOR-02 | Empty path selects `:memory:`; both modes pin pool to one connection | Resolution precedence (D-01/D-02); pool-pinning pattern; probe: memory roundtrip works |
| STOR-03 | Every pooled connection gets DSN pragmas; read-back asserts `journal_mode == wal` for file DBs | Verified DSN recipe + probe read-backs (`wal/5000/1`); read-back test strategy |
| STOR-04 | Idempotent schema bootstrap under `BEGIN IMMEDIATE`; fixed three-column schema | Bootstrap skeleton (`_txlock=immediate` → `begin immediate`, driver tx.go:23-24); DDL constants; no `user_version` (Out of Scope) |
| STOR-05 | `Close` clean shutdown (final checkpoint; `-wal`/`-shm` removed on last close); reopen preserves unexpired entries | Probe: sidecars removed after `db.Close()`; reopen probe (`journal_mode=wal`, value preserved); Close skeleton |
| STOR-06 | Paths with DSN metacharacters (`?`, `#`) rejected or escaped | Path/name validation rules; `ErrInvalidPath`; DSN constants only (no interpolation) |
| STOR-07 | Tune `wal_autocheckpoint`; explicit optimize (checkpoint and/or `VACUUM`) | `WithAutoCheckpoint` via `_pragma=wal_autocheckpoint(N)` (probe-verified); `Optimizable` adapter seam (required — see Critical Finding 2) |
| TTL-01 | Per-call TTL wins, zero/absent falls back to default, none = no expiry; absolute UnixNano survives restarts | `resolveTTL` skeleton copied from postgres; INTEGER column; probe persistence |
| TTL-02 | Expired entries never returned; reads filter; reads never delete rows | Filter-only SELECT (no lazy delete — REQUIREMENTS Out of Scope); `ErrNoRows` → wrapped `ErrMiss` |
| TTL-03 | Best-effort sweep on open, partial index | Sweep skeleton + `SweepSQL`; partial index DDL; synchronous-in-New call site |
| TTL-04 | Optional periodic sweep interval (mem/postgres parity); unset = startup sweep + read filtering only | `WithSweepInterval` + `sweepLoop` skeleton; Close cancels/waits; mem/postgres option-shape parity |
</phase_requirements>
## Summary

Phase 15 is a provider-addition inside a frozen architecture. `cache/sqlite` implements the unexported `cacher[K, V]` primitive structurally and hands it to `cache.NewConcreteCache`; the root package then supplies `Cache`, `BatchCache`, and singleflight `GetOrSet` for free. Everything the phase must build is new-package code plus one dependency edge in `go.mod`: constructor/location resolution, a DSN builder with validated shorthand pragma keys, a pool pinned to exactly one connection, transactional schema bootstrap, CRUD with postgres-parity TTL semantics, a best-effort open sweep plus an opt-in periodic sweeper, and the STOR-07 controls.

This session verified the driver mechanics at primary-source level instead of relying on the milestone research prose: I extracted `modernc.org/sqlite` v1.60.1 from the module cache, read its DSN implementation (`driver.go`, `sqlite.go`, `tx.go`), and ran a throwaway probe module against the real pinned driver. The probe mechanically confirmed: the locked file DSN yields `journal_mode=wal, busy_timeout=5000, synchronous=1`; the `:memory:` DSN works; `_pragma=wal_autocheckpoint(200)` reads back 200; an invalid shorthand value fails the open (fail-fast); `db.Close()` removes `-wal`/`-shm` and `journal_mode` persists as `wal` on reopen; a bad-`_journal_mode` DSN is rejected before any statement runs; and an unconsumed `sql.Rows` blocks the single pinned connection (a hard implementation constraint for every query). The full probe transcript is quoted in Sources.

Four planning-critical findings were discovered that are not obvious from the milestone research and change how the phase must be planned:

1. **The `cacher` interface forces all seven primitive methods into Phase 15.** `cache.NewConcreteCache` requires `MGetFunc/MSetFunc/MDelFunc` to compile, yet batch semantics are Phase-16 scope. The phase must ship minimal, documented loop-based batch methods (or the plan must consciously pull full batch into Phase 15); there is no way to return `BatchCache` without them.
2. **The STOR-07 `Optimizable` type assertion cannot succeed on the raw `NewConcreteCache` value.** The frozen `concreteCache` does not forward provider-specific methods, so `New` must return a thin sqlite-side adapter that embeds `cache.BatchCache` and implements `Checkpoint`/`Vacuum`. This is an implementation requirement, not a nuance.
3. **Reads never delete rows.** REQUIREMENTS TTL-02 and the Out-of-Scope table supersede milestone research ARCHITECTURE D5's "lazy delete on read": the read path is a filter-only SELECT; all reclamation goes through the open sweep and the optional periodic sweep.
4. **No `user_version`.** The Out-of-Scope table explicitly excludes versioning; idempotent DDL under `BEGIN IMMEDIATE` remains, but the version-read/version-write steps recommended by milestone research and PITFALLS P5 must be dropped.

**Primary recommendation:** Build the phase in the dependency order — dependency + skeleton/options/location/DSN/open/Close, then CRUD + TTL + error taxonomy, then sweeps, then the optimize surface + batch placeholders — shipping tests with each commit because the repo's per-commit `make coverage-quick` gate has no `cache/sqlite` override.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Constructor, location resolution, DSN build | `cache/sqlite` (new package) | — | `New` owns config→DSN→pool→bootstrap; no root involvement (frozen) |
| Primitive CRUD semantics (Get/Set/Delete, TTL) | `cache/sqlite` (`sqliteCache`) | SQLite engine | Provider owns SQL + `resolveTTL`; engine enforces types/index |
| `GetOrSet` singleflight dedup | `cache/` root (frozen `concreteCache` + `SingleflightGetOrSet`) | `cache/sqlite` (must return wrapped `ErrMiss`) | Dedup is an inherited contract; provider must not add a second layer (PROV-02) |
| Per-connection pragma enforcement | SQLite driver via DSN | `cache/sqlite` (builds DSN) | Driver applies DSN keys on every physical connection; provider never uses post-open `db.Exec` pragmas |
| Pool serialization (one connection) | `database/sql` pool config | `cache/sqlite` (`New` sets it) | `MaxOpenConns(1)` is the entire in-process concurrency model |
| Schema bootstrap | `cache/sqlite` (`schema.go`) | SQLite engine | `CREATE ... IF NOT EXISTS` under `BEGIN IMMEDIATE`; no migration framework |
| TTL reclamation | SQLite engine (reads filter) | `cache/sqlite` (sweep-on-open + opt-in ticker) | Expiry is evaluated in SQL against a bound `now`; reclamation is best-effort maintenance |
| WAL lifecycle / sidecar cleanup | SQLite engine | `cache/sqlite` (`Close`) | Last-connection close performs the final checkpoint and deletes `-wal`/`-shm`; provider must not `os.Remove` |
| Storage controls (checkpoint/vacuum/autocheckpoint) | `cache/sqlite` (`Optimizable` + adapter) | SQLite engine | Root `BatchCache` cannot carry extra methods — adapter required |
| Error taxonomy (`ErrMiss`, `ErrClosed`, `ErrInvalidPath`) | `cache/sqlite` | `cache/` root (`cache.ErrMiss` imported) | Provider wraps everything with `cache/sqlite:`; miss sentinel is shared |

## Critical Planning Findings

These five items are the phase-scoped resolutions the planner must encode. Each was either discovered by direct code reading or mechanically verified this session.

### Finding 1 — The frozen `cacher` interface requires all 7 methods in Phase 15 (batch stubs mandatory)

`cache.NewConcreteCache[K, V](c cacher[K, V])` takes the unexported 7-method interface. The compiler checks structural satisfaction at the call site, so a provider with only Get/Set/Delete/Close **cannot compile** with `NewConcreteCache`:

```go
// cache/concrete_cache.go:13-23 [VERIFIED: cache/concrete_cache.go:13-23]
cacher[K comparable, V any] interface {
    GetFunc(ctx context.Context, key K) (V, error)
    SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error
    DeleteFunc(ctx context.Context, key K) error
    CloseFunc() error

    // Batch operations — Phase 7 additions.
    MGetFunc(ctx context.Context, keys ...K) map[K]V
    MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error
    MDelFunc(ctx context.Context, keys ...K) error
}
```

CONTEXT says batch operations are out of scope for Phase 15. **Recommendation (planner should verify this is acceptable):** ship minimal loop-based implementations — `MGetFunc` loops `GetFunc` skipping misses; `MSetFunc`/`MDelFunc` loop the point ops — each carrying a `// Phase 16: replace with chunked IN / single-transaction implementations` comment. Phase 15 tests for these should be compile-level + one happy-path smoke only; full BATCH-01..03 semantics and tests stay in Phase 16. The alternative (implementing full chunked/transactional batch now) contradicts the stated phase boundary and is not recommended.

### Finding 2 — STOR-07 needs a sqlite-side adapter; the raw `BatchCache` does not implement `Optimizable`

`cache.NewConcreteCache` returns `&concreteCache[K, V]` (`[VERIFIED: cache/concrete_cache.go:26-30]`), an unexported root type whose method set is exactly the 8 `BatchCache` methods. A caller doing `c.(sqlite.Optimizable)` on that value **fails**, because the frozen root cannot and must not forward provider-specific methods. `New` must therefore return an sqlite-side wrapper whose dynamic type implements `Optimizable` while still being a `cache.BatchCache[K, V]`. Skeleton in Code Examples §5. This preserves D-05's contract ("callers type-assert the returned `cache.BatchCache`") without touching the root package.

### Finding 3 — Reads never delete rows (lazy-delete-on-read is superseded)

Three authoritative sources agree, and the milestone research ARCHITECTURE D5 disagrees:

- REQUIREMENTS.md TTL-02: "Expired entries are never returned: reads filter on `expires_at` and return `ErrMiss`; reads never delete rows." `[VERIFIED: .planning/REQUIREMENTS.md:31]`
- REQUIREMENTS.md Out-of-Scope: "Delete-on-read of expired rows | Converts every expired-key read into a write (lock churn, WAL growth); filter-on-read + sweep instead" `[VERIFIED: .planning/REQUIREMENTS.md:82]`
- CONTEXT: "expired = filtered on read → `ErrMiss`, never delete-on-read" `[VERIFIED: .planning/phases/15-provider-foundation-core-cache-semantics/15-CONTEXT.md:43]`

Read path is therefore a single SELECT with the expiry predicate folded in; expired rows simply produce `sql.ErrNoRows`. All cleanup is the open-time sweep and the opt-in periodic sweep. (Note: PROJECT.md still says "lazy delete on read" in its milestone description — stale; do not follow it.)

### Finding 4 — No `user_version`; bootstrap is DDL-only under `BEGIN IMMEDIATE`

Milestone research (ARCHITECTURE D4, PITFALLS P5) recommended `PRAGMA user_version = 1` as a migration hook. REQUIREMENTS Out-of-Scope explicitly rejects it: "Schema migration framework / `user_version` versioning | YAGNI for a fixed three-column table; targeted documented upgrade path if ever needed" `[VERIFIED: .planning/REQUIREMENTS.md:85]`. The bootstrap keeps only: one `BeginTx` (which `_txlock=immediate` turns into `BEGIN IMMEDIATE`), `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, `Commit`. Two concurrent first-opens still serialize on the immediate lock, and `IF NOT EXISTS` makes the loser's DDL a no-op. No version read/write, no re-read inside the lock.

### Finding 5 — `New` never fails: deferred `initErr` + explicit `ErrClosed`/`ErrInvalidPath`

`New[K, V](opts ...Option) cache.BatchCache[K, V]` — no `error` return (D-02/D-11), unlike postgres. The provider records open/validation failures in `initErr` and every primitive method returns `fmt.Errorf("cache/sqlite: %w", c.initErr)` (valkey's `initErr` precedent `[VERIFIED: cache/valkey/valkey.go:19-23,51-53]`). After a successful open, `Close` sets a `closed` flag; ops return a wrapped `ErrClosed`. The two new sentinels live in the sqlite package: `ErrClosed` and `ErrInvalidPath` (D-12). `GetOrSet` on a broken handle runs the caller's setter and then fails at `SetFunc` — the established lazy-connect behavior of redis/valkey; do not special-case it.

## Integration Seams in `cache/` (exact, frozen)

| Seam | Location | Contract the provider must honor |
|------|----------|----------------------------------|
| `cacher[K, V]` (7 methods) | `cache/concrete_cache.go:13-23` | Structural satisfaction; all 7 methods required (Finding 1) |
| `NewConcreteCache[K, V](c)` | `cache/concrete_cache.go:26-30` | Returns `BatchCache[K, V]`; does **not** forward extra methods (Finding 2) |
| `Cache`/`BatchCache` interfaces | `cache/cache.go:11-48` | Public surface; `Set` TTL variadic; `MGet` returns only found keys |
| `ErrMiss` / `ErrClosed` | `cache/errors.go:38-44` | `ErrMiss` must be wrapped by `GetFunc`; root `ErrClosed` exists but is unused by all providers — sqlite exports its own per D-12 (do not edit root) |
| `SingleflightGetOrSet.Do` | `cache/singleflight.go:35-71` | Fast-path `get` outside the group; group re-checks `get`; setter runs once; error from `get` triggers the setter path — so a deferred `initErr` becomes a setter-then-`Set` failure |
| Provider option pattern | `cache/redis/options.go:14`, `cache/postgres/options.go:15` | Provider-local `type Option func(*Config)` — the root `cache.Option` interface (`cache/options.go:13-16`) is **not consumed by any provider**; do not use it for sqlite |
| `resolveTTL` parity | `cache/postgres/postgres.go:246-256` | Copy verbatim semantics: per-call `> 0` wins; else default `> 0`; else no expiry |
| Sweep option parity | `cache/mem/config.go:25-27`, `cache/postgres/sweeper.go:13-24` | Same option name/semantics, but sqlite's default is **no sweeper** (D-08) — a deliberate divergence from mem/postgres defaults |
| Coverage gate | `Makefile:119-122`, `.testcoverage-quick.yml` | `coverage-quick` runs `go test ./...` + go-test-coverage; thresholds file 70 / package 80 / total 75; **no override for cache/sqlite** — tests must ship with code |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `modernc.org/sqlite` | **v1.60.1** | `database/sql` driver for embedded SQLite; registers driver name `"sqlite"` | Only mature CGO-free SQLite driver; bundles SQLite 3.53.4; builds/tests on linux/darwin/windows with `CGO_ENABLED=0`; milestone-locked |
| `modernc.org/libc` | **v1.77.1 (exact pin, indirect)** | Transpiled libc runtime | Upstream mandate: downstream go.mod must pin the driver's libc version (cznic/sqlite#177); v1.60.1's `go.mod` requires exactly v1.77.1 |
| `database/sql` (stdlib) | Go 1.26 | Pool, prepared statements, context-carrying queries | The driver is a `database/sql` driver; no wrapper library |
| `encoding/json` (stdlib) | Go 1.26 | Value serialization parity | TEXT column stores `json.Marshal` output; identical to all five providers |
| `fmt.Sprint` (stdlib) | Go 1.26 | Key stringification parity | Matches redis/postgres/mem/memcache/valkey wire semantics |
| `os.UserCacheDir` + `path/filepath` (stdlib) | Go 1.26 | Default location | `$XDG_CACHE_HOME`/`~/.cache` (Unix), `~/Library/Caches` (Darwin), `%LocalAppData%` (Windows); **errors** when undetermined or when `$XDG_CACHE_HOME` is relative `[VERIFIED: go doc os.UserCacheDir]` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `log/slog` (stdlib) | Go 1.26 | Sweep/WAL warnings | `var logger = sync.OnceValue[*slog.Logger]` with `slog.String("module", "cache/sqlite")` (postgres precedent) |
| `golang.org/x/sync` | v0.22.0 → **v0.23.0 after tidy** | singleflight (root) | No new direct requirement; MVS will bump the existing direct pin — expect and review the go.mod diff |
| `github.com/stretchr/testify` | v1.11.1 (existing) | Assertions | Unchanged |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `modernc.org/sqlite` | `mattn/go-sqlite3` | Faster on some CRUD but requires CGO + gcc on all 3 CI OSes — rejected at milestone level |
| `modernc.org/sqlite` | `ncruces/go-sqlite3` | Pure-Go Wasm alternative; higher per-connection memory + Windows WAL history — rejected at milestone level |
| `modernc.org/sqlite` | `zombiezen.com/go/sqlite` | Wraps modernc but no `database/sql` driver by design — rejected |
| SQLite | bbolt / badger | Both fail the multi-process requirement (exclusive/dir locks) — rejected |
| Stub batch loops | Full chunked batch now | Full batch implements Phase-16 requirements early and violates the stated phase boundary — not recommended (Finding 1) |

**Installation:**
```bash
# In the repo (milestone branch gsd/v1.8-sqlite-cache-backend):
go get modernc.org/sqlite@v1.60.1
go mod tidy
go list -m modernc.org/sqlite modernc.org/libc golang.org/x/sys golang.org/x/sync
# Expected: modernc.org/sqlite v1.60.1; modernc.org/libc v1.77.1; golang.org/x/sys v0.48.0; golang.org/x/sync v0.23.0
```

**Version verification (performed this session):** `proxy.golang.org/modernc.org/sqlite/@latest` returns `{"Version":"v1.60.1","Time":"2026-09-29T08:41:38Z","Origin":{"VCS":"git","URL":"https://gitlab.com/cznic/sqlite",...}}`; `modernc.org/libc/@latest` returns v1.77.1 (2026-09-21); v1.60.1's `.mod` declares `go 1.26.0`, `golang.org/x/sys v0.48.0`, `modernc.org/libc v1.77.1` `[VERIFIED: proxy.golang.org, checked 2026-10-08]`.

**Observed go.mod delta (from a minimal probe module, `go mod tidy` against the cache):** direct adds `modernc.org/sqlite v1.60.1`; indirect adds `github.com/dustin/go-humanize v1.0.1`, `github.com/google/uuid v1.6.0`, `github.com/mattn/go-isatty v0.0.24`, `github.com/ncruces/go-strftime v1.0.0`, `github.com/remyoudompheng/bigfft v0.0.0-20230129...`, `modernc.org/libc v1.77.1`, `modernc.org/mathutil v1.7.1`, `modernc.org/memory v1.12.1`; existing pins **bump**: `golang.org/x/sys v0.47.0 → v0.48.0` (required by the driver) and `golang.org/x/sync v0.22.0 → v0.23.0` (MVS over the new graph; verified in a second probe that imports singleflight) `[VERIFIED: local probe go.mod/go list -m all]`. Add the libc pin comment to `go.mod` per milestone research; full pin verification/CI assertion is Phase 17 (QUAL-01).

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy check` seam supports `npm|pypi|crates` only — invoking it with `--ecosystem go` returns `Error: Usage: ... <npm|pypi|crates>`. Go-module verification was therefore performed with Go-ecosystem equivalents (module proxy origin metadata + pinned source read + live probe). No package in this phase comes from WebSearch or training-data-only provenance.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `modernc.org/sqlite` v1.60.1 | Go module proxy | module lineage since ~2019; tag 2026-09-29 | n/a (Go modules expose no download counts; widely used, govulncheck-covered) | `gitlab.com/cznic/sqlite` (canonical upstream, per proxy `Origin`) | OK | Approved |
| `modernc.org/libc` v1.77.1 | Go module proxy | transitive, required exact pin | n/a | `gitlab.com/cznic/libc` | OK | Approved (pin exact) |
| other transitives (list in Standard Stack) | Go module proxy | small utility modules, resolved by `go mod tidy` | n/a | public repos | OK | Approved |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none (Go modules run no install scripts; supply-chain exposure is limited to the pinned module zips).

**Provenance rules applied:** the driver name/version was discovered via milestone research, then independently confirmed this session against `proxy.golang.org` and by reading the v1.60.1 module source from the local module cache; it is tagged `[VERIFIED: proxy.golang.org + v1.60.1 module source + local probe]`. If the planner wants a belt-and-suspenders check, `go mod verify` and `govulncheck ./...` (Phase 17) cover the supply chain in the Go ecosystem.

## Architecture Patterns

### System Architecture Diagram

```
Consumer code
  c := sqlite.New[string, string](sqlite.WithName("app"))
  c.Get / c.Set / c.Delete / c.GetOrSet / c.MGet / ... / c.Close
        │
        ▼
cache/ root (FROZEN, untouched)
  concreteCache[K,V]  ──GetOrSet──► SingleflightGetOrSet (fast-path Get, group, setter)
        │                                │
        │ all calls via cacher[K,V]      └─ calls GetFunc/SetFunc back through the interface
        ▼
cache/sqlite (NEW)
  New(opts...) ──► resolve location (memory > path > name > zero)
        │             ├─ os.UserCacheDir()+name → MkdirAll(0700) → <dir>/cache.db
        │             ├─ explicit path (reject '?', '#' → ErrInvalidPath)
        │             └─ WithMemory()/no-options/":memory:" sentinel → :memory:
        │
        ├─ build DSN: <path or :memory:>?_busy_timeout=5000[&_journal_mode=WAL]
        │             [&_synchronous=NORMAL]&_txlock=immediate[&_pragma=wal_autocheckpoint(N)]
        ├─ sql.Open("sqlite", dsn)  +  SetMaxOpenConns(1)/SetMaxIdleConns(1)/zero lifetimes
        ├─ bootstrap: BeginTx (= BEGIN IMMEDIATE) → CREATE TABLE/INDEX IF NOT EXISTS → Commit
        ├─ file mode: PRAGMA journal_mode read-back → expect "wal" (warn on mismatch)
        ├─ sweep-on-open (best-effort, synchronous, before New returns)
        ├─ optional ticker goroutine when WithSweepInterval > 0
        └─ return &batchCache{BatchCache: NewConcreteCache(sqliteCache), provider: sqliteCache}
              ├─ implements sqlite.Optimizable → Checkpoint / Vacuum  (STOR-07)
              └─ on open/validation failure: initErr recorded; every op wraps it (New never fails)
        │
        ▼
database/sql pool (1 conn)  ──►  modernc.org/sqlite driver  ──►  SQLite engine
        reads: SELECT ... WHERE expires_at IS NULL OR expires_at > ?   (never deletes)
        writes: upsert ON CONFLICT(cache_key) DO UPDATE
        sweep: DELETE WHERE expires_at IS NOT NULL AND expires_at <= ?
        file sidecars (-wal/-shm) removed by SQLite on last-connection close
```

### Recommended Project Structure

```
cache/sqlite/
├── doc.go                    # package docs (minimal in P15; full constraint docs land in Phase 17)
├── options.go                # Config, Option func(*Config), defaultConfig,
│                             #   WithName, WithPath, WithMemory, WithDefaultTTL,
│                             #   WithSweepInterval, WithAutoCheckpoint
├── dsn.go                    # location resolution, path/name validation, DSN builder,
│                             #   pragma constants (pure functions — fully unit-testable)
├── schema.go                 # CreateTableSQL, CreateIndexSQL, select/upsert/delete/sweep SQL constants
├── sqlite.go                 # sqliteCache[K,V], New, check(), open sequence, 7 primitive methods,
│                             #   resolveTTL, logger, initErr/closed state
├── sweep.go                  # sweep(ctx) + sweepLoop() (one-shot + opt-in ticker)
├── optimize.go               # Optimizable interface, checkpoint/vacuum provider methods,
│                             #   batchCache adapter returned by New
├── dsn_internal_test.go      # DSN builder + location precedence + validation matrix
├── options_internal_test.go  # defaults + option application (postgres parity)
├── sqlite_internal_test.go   # resolveTTL, check() states, row-count assertions, batch placeholders
└── sqlite_test.go            # black-box: :memory: + file CRUD, TTL, persistence, pragma read-back,
                              #   concurrency/-race, error taxonomy, close/sidecars
```

**Structure rationale:** `dsn.go` and `optimize.go` are additions to the milestone-research skeleton (`options.go`, `schema.go`, `sqlite.go`, `sweep.go`, `doc.go`). They exist because (a) location/DSN logic is pure and needs its own ≥70%-covered test file — keeping `sqlite.go` smaller also de-risks its file threshold — and (b) `Optimizable` is a distinct public API surface with an adapter that `New` returns (Finding 2). This layout is planning discretion, not a locked decision; the only hard constraints are new-package-only, no root changes, and a `doc.go`.

### Pattern 1: Structural satisfaction of the frozen `cacher` interface

**What:** `sqliteCache[K, V]` implements the seven primitive methods; `New` passes it to `cache.NewConcreteCache`, exactly like `cache/postgres`.
**When to use:** Always — this is the integration contract.
**Example:**
```go
// Source: cache/postgres/postgres.go:37-67 (precedent) + cache/concrete_cache.go:26-30
type sqliteCache[K comparable, V any] struct {
    db            *sql.DB
    defaultTTL    time.Duration
    sweepInterval time.Duration
    initErr       error
    closed        atomic.Bool
    stop, done    chan struct{}
}

func New[K comparable, V any](opts ...Option) cache.BatchCache[K, V] {
    cfg := defaultConfig()
    for _, opt := range opts { opt(cfg) }
    c := &sqliteCache[K, V]{defaultTTL: cfg.DefaultTTL, sweepInterval: cfg.SweepInterval}
    c.initErr = c.open(context.Background(), cfg)
    inner := cache.NewConcreteCache[K, V](c)
    return &batchCache[K, V]{BatchCache: inner, provider: c}
}
```

### Pattern 2: Deferred-error provider (`initErr`) + closed flag

**What:** `New` never fails; open/validation failures are stored and returned wrapped by every op; post-Close ops return wrapped `ErrClosed`.
**When to use:** Every primitive method. Check order: `initErr`, then `closed`.
**Example:**
```go
// Source: cache/valkey/valkey.go:19-23,51-53 (initErr precedent); D-11/D-12
var (
    ErrClosed      = errors.New("cache/sqlite: cache is closed")
    ErrInvalidPath = errors.New("cache/sqlite: invalid path")
)

func (c *sqliteCache[K, V]) check() error {
    if c.initErr != nil { return fmt.Errorf("cache/sqlite: %w", c.initErr) }
    if c.closed.Load() { return fmt.Errorf("cache/sqlite: %w", ErrClosed) }
    return nil
}
```
Note: `ErrClosed` is sqlite-local (D-12); the unused root `cache.ErrClosed` (`[VERIFIED: cache/errors.go:38-44]`) is not aliased to avoid a double `cache:`-prefixed message. `Close` itself returns `nil` even when `initErr` is set (cleanup is not an "operation" callers should fail to perform) — planner may standardize either way, but nil matches the redis/valkey cleanup precedent.

### Pattern 3: DSN-carried pragmas + mechanical read-back

**What:** Every per-connection pragma rides in the DSN; never post-open `db.Exec`. File mode additionally verifies `journal_mode` at open and warns (never fails) on mismatch.
**When to use:** Always; the single pinned connection is created by the bootstrap Exec, so the DSN is applied before any statement.
**Example:**
```go
// Source: modernc.org/sqlite v1.60.1 driver.go:110-153 + local probe (see Sources)
const (
    dsnSuffixFile   = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"
    dsnSuffixMemory = "?_busy_timeout=5000&_synchronous=NORMAL&_txlock=immediate"
)

func buildDSN(path string, memory bool, autoCheckpointPages int) string {
    dsn := path
    if memory {
        dsn += dsnSuffixMemory
    } else {
        dsn += dsnSuffixFile
        if autoCheckpointPages > 0 { // strconv.Itoa keeps the value a compile-time-controlled integer
            dsn += "&_pragma=wal_autocheckpoint(" + strconv.Itoa(autoCheckpointPages) + ")"
        }
    }
    return dsn
}
```
The read-back runs after bootstrap (file mode only):
```go
var mode string
if err := c.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
    logger().Warn("cache/sqlite: journal_mode is not wal", "mode", mode, "error", err)
}
```

### Pattern 4: Filter-on-read TTL + best-effort sweeps (no lazy delete)

**What:** Reads fold the expiry predicate into SQL; expired rows behave as misses; reclamation only via sweeps.
**Trade-offs:** Expired rows can linger until the next open/periodic sweep — bounded and intentional per requirements.
**Example:**
```go
// TTL-02: reads never delete rows
const SelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`
// TTL-03/TTL-04 sweep predicate (uses the partial index)
const SweepSQL = `DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at <= ?`
```

### Pattern 5: One-transaction bootstrap under `_txlock=immediate`

**What:** `BeginTx` maps to `BEGIN IMMEDIATE` because `_txlock=immediate` sets the driver's `beginMode`: `sql = "begin " + c.beginMode` `[VERIFIED: modernc.org/sqlite v1.60.1 tx.go:23-24]`. DDL runs inside that transaction; concurrent first-opens serialize at BEGIN.
**Example:**
```go
func (c *sqliteCache[K, V]) bootstrap(ctx context.Context) error {
    tx, err := c.db.BeginTx(ctx, nil) // BEGIN IMMEDIATE via _txlock
    if err != nil { return err }
    defer func() { _ = tx.Rollback() }() // no-op after Commit
    if _, err := tx.ExecContext(ctx, CreateTableSQL); err != nil { return err }
    if _, err := tx.ExecContext(ctx, CreateIndexSQL); err != nil { return err }
    return tx.Commit()
}
```

### Pattern 6: Optimizable adapter (STOR-07)

**What:** `New` returns a wrapper embedding `cache.BatchCache[K, V]` and adding `Checkpoint`/`Vacuum`, so `c.(sqlite.Optimizable)` succeeds (Finding 2).
**Example:** see Code Examples §5.

### Anti-Patterns to Avoid

- **Post-open `db.Exec("PRAGMA ...")`:** sticks to one pooled connection and silently misses reconnects. Use the DSN.
- **Unpinned pool:** `:memory:` splits into multiple private databases; file mode fights itself with `SQLITE_BUSY`. Pin to 1/1 with zero lifetimes.
- **`QueryRowContext(...).Err()` for statements:** a `Row` that is never `Scan`ed keeps the single connection checked out — the probe deadlocked this way. Use `ExecContext` for DDL/DML; `QueryRow...Scan` for singleton SELECTs; always close/consume multi-row `Rows`.
- **Lazy delete on read:** explicitly out of scope; converts reads into writes (lock churn, WAL growth).
- **`os.Remove` on `-wal`/`-shm`:** let SQLite's last-close checkpoint clean up; deleting sidecars of a crashed WAL loses committed transactions.
- **Returning `sql.ErrNoRows` raw or mapping arbitrary DB errors to `ErrMiss`:** only `ErrNoRows` (and the SQL-filtered "expired" case, which also yields no rows) becomes `ErrMiss`.
- **WAL assertion in `:memory:` mode:** `PRAGMA journal_mode` reads back `"memory"` there (probe-verified) — never assert or include `_journal_mode` for memory DSNs.
- **`user_version`/migration framework:** explicitly out of scope (Finding 4).
- **Interpolating caller input into the DSN:** only compile-time-constant pragma values; only the validated path is concatenated, and `?`/`#` are rejected first.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Concurrent `GetOrSet` miss dedup | A provider-level mutex/singleflight | Shared `cache.SingleflightGetOrSet` via `NewConcreteCache` (PROV-02) | Already solved in root with panic recovery, double-check, and TTL passthrough; a second layer breaks SF parity |
| Cache interface surface | Custom `Cache`/`BatchCache` implementation | `cache.NewConcreteCache[K, V](c)` | Frozen contract; provides Cache + BatchCache + GetOrSet from the 7 primitives |
| Per-connection pragma application | Connection hooks / post-open Exec loops | Driver DSN keys (`_busy_timeout`, `_journal_mode`, `_synchronous`, `_txlock`) | Driver applies them on every physical connection in a fixed, validated order |
| SQLite WAL/sidecar lifecycle | `os.Remove` or manual checkpoint on Close | `db.Close()` (last-connection checkpoint deletes `-wal`/`-shm`) | SQLite-native; manual deletion risks losing committed transactions |
| Value/key encoding | Custom codec / binary encoding | `encoding/json` + `fmt.Sprint` | Hard parity contract with the five existing providers |
| TTL resolution | New precedence logic | Copy `postgres.resolveTTL` semantics | Cross-provider behavioral contract (TTL-01) |
| Fallback error wrapper | Custom retry/error machinery | Wrapped sentinels (`ErrMiss`, `ErrClosed`, `ErrInvalidPath`) + `%w` | `errors.Is` is the consumer contract; retry policy is Phase 16 |
| Directory creation | Manual stat/mkdir chains | `os.MkdirAll(filepath.Dir(path), 0o700)` | Handles races and nested dirs; matches D-03 |

## Common Pitfalls

### Pitfall 1: `:memory:` + `database/sql` = multiple separate databases
**What goes wrong:** Every new pooled connection to plain `:memory:` opens a *private* database; writes on one connection are invisible on another (`no such table` / `ErrMiss` right after `Set`), flakily, only under concurrency.
**Why it happens:** `database/sql` is a pool; SQLite documents that every `:memory:` database is distinct.
**How to avoid:** `SetMaxOpenConns(1)` + `SetMaxIdleConns(1)` + zero conn lifetimes in **both** modes. Never use `file::memory:?cache=shared`.
**Warning signs:** parallel test flakes; `db.Stats().OpenConnections > 1`; tests that pass alone but fail in the package run.
**Phase evidence:** probe confirmed memory roundtrip works with the pinned pool; `PRAGMA busy_timeout` reads 5000 on the memory connection.

### Pitfall 2: Pragmas on one connection only / silently ignored
**What goes wrong:** Post-open `db.Exec` pragmas apply to one arbitrary connection; DSN dialects copied from mattn examples may be ignored.
**How to avoid:** All per-connection pragmas in the DSN via the validated shorthand keys; read back `journal_mode` (file mode) and assert all three values in tests. The probe verified the pinned keys apply (`wal`, `5000`, `1`) and that a typo (`_journal_mode=NOPE`) fails the open.
**Warning signs:** `SQLITE_BUSY` in milliseconds despite a timeout; `PRAGMA busy_timeout` returning 0.

### Pitfall 3: WAL silently degrades on network filesystems
**What goes wrong:** `journal_mode=WAL` returns the previous mode instead of erroring when shared memory is unavailable (NFS/SMB/OneDrive/Dropbox); the provider believes WAL is on.
**How to avoid:** Read back `journal_mode` at open; mismatch logs a warning (D-11 forbids failing construction). Full doc warnings land in Phase 17; document in `doc.go` as work proceeds.
**Warning signs:** read-back differs from `wal`; `SQLITE_IOERR_SHM*`; sidecars on remote paths.

### Pitfall 4: `busy_timeout` blind spots (Phase 15 exposure is limited)
**What goes wrong:** Deferred read→write upgrades can fail with `SQLITE_BUSY_SNAPSHOT` (517) regardless of timeout; the WAL switch/checkpoint can also escape the busy handler.
**How to avoid in Phase 15:** `_txlock=immediate` in the DSN (probe/driver-verified: `begin immediate`); writes are single-statement autocommit (upsert/delete) except the one DDL transaction; keep transactions short. The bounded retry backstop and the two-process spike are **Phase 16** (CONC-01/02) — do not build retry here.
**Warning signs:** `database is locked (5)` in ~0 ms; `database is locked (517)`.

### Pitfall 5: Concurrent first-open bootstrap race
**What goes wrong:** Two processes bootstrap a fresh DB simultaneously; check-then-act DDL races.
**How to avoid:** One `BeginTx` (`BEGIN IMMEDIATE` via `_txlock`) wrapping both `CREATE ... IF NOT EXISTS` statements; `IF NOT EXISTS` + immediate lock makes the loser's DDL a no-op. **No `user_version`** (Out of Scope).
**Warning signs:** cold-start-only failures; `table already exists` (non-idempotent DDL); passes with `-p 1`.

### Pitfall 6: Windows file semantics — leaked handles break `t.TempDir`
**What goes wrong:** Unclosed handles make temp-dir cleanup fail only on windows-latest; sidecar deletion assertions can be flaky under AV.
**How to avoid:** Idempotent `Close`; `t.Cleanup(func(){ _ = c.Close() })` immediately after `New` in every test; never leave transactions open; no Unix permission assertions on Windows; assert sidecar removal with `require.Eventually`, not immediately after `Close` (probe shows immediate removal on macOS, but Windows retries are documented).
**Warning signs:** CI red only on windows-latest; `-wal`/`-shm` surviving tests.

### Pitfall 7: File growth is silent (WAL + freelist)
**What goes wrong:** DELETE never shrinks the file; a long-lived process accumulates `-wal` until autocheckpoint; `auto_vacuum` is creation-time and irreversible.
**How to avoid:** Leave `auto_vacuum` unset (default NONE — requirements Out-of-Scope rejects periodic VACUUM/auto_vacuum defaults); expose `WithAutoCheckpoint` + `Checkpoint`/`Vacuum` as opt-ins (STOR-07); document high-water-mark + delete-to-reclaim in Phase 17 docs.
**Warning signs:** `freelist_count` high vs `page_count`; `-wal` larger than a few MB after writes stop.

### Pitfall 8: Error-mapping drift — never turn a DB error into `ErrMiss`
**What goes wrong:** Mapping any `Scan` error to `ErrMiss` hides corruption/permission/BUSY failures as cache misses; returning raw `sql.ErrNoRows` breaks `errors.Is(err, cache.ErrMiss)`.
**How to avoid:** Map **only** `errors.Is(err, sql.ErrNoRows)` to `fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)`; everything else is a wrapped real error; unmarshal failure is an error (not a miss); ops after `Close` return wrapped `ErrClosed` (probe: the driver itself returns `"sql: database is closed"` — do not let that leak raw). Add a test that a corrupted DB does *not* produce `ErrMiss`.
**Warning signs:** `errors.Is(err, cache.ErrMiss)` true for non-miss failures; raw `sql.ErrNoRows` in caller errors; missing `cache/sqlite:` prefix.

### Pitfall 9: DSN construction from a user path
**What goes wrong:** `?` truncates the path and turns the remainder into driver query parameters; `#` is rejected by requirement; path traversal via cache name escapes the cache root.
**How to avoid:** Reject `?`/`#` (→ `ErrInvalidPath`, deferred); validate the name against an allow-list (no separators, no `.`/`..`); pragma values are compile-time constants only; never interpolate caller strings into `_pragma`.
**Warning signs:** DSN built with `+` from raw input; tests only using alphanumeric temp paths.

### Pitfall 10: Default-location failures
**What goes wrong:** `os.UserCacheDir()` errors (`$HOME` unset, relative `$XDG_CACHE_HOME`); the directory may not exist; callers cannot discover the resolved path.
**How to avoid:** Propagate the error through the deferred `initErr` path (D-11, diverges deliberately from PITFALLS P10's "return an error" recommendation); `MkdirAll(..., 0o700)` for both default and explicit paths; optionally log the resolved path at debug level during `New` (cheap, aids bug reports).
**Warning signs:** `os.UserCacheDir: $HOME is not defined` in CI/containers; DB created in unexpected places.

### Pitfall 11: Unconsumed `Rows` deadlocks the single connection (discovered by probe)
**What goes wrong:** With `MaxOpenConns(1)`, a `Query` whose `Rows` is never drained/closed holds the only connection; the next query blocks until its context expires (probe: `err=context deadline exceeded`; a first probe variant deadlocked entirely because `QueryRowContext(...).Err()` was used for DDL without `Scan`).
**Why it happens:** `database/sql` returns the connection to the pool only when the rows are closed/consumed.
**How to avoid:** Use `ExecContext` for all DDL/DML; `QueryRowContext(...).Scan(...)` for singleton SELECTs; for multi-row queries (MGet in Phase 16, internal test helpers) always `defer rows.Close()` and iterate fully before returning. This must be a review point for every query added.
**Warning signs:** context-deadline errors on the second query; hangs under `-race`.

### Pitfall 12: Driver dependency, toolchain, and CI fallout
**What goes wrong:** The dependency graph grows (~9 indirect modules) and cold builds slow down (~22 MB module zip); MVS bumps existing pins.
**How to avoid:** Pin v1.60.1 + exact libc; review the `go mod tidy` diff (expect `x/sys v0.47.0→v0.48.0`, `x/sync v0.22.0→v0.23.0`); CI uses `setup-go`'s default module/build cache, so cold builds are amortized.
**Warning signs:** unreviewed go.mod churn; govulncheck on transitives (Phase 17).

### Pitfall 13: Coverage gates have no `cache/sqlite` override
**What goes wrong:** `.testcoverage-quick.yml` zeroes thresholds for Docker-tested providers only; `cache/sqlite` must meet file 70 / package 80 / total 75 with unit tests alone, enforced per commit (`make coverage-quick`, `Makefile:119-122`).
**How to avoid:** Design for testability on day one (pure location/DSN/`resolveTTL` functions; injectable `userCacheDir` lookup); cover error branches with tests (bad path, deferred open failure, closed ops, marshal failure, corrupted row, sweep failure via closed DB); keep the multi-process helper out of the coverage lane (Phase 16, e2e-tagged).
**Warning signs:** considering a threshold override; subprocess helpers claimed as coverage.

### Pitfall 14: Key/value parity drift
**What goes wrong:** Different key coercion or value encoding than the other five providers.
**How to avoid:** `fmt.Sprint(key)` at every bind site; `json.Marshal`/`Unmarshal`; test empty-string key, unicode, spaces, JSON-hostile values, and non-marshalable values (channels must error, not store garbage).
**Warning signs:** tests using only `"key"`/`"value"`.

### Pitfall 15: TTL semantics drift
**What goes wrong:** Treating `ttl=0` as "use default", storing TEXT dates, or comparing against SQL `datetime('now')`.
**How to avoid:** Copy `postgres.resolveTTL` exactly; store `expires_at` as INTEGER UnixNano, NULL = never; compare against a bound `time.Now().UnixNano()`; test the matrix (`no ttl`, `ttl>0`, `ttl==0`, `ttl<0`, default set/unset); expire tests use short TTLs + `require.Eventually` (no fixed sleeps where avoidable).
**Warning signs:** helper named differently from `resolveTTL`; TEXT dates; flaky sleep-based tests.

### Pitfall 16: `wal_checkpoint(TRUNCATE)` returns three columns
**What goes wrong:** Scanning one value from the PRAGMA errors (`expected 3 destination arguments in Scan, not 1`).
**How to avoid:** Scan `(busy, log, checkpointed)` as ints; treat `busy != 0` as "checkpoint blocked" (recommended: return an error from `Checkpoint`); memory mode returns `busy=0, log=-1, ckpt=-1` (probe-verified) and is a valid no-op success.
**Warning signs:** Scan argument-count errors in optimize tests.

## Code Examples

### 1. Location resolution (pure, testable)
```go
// dsn.go — resolution order per D-02: memory > path > name > zero value (memory)
func resolveLocation(cfg *Config, userCacheDir func() (string, error)) (path string, memory bool, err error) {
    switch {
    case cfg.Memory || cfg.Path == ":memory:":
        return ":memory:", true, nil
    case cfg.Path != "":
        if strings.ContainsAny(cfg.Path, "?#") {
            return "", false, fmt.Errorf("%w: path must not contain '?' or '#'", ErrInvalidPath)
        }
        return cfg.Path, false, nil
    case cfg.Name != "":
        if !validName(cfg.Name) {
            return "", false, fmt.Errorf("%w: invalid cache name %q", ErrInvalidPath, cfg.Name)
        }
        root, err := userCacheDir() // os.UserCacheDir in production
        if err != nil {
            return "", false, fmt.Errorf("%w", err)
        }
        return filepath.Join(root, cfg.Name, "cache.db"), false, nil
    default:
        return ":memory:", true, nil // zero value = memory (D-01/D-02)
    }
}

func validName(name string) bool {
    if name == "" || name == "." || name == ".." { return false }
    for _, r := range name {
        switch {
        case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
            r == '.', r == '_', r == '-':
        default:
            return false
        }
    }
    return true
}
```

### 2. Open sequence with deferred failure
```go
func (c *sqliteCache[K, V]) open(ctx context.Context, cfg *Config) error {
    path, memory, err := resolveLocation(cfg, os.UserCacheDir)
    if err != nil { return err }
    if !memory {
        if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return err }
    }
    db, err := sql.Open("sqlite", buildDSN(path, memory, cfg.AutoCheckpoint))
    if err != nil { return err }
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(0)
    db.SetConnMaxIdleTime(0)
    c.db = db
    if err := c.bootstrap(ctx); err != nil { // BEGIN IMMEDIATE + idempotent DDL
        _ = db.Close()
        c.db = nil
        return err
    }
    if !memory { c.verifyJournalMode(ctx) } // warn-only read-back
    c.sweep(ctx)                            // TTL-03: best-effort, synchronous
    if c.sweepInterval > 0 {                // TTL-04: opt-in
        c.stop, c.done = make(chan struct{}), make(chan struct{})
        go c.sweepLoop()
    }
    return nil
}
```

### 3. Get / Set / Delete + `resolveTTL`
```go
const (
    SelectSQL = `SELECT value FROM cache_entries
WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)`
    UpsertSQL = `INSERT INTO cache_entries (cache_key, value, expires_at) VALUES (?, ?, ?)
ON CONFLICT(cache_key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`
    DeleteSQL = `DELETE FROM cache_entries WHERE cache_key = ?`
)

func (c *sqliteCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
    var zero V
    if err := c.check(); err != nil { return zero, err }
    var data string
    err := c.db.QueryRowContext(ctx, SelectSQL, fmt.Sprint(key), time.Now().UnixNano()).Scan(&data)
    if errors.Is(err, sql.ErrNoRows) {
        return zero, fmt.Errorf("cache/sqlite: %w", cache.ErrMiss)
    }
    if err != nil { return zero, fmt.Errorf("cache/sqlite: %w", err) }
    var value V
    if err := json.Unmarshal([]byte(data), &value); err != nil {
        return zero, fmt.Errorf("cache/sqlite: %w", err)
    }
    return value, nil
}

func (c *sqliteCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
    if err := c.check(); err != nil { return err }
    data, err := json.Marshal(value)
    if err != nil { return fmt.Errorf("cache/sqlite: %w", err) }
    expiresAt, hasTTL := c.resolveTTL(ttl...)
    var exp any
    if hasTTL { exp = expiresAt } // nil binds SQL NULL
    if _, err := c.db.ExecContext(ctx, UpsertSQL, fmt.Sprint(key), string(data), exp); err != nil {
        return fmt.Errorf("cache/sqlite: %w", err)
    }
    return nil
}

func (c *sqliteCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
    if err := c.check(); err != nil { return err }
    if _, err := c.db.ExecContext(ctx, DeleteSQL, fmt.Sprint(key)); err != nil {
        return fmt.Errorf("cache/sqlite: %w", err)
    }
    return nil
}

// resolveTTL mirrors postgres exactly (cache/postgres/postgres.go:246-256).
func (c *sqliteCache[K, V]) resolveTTL(ttl ...time.Duration) (int64, bool) {
    if len(ttl) > 0 && ttl[0] > 0 { return time.Now().Add(ttl[0]).UnixNano(), true }
    if c.defaultTTL > 0 { return time.Now().Add(c.defaultTTL).UnixNano(), true }
    return 0, false
}
```

### 4. Sweep + sweeper + Close
```go
// sweep.go
func (c *sqliteCache[K, V]) sweep(ctx context.Context) {
    if _, err := c.db.ExecContext(ctx, SweepSQL, time.Now().UnixNano()); err != nil {
        logger().Warn("cache/sqlite: sweep failed", "error", err) // D-09: never fails construction
    }
}

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

func (c *sqliteCache[K, V]) CloseFunc() error {
    if c.closed.CompareAndSwap(false, true) {
        if c.stop != nil { // D-08: cancel ticker and wait for exit
            close(c.stop)
            <-c.done
        }
        if c.db != nil {
            return c.db.Close() // STOR-05: last close checkpoints + deletes -wal/-shm
        }
    }
    return nil // idempotent (probe: second Close returns nil)
}
```

### 5. `Optimizable` adapter (STOR-07 — required by Finding 2)
```go
// optimize.go
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

func (c *sqliteCache[K, V]) checkpoint(ctx context.Context) error {
    if err := c.check(); err != nil { return err }
    var busy, log, checkpointed int // PRAGMA returns THREE columns (probe-verified)
    if err := c.db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &log, &checkpointed); err != nil {
        return fmt.Errorf("cache/sqlite: %w", err)
    }
    if busy != 0 {
        return fmt.Errorf("cache/sqlite: checkpoint blocked (busy=%d)", busy)
    }
    return nil
}

func (c *sqliteCache[K, V]) vacuum(ctx context.Context) error {
    if err := c.check(); err != nil { return err }
    if _, err := c.db.ExecContext(ctx, "VACUUM"); err != nil {
        return fmt.Errorf("cache/sqlite: %w", err)
    }
    return nil
}
```

Usage:
```go
c := sqlite.New[string, string](sqlite.WithName("app"))
defer func() { _ = c.Close() }()
if opt, ok := c.(sqlite.Optimizable); ok {
    _ = opt.Checkpoint(ctx) // cheap WAL reclaim
    // _ = opt.Vacuum(ctx)  // heavy; exclusive
}
```

## Test Strategy (Phase 15)

No test framework changes: testify v1.11.1 is already a direct dependency; tests use `t.TempDir()`, `t.Context()`, and `require.Eventually` where timing matters. No Docker. All new tests run in the default `go test ./...` lane that `make coverage-quick` enforces per commit.

| Behavior (Req) | Test type | Where | Notes |
|----------------|-----------|-------|-------|
| Options defaults + each option (PROV-01) | unit (internal) | `options_internal_test.go` | Copy postgres `options_internal_test.go` shape |
| Location precedence matrix: memory/path/name/zero, `:memory:` sentinel, `?`/`#` rejection, invalid name, UserCacheDir error (STOR-01/02/06) | unit (internal) | `dsn_internal_test.go` | Inject `userCacheDir` function — deterministic, no env mutation |
| DSN builder: file/memory suffixes, autocheckpoint append (STOR-03/07) | unit (internal) | `dsn_internal_test.go` | String equality assertions |
| `resolveTTL` matrix (TTL-01) | unit (internal) | `sqlite_internal_test.go` | Mirror postgres `TestResolveTTL` subtests |
| `:memory:` CRUD + GetOrSet + concurrent `-race` (PROV-01..04, STOR-02) | integration (black-box) | `sqlite_test.go` | Concurrent GetOrSet with a counting setter; assert setter runs once |
| File-mode CRUD + TTL expiry (per-call/default/none) (TTL-01/02) | integration (black-box) | `sqlite_test.go` | Short TTL + Eventually; assert expired Get → `ErrMiss` |
| Reads never delete rows (TTL-02) | integration (internal) | `sqlite_internal_test.go` | After an expired Get, `SELECT COUNT(*)` still 1 |
| Pragma read-back: file `wal/5000/1`; memory `busy_timeout=5000` (STOR-03) | integration (black-box) | `sqlite_test.go` | The milestone-mandatory mechanical guard |
| Bootstrap idempotence: two sequential opens of same path (STOR-04) | integration (black-box) | `sqlite_test.go` | Concurrent-goroutine open of fresh path (4 goroutines) |
| Reopen persistence + journal_mode persistence (STOR-05, TTL-01) | integration (black-box) | `sqlite_test.go` | Close, reopen, value present; WAL read-back `wal` |
| Close idempotent + sidecar removal (STOR-05) | integration (black-box) | `sqlite_test.go` | `require.Eventually` for `-wal`/`-shm` absence; second Close nil |
| Open-time sweep deletes expired rows (TTL-03) | integration (internal) | `sqlite_internal_test.go` | Seed an expired row, reopen, COUNT==0 |
| Periodic sweep reclaims + stops on Close (TTL-04) | integration (internal) | `sweep` internal test | `WithSweepInterval(10ms)`, Eventually COUNT==0; Close returns cleanly |
| Deferred errors: invalid path, open failure, `ErrClosed`, corrupted JSON ≠ `ErrMiss`, marshal failure (PROV-04, D-11/12) | integration (black-box + internal seed) | `sqlite_test.go` + internal | Corrupt row inserted via internal handle |
| Optimizable: assertion succeeds, Checkpoint/Vacuum run; WithAutoCheckpoint read-back (STOR-07) | integration (black-box) | `sqlite_test.go` | `c.(sqlite.Optimizable)`; `PRAGMA wal_autocheckpoint` == configured |
| Batch placeholder smoke (compile + happy path) | integration (black-box) | `sqlite_test.go` | Full BATCH-01..03 semantics/tests are Phase 16 |

**Wave 0 gaps:** none beyond the files above — no fixtures, no framework install, no build tags. `go-test-coverage`, `golangci-lint`, `govulncheck` are already installed locally. The Phase 16 helper-process harness is deliberately deferred.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CGO SQLite drivers (mattn) for embedding | Pure-Go `modernc.org/sqlite` | mature since ~v1.2x; shorthand keys since v1.55.0 (2026-07-20) | `CGO_ENABLED=0` on all 3 CI OSes; `-race` and cross-compile work |
| mattn-style DSN keys silently ignored on modernc | Validated shorthand keys (`_busy_timeout`, `_journal_mode`, `_synchronous`, …) applied in fixed order; typo fails the open | driver v1.55.0 | The provider can trust DSN keys and read them back; `_pragma` remains verbatim SQL |
| Delete-on-read of expired rows | Filter-on-read + sweep-on-open (+ opt-in periodic sweep) | this milestone's requirements (Out of Scope: delete-on-read) | Reads never write: no lock churn/WAL growth from reads |
| `PRAGMA user_version` migration scaffolding | Fixed 3-column schema, idempotent DDL only | this milestone's Out-of-Scope decision | No migration framework; YAGNI |
| Unbounded `database/sql` pool for SQLite | `SetMaxOpenConns(1)` (+ idle 1, zero lifetimes) for both modes | driver docs + Go community consensus | Deterministic in-process serialization; `:memory:` correctness |

**Deprecated/outdated:**
- Milestone research ARCHITECTURE D5's "lazy delete on read" — superseded by REQUIREMENTS TTL-02 + Out-of-Scope (Finding 3).
- PITFALLS P5's `user_version` recommendation and P10's "return construction error" — both superseded by requirements/decisions (Findings 4, 5).
- The root `cache/doc.go:54` mention of `ErrCanceled`/`DoChan` has no implementation in the shipped code (`[VERIFIED: grep — only doc.go references]`); do not model sqlite error behavior on it.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Probe results (darwin/arm64, Go 1.27.0) for pragma read-back, sidecar cleanup, and checkpoint column shape transfer to Linux and Windows. The DSN mechanism is platform-independent in the driver source, but cross-OS confirmation is Phase 17 CI's job. | Architecture/Test Strategy | A CI-only pragma assertion fails on another OS; fix is test adjustment, not design change |
| A2 | Minimal loop-based batch methods are the right Phase-15 placeholder for the compile-required `MGetFunc/MSetFunc/MDelFunc` (vs implementing full BATCH semantics now). | Finding 1 | If rejected, Phase 15 scope grows by the full batch surface (~60 lines + tests) |
| A3 | `dsn.go` and `optimize.go` split (vs the milestone research's 5-file skeleton) is acceptable; there is no locked file layout. | Project Structure | Cosmetic replan of file split |
| A4 | `Close` returns `nil` even when `initErr` is set (cleanup succeeds by definition; ops already surface the deferred error). | Pattern 2 | Callers who rely on `Close` to report the deferred error see nil; alternative is returning the wrapped `initErr` |
| A5 | `WITHOUT ROWID` on the fixed three-column table (milestone research recommendation; requirement text does not forbid it). | Schema | Changing later requires a schema change; logical behavior identical |
| A6 | `WithAutoCheckpoint` rides in the DSN as `_pragma=wal_autocheckpoint(<int>)` (no validated shorthand exists for this pragma) — safe because the value is `strconv.Itoa` of a caller integer, never a string. | Pattern 3 | If judged against the "no `_pragma`" hygiene rule, fall back to post-open `db.Exec` on the pinned connection (slightly weaker across reconnects) |
| A7 | The `golang.org/x/sync v0.22.0 → v0.23.0` and `x/sys v0.47.0 → v0.48.0` MVS bumps are behaviorally harmless for existing packages. | Standard Stack | A missed behavior change in x/sync affects singleflight; inspect the tidy diff |
| A8 | `WithPath(":memory:")` selects memory mode (treated like `WithMemory()`); the milestone text only locks `WithMemory()` + the empty-path zero value. | Location resolution | If rejected, the DSN builder must omit `_journal_mode` for the literal `:memory:` path anyway or the WAL read-back warns spuriously |
| A9 | `sqlite.ErrClosed` is a provider-local sentinel, not an alias of the unused root `cache.ErrClosed`. | Pattern 2 | Consumers checking `errors.Is(err, cache.ErrClosed)` get false; aliasing is a one-line change if desired |

**If this table is empty:** N/A — the table above lists the open items; none blocks planning, and all are cheap to resolve.

## Open Questions

1. **`WITHOUT ROWID` vs plain rowid table** — Recommendation: `WITHOUT ROWID` (research-recommended; PK lookups dominate; drop-in change if the planner prefers postgres-convention parity).
2. **`Checkpoint` semantics when `busy != 0`** — Recommendation: return a descriptive error (the caller asked to reclaim; a blocked checkpoint didn't). Alternative: log + nil. Either way consume all three PRAGMA columns.
3. **`WithSweepInterval` clamping** — Recommendation: `d <= 0` means "no sweeper" (same as unset); allow tiny positive values for tests; no upper clamp.
4. **Name validation rule** — Recommendation: allow-list `[A-Za-z0-9._-]+`, reject `.`/`..`/empty/separators/`?`/`#`; invalid → deferred `ErrInvalidPath`.
5. **Batch placeholder shape (Finding 1)** — Recommendation: loop-based placeholders + Phase 16 TODO + smoke tests only. Planner should surface this in the plan's first task so scope is explicit.
6. **`WithBusyTimeout` option** — CONTEXT's locked option list omits it; STOR-03 fixes 5000. Recommendation: do not add it in Phase 15 (minimal surface; the locked DSN constant stands).
7. **Where `doc.go` depth lands** — Recommendation: create `doc.go` with package comment + basic usage in Phase 15 (conventional; keeps lint happy); full constraint statements (network FS, sidecars, growth) are QUAL-03/Phase 17.

None of these block planning; each has a default recommendation.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build/test | ✓ | go1.27.0 darwin/arm64 (repo directive `go 1.26.4`; driver requires ≥1.26.0) | — |
| Module proxy access | `go get` modernc + libc | ✓ | `proxy.golang.org` reachable (verified: driver/libc downloads succeeded) | vendoring (not needed) |
| `go-test-coverage` | `make coverage-quick` gate | ✓ | installed at `~/go/bin` | — |
| `golangci-lint` | lint gate | ✓ | installed | — |
| `govulncheck` | quality report (Phase 17) | ✓ | installed | — |
| Docker | not needed this phase (SQLite tests are in-process) | ✓ | installed | — |
| Linux/Windows runtime | cross-OS CI (Phases 16–17) | ✗ local | — | GitHub Actions matrix (`ubuntu-latest`, `macos-latest`, `windows-latest`) |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** Linux/Windows local execution — covered by CI.

## Security Domain

`security_enforcement` is enabled (ASVS L1) in `.planning/config.json`; the provider is a local library, so the applicable surface is input validation and file handling.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Library has no auth concepts |
| V3 Session Management | no | No sessions |
| V4 Access Control | no | No authorization surface |
| V5 Input Validation | yes | Path/name validation (`?`/`#` rejection → `ErrInvalidPath`, allow-listed name), parameterized SQL only, DSN built solely from compile-time constants |
| V6 Cryptography | no (explicitly) | No crypto in scope; cache contents are unencrypted at rest (ENCR-01 deferred to v2, blocked by the pure-Go driver). Document in Phase 17 docs |
| V7 Error Handling & Logging | yes | Wrapped `%w` errors with `cache/sqlite:` prefix; never log keys/values (avoid leaking cached secrets at info level) |
| V12 Files & Resources | yes | `MkdirAll` 0o700 for cache dirs; SQLite file permissions inherited from process umask; idempotent `Close` releases handles |

### Known Threat Patterns for {Go + SQLite + database/sql}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| DSN injection via path/name (`?` truncates path and injects driver params; `_pragma` values execute verbatim SQL) | Tampering | Reject `?`/`#`; allow-list name; pragma values are constants (the only variable value, `wal_autocheckpoint`, is `strconv.Itoa(int)`) |
| Path traversal via cache name (`../../.ssh/...`) | Tampering | Name allow-list (no separators, no dots-only names), joined under `os.UserCacheDir()` |
| SQL injection via keys/values | Tampering | Every statement binds parameters (`?`); no string-built SQL; keys/values are data, never identifiers |
| Cross-process cache poisoning via shared file | Spoofing/Tampering | Same-host + filesystem permissions are the trust boundary; document WAL's same-host-only constraint (CONC-04 text authored by this phase's docs decision, verified in Phase 16) |
| Secret leakage through logs/errors | Information Disclosure | Log only operational metadata (mode, path at debug); never values or keys |
| Local cache file read by other users | Information Disclosure | 0o700 directory on Unix; document no encryption at rest |

## Sources

### Primary (HIGH confidence)
- `modernc.org/sqlite` v1.60.1 module source (extracted from `~/go/pkg/mod/cache/download/.../@v/v1.60.1.zip`): `driver.go` lines 80–287 (DSN forms, validated shorthand keys at 117–136, fixed apply order 121–128, verbatim `_pragma` 143–153, `_txlock` 194–196), `sqlite.go` lines 285–401 (validation-before-apply, `_txlock` validation), `tx.go` lines 15–27 (`begin immediate` via `beginMode`) — `[VERIFIED: source read this session]`
- Local probe module (`/var/folders/zt/.../T/opencode/sqlite-probe`, throwaway; repo untouched), `modernc.org/sqlite v1.60.1`, darwin/arm64, Go 1.27.0 — verbatim output:
  ```
  FILE  journal_mode="wal" busy_timeout=5000 synchronous=1
  AUTOCKPT wal_autocheckpoint=200
  MEM   busy_timeout=5000 roundtrip="v"
  CLOSE wal_exists=false shm_exists=false db_exists=true
  REOPEN journal_mode="wal" value=1
  CLOSED-OP err="sql: database is closed"
  CHECKPOINT busy=0 log=0 checkpointed=0
  VACUUM err=<nil>
  BAD-DSN failedOpen=true err=invalid _journal_mode "NOPE", expecting one of: DELETE TRUNCATE PERSIST MEMORY WAL OFF
  BLOCKED-BY-OPEN-ROWS err=context deadline exceeded
  CLOSE-TWICE e1=<nil> e2=<nil>
  MEM journal_mode="memory" checkpoint(err=<nil> busy=0 log=-1 ckpt=-1)
  ```
  plus a first-run `fatal error: all goroutines are asleep - deadlock!` from an unconsumed `Row` (Pitfall 11) — `[VERIFIED: local probe]`
- `proxy.golang.org` metadata: `modernc.org/sqlite/@latest` → v1.60.1 (2026-09-29, origin gitlab.com/cznic/sqlite), `modernc.org/libc/@latest` → v1.77.1, `v1.60.1.mod` requires `go 1.26.0`, `modernc.org/libc v1.77.1`, `golang.org/x/sys v0.48.0` — checked 2026-10-08 `[VERIFIED: proxy.golang.org]`
- Local codebase (read this session): `cache/cache.go`, `cache/concrete_cache.go`, `cache/errors.go`, `cache/options.go`, `cache/singleflight.go`, `cache/postgres/{postgres,options,schema,sweeper}.go`, `cache/redis/{redis,options}.go`, `cache/valkey/valkey.go`, `cache/mem/{mem,config,sweeper}.go`, `cache/doc.go`, tests/e2e/bench conventions, `Makefile`, `.testcoverage-quick.yml`, `.github/workflows/go.yml`, `go.mod`, `AGENTS.md` — `[VERIFIED: direct reads]`
- `go doc os.UserCacheDir` (stdlib semantics incl. error conditions) — `[VERIFIED: go doc]`
- Planning artifacts: `15-CONTEXT.md` (locked decisions), `REQUIREMENTS.md` (TTL-02, Out-of-Scope rows), `ROADMAP.md` (Phase 15 criteria), `PROJECT.md`, `.planning/config.json` (nyquist off, security on) — `[VERIFIED: direct reads]`

### Secondary (MEDIUM confidence)
- `sqlite.org/wal.html` + `sqlite.org/walformat.html` — last-connection close checkpoints and unlinks `-wal`/`-shm`; WAL requires shared memory (same-host only) `[CITED: https://sqlite.org/wal.html]`
- `go.dev/doc/database/manage-connections` and community sources (mattn issues, Go forum) — pool behavior, `SetMaxOpenConns(1)` for SQLite `:memory:` isolation `[CITED: https://go.dev/doc/database/manage-connections]`
- Milestone research: `SUMMARY.md`, `STACK.md`, `ARCHITECTURE.md`, `PITFALLS.md` (17 pitfalls; used with the requirement/decision-level corrections recorded in Findings 3–4) — cross-checked against primary sources above

### Tertiary (LOW confidence)
- Individual community reports on WAL sidecar retention (e.g., diesel #3962) used only to inform `require.Eventually` in tests, not as design input.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — version, pins, and go.mod delta verified against the module proxy, the pinned module source, and two local tidy/probe runs.
- Architecture / seams: HIGH — every seam read directly from the frozen `cache/` sources; the two non-obvious constraints (7-method interface, Optimizable adapter) are compile-level facts.
- Driver mechanics (DSN, pragmas, sidecars, checkpoint): HIGH — read from the pinned driver source and mechanically reproduced by a local probe; cross-OS transfer is the only (low-impact) assumption.
- Pitfalls: HIGH for the phase-scoped set — derived from milestone research (17 pitfalls) filtered to Phase 15, plus new empirically discovered ones (unconsumed-Rows deadlock, 3-column checkpoint scan, memory-mode `journal_mode="memory"`).

**Research date:** 2026-10-08
**Valid until:** ~2026-11-07 (30 days — stable module pins and frozen interfaces; re-verify only if `modernc.org/sqlite` or `cache/` root changes).

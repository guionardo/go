# Project Research Summary

**Project:** `github.com/guionardo/go` — milestone v1.8 SQLite Cache Backend (`cache/sqlite`)
**Domain:** Embedded persistent cache provider (sixth backend) added to a stdlib-first Go utility monorepo
**Researched:** 2026-10-08
**Confidence:** HIGH for stack and design; one LOW pocket — multi-process `SQLITE_BUSY` behavior (implementation-time spike required; see Gaps)

## Executive Summary

v1.8 adds the sixth cache backend — `cache/sqlite` — delivering durable, zero-infrastructure local caching for CLIs and small services, safe for multi-process access via WAL. All four research streams converge on the same design: **exactly one new direct dependency** (`modernc.org/sqlite` v1.60.1, the CGO-free pure-Go driver bundling SQLite 3.53.4, requiring Go ≥1.26 and `modernc.org/libc` v1.77.1 exact), everything else stdlib (`database/sql`, `encoding/json`, `os.UserCacheDir`). The provider structurally implements the existing unexported `cacher[K,V]` primitive inside a new package; `cache.NewConcreteCache` then supplies `Cache`, `BatchCache`, and singleflight `GetOrSet` for free, exactly like `cache/postgres`. No root-cache files change; no Docker for tests; no background goroutine anywhere.

The implementation recipe is small and now well specified: DSN-carried pragmas on every pooled connection (WAL, `busy_timeout(5000)`, `synchronous=NORMAL`, `_txlock=immediate`), a pool pinned to one connection for **both** file and `:memory:` modes (the single most important decision), idempotent schema DDL under a `BEGIN IMMEDIATE` bootstrap with a partial expiry index, absolute UnixNano TTLs with lazy delete on read plus a one-shot sweep on open, and one transaction per batch write. Multi-process semantics are verified against sqlite.org primary docs; bbolt and badger are explicitly rejected because both fail the multi-process requirement.

The dominant risks are mechanical-but-invisible failure classes, each with a cheap mandatory countermeasure: `:memory:` + `database/sql` silently creating multiple databases (pin pool to 1); pragmas landing on only one pooled connection or DSN dialects being ignored (per-connection DSN pragmas + read-back verification tests); WAL silently falling back on network filesystems (read-back `journal_mode` check + docs); `SQLITE_BUSY` escaping `busy_timeout` on deferred read→write upgrades (`_txlock=immediate` + bounded retry backstop); Windows temp-dir handle leaks (idempotent `Close` + `t.Cleanup(close)`); and unbounded file/WAL growth (documented high-water-mark + `journal_size_limit`). The one genuine LOW-confidence area — observable two-process `SQLITE_BUSY` behavior under the chosen pragma set — cannot be desk-researched and is carried as a mandatory implementation-time spike and a phase exit criterion.

## Key Findings

### Recommended Stack

One new direct dependency and nothing else at runtime. `modernc.org/sqlite` v1.60.1 (tagged 2026-09-29, verified current on `proxy.golang.org`; its `go.mod` requires Go ≥1.26.0, satisfied by the repo's Go 1.26.4) registers the standard `"sqlite"` `database/sql` driver, bundles SQLite 3.53.4 (past the 3.51.3 WAL-reset and 3.53.4 journal-rollback corruption fixes), and builds on all CI platforms with `CGO_ENABLED=0` — no gcc, no msys2, working cross-compilation and `-race`. The dependency tree grows (~10 small transitive modules) and cold builds are slower (~23 MB module; multi-MB generated source) — cache `GOMODCACHE`/`GOCACHE` in CI.

**Core technologies:**
- `modernc.org/sqlite` v1.60.1 — the only mature actively-maintained CGO-free SQLite driver; the milestone-mandated choice
- `modernc.org/libc` v1.77.1 (indirect, **pin exact**) — upstream mandate (cznic/sqlite#177): downstream must pin the driver's libc version or mysterious corruption follows; add a `go.mod` comment (optionally a CI assertion)
- `database/sql` — pool, per-connection prepared-statement cache, context-carrying queries; no wrapper library
- `encoding/json` + `fmt.Sprint` — value/key parity with all five existing providers (hard contract)
- `os.UserCacheDir` + `path/filepath` — default location (`$XDG_CACHE_HOME`/`~/.cache`, `~/Library/Caches`, `%LocalAppData%`; errors when undetermined — propagate, never fall back)
- Rejected: mattn/go-sqlite3 (needs CGO on all 3 CI OSes), ncruces/go-sqlite3 (Wasm memory overhead, Windows pooled-writer WAL bug history), zombiezen (no `database/sql` by design), bbolt/badger (fail multi-process), ORMs/query builders, `file::memory:?cache=shared`, `synchronous=OFF`, auto-`VACUUM`

### DSN Pragma Syntax — Contradiction Reconciled (Version-Qualified)

The four research files contradicted each other on modernc's DSN pragma syntax. The contradiction resolves on **driver version**, not on a general truth:

- **FEATURES.md / ARCHITECTURE.md (the `_pragma` claim):** "mattn-style `_journal_mode=WAL` shorthand is silently ignored" — this is true for modernc versions **before v1.55.0**, the likely context of the cited production incident (MaorBril/clauder PR #24). It is **not** a property of the pinned driver.
- **STACK.md (the shorthand claim):** STACK read the v1.60.1 module source from `$GOMODCACHE` and verified the validated shorthand keys (`_busy_timeout`, `_journal_mode`, `_synchronous`, …) were **added in v1.55.0 (2026-07-20)**, are validated against the mattn-compatible value set (a typo fails the open), and are applied in a fixed order with `busy_timeout` first. PITFALLS.md independently corroborates: v1.60.1 "now supports validated mattn-compatible shorthands."
- **PITFALLS.md on `_pragma`:** `_pragma` values remain **executed as verbatim SQL per connection** (multi-statement capable after `;`) — functional, but it carries SQL authority for any non-constant value.

**Resolution for v1.8:** build the provider-constructed DSN from the **validated shorthand keys**:

```
<plain-absolute-path>?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate
```

Both forms are functionally correct on v1.60.1, so this is a hygiene decision, not a correctness gap: shorthand is validated (fail-fast on typo) and cannot execute injected SQL, whereas `_pragma` executes verbatim. ARCHITECTURE.md's preference for `_pragma` ("most explicit, universally documented") is legitimate but carries the weaker safety property for zero benefit on the pinned version. Values are compile-time constants either way; never interpolate caller input into either form.

**Mandatory mechanical guard (settles any residual doubt, not because the question is open):** Phase 1 ships PITFALLS.md's verification tests — read back `PRAGMA journal_mode` (`QueryRow`, must be `wal` in file mode; warn/degrade on mismatch) and assert `busy_timeout`/`synchronous` take effect on the pooled connection. This test mechanically falsifies "silently ignored" on the pinned version and is the guard against future minor-version drift. Also reject/escape `?` and `#` in user-supplied paths (everything after the first `?` is driver DSN query territory).

### Expected Features

The milestone is a **provider-addition**, not a new architecture: `cacher[K,V]` conformance through `NewConcreteCache`, TTL semantics, key/value encoding, error taxonomy, and singleflight behavior are inherited contracts from v1.6 — the new work is the SQL implementation of seven primitive methods plus constructor/options. Scope is identical across all four files; no feature disputes.

**Must have (table stakes, P1):**
- `cacher[K,V]` conformance (Get/Set/Delete/Close funcs) via `NewConcreteCache` → returns `BatchCache[K,V]`; GetOrSet singleflight inherited, not reimplemented
- `MGet`/`MSet`/`MDel` with v1.6 batch semantics (only-found-keys, best-effort skip on undecodable, idempotent MDel, single TTL per MSet)
- TTL parity with postgres/mem `resolveTTL` (per-key >0 wins; 0/negative = never; absent → DefaultTTL; DefaultTTL≤0 = no expiry), stored as absolute UnixNano (NULL = none) so expiry survives restarts
- Expired entries never returned (`ErrMiss`); lazy delete on read + one-shot sweep on open (no goroutine)
- Multi-process WAL + `busy_timeout` DSN on every connection; `_txlock=immediate`
- Three location modes: default `os.UserCacheDir` + name; explicit path; `:memory:` (pool-pinned). **Option-API ambiguity to settle:** the milestone phrase "`:memory:` when path empty" collides with "empty = unset" for the default branch — recommend explicit `WithMemory()` (plus `:memory:` sentinel through `WithPath`), per ARCHITECTURE D1
- JSON/`fmt.Sprint`/err/ctx parity, `cache/sqlite:` error prefix, wrapped `cache.ErrMiss`
- Tests (`:memory:` + files, TTL, persistence/reopen, concurrency under `-race`), coverage gates, `doc.go`, runnable example

**Should have (differentiators, P2):**
- Zero-infrastructure durable cache — position as "mem that survives restarts; redis/postgres without the infra"
- `:memory:` as a drop-in fast/test double (no Docker)
- Multi-process shared cache file — the only embedded option that does this
- Transactional batch writes (single tx, one prepared statement, one commit)
- Optional storage lifecycle controls (checkpoint/optimize), periodic sweep, max-entries cap — only with evidence

**Defer / anti-features (explicitly NOT v1.8):**
- LRU/size eviction (read amplification); encryption at rest (impossible in pure-Go); network FS / multi-host (WAL cannot); cross-process invalidation pub/sub; custom codecs; delete-on-read of expired rows; split read/write pools; blanket retry-past-`busy_timeout`; shared-cache `:memory:` DSN; schema migration framework; periodic `VACUUM`/auto_vacuum default; background sweep loop; background write-serializer goroutine

### Architecture Approach

New package only — `cache/` root and all five providers untouched. `sqliteCache[K,V]` implements the seven unexported primitive methods structurally; `New` wraps it with `cache.NewConcreteCache`. Layout mirrors postgres minus the sweeper loop: `doc.go`, `options.go` (Config + functional options), `sqlite.go` (New, DSN builder, 7 methods, resolveTTL), `schema.go` (DDL constants), `sweep.go` (one-shot), plus tests/example.

**Major components and decisions:**
1. `New[K,V](opts...)` — config → location resolution (`os.UserCacheDir` default; `MkdirAll`; `UserCacheDir` errors propagate) → DSN build → `sql.Open("sqlite", dsn)` → pool pinning → schema DDL → WAL read-back → sweep-on-open → wrap
2. Pool: **`SetMaxOpenConns(1)` + `SetMaxIdleConns(1)` + zero conn lifetimes for BOTH modes** — serializes in-process access (no provider mutex needed), removes in-process `SQLITE_BUSY` races; critical for `:memory:` (per-connection private DBs; last close destroys it) and recommended by modernc docs
3. Pragmas via DSN only (never post-open `db.Exec`, which sticks to one pooled connection); `_txlock=immediate` makes every `BeginTx` a `BEGIN IMMEDIATE` (removes deferred read→write upgrade `SQLITE_BUSY_SNAPSHOT`/517)
4. Schema: `CREATE TABLE IF NOT EXISTS cache_entries (cache_key TEXT PRIMARY KEY, value TEXT NOT NULL, expires_at INTEGER) WITHOUT ROWID` + partial index `ON cache_entries(expires_at) WHERE expires_at IS NOT NULL`; wrap bootstrap in `BEGIN IMMEDIATE` + re-read for cold-start race safety (PITFALLS Pitfall 5); write `user_version=1` as the future migration hook; no migration framework. **auto_vacuum: recommend NONE** (3-of-4 sources: STACK, ARCHITECTURE, FEATURES anti-feature; PITFALLS pitches INCREMENTAL for TTL churn) — creation-time irreversible, so record the decision; escape path is an offline `VACUUM` if growth complaints materialize; set `journal_size_limit` (~4–8 MB) as a cheap WAL disk cap and document high-water-mark behavior
5. TTL/read path: SELECT → `sql.ErrNoRows` → wrap `ErrMiss`; expired row → best-effort conditional DELETE (`AND expires_at <= ?` — safe against concurrent refresh) → `ErrMiss`; unmarshal failure = error, not miss. MGet filters expired in SQL and does not lazy-delete (documented asymmetry)
6. Batch: MSet = one tx + one prepared upsert (`ON CONFLICT(cache_key) DO UPDATE`), fail-fast + rollback; MGet = one chunked `IN` query (chunk ~500; verify bundled `SQLITE_MAX_VARIABLE_NUMBER`, 32,766 modern / 999 legacy); MDel = tx + chunked `IN`; always fully consume/close `Rows` (checkpoint starvation)
7. Sweep: one-shot best-effort DELETE on open, `slog.Warn` on failure, never fails construction
8. Error contract: only `sql.ErrNoRows`/expiry → wrapped `ErrMiss`; DB failures never become misses; all errors prefixed `cache/sqlite:`; `Close` idempotent; decide/document `ErrClosed` behavior (mem does not return it — divergence risk, PITFALLS Pitfall 8)

### Critical Pitfalls

Full register in PITFALLS.md (7 critical, 6 moderate, 4 minor). Top 5 by severity for this milestone:

1. **`:memory:` + `database/sql` = multiple separate databases** — pin `MaxOpenConns(1)` + `MaxIdleConns(1)`, zero conn lifetimes; never use shared-cache as the default fix; test with concurrent goroutines asserting `db.Stats().OpenConnections == 1`
2. **Pragmas on one connection only / silently ignored** — all per-connection pragmas ride in the DSN; read back `journal_mode == wal` and assert `busy_timeout`/`synchronous` on the pooled connection; eager `Ping`/verification at `New` so bad DSN fails at construction
3. **WAL on network filesystems — corruption risk and silent fallback** — `PRAGMA journal_mode=WAL` silently returns the previous mode when shared memory is unavailable; check the read-back, error/degrade loudly, document local-disk-only (NFS/SMB/OneDrive/Dropbox)
4. **`busy_timeout` blind spots** — deferred read→write upgrade (`SQLITE_BUSY_SNAPSHOT`, code 517), WAL switch/last-close, checkpoint locking escape the busy handler; use autocommit single-statement writes + `_txlock=immediate` + a bounded, ctx-aware retry backstop (codes 5/6; 517 = restart transaction); never blanket-retry past timeout
5. **Cold-start schema bootstrap race** — two processes on a fresh DB both migrate; wrap version read + DDL + version write in one `BEGIN IMMEDIATE` and re-read inside the lock; concurrent-open test (4 goroutines + ideally 2 processes)

Also critical/moderate: Windows handle leaks (`Close` idempotent, `t.Cleanup(close)`, no permission assertions); DSN path escaping/name traversal (`?`/`#`/`..`, allow-list the cache name); `os.UserCacheDir` errors/missing dirs; file/WAL growth (freelist high-water, checkpoint starvation, `journal_size_limit`); coverage gates with **no override** for an embedded provider; key/TTL parity drift; conformance/bench registration; sidecar file lifecycle docs.

## Implications for Roadmap

**Recommended: a 3-phase structure** — the natural refinement of ARCHITECTURE.md's 2-phase grouping (steps 1–3 core; 4–5 batch + hardening) using PITFALLS.md's four severity tiers compressed to three. The 2-phase alternative (core vs batch+hardening) is viable but buries the milestone's only LOW-confidence unknown (the two-process spike) inside a fat hardening phase; giving batch + multi-process concurrency its own phase boundary lets the spike results shape the retry design and gives the security/coverage/docs sweep a clean home. Recommend 3 phases.

### Phase 1: Provider Foundation + Core Cache Semantics (Get/Set/Delete/TTL/Sweep)

**Rationale:** Every other phase builds on the constructor/open path, and all six Critical P1 pitfalls live here (memory pool, per-connection pragmas, network-FS WAL fallback, cold-start race, DSN escaping, location errors). Core primitives + TTL + sweep leave a *working* `cache.Cache[K,V]` with singleflight `GetOrSet` — a shippable vertical slice per GSD phase discipline.
**Delivers:** `go.mod` pin (`modernc.org/sqlite` v1.60.1 + explicit `modernc.org/libc` v1.77.1, decision recorded in PROJECT.md); package skeleton (`options.go`, `schema.go`, `sqlite.go`, `sweep.go`, `doc.go`); location modes (default UserCacheDir + name allow-listing/MkdirAll/error propagation; explicit path; `WithMemory()`); DSN builder with validated shorthand keys + `?`/`#` rejection; pool pinning both modes; `BEGIN IMMEDIATE` bootstrap DDL + `user_version=1` + auto_vacuum(NONE)+`journal_size_limit` decision; WAL read-back verification; per-connection pragma assertion tests; Get/Set/Delete/Close + `resolveTTL` + lazy delete + sweep-on-open; error taxonomy (`ErrMiss` only for miss/expiry).
**Addresses:** FEATURES P1 items 1, 3–7 — cacher conformance, TTL parity, expiry/sweep, persistence, location modes, parity contracts.
**Avoids:** Pitfalls 1, 2, 3, 5, 8, 9, 10, 14, 15.
**Research flag:** No deep research needed — patterns are documented and source-verified. Two discuss-phase decisions: (a) `WithMemory()` vs literal empty-path semantics (ARCHITECTURE D1 open question); (b) confirmation that the shorthand-key DSN + read-back test is the locked form. The read-back test is a first implementation task, not optional.

### Phase 2: Batch Surface + Multi-Process Concurrency Hardening

**Rationale:** Batch ops complete the interface surface (depends on Phase 1's SQL constants and TTL handling); multi-process safety is the milestone's headline promise but rests on the only LOW-confidence behavior — the two-process `SQLITE_BUSY` spike is this phase's exit criterion and its results pin the retry policy.
**Delivers:** `MGetFunc` (chunked `IN`, expired rows filtered, undecodable skipped), `MSetFunc` (one tx, one prepared upsert, fail-fast + rollback), `MDelFunc` (tx + chunked `IN`, idempotent) with chunk constant validated against the bundled `SQLITE_MAX_VARIABLE_NUMBER`; autocommit audit for single-statement writes; bounded ctx-aware retry helper (codes 5/6, 517 = restart, `interrupted` handling; ~3–5 attempts, 10–50 ms jittered backoff; **backstop only**, never blanket retry-past-timeout); **two-process spike** using the helper-process pattern (`os/exec` re-running the test binary — two `sql.DB` handles in one process do NOT reproduce POSIX cross-process locks); contention/`-race` tests; MGet/MSet/MDel at >1,000 keys.
**Addresses:** FEATURES P1 batch + multi-process items; MVP batch semantics.
**Avoids:** Pitfalls 4 (busy_timeout blind spots), 11 (batch limits/semantics), 12 residual (interrupt classification).
**Research flag:** **Yes — `/gsd-plan-phase --research-phase 2`.** This is the LOW-confidence area: observable residual `SQLITE_BUSY` probability, retry policy shape, modernc context-cancellation interrupt behavior (#198/#241), and WAL + `auto_vacuum` interaction under real multi-process churn.

### Phase 3: Delivery Hardening — CI Matrix, Coverage, Docs, Registrations

**Rationale:** Windows handle semantics, the no-override coverage gates, conformance registrations, and operational docs are delivery-critical work untouched by earlier phases; docs carry the constraints users will otherwise hit (network FS, sidecars, growth). The repo's per-commit coverage gate means tests accrue in every phase — this phase closes the last gaps and proves the gates.
**Delivers:** Windows-green tests (`t.Cleanup(close)` everywhere, idempotent `Close`, no permission assertions, temp-dir determinism); non-Docker `providerCase` registration in `cache/cache_e2e_test.go` + benchmark entry in `bench_test.go`; `make coverage-quick` green with **no `.testcoverage-quick.yml` override** (pkg ≥80 / file ≥70 / total ≥75); `go mod tidy` diff reviewed + `govulncheck` clean; `doc.go` statements (WAL local-disk-only, sidecar lifecycle, growth high-water-mark + delete-to-reclaim, `:memory:` single-connection serialization), `example_test.go`, README/AGENTS table rows; PITFALLS "Looks Done But Isn't" checklist sweep; `make quality-report` before tag.
**Addresses:** FEATURES project Definition of Done; MVP tests/docs items.
**Avoids:** Pitfalls 6, 7 (documentation half), 12 (quality half), 13, 16, 17.
**Research flag:** No — standard project-hardening patterns, fully specified in PITFALLS.md.

### Phase Ordering Rationale

- **Constructor first:** the DSN/pool/schema decisions (all creation-time or irreversible: `auto_vacuum`, expiry index) are prerequisites for every behavior and own the Critical pitfalls
- **Primitives before batch:** batch methods reuse Phase 1's SQL constants and TTL resolution; splitting them avoids blocking the working `Cache` slice on batch complexity
- **Concurrency/spike before declaring the milestone promise:** the milestone *is* multi-process safety; the LOW-confidence behavior must be measured (Phase 2), not assumed, and the retry policy derives from it
- **Delivery hardening last but enforced throughout:** coverage gates run per commit, so each phase ships green; Phase 3 aggregates Windows/docs/registrations/quality-report
- **Frozen interfaces:** no phase may touch `cache/` root or existing providers (v1.6 D-02) — the compile-time contract is exercised by `New`'s return

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Driver/pragma/WAL facts verified against primary sources: module proxy, v1.60.1 source in `$GOMODCACHE` (DSN docs, apply order, `StrictPragmas`, changelog), sqlite.org docs. Version facts date-stamped 2026-10-08 |
| Features | HIGH (MEDIUM pockets) | HIGH for engine/driver behavior and interface-parity requirements (official docs + local code inspection); MEDIUM for Go-ecosystem TTL/sweep/pool idioms (consensus across 4+ independent implementations). One stale claim (DSN shorthand) version-qualified and reconciled above |
| Architecture | MEDIUM | Integration wiring HIGH (direct code inspection); SQLite mechanics MEDIUM (docs + independent implementations converging); multi-process `SQLITE_BUSY` LOW pending spike |
| Pitfalls | HIGH (SQLite semantics), MEDIUM (driver-specific) | HIGH for SQLite semantics from official docs/forum; MEDIUM for modernc version-specific and Windows ecosystem reports (cross-checked community incidents) |

**Overall confidence:** HIGH for the design and stack; the single LOW-confidence pocket (observable multi-process contention behavior) is isolated, scheduled, and does not block design — it blocks only the claim "multi-process safe" until the Phase 2 spike passes.

### Gaps to Address

- **Two-process `SQLITE_BUSY` spike (mandatory, implementation-time):** run two OS processes writing one file through modernc with WAL + `busy_timeout`; measure whether/when errors escape the timeout; confirm the retry helper absorbs residual; use the helper-process pattern (same-process handles don't reproduce cross-process locks). Phase 2 exit criterion. (questions.md #4)
- **DSN shorthand behavior:** settled at source level (v1.60.1 validated shorthand support; the "silently ignored" claim is pre-v1.55.0) — still ship the Phase 1 pragma read-back tests as the mechanical guard; do not ship on assumption regardless of DSN form.
- **`WithPath("")` / `WithMemory()` option-API ambiguity:** milestone's "`:memory:` when path empty" cannot be taken literally alongside "empty = unset" for the default branch; settle in Phase 1 discussion (recommend explicit `WithMemory()` + `:memory:` sentinel).
- **`auto_vacuum` decision (creation-time, irreversible):** recommend NONE + documented high-water-mark + offline-`VACUUM` escape vs PITFALLS' INCREMENTAL pitch; lock in Phase 1 and record it; add `journal_size_limit` (~4–8 MB).
- **Context-cancellation interrupt behavior:** modernc #198/#241 show possible pooled-connection poisoning (`interrupted (9)`) after cancel; verify pinned-version behavior under `-race` stress or classify `interrupted` as retryable.
- **`ErrClosed` semantics:** `mem` does not return it today; decide and test the sqlite behavior explicitly (driver error wrapped vs sentinel) to avoid divergence.
- **Retry scope reconciliation:** PITFALLS requires a bounded retry backstop; FEATURES lists blanket retry-past-timeout as an anti-feature. Reconciled: retry only for the non-timeout-covered cases (WAL switch/bootstrap, 517 restart, `interrupted`), never as a blanket policy — confirm exact policy when the spike reports.
- **Chunk constant (500) and `journal_size_limit` value:** conservative defaults; validate against the bundled build's limits/config in Phase 2 with benchmarks where cheap.

## Sources

### Primary (HIGH confidence)
- `modernc.org/sqlite` v1.60.1 module source read from `$GOMODCACHE` (`driver.go`, `sqlite.go`, `doc.go`, `CHANGELOG.md`, `go.mod`) — DSN keys/order, `StrictPragmas`, platform matrix, libc pin, pool guidance
- `sqlite.org` primary documentation — wal.html, pragma.html, inmemorydb.html, lang_upsert.html, wal-lock.md, rescode.html, busy_timeout C API, howtocorrupt.html, useovernet.html, lang_vacuum.html, lang_transaction.html
- `proxy.golang.org` module metadata (versions, dates, `go.mod` requirements) — checked 2026-10-08
- Go stdlib docs — `go doc os.UserCacheDir`; database/sql connection-management guidance
- Local code inspection — `cache/{cache,concrete_cache,singleflight,errors}.go`, `cache/postgres/*`, `cache/mem/*`, `cache/cache_e2e_test.go`, `cache/bench_test.go`, `.testcoverage-quick.yml`, `.github/workflows/go.yml`, `go.mod`, `AGENTS.md`

### Secondary (MEDIUM confidence)
- `pkg.go.dev/modernc.org/sqlite` v1.60.1 package docs — DSN syntax, pool warnings, native-vs-C performance
- Field reports cross-checked: MaorBril/clauder PR #24 and opentalon PR #314 (silently ignored mattn-style DSN keys), trip2g (517 `SQLITE_BUSY_SNAPSHOT`/`_txlock`), hollis-labs/go-sqlite, mattn FAQ #204/#906/#511/#1205 (`:memory:` pooling, Windows handles), cznic/sqlite #177/#198/#241 (libc pin, interrupt poisoning), Rails PR #57076 (`auto_vacuum=incremental`), NetBird #6701 + checkpoint-starvation write-ups
- Independent benchmarks: `go-sqlite-benchmark-mattn-vs-modernc` (17–54% directional), blinki-io/dingo PR #2499 (fixed-shape prepared SQL)
- TTL/sweep pattern consensus: requests-cache (Python), cameo sqlite-cache (Rust), sqlite-kv (TS)
- Competitor docs: etcd-io/bbolt (exclusive lock, single-process), dgraph badger (directory lock)

### Tertiary (LOW confidence)
- Single-source ecosystem claims used directionally only (individual benchmark figures; ncruces Windows WAL-index report used to reject the alternative, not to make positive design claims)

---
*Research synthesized: 2026-10-08*
*Ready for roadmap: yes — recommended structure: 3 phases; Phase 2 requires deeper research/spike*

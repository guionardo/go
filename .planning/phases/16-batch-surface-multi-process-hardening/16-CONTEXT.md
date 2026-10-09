# Phase 16: Batch Surface + Multi-Process Hardening - Context

**Gathered:** 2026-10-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete the `BatchCache` surface on `cache/sqlite` with v1.6 semantics — `MGet` (chunked IN queries, silent skips), `MSet` (single atomic upsert transaction, one TTL), `MDel` (chunked, idempotent) — replace the Phase 15 placeholders; and prove the milestone's headline promise: two OS processes share one cache file under WAL without corruption, with contention bounded by `busy_timeout`. Includes fixing DI-15-01 (concurrent first-open WAL conversion race) and a CI two-process E2E (CONC-05).

Out of scope: retry middleware beyond `busy_timeout`, background write-serializers, cross-process invalidation pub/sub, network-filesystem support (all milestone-locked).

</domain>

<decisions>
## Implementation Decisions

### Batch Mechanical Details
- **D-01:** MGet/MDel chunk size is a **fixed internal constant of 100 keys per chunk** — no `WithChunkSize` option, no new API surface. Values far below SQLite's default variable limit; adjustable later without breaking callers. Replaces the open "chunk size" item from STATE.md (v1.8 planning).
- **D-02:** MGet/MDel inputs are **deduplicated before chunking**; an empty key list is a **no-op** (MGet returns an empty map without issuing any query; MDel returns nil) — **Reversibility:** reversible.
- **D-03:** MGet error policy — `cache.BatchCache.MGet` returns `map[K]V` with no error channel, so a failed chunk is **logged (slog warn) and skipped**; remaining chunks still run; found-so-far is returned. Undecodable entries stay silently skipped per BATCH-01.
- **D-04:** MSet runs as **one `BEGIN IMMEDIATE`…`COMMIT` transaction** with the single prepared upsert; all values are **marshaled up-front before the first write**, and ANY error (marshal, SQL, context) rolls back the whole batch (BATCH-02 atomicity). One TTL applies to the batch (per `BatchCache.MSet` signature).
- **D-05:** MDel chunk failures return the wrapped error (partial delete is acceptable — MDel is idempotent; caller may retry).

### Contention, Retry & DI-15-01
- **D-06:** DI-15-01 fix: **bounded open retry** — `New` retries the fresh-file open when the per-connection `PRAGMA journal_mode=WAL` conversion hits immediate `SQLITE_BUSY` (busy handler bypassed), bounded by `busy_timeout` with a tiny backoff. `New` stays never-failing: ops defer the wrapped error if retries exhaust. The concurrent-first-open test stays and must be deterministic green (~72% repro evidence from Phase 15 verification).
- **D-07:** Beyond the open window, **no provider-side retries**: ordinary BEGIN contention surfaces as the bounded `busy_timeout` error (`_txlock=immediate`) — milestone's no-retry-middleware decision holds; Checkpoint/Vacuum get no extra retry either.
- **D-08:** **`journal_size_limit` set to 64 MB via DSN pragma** (`_pragma=journal_size_limit(67108864)` in file mode) to bound passive WAL growth between checkpoints in long-running processes — **Reversibility:** costly — silently changes behavior for existing cache files if later adjusted; zero-value/disabled path keeps SQLite's unlimited default, no sentinel semantics (mirrors `WithAutoCheckpoint`).

### Two-Process Proof & CI
- **D-09:** The CONC-05 two-process E2E (helper-process pattern: test binary re-executes itself via env flag) runs on **all three CI OSes** (ubuntu/macos/windows) — Windows WAL sidecar/handle semantics get the real proof.
- **D-10:** E2E covers **contention + crash recovery**: (A) both processes concurrently Set/Get/Delete (+ a batch op) asserting no corruption and correct values — the CONC-01 exit criterion — and (B) kill-mid-write: one process dies while the other holds a write lock; the survivor continues and the file reopens cleanly.
- **D-11:** The two-process spike (folded todo below) runs first as the phase exit criterion; its results confirm the retry policy before the E2E hardens it.

### the agent's Discretion
- MGet result ordering (map return — order follows query rows, no guarantee needed).
- Spike test fixture details (scenario timing, iteration counts, helper-process plumbing) and how the flake-quarantine is disposed once D-06 lands.
- Whether `WithAutoCheckpoint`/`journal_size_limit` interact (both DSN pragmas; keep additive).

### Folded Todos
- **Two-process WAL spike** (`2026-10-08-spike-modernc-sqlite-wal-with-two-concurrent-processes.md`, pending) — original problem: `SQLITE_BUSY` behavior under the chosen pragma set is unverified; folded as the Phase 16 exit criterion (CONC-01): spike two OS processes sharing one cache file under WAL, then bake the proven policy into the provider and the CI E2E.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase & requirements
- `.planning/ROADMAP.md` — Phase 16 section: goal, dependencies, success criteria, execution order
- `.planning/REQUIREMENTS.md` — BATCH-01..03, CONC-01..05 (v1) + Out-of-Scope table
- `.planning/PROJECT.md` — milestone constraints (Go 1.26, minimal deps, 3-OS CI, coverage gates)

### Phase 15 artifacts (what this phase extends)
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-CONTEXT.md` — locked D-01..D-12 (pool pinning, DSN pragmas, filter-only reads, no user_version)
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-RESEARCH.md` — driver mechanics, `SQLITE_BUSY`/WAL claims, two-process spike guidance, chunk/UPSERT shapes
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-PATTERNS.md` — analog files/line ranges for batch + sweep patterns
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-VERIFICATION.md` — deferred evidence block (DI-15-01 at ~72% repro; Phase 16 "addressed_in")
- `.planning/phases/15-provider-foundation-core-cache-semantics/deferred-items.md` — DI-15-01 record: root cause (DSN isolation + per-connection WAL conversion race), bounded-retry recommendation

### Code (integration contract — frozen)
- `cache/cache.go` — `BatchCache[K, V]` interface: `MGet(ctx, keys...) map[K]V`, `MSet(ctx, items map[K]V, ttl ...time.Duration) error`, `MDel(ctx, keys...) error` (frozen; no error channel on MGet)
- `cache/concrete_cache.go` — `cacher[K, V]` 7-method primitive: `MGetFunc/MSetFunc/MDelFunc` are the batch seams this phase implements
- `cache/mem/mem.go` — `MGetFunc/MSetFunc/MDelFunc` semantics precedent (best-effort, skip semantics)
- `cache/sqlite/*.go` — Phase 15 provider; batch placeholders (`15-01-PLAN.md` Task 3) to be replaced in place
- `.opencode/skills/spike-findings-go/SKILL.md` — singleflight/cache integration constraints (GetOrSet parity, delete-vs-inflight)

### Severity evidence
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-REVIEW.md` — WR-01 = DI-15-01 independent reproduction (10/60, 4/20)
- `.planning/phases/15-provider-foundation-core-cache-semantics/15-REVIEW-DISPOSITION.md` — open findings ledger

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cache/sqlite/schema.go` — SELECT/UPSERT SQL blocks with bound positional params; `SelectSQL` filter pattern (expiry + chunked IN will extend these)
- `cache/sqlite/dsn.go` — DSN builder with validated shorthand keys; the `_pragma=` prefix pattern (used for `wal_autocheckpoint`) extends cleanly to `journal_size_limit`
- `cache/mem/mem.go` + `cache/postgres/postgres.go` — MGet/MSet/MDel provider precedent (mem: loops over keys; postgres: transaction + upsert shape)
- `cache/sqlite/sqlite_internal_test.go` — `TestResolveTTL`, `TestDSNBuildDSN` (extend, don't re-declare — single-ownership rule)

### Established Patterns
- Functional options with zero-value = SQLite default (D-07 of Phase 15: `WithAutoCheckpoint`)
- TDD (`type: tdd` tasks; plan-scoped commit messages matching the gate regex)
- Sentinel errors + `%w` wrapping; `cache/sqlite:` prefix; never-failing `New` with deferred `initErr`
- `make coverage-quick` before every commit; co-located `_test.go` + testify

### Integration Points
- `cache/sqlite/sqlite.go` — the `cacher` primitive wiring: replace placeholder `MGetFunc/MSetFunc/MDelFunc` (loop-based stubs from 15-01) with real implementations
- `cache/sqlite/options.go` — option plumbing if any tuning surfaces (none planned per D-01/D-08 defaults)
- CI: `.github/workflows/go.yml` — the two-process E2E runs on the existing 3-OS matrix (D-09)

</code_context>

<specifics>
## Specific Ideas

- Helper-process E2E shape: `TestMain` + `GO_WANT_HELPER_PROCESS`-style env flag re-executes the test binary as the second OS process against a temp database; both processes drive Set/Get/Delete/batch ops concurrently; crash scenario kills via `os.Exit`/SIGKILL mid-write.
- The DI-15-01 open retry should reuse the existing `busy_timeout` value (5000 ms) as its bound — no new tunables.
- BATCH-01's "silently skipped" also covers keys whose chunk query succeeds but a row's JSON fails to decode (log at debug, not warn — decode errors are data hygiene, not system errors; keep warn for actual DB errors per D-03).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

### Reviewed Todos (not folded)
None — the single matching todo (two-process WAL spike) was folded.

</deferred>

---

*Phase: 16-Batch Surface + Multi-Process Hardening*
*Context gathered: 2026-10-08*
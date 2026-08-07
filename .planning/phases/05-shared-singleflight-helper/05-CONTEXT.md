# Phase 5: Shared singleflight helper - Context

**Gathered:** 2026-08-06
**Status:** Ready for planning

<domain>
## Phase Boundary

The `cache` package gains an **exported, shared `SingleflightGetOrSet[K, V]` helper** that wraps only the setter for the GetOrSet miss path. It dedups concurrent misses on the same key (setter runs once), recovers panicking setters into typed errors, offers both blocking (`Do`) and cancel-aware (`DoChan`) entry points with a ctx-aware setter, double-checks Get before computing (clobber guard), and passes TTL through untouched.

Phase 6 wires all 5 providers to this helper. This phase delivers the helper itself plus a full public test suite in the `cache` package.

</domain>

<decisions>
## Implementation Decisions

### Helper Visibility & Placement
- **D-01:** Export `SingleflightGetOrSet[K, V]` from the `cache` package (public API). Providers (`cache/mem`, `cache/redis`, `cache/valkey`, `cache/memcache`, `cache/postgres`) import it directly — an unexported helper is unreachable from subpackages.
- **D-02:** Providers embed the struct (`sf SingleflightGetOrSet[K,V]`) and call a `Do` method — state (the `singleflight.Group`) persists per-provider so dedup works across calls. No free-function form (a free function would create a fresh group per call and lose dedup).

### Setter ctx Propagation
- **D-03:** **Breaking change accepted in v1.6:** the `Cache` interface `GetOrSet` setter signature changes from `func() (V, error)` to `func(context.Context) (V, error)`. Callers are updated accordingly.
- **D-04:** The setter runs only in the leader's goroutine, so it receives the **leader's ctx**. Waiters' contexts only gate their own `DoChan` select — they never reach the setter.
- **D-05:** Leader cancel propagates to the setter: if the leader's ctx is canceled mid-setter, `ctx.Err()` becomes the GetOrSet result.
- **D-06:** Helper uses a single explicit ctx for both waiter-select and setter (no separate setterCtx parameter).
- **D-07:** The helper's own internal double-check Get also uses this ctx.

### Canceled-waiter Semantics
- **D-08:** A canceled waiter returns the **shared value if it is already ready** (leader finished) — non-blocking `select` with `default` on the result channel before honoring `ctx.Done()`. Return the cancellation error only if the value isn't ready.
- **D-09:** Define a sentinel `cache.ErrCanceled`.
- **D-10:** `cache.ErrCanceled` **wraps the underlying `ctx.Err()`**, so `errors.Is(err, cache.ErrCanceled)` AND `errors.Is(err, context.Canceled)` both hold.
- **D-11:** The leader path never delivers `ErrCanceled` (its own cancellation surfaces as the setter's `ctx.Err()`).

### Helper API Shape
- **D-12:** Both methods: `Do(ctx, key, get, set, setter, ttl...)` (blocking, for short setters) and `DoChan(...)` (cancel-aware, for HTTP paths). Identical signatures; both return `(V, error)`. DoChan does not leak the `singleflight.Result` channel.
- **D-13:** Group key is `fmt.Sprint(key)` (validated in spike 005; `K` is comparable; providers are type-parameterized so no cross-type key collisions).
- **D-14:** The helper returns **raw errors** — panics recovered → `cache.PanicError`, canceled waiters → `cache.ErrCanceled`, setter errors verbatim. Provider prefix decoration (redis/valkey) belongs in Phase 6.
- **D-15:** `fmt` is the only non-provider dependency helper needs beyond `golang.org/x/sync` (already a dependency v0.22.0).

### Panic Policy
- **D-16:** Always recover panicking setters in the wrapper and return an error to every waiter — never re-panic (no identifiable "leader" from the caller's view).
- **D-17:** Surface recovered panics as a **typed `cache.Panic` error** wrapping the recovered value and stack, inspectable via `errors.As` (mirrors `singleflight`'s `*panicError`).

### TTL Ownership Boundary
- **D-18:** Helper passes `ttl...` **straight through** to the set closure. Provider `Set` already applies its provider-level default when TTL is empty — no TTL-resolution logic added to shared code.
- **D-19:** When concurrent waiters pass different TTLs, the **leader's ttl wins** (only the leader calls Set). Document this.

### Public Helper Testing
- **D-20:** Full public test suite in the `cache` package covering all validated behaviors: setter runs once for N concurrent misses, `shared` is a global flag, panic recovery via `cache.Panic`, canceled-waiter value-if-ready semantics, `ErrCanceled`, clobber guard (double-check Get), TTL passthrough, and race-detector-clean runs.
- **D-21:** Tests use an **in-package stub** (tiny in-memory get/set closures) so the helper is verified without coupling to any provider implementation.
- **D-22:** Include godoc `Example` functions for both `Do` and `DoChan` since the helper is now public API.

### the agent's Discretion
- None — all surfaced gray areas were decided. Implementation details (naming of internal helper fns, test style) remain open to the planner/executor.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & Requirements
- `.planning/ROADMAP.md` §Phase 5 — Phase goal, requirements (SF-02, SF-05, SF-06, SF-08), success criteria
- `.planning/REQUIREMENTS.md` — SF-02, SF-05, SF-06, SF-08 (this phase); SF-01/03/04/07/09 (Phase 6)

### Validated spike blueprint (MUST read)
- `.opencode/skills/spike-findings-go/references/singleflight-cache.md` — Full validated implementation blueprint: setter-only wrap, DoChan variant, panic recovery, clobber guard, error-glue notes, benchmark numbers. Sources in `.opencode/skills/spike-findings-go/sources/005-singleflight-placement/` etc.

### Existing code (integration anchors)
- `cache/cache.go` — `Cache[K,V]` interface; `GetOrSet` signature changes here (ctx-aware setter)
- `cache/errors.go` — add `ErrCanceled` here; consider `Panic` error type placement
- `cache/options.go`, `cache/doc.go` — package conventions to match
- `cache/*/` provider GetOrSet bodies — replaced in Phase 6; referenced here as the consuming call sites
- `.planning/CONVENTIONS.md` (codebase maps) — Go code style, package conventions
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `golang.org/x/sync/singleflight` v0.22.0 — already a dependency; provides `Do`, `DoChan`, `Forget`, `Result{Val, Err, Shared}`
- `cache/errors.go` — existing error file; natural home for `ErrCanceled` new sentinel
- Provider `Get`/`Set` method signatures — directly passable as closures to the helper

### Established Patterns
- Provider subpackages each expose `func (c *Cache[K,V]) GetOrSet(ctx, key, setter, ttl...)` — identical miss→setter→Set shape across all 5 (see `cache/redis/redis.go:86`, `cache/valkey/valkey.go:106`, `cache/memcache/memcache.go:129`, `cache/postgres/postgres.go:128`)
- redis (`cache/redis:`) and valkey (`cache/valkey:` + initErr guard) wrap setter errors; mem/memcache/postgres return raw — this glue stays provider-side (Phase 6)
- Testify (`require`/`assert`) throughout; `t.Parallel()` used except in global-state-mutating tests (AGENTS.md)

### Integration Points
- `cache.SingleflightGetOrSet[K,V]` exported struct + `Do`/`DoChan` methods — consumed by all 5 providers in Phase 6
- `Cache` interface `GetOrSet` signature change is a public break — ripple to callers/examples/E2E tests (~`cache_test.go`, `example_test.go`, `cache_e2e_test.go`)
</code_context>

<specifics>
## Specific Ideas

- Spikes 002–009 (2026-08-06) validated every behavior locked here; `references/singleflight-cache.md` is the ground truth for HOW.
- The DoChan variant was measured at ~15 ms cancel latency vs ~300 ms blocking (spike 008); herd benchmark ~2× faster, ~half allocs (spike 009).

</specifics>

<deferred>
## Deferred Ideas

- **Reviewed Todos (not folded):** todos for `config` HTTP endpoint, `httptest_mock` ServeMux routing, and Windows header-matching were reviewed in `cross_reference_todos`; none match Phase 5's scope (all non-cache). They remain pending in `.planning/todos/pending/`.

</deferred>

---

*Phase: 5-Shared singleflight helper*
*Context gathered: 2026-08-06*
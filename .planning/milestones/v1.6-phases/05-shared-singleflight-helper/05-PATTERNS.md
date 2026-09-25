# Phase 5: Shared singleflight helper — Pattern Map

**Mapped:** 2026-08-06
**Files analyzed:** 18 (2 new, 16 modified)
**Analogs found:** 17 / 18

Ground truth sources for this map:
- `.planning/phases/05-shared-singleflight-helper/05-CONTEXT.md` (decisions D-01..D-22)
- `.opencode/skills/spike-findings-go/references/singleflight-cache.md` (validated spike blueprint — MUST be re-read by the planner/executor)
- Spike sources: `sources/005-singleflight-placement/main.go`, `sources/008-singleflight-context/main.go`

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cache/singleflight.go` (new) | utility (shared helper) | CRUD (miss-path dedup, request-response) | `sources/005-singleflight-placement/main.go:26-62` + `sources/008-singleflight-context/main.go:99-109` | exact (validated prototype) |
| `cache/singleflight_test.go` (new) | test | CRUD (concurrent) | `cache/mem/mem_test.go:103-141` + `sources/005-singleflight-placement/main.go:64-109` | exact |
| `cache/cache.go` | contract (interface) | CRUD | itself (interface file, only line 23 changes) | exact |
| `cache/errors.go` | utility (error defs) | n/a | itself (existing sentinels lines 7-12); `Panic` mirrors `x/sync singleflight.panicError` | exact / external |
| `cache/doc.go` | config (package docs) | n/a | itself (sentinel list lines 31-34) | exact |
| `cache/cache_test.go` | test | n/a (sentinels) | itself (`TestCacheSentinelErrors` lines 37-53) | exact |
| `cache/example_test.go` | test (godoc examples) | n/a | itself (`ExampleWithDefaultTTL` lines 10-14) | exact |
| `cache/cache_e2e_test.go` | test (E2E) | CRUD | itself (`get_or_set_*` subtests lines 76-108) | exact |
| `cache/mem/mem.go` | provider (controller-ish) | CRUD | itself (`GetOrSet` lines 84-103) | exact |
| `cache/redis/redis.go` | provider | CRUD | itself (`GetOrSet` lines 85-104) | exact |
| `cache/valkey/valkey.go` | provider | CRUD | itself (`GetOrSet` lines 105-129) | exact |
| `cache/memcache/memcache.go` | provider | CRUD | itself (`GetOrSet` lines 128-147) | exact |
| `cache/postgres/postgres.go` | provider | CRUD | itself (`GetOrSet` lines 127-146) | exact |
| `cache/mem/mem_test.go` | test | CRUD | itself (`GetOrSet` tests lines 103-141) | exact |
| `cache/redis/redis_test.go` | test | CRUD | itself (`get_or_set_computes` lines 65-76) | exact |
| `cache/valkey/valkey_test.go` | test | CRUD | itself (`get_or_set_computes` lines 65-76) | exact |
| `cache/memcache/memcache_test.go` | test | CRUD | itself (`get_or_set_computes` lines 60-69) | exact |
| `cache/postgres/postgres_test.go` | test | CRUD | itself (`get_or_set_computes` lines 82-88) | exact |

**Critical ripple note for the planner:** D-03 breaks the `Cache` interface setter signature. The 5 provider methods AND all `GetOrSet` test call sites MUST be updated in this phase (not Phase 6) to keep the build green — the compile-time assertions `var _ cache.Cache[string, any] = (*Cache[string, any])(nil)` (`mem.go:130`, `memcache.go:178`, `postgres.go:177`) fail otherwise. Only the *body delegation to the helper* is Phase 6.

---

## Pattern Assignments

### `cache/singleflight.go` (new — utility, CRUD miss-path)

**Analog:** `sources/005-singleflight-placement/main.go:26-62` (validated Do prototype) + `sources/008-singleflight-context/main.go:99-109` (DoChan select). Replace the spike's `setter func() (V, error)` with `setter func(context.Context) (V, error)` per D-03/D-04 (setter receives the leader's ctx — D-04; single explicit ctx for waiter-select AND setter — D-06; double-check Get uses it too — D-07).

**Imports pattern** (from spike 005 lines 3-14 and `cache/mem/mem.go:3-10`; `fmt` is the only extra dependency beyond `golang.org/x/sync` — D-15):
```go
import (
    "context"
    "fmt"
    "time"

    "golang.org/x/sync/singleflight"
)
```

**Struct + `Do` core pattern** (exported per D-01; embedded per-provider per D-02; adapt spike 005:26-62 — setter now takes ctx, group key `fmt.Sprint(key)` per D-13):
```go
// SingleflightGetOrSet dedups concurrent GetOrSet misses on the same key.
// Providers embed a value of this type; state persists per-provider (D-02).
type SingleflightGetOrSet[K comparable, V any] struct {
    group singleflight.Group
}

// Do blocks until the value is available. Raw errors only (D-11/D-14).
func (s *SingleflightGetOrSet[K, V]) Do(
    ctx context.Context,
    key K,
    get func(context.Context, K) (V, error),
    set func(context.Context, K, V, ...time.Duration) error,
    setter func(context.Context) (V, error),
    ttl ...time.Duration,
) (V, error) {
    var zero V
    if v, err := get(ctx, key); err == nil {
        return v, nil                    // fast-path Get stays OUTSIDE the group (spike blueprint)
    }

    v, err, _ := s.group.Do(fmt.Sprint(key), func() (any, error) {
        // double-check Get: another leader may have Set while we queued (SF-08 / D-clobber-guard)
        if v, err := get(ctx, key); err == nil {
            return v, nil
        }
        computed, err := s.callSetter(setter, ctx)  // recovered: panics → *cache.Panic (SF-05)
        if err != nil {
            return zero, err
        }
        if err := set(ctx, key, computed, ttl...); err != nil {  // Set INSIDE the fn (runs once)
            return zero, err
        }
        return computed, nil
    })
    if err != nil {
        return zero, err
    }
    return v.(V), nil
}
```

**`DoChan` core pattern** (from spike 008:100-109 + D-08 value-if-ready semantics; do NOT leak the `Result` channel — D-12; wrap waiter cancellation with `cache.ErrCanceled` — D-09/D-10):
```go
// DoChan is the cancel-aware variant. A canceled waiter abandons its own wait
// promptly (D-08): first try the result channel non-blocking (shared value wins
// if the leader already finished), then block on ctx vs result.
func (s *SingleflightGetOrSet[K, V]) DoChan(
    ctx context.Context,
    key K,
    get func(context.Context, K) (V, error),
    set func(context.Context, K, V, ...time.Duration) error,
    setter func(context.Context) (V, error),
    ttl ...time.Duration,
) (V, error) {
    var zero V
    if v, err := get(ctx, key); err == nil {
        return v, nil
    }

    ch := s.group.DoChan(fmt.Sprint(key), func() (any, error) {
        // identical body to Do's fn: double-check Get → callSetter → set
        ...
    })

    select {                       // non-blocking: value already ready wins (D-08)
    case res := <-ch:
        if res.Err != nil {
            return zero, res.Err
        }
        return res.Val.(V), nil
    default:
    }

    select {                       // blocking: caller cancel abandons the wait (SF-06)
    case <-ctx.Done():
        return zero, fmt.Errorf("%w: %w", ErrCanceled, ctx.Err())  // both errors.Is checks hold (D-10)
    case res := <-ch:
        if res.Err != nil {
            return zero, res.Err
        }
        return res.Val.(V), nil
    }
}
```

**Panic recovery helper** (SF-05 / D-16/D-17 — recover inside the singleflight fn so every waiter gets the typed error; never re-panic; `Do`'s caller sees `cache.Panic`, not a replayed panic):
```go
// callSetter runs the setter with panic recovery. A recovered panic becomes a
// *cache.Panic wrapping the value and stack (mirrors x/sync's *panicError).
func (s *SingleflightGetOrSet[K, V]) callSetter(setter func(context.Context) (V, error), ctx context.Context) (v V, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = &Panic{Value: r, Stack: debug.Stack()}   // trim first goroutine line like x/sync does
        }
    }()
    return setter(ctx)
}
```

**Documentation to include** (godoc comments for all exported symbols — AGENTS.md): leader's ttl wins when concurrent waiters pass different TTLs (D-19); stale-window semantics (a joining caller gets the result "as of when the leader started"); blocking `Do` is context-blind, `DoChan` is the HTTP-path choice.

---

### `cache/singleflight_test.go` (new — test, CRUD/concurrent)

**Analog:** `cache/mem/mem_test.go:103-141` (closure stubs + `t.Parallel()`) and `sources/005-singleflight-placement/main.go:64-109` (N-concurrent-callers harness with `sync.WaitGroup` + `atomic.Int64` counter). No provider coupling — D-21 (in-test get/set closures over a local map).

**Test harness pattern** (spike 005:64-109 + mem_test conventions; setter-runs-once assertion):
```go
func TestSingleflightGetOrSet_SetterRunsOnce(t *testing.T) {   // D-20: N concurrent misses
    t.Parallel()

    var mu sync.Mutex
    store := map[string]string{}
    var setterRuns atomic.Int64
    sf := &cache.SingleflightGetOrSet[string, string]{}

    get := func(ctx context.Context, k string) (string, error) {
        mu.Lock(); defer mu.Unlock()
        if v, ok := store[k]; ok { return v, nil }
        return "", cache.ErrMiss
    }
    set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
        mu.Lock(); defer mu.Unlock()
        store[k] = v
        return nil
    }
    setter := func(context.Context) (string, error) {
        setterRuns.Add(1)
        time.Sleep(3 * time.Millisecond)
        return "computed", nil
    }

    const callers = 50
    var wg sync.WaitGroup
    results := make([]string, callers)
    for i := 0; i < callers; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            v, err := sf.Do(t.Context(), "k", get, set, setter)
            if err != nil { t.Error(err); return }
            results[i] = v
        }(i)
    }
    wg.Wait()

    assert.Equal(t, int64(1), setterRuns.Load())     // setter runs exactly once
    for _, r := range results { assert.Equal(t, "computed", r) }
}
```

**Assertion idioms for the other D-20 behaviors** (all from existing test files):
- `shared` is global — assert `shared == callers` (never `callers-1`; spike blueprint "What to Avoid").
- Panic recovery — `var p *cache.Panic; require.ErrorAs(t, err, &p)`; also `assert.ErrorIs(t, err, ...)` when the panic value is an error (Unwrap). Note: no `errors.As` exists in the repo yet — this is the first use; follow `testify require.ErrorAs` (mem_test uses `require`/`assert` throughout).
- `ErrCanceled` — `assert.ErrorIs(t, err, cache.ErrCanceled)` AND `assert.ErrorIs(t, err, context.Canceled)` (D-10, both must hold).
- Value-if-ready (D-08) — cancel a waiter's ctx after the leader finished; expect the value, not the error.
- Clobber guard (SF-08) — direct `set` while leader computes; leader's late result must not clobber.
- TTL passthrough (D-18/D-19) — setter returns after a `Set` that records the ttl received; assert the leader's ttl arrives untouched.
- Race-detector clean (SF-09 core) — plain `go test -race ./cache/...` (spikes verified with `go run -race`).

**Test style conventions** (AGENTS.md + mem_test.go): `t.Parallel()` on the top-level test and every subtest EXCEPT any test mutating shared/global state (parallel-test isolation note); `require` for error assertions, `assert` for values; use `t.Context()` not `context.Background()` (mem_test.go:22 convention).

---

### `cache/errors.go` (modified — utility, error defs)

**Analog:** itself — existing sentinel block (lines 7-12). Add `ErrCanceled` beside it, and the new `Panic` type. `ErrCanceled` MUST wrap `ctx.Err()` so both `errors.Is` checks hold (D-10) — implement with multiple `%w` verbs (Go 1.20+):

```go
var (
    // ErrMiss is returned by Get when the key is not in the cache.
    ErrMiss = errors.New("cache: key not found")

    // ErrClosed is returned when operations are attempted on a closed cache.
    ErrClosed = errors.New("cache: cache is closed")

    // ErrCanceled wraps the context error of a waiter that abandoned its wait
    // via DoChan. Both errors.Is(err, ErrCanceled) and
    // errors.Is(err, context.Canceled) hold (D-10).
    ErrCanceled = errors.New("cache: canceled")
)
```

**`Panic` type** — no repo analog (first typed error in the repo); mirror `x/sync/singleflight`'s `panicError` (module cache `golang.org/x/sync@v0.22.0/singleflight/singleflight.go:34-60`): fields `Value any` + `Stack []byte`, `Error()` = `fmt.Sprintf("%v\n\n%s", p.Value, p.Stack)`, `Unwrap()` returns `p.Value` when it is an `error` (so `errors.As`/`errors.Is` work through the recovered value — D-17). Naming: exported `Panic` struct (doc comment per AGENTS.md); optionally trim the first goroutine line of `debug.Stack()` like `singleflight.newPanicError` does.

---

### `cache/cache.go` (modified — contract)

**Analog:** itself. Only line 23 changes (D-03). Update the doc comment on the interface method too:

```go
// GetOrSet returns the existing value or computes, stores, and returns it.
// The setter receives a context.Context (the caller's ctx) and is invoked
// only on a miss. Concurrent misses on the same key may be deduplicated.
GetOrSet(ctx context.Context, key K, setter func(context.Context) (V, error), ttl ...time.Duration) (V, error)
```

---

### `cache/doc.go` (modified — package docs)

**Analog:** itself. Extend the sentinel list (lines 31-34) with `ErrCanceled`, and mention `SingleflightGetOrSet` in the interface description (lines 3-5). Keep the usage-block style (`//\t` indented).

---

### `cache/cache_test.go` (modified — sentinel tests)

**Analog:** itself, `TestCacheSentinelErrors` (lines 37-53). Add subtests for `ErrCanceled`:
```go
t.Run("err_canceled", func(t *testing.T) {
    t.Parallel()
    assert.Error(t, cache.ErrCanceled)
    assert.Contains(t, cache.ErrCanceled.Error(), "canceled")
})
```
(Whether `cache.Panic` gets a unit test here or in `singleflight_test.go` is at the executor's discretion; the sentinel block is the established home.)

---

### `cache/example_test.go` (modified — godoc examples, D-22)

**Analog:** itself, `ExampleWithDefaultTTL` (lines 10-14) — external test package `cache_test`, `fmt.Println` + `// Output:` assertion. Add `ExampleSingleflightGetOrSet_Do` and `ExampleSingleflightGetOrSet_DoChan` using the in-test stub closures (same shape as `singleflight_test.go`'s get/set) with deterministic output. Keep imports minimal (`fmt`, `context`, `time` as needed).

---

### `cache/cache_e2e_test.go` (modified — E2E callers)

**Analog:** itself. The 3 setter closures at lines 78, 90, 102 gain a ctx parameter (unused):
```go
got, err := c.GetOrSet(ctx, "e2e_gos", func(_ context.Context) (string, error) {
    return "computed", nil
})
```
Import `context` (line 6 currently has `errors`, `testing`, `time` — add `context`).

---

### Provider files — signature ripple (5 files)

**Analog for each:** its own `GetOrSet` body. Minimal Phase 5 change: setter param gains `context.Context` and the internal call becomes `setter(ctx)`. Bodies stay hand-rolled until Phase 6 (per CONTEXT: "replaced in Phase 6; referenced here as the consuming call sites").

| File | Current | Change to |
|------|---------|-----------|
| `cache/mem/mem.go:85` | `setter func() (V, error)` | `setter func(context.Context) (V, error)`; line 91 `computed, err := setter()` → `setter(ctx)` |
| `cache/redis/redis.go:86` | same | same; keep `fmt.Errorf("cache/redis: %w", err)` prefix on setter error (line 96) |
| `cache/valkey/valkey.go:106` | same | same; keep initErr guard (lines 114-117) AND `cache/valkey:` prefix (line 121) |
| `cache/memcache/memcache.go:129` | same | same; raw setter error stays (line 138) |
| `cache/postgres/postgres.go:128` | same | same; raw setter error stays (line 137) |

The `var zero V` + `computed, err := setter(...)` + `if err := c.Set(...)` shape is identical across all 5 (see mem.go:84-103 as the canonical excerpt). `setter(ctx)` receives the caller's ctx — no separate setterCtx (D-06). Do NOT touch error decoration/prefixes (D-14, spike 006 — that is Phase 6 parity work).

---

### Provider test files — closure ripple (5 files)

**Analog for each:** its own `get_or_set_computes` subtest. Setter closures gain an ignored ctx param:

| File | Line(s) | Current closure | Change to |
|------|---------|-----------------|-----------|
| `cache/mem/mem_test.go` | 112, 124, 137 | `func() (string, error)` | `func(context.Context) (string, error)` |
| `cache/redis/redis_test.go` | 72 | `func() (string, error) { return "computed", nil }` | `func(context.Context) (string, error) { return "computed", nil }` |
| `cache/valkey/valkey_test.go` | 72 | same | same |
| `cache/memcache/memcache_test.go` | 64 | `func() (string, error) {` | `func(context.Context) (string, error) {` |
| `cache/postgres/postgres_test.go` | 83 | same | same |

Keep `t.Parallel()` and `require`/`assert` conventions as-is; note memcache_test.go:60 and postgres_test.go:82 subtests do NOT call `t.Parallel()` (server-backed) — leave untouched.

---

## Shared Patterns

### Setter closure signature (D-03 breaking change)
**Source:** `cache/cache.go:23`
**Apply to:** All 5 providers + 8 test files (`cache_test`/E2E/provider tests) + new `singleflight_test.go` examples.
The setter type changes from `func() (V, error)` to `func(context.Context) (V, error)` everywhere. Within the helper, the setter receives the leader's ctx (D-04); within providers pre-Phase 6, it receives the caller's ctx.

### Zero-value + miss handling
**Source:** `cache/mem/mem.go:47-50` and every provider GetOrSet
**Apply to:** `cache/singleflight.go` and all provider methods
```go
if !ok {
    var zero V
    return zero, fmt.Errorf("cache/mem: %w", cache.ErrMiss)
}
```
The helper uses `var zero V` at the top (spike 005:39) for all error returns.

### Error sentinel + typed error conventions
**Source:** `cache/errors.go:7-12`
**Apply to:** `cache/errors.go` additions, `cache/singleflight.go` (ErrCanceled wrap), `cache/cache_test.go`
- Sentinels: `errors.New("cache: <phrase>")` with godoc comment.
- Wrapping chain: `fmt.Errorf("%w: %w", ErrCanceled, ctx.Err())` so `errors.Is` matches both (D-10).
- Typed errors: struct with `Error()`, `Unwrap()`; surfaced via `errors.As` (`require.ErrorAs` in tests).

### Concurrency test harness
**Source:** `sources/005-singleflight-placement/main.go:64-109`
**Apply to:** `cache/singleflight_test.go`
`sync.WaitGroup` fan-out over N goroutines + `atomic.Int64` setter-run counter + shared results slice + `wg.Wait()` before assertions. Verified `-race` clean in spikes.

### Panic containment
**Source:** `golang.org/x/sync@v0.22.0/singleflight/singleflight.go:34-60` (`panicError`) + `Do` replay behavior (`singleflight.go:104-108` re-panics `*panicError` to waiters)
**Apply to:** `cache/singleflight.go` (recover INSIDE the singleflight fn, return `*cache.Panic` — never let it escape to waiters) + `cache/errors.go` (`Panic` type).
x/sync's `Do`/`DoChan` re-panic the recovered value to each waiter — the wrapper's recovery must convert it before it reaches callers (D-16).

### TTL passthrough (D-18/D-19)
**Source:** every provider `GetOrSet` (e.g. `cache/mem/mem.go:97`)
**Apply to:** `cache/singleflight.go`
`ttl...` is passed verbatim to the set closure; providers resolve defaults inside `Set`. No TTL logic in shared code. When waiters differ, the leader's ttl wins (document).

### Test style (AGENTS.md)
**Apply to:** all new/modified tests
`testify` (`require` for errors, `assert` for values); `t.Parallel()` everywhere except tests that mutate shared/global state (AGENTS.md parallel-test isolation); `t.Context()`; godoc `Example` functions for all exported symbols with deterministic `// Output:`.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `cache/errors.go` (`Panic` type only) | utility | n/a | First typed error (`struct` with `Unwrap`) in the repo; no in-repo precedent. Use `x/sync singleflight.panicError` (module cache, `singleflight.go:34-60`) as the external analog — same shape: `{value, stack}`, `Error()` formats both, `Unwrap()` delegates to the value when it is an `error`. |

---

## Metadata

**Analog search scope:** `cache/` (all 5 provider subpackages + root package, incl. tests and examples), `.opencode/skills/spike-findings-go/sources/` (validated spike prototypes), `mid/` (global-state test-isolation convention), `golang.org/x/sync@v0.22.0` module cache (`singleflight` API + `panicError`).
**Files scanned:** 24 source/test files + 2 spike sources + 1 module source
**Pattern extraction date:** 2026-08-06

### Key excerpts lineage
| Excerpt | Lives in |
|---------|----------|
| Helper `Do` prototype (setter-only wrap, double-check Get, Set inside fn) | `sources/005-singleflight-placement/main.go:31-62` |
| `DoChan` + select-on-ctx (cancel-aware waiter) | `sources/008-singleflight-context/main.go:99-109` |
| Existing hand-rolled GetOrSet being replaced | `cache/mem/mem.go:84-103` (canonical), `redis.go:85-104`, `valkey.go:105-129`, `memcache.go:128-147`, `postgres.go:127-146` |
| Sentinels | `cache/errors.go:7-12` |
| `panicError` (model for `cache.Panic`) | `golang.org/x/sync@v0.22.0/singleflight/singleflight.go:34-60` |
| Sentinel tests | `cache/cache_test.go:37-53` |
| Provider GetOrSet test closures | `mem_test.go:112/124/137`, `redis_test.go:72`, `valkey_test.go:72`, `memcache_test.go:64`, `postgres_test.go:83` |
| E2E GetOrSet callers | `cache/cache_e2e_test.go:76-108` |
| Godoc example style | `cache/example_test.go:10-14` |

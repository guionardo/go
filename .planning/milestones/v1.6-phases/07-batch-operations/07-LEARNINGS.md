---
phase: 07
phase_name: "Batch operations"
project: "github.com/guionardo/dev/go"
generated: "2026-10-08"
counts:
  decisions: 8
  lessons: 7
  patterns: 7
  surprises: 4
missing_artifacts:
  - "07-VERIFICATION.md"
  - "07-UAT.md"
---

# Phase 07 Learnings: Batch operations

## Decisions

### BatchCache embedding interface — Cache[K,V] untouched
Batch ops were added via a new public `BatchCache[K,V]` interface embedding `Cache[K,V]`; `NewConcreteCache` and all 5 provider `New()` functions return `BatchCache[K,V]`, assignable to `Cache[K,V]` for existing callers. The methods live on the unexported `cacher` interface as `MGetFunc`/`MSetFunc`/`MDelFunc`.

**Rationale:** D-01/D-02 backward compatibility — external `Cache[K,V]` implementors are unaffected; the "separate Batch adapter type" alternative was rejected for import/discovery friction, following the Phase 5/6 singleflight pattern.
**Source:** 07-DISCUSSION-LOG.md

### MGet returns map[K]V with only found keys
`MGet(ctx, keys ...K) map[K]V` includes only keys that were found; missing keys are silently absent, with no per-key error propagation.

**Rationale:** Alternatives — per-key result struct (`map[K]BatchResult[V]`) and split return `(map[K]V, []K)` — were rejected for simplicity; matches BATCH-01.
**Source:** 07-DISCUSSION-LOG.md

### MSet takes one variadic TTL for the whole batch
`MSet(ctx, items map[K]V, ttl ...time.Duration)` applies a single optional TTL to every key, following the existing `Set` variadic TTL pattern; per-key TTL wrappers were rejected as verbose for callers.

**Rationale:** Simpler API for the common case; matches BATCH-02.
**Source:** 07-DISCUSSION-LOG.md

### Best-effort error accumulation via errors.Join
Batch writes accumulate per-key errors and continue processing; the aggregate is returned via `errors.Join`. "Stop on first error" was rejected.

**Rationale:** D-06 — process as many keys as possible and avoid data loss from early termination; `errors.Join` (stdlib, Go 1.20+) needs no custom error type.
**Source:** 07-DISCUSSION-LOG.md

### Provider-native batching for all five providers
Each provider uses its library's native single-round-trip mechanism: mem iterates under one lock acquisition (D-07), memcache uses `GetMulti` (D-08), redis uses go-redis `Pipeline` (D-09), valkey uses valkey-go `DoMulti` (D-10), postgres uses `pgx.SendBatch` (D-11).

**Rationale:** Reduces round trips from O(N) to O(1) — the phase's primary value; per-key loops were explicitly listed under "Don't Hand-Roll".
**Source:** 07-CONTEXT.md

### memcache MGet zips results back onto the original key slice
`GetMulti` returns a `map[string]*Item`, so MGet iterates the caller's original `keys` slice and looks up `keysStr[i]` per key — never reconstructing `K` from the returned string map.

**Rationale:** Pitfall 1 — distinct `K` values can collide under `fmt.Sprint` (e.g. `1` vs `"1"`), and reverse-mapping would lose the original key type; original ordering keeps requested keys authoritative.
**Source:** 07-RESEARCH.md

### redis MGet uses per-key pipe.Get(), not pipe.MGet()
Redis MGet queues one `pipe.Get(ctx, key)` command per key, then checks each `*redis.StringCmd` individually after `pipe.Exec`, instead of issuing a single `MGET`.

**Rationale:** Research open question 1 — `MGET` returns a `*redis.SliceCmd` whose positional results are harder to map back to keys; the round-trip saving comes from pipelining, not from the MGET command.
**Source:** 07-RESEARCH.md

### postgres consumes every SendBatch result before Close
All three postgres batch methods queue one query per key, then iterate exactly the queue count — `br.QueryRow().Scan()` for SELECT, `br.Exec()` for INSERT/DELETE — with `defer br.Close()`.

**Rationale:** pgx requires reading all results from `SendBatch` before closing the batch results; incomplete consumption leaves the connection in a bad state.
**Source:** 07-04-SUMMARY.md

---

## Lessons

### Expanding cacher broke all five providers at once
Adding `MGetFunc`/`MSetFunc`/`MDelFunc` to the unexported `cacher` interface broke compilation for mem, redis, valkey, memcache, and postgres; unplanned per-key-loop stubs were added to all five in commit 61a8009 and optimized later in plans 02–04.

**Context:** Rule 3 (blocking) deviation found during Task 1 verification — a strict Go interface forces every implementor to be updated in the same change.
**Source:** 07-01-SUMMARY.md

### Test doubles break go vet when the interface grows
`fakeCacher` in `concrete_cache_test.go` stopped satisfying `cacher` after the batch methods were added, blocking `go vet ./cache/...`; it needed locking and `f.err`-checked `MGetFunc`/`MSetFunc`/`MDelFunc`.

**Context:** Rule 3 (blocking) deviation in the same commit 61a8009 as the interface change — test doubles are part of the interface blast radius.
**Source:** 07-01-SUMMARY.md

### New untested methods fall below the 70% file gate
Adding 3 methods to `concrete_cache.go` and `mem.go` dropped file coverage below 70%, violating the pre-commit `make coverage-quick` rule; `TestConcreteCache_Batch` (7 subtests) and `TestMemCache_Batch` effectively merged Task 2 (tests) into Task 1.

**Context:** AGENTS.md coverage gate (files ≥70%, packages ≥80%, total ≥75%) — tests must land in the same commit as the new methods.
**Source:** 07-01-SUMMARY.md

### Goroutine-based provider code can tank coverage
Replacing memcache stubs with goroutine-based batch implementations added ~67 uncovered statements, dropping total coverage to 74.6% (below the 75% gate); `TestMemcacheBatch_NoServer` lifted the memcache package from 15.1% to 56.2% and total to 77.1%.

**Context:** The plan expected only `skipIfNoMemcache` external tests, but without a live server those exercise almost none of the new code.
**Source:** 07-02-SUMMARY.md

### valkey-go DoMulti returns a slice, not a single response
The PATTERNS/RESEARCH snippet `c.client.DoMulti(ctx, cmds...).Error()` did not compile — `DoMulti` returns `[]valkey.Response`; the fix iterates responses and checks each `resp.Error()`.

**Context:** Deviation found at the 07-03 Task 2 compilation check — researched library code snippets must be compile-verified before plans rely on them.
**Source:** 07-03-SUMMARY.md

### rtk reports "No tests found" for tests that skip
The `rtk` tool wrapper for `go test` reported "No tests found" for test functions that matched but skipped (memcache/postgres without Docker); running directly or with `-json` confirmed the tests execute correctly.

**Context:** Reporting false negative in the verification tooling — do not read it as a broken test registration.
**Source:** 07-04-SUMMARY.md

### Docker-E2E provider packages are coverage-exempt
redis (36.0%) and valkey (25.3%) fell far below the package:80 bar but passed via the `.testcoverage-quick.yml` provider override of 0 — their batch code paths are exercised only by Docker E2E tests.

**Context:** cache providers tested via E2E with Docker carry threshold overrides (noted in AGENTS.md); local runs without containers skip those tests entirely.
**Source:** 07-03-SUMMARY.md

---

## Patterns

### Batch delegation chain: BatchCache → concreteCache → cacher
Add `*Func` methods to the unexported `cacher` interface, thin delegation methods on `concreteCache` (`return c.cache.MGetFunc(ctx, keys...)`), and the public contract on `BatchCache[K,V]`; providers implement only the cacher methods.

**When to use:** Adding a capability to all cache providers without touching the public `Cache[K,V]` interface — mirror the existing Get/Set/Delete delegation shape.
**Source:** 07-PATTERNS.md

### Best-effort error accumulation with errors.Join
Collect per-key errors into `var errs []error`, keep processing, then `return errors.Join(errs...)` — nil or single-element slices need no special-casing.

**When to use:** Provider batch operations where keys can fail independently (memcache MSet/MDel, postgres MSet/MDel); MGet deliberately skips per-key errors instead of aggregating them.
**Source:** 07-PATTERNS.md

### Native single-round-trip batching per provider
Use the provider's own batching primitive: mem one lock acquisition over `store.get/set/delete`, memcache `GetMulti`, redis `Pipeline`, valkey `DoMulti`, postgres `SendBatch`.

**When to use:** Multi-key cache operations against any backend with a native batch/pipe mechanism — never loop single-key calls, since each is a separate round trip.
**Source:** 07-RESEARCH.md

### Original-key-order zip for string-keyed batch results
When a batch API returns results keyed by string (`GetMulti`), iterate the caller's original `keys` slice and look up `keysStr[i]` rather than iterating the returned map.

**When to use:** Any MGet for generic `K` backed by a string-keyed API — avoids `fmt.Sprint` key-type ambiguity and preserves requested order.
**Source:** 07-RESEARCH.md

### Goroutine+select context cancellation with buffered channel
Launch per-key goroutines that send results to `make(chan ..., len(items))` (buffered so senders never block), then consume in a `select` on `ctx.Done()` vs channel receive; on cancellation return `fmt.Errorf("cache/memcache: %w", ctx.Err())`.

**When to use:** Network providers whose client API takes no context argument (gomemcache) — memcache MSet/MDel and the wrapper around `GetMulti`.
**Source:** 07-PATTERNS.md

### Internal no-server tests for network providers
Construct the provider's cache struct directly with a real library client pointed at no server, exercising goroutine creation, channel send/receive, error accumulation, and error wrapping without Docker or skip gates.

**When to use:** Network-dependent providers whose external tests skip without containers — especially goroutine-heavy batch code that would otherwise stay uncovered.
**Source:** 07-02-SUMMARY.md

### Stub-first interface expansion marked TODO(07-N)
Add minimal per-key-loop implementations to every provider so the build stays green, mark each with a `TODO(07-N):` comment naming the plan that replaces it, then swap in the optimized strategy in later waves.

**When to use:** Wave-planned interface additions where infrastructure (Plan 01) must land before provider optimizations (Plans 02–04) and every wave must compile and pass tests.
**Source:** 07-01-SUMMARY.md

---

## Surprises

### One interface edit broke five providers and the test double
Task 1's cacher expansion compiled nowhere: all 5 provider types plus `fakeCacher` stopped satisfying the interface, requiring unplanned stubs in the same commit (61a8009) before the plan's own tests could run.

**Impact:** Turned a "core infrastructure" task into 6 extra edits; the stubs became the natural stub-first rollout later optimized in plans 02–04.
**Source:** 07-01-SUMMARY.md

### Replacing stubs with optimized code crashed coverage
Removing memcache's simple per-key stubs for goroutine-based MSet/MDel added ~67 uncovered statements and pushed total coverage to 74.6% — below the 75% gate; the memcache package itself was at 15.1% before internal tests.

**Impact:** Required an unplanned `memcache_internal_test.go` suite (5 subtests); coverage recovered to 77.1% total and 56.2% for the memcache package.
**Source:** 07-02-SUMMARY.md

### valkey-go DoMulti signature differed from the documented snippet
Both RESEARCH.md and PATTERNS.md showed `DoMulti(...).Error()`, but the function returns `[]valkey.Response` (a slice), so the planned code did not compile.

**Impact:** Task 2 needed per-response iteration; reinforces compile-verifying researched library snippets before trusting them in plans.
**Source:** 07-03-SUMMARY.md

### rtk wrapper hides tests that skip
The `rtk` output wrapper reported "No tests found" for test functions that matched but skipped (e.g. postgres SKIP tests), while `-json` and direct execution showed correct behavior.

**Impact:** A false "missing test" signal during verification; the wrapper's summary cannot be trusted for skip-only runs.
**Source:** 07-04-SUMMARY.md

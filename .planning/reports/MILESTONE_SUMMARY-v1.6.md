# Milestone v1.6 — Cache Dedup

**Generated:** 2026-08-08
**Purpose:** Team onboarding and project review

---

## 1. Project Overview

**go** — A collection of reusable Go utility packages (`github.com/guionardo/go`). This milestone focused on the `cache` package: eliminating duplicate setter work in cache misses and reducing round trips via singleflight GetOrSet, batch operations, and a benchmarking suite.

**Milestone goal:** Eliminate duplicate setter work in cache misses and reduce round trips — via singleflight GetOrSet across all 5 providers, batch operations (MGet/MSet/MDel), and a benchmark suite.

## 2. Architecture & Technical Decisions

- **Singleflight wraps only the setter** — Dedups concurrent misses without locking the fast-path Get. The shared `SingleflightGetOrSet[K,V]` helper is exported from the `cache` package and embedded by each provider.
  - *Why:* Avoids 5× duplicated miss→setter→Set code across providers; keeps Get outside the singleflight group for zero overhead on hits.
  - *Phase:* 5 (Shared singleflight helper)

- **BatchCache[K,V] embedding interface** — `BatchCache` embeds `Cache[K,V]` and adds MGet/MSet/MDel. Backward compatible — existing `Cache[K,V]` users compile without changes.
  - *Why:* Preserves the public `Cache` interface for external implementors while providing batch ops to consumers who need them.
  - *Phase:* 7 (Batch operations)

- **Provider-optimal batch strategies** — Each provider uses its most efficient approach: single lock (in-memory), native GetMulti (memcache), pipelines (redis/valkey), SendBatch (postgres).
  - *Why:* Leverages each backend's native batching capability instead of one-size-fits-all fallback.
  - *Phase:* 7 (Batch operations)

- **Error accumulation via errors.Join** — Batch operations continue processing remaining keys when one fails, aggregating all errors with `errors.Join`.
  - *Why:* Best-effort batch semantics — partial failure doesn't lose data from successful operations.
  - *Phase:* 7 (Batch operations)

- **DoChan cancel-aware variant** — `SingleflightGetOrSet.DoChan` lets callers abandon their wait promptly on context cancellation, with a non-blocking check for an already-ready result.
  - *Why:* HTTP-path callers need prompt cancellation (15ms vs 300ms blocking in spikes).
  - *Phase:* 5 (Shared singleflight helper)

- **Generation-tombstone delete guard** — A Delete issued while a setter is in flight is not resurrected by the late Set, via generation-tombstone re-check before Set.
  - *Phase:* 6 (Provider integration)

## 3. Phases Delivered

| Phase | Name | Status | Summary |
|-------|------|--------|---------|
| 5 | Shared singleflight helper | ✅ Complete | Exported `SingleflightGetOrSet[K,V]` with Do/DoChan, panic recovery, clobber guard, ErrCanceled sentinel |
| 6 | Provider integration | ✅ Complete (retroactive) | All 5 providers migrated from hand-rolled GetOrSet to shared `concreteCache`/`cacher` architecture |
| 7 | Batch operations | ✅ Complete | `BatchCache[K,V]` with MGet/MSet/MDel across all 5 providers with optimal per-provider strategies |
| 8 | Benchmark suite | ✅ Complete | Thundering-herd dedup and batch benchmarks quantifying wins; `make benchmark` + `make benchmark-quick` targets |

## 4. Requirements Coverage

- ✅ **BENCH-01**: Thundering-herd GetOrSet benchmark (setter runs 1× with singleflight vs N× naive)
- ✅ **BENCH-02**: Batch operations benchmark (native batch 10-49% faster than per-key)
- ✅ **BENCH-03**: `make benchmark` target with `-benchmem`
- ✅ **BATCH-01**: MGet(keys ...K) map[K]V
- ✅ **BATCH-02**: MSet(items map[K]V, ttl ...time.Duration) and MDel(keys ...K)
- ✅ **BATCH-03**: mem single-lock batch iteration
- ✅ **BATCH-04**: memcache native GetMulti for MGet
- ✅ **BATCH-05**: redis/valkey pipelines for MGet/MSet/MDel
- ✅ **BATCH-06**: postgres SendBatch
- ✅ **BATCH-07**: Empty/partial keys handled, race detector passes
- ✅ **SF-01/03/04/07/09**: Provider integration (error prefixes, generation-tombstone, race)
- ✅ **SF-02/05/06/08**: Shared helper (setter-only wrap, panic recovery, DoChan, clobber guard)

## 5. Key Decisions Log

| ID | Decision | Phase | Rationale |
|----|----------|-------|-----------|
| D-01 | Export `SingleflightGetOrSet[K,V]` from cache package | 5 | Providers are subpackages; unexported would be unreachable |
| D-02 | Providers embed the struct, call Do method | 5 | State persists per-provider so dedup works across calls |
| D-03 | Setter signature changes to `func(context.Context)` (v1.6 breaking) | 5 | Setter needs ctx for cancellation and timeout propagation |
| D-08 | Canceled waiter gets shared value if ready (non-blocking select) | 5 | Prevents unnecessary cancellation when leader already finished |
| D-10 | ErrCanceled wraps ctx.Err() — both errors.Is checks hold | 5 | Enables caller to detect both cache-level and context-level cancellation |
| D-01/D-02 | Batch ops on cacher interface, Cache[K,V] unchanged | 7 | Backward compatibility for external implementors |
| D-06 | errors.Join for batch error accumulation | 7 | Best-effort batch semantics |
| D-07..D-11 | Provider-specific strategies (lock, GetMulti, pipeline, SendBatch) | 7 | Optimal per-backend batching |
| D-01 | Single file cache/bench_test.go | 8 | All benchmarks in one place |
| D-02/D-05 | Multi-concurrency / multi-size benchmarks | 8 | Shows how dedup/batching win scales |
| D-11 | make benchmark with -benchtime=1s -count=5 -benchmem | 8 | Statistically meaningful results |

## 6. Tech Debt & Deferred Items

- **Benchmark suite**: `make benchmark-quick` exists for fast dev iteration; the full suite with `-count=5` takes longer but gives reliable numbers.
- **Phase 6 retroactive closure**: Phase 6 was closed retroactively (work already shipped as part of Phase 5/7 refactoring). No separate CONTEXT.md or VERIFICATION.md exists.
- **No security issues found** — cache package has no threat surface beyond what the provider backends expose. Standard Go best practices followed.
- **Out of scope (captured in REQUIREMENTS.md):** Cross-provider batch consistency guarantees, persistent singleflight dedup across processes, cache eviction policies (LRU/LFU).

## 7. Getting Started

- **Run tests:** `go test ./cache/...` (unit), `make test-e2e` (integration, needs Docker), `make coverage-quick` (coverage check)
- **Run benchmarks:** `make benchmark` (full suite, ~30s), `make benchmark-quick` (fast, ~5s)
- **Key directories:**
  - `cache/` — Core cache package (Cache/BatchCache interfaces, concreteCache, singleflight helper)
  - `cache/mem/` — In-memory provider (zero-dependency, background TTL sweep)
  - `cache/redis/` — Redis provider (go-redis/v9, lazy connect)
  - `cache/valkey/` — Valkey provider (valkey-go, eager connect)
  - `cache/memcache/` — Memcache provider (gomemcache, lazy connect)
  - `cache/postgres/` — PostgreSQL provider (pgx/v5, pgxpool, eager connect)
- **Where to look first:** `cache/cache.go` (Cache/BatchCache interfaces), `cache/concrete_cache.go` (shared adapter), `cache/singleflight.go` (dedup helper)
- **Dependencies:** Go 1.26+, testcontainers-go for E2E, go-redis/v9, valkey-go, gomemcache, pgx/v5
- **Quality:** `make coverage-quick` enforces 75% total coverage; `make lint` runs golangci-lint with 48 linters

---

## Stats

- **Timeline:** 2026-08-06 → 2026-08-08 (3 days)
- **Phases:** 4 / 4 complete
- **Commits:** 34
- **Files changed:** 46 files (+7,780 / -162)
- **Contributors:** Guionardo Furlan

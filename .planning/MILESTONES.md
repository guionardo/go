# Milestones

## v1.6 Cache Dedup (Shipped: 2026-08-08)

**Phases completed:** 5 phases, 11 plans, 27 tasks

**Key accomplishments:**

- Exported cache.SingleflightGetOrSet[K,V] blocking dedup helper with panic recovery (cache.Panic) and a double-check Get clobber guard, validated by a 6-behavior TDD suite and a godoc example.
- DoChan cancel-aware variant of SingleflightGetOrSet[K,V] with the cache.ErrCanceled sentinel: canceled waiters abandon at cancel latency (SF-06) returning an error matching both ErrCanceled and context.Canceled, value-if-ready wins (D-08), and the leader path never wraps its own cancellation (D-11)
- The `Cache` interface `GetOrSet` setter now takes `context.Context` — the breaking D-03 change rippled through all 5 provider methods (mem, redis, valkey, memcache, postgres) and every test/E2E call site, with error decoration, valkey's initErr guard, and `t.Parallel()` placement untouched; provider bodies stay hand-rolled (delegation to `SingleflightGetOrSet` is Phase 6).
- Added `BatchCache[K,V]` embedding interface, expanded `cacher` with 3 batch methods, added `concreteCache` delegation, changed `NewConcreteCache` return type, provided provider stubs, and added complete test coverage with 14 batch test subtests.
- mem single-lock batch (D-07), memcache GetMulti + goroutine-per-key batch (D-08), New() return type changed to BatchCache for both providers
- Implemented Pipeline-based batch operations for the redis provider (go-redis Pipeline per D-09) and DoMulti-based batch operations for the valkey provider (valkey-go DoMulti per D-10); changed both `New()` return types to `cache.BatchCache[K,V]`; added provider-level integration tests with skipIfNo* guards.
- Postgres SendBatch batch operations, postgres.New returning BatchCache, provider-level integration tests, E2E infrastructure updated for BatchCache, batch E2E subtests, and race detector verification across all 5 providers
- Thundering-herd singleflight benchmark in cache/bench_test.go quantifies the dedup win: naive path runs the setter N× per iteration, singleflight runs it exactly 1×, across concurrency levels 10–500, with mem (always) and 4 Docker-gated providers
- Batch benchmarks in cache/bench_test.go quantify the MGet/MSet/MDel win factor: native MGet 10-17% faster, native MDel up to 49% faster, and native MSet consistently lower allocations across batch sizes 1-1000 using the mem provider
- `make benchmark` and `make benchmark-quick` targets added to Makefile alongside existing test/coverage targets, with Docker-gated batch benchmark subtests for redis, valkey, memcache, and postgres providers using the shared runBatchBenchmarks helper

---

## v1.5 Self-Update (Shipped: 2026-07-21)

**Phases completed:** 1 phases, 3 plans, 0 tasks

**Key accomplishments:**

- (none recorded)

---

## v1.4 Core Packages (Shipped: 2026-07-21)

**Phases completed:** 1 phase, 3 plans, 50 tasks

**Timeline:** 2026-07-21 (4 hours)
**Commits:** 24
**Files changed:** 50 files, +6,625 / -51

**Key accomplishments:**

1. Generic `Cache[K, V]` interface with Get, Set, Delete, GetOrSet, Close across 5 backends
2. In-memory provider with TTL sweep, sync.RWMutex concurrency safety
3. Redis + Valkey providers using go-redis/v9 and valkey-go
4. Memcache provider with goroutine-per-call context wrapping
5. Postgres provider with UNLOGGED table, pg_prewarm, background TTL sweep
6. 50 E2E tests across all providers using testcontainers-go
7. Build-tag separation: e2e tests require Docker, regular tests pass without

**Verification:** passed
**UAT:** 8/8 tests passed

**Archived artifacts:**

- `milestones/v1.4-ROADMAP.md`
- `milestones/v1.4-REQUIREMENTS.md`
- `milestones/v1.4-phases/`

---

## v1.5 (Planned)

### Planned phases:

- STRNG: String utilities package (truncation, padding, join/split)
- RETRY: Retry with backoff strategies and jitter support

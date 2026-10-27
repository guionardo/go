# cache

Generic key-value cache with pluggable backend providers. Callers depend only on
`cache.Cache[K, V]` (or `cache.BatchCache[K, V]`); swapping a backend is a change
of constructor, not of business code.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/cache)

## Install

```bash
go get github.com/guionardo/go/cache
```

## Interfaces

```go
type Cache[K comparable, V any] interface {
    Get(ctx context.Context, key K) (V, error)
    Set(ctx context.Context, key K, value V, ttl ...time.Duration) error
    Delete(ctx context.Context, key K) error
    GetOrSet(ctx context.Context, key K, setter func(context.Context) (V, error), ttl ...time.Duration) (V, error)
    Close() error
}

// BatchCache embeds Cache for full backward compatibility.
type BatchCache[K comparable, V any] interface {
    Cache[K, V]

    // MGet returns only the keys that were found; misses are silently absent.
    MGet(ctx context.Context, keys ...K) map[K]V
    // MSet applies one TTL to the whole batch; empty ttl uses the provider default.
    MSet(ctx context.Context, items map[K]V, ttl ...time.Duration) error
    // MDel is idempotent: deleting missing keys is not an error.
    MDel(ctx context.Context, keys ...K) error
}
```

Every provider constructor returns a `cache.BatchCache[K, V]`, so batch methods are
available without a cast. `Get`, `Set`, `Delete` and `GetOrSet` take a
`context.Context`; `Close` does not.

## Architecture

Each provider implements a small unexported primitive —
`GetFunc`/`SetFunc`/`DeleteFunc`/`CloseFunc` plus `MGetFunc`/`MSetFunc`/`MDelFunc` —
and its `New` wraps that primitive in `cache.NewConcreteCache`. The adapter supplies
the public `BatchCache` surface, so the interface implementation and the `GetOrSet`
dedup logic exist once for all six backends rather than per backend.

`GetOrSet` is backed by `cache.SingleflightGetOrSet`, embedded per provider instance:

- The fast-path `Get` runs *outside* the singleflight group, so a hit never touches
  the group. On a miss, the group function double-checks the store before computing
  (a concurrent direct `Set` is not clobbered), then runs the setter exactly once in
  the leader's goroutine with the leader's context and stores the result.
- Concurrent misses on the same key share the leader's outcome: one setter call, one
  identical value or error for every waiter. When callers pass different TTLs, the
  **leader's TTL wins**; joiners receive the value as of when the leader started, so
  the staleness window is bounded by the setter's duration.
- `Do` is context-blind: cancelling a waiter's context does not release it — it stays
  blocked until the leader completes.
- A panicking setter yields a `*cache.PanicError` (recovered value + stack) to every
  waiter; no panic escapes to a caller goroutine.

Batch methods use a provider-optimal strategy: one lock acquisition for `cache/mem`,
a go-redis pipeline for `cache/redis`, `DoMulti` for `cache/valkey`, `GetMulti` for
`cache/memcache`, `SendBatch` for `cache/postgres`, and chunked `IN` queries plus a
single `BEGIN IMMEDIATE` upsert transaction for `cache/sqlite`.

Configuration is per provider via functional options; every provider has its own
`WithDefaultTTL`. Options are not interchangeable between providers (each package
defines its own `Option`/`ConfigFunc` type).

## Backends

Each backend is an independently importable sub-package:

| Package | Backend | Driver | Connection | External infrastructure |
|---------|---------|--------|------------|-------------------------|
| [`cache/mem`](https://pkg.go.dev/github.com/guionardo/go/cache/mem) | In-memory | stdlib only (zero dependencies) | None; background TTL sweep, passive expiry check on `Get` | None |
| [`cache/redis`](https://pkg.go.dev/github.com/guionardo/go/cache/redis) | Redis | go-redis/v9 | Lazy — dials on the first command | Redis server |
| [`cache/valkey`](https://pkg.go.dev/github.com/guionardo/go/cache/valkey) | Valkey (Redis wire-compatible) | valkey-go | Eager — client built in `New`; a dial failure is stored and surfaced by the first operation (`New` returns no error) | Valkey server |
| [`cache/memcache`](https://pkg.go.dev/github.com/guionardo/go/cache/memcache) | Memcache | gomemcache | Lazy — dials on first query; operations run in a goroutine for context cancellation | Memcached server |
| [`cache/postgres`](https://pkg.go.dev/github.com/guionardo/go/cache/postgres) | PostgreSQL | pgx/v5 + pgxpool | Eager — pool and `UNLOGGED` table created in `New`, which returns an error | PostgreSQL server |
| [`cache/sqlite`](https://pkg.go.dev/github.com/guionardo/go/cache/sqlite) | Embedded SQLite | modernc.org/sqlite (pure Go, no CGO) | Eager — local file or `:memory:` opened in `New`; open errors are deferred to the first database operation (`New` never fails) | None |

`cache/mem` and `cache/sqlite` need nothing installed. The other four need their
server reachable at the configured address.

`cache/sqlite` is the durable zero-infrastructure option: the zero-value
configuration opens an in-memory cache, `WithPath`/`WithName` open a WAL-mode file
that survives restarts (TTLs are stored as absolute timestamps, so expiry survives
too), and the same file can be shared by multiple processes on one host — WAL
serializes writers and a 5 s busy timeout bounds contention, surfacing as an error at
`BEGIN` rather than silent corruption. The pool is pinned to one connection, which is
what makes `:memory:` correct. File mode is local-storage only: network filesystems
and cross-host sharing are unsupported. `New` returns a `BatchCache` whose dynamic
type also implements `sqlite.Optimizable` (`Checkpoint`, `Vacuum`).

## Usage

The constructors return the interface, so the calling code never names the concrete
provider type:

```go
import (
    "context"

    "github.com/guionardo/go/cache/mem"
    // "github.com/guionardo/go/cache/redis"
)

ctx := context.Background()

// In tests — zero-dependency in-memory cache
c := mem.New[string, string](ctx)
_ = c.Set(ctx, "mykey", "myvalue")

// In production — Redis (same interface, different constructor)
// c := redis.New[string, string](redis.WithAddr("localhost:6379"))

v, err := c.Get(ctx, "mykey")
```

`GetOrSet` computes on a miss and deduplicates concurrent computations of the same
key:

```go
v, err := c.GetOrSet(ctx, "expensive", func(ctx context.Context) (string, error) {
    return compute(ctx) // runs once per key even under concurrent misses
}, 5*time.Minute)
```

Batch operations come from `BatchCache`:

```go
bc := mem.New[string, string](ctx)

_ = bc.MSet(ctx, map[string]string{"a": "alpha", "b": "beta", "c": "charlie"})

results := bc.MGet(ctx, "a", "b", "c", "missing") // "missing" is absent from the map
fmt.Println(results["a"])                         // alpha

_ = bc.MDel(ctx, "a", "b") // idempotent
```

Errors are wrapped with the provider prefix (`cache/redis:`, `cache/postgres:`, …),
so branch with `errors.Is`:

```go
if _, err := c.Get(ctx, "absent"); errors.Is(err, cache.ErrMiss) {
    // key not found (or expired, for the providers with TTL filtering)
}
```

## Notes

**Sentinel errors.** `cache.ErrMiss` is honoured by every provider: `cache/mem`
returns it bare, the others wrap it (`fmt.Errorf("cache/redis: %w", cache.ErrMiss)`,
etc.). `cache.ErrClosed` has partial adoption — `cache/sqlite` is currently the only
provider that returns it; the others do not signal a close. Always use `errors.Is`,
never `==`.

**Serialization.** External providers (redis, valkey, memcache, postgres, sqlite)
encode values with `encoding/json`, so `V` must round-trip through it — unexported
fields and non-JSON types are lost or fail. `cache/mem` stores `V` directly. Keys are
rendered with `fmt.Sprint` by those providers, so two distinct `K` values with the
same string form (`1` and `"1"`) would collide; use a `K` whose `Sprint` form is
unique.

**TTL defaults.** `cache/mem` defaults to a 5-minute TTL; the other providers default
to no expiry. A per-call TTL applies only when positive and otherwise falls back to
the provider default — except in `cache/mem`, where a non-positive per-call TTL means
"no expiry" for that entry.

**`cache/mem` limits.** Entries are bounded by `MaxEntries` (default 1000); when the
bound is exceeded the oldest inserted entries are evicted (insertion order, not LRU).
`WithMaxEntries(0)` disables the bound. The background sweeper only starts when the
`ctx` passed to `New` is non-nil; `Get` still expires entries passively.

**`cache/sqlite` behaviour.** `MGet` silently skips missing, expired and undecodable
keys and, because its signature has no error channel, returns partial results on a
cancelled context or a failed chunk (logged and skipped). `MSet` resolves one TTL for
the whole batch and writes it in a single transaction — all-or-nothing on marshal,
begin, exec or commit error. `MDel` deletes in chunks and is idempotent. `Close`
closes the database, which checkpoints the WAL and removes the `-wal`/`-shm` sidecars.
Cache contents are unencrypted at rest.

**`cache/postgres` schema.** `New` creates the cache table and attempts `pg_prewarm`;
prewarm is best-effort (a warning is logged and startup continues). The table is
`UNLOGGED`, so it is truncated by crash recovery — it is a cache, not storage. The
table defaults to `cache_entries` and `WithTableName` renames it for the DDL, the
queries and the prewarm alike.

## Benchmarks

`cache/bench_test.go` compares, on the `mem` provider, a naive thundering herd against
`GetOrSet`'s singleflight dedup (10/50/100/500 concurrent callers); it also runs batch
methods against a per-key loop (1/10/100/1000 keys) and the single-key primitives across
`mem` and both sqlite modes. Docker-backed providers are benchmarked when Docker is
available. The dedup difference is structural — one setter call instead of one per
caller — while batch timings are machine-dependent and not consistently favourable at
small batch sizes. Rerun before quoting figures:

```bash
make benchmark        # full suite, Docker-backed providers when available
make benchmark-quick  # fast iteration
```

The suite is explicit about what each arm times, so the numbers stay comparable:

- **Key space is bounded.** Batch and primitive arms draw keys from a 1024-key ring,
  so a file-backed run measures the operation rather than a database growing with
  `b.N`.
- **`MDel` arms time only the delete.** The `Set` calls that make the keys exist run
  with the timer stopped; the reported ns/op is the batch (or per-key) delete alone.
- **`GetOrSet` measures the miss path.** The key is deleted before every call
  (untimed), so the setter runs each iteration and the figure is not a warm hit.
- **`Delete` measures a `Set`+`Delete` cycle.** Removing an existing key requires
  creating it first, and both halves are timed.

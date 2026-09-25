# Phase 7: Batch operations - Pattern Map

**Mapped:** 2026-08-07
**Files analyzed:** 9 new/modified files
**Analogs found:** 9 / 9 (all have exact or role-match analogs)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cache/concrete_cache.go` | contract (cacher interface + concreteCache) | request-response (batch) | `cache/concrete_cache.go` (existing Get/Set/Delete) | exact — same file, same delegation pattern |
| `cache/concrete_cache_test.go` | test (unit) | request-response (batch) | `cache/concrete_cache_test.go` (existing fakeCacher) | exact — same file, same test double pattern |
| `cache/mem/mem.go` | provider (mem) | CRUD (batch) | `cache/mem/mem.go` (existing GetFunc/SetFunc/DeleteFunc) | exact — same file, same lock discipline |
| `cache/redis/redis.go` | provider (redis) | CRUD (batch) | `cache/redis/redis.go` (existing GetFunc/SetFunc/DeleteFunc) | exact — same file, same go-redis pattern |
| `cache/valkey/valkey.go` | provider (valkey) | CRUD (batch) | `cache/valkey/valkey.go` (existing GetFunc/SetFunc/DeleteFunc) | exact — same file, same valkey-go pattern |
| `cache/memcache/memcache.go` | provider (memcache) | CRUD (batch) | `cache/memcache/memcache.go` (existing GetFunc/SetFunc/DeleteFunc) | exact — same file, same gomemcache pattern |
| `cache/postgres/postgres.go` | provider (postgres) | CRUD (batch) | `cache/postgres/postgres.go` (existing GetFunc/SetFunc/DeleteFunc) | exact — same file, same pgx pattern |
| `cache/doc.go` | documentation | — | `cache/doc.go` (current package doc) | exact — same file |
| `cache/cache_e2e_test.go` | test (E2E) | CRUD (batch) | `cache/cache_e2e_test.go` (existing providerCase + runCacheE2E) | exact — same file, same E2E orchestration pattern |
| `cache/errors.go` | utility (errors) | — | `cache/errors.go` (existing sentinel errors) | exact — same file |

> **Note:** Not all files listed above need modification. `errors.go` requires no changes — D-06 uses `errors.Join` (stdlib, already imported where needed). `cache.go` is intentionally **not** modified per D-02 (backward compatibility).

## Pattern Assignments

### `cache/concrete_cache.go` (contract — cacher interface + concreteCache)

**Analog:** `cache/concrete_cache.go` lines 8-19 (existing `cacher` interface) and lines 27-41 (existing delegation methods)

**Nature of change:** Modified — adds 3 methods to `cacher` interface + 3 delegation methods on `concreteCache`

**Interface expansion pattern — `cacher` (lines 13-18):**
```go
// FROM: cache/concrete_cache.go lines 8-19
type (
    concreteCache[K comparable, V any] struct {
        sf    SingleflightGetOrSet[K, V]
        cache cacher[K, V]
    }
    cacher[K comparable, V any] interface {
        GetFunc(ctx context.Context, key K) (V, error)
        SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error
        DeleteFunc(ctx context.Context, key K) error
        CloseFunc() error

        // Phase 7 additions:
        MGetFunc(ctx context.Context, keys ...K) map[K]V
        MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error
        MDelFunc(ctx context.Context, keys ...K) error
    }
)
```

**Delegation pattern — concreteCache (lines 27-41 as analog):**
```go
// EXISTING PATTERN (Get/Set/Delete delegation) at concrete_cache.go lines 28-41:
func (c *concreteCache[K, V]) Get(ctx context.Context, key K) (value V, err error) {
    return c.cache.GetFunc(ctx, key)
}

func (c *concreteCache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
    return c.cache.SetFunc(ctx, key, value, ttl...)
}

func (c *concreteCache[K, V]) Delete(ctx context.Context, key K) error {
    return c.cache.DeleteFunc(ctx, key)
}

// NEW batch delegation — same pattern:
func (c *concreteCache[K, V]) MGet(ctx context.Context, keys ...K) map[K]V {
    return c.cache.MGetFunc(ctx, keys...)
}

func (c *concreteCache[K, V]) MSet(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    return c.cache.MSetFunc(ctx, items, ttl...)
}

func (c *concreteCache[K, V]) MDel(ctx context.Context, keys ...K) error {
    return c.cache.MDelFunc(ctx, keys...)
}
```

**Imports pattern (lines 3-6):**
```go
// concrete_cache.go — unchanged imports, no new dependencies needed
import (
    "context"
    "time"
)
```

> **Key observation:** The `concreteCache` proxy methods are thin wrappers that delegate to `cacher`. No error aggregation occurs here — that's the provider's responsibility per D-06.

---

### `cache/concrete_cache_test.go` (test — unit)

**Analog:** `cache/concrete_cache_test.go` lines 17-72 (existing `fakeCacher` pattern)

**Nature of change:** Modified — adds 3 batch methods to `fakeCacher` + `TestConcreteCache_Batch` test function

**fakeCacher batch methods (analog: existing fakeCacher single-key methods, lines 27-72):**
```go
// NEW batch methods on fakeCacher — follows same pattern as existing SetFunc/GetFunc/DeleteFunc
func (f *fakeCacher[K, V]) MGetFunc(_ context.Context, keys ...K) map[K]V {
    f.mu.Lock()
    defer f.mu.Unlock()

    result := make(map[K]V, len(keys))
    for _, key := range keys {
        if v, ok := f.data[key]; ok {
            result[key] = v
        }
    }
    return result
}

func (f *fakeCacher[K, V]) MSetFunc(_ context.Context, items map[K]V, _ ...time.Duration) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.err != nil {
        return f.err
    }
    for key, value := range items {
        f.data[key] = value
    }
    return nil
}

func (f *fakeCacher[K, V]) MDelFunc(_ context.Context, keys ...K) error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.err != nil {
        return f.err
    }
    for _, key := range keys {
        delete(f.data, key)
    }
    return nil
}
```

**Test pattern (analog: existing `TestConcreteCache_GetSetDelete`, lines 74-129):**
```go
// NEW test — follows same t.Parallel + subtest pattern
func TestConcreteCache_Batch(t *testing.T) {
    t.Parallel()

    t.Run("mget_returns_only_found_keys", func(t *testing.T) {
        t.Parallel()

        c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
        require.NoError(t, c.Set(t.Context(), "a", "1"))
        require.NoError(t, c.Set(t.Context(), "b", "2"))

        result := c.MGet(t.Context(), "a", "b", "missing")
        assert.Equal(t, "1", result["a"])
        assert.Equal(t, "2", result["b"])
        assert.NotContains(t, result, "missing")
    })

    t.Run("mget_empty_keys_returns_empty_map", func(t *testing.T) {
        t.Parallel()

        c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
        result := c.MGet(t.Context())
        assert.Empty(t, result)
    })

    t.Run("mset_stores_all_keys", func(t *testing.T) {
        t.Parallel()

        c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
        err := c.MSet(t.Context(), map[string]string{"a": "1", "b": "2"})
        require.NoError(t, err)

        got, err := c.Get(t.Context(), "a")
        require.NoError(t, err)
        assert.Equal(t, "1", got)

        got, err = c.Get(t.Context(), "b")
        require.NoError(t, err)
        assert.Equal(t, "2", got)
    })

    t.Run("mset_empty_map_does_not_error", func(t *testing.T) {
        t.Parallel()

        c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
        err := c.MSet(t.Context(), map[string]string{})
        require.NoError(t, err)
    })

    t.Run("mdel_deletes_all_keys", func(t *testing.T) {
        t.Parallel()

        c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
        _ = c.Set(t.Context(), "a", "1")
        _ = c.Set(t.Context(), "b", "2")

        err := c.MDel(t.Context(), "a", "b")
        require.NoError(t, err)

        _, err = c.Get(t.Context(), "a")
        require.ErrorIs(t, err, cache.ErrMiss)
        _, err = c.Get(t.Context(), "b")
        require.ErrorIs(t, err, cache.ErrMiss)
    })

    t.Run("mdel_empty_keys_does_not_error", func(t *testing.T) {
        t.Parallel()

        c := cache.NewConcreteCache[string, string](newFakeCacher[string, string]())
        err := c.MDel(t.Context())
        require.NoError(t, err)
    })

    t.Run("batch_with_provider_error_returns_aggregated", func(t *testing.T) {
        t.Parallel()

        f := newFakeCacher[string, string]()
        f.err = assert.AnError
        c := cache.NewConcreteCache[string, string](f)

        err := c.MSet(t.Context(), map[string]string{"a": "1"})
        require.Error(t, err)
        // errors.Join aggregates; provider wraps with errors.Join
    })
}
```

---

### `cache/mem/mem.go` (provider — in-memory)

**Analog:** `cache/mem/mem.go` lines 42-68 (existing GetFunc/SetFunc/DeleteFunc with lock discipline)

**Nature of change:** Modified — adds 3 batch methods, single lock acquisition per D-07

**Lock pattern (analog: lines 42-68):**
```go
// EXISTING patterns at mem.go:
// GetFunc (line 42-52): RLock for read
func (c *memoryCache[K, V]) GetFunc(ctx context.Context, key K) (value V, err error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    value, ok := c.store.get(key)
    if !ok {
        err = cache.ErrMiss
    }
    return value, err
}

// SetFunc (line 54-60): Lock for write
func (c *memoryCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.store.set(key, value, c.resolveTTL(ttl...))
    return nil
}

// DeleteFunc (line 62-68): Lock for write
func (c *memoryCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.store.delete(key)
    return nil
}

// NEW batch methods — same lock discipline, single acquisition per batch:
func (c *memoryCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    c.mu.RLock()
    defer c.mu.RUnlock()

    result := make(map[K]V, len(keys))
    for _, key := range keys {
        if v, ok := c.store.get(key); ok {
            result[key] = v
        }
    }
    return result
}

func (c *memoryCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    expiresAt := c.resolveTTL(ttl...)
    for key, value := range items {
        c.store.set(key, value, expiresAt)
    }
    return nil
}

func (c *memoryCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, key := range keys {
        c.store.delete(key)
    }
    return nil
}
```

**Key observation for mem:** No error aggregation needed — map operations cannot fail. `resolveTTL` is reused from existing pattern (lines 84-98).

---

### `cache/redis/redis.go` (provider — redis)

**Analog:** `cache/redis/redis.go` lines 44-87 (existing GetFunc/SetFunc/DeleteFunc with go-redis)

**Nature of change:** Modified — adds 3 batch methods using go-redis Pipeline (D-09)

**MGet with pipeline (analog: GetFunc lines 44-62 for serialization pattern):**
```go
func (c *redisCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    pipe := c.client.Pipeline()
    cmds := make([]*redis.StringCmd, len(keys))
    for i, key := range keys {
        cmds[i] = pipe.Get(ctx, fmt.Sprint(key))
    }
    _, _ = pipe.Exec(ctx) // ignore pipeline-level error; check per command

    result := make(map[K]V, len(keys))
    for i, key := range keys {
        data, err := cmds[i].Bytes()
        if err == redis.Nil {
            continue // missing key — skip silently
        }
        if err != nil {
            continue // per-key error — skip (D-06 best-effort)
        }
        var value V
        if err := json.Unmarshal(data, &value); err != nil {
            continue // deserialization error — skip
        }
        result[key] = value
    }
    return result
}

// MSet with pipeline (analog: SetFunc lines 66-78 for serialization pattern):
func (c *redisCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    pipe := c.client.Pipeline()
    expiration := c.resolveTTL(ttl...)

    for key, value := range items {
        data, err := json.Marshal(value)
        if err != nil {
            return fmt.Errorf("cache/redis: %w", err) // hard failure on marshal
        }
        pipe.Set(ctx, fmt.Sprint(key), data, expiration)
    }

    if _, err := pipe.Exec(ctx); err != nil {
        return fmt.Errorf("cache/redis: %w", err)
    }
    return nil
}

// MDel with pipeline (analog: DeleteFunc lines 80-87):
func (c *redisCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    pipe := c.client.Pipeline()
    for _, key := range keys {
        pipe.Del(ctx, fmt.Sprint(key))
    }
    if _, err := pipe.Exec(ctx); err != nil {
        return fmt.Errorf("cache/redis: %w", err)
    }
    return nil
}
```

**Key observation for redis:** Pipeline-level error (`Exec`) masks per-command failures. For MGet (D-06), per-command errors are silently skipped (best-effort). For MSet and MDel, `Exec` error covers the whole batch.

---

### `cache/valkey/valkey.go` (provider — valkey)

**Analog:** `cache/valkey/valkey.go` lines 47-107 (existing GetFunc/SetFunc/DeleteFunc with valkey-go)

**Nature of change:** Modified — adds 3 batch methods using valkey-go DoMulti (D-10)

**MGet with DoMulti (analog: GetFunc lines 48-69 for pattern):**
```go
// EXISTING initErr guard pattern at valkey.go lines 51-53:
func (c *valkeyCache[K, V]) GetFunc(ctx context.Context, key K) (V, error) {
    var zero V
    if c.initErr != nil {
        return zero, fmt.Errorf("cache/valkey: %w", c.initErr)
    }
    // ... Do ...

// NEW batch methods — each checks initErr first:
func (c *valkeyCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    var zero V // not used but documents zero-value pattern

    if c.initErr != nil {
        return nil // or empty map — consistent with empty-failure behavior
    }

    cmds := make([]valkey.Completed, len(keys))
    for i, key := range keys {
        cmds[i] = c.client.B().Get().Key(fmt.Sprint(key)).Build()
    }
    responses := c.client.DoMulti(ctx, cmds...)

    result := make(map[K]V, len(keys))
    for i, key := range keys {
        data, err := responses[i].ToString()
        if errors.Is(err, valkey.Nil) {
            continue // missing key — skip silently
        }
        if err != nil {
            continue // per-key error — skip (D-06 best-effort)
        }
        var value V
        if err := json.Unmarshal([]byte(data), &value); err != nil {
            continue // deserialization error — skip
        }
        result[key] = value
    }
    return result
}

// MSet with DoMulti (analog: SetFunc lines 73-93):
func (c *valkeyCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    if c.initErr != nil {
        return fmt.Errorf("cache/valkey: %w", c.initErr)
    }

    expiration := c.resolveTTL(ttl...)
    cmds := make([]valkey.Completed, 0, len(items))

    for key, value := range items {
        data, err := json.Marshal(value)
        if err != nil {
            return fmt.Errorf("cache/valkey: %w", err) // hard failure on marshal
        }
        cmd := c.client.B().Set().Key(fmt.Sprint(key)).Value(string(data)).Build()
        if expiration > 0 {
            cmd = c.client.B().Set().Key(fmt.Sprint(key)).Value(string(data)).Px(expiration).Build()
        }
        cmds = append(cmds, cmd)
    }

    if err := c.client.DoMulti(ctx, cmds...).Error(); err != nil {
        return fmt.Errorf("cache/valkey: %w", err)
    }
    return nil
}

// MDel with DoMulti (analog: DeleteFunc lines 96-107):
func (c *valkeyCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    if c.initErr != nil {
        return fmt.Errorf("cache/valkey: %w", c.initErr)
    }

    cmds := make([]valkey.Completed, len(keys))
    for i, key := range keys {
        cmds[i] = c.client.B().Del().Key(fmt.Sprint(key)).Build()
    }

    if err := c.client.DoMulti(ctx, cmds...).Error(); err != nil {
        return fmt.Errorf("cache/valkey: %w", err)
    }
    return nil
}
```

---

### `cache/memcache/memcache.go` (provider — memcache)

**Analog:** `cache/memcache/memcache.go` lines 48-131 (existing GetFunc/SetFunc/DeleteFunc with goroutine-per-key + context cancellation)

**Nature of change:** Modified — adds 3 batch methods (MGet via `GetMulti` per D-08, MSet/MDel via per-key goroutines)

**Imports (no changes needed — `errors` already imported?):**
```go
// Current imports (memcache.go lines 3-12):
import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "time"
    "github.com/bradfitz/gomemcache/memcache"
    "github.com/guionardo/go/cache"
)
```

**MGet using GetMulti (D-08) — key ordering pattern avoids K-type ambiguity (see "Shared Patterns"):**
```go
func (c *memcacheCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    keysStr := make([]string, len(keys))
    for i, k := range keys {
        keysStr[i] = fmt.Sprint(k)
    }

    ch := make(chan struct {
        items map[string]*memcache.Item
        err   error
    }, 1)

    go func() {
        items, err := c.client.GetMulti(keysStr)
        ch <- struct {
            items map[string]*memcache.Item
            err   error
        }{items, err}
    }()

    select {
    case <-ctx.Done():
        return nil
    case r := <-ch:
        result := make(map[K]V, len(keys))
        // Use original key ordering to avoid type ambiguity (see Pitfall 1)
        for i, key := range keys {
            item, ok := r.items[keysStr[i]]
            if !ok {
                continue // missing key — skip
            }
            var value V
            if err := json.Unmarshal(item.Value, &value); err != nil {
                continue // deserialization error — skip (D-06 best-effort)
            }
            result[key] = value
        }
        return result
    }
}

// MSet with per-key goroutines + error accumulation (analog: SetFunc lines 83-110):
func (c *memcacheCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    type result struct {
        err error
    }

    ch := make(chan result, len(items))
    expiration := c.resolveTTL(ttl...)

    for key, value := range items {
        go func(k K, v V) {
            data, err := json.Marshal(v)
            if err != nil {
                ch <- result{fmt.Errorf("cache/memcache: %w", err)}
                return
            }
            item := &memcache.Item{
                Key:        fmt.Sprint(k),
                Value:      data,
                Expiration: expiration,
            }
            if err := c.client.Set(item); err != nil {
                ch <- result{fmt.Errorf("cache/memcache: %w", err)}
                return
            }
            ch <- result{}
        }(key, value)
    }

    var errs []error
    for range items {
        select {
        case <-ctx.Done():
            return fmt.Errorf("cache/memcache: %w", ctx.Err())
        case r := <-ch:
            if r.err != nil {
                errs = append(errs, r.err)
            }
        }
    }

    return errors.Join(errs...)
}

// MDel with per-key goroutines + error accumulation (analog: DeleteFunc lines 113-131):
func (c *memcacheCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    ch := make(chan error, len(keys))

    for _, key := range keys {
        go func(k K) {
            err := c.client.Delete(fmt.Sprint(k))
            if err == memcache.ErrCacheMiss {
                ch <- nil // idempotent — missing key is not an error
                return
            }
            if err != nil {
                ch <- fmt.Errorf("cache/memcache: %w", err)
                return
            }
            ch <- nil
        }(key)
    }

    var errs []error
    for range keys {
        select {
        case <-ctx.Done():
            return fmt.Errorf("cache/memcache: %w", ctx.Err())
        case err := <-ch:
            if err != nil {
                errs = append(errs, err)
            }
        }
    }

    return errors.Join(errs...)
}
```

---

### `cache/postgres/postgres.go` (provider — postgres)

**Analog:** `cache/postgres/postgres.go` lines 68-128 (existing GetFunc/SetFunc/DeleteFunc with pgx)

**Nature of change:** Modified — adds 3 batch methods using pgx SendBatch (D-11)

**Imports — may need to add `"errors"`:**
```go
// Current imports (postgres.go lines 3-15):
import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "sync"
    "time"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/guionardo/go/cache"
)
// MAY need: "errors"  (for errors.Join and errors.Is)
```

**MGet with SendBatch (analog: GetFunc lines 69-93):**
```go
func (c *postgresCache[K, V]) MGetFunc(ctx context.Context, keys ...K) map[K]V {
    query := fmt.Sprintf(
        "SELECT value FROM %s WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > NOW())",
        pgx.Identifier{c.tableName}.Sanitize(),
    )

    batch := &pgx.Batch{}
    for _, key := range keys {
        batch.Queue(query, fmt.Sprint(key))
    }

    br := c.pool.SendBatch(ctx, batch)
    defer br.Close()

    result := make(map[K]V, len(keys))
    for i, key := range keys {
        var valueJSON string
        err := br.QueryRow().Scan(&valueJSON)
        if errors.Is(err, pgx.ErrNoRows) {
            continue // missing key — skip
        }
        if err != nil {
            continue // query error — skip (D-06 best-effort)
        }
        var value V
        if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
            continue // deserialization error — skip
        }
        result[key] = value
    }
    return result
}

// MSet with SendBatch (analog: SetFunc lines 96-114):
func (c *postgresCache[K, V]) MSetFunc(ctx context.Context, items map[K]V, ttl ...time.Duration) error {
    query := fmt.Sprintf(
        "INSERT INTO %s (cache_key, value, expires_at) VALUES ($1, $2, $3) ON CONFLICT (cache_key) DO UPDATE SET value = $2, expires_at = $3",
        pgx.Identifier{c.tableName}.Sanitize(),
    )

    batch := &pgx.Batch{}
    expiresAt := c.resolveTTL(ttl...)

    for key, value := range items {
        data, err := json.Marshal(value)
        if err != nil {
            return fmt.Errorf("cache/postgres: %w", err) // hard failure on marshal
        }
        batch.Queue(query, fmt.Sprint(key), string(data), expiresAt)
    }

    br := c.pool.SendBatch(ctx, batch)
    defer br.Close()

    // Consume results (pgx requires reading all results from SendBatch)
    var errs []error
    for range items {
        if _, err := br.Exec(); err != nil {
            errs = append(errs, fmt.Errorf("cache/postgres: %w", err))
        }
    }

    return errors.Join(errs...)
}

// MDel with SendBatch (analog: DeleteFunc lines 117-128):
func (c *postgresCache[K, V]) MDelFunc(ctx context.Context, keys ...K) error {
    query := fmt.Sprintf(
        "DELETE FROM %s WHERE cache_key = $1",
        pgx.Identifier{c.tableName}.Sanitize(),
    )

    batch := &pgx.Batch{}
    for _, key := range keys {
        batch.Queue(query, fmt.Sprint(key))
    }

    br := c.pool.SendBatch(ctx, batch)
    defer br.Close()

    // Consume results (pgx requires reading all results)
    var errs []error
    for range keys {
        ct, err := br.Exec()
        if err != nil {
            errs = append(errs, fmt.Errorf("cache/postgres: %w", err))
        }
        _ = ct // rows affected — could log but not required
    }

    return errors.Join(errs...)
}
```

**Key observation for postgres:** `SendBatch` requires consuming all results (via `br.Exec()` or `br.QueryRow()`) before closing. The `defer br.Close()` ensures cleanup even on partial failure.

---

### `cache/doc.go` (documentation)

**Analog:** `cache/doc.go` lines 1-45 (current package doc)

**Nature of change:** Modified — add batch operations to the doc comment

**Documentation update pattern:**
```go
// Add to the Cache interface description (currently line 4):
// The Cache[K, V] interface exposes Get, Set, Delete, GetOrSet, and Close —
// all accepting context.Context for cancellation and timeout propagation.
// --->
// The Cache[K, V] interface exposes Get, Set, Delete, GetOrSet, Close,
// MGet, MSet, and MDel — ...

// Add a section after "Providers (importable sub-packages):" (currently line 18):
// Batch operations:
//
//   MGet(ctx, keys...) map[K]V    — retrieve multiple keys (missing keys absent)
//   MSet(ctx, items, ttl...) error — store multiple values (batch best-effort)
//   MDel(ctx, keys...) error       — delete multiple keys (idempotent)
//
// Each provider uses an optimal strategy: mem uses a single lock acquisition,
// redis/valkey use pipelines, memcache uses GetMulti, postgres uses SendBatch.
```

---

### `cache/cache_e2e_test.go` (test — E2E)

**Analog:** `cache/cache_e2e_test.go` lines 24-147 (existing `providerCase` + `runCacheE2E` pattern)

**Nature of change:** Modified — add batch operation test cases to `runCacheE2E`

**E2E batch test cases to add within `runCacheE2E` (analog: existing subtest pattern at lines 37-137):**
```go
t.Run("mget_returns_multiple", func(t *testing.T) {
    _ = c.Set(ctx, "e2e_mget_a", "value_a")
    _ = c.Set(ctx, "e2e_mget_b", "value_b")

    result := c.MGet(ctx, "e2e_mget_a", "e2e_mget_b", "e2e_mget_missing")
    if result["e2e_mget_a"] != "value_a" {
        t.Fatalf("expected value_a, got %q", result["e2e_mget_a"])
    }
    if result["e2e_mget_b"] != "value_b" {
        t.Fatalf("expected value_b, got %q", result["e2e_mget_b"])
    }
    if _, ok := result["e2e_mget_missing"]; ok {
        t.Fatal("missing key should not be in result")
    }
})

t.Run("mget_empty_keys", func(t *testing.T) {
    result := c.MGet(ctx)
    if len(result) != 0 {
        t.Fatalf("expected empty map, got %v", result)
    }
})

t.Run("mset_stores_all_keys", func(t *testing.T) {
    items := map[string]string{
        "e2e_mset_a": "val_a",
        "e2e_mset_b": "val_b",
    }
    if err := c.MSet(ctx, items); err != nil {
        t.Fatal(err)
    }

    got, err := c.Get(ctx, "e2e_mset_a")
    if err != nil {
        t.Fatal(err)
    }
    if got != "val_a" {
        t.Fatalf("got %q, want %q", got, "val_a")
    }

    got, err = c.Get(ctx, "e2e_mset_b")
    if err != nil {
        t.Fatal(err)
    }
    if got != "val_b" {
        t.Fatalf("got %q, want %q", got, "val_b")
    }
})

t.Run("mset_empty_map", func(t *testing.T) {
    if err := c.MSet(ctx, map[string]string{}); err != nil {
        t.Fatal(err)
    }
})

t.Run("mdel_removes_all_keys", func(t *testing.T) {
    _ = c.Set(ctx, "e2e_mdel_a", "val_a")
    _ = c.Set(ctx, "e2e_mdel_b", "val_b")

    if err := c.MDel(ctx, "e2e_mdel_a", "e2e_mdel_b"); err != nil {
        t.Fatal(err)
    }

    if _, err := c.Get(ctx, "e2e_mdel_a"); err == nil {
        t.Fatal("expected error after MDel")
    }
    if _, err := c.Get(ctx, "e2e_mdel_b"); err == nil {
        t.Fatal("expected error after MDel")
    }
})

t.Run("mdel_idempotent", func(t *testing.T) {
    if err := c.MDel(ctx, "e2e_mdel_nonexistent"); err != nil {
        t.Fatalf("MDel of missing keys should not error: %v", err)
    }
})

t.Run("mdel_empty_keys", func(t *testing.T) {
    if err := c.MDel(ctx); err != nil {
        t.Fatal(err)
    }
})
```

---

### `cache/errors.go` (utility — sentinel errors)

**Analog:** `cache/errors.go` (entire file, lines 1-57)

**Nature of change:** No changes needed.

`errors.Join` is a standard library function (available since Go 1.20, project uses Go 1.26.4). No new error types are needed because:
- D-06: Error aggregation uses `errors.Join` which is already in `errors` package.
- D-03: MGet returns `map[K]V` — no error return.
- D-05: MDel is idempotent — no new error semantics.

Each provider wraps errors with `fmt.Errorf("cache/<provider>: %w", err)` as existing pattern.

---

## Shared Patterns

### 1. Cacher delegation pattern (all batch operations)

**Source:** `cache/concrete_cache.go` lines 27-41 (existing Get/Set/Delete)

**Applies to:** All batch methods added to `concreteCache` and all `cacher` interface additions

```go
// The pattern is:
// 1. Add method to cacher interface
// 2. Add delegation method on concreteCache
// 3. Each provider implements the cacher method with provider-specific strategy

// concreteCache delegates:
func (c *concreteCache[K, V]) MGet(ctx context.Context, keys ...K) map[K]V {
    return c.cache.MGetFunc(ctx, keys...)
}
```

### 2. Error aggregation pattern (D-06)

**Source:** Standard library `errors.Join` (Go 1.20+)

**Applies to:** Provider batch methods where partial failures can occur (memcache MSet/MDel, postgres MSet/MDel)

```go
// Pattern: accumulate errors, continue processing, return all at end
var errs []error
for key, value := range items {
    if err := someOperation(key, value); err != nil {
        errs = append(errs, fmt.Errorf("key %v: %w", key, err))
    }
}
return errors.Join(errs...)
```

**Key insight:** `errors.Join` with nil or single-element slice returns nil or that error respectively — no special casing needed.

### 3. Provider-level TTL resolution (MSet)

**Source:** All providers have `resolveTTL` — `cache/mem/mem.go` lines 84-98, `cache/redis/redis.go` lines 96-104, `cache/valkey/valkey.go` lines 119-127, `cache/memcache/memcache.go` lines 140-159, `cache/postgres/postgres.go` lines 146-156

**Applies to:** `MSetFunc` in all providers — resolve TTL once, apply to all items in batch

```go
// All providers follow this pattern:
expiresAt := c.resolveTTL(ttl...)     // resolve once
for key, value := range items {
    c.store.set(key, value, expiresAt) // apply same TTL to all
}
```

### 4. Provider-level `fmt.Sprint(key)` for key serialization

**Source:** All providers — `cache/redis/redis.go` line 48, `cache/valkey/valkey.go` line 55, `cache/memcache/memcache.go` line 50, `cache/postgres/postgres.go` line 76

**Applies to:** All batch methods that convert generic `K` keys to provider strings

```go
// All providers use this pattern for key-to-string conversion:
keyStr := fmt.Sprint(key)
```

### 5. Memcache context cancellation pattern (goroutine-per-operation)

**Source:** `cache/memcache/memcache.go` lines 49-79 (GetFunc)

**Applies to:** memcache MSetFunc, MDelFunc (per-key goroutines), and MGetFunc (single goroutine wrapping GetMulti)

```go
// Goroutine + select pattern for context cancellation:
ch := make(chan resultType, 1)
go func() {
    // ... do work ...
    ch <- result
}()
select {
case <-ctx.Done():
    return zero, fmt.Errorf("cache/memcache: %w", ctx.Err())
case r := <-ch:
    // handle result
}
```

### 6. go-redis Pipeline pattern

**Source:** None in existing code (pipeline is new for Phase 7), but follows go-redis public API

**Applies to:** redis MGetFunc, MSetFunc, MDelFunc

```go
pipe := c.client.Pipeline()
// Queue commands:
pipe.Get(ctx, key)  // returns *redis.StringCmd
pipe.Set(ctx, key, data, expiration)
pipe.Del(ctx, key)
// Execute:
_, err := pipe.Exec(ctx)
// Check individual commands:
data, err := cmds[i].Bytes()
```

### 7. Valkey-go DoMulti pattern

**Source:** None in existing code (DoMulti is new for Phase 7), but follows valkey-go public API

**Applies to:** valkey MGetFunc, MSetFunc, MDelFunc

```go
cmds := make([]valkey.Completed, len(keys))
for i, key := range keys {
    cmds[i] = c.client.B().Get().Key(fmt.Sprint(key)).Build()
}
responses := c.client.DoMulti(ctx, cmds...)
// Check individual results:
data, err := responses[i].ToString()
```

### 8. Postgres SendBatch pattern

**Source:** None in existing code (SendBatch is new for Phase 7), but follows pgx public API

**Applies to:** postgres MGetFunc, MSetFunc, MDelFunc

```go
batch := &pgx.Batch{}
for _, key := range keys {
    batch.Queue(query, fmt.Sprint(key))
}
br := c.pool.SendBatch(ctx, batch)
defer br.Close()
// Consume results:
for range keys {
    // QueryRow().Scan() for SELECT, Exec() for INSERT/DELETE
}
```

### 9. Test double pattern (fakeCacher)

**Source:** `cache/concrete_cache_test.go` lines 17-72

**Applies to:** Batch methods added to `fakeCacher` for unit testing

```go
// All fakeCacher methods follow:
// 1. Lock
// 2. Check f.err for simulated errors
// 3. Operate on f.data map
// 4. Return result/error
```

### 10. E2E test orchestration pattern

**Source:** `cache/cache_e2e_test.go` lines 24-147

**Applies to:** Batch E2E test cases added to `runCacheE2E`

```go
// The pattern is:
// 1. Add new t.Run subtests inside runCacheE2E
// 2. Use t.Fatalf for failure (not testify — E2E uses raw testing.T)
// 3. Key names prefixed with "e2e_" for isolation
// 4. All providers exercise the same subtests via providerCase slices
```

---

## Key Excerpts Lineage

| Pattern | Source File | Lines | Used By |
|---------|-------------|-------|---------|
| Cacher interface expansion | `cache/concrete_cache.go` | 13-18 | All provider files |
| Delegation proxy methods | `cache/concrete_cache.go` | 28-41 | concrete_cache.go batch methods |
| mem lock discipline (RLock/Lock) | `cache/mem/mem.go` | 42-68 | mem batch methods |
| mem resolveTTL | `cache/mem/mem.go` | 84-98 | mem MSetFunc |
| redis go-redis client calls | `cache/redis/redis.go` | 44-87 | redis batch methods |
| redis resolveTTL | `cache/redis/redis.go` | 96-104 | redis MSetFunc |
| valkey initErr guard | `cache/valkey/valkey.go` | 51-53, 74-76, 97-99 | valkey batch methods |
| valkey valkey-go client calls | `cache/valkey/valkey.go` | 55, 84-88, 101 | valkey batch methods |
| valkey resolveTTL | `cache/valkey/valkey.go` | 119-127 | valkey MSetFunc |
| memcache goroutine+select pattern | `cache/memcache/memcache.go` | 49-79 | memcache batch methods |
| memcache resolveTTL | `cache/memcache/memcache.go` | 140-159 | memcache MSetFunc |
| postgres pgx query formatting | `cache/postgres/postgres.go` | 70-73, 104-107, 118-121 | postgres batch methods |
| postgres resolveTTL | `cache/postgres/postgres.go` | 146-156 | postgres MSetFunc |
| fakeCacher pattern | `cache/concrete_cache_test.go` | 17-72 | fakeCacher batch methods |
| Unit test (t.Parallel + subtests) | `cache/concrete_cache_test.go` | 74-129 | TestConcreteCache_Batch |
| E2E test (providerCase + runCacheE2E) | `cache/cache_e2e_test.go` | 24-147 | Batch E2E subtests |
| Error accumulation (errors.Join) | stdlib (Go 1.20+) | — | memcache/postgres batch methods |

---

## No Analog Found

All 9 files have exact or role-match analogs in the existing codebase. No file lacks a pattern to follow.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| — | — | — | All files have close analogs |

**Note:** The go-redis Pipeline pattern, valkey-go DoMulti pattern, and pgx SendBatch pattern are new for this phase and have **no direct analog in the existing codebase**. For these, use the RESEARCH.md code examples and the respective library documentation as the implementation reference. The per-command result checking pattern for pipeline-style APIs is documented in RESEARCH.md lines 308-422.

---

## Metadata

**Analog search scope:** `/Users/guionardo/dev/go/cache/` and all subdirectories
**Files scanned:** 15+ source files (all providers, contracts, tests, utilities)
**Pattern extraction date:** 2026-08-07
**Go version:** 1.26.4
**Key stdlib dependency:** `errors.Join` (available since Go 1.20)

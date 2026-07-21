# Phase 2: Cache Package - Pattern Map

**Mapped:** 2026-07-21
**Files analyzed:** 28 new files + 1 modified (go.mod)
**Analogs found:** 28 / 28

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cache/cache.go` | interface | request-response | `set/set.go` | exact (generic type conventions) |
| `cache/errors.go` | utility | none | `fraction/fraction.go` (lines 37-53) | exact (sentinel errors) |
| `cache/options.go` | config | none | `config/options.go` | exact (functional options) |
| `cache/cache_test.go` | test | CRUD | `set/set_test.go` | exact (test patterns) |
| `cache/example_test.go` | test | CRUD | `set/example_test.go` | exact (example test pattern) |
| `cache/mem/mem.go` | provider | CRUD + event-driven | `config/provider.go` | role-match (generic provider + RWMutex) |
| `cache/mem/entry.go` | model | none | `fraction/fraction.go` (lines 11-24) | exact (type struct with generics) |
| `cache/mem/sweeper.go` | utility | event-driven | `config/provider.go` (RWMutex pattern) + RESEARCH.md pattern | partial (no existing sweeper) |
| `cache/mem/mem_test.go` | test | CRUD | `set/set_test.go` | exact (test patterns) |
| `cache/mem/example_test.go` | test | CRUD | `set/example_test.go` | exact (example test pattern) |
| `cache/redis/redis.go` | provider | CRUD + file-I/O | `config/provider.go` | role-match (provider pattern) |
| `cache/redis/options.go` | config | none | `config/options.go` | exact (functional options) |
| `cache/redis/redis_test.go` | test | CRUD | `set/set_test.go` | exact (test patterns) |
| `cache/redis/example_test.go` | test | CRUD | `set/example_test.go` | exact (example test pattern) |
| `cache/memcache/memcache.go` | provider | CRUD + file-I/O | `config/provider.go` | role-match (provider pattern) |
| `cache/memcache/options.go` | config | none | `config/options.go` | exact (functional options) |
| `cache/memcache/memcache_test.go` | test | CRUD | `set/set_test.go` | exact (test patterns) |
| `cache/memcache/example_test.go` | test | CRUD | `set/example_test.go` | exact (example test pattern) |
| `cache/postgres/postgres.go` | provider | CRUD + file-I/O | `config/provider.go` | role-match (provider pattern) |
| `cache/postgres/options.go` | config | none | `config/options.go` | exact (functional options) |
| `cache/postgres/schema.go` | utility | none | `set/marshal.go` (adjacent concern split) | partial (no schema analog) |
| `cache/postgres/sweeper.go` | utility | event-driven | `config/provider.go` (RWMutex) + RESEARCH.md pattern | partial (no existing sweeper) |
| `cache/postgres/postgres_test.go` | test | CRUD | `set/set_test.go` | exact (test patterns) |
| `cache/postgres/example_test.go` | test | CRUD | `set/example_test.go` | exact (example test pattern) |
| `cache/valkey/valkey.go` | provider | CRUD + file-I/O | `config/provider.go` | role-match (provider pattern) |
| `cache/valkey/options.go` | config | none | `config/options.go` | exact (functional options) |
| `cache/valkey/valkey_test.go` | test | CRUD | `set/set_test.go` | exact (test patterns) |
| `cache/valkey/example_test.go` | test | CRUD | `set/example_test.go` | exact (example test pattern) |
| `go.mod` | config | none | existing `go.mod` | exact (add dependencies) |

## Pattern Assignments

### `cache/cache.go` (interface, request-response)

**Analog:** `set/set.go`

The cache interface follows the same generic type conventions as `Set[T comparable]`, but uses `[K comparable, V any]`. The RESEARCH.md already provides the full interface code, but it should follow the exact same style as `set/set.go`.

**Imports pattern** — `set/set.go` lines 1-8:
```go
// Package cache provides a generic cache interface with multiple backend providers.
package cache

import (
    "context"
    "errors"
    "time"
)
```

**Generic type pattern** — `set/set.go` lines 10-13:
```go
// Set values methods
type Set[T comparable] map[T]struct{}
```

**Package doc comment pattern** — `set/set.go` lines 1-3:
```go
// Package set provides a generic Set implementation backed by a Go map.
// Supports standard set operations (Add, Remove, Union, Diff, Intersection),
// iteration, filtering, JSON/YAML marshaling, and SQL database scanning.
```

**Interface to define in `cache/cache.go`** (from RESEARCH.md lines 189-206):
```go
// Cache is a generic key-value cache interface.
// K must be comparable (for in-memory map keys).
// V can be any type; external providers serialize via encoding/json.
type Cache[K comparable, V any] interface {
    Get(ctx context.Context, key K) (V, error)
    Set(ctx context.Context, key K, value V, ttl ...time.Duration) error
    Delete(ctx context.Context, key K) error
    GetOrSet(ctx context.Context, key K, setter func() (V, error), ttl ...time.Duration) (V, error)
    Close() error
}
```

---

### `cache/errors.go` (utility, none)

**Analog:** `fraction/fraction.go` lines 37-53

**Sentinel error pattern** — `fraction/fraction.go` lines 37-53:
```go
var (
    // ErrDivideByZero is returned when trying to divide by a fraction with a value of 0.
    ErrDivideByZero = errors.New("denominator cannot be zero")
    // ErrInvalid is returned when trying to get a fraction from a NaN float.
    ErrInvalid = errors.New("invalid conversion")
)
```

**Error pattern for cache** (RESEARCH.md lines 402-406):
```go
// ErrMiss is returned by Get when the key is not in the cache.
var ErrMiss = errors.New("cache: key not found")

// ErrClosed is returned when operations are attempted on a closed cache.
var ErrClosed = errors.New("cache: cache is closed")
```

**Error wrapping pattern** (all providers) — `fraction/fraction.go` does not wrap, but CONVENTIONS.md specifies:
```go
fmt.Errorf("cache/mem: %w", cache.ErrMiss)
fmt.Errorf("cache/redis: %w", err)
```

---

### `cache/options.go` (config, none)

**Analog:** `config/options.go`

**Functional options pattern** — `config/options.go` lines 13-14, 36-45:
```go
// providerOption is a functional option for configuring a Provider.
type providerOption func(*provider)
```

```go
// WithProfilesPath sets the base directory for YAML profile files.
// Panics if the directory does not exist.
func WithProfilesPath(profilesPath string) providerOption {
    if _, err := os.Stat(profilesPath); err != nil {
        panic(fmt.Errorf("profiles path does not exist: %w", err))
    }

    return func(p *provider) {
        p.profilesPath = profilesPath
    }
}
```

**Options pattern for cache** (RESEARCH.md lines 216-239):
```go
type CacheOption interface {
    apply(cfg *config)
}

type cacheOption func(*config)

// WithDefaultTTL sets the provider-level default TTL for all keys.
func WithDefaultTTL(ttl time.Duration) CacheOption {
    return cacheOption(func(cfg *config) {
        cfg.defaultTTL = ttl
    })
}
```

---

### `cache/cache_test.go` (test, CRUD)

**Analog:** `set/set_test.go`

**Test structure pattern** — `set/set_test.go` lines 10-127:
```go
package set_test

import (
    "testing"

    "github.com/guionardo/go/set"
    "github.com/stretchr/testify/assert"
)

func TestSet_Set(t *testing.T) { //nolint:funlen
    t.Parallel()
    t.Run("create_new_should_be_empty", func(t *testing.T) {
        t.Parallel()
        s := set.New[int]()
        assert.Empty(t, s)
    })
    t.Run("create_new_with_values_should_have_correct_length", func(t *testing.T) {
        t.Parallel()
        s := set.New(1, 2, 3)
        assert.Len(t, s, 3)
    })
    // ...
}
```

**Key testing patterns to copy:**
- External test package: `package cache_test`
- `t.Parallel()` on every subtest
- `//nolint:funlen` suppression on long test functions
- `assert.*` for soft assertions, `require.*` for hard assertions
- Table-driven tests for parameterized cases (see TESTING.md lines 75-100)
- `require.ErrorIs(t, err, sentinelErr)` for sentinel error checks

---

### `cache/example_test.go` (test, CRUD)

**Analog:** `set/example_test.go`

**Example test pattern** — `set/example_test.go` lines 1-17:
```go
package set_test

import (
    "fmt"

    "github.com/guionardo/go/set"
)

func ExampleNew() {
    s := set.New(1, 2, 3, 2, 1)
    fmt.Println(s.Has(1))
    fmt.Println(s.Has(4))

    // Output:
    // true
    // false
}
```

**Key pattern:** External test package, `ExampleFunctionName()` naming, `// Output:` annotations at end.

---

### `cache/mem/mem.go` (provider, CRUD + event-driven)

**Analog:** `config/provider.go` (RWMutex + generic struct)

**RWMutex pattern** — `config/provider.go` lines 20-26:
```go
type Provider[T any] struct {
    provider
    lock          sync.RWMutex
    configuration T
    loaded        bool
}
```

**Constructor with functional options** — `config/provider.go` lines 41-57:
```go
func NewProvider[T any](options ...providerOption) *Provider[T] {
    provider := &provider{
        defaultScope: DefaultScope,
        scope:        environment.GetEnv(EnvScope, DefaultScope),
        profilesPath: environment.GetEnv(EnvProfilesPath),
    }
    for _, option := range options {
        option(provider)
    }
    return &Provider[T]{provider: *provider.postInit()}
}
```

**In-memory provider pattern** (RESEARCH.md lines 437-535):
```go
type Cache[K comparable, V any] struct {
    mu         sync.RWMutex
    entries    map[K]*entry[V]
    defaultTTL time.Duration
    stop       chan struct{}
}

func New[K comparable, V any](opts ...Option) *Cache[K, V] {
    c := &Cache[K, V]{
        entries: make(map[K]*entry[V]),
        stop:    make(chan struct{}),
    }
    for _, opt := range opts {
        opt(c)
    }
    if c.defaultTTL == 0 {
        c.defaultTTL = 5 * time.Minute
    }
    go c.sweepLoop()
    return c
}
```

**Get with passive TTL check** (RESEARCH.md lines 459-479):
```go
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
    c.mu.RLock()
    e, ok := c.entries[key]
    c.mu.RUnlock()
    if !ok {
        var zero V
        return zero, fmt.Errorf("cache/mem: %w", cache.ErrMiss)
    }
    if e.expiresAt != nil && time.Now().After(*e.expiresAt) {
        c.mu.Lock()
        delete(c.entries, key)
        c.mu.Unlock()
        var zero V
        return zero, fmt.Errorf("cache/mem: %w", cache.ErrMiss)
    }
    return e.value, nil
}
```

**Set/Delete pattern** (RESEARCH.md lines 481-498):
```go
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
    expiresAt := c.resolveTTL(ttl...)
    c.mu.Lock()
    c.entries[key] = &entry[V]{value: value, expiresAt: expiresAt}
    c.mu.Unlock()
    return nil
}

func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
    c.mu.Lock()
    delete(c.entries, key)
    c.mu.Unlock()
    return nil
}

func (c *Cache[K, V]) Close() error {
    close(c.stop)
    return nil
}
```

---

### `cache/mem/entry.go` (model, none)

**Analog:** `fraction/fraction.go` lines 11-24 (grouped type declaration)

**Grouped type with generics pattern:**
```go
type (
    // entry holds a cached value with optional expiration.
    entry[V any] struct {
        value     V
        expiresAt *time.Time // nil = no expiry
    }
)
```

---

### `cache/mem/sweeper.go` (utility, event-driven)

**Analog:** No direct existing analog — RESEARCH.md provides the pattern.

**Sweeper pattern** (RESEARCH.md lines 513-535):
```go
func (c *Cache[K, V]) sweepLoop() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            c.sweep()
        case <-c.stop:
            return
        }
    }
}

func (c *Cache[K, V]) sweep() {
    c.mu.Lock()
    defer c.mu.Unlock()
    now := time.Now()
    for k, e := range c.entries {
        if e.expiresAt != nil && now.After(*e.expiresAt) {
            delete(c.entries, k)
        }
    }
}
```

**Key pattern:** Channel-based stop mechanism (not context), `time.NewTicker` with `defer ticker.Stop()`, `select` for tick vs stop signals.

---

### `cache/redis/redis.go` (provider, CRUD + file-I/O)

**Analog:** `config/provider.go` (provider constructor + functional options)

**Constructor pattern** — reuses `config/options.go` functional option pattern:
```go
type Option func(*Config)

type Cache[K comparable, V any] struct {
    client     *redis.Client
    defaultTTL time.Duration
}

func New[K comparable, V any](opts ...Option) *Cache[K, V] {
    cfg := &Config{
        Addr: "localhost:6379",
    }
    for _, opt := range opts {
        opt(cfg)
    }
    client := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr,
        Password: cfg.Password,
        DB:       cfg.DB,
        PoolSize: cfg.PoolSize,
    })
    return &Cache[K, V]{client: client, defaultTTL: cfg.DefaultTTL}
}
```

**Get with JSON deserialization** (RESEARCH.md lines 587-603):
```go
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
    data, err := c.client.Get(ctx, fmt.Sprint(key)).Bytes()
    if err == redis.Nil {
        var zero V
        return zero, fmt.Errorf("cache/redis: %w", cache.ErrMiss)
    }
    if err != nil {
        var zero V
        return zero, fmt.Errorf("cache/redis: %w", err)
    }
    var value V
    if err := json.Unmarshal(data, &value); err != nil {
        var zero V
        return zero, fmt.Errorf("cache/redis: %w", err)
    }
    return value, nil
}
```

**Set with JSON serialization** (RESEARCH.md lines 605-615):
```go
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("cache/redis: %w", err)
    }
    expiration := c.resolveTTL(ttl...)
    if err := c.client.Set(ctx, fmt.Sprint(key), data, expiration).Err(); err != nil {
        return fmt.Errorf("cache/redis: %w", err)
    }
    return nil
}
```

**Close pattern** (RESEARCH.md lines 617-619):
```go
func (c *Cache[K, V]) Close() error {
    return c.client.Close()
}
```

**Key pattern for `cache/redis/redis.go`:** External providers wrap errors with `"cache/redis: %w"`. Redis `nil` responses are special-cased to `cache.ErrMiss`. Key conversion uses `fmt.Sprint(key)`.

---

### `cache/redis/options.go` (config, none)

**Analog:** `config/options.go`

**Provider-specific options pattern:**
```go
type Config struct {
    Addr        string
    Password    string
    DB          int
    PoolSize    int
    DefaultTTL  time.Duration
}

type Option func(*Config)

func WithAddr(addr string) Option {
    return func(cfg *Config) {
        cfg.Addr = addr
    }
}

func WithPoolSize(n int) Option {
    return func(cfg *Config) {
        cfg.PoolSize = n
    }
}

func WithDefaultTTL(ttl time.Duration) Option {
    return func(cfg *Config) {
        cfg.DefaultTTL = ttl
    }
}
```

**Pattern repeated for:** `cache/memcache/options.go`, `cache/postgres/options.go`, `cache/valkey/options.go` — same structure, different config fields per provider.

---

### `cache/memcache/memcache.go` (provider, CRUD + file-I/O)

**Analog:** `cache/redis/redis.go` (same external-provider pattern) + `config/provider.go`

**Key differences from Redis pattern:**
- Uses `bradfitz/gomemcache` which does NOT support `context.Context`
- Must wrap calls in goroutine for context cancellation (RESEARCH.md lines 344-361):
```go
type result struct {
    item *memcache.Item
    err  error
}
ch := make(chan result, 1)
go func() {
    item, err := mc.Get(key)
    ch <- result{item, err}
}()
select {
case r := <-ch:
    return r.item, r.err
case <-ctx.Done():
    return nil, ctx.Err()
}
```

---

### `cache/postgres/postgres.go` (provider, CRUD + file-I/O)

**Analog:** `config/provider.go` + `cache/redis/redis.go`

**Key differences:**
- Uses `pgx/v5` with `pgxpool` for connection pooling
- TTL checking in SQL query: `SELECT ... WHERE expires_at IS NULL OR expires_at > NOW()`
- Background sweep deletes expired rows
- Table schema defined in separate `schema.go`

---

### `cache/postgres/schema.go` (utility, none)

**Analog:** No existing schema file in codebase. Research provides the SQL constant.

**Pattern** (RESEARCH.md lines 624-635):
```go
package postgres

// CreateTableSQL is the SQL statement for creating the cache entries table.
const CreateTableSQL = `
CREATE TABLE IF NOT EXISTS cache_entries (
    cache_key   TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at
    ON cache_entries (expires_at)
    WHERE expires_at IS NOT NULL;
`
```

---

### `cache/postgres/sweeper.go` (utility, event-driven)

**Analog:** `cache/mem/sweeper.go` (same pattern, different sweep implementation)

**Key difference:** Postgres sweeper uses SQL `DELETE` instead of map manipulation:
```go
func (c *Cache[K, V]) sweep() {
    _, err := c.pool.Exec(context.Background(),
        `DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at < NOW()`)
    if err != nil {
        // Log error but don't return — sweep is best-effort
        slog.Warn("cache/postgres: sweep failed", "error", err)
    }
}
```

---

### `cache/valkey/valkey.go` (provider, CRUD + file-I/O)

**Analog:** `cache/redis/redis.go` (wire-compatible protocol, similar API)

**Key patterns:**
- Same error wrapping pattern: `"cache/valkey: %w"`
- Same JSON serialization pattern as Redis
- Uses `valkey-go` client API instead of `go-redis/v9`
- Close pattern delegates to `valkey.Client.Close()`

---

## Shared Patterns

### 1. Imports Organization

**Source:** `set/set.go` (and all existing packages)
**Apply to:** All new cache files

**Order** (from CONVENTIONS.md lines 66-81):
1. Standard library (no blank lines within)
2. Blank line
3. Third-party packages
4. Blank line
5. Internal project packages (`github.com/guionardo/go/...`)

**Example for `cache/redis/redis.go`:**
```go
import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"

    "github.com/guionardo/go/cache"
)
```

### 2. Error Wrapping

**Source:** CONVENTIONS.md lines 128-148
**Apply to:** All provider files

```go
// All provider errors MUST be wrapped with provider prefix:
fmt.Errorf("cache/mem: %w", cache.ErrMiss)
fmt.Errorf("cache/redis: %w", err)
fmt.Errorf("cache/memcache: %w", err)
fmt.Errorf("cache/postgres: %w", err)
fmt.Errorf("cache/valkey: %w", err)

// Sentinel errors defined as package-level vars:
var (
    ErrMiss   = errors.New("cache: key not found")
    ErrClosed = errors.New("cache: cache is closed")
)
```

### 3. Generic Type Pattern for Providers

**Source:** `config/provider.go` (lines 20-26) + `set/set.go` (line 11)
**Apply to:** All provider structs

```go
// All providers use [K comparable, V any] generics.
// K must be comparable (needed for in-memory map keys).
// External providers convert K to string via fmt.Sprint(key).
type Cache[K comparable, V any] struct { ... }
```

### 4. Multi-file Package Organization

**Source:** `set/` package (set.go, marshal.go, scanner_valuer.go)
**Apply to:** All cache sub-packages

```go
// Each provider splits concerns across files:
//   options.go   — functional options + config struct
//   <provider>.go — New, Get, Set, Delete, GetOrSet, Close
//   entry.go     — entry struct (in-memory only)
//   sweeper.go   — background sweep (mem, postgres only)
//   schema.go    — SQL constants (postgres only)
```

### 5. Package Doc Comments

**Source:** Convention across all existing packages
**Apply to:** Every `.go` file in the cache package tree

```go
// Package cache provides a generic cache interface with multiple backend providers.
// ...
```

### 6. Constructor with Functional Options

**Source:** `config/provider.go` lines 41-57
**Apply to:** All five provider constructors

```go
func New[K comparable, V any](opts ...Option) *Cache[K, V] {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    return &Cache[K, V]{...}
}
```

### 7. TTL Resolution

**Source:** RESEARCH.md lines 501-511
**Apply to:** All providers that support TTL

```go
func (c *Cache[K, V]) resolveTTL(ttl ...time.Duration) time.Duration {
    if len(ttl) > 0 && ttl[0] > 0 {
        return ttl[0]
    }
    return c.defaultTTL
}
```

### 8. Testing Patterns

**Source:** `set/set_test.go`, `set/example_test.go`, TESTING.md
**Apply to:** All `*_test.go` files

```go
package mem_test

import (
    "testing"
    "github.com/guionardo/go/cache/mem"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMemCache_Set(t *testing.T) { //nolint:funlen
    t.Parallel()
    t.Run("set_and_get_should_return_value", func(t *testing.T) {
        t.Parallel()
        c := mem.New[string, string]()
        err := c.Set(context.Background(), "key", "value")
        require.NoError(t, err)
        got, err := c.Get(context.Background(), "key")
        require.NoError(t, err)
        assert.Equal(t, "value", got)
    })
    // ...
}
```

**Example test pattern:**
```go
package mem_test

import (
    "fmt"
    "github.com/guionardo/go/cache/mem"
)

func ExampleNew() {
    c := mem.New[string, string]()
    _ = c.Set(nil, "hello", "world")
    val, _ := c.Get(nil, "hello")
    fmt.Println(val)
    // Output:
    // world
}
```

### 9. Key Conversion for External Providers

**Source:** RESEARCH.md lines 588, 611
**Apply to:** All external providers (Redis, Memcache, Postgres, Valkey)

```go
// Keys are converted to string for external backends
fmt.Sprint(key)
```

### 10. JSON Serialization for External Providers

**Source:** RESEARCH.md lines 598-601, 606-608
**Apply to:** All external providers

```go
// Set: marshal value
data, err := json.Marshal(value)

// Get: unmarshal value into typed variable
var value V
if err := json.Unmarshal(data, &value); err != nil { ... }
```

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `cache/mem/sweeper.go` | utility | event-driven | No existing background goroutine pattern in codebase |
| `cache/postgres/sweeper.go` | utility | event-driven | Same — use RESEARCH.md pattern |
| `cache/postgres/schema.go` | utility | none | No existing embedded SQL constant pattern in codebase |

## Metadata

**Analog search scope:**
- `config/options.go` — functional options pattern
- `config/provider.go` — generic Provider[T] with sync.RWMutex
- `config/provider_base.go` — internal provider struct
- `set/set.go` — generic type conventions
- `set/example_test.go` — example test pattern
- `set/set_test.go` — test structure pattern
- `fraction/fraction.go` — sentinel errors, grouped type declarations
- `fraction/example_test.go` — example test pattern
- `.planning/codebase/CONVENTIONS.md` — coding conventions
- `.planning/codebase/TESTING.md` — testing patterns
- `.planning/codebase/STRUCTURE.md` — file/directory layout

**Files scanned:** 10 existing analog files + 3 codebase documents
**Pattern extraction date:** 2026-07-21

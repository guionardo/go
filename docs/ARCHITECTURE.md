<!-- generated-by: gsd-doc-writer -->
# Architecture

## System Overview

`github.com/guionardo/go` is a **Go module** (Go `1.26.4`) that hosts a collection of small, independent, reusable utility packages. It is not a single application but a library monorepo: each top-level directory is a self-contained package that may be imported alone (`cache`, `config`, `flow`, `set`, `release`, etc.), with no shared runtime or framework coupling between them.

The architectural style is **package-isolation by capability**. Individual packages follow a consistent *pluggable provider* pattern (the `cache` and `config` packages expose a stable generic interface with swappable/backed implementations), a *library + standalone helper binary* pattern (`release` embeds a cross-compiled `swapper` binary for atomic self-update), and otherwise stand-alone value packages (`fraction`, `set`, `time_tools`, `br_docs`, `path_tools`, `reflect_tools`, `shell_tools`, `httptest_mock`, `mid`, `flow`). The primary inputs are caller requests (cache read/write, config load/update, self-update check) and outputs are typed values or concrete effects (cached data, validated configuration, replaced binaries).

## Component Diagram

The module is divided into four broad groups. The `cache` and `config` packages are internally layered; the rest are leaf packages.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           github.com/guionardo/go                        │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
       ┌────────────────────────────┼────────────────────────────┐
       │                            │                            │
 ┌─────▼─────┐              ┌───────▼───────┐            ┌──────▼──────┐
 │  cache    │              │    config     │            │   release   │
 │ (generic) │              │   (generic)   │            │  (self-upd) │
 └─────┬─────┘              └───────┬───────┘            └──────┬──────┘
       │                            │                           │
 Cache[K,V] interface         Provider[T] struct            calls GitHub
       │                            │                      Releases API ...
 ┌─────▼────────────────┐    ┌──────▼─────────────────────┐
 │ concreteCache +        │    │ profile ──► merger        │
 │ SingleflightGetOrSet   │    │ environment ─► validation │
 └─────┬────────────────┘    └────────────────────────────┘
       │                                  ▲
       │ implements                       │ merges default + scope YAML
 ┌─────▼─────────────────────────┐        │
 │ cache/mem    (in-memory)      │        │
 │ cache/redis  (Redis)          │        │
 │ cache/valkey (Valkey)         │        │
 │ cache/memcache (Memcache)     │        │
 │ cache/postgres (Postgres)     │        │
 └───────────────────────────────┘        │
                                          │
 ┌───────────────────────────────────────────────────────────────┐
 │                  Leaf utility packages                       │
 │ br_docs  flow  fraction  set  httptest_mock                  │
 │ mid path_tools reflect_tools shell_tools time_tools           │
 └───────────────────────────────────────────────────────────────┘
```

## Data Flow

Because this is a library rather than a running service, "data flow" is described per major package.

**Cache — `cache.GetOrSet` (leader/deduplication flow):**
1. Caller invokes `Cache.GetOrSet(ctx, key, setter, ttl...)`.
2. `concreteCache.GetOrSet` delegates to `SingleflightGetOrSet.Do`.
3. `Do` runs a **fast-path** `Get` *outside* the singleflight group (`cache/singleflight.go`): a hit returns immediately with no lock contention.
4. On a miss, `group.Do` runs the group function once (**singleflight** deduplicates concurrent misses on the same key):
   - **double-check** `Get` so a concurrent direct `Set` is not clobbered,
   - run the `setter` in the leader's goroutine (with panic recovery via `callSetter` → `*cache.Panic`),
   - call the provider's `Set` with the verbatim TTL (leader's TTL wins on conflict).
5. Every concurrent waiter receives the identical computed value or error. `DoChan` adds per-caller cancellation: a canceled waiter abandons its wait promptly and receives an error wrapping both `cache.ErrCanceled` and its own `ctx.Err()`.

**Config — `Provider.GetConfiguration()` (config/provider.go):**
1. First call acquires the write lock and invokes `loadStaticConfiguration`.
2. `profile.GetScopedProfileContent` loads two YAML files (default scope + active scope) and merges them recursively via `config/merger`; the result is unmarshaled into the target struct `T`.
3. `environment.ParseEnvironment` overlays `env`/`default` struct tags from the process environment.
4. The assembled struct is validated (via `config/validation`) and cached; later calls return the cached value under a read lock.

### Release — `PerformSelfUpdate` (release/self_update.go):
1. Resolve the current version from `debug.ReadBuildInfo()`.
2. `CheckForUpdate` calls the GitHub Releases API to fetch the latest release and compares via `hashicorp/go-version` (owner/repo derive from the module path if not given).
3. If a release is newer, `acquireUpdateLock` (a lock file next to the executable) prevents concurrent updates.
4. `DownloadUpdate` picks the asset matching `runtime.GOOS`/`runtime.GOARCH` and verifies its digest (go-digest).
5. `ExtractSwapper` writes the embedded `swapper` binary from the `embed.FS`, then the parent spawns the swapper and exits.
6. The swapper performs a backup-rename-replace, verifies the SHA256 pre- and post-swap, optionally restores the backup on failure, and relaunches the new binary with the original CLI args.

## Key Abstractions

| Abstraction | Location | Description |
|-------------|----------|-------------|
| `Cache[K, V]` interface | `cache/cache.go` | Generic key-value cache contract: `Get`, `Set`, `Delete`, `GetOrSet` are context-aware; `Close` takes no context. |
| `SingleflightGetOrSet[K, V]` | `cache/singleflight.go` | Deduplicates concurrent `GetOrSet` misses on the same key; provides cancel-aware `DoChan`. |
| `NewConcreteCache` / `cacher` | `cache/concrete_cache.go` | Bridges the public `Cache` interface to provider-specific `GetFunc/SetFunc/DeleteFunc/CloseFunc`. |
| `Panic` error type | `cache/errors.go` | Wrapay recovered setter panics (mirrors `x/sync` singleflight's `panicError`); `Unwrap` traverses the recovered error. |
| `Provider[T]` | `config/provider.go` | Generic typed configuration provider; thread-safe `GetConfiguration`/`UpdateConfiguration` backed by YAML profiles + env + validation. |
| `Logger` interface | `config/provider.go` | Logging seam (implemented by `log/slog`, `testing.T`, or custom) used for configuration events. |
| `UpdateResult`, `UpdateState` | `release/self_update.go` | Result struct + state enum describing the entry of a self-update attempt. |
| `Config` + `Option` | `release/update.go` | Functional-options pattern for `CheckForUpdate`/`PerformSelfUpdate` (`WithOwner`, `WithRepo`, `WithGitHubToken`). |
| Swapper standalone binary | `release/swapper/main.go` | Compile-time embedded helper (`embed.FS`) performing atomic replace + relaunch. |

## Directory Structure Rationale

All directories map one-to-one to an importable package; nesting denotes a provider/layer relationship. Consumers can import only the package they need without pulling in unrelated code.

```text
cache/            Generic cache interface + SingleflightGetOrSet; safe single binary.
  mem/            In-memory backend (stdlib, zero dependency).
  redis/          Redis backend (go-redis/v9), lazy dial.
  valkey/         Valkey backend (valkey-go), eager dial.
  memcache/       Memcache backend (gomemcache).
  postgres/       Postgres backend (pgx/v5, pgxpool at construction).
config/           Generic typed config provider (YAML profiles + env + validation).
  environment/    Env-var parsing into struct fields via tags.
  profile/        Scope-based YAML profile discovery and merging.
  merger/         Recursive deep-merge of map[string]any.
  validation/     Struct validation (go-playground/validator).
release/          Self-update toolkit for CLI tools via GitHub Releases.
  swapper/        Standalone helper binary for atomic replace + relaunch.
cmd/               Runnable examples (example-updater).
br_docs/           CPF/CNPJ validation.
flow/              Generic control flow (ternary, defaults).
fraction/          Immutable fraction arithmetic.
httptest_mock/     HTTP mock server framework for tests.
mid/               Cross-platform machine ID.
path_tools/        File/dir path utilities.
reflect_tools/     Reflection utilities.
set/               Generic set with algebra/JSON/SQL support.
shell_tools/       Shell argument parsing, env lookup.
time_tools/        Adaptive time format parsing.
```

**Rationale notes:**
- `cache` and `config` keep thin public interfaces in the base package and push all backend/loader details into sub-packages, so consumers depend on the smallest surface possible and new backends can be added without touching the public API.
- `release/swapper` is a separate `main` package cross-compiled for four platforms (`linux_amd64`, `darwin_amd64`, `darwin_arm64`, `windows_amd64`) and embedded into the library; it must remain a leaf with no import dependencies so it can be built independently via `make swapper`.
- Most leaf packages (`fraction`, `set`, `time_tools`, …) have no internal dependencies on sibling packages, reinforcing the "import only what you need" design and keeping the module easy to vet/test in isolation. The notable exception is `httptest_mock`, which imports `flow` (`httptest_mock/request.go`) and `reflect_tools` (`httptest_mock/string_parts.go`).
<!-- generated-by: gsd-doc-writer -->
# Configuration

This document describes every configuration surface in the `go` utility monorepo: the `config` package (a generic, typed configuration provider) and the per-provider option structs exposed by the `cache` package.

## Overview

Two distinct configuration mechanisms exist:

- **`config` package** — a typed `Provider[T]` that loads configuration from YAML scope profiles, overlays environment-variable overrides (via struct tags), and validates the result. See `config/provider.go`, `config/options.go`, and the sub-packages `config/environment`, `config/profile`, `config/merger`, and `config/validation`.
- **`cache` package** — each provider (`mem`, `redis`, `valkey`, `memcache`, `postgres`) exposes functional options plus a `Config` struct. The `mem` provider uses a `ConfigFunc`-based option list (`WithDefaultTTL`, `WithMaxEntries`, `WithSweepInterval`).

## Config provider: environment variables

The following environment variables control how the `config.Provider[T]` locates and selects its configuration source. They are read at provider construction time (`config/consts.go`, `config/provider.go`, `config/options.go`).

| Variable            | Required | Default     | Description                                                                                                                  |
|---------------------|----------|-------------|------------------------------------------------------------------------------------------------------------------------------|
| `SCOPE`             | Optional | `default`   | Active scope name used to select a scope-specific YAML profile. Constant `EnvScope`.                                         |
| `PROFILES_PATH`     | Optional | *(empty)*   | Overrides the base directory where YAML profile files are looked up. Constant `EnvProfilesPath`. Read during construction and via `getProfilesPath()`. |
| `CONFIGURATION_LOG` | Optional | `./CONFIGS` | Fallback used to derive `profilesPath` when it would otherwise be empty. Constant `EnvConfigurationLog`.                     |
| `DEFAULT_SCOPE`     | Optional | —           | Declared as constant `EnvDefaultScope = "DEFAULT_SCOPE"` but not currently wired into the loading logic. Not honored by the provider. |

> **Behavior note:** `CONFIGURATION_LOG` is technically used as a fallback for the *profiles path* in `postInit()` (`config/options.go`), not for enabling configuration logging as its name suggests. Prefer `PROFILES_PATH` or the `WithProfilesPath` option to control the profile directory.

When the profiles path resolves to an empty string, `loadStaticConfiguration()` skips profile loading and logs `no profiles path found, skipping profile loading` (`config/provider.go`).

## Config file format (YAML profiles)

The provider reads YAML profile files from a base directory (the "profiles path", `./CONFIGS` by default). Profiles are named after their scope. For a given active `SCOPE` and default scope, the provider locates files by trying the suffixes `""`, `.yml`, `.yaml`, `.YML`, `.YAML` in order for each scope name.

Two files are merged:

1. The **default-scope** file (e.g., `default.yml`).
2. The **active-scope** file (e.g., `production.yml` or `development.yml`).

The scope files are deep-merged with `config/merger.MergeMaps` (nested maps merge recursively; scalar values from the *active-scope* file replace those of the default file). Scope names that escape the base directory are rejected.

The YAML keys map to struct fields via the `yaml:"..."` tag.

```yaml
# ./CONFIGS/default.yml
app:
  name: my-app
  version: 1
db:
  timeout: 30s

# ./CONFIGS/production.yml  (overrides default when SCOPE=production)
app:
  name: my-app-prod
```

## Struct-tag configuration schema

Fields of the `T` struct passed to `config.NewProvider[T]` are populated from environment variables and validated using struct tags. `ParseEnvironment` recurses into nested struct fields automatically.

| Tag               | Purpose                                                                    |
|-------------------|----------------------------------------------------------------------------|
| `env:"NAME"`      | Environment variable name to read the field value from. If the var is unset/empty, the `default` tag is used. |
| `default:"value"` | Fallback value applied when the `env` variable is unset.                   |
| `yaml:"key"`      | YAML key for profile files.                                                |
| `validate:"..."`  | Validation rule applied via go-playground/validator (e.g., `validate:"required"`). |
| `safe:"true"`     | Redacts the field in log output as `********` (`config/logging.go`).       |

Supported scalar kinds: `string`, `int`/`int8`–`int64`, `uint`, `bool`, `float32`/`float64`, and nested structs. Env lookup is case-insensitive.

```go
type AppConfig struct {
	Name    string `yaml:"name" env:"APP_NAME" default:"app"`
	Version int    `yaml:"version" env:"APP_VERSION"`
	Secret  string `yaml:"secret" safe:"true"` // masked in logs as ********
	Port    int    `yaml:"port" env:"APP_PORT" default:"8080"`
}
```

## Provider options (functional options)

`config.NewProvider[T]` accepts the following options (`config/options.go`):

| Option                    | Description                                                                            |
|---------------------------|----------------------------------------------------------------------------------------|
| `WithProfilesPath(path)`  | Sets the YAML profile directory. Panics if the directory does not exist.               |
| `WithScope(scope)`        | Sets the active scope name (e.g., `"production"`).                                     |
| `WithDefaultScope(scope)` | Sets the fallback scope name.                                                          |
| `WithLogger(logger)`      | Injects a custom `Logger` (can wrap `slog`, `testing.T.Logf`, etc.).                   |
| `WithDebugLogger()`       | Enables debug-level logging. Warns that it should not be used in production.           |

## Required vs optional settings

There is **no built-in required set of environment variables** for the provider itself — it constructs successfully with all defaults. A struct field becomes **required** only when its `validate` tag declares it (e.g., `validate:"required"`). If a required field is missing at load time, `GetConfiguration()` / `UpdateConfiguration()` return an error and the configuration is not applied.

Required-field failures surface from `validateConfiguration` (`config/provider_base.go`) — either the type's own `Validator.Validate()` method or the go-playground `validator.Struct` call.

```go
cfg, err := provider.GetConfiguration()
// err != nil when a `validate:"required"` field is absent
```

## Defaults

Defaults come from three layers: struct tags, provider code, and cache provider `Config` defaults.

**Config provider defaults:**

| Setting        | Default     | Source                                                                   |
|----------------|-------------|--------------------------------------------------------------------------|
| `scope`        | `default`   | `config/consts.DefaultScope`, applied when `SCOPE` is unset              |
| `defaultScope` | `default`   | `config/consts.DefaultScope`                                              |
| `profilesPath` | `./CONFIGS` | `config/consts.DefaultConfigurationPath`, via `CONFIGURATION_LOG` fallback |

**Cache providers** (defaults from each provider's `defaultConfig()` and `cache/mem/mem.go`):

| Provider      | Setting                | Default                    |
|---------------|------------------------|----------------------------|
| `cache` (shared) | `DefaultTTL`         | *(zero — no TTL)*          |
| `mem`         | `DefaultTTL`           | `5m`                       |
| `mem`         | `MaxEntries`           | `1000`                     |
| `mem`         | `SweepInterval`        | `1m`                       |
| `redis`       | `Addr`                 | `localhost:6379`           |
| `redis`       | `DB`                   | `0`                        |
| `redis`       | `PoolSize`             | `10`                       |
| `valkey`      | `Addr`                 | `localhost:6379`           |
| `valkey`      | `DB`                   | `0`                        |
| `valkey`      | `PoolSize`             | `10`                       |
| `memcache`    | `Servers`              | `["localhost:11211"]`      |
| `memcache`    | `Timeout`              | `100ms`                    |
| `memcache`    | `MaxIdleConns`         | `2`                        |
| `postgres`    | `TableName`            | `cache_entries`            |
| `postgres`    | `PoolSize`             | `5`                        |
| `postgres`    | `SweepInterval`        | `1m`                       |

A provider `DefaultTTL` is only used when a call does not supply an explicit TTL; a zero value disables expiry. The `mem` provider applies its defaults in `New(...)` (`cache/mem/mem.go`), which also starts a background sweep goroutine when `SweepInterval` is configured (unless a nil `ctx` is passed).

## Cache per-provider option functions

Each cache provider exposes typed functional options:

- **`cache` (shared):** `WithDefaultTTL(ttl time.Duration)` — `cache/options.go`.
- **`mem`:** `WithDefaultTTL`, `WithMaxEntries(max)`, `WithSweepInterval(interval)` — declared as `ConfigFunc` in `cache/mem/config.go`.
- **`redis`:** `WithAddr`, `WithPassword`, `WithDB(db)`, `WithPoolSize(n)`, `WithDefaultTTL` — `cache/redis/options.go`.
- **`valkey`:** `WithAddr`, `WithPassword`, `WithDB(db)`, `WithPoolSize(n)`, `WithDefaultTTL` — `cache/valkey/options.go`.
- **`memcache`:** `WithServers(servers...)`, `WithTimeout`, `WithDefaultTTL`, `WithMaxIdleConns(n)` — `cache/memcache/options.go`.
- **`postgres`:** `WithConnString`, `WithTableName`, `WithPoolSize(n)`, `WithSweepInterval(d)`, `WithDefaultTTL` — `cache/postgres/options.go`.

### Example

```go
// In-memory cache with explicit configuration.
c := mem.New[int, string](ctx,
    mem.WithDefaultTTL(2*time.Minute),
    mem.WithMaxEntries(5000),
    mem.WithSweepInterval(30*time.Second),
)

// Redis cache with connection tuning.
r := redis.New(ctx,
    redis.WithAddr("redis.internal:6379"),
    redis.WithDB(3),
    redis.WithPoolSize(20),
    redis.WithDefaultTTL(time.Hour),
)
```

## Per-environment overrides

Environment selection is expressed as a **scope**. Set the active scope via the `SCOPE` env var or the `WithScope` option, and choose a matching profile file name (`{scope}.yml`) beside the default scope file in the profiles path:

- **Development:** `SCOPE=development` reads `./CONFIGS/development.yml`.
- **Staging:** `SCOPE=staging` reads `./CONFIGS/staging.yml`.
- **Production:** `SCOPE=production` reads `./CONFIGS/production.yml`.

Each environment file is deep-merged over the default-scope file, and **environment variables listed in `env` tags are applied last**, so they take precedence over profile files. For deployment-specific secret values (database credentials, API keys, cloud region identifiers), set them via environment variables or the deployment platform's secret manager rather than in profile YAML.

<!-- VERIFY: deployment-specific secret values (production URLs, cloud regions, cluster names) are environment-specific and cannot be inferred from the repository. Set them per-deployment via your secret manager or `env` tag overrides. -->

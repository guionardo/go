# config

Typed configuration for Go applications: YAML profiles layered by scope, environment
variable overrides applied through struct tags, and struct validation on every load and
update. Reach for it when a service needs one typed config object that can come from
files, the environment, and a runtime update without a custom loader.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/config)

## Install

```bash
go get github.com/guionardo/go/config
```

## Usage

```go
import "github.com/guionardo/go/config"

type AppConfig struct {
	Port   int    `env:"APP_PORT" default:"8080"`
	Host   string `env:"APP_HOST" default:"localhost"`
	DBPath string `env:"DB_PATH"`
}

func main() {
	provider := config.NewProvider[AppConfig]()
	cfg, err := provider.GetConfiguration()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Listening on %s:%d\n", cfg.Host, cfg.Port)
}
```

`GetConfiguration` loads once and caches the result; later calls return the same value.
Use `UpdateConfiguration` to replace it at runtime.

## Provider

`Provider[T]` is safe for concurrent use.

```go
func NewProvider[T any](options ...providerOption) *Provider[T] // providerOption is unexported — pass the option constructors
func (p *Provider[T]) GetConfiguration() (T, error)
func (p *Provider[T]) UpdateConfiguration(configuration T) error
func (p *Provider[T]) HTTPHandler() http.Handler
```

- `NewProvider` panics when `T` is not a struct.
- `UpdateConfiguration` validates before swapping in the new value; an identical
  configuration is accepted without logging an update.
- `HTTPHandler` accepts `PATCH`/`POST` with a JSON body, applies it through
  `UpdateConfiguration`, and answers `200` with the updated configuration as JSON.
  It returns `405` for other methods, `400` for invalid JSON, and `422` when validation
  fails.

```go
mux.Handle("/config", provider.HTTPHandler())
```

## Options

- `WithProfilesPath(path)` — base directory for YAML profile files. Panics if the
  directory does not exist.
- `WithSingleConfigFile(file)` — load one YAML file instead of a default/scope pair.
  The directory part becomes the profiles path, the file's base name becomes both scopes,
  and `SCOPE` is ignored. The extension is optional (`.yml`, `.yaml`, `.YML`, `.YAML` are
  resolved). Panics if no usable file name is given.
- `WithScope(scope)` — active scope name (e.g. `production`, `development`).
- `WithDefaultScope(scope)` — fallback scope name.
- `WithLogger(logger)` — supply a `Logger`. See [Notes](#notes).
- `WithDebugLogger()` — build a debug-level `slog` logger writing to stdout and emit a
  warning that it must not be used in production. Not wired into the provider's log
  output either; see [Notes](#notes).

```go
provider := config.NewProvider[AppConfig](
	config.WithSingleConfigFile("app.yml"),
	config.WithScope("production"),
)
```

## Configuration files

The default scope profile is the base; the active scope profile is deep-merged over it:

```
CONFIGS/
  default.yml     # base values
  production.yml  # overrides default.yml when SCOPE=production
```

Resolution order:

1. Profiles path: `WithProfilesPath`/`WithSingleConfigFile`, then `PROFILES_PATH`, then
   `./CONFIGS`.
2. Scope: `WithScope`, then `SCOPE`, then the default scope (`default`).
3. Environment variables override whatever the profiles produced.
4. The result is validated.

A profile that is missing or unreadable is logged as a warning and skipped, so the
configuration falls back to environment variables and `default` tags. Malformed YAML is
returned as an error.

## Environment variables

- `PROFILES_PATH` — overrides the profiles directory.
- `SCOPE` — overrides the active scope.

Fields are populated from `env` and `default` struct tags: the environment variable wins
when set and non-empty, otherwise the `default` tag applies. Strings, signed and unsigned
integers, floats, and bools are supported, and nested structs are parsed recursively. An
unparseable value is returned as an error naming the field.

## Validation

`validateConfiguration` runs a type's own `Validate() error` method (the
`validation.Validator` interface) and then `go-playground/validator` over the
`validate` struct tags, so `validate:"required"` works out of the box:

```go
type AppConfig struct {
	DBPath string `env:"DB_PATH" validate:"required"`
}
```

Fields tagged `safe:"true"` are redacted as `********` in the configuration update log.

## Sub-packages

Each is independently importable:

| Package | Purpose |
|---------|---------|
| [`config/environment`](https://pkg.go.dev/github.com/guionardo/go/config/environment) | `env`/`default` tag parsing, plus `GetEnv` |
| [`config/profile`](https://pkg.go.dev/github.com/guionardo/go/config/profile) | loads and deep-merges the default and scope YAML files; scope names may not escape the profiles directory |
| [`config/merger`](https://pkg.go.dev/github.com/guionardo/go/config/merger) | recursive deep-merge of `map[string]any` (later maps win; mismatched types are left unchanged) |
| [`config/validation`](https://pkg.go.dev/github.com/guionardo/go/config/validation) | struct validation via `go-playground/validator` and the `Validator` interface |

## Notes

Logging is done through `log/slog` with `module=config`. `WithLogger` stores the supplied
`Logger` but the provider does not yet route its messages through it, and `WithDebugLogger`
only sets a handler on its own internal logger plus the construction-time warning. To
change log output today, configure `slog.SetDefault`.

# environment

Fills a struct from the environment using `env` and `default` struct tags, plus a
one-off `GetEnv` lookup.

Part of [`config`](../README.md) in [`github.com/guionardo/go`](../../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/config/environment)

## Install

```bash
go get github.com/guionardo/go/config/environment
```

## Usage

```go
type AppConfig struct {
	Host  string `env:"APP_HOST" default:"localhost"`
	Port  int    `env:"APP_PORT" default:"8080"`
	Debug bool   `env:"APP_DEBUG"`
}

var cfg AppConfig
if err := environment.ParseEnvironment(&cfg, nil); err != nil {
	log.Fatal(err)
}

port := environment.GetEnv("APP_PORT", "8080") // "8080" when APP_PORT is unset
```

`ParseEnvironment` is what `config.Provider` calls after unmarshalling the YAML
profiles, which is why environment variables win over profile values. Both functions
are exercised in [`example_test.go`](example_test.go); `ParseEnvironment` logs through
`log/slog`, so redirect the default logger if you want silent output.

## Behaviour

**Precedence.** A non-empty environment variable wins over the `default` tag. An
environment variable that is set to the empty string counts as unset, so the `default`
tag applies — the field keeps its zero value only when it has neither. A `default` tag
works without an `env` tag.

**Matching is exact.** `GetEnv("app_port")` does not find `APP_PORT` on Unix. Windows
environment variables are case-insensitive by Windows' own semantics; that is not a
feature of this package. Unlike the tag path, `GetEnv` uses `os.LookupEnv`, so a
variable that is set but empty is returned as `""` and its own default is not
consulted.

**Supported kinds:** `string`, `int`/`int8`/`int16`/`int32`/`int64`,
`uint`/`uint8`/`uint16`/`uint32`/`uint64`, `float32`/`float64`, and `bool`. That is the
whole list — `time.Duration` is not special-cased, so a duration field is handled as
`int64` and only accepts a plain base-10 integer (nanoseconds); `default:"5s"` fails.
Slices, maps, and pointer fields are left untouched, with no error.

**Nested structs** are recursed into for every struct-typed field, tagged or not.

**Errors.** A bad value produces
`invalid field value '<raw>' (<kind>) for field <Name>: <cause>`, one error per failing
field joined into the returned error. Passing something other than a pointer to a
struct fails immediately; panics inside the reflection walk are recovered and returned
as errors. Partial success is normal: the fields that parsed are already set when an
error comes back.

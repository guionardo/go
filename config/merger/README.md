# merger

Recursive deep-merge of `map[string]any` trees, left to right.

Part of [`config`](../README.md) in [`github.com/guionardo/go`](../../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/config/merger)

## Install

```bash
go get github.com/guionardo/go/config/merger
```

## Usage

```go
base := map[string]any{
	"port":   8080,
	"server": map[string]any{"timeout": 30},
}
override := map[string]any{
	"port":   9090,
	"server": map[string]any{"tls": true},
}

merged := merger.MergeMaps(base, override)
// map[port:9090 server:map[tls:true timeout:30]]
```

See [`example_test.go`](example_test.go) for the runnable version. `config/profile`
uses it to layer the scope profile over the default profile.

## Behaviour

**A later value only wins on an exact type match.** `updateMapValues` compares
`reflect.TypeOf(currentValue)` with `reflect.TypeOf(v)` and `continue`s when they
differ, so the earlier value survives untouched. Nothing is logged and no error is
returned — the later value is simply dropped:

```go
merger.MergeMaps(
	map[string]any{"port": 8080},
	map[string]any{"port": "9090"}, // string vs int -> ignored, port stays 8080
)
```

This bites on YAML input, where `port: 8080` decodes as `int` and `port: 9090.5` as
`float64`.

**Only `map[string]any` is merged recursively.** A same-typed nested value that is
not exactly `map[string]any` — `map[string]string`, for example — cannot be merged,
so the later map's value **replaces** the earlier one as a whole. It does not panic:

```go
merger.MergeMaps(
	map[string]any{"labels": map[string]string{"env": "dev"}},
	map[string]any{"labels": map[string]string{"env": "prod"}},
	// labels is now map[string]string{"env": "prod"}
)
```

**Neither input is mutated.** Nested `map[string]any` values are deep-copied into the
accumulator, so after `MergeMaps(base, override)` both `base` and `override` are
unchanged and can be reused. Non-map values are shared by reference (inserted as-is),
which is safe because the merge never writes through them.

The function never returns an error and handles a `nil` map in the argument list.

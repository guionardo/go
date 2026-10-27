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

**Nested maps must be `map[string]any`.** Same-typed values of kind `Map` are recursed
by asserting `map[string]any`; two `map[string]string` values that match in type reach
that assertion and panic. Anything that is not `map[string]any` at every nested level
is unsupported.

**Nested maps are merged in place.** The accumulator for each key is the first map's
own nested `map[string]any`, so merging writes into it: after `MergeMaps(base, override)`
the `base` map can be mutated. Clone the inputs if you need them afterwards.

Keys absent from the accumulator are inserted as-is, so values are shared by reference
with the input, not copied. The function never returns an error and handles a `nil`
map in the argument list.

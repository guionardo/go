# reflecttools

Zero-value detection that understands `time.Time`, `time.Duration`, and empty
collections — the cases where a plain `v == zero` comparison does not compile or does
not mean what you expect. One function, no reflection boilerplate in your code.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/reflect_tools)

## Install

```bash
go get github.com/guionardo/go/reflect_tools
```

## Usage

```go
import reflecttools "github.com/guionardo/go/reflect_tools"

reflecttools.IsZeroValue(nil)              // true
reflecttools.IsZeroValue(0)                // true
reflecttools.IsZeroValue("")               // true
reflecttools.IsZeroValue(false)            // true
reflecttools.IsZeroValue(time.Time{})      // true
reflecttools.IsZeroValue(time.Duration(0)) // true
reflecttools.IsZeroValue([]int{})          // true (empty or nil slice)
reflecttools.IsZeroValue(map[string]int{}) // true (empty or nil map)

reflecttools.IsZeroValue(42)               // false
reflecttools.IsZeroValue(time.Now())       // false
```

## What counts as zero

| Input | Result |
|-------|--------|
| untyped `nil` | `true` |
| `int*`, `uint*`, `float32`, `float64`, `bool`, `string` | `true` when the built-in zero value |
| `time.Time` | `true` when `IsZero()` |
| `time.Duration` | `true` when `0` |
| array, slice, map | `true` when `len() == 0` — so a nil and an empty collection are both zero |
| anything else (structs, pointers, funcs, channels, complex numbers) | `false` |

## Limitations

The type switch matches only the exact built-in types. A named type with the same
underlying type — `type Status string` or `type Count int` — falls through to the
`default` branch and is reported as **not** zero, even when empty:

```go
type Status string
reflecttools.IsZeroValue(Status("")) // false
```

Two related gaps follow from that: `nil` pointers are **not** treated as zero
(`reflect.Kind()` for a pointer is not array/slice/map, so the function returns
`false`), and empty structs always return `false`. Convert named types to their
underlying type first if you need zero-detection on them.

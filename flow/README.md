# flow

Small generic control-flow helpers: a ternary, a zero-value fallback, insertion-ordered
map iteration, and slice lookups. Reach for it to replace one-line `if`/`else` blocks and
`len(s) > 0` guards with expressions that read cleanly.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/flow)

## Install

```bash
go get github.com/guionardo/go/flow
```

## Usage

### Ternary and fallback

```go
import "github.com/guionardo/go/flow"

max := flow.If(x > y, x, y)
name := flow.Default(input, "defaultName")

flow.If(true, "yes", "no")     // "yes"
flow.Default("", "fallback")   // "fallback"
flow.Default("hello", "fallback") // "hello"
```

```go
// If is a generic ternary operator.
func If[T any](condition bool, valueIfTrue T, valueIfFalse T) T

// Default returns valueIfZero when value equals the zero value for type T.
func Default[T comparable](value T, valueIfZero T) T
```

`Default` compares against the zero value, so `0`, `""`, `false`, and `nil` interfaces all
fall back — there is no way to keep a legitimate zero value.

### Ordered map

`OrderedMap` iterates in the order keys were last set.

```go
m := flow.NewOrderedMap[string, int]()
m.Set("a", 1)
m.Set("b", 2)
m.Set("a", 3) // re-setting moves the key to the end

if v, ok := m.Get("a"); ok {
	fmt.Println(v) // 3
}
fmt.Println(m.GetDefault("missing", 0)) // 0

m.Delete("b")

for k, v := range m.Range(0, -1) {
	fmt.Println(k, v) // a 3
}
```

```go
type OrderedMap[T comparable, S any] struct{}

func NewOrderedMap[T comparable, S any]() *OrderedMap[T, S]
func (om *OrderedMap[T, S]) Set(key T, value S)
func (om *OrderedMap[T, S]) Get(key T) (value S, ok bool)
func (om *OrderedMap[T, S]) GetDefault(key T, defaultValue S) S
func (om *OrderedMap[T, S]) Delete(key T)
func (om *OrderedMap[T, S]) Keys(first, last int) iter.Seq[T]
func (om *OrderedMap[T, S]) Range(first, last int) iter.Seq2[T, S]
func (om *OrderedMap[T, S]) Len() int
```

`Keys` and `Range` take a half-open range, `[first, last)`, over the insertion order:

- a negative `first` counts from the end (`-2` starts at the second-to-last key);
- a negative `last` counts back from the end, so `-1` means "through the last key";
- `first` is clamped to the last index and `last` to the length, and `last <= first`
  yields a single key.

`Len()` returns the number of entries.

### Slices

```go
first := flow.SliceFirstOrDefault(values, 0) // values[0], or 0 when empty
last := flow.SliceLastOrDefault(values, 0)   // final element, or 0 when empty
rest := flow.RemoveItem(values, 2)           // values without index 2
```

```go
func SliceFirstOrDefault[T any](s []T, defaultValue T) T
func SliceLastOrDefault[T any](s []T, defaultValue T) T
func RemoveItem[T any](s []T, index int) []T
```

`RemoveItem` returns `s` unchanged when `index` is out of range; otherwise it removes in
place, so the returned slice shares the backing array with `s`.

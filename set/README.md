# set

A generic `Set[T comparable]` backed directly by a Go map, with set algebra
(union, intersection, symmetric difference), `iter.Seq` iteration, JSON
serialization, and `database/sql` Scanner/Valuer support.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/set)

## Install

```bash
go get github.com/guionardo/go/set
```

## Usage

```go
import "github.com/guionardo/go/set"

s := set.New(1, 2, 3, 2, 1) // duplicates are dropped
s.Add(4)
s.AddMultiple(5, 6)         // returns s, so calls can chain
s.Remove(1)

s.Has(2)                    // true
s.HasAll(2, 3)              // true
s.Equals(set.New(2, 3, 4))  // false — exact same members required
s.ToArray()                 // []int{2, 3, 4, 5, 6}, unsorted
s.Clear()
```

## Operations

| Group | Methods |
|-------|---------|
| Construct | `New(values ...T) Set[T]` |
| Mutate | `Add(v)`, `AddMultiple(values ...T) Set[T]`, `Remove(v)`, `UpdateFrom(another) Set[T]`, `Clear()` |
| Inspect | `Has(v) bool`, `HasAll(values ...T) bool`, `Equals(another) bool`, `ToArray() []T` |
| Algebra | `Union(another) Set[T]`, `Intersection(another) Set[T]`, `Diff(another) Set[T]` |
| Iterate | `Iter() iter.Seq[T]`, `Filter(func(T) bool) iter.Seq[T]` |
| Serialize | `MarshalJSON()`, `UnmarshalJSON([]byte)` |
| SQL | `Scan(any) error`, `Value() (driver.Value, error)` |

### Algebra

```go
a := set.New(1, 2, 3)
b := set.New(3, 4, 5)

a.Union(b)        // {1, 2, 3, 4, 5} — new set
a.Intersection(b) // {3}              — new set
a.Diff(b)         // {1, 2, 4, 5}     — new set
a.UpdateFrom(b)   // a == {1, 2, 3, 4, 5} — mutates and returns a
```

`Diff` is the **symmetric** difference: members of either set that are not in both.
It is not `a - b`.

### Iteration and filtering

```go
for v := range s.Iter() {
    fmt.Println(v)
}

evens := set.New(1, 2, 3, 4, 5).Filter(func(v int) bool { return v%2 == 0 })
for v := range evens { // standard range-over-func
    fmt.Println(v)
}
```

`Filter` returns an `iter.Seq[T]`, not a `Set[T]`. Iteration order is undefined
because the backing map has no order.

### JSON

`MarshalJSON` emits a JSON array of the members; `UnmarshalJSON` accepts a JSON
array and replaces the current contents:

```go
b, _ := set.New("b", "a").MarshalJSON() // ["b","a"] — order not guaranteed
```

### database/sql

`Value` writes the members as JSON (`["a","b","c"]`), and `Scan` reads that JSON back
from either a `string` or a `[]byte` column — convenient for storing a set in a text
or JSON column.

```go
type Row struct {
    Tags set.Set[string]
}

// scanning
err := row.Tags.Scan(`["go","sql"]`)

// writing: driver.Valuer
_, err = db.Exec("INSERT INTO t (tags) VALUES ($1)", r.Tags)
```

## Notes

- `Set[T]` is a map type (`map[T]struct{}`), not a struct, so `len(s)` and
  `for v := range s` work directly. The zero value is a nil map, which is usable for
  reads but not for writes.
- `MarshalJSON` output is unordered; do not compare two JSON encodings of equal sets.
- `UnmarshalJSON` and `Scan` write through `AddMultiple`, so a **zero-value** `Set`
  panics with `assignment to entry in nil map`. Initialize before decoding:

  ```go
  var s set.Set[string]        // nil — do not decode into this
  s = set.New[string]()        // ok
  ```

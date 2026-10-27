# timetools

Parse timestamps without picking a layout first: `Parse` walks a list of common Go
layouts and returns the first match, then promotes whichever layout worked so that
frequently seen formats are tried first on later calls.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/time_tools)

## Install

```bash
go get github.com/guionardo/go/time_tools
```

## Usage

```go
import timetools "github.com/guionardo/go/time_tools"

t, err := timetools.Parse("2024-03-15T10:20:30Z")
// t = 2024-03-15 10:20:30 +0000 UTC, err = nil

t, err = timetools.Parse("2024-01-15")
// t = 2024-01-15 00:00:00 +0000 UTC

t, err = timetools.Parse("not-a-date")
// t.IsZero(), errors.Is(err, timetools.ErrTimeParser)
```

Use `SetLayouts` to replace the built-in list with your own formats:

```go
timetools.SetLayouts([]string{"2006/01/02", time.RFC3339})

t, err := timetools.Parse("2024/01/15") // 2024-01-15 00:00:00 +0000 UTC
```

## How layout selection works

The package keeps a single ordered layout list. Default order:

```text
DateTime, ANSIC, UnixDate, RubyDate, RFC822, RFC822Z, RFC850,
RFC1123, RFC1123Z, RFC3339, RFC3339Nano, Kitchen, Stamp,
StampMilli, StampMicro, StampNano, DateOnly, TimeOnly
```

`Parse` tries them in order and returns on the first success. On a success at index
`i > 0` the matched layout is moved to the **front** of the list (the others shift
right), so the next parse of the same format matches on the first attempt. On total
failure `Parse` returns the zero `time.Time` and `ErrTimeParser`.

## Notes

- The layout list is package-global. Promotion means the order is a shared, evolving
  priority queue, not a per-call setting — two goroutines parsing different formats
  will influence each other's order. The list is swapped atomically, so this is safe
  (`doc.go`: thread-safe) but not deterministic.
- `SetLayouts` **replaces** the list; it does not append. After calling it, only your
  layouts are tried — a string that used to parse as RFC3339 will fail unless you
  include `time.RFC3339` in the new list.
- `SetLayouts` stores the slice it receives without copying; do not mutate it
  afterwards.
- Layouts are matched with `time.Parse`, so inputs must include everything the layout
  requires. There is no date-guessing, locale handling, or fuzzy matching.

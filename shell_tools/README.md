# shelltools

Two small helpers for CLI code: parse a command line into arguments while
respecting single and double quotes, and read an environment variable without
juggling the `ok` return value.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/shell_tools)

## Install

```bash
go get github.com/guionardo/go/shell_tools
```

## Usage

### Quoted arguments

`QuotedShellArgs` is a `[]string` with a parser and a renderer:

```go
import shelltools "github.com/guionardo/go/shell_tools"

args := shelltools.NewQuotedShellArgs(`one "two three" 'four five'`)
// args[0] = "one", args[1] = "two three", args[2] = "four five"
```

Quotes are removed from the result, and an escaped space (`\ `) is kept as a literal
space inside its argument:

```go
args := shelltools.NewQuotedShellArgs(`backup a\ b.txt "c d.txt"`)
// {"backup", "a b.txt", "c d.txt"}
```

`String` renders the arguments back into one string, re-quoting members that contain
a space:

```go
cmd := shelltools.QuotedShellArgs{"git", "commit", "-m", "a message"}
cmd.String() // `git commit -m "a message"`
```

### Environment lookup

`GetEnv` returns the variable's value, or `""` when it is unset — no `(string, bool)`
unpacking at the call site:

```go
token := shelltools.GetEnv("API_TOKEN")
if token == "" {
    // unset or empty — the two are indistinguishable here
}
```

Lookups are exact on Linux and macOS. Windows environment blocks are themselves
case-insensitive, so there `GetEnv("path")` finds `PATH`; on Unix it does not.

## Behaviour details

| Rule | Detail |
|------|--------|
| Separators | Any unicode whitespace (`strings.Fields` semantics) |
| Quotes | `'…'` and `"…"`, stripped from the returned argument |
| Escaped space | `\ ` becomes a literal space inside the argument |
| Unterminated quote | The rest of the input becomes a single argument, no error |
| Return value | `NewQuotedShellArgs` never fails; it has no error return |

## Limitations

- `String` only re-quotes arguments that contain a space, and only when the slice has
  two or more elements. A single argument (`QuotedShellArgs{"a b"}.String()`) is
  returned raw, and shell metacharacters such as `$`, `*`, `;`, or `"` are not escaped.
  It is a display helper, not a shell-escaping function — do not build commands from
  untrusted input with it.
- Backslashes are not a general escape mechanism: only `\ ` is interpreted. Inside
  double quotes a `\"` therefore ends the quoted section rather than escaping the
  quote.
- Arguments are not glob-expanded, variable-expanded, or otherwise processed.

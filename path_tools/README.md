# path_tools

Small file-system helpers for the questions most CLIs ask: does this path exist,
create the directory tree, find an executable on `PATH`, walk up to the Go module
root.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/path_tools)

## Install

```bash
go get github.com/guionardo/go/path_tools
```

## Usage

The package is imported as `path_tools` and used as `pathtools`:

```go
import "github.com/guionardo/go/path_tools"

ok := pathtools.DirExists("/etc")         // true
ok = pathtools.FileExists("/etc/hosts")   // true; false for directories
err := pathtools.CreatePath("/tmp/a/b/c") // MkdirAll — nil when it already exists

root, err := pathtools.GetRootFolder("")      // nearest ancestor with go.mod, from the cwd
git, err := pathtools.FindFileInPath("git")   // first hit on PATH
atRoot := pathtools.IsRootDirectory("/")      // true on Unix; "C:\\" on Windows
```

## API

| Function | Behaviour |
|----------|-----------|
| `DirExists(pathName string) bool` | `os.Stat` plus `IsDir`. False on any error, including a missing path. |
| `FileExists(fileName string) bool` | `os.Stat` and *not* a directory. False on error, and false for directories. |
| `CreatePath(path string) error` | No-op when the path is already a directory; otherwise `os.MkdirAll` with mode `0750`. |
| `GetRootFolder(base string) (string, error)` | Walks up from `base` (`os.Getwd()` when empty) to the nearest directory containing `go.mod`. |
| `FindFileInPath(filename string) (string, error)` | Scans `PATH` in order for a regular file with that name and returns the first full path. |
| `IsRootDirectory(directory string) bool` | True when the cleaned path equals its base — `/` on Unix, a drive root such as `C:\` on Windows. |

Errors:

- `pathtools.ErrNotAGoProject` — `GetRootFolder` reached the filesystem root
  without finding a `go.mod`.
- `os.ErrNotExist` — the `GetRootFolder` base is not an existing directory.
- `FindFileInPath` returns distinct errors for a rejected filename (it contains
  `/`, `\`, or `..`), an unset or empty `PATH`, and a name that is not found.

## Notes

- `CreatePath` uses a fixed `0750` mode (subject to the process umask) on every
  OS. The doc comment says permissions are "updated from parent folder", but the
  implementation never reads the parent's mode.
- `FindFileInPath` matches any regular file: it does not check the executable
  bit, and it does not expand `PATHEXT`, so on Windows `FindFileInPath("git")`
  will not find `git.exe`.
- `DirExists` and `FileExists` use `os.Stat`, so symlinks are followed.
- `GetRootFolder` returns the folder that holds `go.mod`, not the module path; it
  stops at the filesystem root and never resolves symlinks along the way.

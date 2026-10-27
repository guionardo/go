# mid

Cross-platform machine identification: `MachineID` returns an identifier for the
host using OS-specific sources, so a CLI can tell machines apart without
carrying its own per-OS probe.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/mid)

## Install

```bash
go get github.com/guionardo/go/mid
```

## Usage

```go
import "github.com/guionardo/go/mid"

id := mid.MachineID()
if id == "" {
    // no source available on this host — common in containers and CI
}
```

There is a single function, `func MachineID() string`, and no error return: an
empty string means every source for the current OS failed.

## Supported platforms

`mid` builds on **Linux, macOS, and Windows only**. There is no generic
fallback file, so `MachineID` is undefined on any other `GOOS`.

- **macOS** — runs `system_profiler SPHardwareDataType SPSecureElementDataType`
  and returns `{model number}|{serial number}|{hardware uuid}`. No fallback.
- **Linux** — tries, in order, `hostnamectl status` (its `Machine ID:` line),
  then `/etc/machine-id`, then `/var/lib/dbus/machine-id`. The first non-empty
  result wins.
- **Windows** — runs `reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient`
  and returns the `MachineId` GUID **without** its braces.

## Notes

- **No error, no cache.** Each call shells out (`hostnamectl`, `system_profiler`,
  `reg`) or reads a file; results are never memoized. `system_profiler` is the
  expensive one, so avoid calling `MachineID` in a loop.
- **A non-empty result can still be useless.** If the command succeeds but the
  expected fields are absent, macOS returns the separator-only string `"||"`.
  Check for actual content, not just for emptiness.
- **The value is not anonymized.** macOS exposes the serial number and hardware
  UUID, and Windows returns the raw SQMClient GUID. Treat it as sensitive when
  logging or transmitting.
- **Testing global state.** On Linux the source order lives in a package-level
  `collectFuncs` slice guarded by `collectFuncsMu`. A test that replaces it must
  **not** use `t.Parallel()`: the mutex prevents a data race, not interference —
  a parallel sibling can observe the swapped list and fail nondeterministically.
  Restore the original slice in `t.Cleanup`.

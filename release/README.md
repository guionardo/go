# release

Self-update for Go CLI tools distributed through GitHub Releases: detect the running
version, compare it with the latest release, download and SHA256-verify the matching
asset, then hand off to an embedded swapper binary that replaces the executable
atomically and relaunches it with the original arguments.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/release)

## Install

```bash
go get github.com/guionardo/go/release
```

## Usage

Call `PerformSelfUpdate` early in `main`:

```go
import "github.com/guionardo/go/release"

func main() {
    result := release.PerformSelfUpdate(context.Background())

    if result.Err != nil {
        fmt.Fprintf(os.Stderr, "update failed: %v\n", result.Err)
    } else if result.Updated {
        fmt.Println("Updated to", result.Release.TagName)
        os.Exit(0) // the swapper is running; stop this instance
    } else if result.Current {
        // already on the latest release
    }

    // normal program execution
}
```

`PerformSelfUpdate` returns; it does not exit the process. Exit on your own once
`Updated` is true — the swapper is already waiting to replace the binary and relaunch
it, and leaving the old process alive means two instances running.

By default the owner and repository are derived from the Go module path of the binary
(`debug.ReadBuildInfo().Main.Path`), which must start with `github.com`. Override or
supply a token with functional options:

```go
result := release.PerformSelfUpdate(
    context.Background(),
    release.WithOwner("myorg"),
    release.WithRepo("mycli"),
    release.WithGitHubToken(os.Getenv("GITHUB_TOKEN")),
)
```

| Option | Effect |
|--------|--------|
| `WithOwner(string)` | GitHub owner, instead of the one parsed from the module path |
| `WithRepo(string)` | GitHub repository, instead of the one parsed from the module path |
| `WithGitHubToken(string)` | Sent as `Authorization: Bearer …` — required for private repos, raises the rate limit for public ones |

### UpdateResult

| Field | Type | Meaning |
|-------|------|---------|
| `Release` | `*Release` | Latest release fetched from the API (nil if the check failed) |
| `Updated` | `bool` | The swapper was spawned; the new binary is being installed |
| `Current` | `bool` | Already on the latest version |
| `State` | `UpdateState` | Progress: `UpdateStateUnknown` → `Checked` → `Downloaded` → `SwapperSpawned` |
| `Err` | `error` | First failure; `ErrUpdateInProgress` when another update holds the lock |

`UpdateResult` also implements `String()`, so `fmt.Println(result)` is a one-line log.

## How the self-update works

1. **Read the current version** from `debug.ReadBuildInfo()` (`GetCurrentVersion`).
2. **Check GitHub** — `GET /repos/{owner}/{repo}/releases/latest`, compared with
   hashicorp/go-version (`CheckForUpdate`). If not newer, return `Current: true`.
3. **Acquire `.update.lock`** next to the executable (`O_CREATE|O_EXCL`). A second
   concurrent update gets `ErrUpdateInProgress`; the lock is removed on return.
4. **Download the asset** whose name contains `runtime.GOOS` and `runtime.GOARCH`
   into the executable's directory, verifying the digest before writing
   (`DownloadUpdate` → `Asset.Download`). A failed download deletes the partial file.
5. **Extract the swapper** for this platform out of the embedded filesystem
   (`ExtractSwapper`, mode `0755`).
6. **Spawn the swapper** with `--new-binary`, `--checksum` (raw SHA256 hex),
   `--target`, followed by your original CLI arguments.
7. **Swapper: pre-swap verification** — re-hashes the new binary.
8. **Swapper: atomic replace** — rename the running binary to `<exe>.bak`, rename the
   new binary into place, re-verify the checksum, drop the backup. Any failure
   restores the backup.
9. **Swapper: relaunch** — `syscall.Exec` on Unix (process replacement, same PID),
   `os.StartProcess` then `exit(0)` on Windows, with the original arguments.

### Why a separate swapper binary

A running executable cannot reliably replace itself, especially on Windows. The
parent process starts the swapper and gets out of the way; the swapper is a small
standalone program that only knows how to do the rename dance and relaunch.

The swapper binaries live in `release/swapper/` and are **generated**, not committed —
`release/swapper/swapper_*` is in `.gitignore`. They are embedded with `//go:embed`
in `embed_swapper.go`, so a clean checkout will not compile `./...` until you build
them:

```bash
make swapper        # cross-compiles all four variants into release/swapper/
make swapper-clean  # removes them
```

| Embedded swapper | Platform |
|------------------|----------|
| `swapper_linux_amd64` | Linux amd64 |
| `swapper_darwin_amd64` | macOS amd64 |
| `swapper_darwin_arm64` | macOS arm64 |
| `swapper_windows_amd64.exe` | Windows amd64 |

These four are the supported targets. `ExtractSwapper` reads the file for the current
`GOOS`/`GOARCH`, so any other platform fails at runtime with
`read embedded swapper swapper_<os>_<arch>`.

## Release assets

`findAsset` walks the release assets in order and returns the first whose lowercased
name contains both the GOOS and GOARCH string. Recommended naming:

| Platform | Asset name |
|----------|-----------|
| Linux amd64 | `myapp_linux_amd64` |
| macOS amd64 | `myapp_darwin_amd64` |
| macOS arm64 | `myapp_darwin_arm64` |
| Windows amd64 | `myapp_windows_amd64.exe` |

The asset must be the **executable itself**. It is downloaded verbatim and renamed
over the running binary — archives (`*.tar.gz`, `*.zip`) are never unpacked, so
publishing one would install a gzip stream as your binary.

### Digest formats

Two different representations are involved:

```
Asset.Digest (from the GitHub API)   sha256:e3b0c442…b855   ← go-digest, with prefix
swapper --checksum (computed here)   e3b0c442…b855         ← raw hex, no prefix
```

`Asset.Download` recomputes `digest.FromBytes(content)` and compares it to
`Asset.Digest`; a missing or mismatched digest aborts the download. The raw hex is
computed by `PerformSelfUpdate` from the downloaded file and checked by the swapper
before and after the rename (case-insensitively, `EqualFold`).

## Limitations and gotchas

- **Version detection needs module build info.** `GetCurrentVersion` uses
  `buildInfo.Main.Version` and errors when it is empty or `(devel)`. A plain
  `go build` sets `(devel)`, so `PerformSelfUpdate` cannot work for a locally built
  binary. Version stamping must come from the module itself (for example
  `go install example.com/cmd@v1.2.3`) — `-ldflags "-X main.version=…"` sets a
  variable of yours, not `Main.Version`. If you stamp your own version, skip
  `PerformSelfUpdate` and call
  `release.CheckForUpdate(ctx, yourVersion, opts...)` + `release.DownloadUpdate`.
- **GitHub only, `github.com` only.** The module path must start with `github.com`
  when owner/repo are not given; hosted or vanity paths fail with
  `just github.com repositories are accepted by now`.
- **HTTPS only, no proxy knobs.** Requests use the package's own `http.Client`
  (30 s timeout) and are not configurable.
- **The lock is a file, not a mutex.** It lives beside the executable, so the
  executable's directory must be writable — the same requirement as downloading the
  update and extracting the swapper.
- **The swapper inherits stdin/stdout/stderr and the environment**, and passes your
  CLI arguments through to the relaunched binary.
- **`GetThisLatestRelease` and `GetLatestRelease`** are exported helpers for the same
  API endpoint, independent of the update flow.

## Security notes

- Redirects on the GitHub API client are restricted to `api.github.com` and
  `github.com`; anything else is rejected.
- The download is verified against the API-provided digest before it is written and
  re-verified by the swapper before and after the rename.
- The swapper rejects `--new-binary` values containing `..` or a null byte, resolves
  symlinks on the target, and restores the `.bak` copy if the post-swap check fails.
- The token is only placed in the `Authorization` header and is never logged.

## CI integration

Each target builds its binary and publishes it under the naming convention above.
Check that the API reports a `digest` for each asset — `Asset.Download` refuses a
binary that has none. The `*.sha256` sidecar files below are for humans and audits;
the package never reads them.

### Go

```yaml
- name: Build release assets
  run: |
    mkdir -p dist
    for target in linux/amd64 darwin/amd64 darwin/arm64 windows/amd64; do
      GOOS=${target%/*}; GOARCH=${target#*/}
      ext=""; [ "$GOOS" = "windows" ] && ext=".exe"
      GOOS=$GOOS GOARCH=$GOARCH go build -o "dist/myapp_${GOOS}_${GOARCH}${ext}" .
    done
    cd dist && for f in myapp_*; do sha256sum "$f" | awk '{print $1}' > "$f.sha256"; done

- uses: softprops/action-gh-release@v2
  with:
    files: dist/*
```

### Python (PyInstaller)

```yaml
- name: Build
  run: |
    pip install pyinstaller -r requirements.txt
    pyinstaller --onefile --name myapp myapp/__main__.py
    ext=""; [ "${{ matrix.os }}" = "windows" ] && ext=".exe"
    mkdir -p dist
    cp "dist/myapp${ext}" "dist/myapp_${{ matrix.os }}_${{ matrix.arch }}${ext}"

- uses: softprops/action-gh-release@v2
  with:
    files: dist/*
```

### .NET (single-file publish)

```yaml
- name: Build
  run: |
    ext=""; [ "${{ matrix.os }}" = "windows" ] && ext=".exe"
    asset="myapp_${{ matrix.os }}_${{ matrix.arch }}${ext}"
    dotnet publish src/MyApp --runtime ${{ matrix.rid }} --configuration Release \
      --self-contained true -p:PublishSingleFile=true -p:Version=${{ github.ref_name }} -o dist
    mv "dist/myapp${ext}" "dist/$asset"

- uses: softprops/action-gh-release@v2
  with:
    files: dist/*
```

## See also

- [`cmd/example-updater/main.go`](../cmd/example-updater/main.go) — minimal CLI using
  `PerformSelfUpdate`
- [`Makefile`](../Makefile) — the `swapper` / `swapper-clean` targets
- [`release/swapper/main.go`](swapper/main.go) — the swap process itself

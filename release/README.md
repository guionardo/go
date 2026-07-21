# release — Self-Update for Go CLI Tools

Package `release` provides a complete self-update mechanism for Go CLI tools distributed via GitHub Releases. It handles version detection, update checking, secure download with SHA256 verification, atomic binary replacement, and automatic relaunch.

## Quick Start

Add the import:

```go
import "github.com/guionardo/go/release"
```

Call self-update in your CLI's main function:

```go
func main() {
    result := release.PerformSelfUpdate(context.Background())
    if result.Updated {
        fmt.Println("Updated to", result.Release.TagName)
        os.Exit(0) // swapper already started; old process exits
    }
    if result.Err != nil {
        fmt.Fprintf(os.Stderr, "update failed: %v\n", result.Err)
        os.Exit(1)
    }
    // normal program execution...
}
```

### With custom repository

```go
result := release.PerformSelfUpdate(
    context.Background(),
    release.WithOwner("myorg"),
    release.WithRepo("mycli"),
)
```

### With GitHub token (for private repos / higher rate limits)

```go
result := release.PerformSelfUpdate(
    context.Background(),
    release.WithGitHubToken(os.Getenv("GITHUB_TOKEN")),
)
```

## API Reference

### Types

| Type | Description |
|------|-------------|
| `Release` | GitHub release metadata (tag, assets, author, timestamps) |
| `Asset` | A downloadable binary asset with name, URL, size, and digest |
| `User` | GitHub user info (login, ID) |
| `Config` | Holder for functional options (Owner, Repo, Token) |
| `Option` | Interface for functional options |
| `UpdateState` | Enum tracking update progress |
| `UpdateResult` | Result of PerformSelfUpdate |

### UpdateState values

| Value | Meaning |
|-------|---------|
| `UpdateStateUnknown` | Initial state before any check |
| `UpdateStateChecked` | Version check completed |
| `UpdateStateDownloaded` | New binary downloaded successfully |
| `UpdateStateSwapperSpawned` | Swapper process has been launched |

### UpdateResult fields

| Field | Type | Description |
|-------|------|-------------|
| `Release` | `*Release` | The latest release (nil if checking failed) |
| `Updated` | `bool` | True if swapper was spawned (new binary being swapped) |
| `Current` | `bool` | True if already on the latest version |
| `State` | `UpdateState` | Current update progress state |
| `Err` | `error` | Error encountered during update, if any |

### Functions

#### ParseVersion

```go
func ParseVersion(s string) (*version.Version, error)
```

Parses a version string using hashicorp/go-version. Supports semver, prereleases (v1.2.3-rc1), and build metadata (v1.2.3+build.1).

#### GetCurrentVersion

```go
func GetCurrentVersion() (*version.Version, error)
```

Reads the current binary's version from `debug.ReadBuildInfo()`. The binary must be built with `-ldflags="-X main.version=vX.Y.Z"` or tagged by the Go toolchain.

#### GetLatestRelease

```go
func GetLatestRelease(owner, repo string) (*Release, error)
```

Fetches the latest release from `https://api.github.com/repos/{owner}/{repo}/releases/latest`.

#### GetThisLatestRelease

```go
func GetThisLatestRelease() (*Release, error)
```

Auto-detects the GitHub owner and repo from the Go module path (`debug.ReadBuildInfo().Main.Path`) and calls `GetLatestRelease`.

#### CheckForUpdate

```go
func CheckForUpdate(ctx context.Context, currentVersion string, opts ...Option) (*Release, bool, error)
```

Compares the current version against the latest GitHub release. Returns the release and `true` if an update is available. Supports `WithOwner`, `WithRepo`, and `WithGitHubToken` options.

#### DownloadUpdate

```go
func DownloadUpdate(ctx context.Context, rel *Release, targetDir string) (string, error)
```

Finds the platform-specific asset (matching runtime.GOOS and runtime.GOARCH by name), downloads it, verifies its SHA256 digest, and writes it to `targetDir`. Returns the path to the downloaded binary.

#### PerformSelfUpdate

```go
func PerformSelfUpdate(ctx context.Context, opts ...Option) *UpdateResult
```

Full update orchestrator: checks version, downloads update, extracts swapper, spawns swap process. Returns immediately if already current. On success, the swapper is running in the background.

#### ExtractSwapper

```go
func ExtractSwapper(targetDir string) (string, error)
```

Extracts the embedded swapper binary for the current platform from the embedded filesystem to `targetDir`.

### Asset.Download

```go
func (asset *Asset) Download(w io.Writer) error
```

Downloads the asset from `BrowserDownloadURL`, verifies the SHA256 digest using go-digest, and writes the content to `w`.

## Architecture

### Self-Update Flow

```
┌──────────────┐
│  Your CLI    │
│  main.go     │
└──────┬───────┘
       │ PerformSelfUpdate(ctx)
       ▼
┌──────────────────┐
│ 1. Read version  │ ← debug.ReadBuildInfo()
│    from binary   │
└──────┬───────────┘
       ▼
┌──────────────────┐
│ 2. Check for     │ ← GitHub API: /releases/latest
│    update         │
└──────┬───────────┘
       │
       ├── No update → return {Current: true}
       │
       ▼ Update available
┌──────────────────┐
│ 3. Acquire lock  │ ← .update.lock (prevents concurrent updates)
└──────┬───────────┘
       ▼
┌──────────────────┐
│ 4. Download new  │ ← Match asset by GOOS/GOARCH
│    binary        │ ← Verify SHA256 (go-digest)
└──────┬───────────┘
       ▼
┌──────────────────┐
│ 5. Extract       │ ← Write embedded swapper binary
│    swapper       │    to exe directory
└──────┬───────────┘
       ▼
┌──────────────────┐
│ 6. Spawn swapper │ ← os.StartProcess with flags:
│                  │    --new-binary  (downloaded)
│                  │    --checksum     (SHA256 hex)
│                  │    --target       (current exe)
│                  │    [original args...]
└──────┬───────────┘
       │ parent exits (proc.Release)
       ▼
┌─────────────────────────┐
│ 7. Swapper: verify      │ ← SHA256 of new binary
│    checksum (pre-swap)  │
└──────┬──────────────────┘
       ▼
┌─────────────────────────┐
│ 8. Swapper: atomic swap │
│    - backup exe → .bak  │
│    - rename new → exe   │
│    - verify (post-swap) │
│    - remove backup      │
└──────┬──────────────────┘
       │
       ├── Failure → restore from .bak, exit(1)
       ▼ Success
┌─────────────────────────┐
│ 9. Swapper: relaunch    │
│    - Unix: syscall.Exec │← replaces process image
│    - Win: StartProcess  │← new process, then exit
│    - Passes original    │
│      CLI args           │
└─────────────────────────┘
```

### Two-Phase Digest Verification

The update process verifies the binary at two separate points:

1. **Download phase** (in `Asset.Download`): Verifies using `go-digest` (canonical `sha256:hex` format) immediately after download.
2. **Swap phase** (in swapper): Re-verifies using raw SHA256 hex before and after the file rename, protecting against filesystem-level corruption or tampering.

### Lock Mechanism

`PerformSelfUpdate` creates a lock file (`.update.lock`) next to the running executable. This prevents concurrent update attempts. The lock is released via deferred cleanup regardless of success or failure.

## Asset Naming Convention

The `findAsset` function matches assets by checking whether the asset name contains the GOOS and GOARCH strings (case-insensitive).

**Recommended naming pattern:**

| Platform | Asset name |
|----------|-----------|
| Linux amd64 | `myapp_linux_amd64` or `myapp_linux_amd64.tar.gz` |
| macOS amd64 | `myapp_darwin_amd64` or `myapp_darwin_amd64.tar.gz` |
| macOS arm64 | `myapp_darwin_arm64` or `myapp_darwin_arm64.tar.gz` |
| Windows amd64 | `myapp_windows_amd64.exe` or `myapp_windows_amd64.zip` |

Naming rules:
- The GOOS string must appear as a contiguous substring in lowercase.
- The GOARCH string must appear as a contiguous substring in lowercase.
- Underscores, hyphens, and dots are allowed.
- On Windows, `.exe` suffix is recommended but not required for matching (it's handled by the code that strips the extension).

## Digest Format

The `Asset.Digest` field uses the [go-digest](https://github.com/opencontainers/go-digest) format:

```
sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

The swapper's `--checksum` flag uses raw lowercase SHA256 hex (64 characters, no `sha256:` prefix):

```
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

Your release workflow must compute both formats:
- **go-digest format** for the Asset.Digest metadata
- **raw hex** for the swapper's embedded checksum (computed automatically by `PerformSelfUpdate`)

## Cross-Platform Support

| Platform | Build | Swap | Relaunch |
|----------|-------|------|----------|
| Linux amd64 | ✓ | `os.Rename` | `syscall.Exec` |
| macOS amd64 | ✓ | `os.Rename` | `syscall.Exec` |
| macOS arm64 | ✓ | `os.Rename` | `syscall.Exec` |
| Windows amd64 | ✓ | `os.Rename` | `os.StartProcess` |

The swapper binary is embedded for each target platform at build time using `//go:embed`. The Makefile provides the `swapper` target that cross-compiles all four variants:

```sh
make swapper
```

## Workflow Integration

Your CI/CD pipeline must:
1. Build binaries for each target platform.
2. Compute SHA256 digests.
3. Upload binaries as GitHub Release assets with a `release-manifest.json` containing digest metadata.

### Workflow Examples

#### Go

```yaml
name: Release (Go)

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Build release assets
        run: |
          mkdir -p dist
          for target in linux/amd64 darwin/amd64 darwin/arm64 windows/amd64; do
            GOOS=${target%/*}
            GOARCH=${target#*/}
            ext=""; [ "$GOOS" = "windows" ] && ext=".exe"
            asset="myapp_${GOOS}_${GOARCH}${ext}"
            GOOS=$GOOS GOARCH=$GOARCH \
              go build -ldflags="-X main.version=${{ github.ref_name }}" \
              -o "dist/$asset" .
          done

      - name: Compute digests and manifest
        run: |
          cd dist
          manifest="{}"
          for f in myapp_*; do
            hex=$(sha256sum "$f" | awk '{print $1}')
            echo "$hex" > "${f}.sha256"
            echo "sha256:$hex" > "${f}.digest"
          done

      - name: Upload release assets
        uses: softprops/action-gh-release@v2
        with:
          files: dist/*
```

#### Python

```yaml
name: Release (Python)

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - os: linux     arch: amd64  pyi: x86_64
          - os: darwin    arch: amd64  pyi: x86_64
          - os: darwin    arch: arm64  pyi: arm64
          - os: windows   arch: amd64  pyi: amd64

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'

      - name: Build with PyInstaller
        run: |
          pip install pyinstaller -r requirements.txt
          pyinstaller --onefile --name myapp myapp/__main__.py
          ext=""; [ "${{ matrix.os }}" = "windows" ] && ext=".exe"
          mkdir -p dist
          cp "dist/myapp${ext}" \
            "dist/myapp_${{ matrix.os }}_${{ matrix.arch }}${ext}"

      - name: Compute digests
        run: |
          cd dist
          for f in myapp_*; do
            hex=$([ "${{ matrix.os }}" = "windows" ] \
              && certutil -hashfile "$f" SHA256 \
              | findstr /v "hash" | tr -d " " \
              || sha256sum "$f" | awk '{print $1}')
            echo "$hex" > "${f}.sha256"
            echo "sha256:$hex" > "${f}.digest"
          done

      - name: Upload release assets
        uses: softprops/action-gh-release@v2
        with:
          files: dist/*
```

#### .NET

```yaml
name: Release (.NET)

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - rid: linux-x64   os: linux    arch: amd64
          - rid: osx-x64     os: darwin   arch: amd64
          - rid: osx-arm64   os: darwin   arch: arm64
          - rid: win-x64     os: windows  arch: amd64

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-dotnet@v4
        with:
          dotnet-version: '8.0'

      - name: Build binary
        run: |
          ext=""; [ "${{ matrix.os }}" = "windows" ] && ext=".exe"
          asset="myapp_${{ matrix.os }}_${{ matrix.arch }}${ext}"
          dotnet publish src/MyApp \
            --runtime ${{ matrix.rid }} \
            --configuration Release \
            --self-contained true \
            -p:DebugType=None -p:PublishSingleFile=true \
            -p:Version=${{ github.ref_name }} \
            -o "dist"
          mv "dist/myapp${ext}" "dist/$asset"

      - name: Compute digests
        run: |
          cd dist
          for f in myapp_*; do
            hex=$(sha256sum "$f" | awk '{print $1}')
            echo "$hex" > "${f}.sha256"
            echo "sha256:$hex" > "${f}.digest"
          done

      - name: Upload release assets
        uses: softprops/action-gh-release@v2
        with:
          files: dist/*
```

All three examples produce assets following the naming convention required by `findAsset` (`{name}_{goos}_{goarch}`) and include digest metadata for `Asset.Download` verification.

## Security

- **Redirect validation**: The GitHub HTTP client rejects redirects to untrusted hosts (only `api.github.com` and `github.com` are allowed).
- **SSRF mitigation**: `CheckRedirect` in the GitHub client prevents server-side request forgery.
- **Two-phase verification**: The binary digest is verified at download time (go-digest) and at swap time (stdlib SHA256).
- **Path traversal protection**: The swapper rejects `--new-binary` paths containing `..` or null bytes.
- **Atomic replacement**: The old binary is backed up before the new one is moved into place; any failure triggers a restore.
- **Lock file**: Prevents concurrent update processes from interfering with each other.
- **Token authentication**: GitHub tokens are passed via the `Authorization` header (Bearer scheme); never logged.

## See also

- [cmd/example-updater/main.go](/cmd/example-updater/main.go) — Minimal CLI demonstrating `PerformSelfUpdate`
- [Makefile](/Makefile) — `swapper` target for cross-compiling the embedded binary
- [swapper/main.go](/release/swapper/main.go) — The atomic swap process implementation

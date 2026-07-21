# Plan 04-03 Summary: Integration + CLI

## Objective
Create the integration layer (`PerformSelfUpdate`) that ties together version checking, binary download, swapper extraction, and atomic swap via a spawned child process. Also create the example CLI and build infrastructure for swapper platform binaries.

## Deliverables

### 1. `release/embed_swapper.go`
- Embeds pre-compiled swapper binaries for 4 platforms (linux/amd64, darwin/amd64, darwin/arm64, windows/amd64) via `//go:embed`
- `ExtractSwapper(targetDir)` — extracts the correct binary for `runtime.GOOS`/`runtime.GOARCH` to a target directory

### 2. `release/self_update.go`
- `UpdateState` enum: `Unknown → Checked → Downloaded → SwapperSpawned`
- `UpdateResult` struct with `Release`, `Updated`, `Current`, `State`, `Err` fields + `String()` method
- `updateLockPath()` — returns path to `.update.lock` next to current executable
- `computeFileSHA256(path)` — SHA-256 hex digest of a file
- `ErrUpdateInProgress` sentinel error
- `PerformSelfUpdate(ctx, opts...)` — full orchestrator:
  1. Resolve current version (with `testCurrentVersion` override for testing)
  2. `CheckForUpdate` via GitHub API
  3. Acquire exclusive update lock (atomic `O_EXCL|O_CREATE`)
  4. `DownloadUpdate` to executable directory
  5. `computeFileSHA256` of downloaded binary
  6. `ExtractSwapper` to executable directory
  7. `os.StartProcess` swapper with `--new-binary`, `--checksum`, and original args
  8. `proc.Release()`
- Refactored into helper functions: `resolveCurrentVersion()`, `acquireUpdateLock()`, `downloadAndSwap()` for cyclomatic complexity compliance

### 3. `cmd/example-updater/main.go`
- Demonstration CLI calling `release.PerformSelfUpdate(context.Background())`
- Prints result, exits 0 on update, 1 on error

### 4. Makefile targets
- `swapper` — builds all 4 platform binaries
- `swapper-linux`, `swapper-darwin`, `swapper-windows` — individual builds
- `swapper-clean` — removes built binaries

### 5. `.gitignore` update
- Added `release/swapper/swapper_*` and `release/swapper/*.exe`

### 6. Tests (`release/self_update_test.go`, 8 tests)
- `TestComputeFileSHA256` — verifies SHA-256 computation and error handling
- `TestUpdateLockPath` — verifies path format and absolute path
- `TestExtractSwapper` — verifies extraction to temp dir (skips if swapper not built)
- `TestUpdateResultString` — verifies all formatting states
- `TestPerformSelfUpdate_NoVersion` — error when no build info available
- `TestPerformSelfUpdate_Current` — no update needed (GitHub mock)
- `TestPerformSelfUpdate_LockExists` — `ErrUpdateInProgress` when lock file present
- `TestPerformSelfUpdate_APIError` — error propagation from API call

## Verification
- `go test ./release/ -count=1 -race` — 54 tests passed
- `golangci-lint run ./release/...` — no issues
- `go build ./cmd/example-updater/` — compiles successfully
- All swapper platform binaries built via `make swapper`

## Key Decisions
- `testCurrentVersion` package variable used to bypass `GetCurrentVersion()` in tests (same pattern as `githubAPIBase` override)
- `lockRef` struct + `acquireUpdateLock()` extracted to reduce cyclomatic complexity of `PerformSelfUpdate`
- `downloadAndSwap()` extracted as separate function for the download→verify→extract→spawn pipeline
- `nolint` directives used on `gosec` issues where the flagged variable paths are internal (updateLockPath, ExtractSwapper) and not user-controlled

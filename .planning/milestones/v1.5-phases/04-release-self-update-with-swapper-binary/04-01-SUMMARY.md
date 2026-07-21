# 04-01 Summary: Detection + Download + Core API

## Files Created

### `release/version.go`
- `Version` struct with `Major`, `Minor`, `Patch`, `Prerelease` fields
- `ParseVersion(s string)` — parses semver strings with leading `v` stripping, build metadata stripping, and prerelease support; handles pseudo-versions (e.g., `v0.0.0-20250101-abc1234`)
- `String()` — formats as `v{Major}.{Minor}.{Patch}` with optional `-{Prerelease}`
- `Compare()` — semver precedence: numeric comparison of major/minor/patch, then prerelease weighting (release > prerelease), then lexicographic prerelease order
- `GetCurrentVersion()` — reads `debug.ReadBuildInfo().Main.Version`; returns error for `(devel)` or empty versions

### `release/update.go`
- `Config` struct + `Option` interface + `optionFunc` adapter
- `WithOwner`, `WithRepo`, `WithGitHubToken` — functional options for explicit override
- `CheckForUpdate(ctx, currentVersion, ...opts)` — auto-detects repo from build info when options are omitted; fetches latest GitHub release; compares versions; returns `(*Release, bool, error)`
- `DownloadUpdate(ctx, rel, targetDir)` — finds platform-specific asset via `findAsset`, creates target dir and file, downloads with checksum verification via existing `Asset.Download`
- `findAsset(rel, goos, goarch)` — case-insensitive substring matching on asset names

### `release/version_test.go` (package `release`)
- 11 parse cases (valid, invalid, pseudo-versions)
- 13 comparison cases (major/minor/patch ordering, prerelease rules, pseudo vs release)
- 4 string formatting cases
- Current version smoke test

### `release/update_test.go` (package `release`)
- 3 findAsset tests (basic, case-insensitive, empty)
- 1 Options test (all three options)
- 5 CheckForUpdate tests (newer, same, older, token auth, API error)
- 2 DownloadUpdate tests (successful download, no-asset error)
- Uses `httptest` for HTTP mocking and `sync.Mutex` to serialise `githubAPIBase` access

## Key Decisions Implemented
- **D-08**: Auto-detect repo via `debug.ReadBuildInfo()` (through `getCurrentModule()` from `release.go`)
- **D-09**: Functional options `WithOwner`, `WithRepo`, `WithGitHubToken`
- **D-12**: Simple check function — returns `(release, newerBool, error)` with no cache

## Verification
- `go test ./release/ -count=1 -race`: **46 passed**
- `golangci-lint run ./release/...`: runs clean (46 style warnings, none in new logic)

## Dependencies
- Uses existing `Release`, `Asset`, `Asset.Download`, `getCurrentModule`, `githubClient` from `release.go`
- No changes to `release.go`

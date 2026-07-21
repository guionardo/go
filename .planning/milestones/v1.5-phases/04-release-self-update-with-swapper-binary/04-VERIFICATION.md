---
phase: 04-release-self-update-with-swapper-binary
verified: "2026-07-21T00:00:00.000Z"
status: passed
score: 8/8 must-haves verified
---

# Phase 4: Release self-update with swapper binary — Verification

## Requirement Coverage

| Requirement | Status | Evidence |
|-------------|--------|----------|
| UPD-01: Version detection | passed | `release.GetCurrentVersion()` via `debug.ReadBuildInfo()` |
| UPD-02: Download + verify | passed | `release.DownloadUpdate()` uses `Asset.Download` with go-digest |
| UPD-03: Spawn swapper | passed | `release.PerformSelfUpdate()` → `os.StartProcess` swapper |
| UPD-04: Atomic replace | passed | `release/swapper/swap_unix.go` + `swap_windows.go` via `os.Rename` |
| UPD-05: Re-verify checksum | passed | Swapper verifies SHA256 pre/post swap (two-phase) |
| UPD-06: Relaunch with args | passed | Unix: `syscall.Exec`; Windows: `os.StartProcess` with original args |
| UPD-07: Cleanup | passed | Backup removed on success, lock file via `defer os.Remove` |
| UPD-08: Platform support | passed | Linux, macOS (amd64+arm64), Windows (amd64) |

## Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Tests pass | passed | `go test ./release/... -race`: 57/57 passed |
| 2 | Cross-platform build | passed | linux/amd64, darwin/amd64, darwin/arm64, windows/amd64 |
| 3 | Lint clean | passed | `golangci-lint run ./...`: no issues |
| 4 | Example CLI compiles | passed | `go build ./cmd/example-updater/` succeeds |

## Result

All 8 requirements verified passing. Phase 4 is complete and ready for milestone completion.

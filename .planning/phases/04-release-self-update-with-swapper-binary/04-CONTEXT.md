# Phase 4: Release self-update with swapper binary - Context

**Gathered:** 2026-07-21
**Status:** Ready for planning

<domain>
## Phase Boundary

A self-update mechanism for Go programs published via GitHub releases. Detects the importing program's GitHub repository, checks for newer releases, downloads and verifies release artifacts, and replaces the running binary via a separately built swapper binary embedded with `//go:embed`.

The swapper handles binary swap with backup/rollback, re-verifies checksums before exec, and forwards original program arguments to the new version.

</domain>

<decisions>
## Implementation Decisions

### Trust & Verification Chain
- **D-01:** Two-phase verification — download checksum verified first, then swapper re-verifies checksum before calling the new binary
- **D-02:** Never runs unverified code — if either verification fails, abort and restore backup

### Swapper Architecture
- **D-03:** Swapper binary embedded in main binary via `//go:embed` — avoids AV false positives from downloading executables
- **D-04:** Swapper receives `--new-binary=<path>` + original `os.Args`
- **D-05:** Flow: main spawns swapper → main exits → swapper waits → backup old binary → copy new → swapper re-verifies checksum → exec new binary with original args

### Safety & Rollback
- **D-06:** Backup old binary before swap, restore on any failure (checksum mismatch, copy error, new binary exits with non-zero)
- **D-07:** Swapper cleans up temp files and backup after successful relaunch

### Resolution Modes
- **D-08:** Auto-detect repository via `debug.ReadBuildInfo()` — works for any Go program built with module info
- **D-09:** Explicit override — caller can provide owner/repo directly

### Milestone
- **D-10:** v1.5 milestone — named "Self-Update", standalone deliverable
- **D-11:** Strings Package and Retry Package removed from roadmap

### Update Detection
- **D-12:** Library exposes a version-check function — calling program decides when/how to invoke it (CLI command, startup check, etc.)

### Swapper Notification
- **D-13:** Swapper reports result via stdout and exit code — calling program's wrapper script or process manager interprets them

### Platform Support
- **D-14:** Must support Linux, macOS, Windows — platform-specific binary replacement handled in swapper (rename semantics for Windows, exec behavior per OS)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Release Package
- `release/release.go` — Current implementation: `GetLatestRelease`, `Download`, GitHub API client
- `release/release.go` §55-67 — `getCurrentModule()` reads build info for auto-detection
- `release/release.go` §109-128 — `Asset.Download` with checksum verification via `go-digest`

### Roadmap & Planning
- `.planning/ROADMAP.md` — Phase 4 entry with milestone definition

### Notes from Exploration
- `.planning/notes/release-architecture-decisions.md` — Architecture decisions from Socratic exploration

### Pending Todos
- `.planning/todos/implement-release-detection-download.md` — Detection/download implementation tasks
- `.planning/todos/implement-swapper-binary.md` — Swapper binary implementation tasks

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `release/release.go` — Existing GitHub release fetcher with digest verification (SHA256 via `go-digest`)
- `release/release.go` §55-67 — Build info reader for repository auto-detection
- `release/release.go` §69-107 — `GetLatestRelease` / `GetThisLatestRelease` — detection and API interaction
- `release/release.go` §109-128 — `Asset.Download` with checksum verification against asset digest field

### Established Patterns
- Functional options pattern for configuration (consistent with `cache/`, `config/` packages)
- `//go:embed` for bundling assets (follow Go stdlib approach)
- Separate sub-packages for distinct concerns (consistent with `cache/` provider pattern)

### Integration Points
- `release` package — extension point for download/verify
- New `release/swapper` package — embedded binary and swap logic
- `debug.ReadBuildInfo()` — auto-detection integration (stdlib, no dependency)

</code_context>

<specifics>
## Specific Ideas

- Swapper should re-verify checksum before exec as second line of defense
- Original program args forwarded identically to new binary
- Backup/restore for bulletproof replacement on any platform
- `go embed` solves both trust chain and AV false-positive concerns

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 4-Release self-update with swapper binary*
*Context gathered: 2026-07-21*

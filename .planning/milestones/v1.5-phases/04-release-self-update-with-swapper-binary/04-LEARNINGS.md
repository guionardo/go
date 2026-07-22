---
phase: 04
phase_name: Release self-update with swapper binary
project: "go - Golang tools, examples, and packages"
generated: "2026-07-21"
counts:
  decisions: 7
  lessons: 4
  patterns: 4
  surprises: 2
missing_artifacts:
  - UAT.md
---

# Phase 4 Learnings: Release self-update with swapper binary

## Decisions

### Two-Phase SHA256 Verification
Binary digest is verified at download time (go-digest) and again at swap time (stdlib SHA256).

**Rationale:** Defense-in-depth. The download verification catches network corruption; the swap-time verification protects against filesystem-level corruption or tampering between download and replacement.

**Source:** 04-02-SUMMARY.md

### Spawn-Exit-Swap-Exec Flow
Self-update spawns a swapper child process, exits the parent, the swapper replaces the binary, then relaunches.

**Rationale:** On Windows, you cannot replace a running executable (file-in-use lock). Spawning a separate process that performs the swap while the parent is still running avoids this issue entirely. The parent exits after spawning, and the swapper handles the rest.

**Source:** 04-02-SUMMARY.md

### --target Flag for Correct Self-Replacement
Swapper accepts a `--target` flag to specify which binary to replace, rather than inferring it from `os.Executable()`.

**Rationale:** Without `--target`, the swapper would try to replace itself (its own process binary) instead of the parent binary. The parent passes its own exe path explicitly via `--target`.

**Source:** 04-02-SUMMARY.md

### hashicorp/go-version Instead of Custom Version Parser
Replaces a custom semver parser with the hashicorp/go-version library.

**Rationale:** Custom semver parsing had edge cases with prereleases, pseudo-versions, and build metadata. `hashicorp/go-version` handles all of these correctly and is a widely-used, stable dependency.

**Source:** 04-01-SUMMARY.md

### //go:embed for Swapper Distribution
Pre-compiled swapper binaries are embedded into the release package via `//go:embed`.

**Rationale:** No installer needed — the swapper binary is extracted at runtime from the running executable. Works across all 4 target platforms without additional distribution mechanisms.

**Source:** 04-03-SUMMARY.md

### File-Based Lock for Concurrency Control
A `.update.lock` file next to the executable prevents concurrent update processes.

**Rationale:** Atomic `O_EXCL|O_CREATE` file creation provides cross-process synchronization without external dependencies. The lock is cleaned up via defer regardless of success or failure.

**Source:** 04-03-SUMMARY.md

### Functional Options for Update Configuration
`WithOwner`, `WithRepo`, `WithGitHubToken` follow the established functional options pattern.

**Rationale:** Consistent with the cache package's functional options pattern. Users only specify what they need to override — owner/repo auto-detect from build info by default.

**Source:** 04-01-SUMMARY.md

## Lessons

### VERIFICATION.md Requires YAML Frontmatter
GSD tools parse `status: passed` from YAML frontmatter, not from markdown `**Status:** passed`.

**Context:** The initial VERIFICATION.md used markdown formatting which the GSD verification query couldn't parse, reporting "missing" status despite the file existing.

**Source:** 04-VERIFICATION.md

### Swapper Binaries Must Be Pre-Built Before go build
The `//go:embed` directives require swapper binaries to exist before compilation. They are gitignored (build artifacts).

**Context:** CI failed on `go build ./...` because `make swapper` hadn't been run first. Both workflows need explicit swapper build steps.

**Source:** 04-03-SUMMARY.md

### govulncheck Output Is Line-Delimited JSON
govulncheck outputs multiple JSON objects separated by newlines, not a single JSON document.

**Context:** The quality report script initially tried to parse govulncheck output as a single JSON object, which failed. Must iterate line-by-line to find the `Vulnerabilities` block.

**Source:** quality-report.sh

### Windows File Permissions Differ from Unix
`os.WriteFile` with 0755 mode produces 0666 on Windows because Windows doesn't support Unix executable bits.

**Context:** The `TestExtractSwapper` test checked `info.Mode().Perm() == 0o755`, which failed on Windows CI where it returned `0o1b6` (0666). Fixed by skipping the permission check on Windows.

**Source:** 04-03-SUMMARY.md

## Patterns

### Embedded Binary for Self-Update
The self-update mechanism uses an embedded swapper binary (spawn → exit → swap → exec).

**When to use:** When a CLI tool needs to update itself atomically across platforms, especially Windows where file-in-use locks prevent replacing the running binary.

**Source:** 04-02-SUMMARY.md

### Cross-Platform via Build Tags
Platform-specific implementations use `//go:build !windows` and `//go:build windows` build tags.

**When to use:** When a feature needs different implementations per OS. One file per OS with build tags is cleaner than runtime OS checks.

**Source:** 04-02-SUMMARY.md

### Two-Phase Verification for Critical Operations
Verify at every handoff point in a multi-step pipeline.

**When to use:** When an operation involves multiple independent steps (download → store → replace), verify at each transition. Catches failures mid-pipeline rather than at the end.

**Source:** 04-02-SUMMARY.md

### Lock File for Cross-Process Synchronization
A file-level lock using `O_EXCL|O_CREATE` provides simple, reliable mutual exclusion.

**When to use:** When you need to prevent concurrent access to a resource across process boundaries without external dependencies (no Redis, no database).

**Source:** 04-03-SUMMARY.md

## Surprises

### CI Blocks Force Push
GitHub branch protection rules on `main` reject `--force` pushes.

**Impact:** Required `--force-with-lease` or normal push with new commits, adding overhead when amending commits.

**Source:** Development workflow — observed during milestone completion

### govulncheck JSON Format Is Non-Standard
govulncheck uses line-delimited JSON instead of a single JSON object.

**Impact:** Required custom parsing logic in the quality report script. Not a bug, but an unexpected format that caused initial parsing failures.

**Source:** quality-report.sh

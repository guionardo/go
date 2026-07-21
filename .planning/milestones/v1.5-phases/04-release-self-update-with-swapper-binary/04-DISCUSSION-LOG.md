# Phase 4: Release self-update with swapper binary - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-21
**Phase:** 4-Release self-update with swapper binary
**Areas discussed:** Architecture design, Milestone placement

---

## Architecture Design (from earlier Socratic exploration)

| Option | Description | Selected |
|--------|-------------|----------|
| Download binary directly | Simple but AV false positives, trust chain unclear | |
| Embed swapper via `go:embed` | Ships swapper with main binary, solved trust + AV issues | ✓ |

**User's choice:** Embed swapper via `//go:embed`
**Notes:** Avoids downloading executables (AV false positive risk). The trust chain is: download → verify → swapper re-verifies before exec.

| Option | Description | Selected |
|--------|-------------|----------|
| Replace binary in-place | File locking issues on Windows, no rollback | |
| Spawn swapper → exit → swap → relaunch | Clean, avoids file-in-use, enables backup/restore | ✓ |

**User's choice:** Spawn swapper → exit → swap → relaunch with original os.Args

---

## Milestone Placement

| Option | Description | Selected |
|--------|-------------|----------|
| v1.5 alongside Strings/Retry | Ship all three together | |
| v1.6 after Strings and Retry | Keep v1.5 focused on current roadmap | |
| v1.5 standalone, Strings/Retry removed | Self-update becomes v1.5 deliverable | ✓ |

**User's choice:** v1.5 standalone, Strings/Retry removed from roadmap entirely
**Notes:** Strings and Retry packages no longer needed. v1.5 milestone renamed to "Self-Update".

---

## Remaining Open Questions (deferred by user)

- Platforms specifics (Windows rename, macOS code signing)
- Update checking frequency (CLI command vs scheduled)
- Swapper notification mechanism (stdout vs exit code vs temp file)

---
title: Release Package Architecture Decisions
date: 2026-07-21
context: Socratic exploration of self-update mechanism for go monorepo
---

# Release Package Architecture Decisions

## Core Objective

Detect importing program's GitHub repository, check for newer releases, download, verify, and replace the binary in-place.

## Self-Update Flow

1. Main program detects version / checks GitHub API for newer release
2. Downloads release artifact and verifies checksum (SHA256 via go-digest)
3. Spawns embedded swapper binary with `--new-binary=<path>` + original program args
4. Main program exits
5. Swapper waits for exit, backs up old binary, copies new one
6. Swapper re-verifies checksum of new binary before exec
7. If swap fails → restore backup. If success → relaunch with original args
8. Swapper itself never needs updating — it ships embedded in every release

## Resolution Modes

- **Auto:** Detect via `debug.ReadBuildInfo()` — works for any Go program built with module info
- **Explicit:** Caller provides owner/repo directly

## Trust Chain

- Download → verify checksum → temp hold → swapper re-verifies before exec
- Never runs unverified code, even if download verification was bypassed

## Swapper Architecture

- Separate package: `release/swapper`
- Embedded in main binary via `//go:embed`
- Receives: `--new-binary=<path>` + original os.Args
- Handles: backup, copy, rollback, relaunch, cleanup

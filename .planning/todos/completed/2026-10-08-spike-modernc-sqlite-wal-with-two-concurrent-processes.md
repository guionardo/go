---
created: 2026-10-08T11:46:07Z
title: Spike modernc SQLite WAL with two concurrent processes
area: database
severity: minor
resolves_phase: 16
files:
  - cache/ (future cache/sqlite provider)
---

## Problem

The planned embedded SQLite cache provider (seed SEED-261008-2o5) assumes multi-process access works through WAL + `busy_timeout` with the pure-Go `modernc.org/sqlite` driver. The 2026-10-08 research pass that supported this was tier-floored (unresolved) — the behavior has not been verified against a running system.

## Solution

Build a throwaway spike: two processes writing concurrently to one SQLite file via `modernc.org/sqlite` with `journal_mode=WAL` and `busy_timeout`, and observe SQLITE_BUSY behavior, retry needs, checkpoint timing, and WAL growth. Record the pragma set that works. Run before the provider is planned (or at latest before implementation). Findings belong in `.planning/research/questions.md` and the spike's own notes.

# Phase 16: Batch Surface + Multi-Process Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-08
**Phase:** 16-batch-surface-multi-process-hardening
**Areas discussed:** Batch chunk size, MGet error policy, Contention + retry policy, CI E2E scope

---

## Batch chunk size

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed constant 100 | Chunked IN-lists capped at 100 keys per query; no new API surface | ✓ |
| Fixed constant 500 | Fewer round-trips for big batches, still under SQLite parameter limit | |
| Configurable option | WithChunkSize option for per-workload tuning | |

**User's choice:** Fixed constant 100
**Notes:** Matches pragmatic SQLite limits; changeable later without breaking callers.

| Option | Description | Selected |
|--------|-------------|----------|
| Dedup + no-op empty | MGet empty → empty map, no query; duplicate keys dedup before chunking | ✓ |
| Pass-through duplicates | Keys passed verbatim; IN(k,k) reads identically | |

**User's choice:** Dedup + no-op empty

---

## MGet error policy

| Option | Description | Selected |
|--------|-------------|----------|
| Log + skip chunk | slog warn, skip failed chunk's keys, continue remaining chunks, return found-so-far | ✓ |
| Abort remaining chunks | First errored chunk stops all later chunk queries | |

**User's choice:** Log + skip chunk
**Notes:** MGet's map return has no error channel — best-effort parity with mem/redis.

| Option | Description | Selected |
|--------|-------------|----------|
| Atomic, pre-validate marshal | Values marshaled up-front; any error rolls back the single transaction | ✓ |
| Lazy marshal in-tx | Marshal inside the transaction; same atomicity, wasted tx work | |

**User's choice:** Atomic, pre-validate marshal

---

## Contention + retry policy

| Option | Description | Selected |
|--------|-------------|----------|
| Bounded open retry | New retries the WAL-conversion open up to busy_timeout with tiny backoff | ✓ |
| Quarantine test only | No retry; quarantine the flaky concurrent-first-open test | |
| BEGIN-level retry only | Retry at BEGIN-immediate — does not fix the conversion race | |

**User's choice:** Bounded open retry
**Notes:** DI-15-01 fix; New stays never-failing; ops defer error if retries exhaust.

| Option | Description | Selected |
|--------|-------------|----------|
| 64MB default via DSN | `_pragma=journal_size_limit(67108864)` in file mode | ✓ |
| Keep unlimited | SQLite default; rely on checkpoint + sweeps | |
| Explicit option | WithJournalSizeLimit alongside WithAutoCheckpoint | |

**User's choice:** 64MB default via DSN
**Notes:** Bounds passive WAL growth between checkpoints in long-running processes.

| Option | Description | Selected |
|--------|-------------|----------|
| Open-only, stateless ops | Retry only open/WAL-conversion; busy_timeout for BEGIN contention | ✓ |
| Also maintenance ops | Add bounded retry to Checkpoint/Vacuum | |

**User's choice:** Open-only, stateless ops
**Notes:** Milestone's no-retry-middleware decision holds.

---

## CI E2E scope

| Option | Description | Selected |
|--------|-------------|----------|
| All 3 CI OSes | Two-process E2E on ubuntu/macos/windows — Windows WAL semantics proven | ✓ |
| Ubuntu only | Multi-process proof deferred to Phase 17 on Windows/macOS | |

**User's choice:** All 3 CI OSes

| Option | Description | Selected |
|--------|-------------|----------|
| Contention + crash recovery | Concurrent read/write + kill-mid-write; survivor continues, reopen clean | ✓ |
| Contention only | Basic concurrent scenario; crash recovery implicit | |

**User's choice:** Contention + crash recovery

---

## the agent's Discretion

- MGet result ordering (map return — no ordering guarantee needed)
- Spike fixtures/timing, helper-process plumbing details
- AutoCheckpoint/journal_size_limit DSN pragma interaction (keep additive)

## Deferred Ideas

- None — discussion stayed within phase scope. The pending two-process WAL spike todo was folded into decisions (CONC-01 exit criterion).
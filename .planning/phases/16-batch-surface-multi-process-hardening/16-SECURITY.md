---
phase: "16"
slug: "batch-surface-multi-process-hardening"
status: verified
threats_open: 0
asvs_level: 1
created: "2026-10-26"
---

# Phase 16 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Retroactive audit of a completed phase (State B — no prior SECURITY.md). Register authored at plan time across 16-01…16-05; verified at ASVS L1 (grep depth) against the shipped implementation.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| caller → `cache/sqlite` BatchCache | Untrusted keys, TTLs and decoded values cross into generated SQL and the pinned pool | Caller keys/values (arbitrary Go types, JSON-marshalled) |
| process A ↔ process B → shared DB file | Two OS processes open the same WAL database concurrently | Cache rows, WAL/journal data, busy-lock state |
| test harness → re-exec'd child | The e2e harness re-executes its own binary as a helper process | Env `SQLITE_E2E_HELPER`, child stdout/stderr |
| caller → `release.DownloadUpdate` | Untrusted GitHub release metadata drives a file write into the target dir | Asset name/URL/digest, downloaded bytes |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-16-01 | Spoofing | `twoprocess_e2e_test.go` child re-exec (TestMain env intercept) | low | mitigate | Child role activates only on the exact env value `SQLITE_E2E_HELPER=1`; harness is `//go:build e2e`-tagged, absent from production binaries and the unit lane | closed |
| T-16-02 | Denial of Service | Spawned child processes on failure paths | low | mitigate | Children killed and `Wait`-ed in defer/cleanup; bounded polling/marker waits; child stdout/stderr captured; `t.TempDir` cleanup owned | closed |
| T-16-03 | Tampering | `buildDSN` `journal_size_limit` `_pragma` value | high | mitigate | Compile-time constant / `strconv.FormatInt` of int64 — never a caller string; appended in file mode only; existing `?`/`#` path rejection unchanged | closed |
| T-16-04 | Denial of Service | `retryBusy` loop in `open()` | medium | mitigate | Busy-only classifier, absolute 5 s deadline checked before every attempt, ctx cancellation honored, single call site, non-busy errors fail fast | closed |
| T-16-05 | Tampering | Busy classifier misclassifying extended result codes | medium | mitigate | Primary code compared via `sqliteBusyCode`/`sqliteBusyMask` (`Code() & 0xff == 5`); unit matrix covers 5, 261, 517, 1, plain and wrapped errors; no string matching | closed |
| T-16-06 | Tampering | Chunked IN SQL generation in `batch.go` | high | mitigate | Only `?` markers generated (`inPlaceholders`); keys/values/`now` are bound parameters; SQL templates carry no data; dedup-then-chunk order fixed | closed |
| T-16-07 | Information Disclosure | Batch logging (chunk warns, decode debug) | low | mitigate | Warn logs carry key counts + wrapped error only; key strings appear only at debug for decode skips; no new logging surface | closed |
| T-16-08 | Denial of Service | Unconsumed `*sql.Rows` on the pinned connection | medium | mitigate | `defer rows.Close()` inside `mgetChunk` scope — the handle closes before the next chunk query; `rows.Err()` checked | closed |
| T-16-09 | Tampering | Partial writes / atomicity break in `MSet` | medium | mitigate | Pre-marshal before `BeginTx`; single transaction with `defer Rollback`; one prepared upsert; 0-or-all invariant test repeated across cancellation runs | closed |
| T-16-10 | Denial of Service | E2E child processes / handles on CI runners | low | mitigate | Every child killed and waited in cleanup; bounded `require.Eventually` marker checks; 180 s per-step timeouts; temp-dir owned files; no permission assertions | closed |
| T-16-11 | Information Disclosure | Child stdout/stderr captured into CI logs on failure | low | mitigate | Children print operational diagnostics only — no values beyond test fixtures, no secrets, no machine paths; buffers surfaced only on assertion failure | closed |
| T-16-12 | Tampering | `release/update.go` `DownloadUpdate` cleanup path | low | mitigate | Explicit `f.Close()` before `removeFile(filePath)` so Windows `DeleteFile` succeeds; removal failures surfaced via `errors.Join` (never silent); `TestDownloadUpdate_DigestMismatch` + `TestDownloadUpdate_CleanupFailureSurfaced` pin both behaviors | closed |
| T-16-SC | Tampering | npm/pip/cargo installs | high | mitigate | No package installs in this phase — `go.mod`/`go.sum` untouched across all 9 phase commits; the reserved row stands for any future induced install | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

No accepted risks.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-26 | 13 | 13 | 0 | gsd-secure-phase (orchestrator, ASVS L1 grep depth; register authored at plan time — auditor spawn skipped per the L1 short-circuit) |

---

## Verification Notes

- **Register origin:** authored at plan time — every plan 16-01…16-05 carries a parseable `<threat_model>` STRIDE block; `register_authored_at_plan_time: true`.
- **Depth:** ASVS level 1 (grep depth) is sufficient by policy; short-circuit applied (`threats_open: 0` ∧ plan-time register ∧ L1), so no `gsd-security-auditor` spawn was required.
- **Evidence gathered:** build tags + env guard (`twoprocess_e2e_test.go`), child Kill/Wait + `require.Eventually` cleanup, `journal_size_limit` as an int parameter with DSN test assertions, `sqliteBusyCode`/`sqliteBusyMask` + absolute deadline in `retry.go`, `inPlaceholders`-only SQL generation, `logger().Warn/Debug` payload shapes in `batch.go`, `defer rows.Close()` + `rows.Err()`, `BeginTx`/`defer Rollback`/`PrepareContext` in `MSetFunc`, close-then-remove + `errors.Join` in `release/update.go`, and zero `go.mod`/`go.sum` changes in the phase diff.
- **Out-of-scope observation (recorded, not a phase-16 threat):** the code review flagged a pre-existing defense-in-depth gap where `Asset.Name` from GitHub metadata is joined into `os.Create` without sanitization in `release/update.go`. It predates this phase, is not covered by any phase-16 requirement or threat row, and is tracked as `open` in `16-REVIEW-DISPOSITION.md`.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log (none)
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-26

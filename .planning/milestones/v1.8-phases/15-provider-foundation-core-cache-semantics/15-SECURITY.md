---
phase: "15"
slug: provider-foundation-core-cache-semantics
status: verified
threats_open: 0
asvs_level: 1
created: "2026-10-08"
---

# Phase 15 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| caller path/name → DSN construction | Untrusted caller strings become part of a driver DSN; everything after the first `?` is driver query territory | cache paths, cache names |
| caller keys/values → SQL statements | Untrusted key/value data crosses into SQLite statements | cached keys and values (JSON) |
| cache file → local filesystem | The database file and its sidecars live in a user-writable directory; other local users may read them | cache entries at rest |
| provider errors/logs → operator output | Error strings and log records must not leak cached keys/values | error strings, log records |
| time source → expiry predicate | Wall-clock comparison decides visibility of cached data; a divergent or drift-prone time expression would resurrect or hide entries | expiry timestamps |
| sweep goroutine → cache operations | Background maintenance shares the single pinned connection with foreground operations | SQL statements, connection state |
| last-connection close → on-disk state | Sidecar removal and checkpoint decide whether committed transactions survive a restart | WAL files, database file |
| caller → maintenance API | Checkpoint/Vacuum are caller-invoked and can lock or rewrite the database; misuse is a self-inflicted availability risk | VACUUM / wal_checkpoint calls |
| option value → DSN | `WithAutoCheckpoint`'s int crosses into DSN construction | autocheckpoint page count |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-15-01 | Tampering | `resolveLocation`/`buildDSN` path+name handling (DSN injection, path traversal) | high | mitigate | `strings.ContainsAny(path, "?#")` → `ErrInvalidPath`; name allow-list `[A-Za-z0-9._-]` rejecting `''`, `.`, `..`, separators; name mode always joins under `os.UserCacheDir()`; DSN pragma values are compile-time constants only | closed |
| T-15-02 | Tampering | SQL statements in primitive methods (SQL injection via keys/values) | high | mitigate | Every statement binds positional `?` parameters (`WHERE cache_key = ?`, upsert with bound value/expires_at); keys/values are data, never identifiers; no string-built SQL; parity tests exercise quotes/unicode/empty values | closed |
| T-15-03 | Information Disclosure | cache file + sidecars readable by other local users | medium | mitigate | Cache directory created with `MkdirAll(…, cacheDirPerm)` where `cacheDirPerm = 0o700`; file permissions inherit the process umask; no encryption at rest (ENCR-01 deferred) — documented in doc.go as data-at-rest caveat | closed |
| T-15-04 | Information Disclosure | keys/values leaking through errors or logs | medium | mitigate | Errors wrap err values and never interpolate keys/values; logger calls carry only mode/path/error metadata (`cache/sqlite:` prefix); no debug logging of entries | closed |
| T-15-05 | Denial of Service | unconsumed `sql.Rows` / `Row` holding the single pinned connection (deadlock) | medium | mitigate | `ExecContext` for all DDL/DML; `QueryRowContext` + `Scan` for singleton SELECTs (6 context-aware call sites); batch placeholders iterate point ops only; Pitfall 11 encoded as a review rule | closed |
| T-15-06 | Tampering | expiry evaluation (`SelectSQL`/`SweepSQL` bound now) | medium | mitigate | Expiry compared in SQL against a per-call bound `time.Now().UnixNano()` against the stored absolute UnixNano column; never SQL `datetime('now')`; TTL matrix + across-restart tests | closed |
| T-15-07 | Denial of Service | sweeper goroutine lifecycle (leak, tick pile-up, Close hang) | medium | mitigate | Single ticker; each tick completes before the next select; Close cancels via stop channel and waits on done before DB close; `d <= 0` creates no goroutine; disabled path asserted via nil channels | closed |
| T-15-08 | Repudiation / Data loss | unclean shutdown trading committed rows for sidecar removal | high | mitigate | Sidecar cleanup only via the engine's last-connection close (probe-verified); STOR-05 tests assert file survives + sidecars vanish + reopen returns unexpired entries; `os.Remove` absent from provider code (prohibition grep clean) | closed |
| T-15-09 | Denial of Service | VACUUM / wal_checkpoint misuse (exclusive lock, blocked checkpoints) | medium | mitigate | Explicit-caller-only surface (no hidden maintenance); cost classes documented in Optimizable comments; blocked checkpoint returns a descriptive error (busy count) instead of hanging; memory mode documented no-op | closed |
| T-15-10 | Tampering | autocheckpoint DSN payload injection | medium | mitigate | The only variable DSN component is `strconv.Itoa` of an int; `pages <= 0` appends nothing; memory mode ignores the option; string-equality DSN tests plus `wal_autocheckpoint` read-back | closed |
| T-15-11 | Information Disclosure / Repudiation | docs overpromising durability (WAL portability) | low | accept | Minimal doc.go states the local-storage/same-host constraint; full operational statements are Phase 17 QUAL-03 — tracked, not silently dropped | closed — accepted risk |
| T-15-SC | Tampering | package installs (Go modules) | high | mitigate | Package Legitimacy Audit in 15-RESEARCH (modernc.org/sqlite + transitives OK/Approved); exact pins (`modernc.org/sqlite v1.60.1`, `modernc.org/libc v1.77.1`); no install scripts in Go modules; `go list -m` assertion in task 1 | closed |

*Status: open · closed · open — below `high` block_on threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-15-11 | doc.go durability wording kept minimal; full operational docs are Phase 17 QUAL-03 scope | planner decision (documented in plan) | 2026-10-08 |
| AR-02 | T-15-03 (note) | No encryption at rest — ENCR-01 deferred to future release (blocked by pure-Go driver; only if CGO becomes acceptable); data-at-rest caveat documented in doc.go | milestone decision (REQUIREMENTS.md v2) | 2026-10-08 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-08 | 12 | 12 | 0 | gsd-secure-phase (L1 grep evidence, ASVS L1 short-circuit) |

/gsd-secure-phase re-runs with `threats_open: 0`, register authored at plan time, ASVS L1 → L1 grep-depth classification; auditor not required (short-circuit rule).

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-08
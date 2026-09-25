---
phase: 05
slug: shared-singleflight-helper
status: secured
threats_open: 0
asvs_level: 1
created: 2026-08-06
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Caller ↔ cache helper | Concurrent callers invoke `SingleflightGetOrSet` `Do`/`DoChan`; the helper dedups setters per key | keys, values, errors |
| Leader ↔ setter | The singleflight fn calls the user setter with the leader's `context.Context` | value, error, ctx cancellation |
| Helper ↔ error surface | `cache.Panic` and `cache.ErrCanceled` cross back to every waiter's error return | error values, panic stack traces |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | DoS | callSetter / Do | high | mitigate | Recover inside the singleflight fn (D-16) and return `*cache.Panic` to every waiter (D-17); verified by `TestSingleflightGetOrSet_PanicRecovery` + `require.ErrorAs` | closed |
| T-05-02 | Tampering | Do fn body | high | mitigate | Double-check Get inside the fn (SF-08) so a stale in-flight computation cannot clobber a concurrent direct Set; verified by `TestSingleflightGetOrSet_ClobberGuard` | closed |
| T-05-03 | DoS | Do fast-path | medium | mitigate | Fast-path Get stays OUTSIDE the group (SF-02) to avoid singleflight lock contention on cache hits; verified by the second-phase cached-read assertion | closed |
| T-05-04 | Information Disclosure | Panic.Error() | low | accept | Stack traces surface in error text — same behavior as x/sync panicError (D-17); errors are caller-visible by design; no secret material expected in recovered values | closed |
| T-05-05 | DoS | Do (blocking) | medium | accept | Do is context-blind for waiters (D-06) — a long setter holds waiters until completion; documented on godoc; the DoChan variant (plan 05-02) is the HTTP-path choice (spike 008) | closed |
| T-05-06 | DoS | DoChan waiter select | high | mitigate | Select on ctx.Done() so a canceled waiter abandons at cancel latency instead of blocking for the setter; verified by `TestSingleflightGetOrSet_DoChan_CanceledWaiter` (~15 ms validated in spike 008) | closed |
| T-05-07 | Spoofing | ErrCanceled / error identity | medium | mitigate | `ErrCanceled` wraps the waiter's ctx.Err() with two `%w` verbs so `errors.Is` matches both `cache.ErrCanceled` and `context.Canceled` (D-10); test asserts both | closed |
| T-05-08 | DoS | shared computation | medium | mitigate | A waiter cancel must not cancel/abort the leader (D-04): only the waiter's own select is gated by its ctx; leader's fn completes and Set lands — asserted in the canceled-waiter post-release store check | closed |
| T-05-09 | Information Disclosure | Result channel | low | mitigate | DoChan never returns or exposes the `singleflight.Result` channel (D-12) — callers only see `(V, error)` | closed |
| T-05-10 | Tampering | provider GetOrSet bodies | high | mitigate | Every provider body kept byte-identical except the setter signature + call — no error-prefix changes (D-14), no logic changes; verified by `go build` + acceptance string checks on `cache/redis: %w` / `cache/valkey:` | closed |
| T-05-11 | Elevation of Privilege | Cache interface assertion | medium | mitigate | Compile-time assertions (`var _ cache.Cache[string, any]`) fail loudly if any provider signature drifts from the interface — enforced at compile time | closed |
| T-05-12 | Tampering | setter ctx propagation | low | mitigate | Providers pass the caller's ctx to the setter exactly once (D-06); a setter never receives a foreign/background ctx — verified by reading the five GetOrSet bodies | closed |
| T-05-13 | DoS | DoChan leader path | high | mitigate | ctx.Done() case re-reads ch non-blocking then consults the leader signal (leaderCh); a leader blocks on ch and returns the raw setter ctx.Err() (D-04/D-11); verified by `TestSingleflightGetOrSet_DoChan_LeaderCancel` asserting `errors.Is(err, context.Canceled)` && `!errors.Is(err, cache.ErrCanceled)` | closed |
| T-05-SC | Tampering | module deps | low | accept | No new packages this phase; golang.org/x/sync v0.22.0 already pinned in go.mod (D-15) — no package-legitimacy checkpoint required | closed |

*Status: open · closed · open — below block_on threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-05-01 | T-05-04 | Panic stack traces in error text are caller-visible by design; same as x/sync panicError; no secret material expected | orchestrator | 2026-08-06 |
| R-05-02 | T-05-05 | Blocking `Do` is context-blind for waiters by design; `DoChan` is the cancel-aware HTTP-path choice | orchestrator | 2026-08-06 |
| R-05-03 | T-05-SC | No new module dependencies this phase; x/sync already pinned — no package-legitimacy checkpoint required | orchestrator | 2026-08-06 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-06 | 14 | 14 | 0 | orchestrator (L1 grep-depth, ASVS 1, register authored at plan time) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed

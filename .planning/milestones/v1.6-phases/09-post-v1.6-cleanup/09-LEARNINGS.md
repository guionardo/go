---
phase: 09
phase_name: "Post-v1.6 cleanup"
project: "github.com/guionardo/go"
generated: "2026-10-08"
counts:
  decisions: 6
  lessons: 3
  patterns: 4
  surprises: 3
missing_artifacts:
  - "09-VERIFICATION.md"
  - "09-UAT.md"
---

# Phase 09 Learnings: Post-v1.6 cleanup

## Decisions

### Config HTTP endpoint uses stdlib net/http — no new dependencies (D-04)
`HTTPHandler()` on `Provider[T]` is built on `net/http` alone; `encoding/json` and `net/http` were the only imports added.
**Rationale:** Keeps the module dependency-free (threat T-09-SC: "No new dependencies added"); the Provider's internal write lock already provides thread safety.
**Source:** 09-01-SUMMARY.md

### ServeMux groups mocks by "METHOD /path" and rebuilds on AddMocks (D-05)
The manual for-loop in `ServeHTTP` was replaced with a mux-backed routing table; `AddMocks()` rebuilds the mux after appending so dynamically registered mocks take effect.
**Rationale:** Backward compatible — the `AddMocks` interface is unchanged, and partial-match detection still works via the path-grouped handler.
**Source:** 09-CONTEXT.md

### Header keys normalized to lowercase with underscore→hyphen before comparison (D-06)
`matchHeaders` normalizes both expected and incoming request keys via `strings.ToLower(strings.ReplaceAll(k, "_", "-"))` and stores `readData` under the normalized key.
**Rationale:** Windows `http.DefaultClient` may not canonicalize `Api_key` → `Api-Key` before sending; comparing lowercase keys handles all OS transport variances.
**Source:** 09-CONTEXT.md

### PATCH/POST only, method check before any body parsing (T-09-02)
The handler rejects non-PATCH/POST with 405 before decoding the body; JSON decode failures return 400 and validation failures return 422.
**Rationale:** Tampering mitigation — the method gate runs before body parsing, and the JSON body is validated against the config struct schema by `Provider.UpdateConfiguration`.
**Source:** 09-01-PLAN.md

### DoS exposure on the config handler accepted (T-09-03)
Rate limiting was not implemented for `HTTPHandler()`.
**Rationale:** The handler is for admin/config tooling, not public-facing; rate limiting is explicitly the caller's concern.
**Source:** 09-01-PLAN.md

### Empty header values: empty matches empty, empty never matches non-empty
`matchHeaders` treats an empty found value as a match only when the expected value is also empty.
**Rationale:** Prevents false positives on missing/blank headers; pinned by the 8 `TestRequest_matchHeaders` subtests including empty-value cases.
**Source:** 09-01-SUMMARY.md

---

## Lessons

### The ServeMux routing swap was fully behavior-preserving
All 75 existing `httptest_mock` tests passed unchanged with zero test modifications and zero new tests for Task 2.
**Context:** The plan anticipated possibly adjusting tests that depended on cross-path iteration order; no such test existed, so the refactor landed as a pure drop-in.
**Source:** 09-01-SUMMARY.md

### Fixing header matching also required fixing header storage
Storing `readData` under the normalized key is what makes `GetHeaderValue("api-key")` and `GetHeaderValue("Api-Key")` both find the value — comparison normalization alone would not have fixed retrieval.
**Context:** The Windows fix spans both the match path and the read-data lookup path; empty-value semantics were pinned by 8 subtests.
**Source:** 09-01-SUMMARY.md

### New feature code landed without coverage remediation
Adding `HTTPHandler()` plus 5 handler tests and 8 header subtests kept the project at 76.2% total coverage, above the 75% `make coverage-quick` gate.
**Context:** Verification shows config 50/50 and httptest_mock 84/84 passing; no coverage carve-outs or extra tests were needed.
**Source:** 09-01-SUMMARY.md

---

## Patterns

### Go 1.22+ ServeMux "METHOD /path" pattern routing
Each mock group is registered as `mux.HandleFunc("METHOD /path", handler)`; the stdlib mux handles method+path dispatch and the grouped handler iterates only its mocks.
**When to use:** Replacing manual request/mock iteration loops — method-aware patterns are native to ServeMux, so no third-party router is needed.
**Source:** 09-01-SUMMARY.md

### Double-checked locking for lazy mux initialisation
`ensureMux()` lazily builds the mux under the write lock only when nil; `rebuildMux()` refreshes it on `AddMocks`, while `ServeHTTP` reads it under the read lock.
**When to use:** Shared routing/building structures rebuilt rarely but read on every request; avoids stale mux after dynamic mock registration.
**Source:** 09-01-SUMMARY.md

### Header key normalization (lowercase + underscore→hyphen) on both sides and at storage
Normalize the expected key, each incoming `req.Header` key, and the `readData` storage key with `strings.ToLower(strings.ReplaceAll(k, "_", "-"))`.
**When to use:** Cross-platform HTTP header matching — handles Windows non-canonical keys such as `Api_key` and makes `GetHeaderValue` casing-agnostic.
**Source:** 09-01-SUMMARY.md

### HTTP config-update handler error mapping
`HTTPHandler()` maps: wrong method → 405, invalid JSON → 400, validation failure (via `UpdateConfiguration`) → 422, success → 200 + JSON-encoded updated config.
**When to use:** Exposing a typed config provider for admin HTTP updates; method check precedes body parsing and validation is delegated to the existing provider.
**Source:** 09-01-PLAN.md

---

## Surprises

### Routing refactor needed zero test changes or additions
The plan's contingency for order-dependent tests never triggered — all 75 existing tests passed unchanged and Task 2 added 0 new tests.
**Impact:** The ServeMux swap was proven a true drop-in refactor; commit `8f6167b` is a pure `refactor` with no behavioral contract change.
**Source:** 09-01-SUMMARY.md

### The header fix leaked into the retrieval API
Storing `readData` under the normalized key means `GetHeaderValue("api-key")` and `GetHeaderValue("Api-Key")` both work — the fix changed lookup behavior, not just match evaluation.
**Impact:** Header retrieval became casing-agnostic; pinned by the 8 `TestRequest_matchHeaders` subtests.
**Source:** 09-01-SUMMARY.md

### No threat flags — the planned threat model covered everything
All implemented surface (HTTP handler method, ServeMux routing, header matching) stayed within the plan's `<threat_model>` scope; no unplanned risk surfaced despite adding an HTTP entry point.
**Impact:** T-09-01/T-09-02 mitigations (schema validation, method-check-first) were sufficient; T-09-03 (DoS) remains accepted as a caller concern.
**Source:** 09-01-SUMMARY.md

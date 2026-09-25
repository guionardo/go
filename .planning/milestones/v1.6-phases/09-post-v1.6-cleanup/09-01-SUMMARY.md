---
phase: 09-post-v1.6-cleanup
plan: 01
subsystem: [config, httptest_mock]
tags: [http-handler, servemux, header-normalization, cleanup]
requires: []
provides: [HTTPHandler, ServeMux-routing, Windows-header-fix]
affects: [config, httptest_mock]
tech-stack:
  added: [encoding/json, net/http]
  patterns:
    - Go 1.22+ ServeMux METHOD /path pattern routing
    - Double-checked locking for lazy mux initialisation
    - Header key normalization (lowercase + underscore→hyphen)
key-files:
  created: []
  modified:
    - config/provider.go
    - config/config_test.go
    - httptest_mock/handler.go
    - httptest_mock/request.go
    - httptest_mock/request_test.go
decisions:
  - D-04: Config HTTP endpoint uses net/http stdlib, no new deps
  - D-05: ServeMux routing groups mocks by method+path, rebuilds on AddMocks
  - D-06: Header keys normalized to lowercase with underscore→hyphen before comparison
metrics:
  duration: 0h 10m
  completed_date: 2026-08-08
status: complete
---

# Phase 9 Plan 1: Post-v1.6 Cleanup Summary

Resolved 3 outstanding cleanup items across `config` and `httptest_mock` packages for v1.6.

## Tasks

### Task 1: Config HTTP endpoint — add HTTPHandler() method to Provider[T]

- **Files:** `config/provider.go`, `config/config_test.go`
- **Commit:** `c0f1ed3`
- **Description:** Added `HTTPHandler()` method on `Provider[T]` that returns an `http.Handler` accepting PATCH/POST requests with JSON body to update configuration. Thread-safe via Provider's internal write lock. Responds with 200 OK + updated config on success, 400 on invalid JSON, 405 on wrong method, 422 on validation failure.
- **Tests (5):** `TestProvider_HTTPHandler_method_not_allowed`, `TestProvider_HTTPHandler_invalid_json`, `TestProvider_HTTPHandler_successful_patch`, `TestProvider_HTTPHandler_post_also_accepted`, `TestProvider_HTTPHandler_validation_error`
- **Threat mitigations:** T-09-01 (validate JSON body against struct schema), T-09-02 (PATCH/POST only, method check before body parsing)

### Task 2: httptest_mock ServeMux routing — replace manual for-loop with http.ServeMux

- **Files:** `httptest_mock/handler.go`
- **Commit:** `8f6167b`
- **Description:** Replaced the mock-by-mock for-loop iteration in `ServeHTTP` with `http.ServeMux` pattern-based routing. Added `mux *http.ServeMux` field, `ensureMux()` with double-checked locking, `rebuildMux()` grouping by `"METHOD /path"`, and `handleMockGroup()` helper extracted from the old loop body. `AddMocks()` now rebuilds the mux after appending mocks. All 75 existing tests pass unchanged.
- **Tests:** 0 new tests; 75 existing tests unchanged with zero behavioral change

### Task 3: httptest_mock Windows header matching — normalize keys to lowercase

- **Files:** `httptest_mock/request.go`, `httptest_mock/request_test.go`
- **Commit:** `ffa382f`
- **Description:** Fixed `matchHeaders` to normalize both expected and request header keys to lowercase with underscores replaced by hyphens before comparison, handling Windows HTTP transport non-canonical header key variances. `readData` is now stored under normalized key, making `GetHeaderValue("api-key")` and `GetHeaderValue("Api-Key")` both find the value. Empty header values are handled correctly.
- **Tests (8 subtests):** exact match, case-insensitive match, underscore-to-hyphen match, Windows non-canonical match, missing header, multiple headers, empty value matches empty expected, non-empty expected with empty found = no match

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

| Check | Status |
|-------|--------|
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ PASS |
| `go test ./config/ -count=1` | ✅ 50/50 passed |
| `go test ./httptest_mock/ -count=1` | ✅ 84/84 passed |
| `make coverage-quick` | ✅ All thresholds met (76.2% total) |

## Commits

| Commit | Type | Description |
|--------|------|-------------|
| `c0f1ed3` | `feat` | `feat(config): add HTTPHandler() method to Provider[T]` |
| `8f6167b` | `refactor` | `refactor(httptest_mock): replace mock-iteration for-loop with http.ServeMux routing` |
| `ffa382f` | `fix` | `fix(httptest_mock): normalize header keys to lowercase in matchHeaders` |

## Known Stubs

None.

## Threat Flags

None — all threat surface (HTTP handler method, ServeMux routing, header matching) was within the planned `<threat_model>` scope.

## Self-Check: PASSED

- [x] `config/provider.go` — HTTPHandler() method added
- [x] `config/config_test.go` — 5 HTTP handler tests added
- [x] `httptest_mock/handler.go` — mux field, rebuildMux(), ServeHTTP delegation, handleMockGroup()
- [x] `httptest_mock/request.go` — matchHeaders lowercase normalization
- [x] `httptest_mock/request_test.go` — 8 matchHeaders subtests
- [x] Commits exist: c0f1ed3, 8f6167b, ffa382f
- [x] All tests pass (50 config + 84 httptest_mock)
- [x] Coverage thresholds met

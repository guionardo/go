# Phase 9: Post-v1.6 cleanup - Context

**Gathered:** 2026-08-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Resolve 3 outstanding todos across config and httptest_mock packages.

1. **Config HTTP endpoint** — Provider exposes an HTTP handler for accepting configuration updates via HTTP
2. **httptest_mock ServeMux routing** — Replace manual for-loop mock matching with http.ServeMux patterns
3. **httptest_mock Windows header matching** — Fix header matching failure on Windows CI
</domain>

<decisions>
## Implementation Decisions

### Config HTTP Endpoint
- Use Go 1.26 `net/http` stdlib — no new dependencies
- Add `HTTPHandler()` method on Provider that returns `http.Handler`
- Handler accepts PATCH/POST with JSON body to update config fields
- Thread-safe: acquire write lock, decode JSON, update fields, release
- Follow existing Provider functional options pattern for configuration

### httptest_mock ServeMux Routing
- Replace the manual for-loop in `ServeHTTP` with `http.ServeMux` pattern-based routing
- Register each mock as a handler on the ServeMux at setup time
- Backward compatible — `AddMocks` interface unchanged
- Partial match detection still works via mux handler chaining

### Windows Header Matching Fix
- Root cause: `http.DefaultClient` on Windows may not canonicalize `Api_key` → `Api-Key` before sending
- Fix: In `matchHeaders`, normalize incoming request headers to lowercase BEFORE lookup
- Always compare lowercase keys — handles all OS transport variances
</decisions>

<canonical_refs>
## Canonical References

- `.planning/todos/pending/config-http-endpoint.md` — Original todo
- `.planning/todos/pending/httptest-mock-servemux.md` — Original todo
- `.planning/todos/pending/httptest-mock-windows-header-match.md` — Original todo
- `config/provider.go` — Provider implementation (add HTTP handler)
- `httptest_mock/handler.go` — MockHandler (replace routing with ServeMux)
- `httptest_mock/request.go` — matchHeaders function (fix Windows normalization)
</canonical_refs>

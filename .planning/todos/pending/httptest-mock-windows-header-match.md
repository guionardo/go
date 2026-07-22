---
title: "Debug httptest_mock header matching failure on Windows"
status: pending
priority: high
created: 2026-07-21
---

**Task:** Investigate and fix `TestMockHandler_ServeHTTP` header matching failure on Windows.

**Symptoms:**
- All `TestMockHandler_ServeHTTP` sub-tests fail on Windows CI with `❌ HEADER Api-Key != test_key`
- The mock defines `Api-Key: test_key`, test sets `req.Header.Add("Api_key", "test_key")`
- Go canonicalizes both to `Api-Key` — the mock shows `Api-Key: test_key`
- Yet `req.Header.Get("Api-Key")` returns empty string on Windows
- The case-insensitive normalization fallback (underscore→hyphen) also fails to find it
- Test passes on Linux and macOS

**Needs:** Debug on a Windows machine to inspect what `req.Header` actually contains when the httptest.Server handler receives the request.

**Hypotheses:**
1. `http.DefaultClient.Do(req)` may strip or transform headers differently on Windows
2. The HTTP/2 transport on Windows may handle header case differently
3. `httptest.Server` may reconstruct request headers differently on Windows

**To reproduce:**
```bash
# Run on Windows
cd go
go test ./httptest_mock/ -count=1 -v -run TestMockHandler_ServeHTTP/ServeHTTP_with_correct_matching_request
```

**Files involved:**
- `httptest_mock/request.go` — `matchHeaders()` function
- `httptest_mock/handler_test.go` — test cases
- `httptest_mock/mocks/examples/example_1.yaml` — mock definition

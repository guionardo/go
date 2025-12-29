# feat(mockhandler): add option to disable partial matching

## Summary
This PR introduces a new `WithDisabledPartialMatch()` option that enforces strict request matching in the HTTP mock handler. When enabled, requests must fully match all defined criteria, and partial matches are treated as no match, returning `404 Not Found` instead of `400 Bad Request`.

## Motivation
In some testing scenarios, developers want strict validation where only complete matches are accepted. The default behavior of showing candidate mocks on partial matches (400 Bad Request) can be too permissive for certain test cases. This feature provides a cleaner, more rigid validation approach.

## Changes

### New Features
1. **`WithDisabledPartialMatch()` option** ([httptest_mock/setup.go](httptest_mock/setup.go#L179-L186))
   - New configuration option to disable partial matching behavior
   - When enabled, only full matches are accepted
   - Partial matches return `404 Not Found` instead of `400 Bad Request`

### Core Changes
2. **Updated request matching logic** ([httptest_mock/mock_request.go](httptest_mock/mock_request.go#L68-L88))
   - Modified `match()` method to accept `disablePartialMatch` parameter
   - Returns `matchLevelNone` instead of `matchLevelPartial` when disabled

3. **MockHandler enhancement** ([httptest_mock/mock_handler.go](httptest_mock/mock_handler.go#L40-L44))
   - Added `disablePartialMatch` field to track the configuration
   - Updated `ServeHTTP` to pass the flag to request matching

### Testing
4. **New test case** ([httptest_mock/mock_handler_test.go](httptest_mock/mock_handler_test.go#L188-L201))
   - `TestMockHandlerNoPartialRequests` verifies the strict matching behavior
   - Confirms that partial matches return `404 Not Found` when the option is enabled

### Documentation
5. **Comprehensive README updates** ([httptest_mock/README.md](httptest_mock/README.md))
   - Added `WithDisabledPartialMatch()` to Options section
   - Updated "Partial Match" section with note about disabling the feature
   - Updated "No Match Behavior" section with new case
   - Added "Strict Matching" example demonstrating the feature

## Benefits
- **Stricter validation**: Enforces complete match requirements for test scenarios requiring rigid behavior
- **Cleaner responses**: Returns standard `404 Not Found` instead of exposing internal mock details
- **Flexible testing**: Allows developers to choose between permissive (default) or strict matching modes
- **Better test clarity**: Makes test expectations more explicit

## Usage Example

```go
server, assertFunc := httptestmock.SetupServer(t,
    httptestmock.WithRequestsFrom("mocks"),
    httptestmock.WithDisabledPartialMatch())
defer assertFunc(t)

// Only fully matching requests will succeed
// Partial matches return 404 instead of 400
resp, err := http.Get(server.URL + "/api/v1/users/123")
```

## Backward Compatibility
✅ This change is fully backward compatible. The default behavior remains unchanged (partial matching enabled), and the new option is opt-in.

## Files Changed
- `httptest_mock/README.md` - Documentation updates (39 additions)
- `httptest_mock/mock_handler.go` - Core handler changes (6 additions, 1 deletion)
- `httptest_mock/mock_handler_test.go` - New test case (15 additions)
- `httptest_mock/mock_request.go` - Request matching logic (6 additions, 1 deletion)
- `httptest_mock/setup.go` - New configuration option (9 additions)

**Total**: 75 additions, 2 deletions across 5 files

---

This PR is ready for review and merging.

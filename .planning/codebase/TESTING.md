# Testing Patterns

**Analysis Date:** 2026-07-21

## Test Framework

**Runner:**
- Go's built-in `testing` package (Go 1.26.4)
- Config: `go.mod` declares `go 1.26.4`
- No external test runner (no `gotestsum` or similar)

**Assertion Library:**
- `github.com/stretchr/testify` v1.11.1 — both `assert` (soft checks) and `require` (hard checks) used
- `testifylint` linter enabled in `.golangci.yml` to enforce testify idioms

**Run Commands:**
```bash
go test ./... -v                    # Run all tests verbosely
go test ./...                       # Run all tests (pre-commit hook)
go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./... -count=1  # Coverage
make test                           # Run all tests
make coverage                       # Run coverage check
```

## Test File Organization

**Location:**
- Test files co-located with source files in the same package directory
- One `*_test.go` per package, typically broken into:
  - `<package>_test.go` — main unit tests
  - `example_test.go` — Example functions
  - Additional grouped `*_test.go` files for large packages (e.g., `set`: `marshal_test.go`, `scanner_valuer_test.go`)

**Naming:**
- Files: `<source>_test.go` (e.g., `set.go` → `set_test.go`, `shell_args.go` → `shell_args_test.go`)
- Functions: `Test<FunctionName>(t *testing.T)` for unit tests
- Examples: `Example<FunctionName>()` for example functions
- Subtests: snake_case names (e.g., `"create_new_should_be_empty"`, `"existing_env_should_return_value"`)

**Package naming convention:**
- **External test packages** (preferred): `package <package>_test` — used for most tests
- **Internal test packages** (white-box): `package <package>` — used when testing unexported functions (e.g., `config/config_test.go` is `package config`, `httptest_mock/setup_test.go` is `package httptestmock`)

**Structure:**
```
pkg/
├── foo.go
├── foo_test.go        # External or internal tests
├── example_test.go    # Example functions
├── bar_test.go        # Additional test file for bar.go
```

## Test Structure

**Primary pattern — subtests with `t.Run`:**
```go
func TestSet_Set(t *testing.T) { //nolint:funlen
    t.Parallel()
    t.Run("create_new_should_be_empty", func(t *testing.T) {
        t.Parallel()

        s := set.New[int]()
        assert.Empty(t, s)
    })
    t.Run("create_new_with_values_should_have_correct_length", func(t *testing.T) {
        t.Parallel()

        s := set.New(1, 2, 3)
        assert.Len(t, s, 3)
    })
    // ...
}
```

**Table-driven tests — used for parameterized cases:**
```go
func TestFromFloat64(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name        string
        input       float64
        expectedNum int64
        expectedDen int64
    }{
        {"zero", 0, 0, 1},
        {"one", 1, 1, 1},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            f, err := fraction.FromFloat64(tt.input)
            fatalIfErr(t, err)
            compare(t, f, tt.expectedNum, tt.expectedDen)
        })
    }
}
```

**Key characteristics:**
- Every subtest calls `t.Parallel()` — maximum parallelism
- Table-driven tests also use `t.Parallel()` inside the subtest closure
- Descriptive test function names starting with `Test`
- Test helper functions use `t.Helper()` to attribute failures correctly
- `//nolint:funlen` suppression on long test functions

## Mocking

**Mock framework:** Custom `httptestmock` package (`/Users/guionardo/dev/go/httptest_mock/`)

**Patterns — standard test server setup:**
```go
func TestMockHandler_ServeHTTP(t *testing.T) { //nolint:funlen
    t.Parallel()

    s, assertFunc := httptestmock.SetupServer(t,
        httptestmock.WithRequestsFrom(path.Join("mocks", "examples")),
        httptestmock.WithAddMockInfoToResponse(),
        httptestmock.WithAcceptingPartialMatch())
    defer assertFunc(t)

    t.Run("example_1_exactly_matching_should_return_200_OK", func(t *testing.T) {
        t.Parallel()
        req := httptestmock.CreateTestRequest(t, s,
            http.MethodPost, "/api/v1/users/123?user_id=123",
            "TEST_BODY")
        req.Header.Add("Api_key", "test_key")

        resp, respBody, mockName, err := doRequest(t, req)
        require.NoError(t, err)
        assert.Equal(t, "example_1", mockName)
        require.Equal(t, http.StatusOK, resp.StatusCode)
        require.JSONEq(t, `{"message":"Hello, world!"}`, string(respBody))
    })
}
```

**Patterns — programmatic mock with builder:**
```go
mock := httptestmock.NewMock("GET", "/hello").
    WithResponseStatus(200).
    WithResponseBody("Hello, World!")
server, assert := mock.FastServe(t)
defer assert(t)
```

**Mock configuration from files:**
- Mocks defined as JSON or YAML in `mocks/` directories
- Loaded with `WithRequestsFrom("mocks/examples")`
- Auto-named from filename if `name` field not set

**What to mock:**
- External HTTP services (via `httptest.Server`)
- Timing and delays (via `DelayMs` field in mock response)
- Environment variables (via `t.Setenv()`)

**What NOT to mock:**
- Internal functions — prefer real implementations with test fixtures
- File I/O — use `t.TempDir()` with real files

## Fixtures and Factories

**Test data:** Defined inline as struct fields or local variables

**Temp directories:** `t.TempDir()` for filesystem operations:
```go
tmp := t.TempDir()
profilePath := path.Join(tmp, "default.yml")
require.NoError(t, os.WriteFile(profilePath, []byte("name: test"), 0644))
```

**Environment variables in tests:** `t.Setenv()` for temporary env var overrides:
```go
t.Setenv("TESTCFG_NAME", "env-name")
t.Setenv("TESTCFG_VERSION", "99")
```

**Custom helper functions with `t.Helper()`:**
```go
func fatalIfErr(t *testing.T, err error) {
    t.Helper()
    assert.NoError(t, err)
}

func compare(t *testing.T, f fraction.Fraction, numerator, denominator int64) {
    t.Helper()
    assert.Equalf(t, numerator, f.Numerator(), "expected numerator value to be %v, got %v", numerator, f.Numerator())
    assert.Equal(t, denominator, f.Denominator(), ...)
}

func doRequest(t *testing.T, req *http.Request) (resp *http.Response, body []byte, mockName string, err error) {
    t.Helper()
    // ...
}
```

**YAML profile test data:** Written to `t.TempDir()` by the test:
```go
tmp := t.TempDir()
profilePath := path.Join(tmp, "default.yml")
err := os.WriteFile(profilePath, []byte("name: profile-name\nversion: 42"), 0644)
```

## Coverage

**Requirements (`.testcoverage.yml`):**
- **Total:** 95%
- **Package:** 80%
- **File:** 70%
- **Override:** `^pkg/mid/$` → 50% (platform-specific machine ID package)

**Run coverage:**
```bash
make coverage
# Executes:
#   go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./... -count=1
#   go-test-coverage --config=./.testcoverage.yml
```

**Tool:** `github.com/vladopajic/go-test-coverage/v2`

**View Coverage:**
```bash
go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
go tool cover -html=./cover.out
```

## Test Types

**Unit Tests:**
- Every package has comprehensive unit tests
- Standard pattern: `TestFunctionName` with multiple `t.Run` subtests
- Table-driven tests for parameterized scenarios
- External test packages preferred (`package xxx_test`) for black-box testing

**Integration Tests:**
- HTTP integration tests via `httptestmock` package using real `httptest.Server`
- Tests make real HTTP calls to the test server
- Mock definitions stored as JSON/YAML fixture files
- Pre/post request hooks for custom behavior

**Example Tests:**
- Every package has `example_test.go` with `ExampleXxx()` functions
- Used as both documentation and regression tests
- Verified by `go test` with `// Output:` annotations

**Concurrent/Safety Tests:**
- Concurrent access tests for shared state (e.g., `Provider.GetConfiguration()` tested with goroutines and channels)
```go
t.Run("concurrent_safe", func(t *testing.T) {
    done := make(chan struct{})
    go func() {
        _, _ = provider.GetConfiguration()
        close(done)
    }()
    _, err = provider.GetConfiguration()
    require.NoError(t, err)
    <-done
})
```

## Common Patterns

**Async Testing:**
```go
eg := &errgroup.Group{}
eg.SetLimit(10)
for range totalRequests {
    eg.Go(func() error {
        req := httptestmock.CreateTestRequest(t, mockServer,
            http.MethodGet, "/health", nil)
        _, _, _, err := doRequest(t, req)
        return err
    })
}
err := eg.Wait()
require.NoError(t, err)
```

**Error Testing:**
```go
// Using require.ErrorIs for sentinel errors
_, err = fraction.New(1, 0)
require.ErrorIs(t, err, fraction.ErrZeroDenominator)

// Using errors.Is directly
if _, err = fraction.FromFloat64(math.NaN()); !errors.Is(err, fraction.ErrInvalid) {
    t.Fatalf("expected ErrInvalid, got %v", err)
}

// Using require.Error / require.NoError for generic checks
require.NoError(t, err)
require.Error(t, err)
```

**Panic Testing:**
```go
assert.Panics(t, func() {
    NewProvider[string]()
})
```

**JSON Response Matching:**
```go
require.JSONEq(t, `{"message":"Hello, world!"}`, string(respBody))
```

**Coverage for edge cases:**
```go
// For coverage on break
for range s1.Filter(func(string) bool { return true }) {
    break
}
```

## Assertion Patterns

**Soft assertions** (`assert.*`) — test continues on failure:
- `assert.Equal(t, expected, got)`
- `assert.True(t, condition)`
- `assert.False(t, condition)`
- `assert.Empty(t, value)`
- `assert.NotEmpty(t, value)`
- `assert.NoError(t, err)`
- `assert.Error(t, err)`
- `assert.Len(t, collection, expectedLen)`
- `assert.ElementsMatch(t, expected, actual)`
- `assert.InEpsilon(t, expected, actual, epsilon)`
- `assert.JSONEq(t, expectedJSON, actualJSON)`
- `assert.FileExists(t, path)`
- `assert.NotNil(t, value)`
- `assert.Panics(t, panickingFunc)`

**Hard assertions** (`require.*`) — test stops on failure:
- `require.NoError(t, err)`
- `require.Error(t, err)`
- `require.ErrorIs(t, err, sentinelErr)`
- `require.NotNil(t, value)`
- `require.Equal(t, expected, got)`
- `require.Errorf(t, err, msg)`

---

*Testing analysis: 2026-07-21*

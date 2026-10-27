# httptest_mock

HTTP mock server framework for tests: declare request/response mocks in YAML, JSON, or Go code, serve them from an `httptest.Server`, and match requests by method, path, query parameters, path parameters, headers, and body.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/httptest_mock)

## Install

```bash
go get github.com/guionardo/go/httptest_mock
```

## Usage

Mocks can come from files or from code. Both end up in the same `MockHandler` and follow the same matching rules.

### Mocks from files

`WithRequestsFrom` loads every `.json`, `.yaml`, and `.yml` file in a directory (top level only, subdirectories are skipped) or explicit file paths.

`mocks/get_user.yaml`:

```yaml
name: get_user
request:
  method: GET
  path: /api/v1/users/123
response:
  status: 200
  body:
    id: 123
    name: "John Doe"
    email: "john@example.com"
  headers:
    Content-Type: "application/json"
```

Then, in your test:

```go
package mypackage_test

import (
    "io"
    "net/http"
    "testing"

    httptestmock "github.com/guionardo/go/httptest_mock"
    "github.com/stretchr/testify/require"
)

func TestGetUser(t *testing.T) {
    // Setup mock server with mocks from directory
    server, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("mocks"))
    defer assertFunc(t) // Verify mock assertions at the end

    // Make request to mock server
    response, err := http.Get(server.URL + "/api/v1/users/123")
    require.NoError(t, err)
    defer func() { _ = response.Body.Close() }()

    // Assert response
    require.Equal(t, http.StatusOK, response.StatusCode)

    body, err := io.ReadAll(response.Body)
    require.NoError(t, err)
    require.JSONEq(t, `{"id":123,"name":"John Doe","email":"john@example.com"}`, string(body))
}
```

### Mocks from code

`NewMock` builds the same structure with a fluent API; `FastServe` registers it on a server of its own, `WithRequests` registers it alongside other mocks.

```go
mock := httptestmock.NewMock(http.MethodGet, "/api/resource").
    WithQueryParam("page", "1").
    WithResponseStatus(http.StatusOK).
    WithResponseBody(map[string]string{"key": "value"})

server, assertFunc := mock.FastServe(t) // or SetupServer(t, httptestmock.WithRequests(mock))
defer assertFunc(t)
```

Use `WithCustomHandler` when the response has to be computed from the request: the handler reads matched values with `m.GetPathValue`, `m.GetQueryValue`, and `m.GetHeaderValue`.

## Features

- **Declarative mocks**: Define mocks in YAML or JSON files
- **Automatic cleanup**: Server closes automatically when the test ends
- **Request matching**: Match requests by method, path (including path parameters with `{param}` syntax), query parameters, headers, and body
- **Validation**: Built-in validation for mock definitions using struct tags
- **Flexible responses**: Support for JSON, string, and byte body responses with custom headers and status codes
- **Automatic JSON headers**: When `response.body` is an object/map, the library JSON-encodes it and sets `Content-Type: application/json` automatically (unless you override it)
- **Response delays**: Simulate network latency or processing time with configurable delays
- **Partial matching**: Accept requests even when not all parameters match (useful for flexible testing)
- **Request assertions**: Verify that mocks were called the expected number of times
- **Post-request hooks**: Modify responses or perform actions before sending responses
- **Debugging support**: Add the matched mock name to response headers for easier debugging
- **Dynamic mock management**: Add new mocks to running servers at runtime
- **Structured logging**: Optional slog.Logger integration for structured logging
- **Helper utilities**: Load mocks from files, retrieve handlers from servers, and more
- **Builder API**: Fluent helpers to build mocks in code (`NewMock`, `FastServe`) and create custom handlers
- **Captured request values**: Access matched `{path}` params, query params, and headers from inside custom handlers
- **Cross-platform header matching**: Header keys are normalized for comparison (lowercase, underscores→hyphens), so `API_KEY` matches `api-key` on all platforms

## Mock File Structure

### YAML Format

```yaml
name: descriptive_mock_name          # Optional: defaults to filename
request:
  method: GET                        # Required: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
  path: /api/v1/resource             # Required: URL path to match (supports {param} for path params)
  query_params:                      # Optional: query parameters to match (all must match)
    page: "1"
    limit: "10"
  path_params:                       # Optional: path parameters to match (use with {param} in path)
    id: "123"
  headers:                           # Optional: headers to match (case-insensitive, all must match)
    Authorization: "Bearer token"
    Content-Type: "application/json"
  body:                              # Optional: request body to match (can be string, object, or null)
    key: "value"
  partial_match: false               # Optional: accept request even if not all params match (default: false)
response:
  status: 200                        # Required: HTTP status code (100-599)
  body:                              # Optional: response body (object, string, or null)
    message: "Success"
  headers:                           # Optional: response headers
    Content-Type: "application/json"
    X-Custom-Header: "value"
  delay_ms: 100                      # Optional: delay in milliseconds before sending response
assertion: true                      # Optional: enable assertion checking (default: false)
expected_hits: 1                     # Optional: expected number of times this mock should be called
```

### JSON Format

```json
{
  "name": "descriptive_mock_name",
  "request": {
    "method": "POST",
    "path": "/api/v1/resource",
    "query_params": {
      "validate": "true"
    },
    "headers": {
      "Content-Type": "application/json"
    }
  },
  "response": {
    "status": 201,
    "body": {
      "id": 1,
      "created": true
    },
    "headers": {
      "Content-Type": "application/json",
      "Location": "/api/v1/resource/1"
    },
    "delay_ms": 100
  },
  "assertion": true,
  "expected_hits": 1
}
```

> **Note**: Both JSON and YAML use snake_case for field names (`query_params`, `path_params`, `delay_ms`, etc.).

## API Reference

### SetupServer

Creates and starts a new HTTP test server with the provided mock configurations.

```go
func SetupServer(t *testing.T, options ...func(*MockHandler)) (server *httptest.Server, assertFunc func(*testing.T))
```

The server automatically closes when the test context ends.

### Options

#### WithRequestsFrom

Loads all mock definitions from files and directories. Supports `.json`, `.yaml`, and `.yml` files.

```go
httptestmock.WithRequestsFrom("path/to/mocks","path/to/explicit_mock.json")
```

#### WithRequests

Provides mock definitions programmatically.

```go
httptestmock.WithRequests(&httptestmock.Mock{
    MockName: "health_check",
    Request:  httptestmock.Request{Method: "GET", Path: "/health"},
    Response: httptestmock.Response{Status: 200, Body: "OK"},
})
```

#### WithPostRequestHook

Adds a hook function that is called before sending the response. This allows you to modify the response or perform additional actions.

```go
httptestmock.WithPostRequestHook(func(mr httptestmock.Mocker, w http.ResponseWriter) {
    w.Header().Set("X-Custom-Header", "custom-value")
    // Log or perform other actions
})
```

#### WithAddMockInfoToResponse

Adds mock debugging information to response headers. By default, adds `HTTPTestMock-Name` (header names are case-insensitive). You can customize the header prefix:

```go
// Default header: HTTPTestMock-Name
httptestmock.WithAddMockInfoToResponse()

// Custom header: X-Mock-Name
httptestmock.WithAddMockInfoToResponse("X-Mock")
```

#### WithoutLog

Disables logging output from the mock handler. Useful when you want to suppress verbose test logs.

```go
httptestmock.WithoutLog()
```

#### WithExtraLogger

Allows setting an additional structured logger (slog.Logger) for the MockHandler. This provides structured logging alongside the default test logging.

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
httptestmock.WithExtraLogger(logger)
```

#### WithAcceptingPartialMatch

Enables handler-wide partial matching. With it, a request that matches `method` and `path` but misses another criterion becomes a partial match; a mock only accepts that partial match when it also sets `partial_match: true`, otherwise it is logged as a candidate. Disabled by default, so anything short of a full match is a non-match.

```go
httptestmock.WithAcceptingPartialMatch()
```

## Helper Functions

### GetMockHandlerFromServer

Retrieves the MockHandler from an httptest.Server instance. This is useful when you need to dynamically add more mocks to an existing server.

```go
server, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("mocks"))
defer assertFunc(t)
handler, err := httptestmock.GetMockHandlerFromServer(server)
if err != nil {
    t.Fatal(err)
}
```

### AddMocks

Dynamically adds new mock requests to an existing MockHandler. This allows you to modify the server behavior during test execution.

```go
server, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("mocks"))
defer assertFunc(t)
handler, _ := httptestmock.GetMockHandlerFromServer(server)

newMock := &httptestmock.Mock{
    MockName: "dynamic_mock",
    Request: httptestmock.Request{
        Method: "GET",
        Path:   "/dynamic",
    },
    Response: httptestmock.Response{
        Status: 200,
        Body:   "Added at runtime",
    },
}

err := handler.AddMocks(newMock)
if err != nil {
    t.Fatal(err)
}
```

### GetMocksFrom

Loads mock definitions from file paths or directories without creating a server. This is useful when you want to load and inspect mocks before server setup.

```go
mocks, err := httptestmock.GetMocksFrom("mocks", "custom_mock.yaml")
if err != nil {
    t.Fatal(err)
}
// Use mocks as needed
```

## Request Matching

Mocks are grouped by `method + path`; within a group they are tried in registration order and the first full match wins.

### Full Match

A request is considered a **full match** when:

1. **Method** matches exactly (case-sensitive: GET, POST, etc.)
2. **Path** matches exactly or all path parameters match
3. **Query parameters** (if specified in mock) all match
4. **Path parameters** (if specified in mock) all match
5. **Headers**: header names are normalized (lowercase, `_` → `-`) and compared case-insensitively; header values are compared case-sensitively
6. **Body** (if specified in mock) matches

### Partial Match

When a request matches method and path but not all other criteria, it's a **partial match**. Two switches must both be on for the mock to answer it: the handler-wide `WithAcceptingPartialMatch()` option and `partial_match: true` on the individual mock. This is useful for flexible testing scenarios.

If no full match is found, the server returns `404 Not Found`; candidate mocks and the criteria that failed are written to the test log.

**Note**: partial matching is off by default. With `WithAcceptingPartialMatch()` not enabled, a partial match is treated as no match and returns `404 Not Found`.

### Path Parameters

Path parameters are defined using curly braces in the path:

```yaml
request:
  path: /api/v1/users/{id}
  path_params:
    id: "123"
```

This will match `/api/v1/users/123` and extract `id=123` for validation against `path_params`.

### No Match Behavior

When a request doesn't match any mock, the server always returns `404 Not Found`. What varies is the logging: a partial match is logged with the criteria that failed and listed as a candidate, while a request that matches no method+path group is logged as unmatched.

## Response Body Types

The response body supports multiple types:

- **Object/Map**: Automatically encoded as JSON
- **String**: Written as-is (raw text)
- **Bytes**: Written as-is (binary data)
- **nil**: No body written (empty response)

Note: When using object/map bodies, `Content-Type: application/json` is set automatically. If you want a different content type, set it explicitly in `response.headers`:

```yaml
response:
  status: 200
  body:
    message: "Success"
  headers:
    Content-Type: "application/vnd.myapp+json"
```

## Request Body Matching

The request body can be matched in different ways:

- **String**: Exact string match
- **Bytes**: Exact byte match
- **Object/Map**: JSON comparison (order-independent)
- **nil**: No body expected

Example with JSON body matching:

```yaml
request:
  method: POST
  path: /api/v1/users
  body:
    name: "John Doe"
    email: "john@example.com"
```

## Examples

### Multiple Mocks

#### **mocks/list_users.yaml**

```yaml
name: list_users
request:
  method: GET
  path: /api/v1/users
  query_params:
    page: "1"
response:
  status: 200
  body:
    users: []
    total: 0
  headers:
    Content-Type: "application/json"
```

#### **mocks/create_user.yaml**

```yaml
name: create_user
request:
  method: POST
  path: /api/v1/users
response:
  status: 201
  body:
    id: 1
    message: "User created"
  headers:
    Content-Type: "application/json"
    Location: "/api/v1/users/1"
```

### Error Responses

```yaml
name: not_found
request:
  method: GET
  path: /api/v1/users/999
response:
  status: 404
  body:
    error: "User not found"
    code: "USER_NOT_FOUND"
  headers:
    Content-Type: "application/json"
```

### Path Parameters Example

```yaml
name: get_user_by_id
request:
  method: GET
  path: /api/v1/users/{userId}
  path_params:
    userId: "123"
response:
  status: 200
  body:
    id: 123
    name: "John Doe"
  headers:
    Content-Type: "application/json"
```

### Request Assertion Example

Use assertions to verify that a mock was called the expected number of times:

```yaml
name: health_check
request:
  method: GET
  path: /health
response:
  status: 200
  body: "OK"
assertion: true
expected_hits: 1
```

In your test:

```go
server, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("mocks"))
defer assertFunc(t) // This will fail the test if health_check wasn't called exactly once

// ... make your requests ...
```

### Response Delay Example

Simulate slow responses or network latency:

```yaml
name: slow_endpoint
request:
  method: GET
  path: /api/v1/slow
response:
  status: 200
  body: "This took a while"
  delay_ms: 2000  # 2 second delay
```

### Programmatic Mock Definition

Instead of using files, you can define mocks in code:

```go
server, assertFunc := httptestmock.SetupServer(t,
    httptestmock.WithRequests(&httptestmock.Mock{
        MockName: "health_check",
        Request: httptestmock.Request{
            Method: "GET",
            Path:   "/health",
        },
        Response: httptestmock.Response{
            Status: 200,
            Body:   map[string]string{"status": "ok"},
            Headers: map[string]string{
                "Content-Type": "application/json",
            },
        },
    }),
)
defer assertFunc(t)
```

### Builder API (NewMock / FastServe) and Custom Handlers

If you prefer defining mocks in code (instead of files), you can use the fluent builder:

```go
mock := httptestmock.NewMock(http.MethodPost, "/example/{id}").
    WithQueryParam("key", "value").
    WithPathParam("id", "123").
    WithHeader("Authorization", "Bearer token").
    WithBody(map[string]string{"field": "data"}).
    WithResponseStatus(http.StatusOK).
    WithResponseBody(map[string]string{"response": "success"}).
    WithCustomHandler(func(m httptestmock.Mocker, w http.ResponseWriter, r *http.Request) {
        // Access captured values from the matched request:
        id := m.GetPathValue("id")
        key := m.GetQueryValue("key")
        token := m.GetHeaderValue("Authorization")

        _ = id
        _ = key
        _ = token

        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("custom handler response"))
    })

server, assertFunc := mock.FastServe(t)
defer assertFunc(t)
```

### Dynamic Mock Management

Add new mocks to a running server dynamically:

```go
func TestDynamicMockManagement(t *testing.T) {
    // Start server with initial mocks
    server, assertFunc := httptestmock.SetupServer(t,
        httptestmock.WithRequestsFrom("mocks"))
    defer assertFunc(t)

    // Get the handler from the server
    handler, err := httptestmock.GetMockHandlerFromServer(server)
    require.NoError(t, err)

    // Make initial request
    resp, err := http.Get(server.URL + "/api/v1/users")
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
    resp.Body.Close()

    // Add a new mock at runtime
    newMock := &httptestmock.Mock{
        MockName: "dynamic_endpoint",
        Request: httptestmock.Request{
            Method: "POST",
            Path:   "/api/v1/users",
        },
        Response: httptestmock.Response{
            Status: 201,
            Body:   map[string]any{"id": 123, "created": true},
            Headers: map[string]string{
                "Content-Type": "application/json",
            },
        },
    }

    err = handler.AddMocks(newMock)
    require.NoError(t, err)

    // Now the new endpoint is available
    resp, err = http.Post(server.URL+"/api/v1/users", "application/json", nil)
    require.NoError(t, err)
    require.Equal(t, http.StatusCreated, resp.StatusCode)
    resp.Body.Close()
}
```

### Partial vs Strict Matching

Strict matching is the default: a mock answers only when every criterion in its `request` block matches.

Lenient matching takes both switches — `WithAcceptingPartialMatch()` on the server and `partial_match: true` on the individual mock. Only then does a request that matches method and path but misses a query param, header, or body still receive the mock's response.

```go
func TestPartialMatch(t *testing.T) {
    server, assertFunc := httptestmock.SetupServer(t,
        httptestmock.WithRequestsFrom("mocks"),
        httptestmock.WithAcceptingPartialMatch())
    defer assertFunc(t)

    // /api/v1/users/123 matches method+path. A missing header makes it a
    // partial match, accepted only by mocks that set partial_match: true.
    resp, err := http.Get(server.URL + "/api/v1/users/123")
    require.NoError(t, err)
    defer func() { _ = resp.Body.Close() }()
}
```

## License

MIT

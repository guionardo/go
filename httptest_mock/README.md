# httptestmock

A Go library for creating HTTP mock servers in tests using declarative JSON/YAML mock definitions.

## Overview

`httptestmock` simplifies HTTP integration testing by allowing you to define request/response mocks in external files. Instead of writing verbose mock handlers in Go code, you declare your expected requests and responses in human-readable YAML or JSON files.

## Features

- **Declarative mocks**: Define mocks in YAML or JSON files
- **Automatic cleanup**: Server closes automatically when the test ends
- **Request matching**: Match requests by method, path, query parameters, and headers
- **Validation**: Built-in validation for mock definitions
- **Flexible responses**: Support for JSON, string, and byte body responses

## Installation

```bash
go get github.com/guionardo/go/httptest_mock@latest
```

## Quick Start

### 1. Create a mock file

Create a directory for your mocks (e.g., `mocks/`) and add a YAML or JSON file:

#### **mocks/get_user.yaml**

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

### 2. Use in your test

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
    server := httptestmock.SetupServer(t, httptestmock.WithRequestsFromDir("mocks"))

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

## Mock File Structure

### YAML Format

```yaml
name: descriptive_mock_name          # Optional: defaults to filename
request:
  method: GET                        # Required: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
  path: /api/v1/resource             # Required: URL path to match
  queryParams:                       # Optional: query parameters to match
    page: "1"
    limit: "10"
  pathParams:                        # Optional: path parameters to match
    id: 10
  headers:                           # Optional: headers to match
    Authorization: "Bearer token"
  body: null                         # Optional: request body
  partial_match: true                # Optional: if a full match is not found, can accept missing params
response:
  status: 200                        # Required: HTTP status code (100-599)
  body:                              # Optional: response body (object, string, or null)
    message: "Success"
  headers:                           # Optional: response headers
    Content-Type: "application/json"
    X-Custom-Header: "value"
  delay_ms: 100                      # Optional: delay before response to emulate timeout/process time
assertion: true                      # Optional: check if the number of hits matches the expected_hits
expected_hits: 10                    # Must call the func assertFunc returned by the SetupServer func
```

### JSON Format

```json
{
  "name": "descriptive_mock_name",
  "request": {
    "method": "POST",
    "path": "/api/v1/resource",
    "queryParams": {
      "validate": "true"
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
  }
}
```

## API Reference

### SetupServer

Creates and starts a new HTTP test server with the provided mock configurations.

```go
func SetupServer(t *testing.T, options ...func(*server)) (server *httptest.Server, assertFunc func(*testing.T))
```

The server automatically closes when the test context ends.

### Options

#### WithRequestsFromDir

Loads all mock definitions from a directory. Supports `.json`, `.yaml`, and `.yml` files.

```go
httptestmock.WithRequestsFromDir("path/to/mocks")
```

#### WithRequests

Provides mock definitions programmatically.

```go
httptestmock.WithRequests([]*httptestmock.MockRequest{
    {
        Name: "custom_mock",
        Request: httptestmock.Request{
            Method: "GET",
            Path:   "/api/health",
        },
        Response: httptestmock.Response{
            Status: 200,
            Body:   map[string]string{"status": "ok"},
        },
    },
})
```

#### WithPostRequestHook

Adds a hook to modify the response before sending it

#### WithAddMockInfoToResponse

Adds mock information to response headers (mock name and source file)

#### WithoutLog

Disable logging for the mock handler

## Request Matching

Requests are matched in the order they are defined. The first matching mock wins. A request matches when:

1. **Method** matches exactly (case-sensitive)
2. **Path** matches exactly
3. **Query parameters** (if specified) all match
4. **Headers** matches (case-insensitive)

## Response Body Types

The response body supports multiple types:

- **Object/Map**: Encoded as JSON
- **String**: Written as-is
- **Bytes**: Written as-is
- **nil**: No body written

## Examples

### Multiple Mocks

#### **mocks/list_users.yaml**

```yaml
name: list_users
request:
  method: GET
  path: /api/v1/users
  queryParams:
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

## License

MIT

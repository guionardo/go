package httptestmock

import (
	"net/http/httptest"
	"testing"
)

// NewMock creates a new Mock with the specified HTTP method and path.
// You should further configure the mock using the provided builder methods.
func NewMock(method string, path string) *Mock {
	return &Mock{
		Request: Request{
			Method:      method,
			Path:        path,
			QueryParams: make(map[string]string),
			PathParams:  make(map[string]string),
			Headers:     make(map[string]string),
			readenData:  make(map[string]string),
		},
		Response: Response{
			Headers: make(map[string]string),
		},
	}
}

// WithQueryParam adds a query parameter to the mock's request definition.
func (m *Mock) WithQueryParam(key, value string) *Mock {
	m.Request.QueryParams[key] = value
	return m
}

// WithPathParam adds a path parameter to the mock's request definition.
func (m *Mock) WithPathParam(key, value string) *Mock {
	m.Request.PathParams[key] = value
	return m
}

// WithHeader adds a header to the mock's request definition.
func (m *Mock) WithHeader(key, value string) *Mock {
	m.Request.Headers[key] = value
	return m
}

// WithBody sets the body of the mock's request definition.
func (m *Mock) WithBody(body any) *Mock {
	m.Request.Body = body
	return m
}

// WithResponseStatus sets the HTTP status code of the mock's response definition.
func (m *Mock) WithResponseStatus(status int) *Mock {
	m.Response.Status = status
	return m
}

// WithResponseBody sets the body of the mock's response definition.
func (m *Mock) WithResponseBody(body any) *Mock {
	m.Response.Body = body
	return m
}

// WithResponseHeader adds a header to the mock's response definition.
func (m *Mock) WithResponseHeader(key, value string) *Mock {
	m.Response.Headers[key] = value
	return m
}

// WithAssertion configures assertion settings for the mock.
func (m *Mock) WithAssertion(enabled bool, expectedHits uint) *Mock {
	m.AssertionEnabled = enabled
	m.ExpectedHits = expectedHits

	return m
}

// WithCustomHandler sets a custom HTTP handler function for the mock.
func (m *Mock) WithCustomHandler(handler CustomHandlerFunc) *Mock {
	m.customHandler = handler
	return m
}

// FastServe is a convenience method to quickly start a mock server with this single mock.
// It accepts additional configuration options for the server.
//
// Example:
//
// mock:= httptestmock.NewMock("GET", "/hello").
//
//	WithResponseStatus(200).
//	WithResponseBody("Hello, World!")
//
// server,assert:=mock.FastServe(t)
//
//	defer assert(t)
func (m *Mock) FastServe(
	t *testing.T,
	options ...func(*MockHandler),
) (server *httptest.Server, assert func(*testing.T)) {
	return SetupServer(t, append(options, WithRequests(m))...)
}

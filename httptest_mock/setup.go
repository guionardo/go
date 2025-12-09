package httptestmock

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const defaultLogHeader = "HTTPTestMock"

// SetupServer creates and starts a new HTTP test server with the provided mock configurations.
// The server automatically closes when the test context ends, so no manual cleanup is required.
//
// Example:
//
//	server := httptestmock.SetupServer(t, httptestmock.WithRequestsFromDir("mocks"))
//	response, err := http.Get(server.URL + "/api/v1/example")
//	require.NoError(t, err)
//	defer func() { _ = response.Body.Close() }()
//	require.Equal(t, http.StatusOK, response.StatusCode)
//
// Available options:
//   - WithRequests: Provide mock definitions programmatically
//   - WithRequestsFromDir: Load mock definitions from a directory of JSON/YAML files
//   - WithPostRequestHook: Add a hook to modify the response before sending it
//   - WithAddMockInfoToResponse: Add mock information to response headers
//   - WithoutLog: Disable logging for the mock handler
//
// The function will call t.Fatalf if server validation fails (no mocks or invalid mock definitions).
func SetupServer(t *testing.T, options ...func(*MockHandler)) (server *httptest.Server, assertFunc func(*testing.T)) {
	mockHandler := &MockHandler{
		T:         t,
		logHeader: defaultLogHeader}
	for _, option := range options {
		option(mockHandler)
	}

	if mockHandler.setupError != nil {
		t.Fatalf("failed to setup mock server: %v", mockHandler.setupError) // nocover
	}

	if err := mockHandler.Validate(); err != nil {
		t.Fatalf("failed to validate server: %v", err) // nocover
	}

	mockServer := httptest.NewServer(mockHandler)

	t.Logf("%s server started", mockHandler.logHeader)

	// Start cleanup goroutine that closes the server when test ends
	go func() {
		<-t.Context().Done()
		mockServer.Close()
	}()

	return mockServer, mockHandler.Assert
}

// WithRequests configures the server with programmatically defined mock requests.
// Use this option when you need to create mocks dynamically in code.
//
// Example:
//
//	server := httptestmock.SetupServer(t, httptestmock.WithRequests(
//	    &httptestmock.MockRequest{
//	        Name: "health_check",
//	        Request:  httptestmock.Request{Method: "GET", Path: "/health"},
//	        Response: httptestmock.Response{Status: 200, Body: "OK"},
//	    },
//	))
func WithRequests(requests ...*MockRequest) func(*MockHandler) {
	return func(s *MockHandler) {
		s.requests = requests
		for _, req := range requests {
			s.log("%s registered %s", s.logHeader, req.String())
		}
	}
}

// WithRequestsFromDir configures the server with mock requests loaded from a directory.
// All files with .json, .yaml, or .yml extensions in the directory will be parsed as mock definitions.
// Subdirectories are not traversed.
//
// Example:
//
//	server := httptestmock.SetupServer(t, httptestmock.WithRequestsFromDir("testdata/mocks"))
//
// The function will call t.Fatalf if the directory cannot be read or if any mock file is invalid.
func WithRequestsFromDir(dir string) func(*MockHandler) {
	return func(s *MockHandler) {
		requests, err := readMocks(dir)
		if err != nil {
			s.setupError = errors.Join(s.setupError, fmt.Errorf("failed to read mocks from dir: %w", err))
			return
		}

		WithRequests(requests...)(s)
	}
}

// WithPostRequestHook adds a hook that will be called before sending the response.
// This can be used to modify the response or perform additional actions before sending it.
// The hook receives the matched MockRequest and the http.ResponseWriter to modify the response.
// This is useful for adding custom headers, logging, or other pre-response logic.
//
// Example:
//
//	httptestmock.WithPostRequestHook(func(mr *httptestmock.MockRequest, w http.ResponseWriter) {
//	    w.Header().Set("X-Custom-Header", "value")
//	})
func WithPostRequestHook(hook func(*MockRequest, http.ResponseWriter)) func(*MockHandler) {
	return func(s *MockHandler) {
		s.preResponseHooks = append(s.preResponseHooks, hook)
	}
}

// WithAddMockInfoToResponse adds mock information to the response headers.
// This is useful for debugging and tracking which mock was used for the response.
// The headers will include the mock name and path.
// You can customize the header prefix by passing a string argument.
//
// Example:
//
//	httptestmock.WithAddMockInfoToResponse("MyMock")
//
// The default prefix is "HTTPTestMock-".
func WithAddMockInfoToResponse(headerPrefix ...string) func(*MockHandler) {
	prefix := defaultLogHeader
	if len(headerPrefix) > 0 && len(headerPrefix[0]) > 0 {
		prefix = headerPrefix[0]
	}

	prefix = strings.Trim(prefix, "-_.")

	return WithPostRequestHook(func(mr *MockRequest, w http.ResponseWriter) {
		// Add mock information to the response
		w.Header().Set(prefix+"-Name", mr.Name)
		w.Header().Set(prefix+"-Path", mr.Request.Path)
	})
}

// WithoutLog disables logging for the mock handler.
// This is useful for tests where you want to suppress log output.
// By default, the mock handler logs request matching details.
func WithoutLog() func(*MockHandler) {
	return func(s *MockHandler) {
		s.logDisabled = true
	}
}

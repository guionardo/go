package httptestmock

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert/yaml"
)

const defaultLogHeader = "HTTPTestMock"

// SetupServer creates and starts a new HTTP test server with the provided mock configurations.
// The server automatically closes when the test context ends, so no manual cleanup is required.
//
// Example:
//
//	server := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("mocks"))
//	response, err := http.Get(server.URL + "/api/v1/example")
//	require.NoError(t, err)
//	defer func() { _ = response.Body.Close() }()
//	require.Equal(t, http.StatusOK, response.StatusCode)
//
// Available options:
//   - WithRequests: Provide mock definitions programmatically
//   - WithRequestsFrom: Load mock definitions from a directory of JSON/YAML files or specific files
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
//	    &httptestmock.Mock{
//	        Name: "health_check",
//	        Request:  httptestmock.Request{Method: "GET", Path: "/health"},
//	        Response: httptestmock.Response{Status: 200, Body: "OK"},
//	    },
//	))
func WithRequests(requests ...*Mock) func(*MockHandler) {
	return func(s *MockHandler) {
		s.requests = requests
		for _, req := range requests {
			s.log("%s registered %s", s.logHeader, req.String())
		}
	}
}

// WithRequestsFrom configures the server with mock requests loaded from specified files or folders.
// Each path can be a file (JSON/YAML) or a directory containing mock definitions.
// Path can be a mix of files and directories and contain patterns.
// If a directory is provided, all valid mock files within it will be loaded.
// Subdirectories are not traversed.
//
// Example:
//
//	server := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("testdata/mocks"))
func WithRequestsFrom(paths ...string) func(*MockHandler) {
	return func(s *MockHandler) {
		var requests []*Mock

		for _, p := range paths {
			mocks, err := readMocksFromPath(p)
			if err != nil {
				s.setupError = errors.Join(s.setupError, fmt.Errorf("failed to read mocks from path %q: %w", p, err))
				continue
			}

			requests = append(requests, mocks...)
		}

		WithRequests(requests...)(s)
	}
}

// WithPostRequestHook adds a hook that will be called before sending the response.
// This can be used to modify the response or perform additional actions before sending it.
// The hook receives the matched Mock and the http.ResponseWriter to modify the response.
// This is useful for adding custom headers, logging, or other pre-response logic.
//
// Example:
//
//	httptestmock.WithPostRequestHook(func(mr *httptestmock.Mock, w http.ResponseWriter) {
//	    w.Header().Set("X-Custom-Header", "value")
//	})
func WithPostRequestHook(hook func(*Mock, http.ResponseWriter)) func(*MockHandler) {
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

	return WithPostRequestHook(func(mr *Mock, w http.ResponseWriter) {
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

func readMocksFromPath(sourcePath string) (requests []*Mock, err error) {
	matches, err := filepath.Glob(sourcePath)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		stat, err := os.Stat(filepath.Clean(match))
		if err != nil {
			continue
		}

		if stat.IsDir() {
			dirRequests, err := readMocks(match)
			if err == nil && len(dirRequests) > 0 {
				requests = append(requests, dirRequests...)
			}

			continue
		}

		if mock, err := readMock(match); err == nil {
			requests = append(requests, mock)
		}
	}

	return requests, nil
}

// readMocks reads all mock definitions from a directory.
// Processes files with .json, .yaml, or .yml extensions.
// Subdirectories are skipped.
func readMocks(dir string) ([]*Mock, error) {
	files, err := os.ReadDir(filepath.Clean(dir))
	if err != nil {
		return nil, err
	}

	requests := make([]*Mock, 0, len(files))
	for _, file := range files {
		ext := strings.ToLower(path.Ext(file.Name()))
		// Skip directories and non-mock files
		if file.IsDir() || (ext != ".json" && ext != ".yaml" && ext != ".yml") {
			continue
		}

		mock, err := readMock(path.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}

		requests = append(requests, mock)
	}

	return requests, nil
}

// readMock reads and parses a mock definition from a JSON or YAML file.
// It first attempts JSON parsing, then falls back to YAML if JSON fails.
func readMock(path string) (*Mock, error) {
	file, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	mock, err := unmarshalMock(file)
	if err == nil {
		// Use filename as mock name if not specified
		if mock.Name == "" {
			mock.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}

		mock.source = path
	}

	return mock, err
}

// unmarshalMock unmarshals mock data from JSON or YAML format.
func unmarshalMock(data []byte) (request *Mock, err error) {
	if len(data) == 0 {
		return nil, errors.New("empty mock data")
	}

	var mock Mock
	if data[0] == '{' {
		err = json.Unmarshal(data, &mock)
	} else {
		err = yaml.Unmarshal(data, &mock)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json/yaml: %w", err)
	}

	return &mock, nil
}

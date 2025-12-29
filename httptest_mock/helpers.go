package httptestmock

import (
	"errors"
	"net/http/httptest"
)

// GetMockHandlerFromServer retrieves the MockHandler from the given http.Server.
func GetMockHandlerFromServer(server *httptest.Server) (*MockHandler, error) {
	if server == nil {
		return nil, errors.New("server is nil")
	}

	mockHandler, ok := server.Config.Handler.(*MockHandler)
	if !ok {
		return nil, errors.New("handler is not of type MockHandler")
	}

	return mockHandler, nil
}

// GetMocksFrom loads mock definitions from the provided file paths or directories.
// It returns a slice of Mock pointers and any error encountered during loading.
// Each path can be a file (JSON/YAML) or a directory containing mock definitions.
// Errors from multiple paths are aggregated using errors.Join.
func GetMocksFrom(paths ...string) (requests []Mocker, err error) {
	for _, p := range paths {
		mocks, readErr := readMocksFromPath(p)
		if readErr != nil {
			err = errors.Join(err, readErr)
		}

		requests = append(requests, mocks...)
	}

	return requests, err
}

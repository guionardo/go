package httptestmock

import (
	"net/http"
	"testing"
)

type (
	// Mocker defines the interface for types that can match HTTP requests and respond accordingly.
	Mocker interface {
		// Matches checks if the given HTTP request matches the mock's criteria.
		// It returns the level of match: none, partial, or full.
		// If the allowPartialMatch parameter is true, at least path with parameters and method must match.
		Matches(r *http.Request, allowPartialMatch bool) RequestMatchLevel

		Name() string

		// String returns a human-readable representation of the mock for logging.
		String() string

		// WriteResponse writes the mock's response to the given http.ResponseWriter.
		WriteResponse(r *http.Request, w http.ResponseWriter)

		// RegisterHit records that the mock has been hit, to enable assertions.
		RegisterHit(t *testing.T)

		// AcceptsPartialMatch, when true, indicates that the mock will consider at least path with parameters and
		// method
		// matches as valid.
		AcceptsPartialMatch() bool

		// Validate checks if the mock's configuration is valid.
		Validate() error

		// Assert checks if the mock was hit the expected number of times during the test.
		Assert(t *testing.T)

		// AppendLog appends a log message to the mock's internal log.
		AppendLog(log string)

		// Logs returns all log messages associated with the mock.
		Logs() []string

		// GetPathValue returns the value of the captured path parameter identified by key
		// from the matched request path. It returns an empty string if the key is not present.
		GetPathValue(key string) string

		// GetQueryValue returns the value of the query parameter identified by key from the
		// matched request URL. It returns an empty string if the key is not present.
		GetQueryValue(key string) string

		// GetHeaderValue returns the value of the request header identified by key from the
		// matched request. It returns an empty string if the header is not present.
		GetHeaderValue(key string) string
	}

	// CustomHandlerFunc defines the signature for a custom HTTP handler function for a mock.
	CustomHandlerFunc func(Mocker, http.ResponseWriter, *http.Request)
)

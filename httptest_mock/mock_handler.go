package httptestmock

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// MockHandler is the internal HTTP handler that manages mock requests.
// It implements http.Handler to serve as the handler for httptest.MockHandler.
type MockHandler struct {
	// requests holds all registered mock definitions to match against incoming requests.
	requests []*Mock

	// T is the testing context, used for logging and cleanup.
	T *testing.T

	// logHeader is the prefix used for all log messages from this server.
	logHeader string

	// preResponseHook is called before a response is sent.
	preResponseHooks []func(*Mock, http.ResponseWriter)

	// logDisabled indicates whether logging is enabled for this handler.
	logDisabled bool

	// setupError is set if there was an error during setup.
	// This is used to fail the test if the setup fails.
	// It should be checked after calling SetupServer.
	setupError error
}

// ServeHTTP implements the http.Handler interface.
// It iterates through registered mocks and returns the response for the first match.
// If no mock matches, the request receives no response (empty 200).
func (s *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	partialMatchRequests := make([]*Mock, 0)

	for _, request := range s.requests {
		switch request.Request.match(r) {
		case matchLevelFull:
			s.log("%s request matched %s", s.logHeader, request.String())
			s.DoPreResponseHook(request, w)
			request.Response.writeResponse(w)
			request.RegisterHit(s.T)

			return

		case matchLevelPartial:
			if request.Request.PartialMatch {
				s.log("%s request partially matched %s", s.logHeader, request.String())
				s.DoPreResponseHook(request, w)
				request.Response.writeResponse(w)
				request.RegisterHit(s.T)

				return
			}

			partialMatchRequests = append(partialMatchRequests, request)
			// the request did not match, let's continue to the next one
			s.log("%s request did not match %s:\n%s", s.logHeader,
				request.String(), strings.Join(request.Request.matchLog, "\n"))
		}
	}

	if len(partialMatchRequests) > 0 {
		s.log("Mocks candidates for request %s %s", r.Method, r.URL.String())

		for _, req := range partialMatchRequests {
			s.log("%s partial match details: %s", s.logHeader, req.String())
		}

		w.WriteHeader(http.StatusBadRequest)

		return
	}

	s.log("%s request not matched %s", s.logHeader, r.URL.String())
	w.WriteHeader(http.StatusNotFound)
}

// Validate ensures the server has valid configuration before starting.
// Returns an error if no mocks are registered or if any mock fails validation.
func (s *MockHandler) Validate() error {
	if len(s.requests) == 0 {
		return errors.New("no requests found")
	}

	// Collect all validation errors to report them together
	reqValidateErrors := make([]error, 0, len(s.requests))
	for _, request := range s.requests {
		if err := request.Validate(); err != nil {
			reqValidateErrors = append(reqValidateErrors, err)
		}
	}

	if len(reqValidateErrors) > 0 {
		return fmt.Errorf("%s invalid requests: %w", s.logHeader, errors.Join(reqValidateErrors...))
	}

	return nil
}

func (s *MockHandler) DoPreResponseHook(m *Mock, r http.ResponseWriter) {
	for _, hook := range s.preResponseHooks {
		hook(m, r)
	}
}

// Assert checks if all registered requests were hit during the test.
// It will fail the test if any request was not hit.
// This is useful to ensure all mocks were used as expected.
// Call this at the end of your test to verify all mocks were hit.
// Example usage:
//
//	mockHandler, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequestsFromDir("testdata/mocks"))
//	defer assertFunc(t)
func (s *MockHandler) Assert(t *testing.T) {
	for _, request := range s.requests {
		request.Assert(t)
	}
}

func (s *MockHandler) log(format string, args ...any) {
	if s.logDisabled {
		return
	}

	s.T.Logf("%s "+format, append([]any{s.logHeader}, args...)...)
}

package httptestmock

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// MockHandler is the internal HTTP handler that manages mock requests.
// It implements http.Handler to serve as the handler for httptest.Server.
type (
	MockHandler struct {
		// mocks holds all registered mock definitions to match against incoming mocks.
		mocks []Mocker

		// T is the testing context, used for logging and cleanup.
		T *testing.T

		// logHeader is the prefix used for all log messages from this server.
		logHeader string

		// preResponseHook is called before a response is sent.
		preResponseHooks []func(Mocker, http.ResponseWriter)

		// logDisabled indicates whether logging is enabled for this handler.
		logDisabled bool

		// setupError is set if there was an error during setup.
		// This is used to fail the test if the setup fails.
		// It should be checked after calling SetupServer.
		setupError error

		// mu protects concurrent access to the handler's requests.
		mu sync.RWMutex

		// extraLogger is an optional additional logger for more detailed logs.
		extraLogger *slog.Logger

		// disablePartialMatch indicates whether partial matching is disabled.
		disablePartialMatch bool

		// server is the httptest.Server instance that is used to serve the requests.
		server *httptest.Server
	}
)

// ServeHTTP implements the http.Handler interface.
// It iterates through registered mocks and returns the response for the first match.
// If no mock matches, the handler returns 404 Not Found.
// If there are partial matches, the handler returns 400 Bad Request.
func (s *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	partialMatchRequests := make([]Mocker, 0)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, mock := range s.mocks {
		switch mock.Matches(r, s.disablePartialMatch) {
		case MatchLevelFull:
			s.log("%s request matched %s", s.logHeader, mock.String())
			s.extraLogger.Info(s.logHeader+" matched", slog.String("mock", mock.String()))
			s.DoPreResponseHook(mock, w)
			mock.WriteResponse(r, w)
			mock.RegisterHit(s.T)

			return

		case MatchLevelPartial:
			if mock.AcceptsPartialMatch() {
				s.log("%s request partially matched %s", s.logHeader, mock.String())
				s.extraLogger.Info(s.logHeader+" partially matched", slog.String("mock", mock.String()))
				s.DoPreResponseHook(mock, w)
				mock.WriteResponse(r, w)
				mock.RegisterHit(s.T)

				return
			}

			partialMatchRequests = append(partialMatchRequests, mock)
			// the request did not match, let's continue to the next one
			s.log("%s request did not match %s:\n%s", s.logHeader,
				mock.String(), strings.Join(mock.Logs(), "\n"))
			s.extraLogger.Warn(s.logHeader+" request did not match",
				slog.String("request", mock.String()),
				slog.String("log", strings.Join(mock.Logs(), "\n")))
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
	if len(s.mocks) == 0 {
		return errors.New("no requests found")
	}

	// Collect all validation errors to report them together
	reqValidateErrors := make([]error, 0, len(s.mocks))
	for _, mock := range s.mocks {
		if err := mock.Validate(); err != nil {
			reqValidateErrors = append(reqValidateErrors, err)
		}
	}

	if len(reqValidateErrors) > 0 {
		return fmt.Errorf("%s invalid requests: %w", s.logHeader, errors.Join(reqValidateErrors...))
	}

	return nil
}

func (s *MockHandler) DoPreResponseHook(m Mocker, r http.ResponseWriter) {
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
//	mockHandler, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequestsFrom("testdata/mocks"))
//	defer assertFunc(t)
func (s *MockHandler) Assert(t *testing.T) {
	for _, mock := range s.mocks {
		mock.Assert(t)
	}
}

// AddMocks appends new mock requests to the existing ones in the handler.
func (s *MockHandler) AddMocks(mocks ...Mocker) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mocks = append(s.mocks, mocks...)

	for _, req := range mocks {
		s.log("%s registered %s", s.logHeader, req.String())
		s.extraLogger.Info(s.logHeader+" registered", slog.String("mock", req.String()))
	}

	return s.Validate()
}

func (s *MockHandler) log(format string, args ...any) {
	if s.logDisabled {
		return
	}

	s.T.Logf("%s "+format, append([]any{s.logHeader}, args...)...)
}

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

		// mu protects concurrent access to the handler's requests and mux.
		mu sync.RWMutex

		// mux is a ServeMux that routes requests to mock groups by method+path pattern.
		// Rebuilt by rebuildMux whenever mocks change. Thread-safe via mu.
		mux *http.ServeMux

		// extraLogger is an optional additional logger for more detailed logs.
		extraLogger *slog.Logger

		// allowPartialMatch indicates whether partial matching is enabled.
		allowPartialMatch bool

		// server is the httptest.Server instance that is used to serve the requests.
		server *httptest.Server
	}
)

// ServeHTTP implements the http.Handler interface.
// It uses a ServeMux to route requests by method+path pattern, then
// iterates through the matching mock group to find a full or partial match.
// If no mock matches, the handler returns 404 Not Found.
func (s *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			s.log("%s PANIC in handler: %v", s.logHeader, rec)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	s.ensureMux()

	s.mu.RLock()
	defer s.mu.RUnlock()
	s.mux.ServeHTTP(w, r)
}

// ensureMux lazily initialises the ServeMux under a write lock if nil.
// Thread-safe; call before reading s.mux under read lock.
func (s *MockHandler) ensureMux() {
	s.mu.RLock()
	if s.mux != nil {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	s.mu.Lock()
	if s.mux == nil {
		s.rebuildMux()
	}
	s.mu.Unlock()
}

// rebuildMux groups mocks by "METHOD /path" pattern and creates a new ServeMux.
// Each group is registered as a ServeMux handler that iterates only mocks
// with matching method and path. Non-*Mock Mocker implementations are
// grouped into a fallback handler that matches all paths.
// Caller MUST hold a write lock on s.mu.
func (s *MockHandler) rebuildMux() {
	groups := make(map[string][]Mocker)
	var fallback []Mocker

	for _, mock := range s.mocks {
		if m, ok := mock.(*Mock); ok {
			pattern := m.Request.Method + " " + m.Request.Path
			groups[pattern] = append(groups[pattern], mock)
		} else {
			fallback = append(fallback, mock)
		}
	}

	newMux := http.NewServeMux()
	for pattern, mocks := range groups {
		mocks := mocks
		newMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			s.handleMockGroup(mocks, w, r)
		})
	}

	if len(fallback) > 0 {
		newMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			s.handleMockGroup(fallback, w, r)
		})
	}

	s.mux = newMux
}

// handleMockGroup iterates a slice of mocks (all sharing the same method+path)
// and returns the first full match, first accepting partial match, or 404.
// Collects partial match candidates for diagnostic logging when no match is found.
func (s *MockHandler) handleMockGroup(mocks []Mocker, w http.ResponseWriter, r *http.Request) {
	partialMatchRequests := make([]Mocker, 0)

	for _, mock := range mocks {
		switch mock.Matches(r, s.allowPartialMatch) {
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
	} else {
		s.log("%s request not matched %s", s.logHeader, r.URL.String())
	}
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

// AddMocks appends new mock requests to the existing ones in the handler
// and rebuilds the ServeMux to include the new mocks.
func (s *MockHandler) AddMocks(mocks ...Mocker) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mocks = append(s.mocks, mocks...)
	s.rebuildMux()

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

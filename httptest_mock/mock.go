// Package httptestmock provides utilities for creating HTTP mock servers in Go tests.
// It allows defining request/response mocks in external JSON or YAML files for cleaner,
// more maintainable integration tests.
package httptestmock

import (
	"net/http"
	"sync"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type (
	// Mock represents a complete mock definition containing both
	// the expected request to match and the response to return.
	Mock struct {
		// MockName is an optional identifier for the mock, used in logging.
		// If not specified, defaults to the file path.
		MockName string `json:"name" yaml:"name"`

		// Request defines the criteria for matching incoming HTTP requests.
		Request Request `json:"request" yaml:"request" validate:"required"`

		// Response defines the HTTP response to return when a request matches.
		Response Response `json:"response" yaml:"response" validate:"required"`

		source string

		AssertionEnabled bool `json:"assertion" yaml:"assertion"`

		// Expected is the expected number of times this mock should be hit.
		ExpectedHits uint `json:"expected_hits" yaml:"expected_hits"`

		assertionActual map[string]uint
		assertionLock   sync.Mutex
		customHandler   CustomHandlerFunc
	}

	RequestMatchLevel uint8
)

const (
	// MatchLevelNone indicates no match.
	MatchLevelNone RequestMatchLevel = iota
	// MatchLevelPartial indicates a partial match.
	MatchLevelPartial
	// MatchLevelFull indicates a full match.
	MatchLevelFull

	// readDataPrefixes are used to store read data from the request.
	readDataPathParamPrefix  = "__path_param__"
	readDataQueryParamPrefix = "__query_param__"
	readDataHeaderPrefix     = "__header__"
)

var (
	// validate is the validator instance used to validate mock definitions.
	validate = validator.New(validator.WithRequiredStructEnabled())

	_ Mocker = (*Mock)(nil)
)

// String returns a human-readable representation of the mock for logging.
func (m *Mock) String() string {
	sp := StringParts{}.Set("name", m.MockName).
		Set("from", m.source).
		Set("req", m.Request.String()).
		Set("resp", m.Response.String())

	return "Mock: " + sp.String()
}

// Validate validates the mock definition using struct validation tags.
// Returns an error if required fields are missing or have invalid values.
func (m *Mock) Validate() error {
	m.Request.readData = make(map[string]string)
	return validate.Struct(m)
}

// RegisterHit records a hit for this mock request during the test.
func (m *Mock) RegisterHit(t *testing.T) {
	if !m.AssertionEnabled {
		return
	}

	m.assertionLock.Lock()
	defer m.assertionLock.Unlock()

	if m.assertionActual == nil {
		m.assertionActual = make(map[string]uint)
	}

	m.assertionActual[t.Name()]++
}

// Assert checks if the mock request was hit the expected number of times during the test.
func (m *Mock) Assert(t *testing.T) {
	if !m.AssertionEnabled {
		return
	}

	m.assertionLock.Lock()
	defer m.assertionLock.Unlock()

	if m.assertionActual == nil {
		m.assertionActual = make(map[string]uint)
	}

	count := m.assertionActual[t.Name()]
	assert.Equalf(t, m.ExpectedHits, count, "%s: expected %d hits, got %d", m.String(), m.ExpectedHits, count)
}

func (m *Mock) Matches(r *http.Request, allowPartialMatch bool) RequestMatchLevel {
	// disablePartialMatch=true must disable partial matching; invert to get allowPartialMatch.
	return m.Request.match(r, allowPartialMatch)
}

func (m *Mock) WriteResponse(r *http.Request, w http.ResponseWriter) {
	if m.customHandler != nil {
		m.customHandler(m, w, r)
	} else {
		m.Response.writeResponse(w)
	}
}

func (m *Mock) AcceptsPartialMatch() bool {
	return m.Request.PartialMatch
}

func (m *Mock) AppendLog(log string) {
	m.Request.matchLog = append(m.Request.matchLog, log)
}

func (m *Mock) Logs() []string {
	return m.Request.matchLog
}

func (m *Mock) Name() string {
	return m.MockName
}

func (m *Mock) GetPathValue(key string) (value string) {
	return m.Request.readData[readDataPathParamPrefix+key]
}
func (m *Mock) GetQueryValue(key string) (value string) {
	return m.Request.readData[readDataQueryParamPrefix+key]
}
func (m *Mock) GetHeaderValue(key string) (value string) {
	return m.Request.readData[readDataHeaderPrefix+key]
}

// Package httptestmock provides utilities for creating HTTP mock servers in Go tests.
// It allows defining request/response mocks in external JSON or YAML files for cleaner,
// more maintainable integration tests.
package httptestmock

import (
	"sync"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type (
	// Mock represents a complete mock definition containing both
	// the expected request to match and the response to return.
	Mock struct {
		// Name is an optional identifier for the mock, used in logging.
		// If not specified, defaults to the file path.
		Name string `json:"name" yaml:"name"`

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
	}

	requestMatchLevel uint8
)

const (
	// matchLevelNone indicates no match.
	matchLevelNone requestMatchLevel = iota
	// matchLevelPartial indicates a partial match.
	matchLevelPartial
	// matchLevelFull indicates a full match.
	matchLevelFull
)

// validate is the validator instance used to validate mock definitions.
var validate = validator.New(validator.WithRequiredStructEnabled())

// String returns a human-readable representation of the mock for logging.
func (m *Mock) String() string {
	sp := StringParts{}.Set("name", m.Name).
		Set("from", m.source).
		Set("req", m.Request.String()).
		Set("resp", m.Response.String())

	return "Mock: " + sp.String()
}

// Validate validates the mock definition using struct validation tags.
// Returns an error if required fields are missing or have invalid values.
func (m *Mock) Validate() error {
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

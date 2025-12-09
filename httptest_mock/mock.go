// Package httptestmock provides utilities for creating HTTP mock servers in Go tests.
// It allows defining request/response mocks in external JSON or YAML files for cleaner,
// more maintainable integration tests.
package httptestmock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v3"
)

type (
	// MockRequest represents a complete mock definition containing both
	// the expected request to match and the response to return.
	MockRequest struct {
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
func (m *MockRequest) String() string {
	sp := StringParts{}.Set("name", m.Name).Set("from", m.source).Set("req", m.Request).Set("resp", m.Response)

	return "Mock: " + sp.String()
}

// Validate validates the mock definition using struct validation tags.
// Returns an error if required fields are missing or have invalid values.
func (m *MockRequest) Validate() error {
	return validate.Struct(m)
}

// RegisterHit records a hit for this mock request during the test.
func (m *MockRequest) RegisterHit(t *testing.T) {
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
func (m *MockRequest) Assert(t *testing.T) {
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

// writeResponse writes the response headers, status code, and body to the ResponseWriter.
func (m *Response) writeResponse(w http.ResponseWriter) {
	if m.DelayMs > 0 {
		// Introduce delay before sending response
		time.Sleep(time.Duration(m.DelayMs) * time.Millisecond)
	}

	for key, value := range m.Headers {
		w.Header().Add(key, value)
	}

	w.WriteHeader(m.Status)
	m.writeBody(w)
}

// readMock reads and parses a mock definition from a JSON or YAML file.
// It first attempts JSON parsing, then falls back to YAML if JSON fails.
func readMock(path string) (*MockRequest, error) {
	file, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var mock MockRequest

	// Try JSON first, then fall back to YAML
	err = json.Unmarshal(file, &mock)
	if err != nil {
		err = yaml.Unmarshal(file, &mock)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal json/yaml %s: %w", path, err)
		}
	}

	// Use filename as mock name if not specified
	if mock.Name == "" {
		mock.Name = path
	}

	mock.source = path

	return &mock, nil
}

// readMocks reads all mock definitions from a directory.
// Processes files with .json, .yaml, or .yml extensions.
// Subdirectories are skipped.
func readMocks(dir string) ([]*MockRequest, error) {
	files, err := os.ReadDir(filepath.Clean(dir))
	if err != nil {
		return nil, err
	}

	requests := make([]*MockRequest, 0, len(files))
	for _, file := range files {
		ext := strings.ToLower(path.Ext(file.Name()))
		// Skip directories and non-mock files
		if file.IsDir() || (ext != ".json" && ext != ".yaml" && ext != ".yml") {
			continue
		}

		if mock, err := readMock(path.Join(dir, file.Name())); err != nil {
			return nil, err
		} else {
			requests = append(requests, mock)
		}
	}

	return requests, nil
}

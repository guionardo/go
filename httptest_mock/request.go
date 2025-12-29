package httptestmock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/guionardo/go/pkg/flow"
)

type (
	// Request defines the matching criteria for an incoming HTTP request.
	// A request matches when method, path, and all specified query parameters match.
	Request struct {
		// Method is the HTTP method to match (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS).
		Method string `json:"method" yaml:"method" validate:"required,oneof=GET POST PUT DELETE PATCH HEAD OPTIONS"` //nolint:lll

		// Path is the URL path to match (e.g., "/api/v1/users").
		Path string `json:"path" yaml:"path" validate:"required"`

		// QueryParams are optional query parameters that must all be present and match.
		QueryParams map[string]string `json:"query_params" yaml:"query_params" validate:"omitempty,dive,keys,endkeys"`

		// PathParams are optional path parameters that must all be present and match.
		PathParams map[string]string `json:"path_params" yaml:"path_params" validate:"omitempty,dive,keys,endkeys"`

		// Headers are optional request headers to match (not yet implemented).
		Headers map[string]string `json:"headers" yaml:"headers" validate:"omitempty,dive,keys,required,endkeys,required"`

		// Body is the expected request body (not yet implemented).
		Body any `json:"body" yaml:"body"`

		// Accept partial matching level
		PartialMatch bool `json:"partial_match" yaml:"partial_match"`

		readenData map[string]string // used internally to store readen data from the request

		// matchLog is used for debugging and logging purposes.
		// It contains the match log for the request.
		// This is not used in production code, but can be useful for debugging.
		// It is not serialized to JSON or YAML.
		matchLog []string
	}
)

const (
	noMatchEmoji = "❌"
	matchEmoji   = "✅"
)

// String returns a human-readable representation of the request for logging.
func (m Request) String() string {
	sp := StringParts{}.Set("method", m.Method).
		Set("path", m.Path).
		Set("query_params", m.QueryParams).
		Set("path_params", m.PathParams).
		Set("headers", m.Headers).
		Set("body", m.Body)

	return "Req: " + sp.String()
}

// match checks if the HTTP request matches the defined criteria.
// Compares method, path, query parameters, headers, and body.
func (m *Request) match(r *http.Request, disablePartialMatch bool) RequestMatchLevel {
	m.readenData = make(map[string]string)

	m.matchLog = make([]string, 0)
	if m.Method != r.Method {
		m.setMatchLog("METHOD", m.Method, r.Method)
		return matchLevelNone
	}

	if !m.matchPath(r) {
		m.setMatchLog("PATH", m.Path, r.URL.Path)
		return matchLevelNone
	}

	// The following checks are only performed when method and path match
	if m.matchQueryParams(r) && m.matchPathParams(r) && m.matchHeaders(r) && m.matchBody(r) {
		m.matchLog = append(m.matchLog, matchEmoji+" MATCH")
		return matchLevelFull
	}

	if disablePartialMatch {
		return matchLevelNone
	}

	return matchLevelPartial
}

// setMatchLog is a helper to append a formatted no-match message to the match log.
func (m *Request) setMatchLog(part string, expected string, actual string) {
	if expected == "" && actual != "" {
		m.matchLog = append(m.matchLog, fmt.Sprintf("%s %s expected empty but got %s", noMatchEmoji, part, actual))
		return
	}

	if expected != "" && actual == "" {
		m.matchLog = append(m.matchLog, fmt.Sprintf("%s %s expected %s but got empty", noMatchEmoji, part, expected))
		return
	}

	m.matchLog = append(m.matchLog, fmt.Sprintf("%s %s expected %s but got %s", noMatchEmoji, part, expected, actual))
}

// matchPath checks if the request path matches the defined path.
func (m *Request) matchPath(r *http.Request) bool {
	if strings.Contains(m.Path, "{") {
		// path with parameters
		mParts := strings.Split(m.Path, "/")

		rParts := strings.Split(r.URL.Path, "/")
		if len(mParts) != len(rParts) {
			return false
		}

		for i := range mParts {
			if strings.HasPrefix(mParts[i], "{") && strings.HasSuffix(mParts[i], "}") {
				// this is a path parameter, store it
				paramName := strings.Trim(mParts[i], "{}")
				m.readenData[readenDataPathParamPrefix+paramName] = rParts[i]
				// path parameter, skip matching
				continue
			}

			if mParts[i] != rParts[i] {
				return false
			}
		}

		return true
	}
	// exact path match
	return m.Path == r.URL.Path
}

// matchPathParams checks if all specified path parameters match the request.
func (m *Request) matchPathParams(r *http.Request) bool {
	if len(m.PathParams) == 0 {
		return true
	}

	for key, value := range m.PathParams {
		pathValue := flow.Default(r.PathValue(key), m.readenData[readenDataPathParamPrefix+key])
		if pathValue != value {
			m.setMatchLog("PATH PARAM ["+key+"]", value, pathValue)

			return false
		}
	}

	return true
}

// matchQueryParams checks if all specified query parameters match the request.
func (m *Request) matchQueryParams(r *http.Request) bool {
	if len(m.QueryParams) == 0 {
		return true
	}

	for key, value := range m.QueryParams {
		queryValue := r.URL.Query().Get(key)
		if queryValue != value {
			m.setMatchLog("QUERY PARAM ["+key+"]", value, queryValue)
			return false
		}

		m.readenData[readenDataQueryParamPrefix+key] = queryValue
	}

	return true
}

// matchHeaders checks if all specified headers match the request.
func (m *Request) matchHeaders(r *http.Request) bool {
	for key, value := range m.Headers {
		if queryValue := r.Header.Get(key); queryValue != value {
			m.matchLog = append(m.matchLog, fmt.Sprintf("%s HEADER %s != %s", noMatchEmoji, key, value))
			return false
		} else {
			m.readenData[readenDataHeaderPrefix+key] = r.Header.Get(key)
		}
	}

	return true
}

// matchBody checks if the request body matches the expected body.
func (m *Request) matchBody(r *http.Request) bool {
	if m.Body == nil {
		return true
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		m.matchLog = append(m.matchLog, fmt.Sprintf("%s BODY READ ERROR: %v", noMatchEmoji, err))
		return false
	}

	_ = r.Body.Close()

	// After reading, must replace the body so it can be read again
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	if !compareBody(m.Body, body) {
		m.matchLog = append(m.matchLog, fmt.Sprintf("%s BODY %s != %s", noMatchEmoji, body, m.Body))
		return false
	}

	return true
}

func compareBody(expected any, fromRequest []byte) bool {
	switch expected := expected.(type) {
	case string:
		return string(fromRequest) == expected
	case []byte:
		return bytes.Equal(fromRequest, expected)
	default:
		// For any other type (maps, structs, etc.), encode as JSON
		expectedMarshaled, errExp := marshalSorted(expected)
		requestBody, errReq := marshalSorted(fromRequest)

		return (errExp == nil) && (errReq == nil) && bytes.Equal(expectedMarshaled, requestBody)
	}
}

func marshalSorted(data any) (sortedBytes []byte, err error) {
	// if data is []byte, unmarshal to any
	if bytes, ok := data.([]byte); ok {
		var anyData any

		if err := json.Unmarshal(bytes, &anyData); err != nil {
			return nil, err
		}

		return json.Marshal(anyData)
	}
	// first marshal to json
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// then sort the json by keys
	var sortedData any

	err = json.Unmarshal(jsonData, &sortedData)
	if err == nil {
		// then marshal the sorted data to json
		sortedBytes, err = json.Marshal(sortedData)
	}

	return sortedBytes, err
}

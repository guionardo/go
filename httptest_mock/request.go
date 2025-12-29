package httptestmock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/guionardo/go/flow"
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

		// used internally to store read data from the request
		readData map[string]string

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
func (r Request) String() string {
	sp := StringParts{}.Set("method", r.Method).
		Set("path", r.Path).
		Set("query_params", r.QueryParams).
		Set("path_params", r.PathParams).
		Set("headers", r.Headers).
		Set("body", r.Body)

	return "Req: " + sp.String()
}

// match checks if the HTTP request matches the defined criteria.
// Compares method, path, query parameters, headers, and body.
func (r *Request) match(req *http.Request, allowPartialMatch bool) RequestMatchLevel {
	r.readData = make(map[string]string)

	r.matchLog = make([]string, 0)
	if r.Method != req.Method {
		r.setMatchLog("METHOD", r.Method, req.Method)
		return MatchLevelNone
	}

	if !r.matchPath(req) {
		r.setMatchLog("PATH", r.Path, req.URL.Path)
		return MatchLevelNone
	}

	// The following checks are only performed when method and path match
	if r.matchQueryParams(req) && r.matchPathParams(req) && r.matchHeaders(req) && r.matchBody(req) {
		r.matchLog = append(r.matchLog, matchEmoji+" MATCH")
		return MatchLevelFull
	}

	if allowPartialMatch {
		return MatchLevelPartial
	}

	return MatchLevelNone
}

// setMatchLog is a helper to append a formatted no-match message to the match log.
func (r *Request) setMatchLog(part string, expected string, actual string) {
	if expected == "" && actual != "" {
		r.matchLog = append(r.matchLog, fmt.Sprintf("%s %s expected empty but got %s", noMatchEmoji, part, actual))
		return
	}

	if expected != "" && actual == "" {
		r.matchLog = append(r.matchLog, fmt.Sprintf("%s %s expected %s but got empty", noMatchEmoji, part, expected))
		return
	}

	r.matchLog = append(r.matchLog, fmt.Sprintf("%s %s expected %s but got %s", noMatchEmoji, part, expected, actual))
}

// matchPath checks if the request path matches the defined path.
func (r *Request) matchPath(req *http.Request) bool {
	if strings.Contains(r.Path, "{") {
		// path with parameters
		mParts := strings.Split(r.Path, "/")

		rParts := strings.Split(req.URL.Path, "/")
		if len(mParts) != len(rParts) {
			return false
		}

		for i := range mParts {
			if strings.HasPrefix(mParts[i], "{") && strings.HasSuffix(mParts[i], "}") {
				// this is a path parameter, store it
				paramName := strings.Trim(mParts[i], "{}")
				r.readData[readDataPathParamPrefix+paramName] = rParts[i]
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
	return r.Path == req.URL.Path
}

// matchPathParams checks if all specified path parameters match the request.
func (r *Request) matchPathParams(req *http.Request) bool {
	if len(r.PathParams) == 0 {
		return true
	}

	for key, value := range r.PathParams {
		pathValue := flow.Default(req.PathValue(key), r.readData[readDataPathParamPrefix+key])
		if pathValue != value {
			r.setMatchLog("PATH PARAM ["+key+"]", value, pathValue)

			return false
		}
	}

	return true
}

// matchQueryParams checks if all specified query parameters match the request.
func (r *Request) matchQueryParams(req *http.Request) bool {
	if len(r.QueryParams) == 0 {
		return true
	}

	for key, value := range r.QueryParams {
		queryValue := req.URL.Query().Get(key)
		if queryValue != value {
			r.setMatchLog("QUERY PARAM ["+key+"]", value, queryValue)
			return false
		}

		r.readData[readDataQueryParamPrefix+key] = queryValue
	}

	return true
}

// matchHeaders checks if all specified headers match the request.
func (r *Request) matchHeaders(req *http.Request) bool {
	for key, value := range r.Headers {
		if queryValue := req.Header.Get(key); queryValue != value {
			r.matchLog = append(r.matchLog, fmt.Sprintf("%s HEADER %s != %s", noMatchEmoji, key, value))
			return false
		} else {
			r.readData[readDataHeaderPrefix+key] = req.Header.Get(key)
		}
	}

	return true
}

// matchBody checks if the request body matches the expected body.
func (r *Request) matchBody(req *http.Request) bool {
	if r.Body == nil {
		return true
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.matchLog = append(r.matchLog, fmt.Sprintf("%s BODY READ ERROR: %v", noMatchEmoji, err))
		return false
	}

	_ = req.Body.Close()

	// After reading, must replace the body so it can be read again
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	if !compareBody(r.Body, body) {
		r.matchLog = append(r.matchLog, fmt.Sprintf("%s BODY %s != %s", noMatchEmoji, body, r.Body))
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

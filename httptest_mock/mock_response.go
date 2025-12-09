package httptestmock

import (
	"encoding/json"
	"net/http"
)

type (
	// Response defines the HTTP response to return when a request matches.
	Response struct {
		// Status is the HTTP status code to return (100-599).
		Status int `json:"status" yaml:"status" validate:"required,min=100,max=599"`

		// Body is the response body. Can be a string, []byte, or any JSON-serializable type.
		// If nil, no body is written. Objects are JSON-encoded automatically.
		Body any `json:"body" yaml:"body"`

		// Headers are the response headers to include in the response.
		Headers map[string]string `json:"headers" yaml:"headers"`

		// DelayMs is an optional delay in milliseconds before sending the response
		DelayMs int `json:"delay_ms" yaml:"delay_ms"` //nolint:unused
	}
)

// String returns a human-readable representation of the response for logging.
// String returns a human-readable representation of the response for logging.
func (m *Response) String() string {
	sp := StringParts{}.Set("status", http.StatusText(m.Status)).
		Set("body", m.Body).
		Set("headers", m.Headers).
		Set("delay_ms", m.DelayMs)

	return "Resp: " + sp.String()
}

// writeBody writes the response body to the ResponseWriter.
// Handles string, []byte, and JSON-serializable types.
func (m *Response) writeBody(w http.ResponseWriter) {
	if m.Body == nil {
		return
	}

	var err error

	switch body := m.Body.(type) {
	case string:
		_, err = w.Write([]byte(body))
	case []byte:
		_, err = w.Write(body)
	default:
		// For any other type (maps, structs, etc.), encode as JSON
		err = json.NewEncoder(w).Encode(body)
	}

	if err != nil {
		// Do not call WriteHeader again; just write the error message to the body.
		_, _ = w.Write([]byte(err.Error()))
	}
}

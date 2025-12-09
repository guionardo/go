package httptestmock

import (
	"encoding/json"
	"net/http"
	"time"
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
		DelayMs int `json:"delay_ms" yaml:"delay_ms"`
	}
)

// String returns a human-readable representation of the response for logging.
func (m *Response) String() string {
	sp := StringParts{}.Set("status", http.StatusText(m.Status)).
		Set("body", m.Body).
		Set("headers", m.Headers).
		Set("delay_ms", m.DelayMs)

	return "Resp: " + sp.String()
}

// writeResponse writes the response headers, status code, and body to the ResponseWriter.
func (m *Response) writeResponse(w http.ResponseWriter) {
	if m.DelayMs > 0 {
		// Introduce delay before sending response
		time.Sleep(time.Duration(m.DelayMs) * time.Millisecond)
	}

	m.writeHeaderAndBody(w)
}

// writeHeaderAndBody writes the response headers and body to the given ResponseWriter.
// error catching prevents inconsistent status codes when marshaling fails.
func (m *Response) writeHeaderAndBody(w http.ResponseWriter) {
	var (
		bodyContent []byte
		statusCode  = m.Status
	)
	if m.Body != nil {
		switch body := m.Body.(type) {
		case string:
			bodyContent = []byte(body)
		case []byte:
			bodyContent = body
		default:
			var err error

			bodyContent, err = json.Marshal(body)
			if err != nil {
				bodyContent = []byte(err.Error())
				statusCode = http.StatusInternalServerError
			} else {
				// Set Content-Type to application/json if body is JSON
				m.Headers["Content-Type"] = "application/json"
			}
		}
	}

	for key, value := range m.Headers {
		w.Header().Add(key, value)
	}

	w.WriteHeader(statusCode)

	if len(bodyContent) > 0 {
		_, _ = w.Write(bodyContent)
	}
}

package httptestmock_test

import (
	"net/url"
	"testing"

	httptestmock "github.com/guionardo/go/httptest_mock"
	"github.com/stretchr/testify/assert"
)

func TestStringParts(t *testing.T) { //nolint:funlen
	t.Parallel()

	turl, _ := url.Parse("http://example.com") // to use in test cases

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		kvs []struct {
			key   string
			value any
		}
		want string
	}{
		{"empty StringParts", nil, ""},
		{"single part", []struct {
			key   string
			value any
		}{{"method", "GET"}}, "[method: GET]"},
		{"multiple parts", []struct {
			key   string
			value any
		}{
			{"method", "POST"},
			{"path", "/api/v1/resource"},
			{"status", 200},
			{"method", "GET"},
		}, "[method: GET] [path: /api/v1/resource] [status: 200]"},
		{"part with empty string value", []struct {
			key   string
			value any
		}{
			{"method", ""},
			{"path", "/api/v1/resource"},
			{"duration", float32(12.5)},
		}, "[path: /api/v1/resource] [duration: 12.5]"},
		{"part with nil value", []struct {
			key   string
			value any
		}{
			{"method", nil},
			{"status", 404},
			{"url", turl},
		}, "[status: 404] [url: http://example.com]"},
		{"part with request and response", []struct {
			key   string
			value any
		}{
			{"request", httptestmock.Request{Method: "GET", Path: "/test"}},
			{"response", httptestmock.Response{Status: 200, Body: "OK"}},
		}, "[request: Req: [method: GET] [path: /test]] [response: {Status:200 Body:OK Headers:map[] DelayMs:0}]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var s httptestmock.StringParts
			for _, kv := range tt.kvs {
				s = s.Set(kv.key, kv.value)
			}

			got := s.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

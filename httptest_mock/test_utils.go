package httptestmock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func CreateTestRequest(t *testing.T, p *httptest.Server, method string, path string, body any) *http.Request {
	t.Helper()

	bodyReader := getBodyReader(t, body)
	req, err := http.NewRequest(method, p.URL+"/"+strings.TrimPrefix(path, "/"), bodyReader)
	require.NoError(t, err)

	return req
}

func getBodyReader(t *testing.T, body any) io.Reader {
	t.Helper()

	if body == nil {
		return nil
	}

	switch body := body.(type) {
	case string:
		return bytes.NewBufferString(body)

	case []byte:
		return bytes.NewBuffer(body)
	default:
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err, "failed to marshal body")

		return bytes.NewReader(bodyBytes)
	}
}

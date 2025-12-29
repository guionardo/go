package httptestmock_test

import (
	"encoding/json"
	"net/http"
	"testing"

	httptestmock "github.com/guionardo/go/httptest_mock"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	t.Parallel()

	mock := httptestmock.NewMock(http.MethodPost, "/example/{id}").
		WithQueryParam("key", "value").
		WithPathParam("id", "123").
		WithHeader("Authorization", "Bearer token").
		WithBody(map[string]string{"field": "data"}).
		WithResponseStatus(200).
		WithResponseBody(map[string]string{"response": "success"}).
		WithResponseHeader("Content-Type", "application/json").
		WithAssertion(true, 1).
		WithCustomHandler(func(m httptestmock.Mocker, w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)

			id := m.GetPathValue("id")
			response := map[string]string{
				"id":     id,
				"custom": "handler",
			}
			_ = json.NewEncoder(w).Encode(response)
		})

	server, assert := mock.FastServe(t)
	defer assert(t)

	req := httptestmock.CreateTestRequest(t, server,
		http.MethodPost, "/example/123?key=value",
		map[string]string{"field": "data"})

	req.Header.Set("Authorization", "Bearer token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	var respData map[string]string

	err = json.NewDecoder(resp.Body).Decode(&respData)
	require.NoError(t, err)

	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "123", respData["id"])
	require.Equal(t, "handler", respData["custom"])
}

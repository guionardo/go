package httptestmock_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httptestmock "github.com/guionardo/go/httptest_mock"
	"github.com/stretchr/testify/require"
)

func TestGetMockHandlerFromServer(t *testing.T) {
	t.Parallel()

	t.Run("mock_server_with_extra_logger_and_mock_appended", func(t *testing.T) {
		t.Parallel()

		server, assert := httptestmock.SetupServer(t,
			httptestmock.WithRequestsFrom("mocks"),
			httptestmock.WithExtraLogger(slog.New(slog.NewTextHandler(t.Output(), nil))))

		defer assert(t)

		handler, err := httptestmock.GetMockHandlerFromServer(server)
		require.NoError(t, err)
		err = handler.AddMocks(&httptestmock.Mock{
			Name: "appended_request",
			Request: httptestmock.Request{
				Method: http.MethodGet,
				Path:   "/appended",
			},
			Response: httptestmock.Response{
				Status: http.StatusOK,
				Body:   "Hello, appending",
			},
		})
		require.NoError(t, err)

		url := server.URL + "/appended"
		resp, err := http.Get(url) // nolint:gosec
		require.NoError(t, err)

		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		require.Equal(t, "Hello, appending", string(body))
	})

	t.Run("nil_server", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(nil)
		defer server.Close()

		handler, err := httptestmock.GetMockHandlerFromServer(server)
		require.Error(t, err)
		require.Nil(t, handler)
	})
}

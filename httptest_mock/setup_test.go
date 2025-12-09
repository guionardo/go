package httptestmock

import (
	"io"
	"net/http"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupServer(t *testing.T) {
	t.Parallel()
	t.Run("happy path with simple mock", func(t *testing.T) {
		t.Parallel()

		mockServer, assertFunc := SetupServer(t, WithRequestsFromDir("mocks"))
		defer assertFunc(t)

		response, err := http.Get(mockServer.URL + "/api/v1/example")
		require.NoError(t, err)

		defer func() { _ = response.Body.Close() }()

		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, "application/json", response.Header.Get("Content-Type"))

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"message":"Hello, world!"}`, string(body))
	})
}

func TestSetupOptions(t *testing.T) {
	t.Parallel()
	t.Run("WithRequestsFromDir : success", func(t *testing.T) {
		t.Parallel()
		s := MockHandler{T: t}
		WithRequestsFromDir(path.Join("mocks", "examples"))(&s)
		require.NoError(t, s.Validate())
	})
	t.Run("WithRequestsFromDir : directory does not exist", func(t *testing.T) {
		t.Parallel()
		s := MockHandler{T: t}
		WithRequestsFromDir("non_existing_directory")(&s)
		require.Error(t, s.Validate())
	})
	t.Run("WithAddMockInfoToResponse : empty header name", func(t *testing.T) {
		t.Parallel()

		s, assertFunc := SetupServer(t, WithAddMockInfoToResponse(),
			WithRequestsFromDir(path.Join("mocks", "examples")))
		defer assertFunc(t)

		response, err := http.Get(s.URL + "/api/v1/users")
		require.NoError(t, err)

		defer func() { _ = response.Body.Close() }()

		require.Equal(t, "example_2", response.Header.Get("HTTPTestMock-Name"))
	})
	t.Run("WithAddMockInfoToResponse : custom header name", func(t *testing.T) {
		t.Parallel()

		s, assertFunc := SetupServer(t, WithAddMockInfoToResponse("X-Custom-Mock-Info"),
			WithRequestsFromDir(path.Join("mocks", "examples")))
		defer assertFunc(t)

		response, err := http.Get(s.URL + "/api/v1/users")
		require.NoError(t, err)

		defer func() { _ = response.Body.Close() }()

		require.Equal(t, "example_2", response.Header.Get("X-Custom-Mock-Info-Name"))
	})
}

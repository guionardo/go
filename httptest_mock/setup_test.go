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

		mockServer, assertFunc := SetupServer(t, WithRequestsFrom("mocks"))
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

func TestSetupOptions_WithRequestsFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		paths         []string
		expectedError bool
	}{
		{
			name:          "single_valid_file",
			paths:         []string{"mocks/get_user.json"},
			expectedError: false},
		{
			name:          "multiple_valid_files",
			paths:         []string{"mocks/get_user.json", "mocks/example_1.yaml"},
			expectedError: false,
		},
		{
			name:          "non_existing_file",
			paths:         []string{"mocks/non_existing_file.json"},
			expectedError: true},
		{
			name:          "mixed_existing_and_non_existing_files",
			paths:         []string{"mocks/get_user.json", "mocks/non_existing_file.json"},
			expectedError: false,
		}, {
			name:          "from_directory_and_file",
			paths:         []string{path.Join("mocks", "examples"), "mocks/get_user.json"},
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := MockHandler{T: t}
			WithRequestsFrom(tt.paths...)(&s)

			err := s.Validate()
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSetupOptions_WithAddMockInfoToResponse(t *testing.T) {
	t.Parallel()
	t.Run("empty_header_name", func(t *testing.T) {
		t.Parallel()

		s, assertFunc := SetupServer(t, WithAddMockInfoToResponse(),
			WithRequestsFrom(path.Join("mocks", "examples")))
		defer assertFunc(t)

		response, err := http.Get(s.URL + "/api/v1/users")
		require.NoError(t, err)

		defer func() { _ = response.Body.Close() }()

		require.Equal(t, "example_2", response.Header.Get("Httptestmock-Name"))
	})
	t.Run("custom_header_name", func(t *testing.T) {
		t.Parallel()

		s, assertFunc := SetupServer(t, WithAddMockInfoToResponse("X-Custom-Mock-Info"),
			WithRequestsFrom(path.Join("mocks", "examples")))
		defer assertFunc(t)

		response, err := http.Get(s.URL + "/api/v1/users")
		require.NoError(t, err)

		defer func() { _ = response.Body.Close() }()

		require.Equal(t, "example_2", response.Header.Get("X-Custom-Mock-Info-Name"))
	})
}

func Test_unmarshalMock(t *testing.T) {
	t.Parallel()

	t.Run("when_input_is_empty_then_error", func(t *testing.T) {
		t.Parallel()

		_, err := unmarshalMock([]byte{})
		require.Error(t, err)
	})

	t.Run("when_input_is_valid_json_then_success", func(t *testing.T) {
		t.Parallel()

		mock, err := readMock(path.Join("mocks", "get_user.json"))
		require.NoError(t, err)
		require.Equal(t, "get_user", mock.MockName)
	})

	t.Run("when_input_is_valid_yaml_then_success", func(t *testing.T) {
		t.Parallel()

		mock, err := readMock(path.Join("mocks", "example_1.yaml"))
		require.NoError(t, err)
		require.Equal(t, "example_1", mock.MockName)
	})

	t.Run("when_input_is_invalid_then_error", func(t *testing.T) {
		t.Parallel()

		_, err := unmarshalMock([]byte(`{invalid_json: true`))
		require.Error(t, err)

		_, err = unmarshalMock([]byte(`"""name: example"`))
		require.Error(t, err)
	})
}
func Test_readMock(t *testing.T) {
	t.Parallel()

	t.Run("file_does_not_exist_should_return_error", func(t *testing.T) {
		t.Parallel()

		_, err := readMock("non_existing_file.json")
		require.Error(t, err)
	})

	t.Run("mock_with_no_name_should_default_to_file_name", func(t *testing.T) {
		t.Parallel()

		mock, err := readMock("mocks/example_2_noname.yaml")
		require.NoError(t, err)
		require.Equal(t, "example_2_noname", mock.MockName)
	})

	t.Run("simple mock should be read successfully", func(t *testing.T) {
		t.Parallel()

		mock, err := readMock("mocks/get_user.json")
		require.NoError(t, err)
		require.Equal(t, "get_user", mock.MockName)
		require.Equal(t, "GET", mock.Request.Method)
		require.Equal(t, "/api/v1/users/123", mock.Request.Path)
	})
	t.Run("invalid mock should raise an error", func(t *testing.T) {
		t.Parallel()

		_, err := readMock("mocks/bad_mock.json")
		require.Error(t, err)
	})

	t.Run("invalid json mock should raise an error", func(t *testing.T) {
		t.Parallel()

		_, err := readMock("mocks/bad_mock/bad_mock_invalid_json.json")
		require.Error(t, err)
	})
}

func Test_readMocksFromPath(t *testing.T) {
	t.Parallel()

	mocks, err := readMocksFromPath(path.Join("mocks", "examples", "*"))
	require.NoError(t, err)
	require.Len(t, mocks, 4)
}

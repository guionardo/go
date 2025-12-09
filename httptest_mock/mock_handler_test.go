package httptestmock_test

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"testing"

	httptestmock "github.com/guionardo/go/httptest_mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func doRequest(t *testing.T, req *http.Request) (resp *http.Response, body []byte, mockName string, err error) {
	t.Helper()

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	mockName = resp.Header.Get("Httptestmock-Name")

	return resp, body, mockName, nil
}
func TestMockHandler_ServeHTTP(t *testing.T) { //nolint:funlen
	t.Parallel()

	s, assertFunc := httptestmock.SetupServer(t,
		httptestmock.WithRequestsFrom(path.Join("mocks", "examples")),
		httptestmock.WithAddMockInfoToResponse())
	defer assertFunc(t)

	t.Run("example_1_exactly_matching_should_return_200_OK", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("POST", s.URL+"/api/v1/users/123?user_id=123", bytes.NewBufferString("TEST_BODY"))
		req.Header.Add("Api_key", "test_key")

		resp, respBody, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "example_1", mockName)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t, `{"message":"Hello, world!"}`, string(respBody))
	})
	t.Run("example_1_query_unmatch_should_return_400_Bad_Request", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("POST", s.URL+"/api/v1/users/123?user_id=456", bytes.NewBufferString("TEST_BODY"))
		req.Header.Add("Api_key", "test_key")

		resp, _, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, mockName)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("example_1_path_unmatch_should_return_400_Bad_Request", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("POST", s.URL+"/api/v1/users/456?user_id=123", bytes.NewBufferString("TEST_BODY"))
		req.Header.Add("Api_key", "test_key")

		resp, _, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, mockName)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("example_1_body_unmatch_should_return_400_Bad_Request", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest(
			"POST",
			s.URL+"/api/v1/users/123?user_id=123",
			bytes.NewBufferString("DIFFERENT_BODY"),
		)
		req.Header.Add("Api_key", "test_key")

		resp, _, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, mockName)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("ServeHTTP with non-matching request", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("GET", s.URL+"/api/v1/customers", nil)
		resp, _, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, mockName)

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
	t.Run("ServeHTTP with partial-matching request - should return 400 Bad Request", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("POST", s.URL+"/api/v1/users/123?user_id=123", bytes.NewBufferString("TEST_BODY"))
		req.Header.Add("Api_key", "unexpected key")
		resp, _, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, mockName)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("example_3_partial_match_should_return_200_OK", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("POST", s.URL+"/api/v1/owners", bytes.NewBufferString("TEST_BODY"))
		req.Header.Add("Api_key", "unexpected key")
		resp, _, mockName, err := doRequest(t, req) //nolint:bodyclose
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "example_3", mockName)

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestAssertion(t *testing.T) {
	t.Parallel()

	mockServer, assertFunc := httptestmock.SetupServer(t,
		httptestmock.WithRequestsFrom(path.Join("mocks", "assertions")),
		httptestmock.WithoutLog())
	defer assertFunc(t) // assert that the server received the expected number of requests

	// simulates multiple requests to the server
	const totalRequests = 100

	eg := &errgroup.Group{}
	eg.SetLimit(10)

	for range totalRequests {
		eg.Go(func() error {
			req, _ := http.NewRequest("GET", mockServer.URL+"/health", nil)
			_, _, _, err := doRequest(t, req) //nolint:bodyclose

			return err
		})
	}

	err := eg.Wait()
	require.NoError(t, err)
}

func TestMockHandler_Validate(t *testing.T) {
	t.Parallel()
	t.Run("mock handler with no requests should return error", func(t *testing.T) {
		t.Parallel()

		var s httptestmock.MockHandler
		require.Error(t, s.Validate())
	})
	t.Run("mock handler with invalid requests should return error", func(t *testing.T) {
		t.Parallel()

		var s = httptestmock.MockHandler{T: t}

		request := &httptestmock.Mock{
			Name: "invalid_request",
			Request: httptestmock.Request{
				Method:      "GETCH", // invalid HTTP method
				Path:        "/api/v1/users/123",
				QueryParams: make(map[string]string),
				PathParams:  make(map[string]string),
			},
		}
		httptestmock.WithRequests(request)(&s)

		require.Error(t, s.Validate())
	})
}

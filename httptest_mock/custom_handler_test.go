package httptestmock_test

import (
	"io"
	"net/http"
	"testing"

	httptestmock "github.com/guionardo/go/httptest_mock"
	"github.com/stretchr/testify/require"
)

type customMock struct {
	logMessages []string
}

var _ httptestmock.Mocker = (*customMock)(nil)

func (c *customMock) AcceptsPartialMatch() bool {
	return false
}

func (c *customMock) Matches(r *http.Request, allowPartialMatch bool) httptestmock.RequestMatchLevel {
	if r.URL.Path == "/" && r.Method == http.MethodGet {
		return httptestmock.MatchLevelFull
	}

	return httptestmock.MatchLevelNone
}

func (c *customMock) Name() string {
	return "customMock"
}

func (c *customMock) String() string {
	return "customMock GET / -> response customMock"
}

func (c *customMock) AppendLog(log string) {
	c.logMessages = append(c.logMessages, log)
}

func (c *customMock) Logs() []string {
	return c.logMessages
}

func (c *customMock) GetPathValue(key string) string {
	return ""
}

func (c *customMock) GetQueryValue(key string) string {
	return ""
}

func (c *customMock) GetHeaderValue(key string) string {
	return ""
}

func (c *customMock) WriteResponse(r *http.Request, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("customMock"))
}

func (c *customMock) RegisterHit(t *testing.T) {
	c.logMessages = append(c.logMessages, "hit registered")
}

func (c *customMock) Assert(t *testing.T) {
	require.Len(t, c.logMessages, 1)
	require.Equal(t, "hit registered", c.logMessages[0])
}

func (c *customMock) Validate() error {
	return nil
}

func TestCustomHandler(t *testing.T) {
	t.Parallel()

	customMock := &customMock{
		logMessages: make([]string, 0),
	}

	server, assertFunc := httptestmock.SetupServer(t, httptestmock.WithRequests(customMock))

	defer assertFunc(t)

	req := httptestmock.CreateTestRequest(t, server, http.MethodGet, "/", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "customMock", string(body))
}

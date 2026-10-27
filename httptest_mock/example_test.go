package httptestmock_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	httptestmock "github.com/guionardo/go/httptest_mock"
)

// newHandler wires mocks declared in code into a MockHandler.
//
// A test would rely on mock.FastServe(t) or httptestmock.SetupServer(t, ...):
// those install t as the log sink and close the server when the test ends.
// Examples receive no *testing.T, so the logging sink is switched off and the
// structured logger is replaced with a discard logger instead.
func newHandler(mocks []httptestmock.Mocker, options ...func(*httptestmock.MockHandler)) *httptestmock.MockHandler {
	handler := &httptestmock.MockHandler{}

	handlerOptions := append([]func(*httptestmock.MockHandler){
		httptestmock.WithoutLog(),
		httptestmock.WithExtraLogger(nil),
	}, options...)

	for _, option := range handlerOptions {
		option(handler)
	}

	httptestmock.WithRequests(mocks...)(handler)

	if err := handler.Validate(); err != nil {
		panic(err)
	}

	return handler
}

// doGet performs a GET and returns the status code, the Content-Type header and
// the response body.
func doGet(url string) (status int, contentType, body string) {
	response, err := http.Get(url) //nolint:gosec // URL comes from the httptest.Server under test
	if err != nil {
		panic(err)
	}

	defer func() { _ = response.Body.Close() }()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	return response.StatusCode, response.Header.Get("Content-Type"), string(content)
}

// ExampleMockHandler serves a mock declared with the builder from a real HTTP
// server. The mock's path is a pattern, so /api/users/{id} matches
// /api/users/42. A map body is JSON-encoded and the response gets a JSON
// content type automatically. A request matching no mock is answered with 404.
func ExampleMockHandler() {
	mock := httptestmock.NewMock(http.MethodGet, "/api/users/{id}").
		WithResponseStatus(http.StatusOK).
		WithResponseBody(map[string]string{"name": "Ada"})

	server := httptest.NewServer(newHandler([]httptestmock.Mocker{mock}))
	defer server.Close()

	status, contentType, body := doGet(server.URL + "/api/users/42")
	fmt.Println(status, contentType)
	fmt.Println(body)

	status, _, _ = doGet(server.URL + "/api/users")
	fmt.Println(status)

	// Output:
	// 200 application/json
	// {"name":"Ada"}
	// 404
}

// ExampleMockHandler_partialMatch shows that lenient matching needs both
// switches: WithAcceptingPartialMatch() on the handler and PartialMatch on the
// individual mock. With either one missing, a request that matches method and
// path but not the query parameter is treated as no match and gets 404.
func ExampleMockHandler_partialMatch() {
	newMock := func() *httptestmock.Mock {
		mock := httptestmock.NewMock(http.MethodGet, "/search").
			WithQueryParam("page", "1").
			WithResponseStatus(http.StatusOK).
			WithResponseBody("first page")
		mock.Request.PartialMatch = true

		return mock
	}

	strict := httptest.NewServer(newHandler([]httptestmock.Mocker{newMock()}))
	defer strict.Close()

	lenient := httptest.NewServer(newHandler(
		[]httptestmock.Mocker{newMock()},
		httptestmock.WithAcceptingPartialMatch(),
	))
	defer lenient.Close()

	status, _, _ := doGet(strict.URL + "/search?page=2")
	fmt.Println("strict:", status)

	status, _, body := doGet(lenient.URL + "/search?page=2")
	fmt.Println("lenient:", status, body)

	// Output:
	// strict: 404
	// lenient: 200 first page
}

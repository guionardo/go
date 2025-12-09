package httptestmock

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type badMarshaler struct{}

func (b badMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal error")
}

func TestResponse_writeBody(t *testing.T) {
	t.Parallel()
	t.Run("empty body should write nothing", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK}
		response.writeBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, w.Body.String())
	})
	t.Run("string body should be written", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: "Hello, world!"}
		response.writeBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "Hello, world!", w.Body.String())
	})
	t.Run("byte array body should be written", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: []byte("Hello, world!")}
		response.writeBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "Hello, world!", w.Body.String())
	})
	t.Run("struct body should be written", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: struct {
			Name string
			Age  int
		}{Name: "John", Age: 30}}
		response.writeBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"Name":"John","Age":30}`, w.Body.String())
	})
	t.Run("invalid body should return internal server error", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: &badMarshaler{}} // body is invalid, cannot be marshaled
		response.writeBody(w)
		require.Equal(
			t,
			"json: error calling MarshalJSON for type *httptestmock.badMarshaler: marshal error",
			w.Body.String(),
		)
	})
}

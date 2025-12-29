package httptestmock

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponse_writeHeaderAndBody(t *testing.T) { //nolint:funlen
	t.Parallel()
	t.Run("empty_body_should_write_nothing", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK}
		response.writeHeaderAndBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, w.Body.String())
	})
	t.Run("string_body_should_write_string", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: "Hello, world!"}
		response.writeHeaderAndBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "Hello, world!", w.Body.String())
	})
	t.Run("byte_array_body_should_write_bytes", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: []byte("Hello, world!")}
		response.writeHeaderAndBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "Hello, world!", w.Body.String())
	})
	t.Run("struct_body_should_write_json", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{
			Status: http.StatusOK,
			Body: struct {
				Name string
				Age  int
			}{Name: "John", Age: 30},
			Headers: make(map[string]string),
		}
		response.writeHeaderAndBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"Name":"John","Age":30}`, w.Body.String())
		require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
	t.Run("struct_body_should_not_override_existing_content_type", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{
			Status:  http.StatusOK,
			Body:    map[string]any{"ok": true},
			Headers: map[string]string{"content-type": "application/vnd.custom+json"},
		}
		response.writeHeaderAndBody(w)
		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"ok":true}`, w.Body.String())
		require.Equal(t, "application/vnd.custom+json", w.Header().Get("Content-Type"))
	})
	t.Run("invalid_body_should_return_internal_server_error", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		response := &Response{Status: http.StatusOK, Body: &badMarshaler{}} // body is invalid, cannot be marshaled
		response.writeHeaderAndBody(w)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(
			t,
			"json: error calling MarshalJSON for type *httptestmock.badMarshaler: marshal error",
			w.Body.String(),
		)
	})
}

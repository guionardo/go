package httptestmock

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	bodyWithError struct{}
)

func (b *bodyWithError) Read(p []byte) (n int, err error) {
	return 0, errors.New("error reading body")
}

func TestRequest_matchBody(t *testing.T) {
	t.Parallel()

	t.Run("when_expected_body_but_invalid_received_should_return_false", func(t *testing.T) {
		t.Parallel()

		r := Request{
			Body: []byte("test body"),
		}

		request := httptest.NewRequest("POST", "http://localhost/test", &bodyWithError{})
		assert.False(t, r.matchBody(request))
	})

	t.Run("when_expected_body_but_nil_received_should_return_false", func(t *testing.T) {
		t.Parallel()

		r := Request{
			Body: []byte("test body"),
		}

		request := httptest.NewRequest("POST", "http://localhost/test", nil)
		assert.False(t, r.matchBody(request))
	})
}

func TestRequest_setMatchLog(t *testing.T) {
	t.Parallel()

	var r Request
	r.setMatchLog("PART", "waiting for", "got") // got is different from expected
	r.setMatchLog("PART2", "expected", "")      // expected value but none got
	r.setMatchLog("PART3", "", "got")           // got value but none expected
	assert.Len(t, r.matchLog, 3)
}

func TestRequest_matchPath(t *testing.T) {
	t.Parallel()
	t.Run("full_match_should_return_true", func(t *testing.T) {
		t.Parallel()

		r := Request{Path: "/api/v1/resource"}
		req := httptest.NewRequest("GET", "http://localhost/api/v1/resource", nil)
		assert.True(t, r.matchPath(req))
	})
	t.Run("valid_path_param_should_return_true", func(t *testing.T) {
		t.Parallel()

		r := Request{Path: "/api/v1/resource/{id}", PathParams: map[string]string{"id": "123"}}
		req := httptest.NewRequest("GET", "http://localhost/api/v1/resource/123", nil)
		assert.True(t, r.matchPath(req))
	})

	t.Run("unmatched_path_should_return_false", func(t *testing.T) {
		t.Parallel()

		r := Request{Path: "/api/v1/resource/{id}", PathParams: map[string]string{"id": "123"}}
		req := httptest.NewRequest("GET", "http://localhost/api/v1/other/123", nil)
		assert.False(t, r.matchPath(req))
	})
}

func Test_compareBody(t *testing.T) {
	t.Parallel()
	t.Run("string should be equal", func(t *testing.T) {
		t.Parallel()

		expected := "Hello, world!"
		fromRequest := []byte("Hello, world!")
		require.True(t, compareBody(expected, fromRequest))
	})

	t.Run("byte array should be equal", func(t *testing.T) {
		t.Parallel()

		expected := []byte("Hello, world!")
		fromRequest := []byte("Hello, world!")
		require.True(t, compareBody(expected, fromRequest))
	})
	t.Run("struct should be equal", func(t *testing.T) {
		t.Parallel()

		expected := struct {
			Name string
			Age  int
		}{Name: "John", Age: 30}
		fromRequest := []byte(`{"Name":"John","Age":30}`)
		require.True(t, compareBody(expected, fromRequest))
	})
}

func Test_marshalSorted(t *testing.T) {
	t.Parallel()

	t.Run("byte_array_input_should_marshal_successfully", func(t *testing.T) {
		t.Parallel()

		input := []byte(`{"b":2,"a":1}`)
		expected := []byte(`{"a":1,"b":2}`)

		marshaled, err := marshalSorted(input)
		require.NoError(t, err)
		require.JSONEq(t, string(expected), string(marshaled))
	})

	t.Run("bad_input_should_return_error", func(t *testing.T) {
		t.Parallel()

		input := []byte(`{invalid_json: true`)

		_, err := marshalSorted(input)
		require.Error(t, err)
	})

	t.Run("struct_input_should_marshal_successfully", func(t *testing.T) {
		t.Parallel()

		input := struct {
			B int `json:"b"`
			A int `json:"a"`
		}{B: 2, A: 1}
		expected := []byte(`{"a":1,"b":2}`)

		marshaled, err := marshalSorted(input)
		require.NoError(t, err)
		require.JSONEq(t, string(expected), string(marshaled))
	})

	t.Run("bad_struct_input_should_return_error", func(t *testing.T) {
		t.Parallel()

		input := badMarshaler{}

		_, err := marshalSorted(input)
		require.Error(t, err)
	})
}

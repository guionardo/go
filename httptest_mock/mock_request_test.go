package httptestmock

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	bodyWithError struct{}
)

func (b *bodyWithError) Read(p []byte) (n int, err error) {
	return 0, errors.New("error reading body")
}

func TestRequest_matchBody(t *testing.T) {
	t.Parallel()

	t.Run("should not match when body has read error", func(t *testing.T) {
		t.Parallel()

		r := Request{
			Body: []byte("test body"),
		}

		request := httptest.NewRequest("POST", "http://localhost/test", &bodyWithError{})
		assert.False(t, r.matchBody(request))
	})

	t.Run("should not match when request body is nil but mock expects body", func(t *testing.T) {
		t.Parallel()

		r := Request{
			Body: []byte("test body"),
		}

		request := httptest.NewRequest("POST", "http://localhost/test", nil)
		assert.False(t, r.matchBody(request))
	})
}

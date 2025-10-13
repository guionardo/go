package flow_test

import (
	"testing"

	"github.com/guionardo/go/pkg/flow"
	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 1, flow.Default(0, 1))
		assert.Equal(t, 1, flow.Default(1, 2))
	})
	t.Run("float", func(t *testing.T) {
		assert.Equal(t, 1.1, flow.Default(float64(0), 1.1))
	})
	t.Run("struct", func(t *testing.T) {
		type X struct {
			Name string
		}
		assert.Equal(t, X{"ABC"}, flow.Default(X{}, X{"ABC"}))
	})
}

package flow_test

import (
	"testing"

	"github.com/guionardo/go/flow"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	t.Parallel()
	t.Run("int", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, 1, flow.Default(0, 1))
		require.Equal(t, 1, flow.Default(1, 2))
	})
	t.Run("float", func(t *testing.T) {
		t.Parallel()
		require.InEpsilon(t, 1.1, flow.Default(float64(0), 1.1), 0.01)
	})
	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		type X struct {
			Name string
		}

		require.Equal(t, X{"ABC"}, flow.Default(X{}, X{"ABC"}))
	})
}

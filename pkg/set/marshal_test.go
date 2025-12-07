package set_test

import (
	"encoding/json"
	"testing"

	"github.com/guionardo/go/pkg/set"
	"github.com/stretchr/testify/require"
)

func TestSet_MarshalingJSON(t *testing.T) {
	t.Parallel()
	t.Run("int", func(t *testing.T) {
		t.Parallel()

		s := set.New(1, 2, 3, 4)
		j, err := json.Marshal(s)
		require.NoError(t, err)
		require.NotEmpty(t, j)

		s2 := set.New[int]()
		err = json.Unmarshal(j, &s2)
		require.NoError(t, err)

		require.True(t, s.Equals(s2))
	})
	t.Run("string", func(t *testing.T) {
		t.Parallel()

		s := set.New("A", "B", "C", "D")
		j, err := json.Marshal(s)
		require.NoError(t, err)
		require.NotEmpty(t, j)

		s2 := set.New[string]()
		err = json.Unmarshal(j, &s2)
		require.NoError(t, err)

		require.True(t, s.Equals(s2))
	})
}

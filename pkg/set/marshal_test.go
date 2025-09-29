package set_test

import (
	"encoding/json"
	"testing"

	"github.com/guionardo/go/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestSet_MarshalingJSON(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		s := set.New(1, 2, 3, 4)
		j, err := json.Marshal(s)
		assert.NoError(t, err)
		assert.NotEmpty(t, j)

		s2 := set.New[int]()
		err = json.Unmarshal(j, &s2)
		assert.NoError(t, err)

		assert.True(t, s.Equals(s2))
	})
	t.Run("string", func(t *testing.T) {
		s := set.New("A", "B", "C", "D")
		j, err := json.Marshal(s)
		assert.NoError(t, err)
		assert.NotEmpty(t, j)

		s2 := set.New[string]()
		err = json.Unmarshal(j, &s2)
		assert.NoError(t, err)

		assert.True(t, s.Equals(s2))
	})

}

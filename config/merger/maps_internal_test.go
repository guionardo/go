package merger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateMapValues_NilCurrent(t *testing.T) {
	t.Parallel()

	m := map[string]any{"a": 1}
	updateMapValues(nil, m)

	result := MergeMaps(m)
	require.Equal(t, map[string]any{"a": 1}, result)
}

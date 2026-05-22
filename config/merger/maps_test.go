package merger_test

import (
	"testing"

	"github.com/guionardo/go/config/merger"
	"github.com/stretchr/testify/require"
)

func TestMergeMaps(t *testing.T) {
	t.Parallel()

	m1 := map[string]any{
		"a": 1,
		"b": map[string]any{},
	}
	m2 := map[string]any{
		"a": 4,
		"b": map[string]any{
			"b.1": 5,
			"b.2": 6,
		},
		"c": 6,
	}
	m3 := map[string]any{
		"a": true,
		"b": 8,
		"c": 9,
	}

	want := map[string]any{
		"a": 4,
		"b": map[string]any{
			"b.1": 5,
			"b.2": 6,
		},
		"c": 9,
	}

	got := merger.MergeMaps(m1, m2, m3)
	require.Equal(t, want, got)
}

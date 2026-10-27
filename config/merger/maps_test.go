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

func TestMergeMaps_NestedNonAnyMapIsReplacedWithoutPanic(t *testing.T) {
	t.Parallel()

	baseLabels := map[string]string{"env": "dev"}
	overrideLabels := map[string]string{"env": "prod"}
	base := map[string]any{"labels": baseLabels}
	override := map[string]any{"labels": overrideLabels}

	var got map[string]any

	require.NotPanics(t, func() {
		got = merger.MergeMaps(base, override)
	})
	require.Equal(t, overrideLabels, got["labels"])
	require.Equal(t, baseLabels, base["labels"])
	require.Equal(t, overrideLabels, override["labels"])
}

//nolint:goconst // repeated map keys are the fixture being merged
func TestMergeMaps_DoesNotMutateInputs(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		"server": map[string]any{
			"timeout": 30,
			"tls":     false,
		},
	}
	override := map[string]any{
		"server": map[string]any{
			"tls":   true,
			"certs": map[string]any{"ca": "ca.pem"},
		},
	}
	baseBefore := copyNestedMaps(base)
	overrideBefore := copyNestedMaps(override)

	got := merger.MergeMaps(base, override)

	require.Equal(t, baseBefore, base)
	require.Equal(t, overrideBefore, override)

	server := got["server"].(map[string]any)
	require.Equal(t, 30, server["timeout"])
	require.Equal(t, true, server["tls"])
	require.Equal(t, map[string]any{"ca": "ca.pem"}, server["certs"])
}

//nolint:goconst // repeated map keys are the fixture being merged
func TestMergeMaps_DifferentTypesAreSkipped(t *testing.T) {
	t.Parallel()

	got := merger.MergeMaps(
		map[string]any{"port": 8080, "nested": map[string]any{"a": 1}},
		map[string]any{"port": "9090", "nested": "not-a-map"},
	)

	require.Equal(t, map[string]any{"port": 8080, "nested": map[string]any{"a": 1}}, got)
}

// copyNestedMaps snapshots nested map[string]any values so a test can assert
// that MergeMaps left its inputs untouched.
func copyNestedMaps(src map[string]any) map[string]any {
	snapshot := make(map[string]any, len(src))

	for k, v := range src {
		if nested, ok := v.(map[string]any); ok {
			snapshot[k] = copyNestedMaps(nested)
			continue
		}

		snapshot[k] = v
	}

	return snapshot
}

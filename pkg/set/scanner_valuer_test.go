package set_test

import (
	"testing"

	"github.com/guionardo/go/pkg/set"
	"github.com/stretchr/testify/require"
)

func TestSet_ScanValue(t *testing.T) {
	t.Parallel()

	set1 := set.New(1, 2, 3)
	set2 := set.New[int]()

	value, err := set1.Value()
	require.NoError(t, err)

	err = set2.Scan(value)
	require.NoError(t, err)

	require.True(t, set1.Equals(set2))
}

func TestSet_Scan_InvalidType(t *testing.T) {
	t.Parallel()

	set1 := set.New[int]()
	err := set1.Scan(123)
	require.Error(t, err)
	require.Equal(t, "invalid type for scan", err.Error())
}

func TestSet_Scan_StringJSON(t *testing.T) {
	t.Parallel()

	set1 := set.New[string]()
	jsonStr := `["a","b","c"]`
	err := set1.Scan(jsonStr)
	require.NoError(t, err)
	require.True(t, set1.Equals(set.New("a", "b", "c")))
}

func TestSet_Scan_ByteJSON(t *testing.T) {
	t.Parallel()

	set1 := set.New[string]()
	jsonBytes := []byte(`["x","y"]`)
	err := set1.Scan(jsonBytes)
	require.NoError(t, err)
	require.True(t, set1.Equals(set.New("x", "y")))
}

func TestSet_Value_And_Scan_RoundTrip(t *testing.T) {
	t.Parallel()

	original := set.New("foo", "bar")
	value, err := original.Value()
	require.NoError(t, err)

	restored := set.New[string]()
	err = restored.Scan(value)
	require.NoError(t, err)
	require.True(t, original.Equals(restored))
}

func TestSet_Value_EmptySet(t *testing.T) {
	t.Parallel()

	empty := set.New[int]()
	value, err := empty.Value()
	require.NoError(t, err)
	require.NotNil(t, value)

	restored := set.New[int]()
	err = restored.Scan(value)
	require.NoError(t, err)
	require.True(t, empty.Equals(restored))
}

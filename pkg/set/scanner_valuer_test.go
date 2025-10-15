package set_test

import (
	"testing"

	"github.com/guionardo/go/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestSet_ScanValue(t *testing.T) {
	set1 := set.New(1, 2, 3)
	set2 := set.New[int]()

	value, err := set1.Value()
	if !assert.NoError(t, err) {
		return
	}

	err = set2.Scan(value)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, set1.Equals(set2))

}

func TestSet_Scan_InvalidType(t *testing.T) {
	set1 := set.New[int]()
	err := set1.Scan(123)
	assert.Error(t, err)
	assert.Equal(t, "invalid type for scan", err.Error())
}

func TestSet_Scan_StringJSON(t *testing.T) {
	set1 := set.New[string]()
	jsonStr := `["a","b","c"]`
	err := set1.Scan(jsonStr)
	assert.NoError(t, err)
	assert.True(t, set1.Equals(set.New("a", "b", "c")))
}

func TestSet_Scan_ByteJSON(t *testing.T) {
	set1 := set.New[string]()
	jsonBytes := []byte(`["x","y"]`)
	err := set1.Scan(jsonBytes)
	assert.NoError(t, err)
	assert.True(t, set1.Equals(set.New("x", "y")))
}

func TestSet_Value_And_Scan_RoundTrip(t *testing.T) {
	original := set.New("foo", "bar")
	value, err := original.Value()
	assert.NoError(t, err)

	restored := set.New[string]()
	err = restored.Scan(value)
	assert.NoError(t, err)
	assert.True(t, original.Equals(restored))
}

func TestSet_Value_EmptySet(t *testing.T) {
	empty := set.New[int]()
	value, err := empty.Value()
	assert.NoError(t, err)
	assert.NotNil(t, value)

	restored := set.New[int]()
	err = restored.Scan(value)
	assert.NoError(t, err)
	assert.True(t, empty.Equals(restored))
}

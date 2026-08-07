package flow_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/guionardo/go/flow"
	"github.com/stretchr/testify/assert"
)

func TestNewOrderedMap(t *testing.T) {
	t.Parallel()

	m := flow.NewOrderedMap[string, int]()
	m.Set("A", 1)
	m.Set("C", 3)
	m.Set("B", 2)
	m.Set("Z", 26)
	m.Set("Y", 25)

	assert.Equal(t, 1, m.GetDefault("A", 0))
	assert.Equal(t, 3, m.GetDefault("C", 0))

	rangeKeys := strings.Join(slices.Collect(m.Keys(1, 4)), "")
	assert.Equal(t, "CBZ", rangeKeys)

	rangeKeys = strings.Join(slices.Collect(m.Keys(2, -1)), "")
	assert.Equal(t, "BZY", rangeKeys)

	rangeKeys = strings.Join(slices.Collect(m.Keys(-2, -1)), "")
	assert.Equal(t, "ZY", rangeKeys)

	rangeKeys = strings.Join(slices.Collect(m.Keys(3, 2)), "")
	assert.Equal(t, "Z", rangeKeys)

	// Set key again should move the last set to the end of the list
	m.Set("C", 33)
	rangeKeys = strings.Join(slices.Collect(m.Keys(0, -1)), "")
	assert.Equal(t, "ABZYC", rangeKeys)

	assert.Equal(t, 0, m.GetDefault("NOT", 0))

	m.Delete("B")
	rangeKeys = strings.Join(slices.Collect(m.Keys(0, -1)), "")
	assert.Equal(t, "AZYC", rangeKeys)

	result := ""

	var resultSb50 strings.Builder
	for k, v := range m.Range(0, -1) {
		fmt.Fprintf(&resultSb50, "%s=%d ", k, v)
	}

	result += resultSb50.String()

	assert.Equal(t, "A=1 Z=26 Y=25 C=33 ", result)
}

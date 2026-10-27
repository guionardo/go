package flow_test

import (
	"fmt"
	"iter"
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

// collectRange returns the "key=value" pairs yielded by seq in order.
func collectRange(seq iter.Seq2[string, int]) []string {
	pairs := make([]string, 0)

	for key, value := range seq {
		pairs = append(pairs, fmt.Sprintf("%s=%d", key, value))
	}

	return pairs
}

// expectedPairs renders the "key=value" pairs for keys using their insertion index.
func expectedPairs(keys []string, values map[string]int) []string {
	pairs := make([]string, 0, len(keys))

	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%d", key, values[key]))
	}

	return pairs
}

func TestOrderedMapEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		first, last int
	}{
		{name: "zero range", first: 0, last: 0},
		{name: "negative range", first: -1, last: -1},
		{name: "whole range", first: 0, last: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := flow.NewOrderedMap[string, int]()

			assert.Empty(t, slices.Collect(m.Keys(tt.first, tt.last)))
			assert.Empty(t, collectRange(m.Range(tt.first, tt.last)))
		})
	}
}

func TestOrderedMapSingleElement(t *testing.T) {
	t.Parallel()

	const onlyKey = "only"

	m := flow.NewOrderedMap[string, int]()
	m.Set(onlyKey, 1)

	tests := []struct {
		name        string
		first, last int
		expected    []string
	}{
		{name: "zero range", first: 0, last: 0, expected: []string{onlyKey}},
		{name: "negative range", first: -1, last: -1, expected: []string{onlyKey}},
		{name: "through the end", first: 0, last: -1, expected: []string{onlyKey}},
		{name: "reversed range", first: 1, last: 0, expected: []string{onlyKey}},
		{name: "out of bounds", first: 5, last: 9, expected: []string{onlyKey}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := expectedPairs(tt.expected, map[string]int{onlyKey: 1})

			assert.Equal(t, tt.expected, slices.Collect(m.Keys(tt.first, tt.last)))
			assert.Equal(t, want, collectRange(m.Range(tt.first, tt.last)))
		})
	}
}

func TestOrderedMapRangeClamping(t *testing.T) {
	t.Parallel()

	m := flow.NewOrderedMap[string, int]()
	index := make(map[string]int)

	for i, key := range []string{"A", "B", "C", "D", "E"} {
		m.Set(key, i)
		index[key] = i
	}

	tests := []struct {
		name        string
		first, last int
		expected    []string
	}{
		{name: "plain range", first: 0, last: 2, expected: []string{"A", "B"}},
		{name: "negative first counts from the end", first: -2, last: -1, expected: []string{"D", "E"}},
		{name: "last before first yields one key", first: 3, last: 2, expected: []string{"D"}},
		{name: "negative last counts back", first: 1, last: -3, expected: []string{"B", "C"}},
		{name: "first before the start clamps", first: -10, last: 2, expected: []string{"A", "B"}},
		{name: "last past the end clamps", first: 4, last: 100, expected: []string{"E"}},
		{name: "first past the end clamps", first: 10, last: 20, expected: []string{"E"}},
		{name: "whole range", first: 0, last: -1, expected: []string{"A", "B", "C", "D", "E"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := expectedPairs(tt.expected, index)

			assert.Equal(t, tt.expected, slices.Collect(m.Keys(tt.first, tt.last)))
			assert.Equal(t, want, collectRange(m.Range(tt.first, tt.last)))
		})
	}
}

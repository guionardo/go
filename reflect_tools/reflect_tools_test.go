package reflecttools_test

import (
	"testing"
	"time"

	reflecttools "github.com/guionardo/go/reflect_tools"
	"github.com/stretchr/testify/assert"
)

func TestIsZeroValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		valueZero    any
		valueNonZero any
	}{
		{name: "nil value", valueZero: nil, valueNonZero: &struct{ name string }{}},
		{name: "int", valueZero: 0, valueNonZero: 1},
		{name: "int8", valueZero: int8(0), valueNonZero: int8(1)},
		{name: "int16", valueZero: int16(0), valueNonZero: int16(1)},
		{name: "int32", valueZero: int32(0), valueNonZero: int32(1)},
		{name: "int64", valueZero: int64(0), valueNonZero: int64(1)},
		{name: "uint", valueZero: uint(0), valueNonZero: uint(1)},
		{name: "uint8", valueZero: uint8(0), valueNonZero: uint8(1)},
		{name: "uint16", valueZero: uint16(0), valueNonZero: uint16(1)},
		{name: "uint32", valueZero: uint32(0), valueNonZero: uint32(1)},
		{name: "uint64", valueZero: uint64(0), valueNonZero: uint64(1)},
		{name: "float32", valueZero: float32(0.0), valueNonZero: float32(1.0)},
		{name: "float64", valueZero: float64(0.0), valueNonZero: float64(1.0)},
		{name: "boolean", valueZero: false, valueNonZero: true},
		{name: "string", valueZero: "", valueNonZero: "non-empty"},
		{name: "time.Time", valueZero: time.Time{}, valueNonZero: time.Now()},
		{name: "time.Duration", valueZero: time.Duration(0), valueNonZero: time.Second},
		{name: "map", valueZero: map[string]any{}, valueNonZero: map[string]any{"key": "value"}},
		{name: "slice", valueZero: []any{}, valueNonZero: []any{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(
				t,
				reflecttools.IsZeroValue(tt.valueZero),
				"expected zero value from %T %v",
				tt.valueZero,
				tt.valueZero,
			)
			assert.False(
				t,
				reflecttools.IsZeroValue(tt.valueNonZero),
				"expected non-zero value from %T %v",
				tt.valueNonZero,
				tt.valueNonZero,
			)
		})
	}
}

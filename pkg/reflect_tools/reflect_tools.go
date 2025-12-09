package reflecttools

import (
	"reflect"
	"time"
)

// IsZeroValue checks if the provided value is considered a zero value.
// It handles various types including numeric types, strings, booleans,
// time.Time, time.Duration, slices, arrays, maps, and pointers.
// Returns true if the value is zero, nil or empty,  false otherwise.
func IsZeroValue(value any) bool { //nolint:cyclop,funlen
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		bool, string:
		return reflect.ValueOf(value).IsZero()

	case time.Time:
		return v.IsZero()
	case time.Duration:
		return v == 0

	default:
		t := reflect.TypeOf(value)
		if t.Kind() == reflect.Array || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
			return reflect.ValueOf(value).Len() == 0
		}

		return false
	}
}

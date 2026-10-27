package merger

import "reflect"

// MergeMaps performs a recursive deep-merge of multiple maps.
// Each subsequent map's values are merged into the accumulator. Nested
// map[string]any values are merged recursively and deep-copied, so the inputs
// are never mutated. Any other value replaces an earlier one only when both
// have the same dynamic type, otherwise the later value is skipped and the
// earlier one is kept. Two nested maps that share a type but are not
// map[string]any are replaced rather than merged.
func MergeMaps(maps ...map[string]any) map[string]any {
	current := make(map[string]any)
	for _, m := range maps {
		updateMapValues(current, m)
	}

	return current
}

// updateMapValues updates the values of a map with the values of another map.
func updateMapValues(current, from map[string]any) {
	if current == nil {
		current = make(map[string]any)
	}

	for k, v := range from {
		currentValue, ok := current[k]
		if !ok {
			current[k] = copyValue(v)
			continue
		}

		if reflect.TypeOf(currentValue) != reflect.TypeOf(v) {
			continue
		}

		if nested, isMap := currentValue.(map[string]any); isMap {
			updateMapValues(nested, v.(map[string]any))
			continue
		}

		current[k] = copyValue(v)
	}
}

// copyValue deep-copies nested map[string]any values so later merges cannot
// write through to the caller's maps. Other values are returned as-is.
func copyValue(v any) any {
	nested, ok := v.(map[string]any)
	if !ok {
		return v
	}

	copied := make(map[string]any, len(nested))
	for k, item := range nested {
		copied[k] = copyValue(item)
	}

	return copied
}

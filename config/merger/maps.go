package merger

import "reflect"

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
			current[k] = v
			continue
		}

		if reflect.TypeOf(currentValue) != reflect.TypeOf(v) {
			continue
		}

		if reflect.ValueOf(currentValue).Kind() == reflect.Map {
			updateMapValues(currentValue.(map[string]any), v.(map[string]any))
			continue
		}

		current[k] = v
	}
}

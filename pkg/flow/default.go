package flow

// Default returns valueIfZero if value equals the zero value for type T, otherwise returns value.
func Default[T comparable](value T, valueIfZero T) T {
	var zero T
	if value == zero {
		return valueIfZero
	}
	return value
}

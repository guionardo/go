package flow

// Default returns the second argument (valueIfZero) when the value has the default (zero)
func Default[T comparable](value T, valueIfZero T) T {
	var zero T
	if value == zero {
		return valueIfZero
	}
	return value
}

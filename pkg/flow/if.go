package flow

// If is a generic ternary operator
func If[T any](condition bool, valueIfTrue T, valueIfFalse T) T {
	if condition {
		return valueIfTrue
	}
	return valueIfFalse
}

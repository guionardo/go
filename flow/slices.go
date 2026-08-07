package flow

func SliceFirstOrDefault[T any](s []T, defaultValue T) T {
	if len(s) > 0 {
		return s[0]
	}

	return defaultValue
}

func SliceLastOrDefault[T any](s []T, defaultValue T) T {
	if len(s) > 0 {
		return s[len(s)-1]
	}

	return defaultValue
}

func RemoveItem[T any](s []T, index int) []T {
	if index < 0 || index >= len(s) {
		return s
	}

	if index == len(s)-1 {
		return s[:index]
	}

	return append(s[:index], s[index+1:]...)
}

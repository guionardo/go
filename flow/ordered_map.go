package flow

import (
	"iter"
	"slices"
)

type OrderedMap[T comparable, S any] struct {
	keys []T
	data map[T]S
}

func NewOrderedMap[T comparable, S any]() *OrderedMap[T, S] {
	return &OrderedMap[T, S]{
		keys: make([]T, 0),
		data: make(map[T]S),
	}
}
func (om *OrderedMap[T, S]) Set(key T, value S) {
	om.data[key] = value
	om.keys = append(RemoveItem(om.keys, slices.Index(om.keys, key)), key)
}

func (om *OrderedMap[T, S]) Get(key T) (value S, ok bool) {
	value, ok = om.data[key]
	return
}

func (om *OrderedMap[T, S]) GetDefault(key T, defaultValue S) S {
	if value, ok := om.Get(key); ok {
		return value
	}

	return defaultValue
}

func (om *OrderedMap[T, S]) Delete(key T) {
	delete(om.data, key)
	om.keys = RemoveItem(om.keys, slices.Index(om.keys, key))
}
func (om *OrderedMap[T, S]) Keys(first, last int) iter.Seq[T] {
	first, last = om.validateRange(first, last)

	return func(yield func(T) bool) {
		for _, key := range om.keys[first:last] {
			if !yield(key) {
				return
			}
		}
	}
}

// Range iterates by the ordered key/values
// first should be between 0 and data length -1. If first < 0, it will be considered from the end of data
func (om *OrderedMap[T, S]) Range(first, last int) iter.Seq2[T, S] {
	return func(yield func(T, S) bool) {
		for key := range om.Keys(first, last) {
			if entry, ok := om.data[key]; ok && !yield(key, entry) {
				return
			}
		}
	}
}

func (om *OrderedMap[T, S]) Len() int {
	return len(om.data)
}

func (om *OrderedMap[T, S]) validateRange(first, last int) (int, int) {
	if first < 0 {
		first = max(0, len(om.keys)+first)
	}

	first = min(len(om.keys)-1, first)
	if last < 0 {
		last = max(0, len(om.keys)+last+1)
	} else if last <= first {
		last = first + 1
	}

	last = min(last, len(om.keys))

	return first, last
}

package set

import (
	"iter"
)

// Set values methods
type Set[T comparable] map[T]struct{}

var emptyStruct = struct{}{}

// New create a set with optional values
func New[T comparable](values ...T) Set[T] {
	s := make(Set[T], len(values))
	s.AddMultiple(values...)
	return s
}

// Add value to set
func (s Set[T]) Add(v T) {
	s[v] = emptyStruct
}

// Add multiple values to set
func (s Set[T]) AddMultiple(values ...T) Set[T] {
	for _, value := range values {
		s.Add(value)
	}
	return s
}

// Union two sets to create a new set with all values
func (s Set[T]) Union(another Set[T]) Set[T] {
	out := New(s.ToArray()...).UpdateFrom(another)
	return out
}

// Diff results a Set with values that are not common to two sets
func (s Set[T]) Diff(another Set[T]) Set[T] {
	out := Set[T]{}
	for v := range s.Iter() {
		if !another.Has(v) {
			out.Add(v)
		}
	}
	for v := range another.Iter() {
		if !s.Has(v) {
			out.Add(v)
		}
	}
	return out
}

// Intersection results a Set with values common to the two sets
func (s Set[T]) Intersection(another Set[T]) Set[T] {
	out := Set[T]{}
	for v := range s {
		if another.Has(v) {
			out.Add(v)
		}
	}
	return out
}

// Iter returns a iterable of the values in the set
// Due map characteristics, the order is not garanteeded
func (s Set[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s {
			if !yield(k) {
				return
			}
		}
	}
}

// UpdateFrom adds the values from another set to current
func (s Set[T]) UpdateFrom(another Set[T]) Set[T] {
	for k := range another.Iter() {
		s.Add(k)
	}
	return s
}

// ToArray returns an unsorted array with the values of the set
func (s Set[T]) ToArray() []T {
	a := make([]T, 0, len(s))
	for v := range s.Iter() {
		a = append(a, v)
	}
	return a
}

// Has returns true if the value is in the set
func (s Set[T]) Has(value T) bool {
	_, ok := s[value]
	return ok
}

// HasAll returns true if all the values are in the set
func (s Set[T]) HasAll(values ...T) bool {
	for _, v := range values {
		if !s.Has(v) {
			return false
		}
	}
	return true
}

// Filter returns an iterable with the values that satisfies the filter condition
func (s Set[T]) Filter(filter func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s {
			if filter(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Equals returns true if the set has exactly the same values of the another set
func (s Set[T]) Equals(another Set[T]) bool {
	if len(s) != len(another) {
		return false
	}
	for v := range s {
		if !another.Has(v) {
			return false
		}
	}
	return true
}

// Clear empties all itens
func (s Set[T]) Clear() {
	keys := s.ToArray()
	for _, key := range keys {
		delete(s, key)
	}
}

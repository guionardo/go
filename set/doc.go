// Package set provides a generic Set[T comparable] implementation.
//
// Set[T] is backed by a map[T]struct{} and supports:
//   - Add, Remove, Has, HasAll, Clear
//   - Union, Diff, Intersection, UpdateFrom (set algebra)
//   - Iter, Filter (iter.Seq[T] iteration)
//   - ToArray, Equals
//   - MarshalJSON / UnmarshalJSON
//   - database/sql.Scanner and driver.Valuer (Scan / Value)
//
// Usage:
//
//	s := set.New(1, 2, 3)
//	s.Add(4)
//	subset := s.Filter(func(v int) bool { return v > 2 })
package set

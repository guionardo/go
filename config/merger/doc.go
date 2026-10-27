// Package merger provides recursive deep-merge of map[string]any.
//
// Maps are folded left to right into a fresh accumulator. A key that exists only
// in a later map is added; a key already present is replaced only when the new
// value has exactly the same dynamic type (reflect.TypeOf) — a later value of a
// different type is silently skipped, so the earlier value survives. Nested
// map[string]any values are merged recursively and deep-copied, so neither input
// is mutated by the call. Nested maps of any other type cannot be merged: they
// are replaced as a whole instead of panicking.
//
// Usage:
//
//	merged := merger.MergeMaps(base, override)
package merger

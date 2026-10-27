// Package merger provides recursive deep-merge of map[string]any.
//
// Maps are folded left to right into a fresh accumulator. A key that exists only
// in a later map is added; a key already present is replaced only when the new
// value has exactly the same dynamic type (reflect.TypeOf) — a later value of a
// different type is silently skipped, so the earlier value survives. Nested maps
// are merged recursively, and only values that are exactly map[string]any are
// merged this way: any other map type panics on the type assertion.
//
// Usage:
//
//	merged := merger.MergeMaps(base, override)
package merger

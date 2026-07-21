// Package merger provides recursive deep-merge of map[string]any.
//
// Later maps overwrite earlier ones for non-map values.
// Nested maps are merged recursively.
//
// Usage:
//
//	merged := merger.MergeMaps(base, override)
package merger

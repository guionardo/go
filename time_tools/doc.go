// Package timetools provides adaptive time parsing.
//
// Parse tries multiple common time layouts (RFC3339, DateTime, DateOnly,
// Kitchen, ANSIC, etc.) in priority order. Successful parses promote the
// matched layout to the front for faster subsequent parsing.
//
// Functions:
//   - Parse: parse a time string using adaptive layout matching
//   - SetLayouts: replace the default layout list with a custom priority order
//
// Thread-safe.
package timetools

// Package shelltools provides shell environment interaction utilities.
//
// Types:
//   - QuotedShellArgs ([]string): parsed shell arguments that handle quoting
//
// Functions:
//   - GetEnv: case-insensitive environment variable lookup
//   - NewQuotedShellArgs: parse a string into QuotedShellArgs (supports ' " and \ escaping)
//
// Methods on QuotedShellArgs:
//   - String: reconstruct a shell-safe quoted string
//
// Example:
//
//	args := shelltools.NewQuotedShellArgs(`one "two three" 'four five'`)
//	// args[0] = "one", args[1] = "two three", args[2] = "four five"
package shelltools

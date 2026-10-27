// Package shelltools provides shell environment interaction utilities.
//
// Types:
//   - QuotedShellArgs ([]string): parsed shell arguments that handle quoting
//
// Functions:
//   - GetEnv: environment variable lookup backed by os.LookupEnv — the name is
//     matched exactly (case-sensitively on Unix, case-insensitively on Windows
//     because the OS is), and an unset variable yields ""
//   - NewQuotedShellArgs: parse a string into QuotedShellArgs; single and double
//     quotes are honoured, but the only backslash sequence interpreted is "\ "
//     (an escaped space) — `"hello\"world"` splits at the quote instead of
//     escaping it
//
// Methods on QuotedShellArgs:
//   - String: reconstruct a shell-safe quoted string
//
// Example:
//
//	args := shelltools.NewQuotedShellArgs(`one "two three" 'four five'`)
//	// args[0] = "one", args[1] = "two three", args[2] = "four five"
package shelltools

// Package reflecttools provides reflection utilities.
//
// Functions:
//   - IsZeroValue: reports whether a value is zero. A nil interface is true;
//     the exact built-in numeric types, strings and bools use reflect's IsZero;
//     time.Time and time.Duration have dedicated checks; arrays, slices and maps
//     are true when empty. Everything else falls through to false — including
//     nil pointers, structs other than time.Time, and named types such as
//     `type MyInt int`, because only the exact built-in types match.
package reflecttools

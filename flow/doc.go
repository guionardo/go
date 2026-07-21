// Package flow provides generic control flow utilities.
//
// Functions:
//   - If[T]: generic ternary operator — returns valueIfTrue or valueIfFalse based on condition
//   - Default[T]: zero-value fallback — returns valueIfZero when value is the zero value for its type
//
// Example:
//
//	max := flow.If(x > y, x, y)
//	name := flow.Default(input, "defaultName")
package flow

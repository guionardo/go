// Package fraction provides an immutable fraction type with arithmetic operations.
//
// Originally based on github.com/nethruster/go-fraction by Miguel Dorta.
//
// A Fraction is always valid (never zero denominator) and always simplified
// to lowest terms via GCD. Operations return new Fraction values — the original
// is never modified.
//
// Key types:
//   - Fraction: immutable fraction with Numerator() and Denominator() accessors
//   - integer: generic constraint for int/int8..int64/uint/uint8..uint64
//
// Functions:
//   - New[T, K integer]: create a fraction from a numerator and denominator
//   - FromFloat64: create a fraction approximating a float64 value
//
// Methods on Fraction:
//   - Add, Subtract, Multiply, Divide: arithmetic
//   - Equal: equality comparison
//   - Float64: convert to float64
package fraction

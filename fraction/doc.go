// Package fraction provides an immutable fraction type with arithmetic operations.
//
// Originally based on github.com/nethruster/go-fraction by Miguel Dorta.
//
// Values returned by New, FromFloat64, Multiply and Divide are in lowest terms
// with a non-zero denominator. Add and Subtract are the exception: they return
// the common-denominator result unreduced, so 1/3 + 1/6 is 3/6, not 1/2.
// Because Equal compares numerator and denominator literally, an unreduced
// result is not Equal to its reduced form — compare Float64 values (or
// cross-multiply) when a fraction may have come out of Add or Subtract.
//
// The zero value Fraction{} is 0/0, which is not a valid fraction: Float64()
// returns NaN, Multiply returns 0/1 because it discards New's error, and Divide
// returns ErrDivideByZero. Add and Subtract treat a zero denominator as 0/1, so
// the zero value behaves as zero in those two operations. Prefer building
// fractions with New or FromFloat64. Operations return new Fraction values — the
// original is never modified.
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
//   - Add, Subtract, Multiply, Divide: arithmetic (Divide reports ErrDivideByZero)
//   - Equal: equality comparison of numerator and denominator
//   - Float64: convert to float64
package fraction

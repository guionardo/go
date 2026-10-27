# fraction

An immutable fraction type backed by `int64`, with arithmetic that returns new values.
Reach for it when you need exact rational arithmetic with small numbers without the
overhead of `math/big.Rat`.

Based on [go-fraction](https://github.com/nethruster/go-fraction) by Miguel Dorta, carried
here because of release-name restrictions on the original module.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/fraction)

## Install

```bash
go get github.com/guionardo/go/fraction
```

## Usage

```go
import "github.com/guionardo/go/fraction"

f1, err := fraction.New(1, 2) // 1/2, nil
f2, err := fraction.New(2, 3) // 2/3, nil
_, err = fraction.New(1, 0)   // ErrZeroDenominator

f3 := f1.Add(f2)          // 7/6
f4 := f1.Subtract(f2)     // -1/6
f5 := f1.Multiply(f2)     // 1/3
f6, err := f1.Divide(f2)  // 3/4, nil

value := f1.Float64()               // 0.5
f7, err := fraction.FromFloat64(0.75) // 3/4, nil
```

Type parameters are inferred, so `fraction.New(1, 2)` and
`fraction.New[int, int](1, 2)` are equivalent.

## API

```go
type Fraction struct{} // immutable

func New[T, K integer](numerator T, denominator K) (Fraction, error)
func FromFloat64(f float64) (Fraction, error)

func (f Fraction) Add(other Fraction) Fraction
func (f Fraction) Subtract(other Fraction) Fraction
func (f Fraction) Multiply(other Fraction) Fraction
func (f Fraction) Divide(other Fraction) (Fraction, error)
func (f Fraction) Equal(other Fraction) bool
func (f Fraction) Float64() float64
func (f Fraction) Numerator() int64
func (f Fraction) Denominator() int64
```

`integer` covers `int`, `int8`–`int64`, and `uint`–`uint64`.

## Behaviour

- `New` reduces to lowest terms via GCD and moves the sign to the numerator, so the
  denominator is always positive when it is non-zero.
- `Multiply`, `Divide` and `FromFloat64` also return reduced fractions, because they
  construct their result through `New`.
- `Add` and `Subtract` return the common-denominator result **as-is**, without reduction:
  `1/3 + 1/6` yields `3/6`, not `1/2`. Re-normalise by passing the result back through
  `New(f.Numerator(), f.Denominator())` if you depend on canonical form.
- `Equal` compares the stored numerator and denominator, not the value, so `3/6` is not
  equal to `1/2`. Because of the previous point, compare reduced values only.
- The zero value `Fraction{}` has a `0` denominator and reports `0/0` as its parts;
  `Float64` on it returns `NaN`. `Add` and `Subtract` treat a zero denominator as
  `0/1`, so the zero value behaves as zero there: adding `1/2` to `Fraction{}` yields
  `1/2`. Prefer constructing with `New` or `FromFloat64`.

## Errors

```go
var ErrZeroDenominator = errors.New("denominator cannot be zero") // New with denominator 0
var ErrDivideByZero   = errors.New("denominator cannot be zero") // Divide by a zero fraction
var ErrInvalid        = errors.New("invalid conversion")          // FromFloat64(NaN)
var ErrOutOfRange     = errors.New("the number is out of range for this library")
```

On error the zero fraction (`0/1`) is returned alongside the error.

## Limitations

The type stores `int64` numerator and denominator, which keeps it lightweight and usable
as a primitive-like value, but means arithmetic on large fractions can overflow. Floats
also cover a wider range than `int64` can express, so `FromFloat64` returns
`ErrOutOfRange` for extreme values and may lose precision for others because a `float64`
is itself an approximation. If you need arbitrary-size rationals regardless of memory and
CPU cost, use `math/big.Rat` instead.

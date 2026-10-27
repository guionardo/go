package fraction_test

import (
	"errors"
	"math"
	"testing"

	"github.com/guionardo/go/fraction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fatalIfErr(t *testing.T, err error) {
	t.Helper()
	assert.NoError(t, err)
}

func compare(t *testing.T, f fraction.Fraction, numerator, denominator int64) {
	t.Helper()
	assert.Equalf(t, numerator, f.Numerator(), "expected numerator value to be %v, got %v", numerator, f.Numerator())
	assert.Equal(
		t,
		denominator,
		f.Denominator(),
		"expected denominator value to be %v, got %v",
		denominator,
		f.Denominator(),
	)
}

func approxFloat(t *testing.T, fr fraction.Fraction, expected float64) {
	t.Helper()

	const epsilonFactor = 1e-9

	fl := fr.Float64()
	require.InEpsilonf(
		t,
		expected,
		fl,
		epsilonFactor,
		"expected fraction around %v with %v of error, got %v",
		expected,
		epsilonFactor,
		fl,
	)
}

func TestNew(t *testing.T) {
	t.Parallel()

	_, err := fraction.New(-3, 5)
	require.NoError(t, err)
	_, err = fraction.New(int32(1), uint16(2))
	require.NoError(t, err)
	_, err = fraction.New(0, 2)
	require.NoError(t, err)

	_, err = fraction.New(1, 0)
	require.ErrorIs(t, err, fraction.ErrZeroDenominator)
	_, err = fraction.New(0, 0)
	require.ErrorIs(t, err, fraction.ErrZeroDenominator)
}

func TestNewSimplify(t *testing.T) {
	t.Parallel()

	f, err := fraction.New(402, 21)
	require.NoError(t, err)
	compare(t, f, 134, 7)

	f, err = fraction.New(-10, 20)
	require.NoError(t, err)
	compare(t, f, -1, 2)

	f, err = fraction.New(6, -9)
	require.NoError(t, err)
	compare(t, f, -2, 3)

	f, err = fraction.New(-44, -11)
	require.NoError(t, err)
	compare(t, f, 4, 1)

	f, err = fraction.New(0, 9)
	require.NoError(t, err)
	compare(t, f, 0, 1)

	f, err = fraction.New(0, -6)
	require.NoError(t, err)
	compare(t, f, 0, 1)
}

func TestEquals(t *testing.T) {
	t.Parallel()

	f1, _ := fraction.New(-19, 27)
	f2, _ := fraction.New(57, -81)
	f3, _ := fraction.New(-57, -81)

	require.True(t, f1.Equal(f2), "expected both fractions (-19/27) to be equal, got not equal")
	require.False(t, f1.Equal(f3), "expected fraction -19/27 not to be equal to 19/27, got equal")

	f1, _ = fraction.New(0, 23)

	f2, _ = fraction.New(0, 2)
	require.True(t, f1.Equal(f2), "expected both fractions (0/1) to be equal, got not equal")
}

func TestAdd(t *testing.T) {
	t.Parallel()

	f1, _ := fraction.New(6, 36)
	f2, _ := fraction.New(14, 18)
	compare(t, f1.Add(f2), 17, 18)

	f1, _ = fraction.New(26, 33)
	f2, _ = fraction.New(49, -27)
	compare(t, f1.Add(f2), -305, 297)

	f1, _ = fraction.New(49, 42)
	f2, _ = fraction.New(0, -29)
	compare(t, f1.Add(f2), 7, 6)
}

func TestSubtract(t *testing.T) {
	t.Parallel()

	f1, _ := fraction.New(6, 36)
	f2, _ := fraction.New(14, 18)
	compare(t, f1.Subtract(f2), -11, 18)

	f1, _ = fraction.New(26, 33)
	f2, _ = fraction.New(-49, 27)
	compare(t, f1.Subtract(f2), 773, 297)

	f1, _ = fraction.New(49, 42)
	f2, _ = fraction.New(0, -29)
	compare(t, f1.Subtract(f2), 7, 6)

	f1, _ = fraction.New(-12, 22)
	f2, _ = fraction.New(47, -5)
	compare(t, f1.Subtract(f2), 487, 55)
}

// TestAddSubtractExact pins the common-denominator, unreduced results of Add and
// Subtract so the zero-value handling cannot change them.
func TestAddSubtractExact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		n1, d1, n2, d2         int64
		sumNum, sumDen         int64
		differenceNum, diffDen int64
	}{
		{
			name: "thirds and sixths", n1: 1, d1: 3, n2: 1, d2: 6,
			sumNum: 3, sumDen: 6, differenceNum: 1, diffDen: 6,
		},
		{
			name: "unlike denominators", n1: 26, d1: 33, n2: -49, d2: 27,
			sumNum: -305, sumDen: 297, differenceNum: 773, diffDen: 297,
		},
		{
			name: "same denominator", n1: 7, d1: 9, n2: 2, d2: 9,
			sumNum: 9, sumDen: 9, differenceNum: 5, diffDen: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f1, err := fraction.New(tt.n1, tt.d1)
			require.NoError(t, err)
			f2, err := fraction.New(tt.n2, tt.d2)
			require.NoError(t, err)

			compare(t, f1.Add(f2), tt.sumNum, tt.sumDen)
			compare(t, f1.Subtract(f2), tt.differenceNum, tt.diffDen)
		})
	}
}

func TestZeroValueAddSubtract(t *testing.T) {
	t.Parallel()

	var zero fraction.Fraction

	half, err := fraction.New(1, 2)
	require.NoError(t, err)
	third, err := fraction.New(1, 3)
	require.NoError(t, err)

	tests := []struct {
		name      string
		got       fraction.Fraction
		expectedN int64
		expectedD int64
	}{
		{name: "zero + 1/2", got: zero.Add(half), expectedN: 1, expectedD: 2},
		{name: "1/2 + zero", got: half.Add(zero), expectedN: 1, expectedD: 2},
		{name: "zero - 1/2", got: zero.Subtract(half), expectedN: -1, expectedD: 2},
		{name: "1/2 - zero", got: half.Subtract(zero), expectedN: 1, expectedD: 2},
		{name: "zero + 1/3", got: zero.Add(third), expectedN: 1, expectedD: 3},
		{name: "zero + zero", got: zero.Add(zero), expectedN: 0, expectedD: 1},
		{name: "zero - zero", got: zero.Subtract(zero), expectedN: 0, expectedD: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			compare(t, tt.got, tt.expectedN, tt.expectedD)
		})
	}
}

func TestMultiply(t *testing.T) {
	t.Parallel()

	f1, _ := fraction.New(49, 14)
	f2, _ := fraction.New(7, 15)
	compare(t, f1.Multiply(f2), 49, 30)

	f1, _ = fraction.New(26, 33)
	f2, _ = fraction.New(0, 27)
	compare(t, f1.Multiply(f2), 0, 1)

	f1, _ = fraction.New(48, 9)
	f2, _ = fraction.New(6, -16)
	compare(t, f1.Multiply(f2), -2, 1)
}

func TestDivide(t *testing.T) {
	t.Parallel()

	f1, _ := fraction.New(49, 14)
	f2, _ := fraction.New(7, 15)
	result, err := f1.Divide(f2)
	fatalIfErr(t, err)
	compare(t, result, 15, 2)

	f1, _ = fraction.New(26, 33)

	f2, _ = fraction.New(0, 27)
	if _, err = f1.Divide(f2); !errors.Is(err, fraction.ErrDivideByZero) {
		t.Fatalf("expected ErrDivideByZero, got %v", err)
	}

	f1, _ = fraction.New(48, 9)
	f2, _ = fraction.New(6, -16)
	result, err = f1.Divide(f2)
	fatalIfErr(t, err)
	compare(t, result, -128, 9)
}

func TestFloat64(t *testing.T) {
	t.Parallel()

	f, _ := fraction.New(49, 14)
	if f.Float64() != 3.5 {
		t.Fatalf("expected 3.5, got %v", f.Float64())
	}

	f, _ = fraction.New(0, -27)
	if f.Float64() != 0 {
		t.Fatalf("expected 0, got %v", f.Float64())
	}

	f, _ = fraction.New(8, -64)
	if f.Float64() != -0.125 {
		t.Fatalf("expected -0.125, got %v", f.Float64())
	}
}

func TestFromFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       float64
		expectedNum int64
		expectedDen int64
	}{
		{"zero", 0, 0, 1},
		{"negative zero", -0, 0, 1},
		{"one", 1, 1, 1},
		{"negative one", -1, -1, 1},
		{"positive simple fraction", 1.25, 5, 4},
		{"negative simple fraction", -1.25, -5, 4},
		{"large positive integer", 4.5e10, 45000000000, 1},
		{"large negative integer", -4.5e10, -45000000000, 1},
		// Max number that float64 can represent that fits in an int64.
		// Confusingly, printing this float returns 9223372036854775000, but this is an approximation, because if we do
		// the correct conversion based on the binary data following the IEEE 754 standard, we can see that the number
		// that the float holds it's 2^62 * 1.1111111111111111111111111111111111111111111111111111 (base 2), which is
		// exactly 9223372036854774784.
		{"max float64 that fits in int64", 9223372036854774784, 9223372036854774784, 1},
		{"min float64 that fits in int64", -9223372036854774784, -9223372036854774784, 1},
		{"small positive fraction", math.Pow(2, -62), 1, 1 << 62},
		{"small negative fraction", math.Pow(2, -62) * (-1), -1, 1 << 62},
		{"small positive fraction (subnormal)", math.Pow(2, -63), 0, 1},
		{"small negative fraction (subnormal)", math.Pow(2, -63) * (-1), 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := fraction.FromFloat64(tt.input)
			fatalIfErr(t, err)
			compare(t, f, tt.expectedNum, tt.expectedDen)
		})
	}
}

func TestFromFloat64Epsilon(t *testing.T) {
	t.Parallel()
	// 4.5e-10 cannot be represented in a float64, the closest representation is
	// 2^(-32) * 1.1110111011000111101111010101000100101011010101110010 (base 2), which is
	// 4.4999999999999999700744318526239758082585495913008344359695911407470703125 * 10^(-10). The fractions in this
	// library cannot represent real numbers with arbitrary precision, so it will approximate the result.
	f, err := fraction.FromFloat64(4.5e-10)
	fatalIfErr(t, err)

	approxFloat(t, f, 4.5e-10)

	f, err = fraction.FromFloat64(-4.5e-10)
	fatalIfErr(t, err)
	approxFloat(t, f, -4.5e-10)
}

func TestFromFloat64Errors(t *testing.T) {
	t.Parallel()

	var err error

	if _, err = fraction.FromFloat64(9223372036854776000); !errors.Is(err, fraction.ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}

	if _, err = fraction.FromFloat64(-9223372036854776000); !errors.Is(err, fraction.ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}

	if _, err = fraction.FromFloat64(math.Inf(1)); !errors.Is(err, fraction.ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}

	if _, err = fraction.FromFloat64(math.Inf(-1)); !errors.Is(err, fraction.ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}

	if _, err = fraction.FromFloat64(math.NaN()); !errors.Is(err, fraction.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

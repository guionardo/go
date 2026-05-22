package fraction_test

import (
	"fmt"

	"github.com/guionardo/go/fraction"
)

func ExampleNew() {
	f, _ := fraction.New[int, int](1, 2)
	fmt.Printf("%d/%d", f.Numerator(), f.Denominator())

	// Output:
	// 1/2
}

func ExampleFromFloat64() {
	f, _ := fraction.FromFloat64(0.75)
	fmt.Printf("%d/%d", f.Numerator(), f.Denominator())

	// Output:
	// 3/4
}

func ExampleFraction_Add() {
	a, _ := fraction.New[int, int](1, 3)
	b, _ := fraction.New[int, int](1, 6)
	sum := a.Add(b)
	fmt.Printf("%d/%d", sum.Numerator(), sum.Denominator())

	// Output:
	// 3/6
}

func ExampleFraction_Multiply() {
	a, _ := fraction.New[int, int](2, 3)
	b, _ := fraction.New[int, int](3, 4)
	product := a.Multiply(b)
	fmt.Printf("%d/%d", product.Numerator(), product.Denominator())

	// Output:
	// 1/2
}

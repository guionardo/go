package set_test

import (
	"fmt"

	"github.com/guionardo/go/set"
)

func ExampleNew() {
	s := set.New(1, 2, 3, 2, 1)
	fmt.Println(s.Has(1))
	fmt.Println(s.Has(4))

	// Output:
	// true
	// false
}

func ExampleSet_Union() {
	a := set.New(1, 2, 3)
	b := set.New(3, 4, 5)
	union := a.Union(b)
	fmt.Println(union.Has(1))
	fmt.Println(union.Has(5))
	fmt.Println(union.Has(6))

	// Output:
	// true
	// true
	// false
}

func ExampleSet_Intersection() {
	a := set.New(1, 2, 3)
	b := set.New(3, 4, 5)
	intersection := a.Intersection(b)
	fmt.Println(intersection.ToArray())

	// Output:
	// [3]
}

func ExampleSet_Diff() {
	a := set.New(1, 2, 3, 4)
	b := set.New(3, 4, 5, 6)
	diff := a.Diff(b)
	fmt.Println(len(diff.ToArray()))
	fmt.Println(diff.Has(1))
	fmt.Println(diff.Has(5))

	// Output:
	// 4
	// true
	// true
}

func ExampleSet_Filter() {
	s := set.New(1, 2, 3, 4, 5)

	evens := s.Filter(func(v int) bool { return v%2 == 0 })
	for v := range evens {
		fmt.Println(v)
	}

	// Unordered output:
	// 2
	// 4
}

func ExampleSet_Clear() {
	s := set.New(1, 2, 3)
	s.Clear()
	fmt.Println(s.Has(1))
	fmt.Println(len(s))

	// Output:
	// false
	// 0
}

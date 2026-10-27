package reflecttools_test

import (
	"fmt"
	"time"

	reflecttools "github.com/guionardo/go/reflect_tools"
)

// ExampleIsZeroValue walks through the types the function recognises directly.
func ExampleIsZeroValue() {
	fmt.Println(reflecttools.IsZeroValue(nil))
	fmt.Println(reflecttools.IsZeroValue(0))
	fmt.Println(reflecttools.IsZeroValue(""))
	fmt.Println(reflecttools.IsZeroValue(false))
	fmt.Println(reflecttools.IsZeroValue(time.Time{}))
	fmt.Println(reflecttools.IsZeroValue(time.Duration(0)))
	fmt.Println(reflecttools.IsZeroValue([]int{}))
	fmt.Println(reflecttools.IsZeroValue(map[string]int{}))
	fmt.Println(reflecttools.IsZeroValue(42))

	// Output:
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// false
}

// ExampleIsZeroValue_limitations shows where the type switch stops: a named type
// is not one of the recognised built-ins even when it is empty, and a nil pointer
// is not treated as zero. Converting to the underlying type restores detection.
func ExampleIsZeroValue_limitations() {
	type Status string

	var missing *int

	fmt.Println(reflecttools.IsZeroValue(Status("")))
	fmt.Println(reflecttools.IsZeroValue(missing))
	fmt.Println(reflecttools.IsZeroValue(string(Status(""))))

	// Output:
	// false
	// false
	// true
}

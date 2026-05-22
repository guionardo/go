package timetools_test

import (
	"fmt"
	"time"

	timetools "github.com/guionardo/go/time_tools"
)

func ExampleParse() {
	t, err := timetools.Parse("2024-01-15")
	fmt.Println(t.Format("2006-01-02"), err)

	t2, err2 := timetools.Parse("not-a-date")
	fmt.Println(t2.IsZero(), err2 != nil)

	// Output:
	// 2024-01-15 <nil>
	// true true
}

func ExampleParse_rfc3339() {
	t, err := timetools.Parse("2024-01-15T14:30:00Z")
	fmt.Println(t.Format(time.RFC3339), err)

	// Output:
	// 2024-01-15T14:30:00Z <nil>
}

func ExampleSetLayouts() {
	timetools.SetLayouts([]string{"2006/01/02"})

	t, err := timetools.Parse("2024/01/15")
	fmt.Println(t.Format("2006-01-02"), err)

	// Output:
	// 2024-01-15 <nil>
}

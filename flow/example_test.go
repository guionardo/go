package flow_test

import (
	"fmt"

	"github.com/guionardo/go/flow"
)

func ExampleDefault() {
	result := flow.Default("", "fallback")
	fmt.Println(result)

	result2 := flow.Default("hello", "fallback")
	fmt.Println(result2)

	// Output:
	// fallback
	// hello
}

func ExampleIf() {
	result := flow.If(true, "yes", "no")
	fmt.Println(result)

	result2 := flow.If(false, "yes", "no")
	fmt.Println(result2)

	// Output:
	// yes
	// no
}

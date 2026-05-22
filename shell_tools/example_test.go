package shelltools_test

import (
	"fmt"

	shelltools "github.com/guionardo/go/shell_tools"
)

func ExampleGetEnv() {
	// Assuming ENV_EXAMPLE is not set
	result := shelltools.GetEnv("ENV_EXAMPLE")
	fmt.Println(result)

	// Output:
	//
}

func ExampleNewQuotedShellArgs() {
	args := shelltools.NewQuotedShellArgs(`echo "hello world" foo`)
	fmt.Println(len(args))
	fmt.Println(args[0])
	fmt.Println(args[1])
	fmt.Println(args[2])

	// Output:
	// 3
	// echo
	// hello world
	// foo
}

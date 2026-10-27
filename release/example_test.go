package release_test

import (
	"fmt"
	"log"

	"github.com/guionardo/go/release"
)

func ExampleParseVersion() {
	tag, err := release.ParseVersion("v1.4.2-beta1")
	if err != nil {
		log.Fatal(err)
	}

	installed, err := release.ParseVersion("1.4.1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tag.String())
	fmt.Println(tag.GreaterThan(installed))
	fmt.Println(tag.Segments()[0], tag.Segments()[1], tag.Segments()[2])
	// Output:
	// 1.4.2-beta1
	// true
	// 1 4 2
}

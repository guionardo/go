package main

import (
	"context"
	"fmt"
	"os"

	"github.com/guionardo/go/release"
)

func main() {
	result := release.PerformSelfUpdate(context.Background())
	fmt.Println(result)
	if result.Updated {
		os.Exit(0)
	}
	if result.Err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", result.Err)
		os.Exit(1)
	}
}

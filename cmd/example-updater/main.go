// Package main provides an example CLI demonstrating the release self-update package.
//
// It calls release.PerformSelfUpdate and exits:
//   - 0 if the swapper was spawned (update in progress) or already current
//   - 1 if the update failed
//
// Build with ldflags to set version:
//
//	go build -ldflags="-X main.version=v1.0.0" ./cmd/example-updater/
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

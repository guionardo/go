package sqlite_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/guionardo/go/cache/sqlite"
)

// Example demonstrates a Set/Get roundtrip through an in-memory SQLite cache.
// File-backed caches support the same surface with sqlite.WithPath or
// sqlite.WithName.
func Example() {
	c := sqlite.New[string, string](sqlite.WithMemory())
	defer func() { _ = c.Close() }()

	ctx := context.Background()

	if err := c.Set(ctx, "greeting", "hello"); err != nil {
		fmt.Println("set failed:", err)

		return
	}

	value, err := c.Get(ctx, "greeting")
	if err != nil {
		fmt.Println("get failed:", err)

		return
	}

	fmt.Println("value:", value)
	// Output: value: hello
}

// ExampleNew_withPath demonstrates durability across a close/reopen cycle.
func ExampleNew_withPath() {
	dir, err := os.MkdirTemp("", "sqlite-cache-example")
	if err != nil {
		fmt.Println("temp dir failed:", err)

		return
	}

	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "cache.db")
	ctx := context.Background()

	first := sqlite.New[string, string](sqlite.WithPath(path))
	// Close on every path: the explicit Close below is part of the demo, and
	// this deferred Close covers the error returns before it (IN-01).
	defer func() { _ = first.Close() }()

	if err := first.Set(ctx, "greeting", "hello"); err != nil {
		fmt.Println("set failed:", err)

		return
	}

	if err := first.Close(); err != nil {
		fmt.Println("close failed:", err)

		return
	}

	second := sqlite.New[string, string](sqlite.WithPath(path))
	defer func() { _ = second.Close() }()

	value, err := second.Get(ctx, "greeting")
	if err != nil {
		fmt.Println("get failed:", err)

		return
	}

	fmt.Println("value:", value)
	// Output: value: hello
}

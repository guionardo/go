package sqlite_test

import (
	"context"
	"fmt"

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

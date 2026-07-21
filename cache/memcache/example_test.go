//go:build e2e

package memcache_test

import (
	"context"
	"fmt"

	"github.com/guionardo/go/cache/memcache"
)

func ExampleNew() {
	c := memcache.New[string, string]()

	if err := c.Set(context.Background(), "example", "memcache-value"); err != nil {
		fmt.Println("error:", err)
		return
	}

	value, err := c.Get(context.Background(), "example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(value)

	if err := c.Close(); err != nil {
		fmt.Println("error:", err)
	}

	// Output: memcache-value
}

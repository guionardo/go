package memcache_test

import (
	"context"
	"fmt"

	"github.com/guionardo/go/cache/memcache"
)

func ExampleNew() {
	c := memcache.New[string, string]()
	defer c.Close()

	_ = c.Set(context.Background(), "example", "memcache-value")
	val, _ := c.Get(context.Background(), "example")
	fmt.Println(val)

	// Output: memcache-value
}

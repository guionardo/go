package mem_test

import (
	"context"
	"fmt"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/mem"
)

func ExampleNew() {
	c := mem.New[string, string]()
	_ = c.Set(context.Background(), "hello", "world")
	val, _ := c.Get(context.Background(), "hello")
	fmt.Println(val)

	// Output: world
}

func ExampleNew_withTTL() {
	c := mem.New[string, string](cache.WithDefaultTTL(5 * time.Minute))
	_ = c.Set(context.Background(), "key", "value")
	val, _ := c.Get(context.Background(), "key")
	fmt.Println(val)

	// Output: value
}

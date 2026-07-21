package redis_test

import (
	"context"
	"fmt"

	"github.com/guionardo/go/cache/redis"
)

func ExampleNew() {
	c := redis.New[string, string](redis.WithAddr("localhost:6379"))

	if err := c.Set(context.Background(), "example", "redis-value"); err != nil {
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

	// Output: redis-value
}

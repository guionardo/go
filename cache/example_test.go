package cache_test

import (
	"fmt"
	"time"

	"github.com/guionardo/go/cache"
)

func ExampleWithDefaultTTL() {
	_ = cache.WithDefaultTTL(5 * time.Minute)
	fmt.Println("option created")

	// Output: option created
}

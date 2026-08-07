package cache_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guionardo/go/cache"
)

func ExampleWithDefaultTTL() {
	_ = cache.WithDefaultTTL(5 * time.Minute)
	fmt.Println("option created")

	// Output: option created
}

func ExampleSingleflightGetOrSet_Do() {
	// In-test stub store over a local map, so the example is deterministic.
	var mu sync.Mutex

	store := map[string]string{}

	get := func(ctx context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()

		if v, ok := store[k]; ok {
			return v, nil
		}

		return "", cache.ErrMiss
	}
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()

		store[k] = v

		return nil
	}
	setter := func(context.Context) (string, error) {
		return "computed", nil
	}

	sf := &cache.SingleflightGetOrSet[string, string]{}

	v, err := sf.Do(context.Background(), "k", get, set, setter)
	if err != nil {
		panic(err)
	}

	fmt.Printf("returned=%q stored=%q\n", v, store["k"])

	// Output: returned="computed" stored="computed"
}

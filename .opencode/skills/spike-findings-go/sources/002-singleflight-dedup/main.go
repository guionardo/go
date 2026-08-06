package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// simulates the mem provider's GetOrSet but with singleflight wrapping
// the setter, so a concurrent miss collapses to one setter execution.
func main() {
	const callers = 50
	var setterRuns atomic.Int64
	var sharedCount atomic.Int64

	var sf singleflight.Group

	run := func(key string) (string, error) {
		v, err, shared := sf.Do(key, func() (any, error) {
			setterRuns.Add(1)
			time.Sleep(5 * time.Millisecond) // simulate expensive computation
			return "computed-value", nil
		})
		if shared {
			sharedCount.Add(1)
		}
		return v.(string), err
	}

	var wg sync.WaitGroup
	results := make([]string, callers)
	start := time.Now()
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, _ := run("hot-key")
			results[i] = r
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("callers:      %d\n", callers)
	fmt.Printf("setter runs:  %d (expected 1)\n", setterRuns.Load())
	fmt.Printf("shared:       %d (leader may ALSO get shared=true if followers arrived during execution)\n", sharedCount.Load())
	fmt.Printf("elapsed:      %v (single setter, parallel waiters)\n", elapsed)

	allSame := true
	for _, r := range results {
		if r != "computed-value" {
			allSame = false
			break
		}
	}
	fmt.Printf("all same val: %v\n", allSame)

	// shared semantics: Do returns c.dups > 0 at RETURN time, so the leader
	// reports shared=true whenever any follower was waiting. It does NOT mean
	// "this caller was a follower".
	if setterRuns.Load() == 1 && allSame && sharedCount.Load() == int64(callers) {
		fmt.Println("SPIKE 002: VALIDATED — dedup works; note: shared=true for leader too when followers waited")
	} else {
		fmt.Println("SPIKE 002: FAILED")
		panic(fmt.Sprintf("setterRuns=%d allSame=%v shared=%d", setterRuns.Load(), allSame, sharedCount.Load()))
	}
}
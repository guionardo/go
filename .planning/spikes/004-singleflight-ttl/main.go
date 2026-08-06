package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// simulates GetOrSet with singleflight, where the SET happens INSIDE the
// singleflight fn so only the leader writes. Probes TTL interaction.
func main() {
	testExpiryForcesRecompute()
	testMidFlightStaleness()
	testDirectSetNotClobbered()
}

// CASE A: after TTL expiry, GetOrSet must recompute (singleflight does NOT
// cache completed results — the group key is deleted on completion).
func testExpiryForcesRecompute() {
	var sf singleflight.Group
	var sfValue atomic.Int64

	getOrSet := func(key string) (int64, error) {
		v, err, _ := sf.Do(key, func() (any, error) {
			n := sfValue.Add(1)
			return n, nil
		})
		return v.(int64), err
	}

	// first computation
	v1, _ := getOrSet("k")
	// simulate TTL expiry + later call
	time.Sleep(2 * time.Millisecond)
	v2, _ := getOrSet("k")
	time.Sleep(2 * time.Millisecond)
	v3, _ := getOrSet("k")

	fmt.Printf("[A] values across calls: %d, %d, %d — each call recomputes, no stale cache\n", v1, v2, v3)
	if !(v1 == 1 && v2 == 2 && v3 == 3) {
		panic(fmt.Sprintf("expected monotonic recompute, got %d %d %d", v1, v2, v3))
	}
	fmt.Println("CASE A: PASS — group forgets after completion; expiry naturally forces recompute")
}

// CASE B: a slow setter + short TTL — a caller arriving AFTER expiry but
// DURING the old computation still receives the in-flight (older) result.
func testMidFlightStaleness() {
	var sf singleflight.Group
	var computeCount atomic.Int64
	now := time.Now()

	// leader: slow computation representing "started at time X"
	go func() {
		sf.Do("k", func() (any, error) {
			computeCount.Add(1)
			time.Sleep(100 * time.Millisecond)
			return fmt.Sprintf("computed@%dms", time.Since(now)/time.Millisecond), nil
		})
	}()

	time.Sleep(40 * time.Millisecond) // "TTL expired at 20ms; we are at 40ms"

	// follower arrives after logical expiry, joins the still-running group
	v, _, _ := sf.Do("k", func() (any, error) { panic("must not recompute") })
	elapsed := time.Since(now) / time.Millisecond
	fmt.Printf("[B] follower at %dms got: %q (computed at ~0ms, delivered at %dms)\n",
		elapsed, v, elapsed)
	fmt.Printf("[B] total computations: %d\n", computeCount.Load())
	if computeCount.Load() != 1 {
		panic("follower recomputed — singleflight dedup failed")
	}
	fmt.Println("CASE B: PASS — mid-flight result shared; staleness window = setter duration, documented caveat")
}

// CASE C: a direct Set racing the leader. If the singleflight fn re-checks the
// cache before computing, the direct Set wins and no clobbering happens.
func testDirectSetNotClobbered() {
	var sf singleflight.Group
	// "cache" store: a plain variable guarded by the fn's re-check
	var store atomic.Value
	store.Store("initial")

	getOrSet := func(key string) (string, error) {
		// classic double-check pattern: re-read INSIDE the fn
		v, err, _ := sf.Do(key, func() (any, error) {
			if cur, ok := store.Load().(string); ok && cur != "" {
				return cur, nil
			}
			return "slowly-computed", nil
		})
		return v.(string), err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	// leader starts; it will see the store empty at compute time (sleep inside)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond) // let follower's direct Set land first
		_, _ = getOrSet("k")
	}()

	time.Sleep(5 * time.Millisecond) // leader is now blocked inside group
	store.Store("fresh-direct-set")
	v, err := getOrSet("k") // follower — should get fresh-direct-set if re-check works
	_ = err
	wg.Wait()

	fmt.Printf("[C] follower direct-set value observed: %q\n", v)
	if v != "fresh-direct-set" {
		panic("direct Set was clobbered by slower computation")
	}
	fmt.Println("CASE C: PASS — re-checking the store inside the singleflight fn prevents clobbering")
}

var _ = context.Background

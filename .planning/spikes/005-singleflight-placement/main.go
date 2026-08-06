package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/mem"
)

// Proposed shared placement: a helper type in the cache package. Each provider
// embeds one Group and delegates GetOrSet to it, passing the provider's own
// Get/Set closures. Single copy of the singleflight logic for all 5 providers.
//
// Design decisions proven by spikes 002/003/004 baked in:
//   - setter runs inside Do (dedup, spike 002)
//   - double-check re-read inside fn before computing (clobber guard, spike 004)
//   - error returned to all waiters (spike 003)
//   - uses Do (blocking) — no per-caller cancel (spike 003: Do is ctx-blind,
//     acceptable tradeoff, documented)
type singleflightGetOrSet[K comparable, V any] struct {
	group singleflight.Group
}

// getter wraps a provider's Get; setter wraps provider's Set.
func (s *singleflightGetOrSet[K, V]) do(
	ctx context.Context,
	key K,
	get func(context.Context, K) (V, error),
	set func(context.Context, K, V, ...time.Duration) error,
	setter func() (V, error),
	ttl ...time.Duration,
) (V, error) {
	var zero V
	if v, err := get(ctx, key); err == nil {
		return v, nil
	}

	v, err, _ := s.group.Do(fmt.Sprint(key), func() (any, error) {
		// double-check: another leader may have Set while we queued
		if v, err := get(ctx, key); err == nil {
			return v, nil
		}
		computed, err := setter()
		if err != nil {
			return zero, err
		}
		if err := set(ctx, key, computed, ttl...); err != nil {
			return zero, err
		}
		return computed, nil
	})
	if err != nil {
		return zero, err
	}
	return v.(V), nil
}

func main() {
	const callers = 40
	c := mem.New[string, string](cache.WithDefaultTTL(time.Second))
	defer c.Close()

	sf := &singleflightGetOrSet[string, string]{}
	var setterRuns atomic.Int64

	var wg sync.WaitGroup
	results := make([]string, callers)
	start := time.Now()
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := sf.do(context.Background(), "k",
				c.Get, c.Set,
				func() (string, error) {
					setterRuns.Add(1)
					time.Sleep(3 * time.Millisecond)
					return "computed", nil
				})
			if err != nil {
				results[i] = "ERR:" + err.Error()
				return
			}
			results[i] = v
		}(i)
	}
	wg.Wait()

	fmt.Printf("callers=%d setterRuns=%d elapsed=%v\n", callers, setterRuns.Load(), time.Since(start))
	ok := setterRuns.Load() == 1
	for _, r := range results {
		if r != "computed" {
			ok = false
			fmt.Printf("bad result: %q\n", r)
		}
	}
	// second phase: value now cached, GetOrSet must hit cache (setter NOT re-run)
	v, err := sf.do(context.Background(), "k", c.Get, c.Set,
		func() (string, error) { panic("must not run — cached") })
	fmt.Printf("cached read: %q err=%v setterRuns=%d\n", v, err, setterRuns.Load())
	if !ok || err != nil || v != "computed" || setterRuns.Load() != 1 {
		panic("placement prototype failed")
	}
	fmt.Println("SPIKE 005: VALIDATED — shared singleflightGetOrSet works against real mem provider")
}
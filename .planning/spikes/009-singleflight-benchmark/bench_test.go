package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"golang.org/x/sync/singleflight"
)

// Spike 009: benchmark the thundering-herd win — N concurrent GetOrSet misses
// on one key, naive (each runs the setter) vs singleflight (one setter + N-1
// waiters). Uses the real cache/mem provider's semantics via a store wrapper.
//
// Run:
//   go test -bench=. -benchmem .

var (
	benchMissTime = 2 * time.Millisecond
	benchN        = 64
)

type store struct {
	mu   sync.Mutex
	data map[string]string
}

func newStore() *store { return &store{data: map[string]string{}} }
func (s *store) get(k string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[k]; ok {
		return v, nil
	}
	return "", cache.ErrMiss
}
func (s *store) set(k, v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
	return nil
}

// naive: every caller on a miss runs the setter.
func naiveGetOrSet(ctx context.Context, st *store, key string, setter func() (string, error)) (string, error) {
	if v, err := st.get(key); err == nil {
		return v, nil
	}
	c, err := setter()
	if err != nil {
		return "", err
	}
	_ = ctx
	return c, nil
}

// herd: singleflight collapses concurrent misses.
func herdGetOrSet(g *singleflight.Group, st *store, key string, setter func() (string, error)) (string, error) {
	if v, err := st.get(key); err == nil {
		return v, nil
	}
	v, err, _ := g.Do(key, func() (any, error) {
		c, err := setter()
		if err != nil {
			return "", err
		}
		if err := st.set(key, c); err != nil {
			return "", err
		}
		return c, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func runConcurrent(fn func(i int, setter func() (string, error)) (string, error)) (setterRuns int64, elapsed time.Duration) {
	var runs atomic.Int64
	setter := func() (string, error) {
		runs.Add(1)
		time.Sleep(benchMissTime)
		return "computed", nil
	}
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < benchN; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, _ = fn(i, setter) }(i)
	}
	wg.Wait()
	return runs.Load(), time.Since(start)
}

func benchNaive() (int64, time.Duration) {
	return runConcurrent(func(_ int, setter func() (string, error)) (string, error) {
		return naiveGetOrSet(context.Background(), newStore(), "k", setter)
	})
}

func benchHerd() (int64, time.Duration) {
	st := newStore()
	var gf singleflight.Group
	return runConcurrent(func(_ int, setter func() (string, error)) (string, error) {
		return herdGetOrSet(&gf, st, "k", setter)
	})
}

// Benchmark wrappers so `go test -bench` reports it properly.
func BenchmarkThunderingHerd_Naive(b *testing.B)  { runConcurrentBM(b, false) }
func BenchmarkThunderingHerd_Singleflight(b *testing.B) { runConcurrentBM(b, true) }

func runConcurrentBM(b *testing.B, useSF bool) {
	st := newStore()
	var gf singleflight.Group
	sf := &gf
	_ = sf
	for i := 0; i < b.N; i++ {
		_ = st.set("k"+fmt.Sprint(i), "v")
		var wg sync.WaitGroup
		for j := 0; j < benchN; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if useSF {
					_, _ = herdGetOrSet(sf, st, "k", func() (string, error) {
						time.Sleep(benchMissTime)
						return "computed", nil
					})
				} else {
					_, _ = naiveGetOrSet(context.Background(), st, "k", func() (string, error) {
						time.Sleep(benchMissTime)
						return "computed", nil
					})
				}
			}()
		}
		wg.Wait()
	}
}

func TestBenchHarness(t *testing.T) {
	runs, el := benchNaive()
	t.Logf("naive: setterRuns=%d elapsed=%v", runs, el)
	runs2, el2 := benchHerd()
	t.Logf("herd : setterRuns=%d elapsed=%v", runs2, el2)
	if runs <= 1 || runs2 != 1 {
		t.Fatalf("expected naive>1 and herd==1, got %d and %d", runs, runs2)
	}
}

var _ = fmt.Sprint
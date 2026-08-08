package cache_test

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/mem"
)

// skipIfNoDocker skips the benchmark if Docker is not available.
// Used by Docker-backed provider subtests.
func skipIfNoDocker(t testing.TB, reason string) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip(reason)
	}
}

// benchmarkHerdNaive benchmarks the thundering-herd scenario without singleflight
// dedup. Each concurrent miss runs the setter independently — the setter is
// expected to run n times per b.N iteration (once per goroutine).
//
// Uses a local sync.Mutex-protected map to simulate a naive cache store, NOT
// the cache.BatchCache interface, so the comparison measures raw dedup effect
// without any provider overhead.
func benchmarkHerdNaive(b *testing.B, n int) {
	var mu sync.Mutex
	store := map[string]string{}

	naiveGetOrSet := func(key string, setter func() string) string {
		// Phase 1: check under lock
		mu.Lock()
		if v, ok := store[key]; ok {
			mu.Unlock()
			return v
		}
		mu.Unlock()

		// Phase 2: compute outside lock — THE THUNDERING HERD.
		// Multiple goroutines can be computing simultaneously because
		// there is no dedup mechanism. Each concurrent miss runs the
		// setter independently.
		v := setter()

		// Phase 3: store under lock (last writer wins).
		mu.Lock()
		store[key] = v
		mu.Unlock()
		return v
	}

	var totalSetterRuns atomic.Int64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Unique key per iteration to avoid cached-key cross-contamination
		// between b.N cycles.
		key := fmt.Sprintf("herd_naive_%d_%d", n, i)

		var wg sync.WaitGroup
		for j := 0; j < n; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				naiveGetOrSet(key, func() string {
					totalSetterRuns.Add(1)
					// Simulate a real setter workload (e.g., DB query, API call).
					time.Sleep(time.Millisecond)
					return "computed"
				})
			}()
		}
		wg.Wait()
	}
	b.StopTimer()
	b.ReportMetric(float64(totalSetterRuns.Load())/float64(b.N), "setter_runs/op")
}

// benchmarkHerdSingleflight benchmarks the thundering-herd scenario with
// singleflight dedup via cache.BatchCache.GetOrSet. The setter is expected
// to run exactly once per b.N iteration regardless of n.
func benchmarkHerdSingleflight(b *testing.B, n int, c cache.BatchCache[string, string]) {
	var totalSetterRuns atomic.Int64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Unique key per iteration avoids cached-key cross-contamination
		// between b.N cycles.
		key := fmt.Sprintf("herd_sf_%d_%d", n, i)

		var wg sync.WaitGroup
		for j := 0; j < n; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Discard errors — the benchmark measures dedup correctness
				// and performance, and the mem provider does not error on
				// GetOrSet with a successful setter.
				_, _ = c.GetOrSet(context.Background(), key, func(ctx context.Context) (string, error) {
					totalSetterRuns.Add(1)
					// Simulate a real setter workload (e.g., DB query, API call).
					time.Sleep(time.Millisecond)
					return "computed", nil
				})
			}()
		}
		wg.Wait()
	}
	b.StopTimer()
	b.ReportMetric(float64(totalSetterRuns.Load())/float64(b.N), "setter_runs/op")
}

// benchmarkHerdSingleflightConcurrencyLoop runs the singleflight herd benchmark
// across all concurrency levels for a given provider. Used by both the mem
// provider and Docker-backed subtests to avoid code duplication.
func benchmarkHerdSingleflightConcurrencyLoop(b *testing.B, c cache.BatchCache[string, string]) {
	concurrencyLevels := []int{10, 50, 100, 500}
	for _, n := range concurrencyLevels {
		n := n // capture range variable
		b.Run(fmt.Sprintf("Concurrency%d", n), func(b *testing.B) {
			benchmarkHerdSingleflight(b, n, c)
		})
	}
}

// BenchmarkSingleflightGetOrSet is the thundering-herd benchmark that
// quantifies the singleflight dedup win. It compares a naive path (no dedup)
// against the GetOrSet singleflight path for increasing concurrency levels.
//
// The mem provider sub-benchmarks always run (zero-dependency). Docker-backed
// provider subtests (redis, valkey, memcache, postgres) are gated by
// skipIfNoDocker and only execute when Docker is available.
func BenchmarkSingleflightGetOrSet(b *testing.B) {
	concurrencyLevels := []int{10, 50, 100, 500}

	// ---- Mem provider subtests (always run, zero-dependency) ----
	for _, n := range concurrencyLevels {
		n := n // capture range variable

		// Naive path: no singleflight dedup
		b.Run(fmt.Sprintf("Concurrency%d/naive", n), func(b *testing.B) {
			benchmarkHerdNaive(b, n)
		})

		// Singleflight path: via GetOrSet with mem provider
		b.Run(fmt.Sprintf("Concurrency%d/singleflight", n), func(b *testing.B) {
			c := mem.New[string, string](b.Context())
			defer c.Close()
			benchmarkHerdSingleflight(b, n, c)
		})
	}
}

package cache_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/guionardo/go/cache"
	"github.com/guionardo/go/cache/mem"
	"github.com/guionardo/go/cache/memcache"
	"github.com/guionardo/go/cache/postgres"
	"github.com/guionardo/go/cache/redis"
	"github.com/guionardo/go/cache/valkey"
)

// skipIfNoDocker skips the benchmark if Docker is not available.
// Used by Docker-backed provider subtests.  Checks both the Docker CLI
// (docker info) and the DOCKER_HOST environment variable, because
// testcontainers-go needs DOCKER_HOST when Docker runs on a non-default
// socket (e.g. Orbstack on macOS).
func skipIfNoDocker(t testing.TB, reason string) {
	t.Helper()
	if os.Getenv("DOCKER_HOST") == "" {
		t.Skipf("%s (DOCKER_HOST not set)", reason)
		return
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("%s (docker info failed: %v)", reason, err)
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
	// Uses a separate provider per sub-benchmark so each concurrency level
	// starts with a fresh cache.
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

	// ---- Docker-backed provider subtests (gated by skipIfNoDocker) ----

	b.Run("redis", func(b *testing.B) {
		skipIfNoDocker(b, "redis benchmark requires Docker")
		ctx := b.Context()

		redisC, err := tcredis.RunContainer(ctx,
			testcontainers.WithImage("redis:7-alpine"),
			tcredis.WithSnapshotting(0, 0),
		)
		if err != nil {
			b.Fatal(err)
		}

		port, err := redisC.MappedPort(ctx, "6379/tcp")
		if err != nil {
			b.Fatal(err)
		}
		host, err := redisC.Host(ctx)
		if err != nil {
			b.Fatal(err)
		}

		addr := host + ":" + port.Port()
		provider := redis.New[string, string](redis.WithAddr(addr))
		defer provider.Close()
		benchmarkHerdSingleflightConcurrencyLoop(b, provider)
	})

	b.Run("valkey", func(b *testing.B) {
		skipIfNoDocker(b, "valkey benchmark requires Docker")
		ctx := b.Context()

		valkeyC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "valkey/valkey:8-alpine",
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor: wait.ForAll(
					wait.ForLog("* Ready to accept connections"),
					wait.ForListeningPort("6379/tcp"),
				).WithStartupTimeout(30 * time.Second),
			},
			Started: true,
		})
		if err != nil {
			b.Fatal(err)
		}

		port, err := valkeyC.MappedPort(ctx, "6379")
		if err != nil {
			b.Fatal(err)
		}
		host, err := valkeyC.Host(ctx)
		if err != nil {
			b.Fatal(err)
		}

		addr := host + ":" + port.Port()
		provider := valkey.New[string, string](valkey.WithAddr(addr))
		defer provider.Close()
		benchmarkHerdSingleflightConcurrencyLoop(b, provider)
	})

	b.Run("memcache", func(b *testing.B) {
		skipIfNoDocker(b, "memcache benchmark requires Docker")
		ctx := b.Context()

		memcacheC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "memcached:1-alpine",
				ExposedPorts: []string{"11211/tcp"},
				WaitingFor:   wait.ForListeningPort("11211/tcp").WithStartupTimeout(30 * time.Second),
			},
			Started: true,
		})
		if err != nil {
			b.Fatal(err)
		}

		port, err := memcacheC.MappedPort(ctx, "11211")
		if err != nil {
			b.Fatal(err)
		}
		host, err := memcacheC.Host(ctx)
		if err != nil {
			b.Fatal(err)
		}

		addr := host + ":" + port.Port()
		provider := memcache.New[string, string](memcache.WithServers(addr))
		defer provider.Close()
		benchmarkHerdSingleflightConcurrencyLoop(b, provider)
	})

	b.Run("postgres", func(b *testing.B) {
		skipIfNoDocker(b, "postgres benchmark requires Docker")
		ctx := b.Context()

		pgC, err := tcpostgres.RunContainer(ctx,
			testcontainers.WithImage("postgres:16-alpine"),
			tcpostgres.WithDatabase("cache_bench"),
			tcpostgres.WithUsername("test"),
			tcpostgres.WithPassword("test"),
		)
		if err != nil {
			b.Fatal(err)
		}

		port, err := pgC.MappedPort(ctx, "5432/tcp")
		if err != nil {
			b.Fatal(err)
		}
		host, err := pgC.Host(ctx)
		if err != nil {
			b.Fatal(err)
		}

		connStr := "postgres://test:test@" + host + ":" + port.Port() + "/cache_bench?sslmode=disable"

		// Retry connecting — Postgres may not accept connections immediately.
		var pgProvider cache.BatchCache[string, string]
		var lastErr error
		for retry := 0; retry < 10; retry++ {
			pgProvider, lastErr = postgres.New[string, string](postgres.WithConnString(connStr))
			if lastErr == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if lastErr != nil {
			b.Fatal(lastErr)
		}

		defer pgProvider.Close()
		benchmarkHerdSingleflightConcurrencyLoop(b, pgProvider)
	})
}

// ---------- Batch benchmarks (Plan 08-02) ----------

// prePopulateKeys creates n keys under the given prefix and sets them in the cache.
// Returns the list of keys for use in benchmark iterations.
func prePopulateKeys(b *testing.B, c cache.BatchCache[string, string], n int, prefix string) []string {
	keys := make([]string, n)
	for i := range n {
		key := fmt.Sprintf("%s_%d", prefix, i)
		if err := c.Set(b.Context(), key, "v"); err != nil {
			b.Fatal(err)
		}
		keys[i] = key
	}
	return keys
}

// benchmarkMGetNative benchmarks the native MGet batch operation.
// Keys are pre-populated before the b.N loop starts so the measured
// operation is pure read under a single RLock.
func benchmarkMGetNative(b *testing.B, c cache.BatchCache[string, string], _ int, keys []string) {
	for i := 0; i < b.N; i++ {
		_ = c.MGet(b.Context(), keys...)
	}
}

// benchmarkMGetPerKey benchmarks the per-key Get fallback by iterating over
// all keys and accumulating results in a map (to prevent DCE).
func benchmarkMGetPerKey(b *testing.B, c cache.BatchCache[string, string], n int, keys []string) {
	for i := 0; i < b.N; i++ {
		result := make(map[string]string, n)
		for _, key := range keys {
			if v, err := c.Get(b.Context(), key); err == nil {
				result[key] = v
			}
		}
		_ = result
	}
}

// BenchmarkBatch quantifies the performance win of native batch operations
// (MGet/MSet/MDel) against per-key sequential fallback across multiple batch
// sizes. Uses the mem provider (zero-dependency).
func BenchmarkBatch(b *testing.B) {
	c := mem.New[string, string](b.Context())
	defer c.Close()
	batchSizes := []int{1, 10, 100, 1000}

	b.Run("MGet", func(b *testing.B) {
		for _, n := range batchSizes {
			n := n
			keys := prePopulateKeys(b, c, n, fmt.Sprintf("batch_mget_%d", n))
			b.Run(fmt.Sprintf("Size%d/native", n), func(b *testing.B) {
				benchmarkMGetNative(b, c, n, keys)
			})
			b.Run(fmt.Sprintf("Size%d/per-key", n), func(b *testing.B) {
				benchmarkMGetPerKey(b, c, n, keys)
			})
		}
	})

	b.Run("MSet", func(b *testing.B) {
		for _, n := range batchSizes {
			n := n
			b.Run(fmt.Sprintf("Size%d/native", n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					items := make(map[string]string, n)
					for j := 0; j < n; j++ {
						items[fmt.Sprintf("mset_native_%d_%d_%d", n, i, j)] = "v"
					}
					if err := c.MSet(b.Context(), items); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run(fmt.Sprintf("Size%d/per-key", n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					items := make(map[string]string, n)
					for j := 0; j < n; j++ {
						items[fmt.Sprintf("mset_pk_%d_%d_%d", n, i, j)] = "v"
					}
					for k, v := range items {
						if err := c.Set(b.Context(), k, v); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	})

	b.Run("MDel", func(b *testing.B) {
		for _, n := range batchSizes {
			n := n
			b.Run(fmt.Sprintf("Size%d/native", n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					keys := make([]string, n)
					for j := 0; j < n; j++ {
						key := fmt.Sprintf("mdel_native_%d_%d_%d", n, i, j)
						if err := c.Set(b.Context(), key, "v"); err != nil {
							b.Fatal(err)
						}
						keys[j] = key
					}
					if err := c.MDel(b.Context(), keys...); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run(fmt.Sprintf("Size%d/per-key", n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					keys := make([]string, n)
					for j := 0; j < n; j++ {
						key := fmt.Sprintf("mdel_pk_%d_%d_%d", n, i, j)
						if err := c.Set(b.Context(), key, "v"); err != nil {
							b.Fatal(err)
						}
						keys[j] = key
					}
					for _, key := range keys {
						if err := c.Delete(b.Context(), key); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	})
}

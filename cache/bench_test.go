package cache_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	"github.com/guionardo/go/cache/sqlite"
	"github.com/guionardo/go/cache/valkey"
)

// primitiveKeyRing bounds key growth during timed loops so file-mode Set
// measures overwrite cost, not b-tree growth.
const primitiveKeyRing = 1024

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

// batchKeyRing builds a bounded ring of keys under prefix. The batch arms draw
// their per-iteration key windows from it, so their key space is capped at
// primitiveKeyRing exactly as the primitive arms' is — a file-backed provider
// then measures the batch operation, not b-tree growth.
func batchKeyRing(prefix string) []string {
	ring := make([]string, primitiveKeyRing)
	for i := range ring {
		ring[i] = fmt.Sprintf("%s_ring_%d", prefix, i)
	}

	return ring
}

// fillBatchKeys fills dst with the len(dst) keys of the window starting at
// offset, wrapping around ring. dst is reused across iterations, so selecting
// the window costs no allocation inside the timed loop.
func fillBatchKeys(dst, ring []string, offset int) {
	start := offset % len(ring)
	for j := range dst {
		dst[j] = ring[(start+j)%len(ring)]
	}
}

// runBatchBenchmarks runs the full MGet/MSet/MDel benchmark suite on a given
// provider. Extracted as a shared helper so both the mem subtests and the
// Docker-backed provider subtests reuse the same measurement structure.
//
// What is timed: MGet reads keys populated before the timed loop; MSet writes a
// window of n keys drawn from a bounded key ring; MDel deletes such a window,
// with the Set calls that make those keys exist run with the timer stopped, so
// ns/op covers only the delete and never the seeding. The key space of every
// arm is capped at primitiveKeyRing, so b.N cannot grow the database.
func runBatchBenchmarks(b *testing.B, c cache.BatchCache[string, string], name string) {
	batchSizes := []int{1, 10, 100, 1000}

	b.Run("MGet", func(b *testing.B) {
		for _, n := range batchSizes {
			n := n
			keys := prePopulateKeys(b, c, n, fmt.Sprintf("%s_mget_%d", name, n))
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
			ring := batchKeyRing(fmt.Sprintf("%s_mset_%d", name, n))
			keys := make([]string, n)

			b.Run(fmt.Sprintf("Size%d/native", n), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					fillBatchKeys(keys, ring, i*n)

					items := make(map[string]string, n)
					for _, key := range keys {
						items[key] = "v"
					}

					requireNoErr(b, c.MSet(b.Context(), items))
				}
			})

			b.Run(fmt.Sprintf("Size%d/per-key", n), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					fillBatchKeys(keys, ring, i*n)

					for _, key := range keys {
						requireNoErr(b, c.Set(b.Context(), key, "v"))
					}
				}
			})
		}
	})

	b.Run("MDel", func(b *testing.B) {
		for _, n := range batchSizes {
			n := n
			ring := batchKeyRing(fmt.Sprintf("%s_mdel_%d", name, n))
			keys := make([]string, n)

			b.Run(fmt.Sprintf("Size%d/native", n), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Seeding is setup, not the operation: it runs with the
					// timer stopped so ns/op covers only the batch delete.
					b.StopTimer()

					fillBatchKeys(keys, ring, i*n)

					for _, key := range keys {
						requireNoErr(b, c.Set(b.Context(), key, "v"))
					}

					b.StartTimer()

					requireNoErr(b, c.MDel(b.Context(), keys...))
				}
			})

			b.Run(fmt.Sprintf("Size%d/per-key", n), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()

					fillBatchKeys(keys, ring, i*n)

					for _, key := range keys {
						requireNoErr(b, c.Set(b.Context(), key, "v"))
					}

					b.StartTimer()

					for _, key := range keys {
						requireNoErr(b, c.Delete(b.Context(), key))
					}
				}
			})
		}
	})
}

// BenchmarkBatch quantifies the performance win of native batch operations
// (MGet/MSet/MDel) against per-key sequential fallback across multiple batch
// sizes. The mem family runs first through runBatchBenchmarks, the same helper
// the sqlite and Docker-backed families use, so every family shares one
// measurement structure: bounded key ring, and only the batch delete timed in
// the MDel arms.
func BenchmarkBatch(b *testing.B) {
	c := mem.New[string, string](b.Context())
	defer c.Close()

	runBatchBenchmarks(b, c, "mem")

	// ---- SQLite provider batch subtests (always run, no Docker) ----

	b.Run("sqlite", func(b *testing.B) {
		b.Run("Memory", func(b *testing.B) {
			c := sqlite.New[string, string](sqlite.WithMemory())

			b.Cleanup(func() { _ = c.Close() })
			runBatchBenchmarks(b, c, "sqlite_memory")
		})

		b.Run("File", func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "batch.db")
			c := sqlite.New[string, string](sqlite.WithPath(path))

			b.Cleanup(func() { _ = c.Close() })
			runBatchBenchmarks(b, c, "sqlite_file")
		})
	})

	// ---- Docker-backed provider batch subtests (gated by skipIfNoDocker) ----

	type dockerProvider struct {
		name  string
		setup func(b *testing.B) cache.BatchCache[string, string]
	}

	providers := []dockerProvider{
		{
			"redis", func(b *testing.B) cache.BatchCache[string, string] {
				skipIfNoDocker(b, "redis batch benchmark requires Docker")
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
				return redis.New[string, string](redis.WithAddr(addr))
			},
		},
		{
			"valkey", func(b *testing.B) cache.BatchCache[string, string] {
				skipIfNoDocker(b, "valkey batch benchmark requires Docker")
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
				return valkey.New[string, string](valkey.WithAddr(addr))
			},
		},
		{
			"memcache", func(b *testing.B) cache.BatchCache[string, string] {
				skipIfNoDocker(b, "memcache batch benchmark requires Docker")
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
				return memcache.New[string, string](memcache.WithServers(addr))
			},
		},
		{
			"postgres", func(b *testing.B) cache.BatchCache[string, string] {
				skipIfNoDocker(b, "postgres batch benchmark requires Docker")
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

				return pgProvider
			},
		},
	}

	for _, p := range providers {
		p := p // capture range variable
		b.Run(p.name, func(b *testing.B) {
			c := p.setup(b)
			defer c.Close()
			runBatchBenchmarks(b, c, p.name)
		})
	}
}

// requireNoErr fails the benchmark on the first error, so a provider that stops
// working mid-run is reported instead of silently measuring an error return.
// Healthy providers never take this branch, so it costs one predictable check.
func requireNoErr(b *testing.B, err error) {
	b.Helper()

	if err != nil {
		b.Fatal(err)
	}
}

// requireUsable probes the provider once before any timing starts. sqlite
// defers open/validation errors to the first operation (New stores them in
// initErr), so without this probe a broken cache would report fabricated ns/op.
func requireUsable(b *testing.B, c cache.BatchCache[string, string]) {
	b.Helper()

	const probeKey = "prim_smoke"

	err := c.Set(b.Context(), probeKey, "v")
	if err != nil {
		b.Fatalf("provider unusable: Set failed: %v", err)
	}

	_, err = c.Get(b.Context(), probeKey)
	if err != nil {
		b.Fatalf("provider unusable: Get failed: %v", err)
	}
}

// runPrimitiveBenchmarks measures the single-key operations (Get/Set/Delete/
// GetOrSet) against one provider. Keys are bounded by primitiveKeyRing so a
// file-backed provider measures the operation instead of database growth.
//
// Exactly what each arm times:
//   - Get: one read of a pre-populated key.
//   - Set: one overwrite of a key picked from the ring.
//   - Delete: a Set+Delete cycle, because removing an existing key requires
//     creating it first — the reported ns/op is that whole cycle.
//   - GetOrSet: the miss path. The key is deleted before every call with the
//     timer stopped, so the setter runs each iteration and the reported ns/op
//     is GetOrSet's miss cost, not a warm hit.
//
// Every operation is error-checked, matching the b.Fatal pattern
// runBatchBenchmarks already uses.
func runPrimitiveBenchmarks(b *testing.B, c cache.BatchCache[string, string]) {
	requireUsable(b, c)

	b.Run("Get", func(b *testing.B) {
		keys := prePopulateKeys(b, c, 1, "prim_get")

		for i := 0; i < b.N; i++ {
			_, err := c.Get(b.Context(), keys[0])
			requireNoErr(b, err)
		}
	})

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := c.Set(b.Context(), fmt.Sprintf("prim_set_%d", i%primitiveKeyRing), "v")
			requireNoErr(b, err)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("prim_del_%d", i%primitiveKeyRing)

			err := c.Set(b.Context(), key, "v")
			requireNoErr(b, err)

			err = c.Delete(b.Context(), key)
			requireNoErr(b, err)
		}
	})

	b.Run("GetOrSet", func(b *testing.B) {
		setter := func(context.Context) (string, error) { return "v", nil }

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("prim_gos_%d", i%primitiveKeyRing)

			// Force a miss: the delete is setup, so it is untimed.
			b.StopTimer()
			requireNoErr(b, c.Delete(b.Context(), key))
			b.StartTimer()

			_, err := c.GetOrSet(b.Context(), key, setter)
			requireNoErr(b, err)
		}
	})
}

// BenchmarkPrimitives compares the primitive cache operations across the mem
// baseline and both sqlite modes, so the durable path is measured against the
// in-memory one. Every provider is always-run (no Docker gate).
func BenchmarkPrimitives(b *testing.B) {
	b.Run("mem", func(b *testing.B) {
		c := mem.New[string, string](b.Context())

		b.Cleanup(func() { _ = c.Close() })
		runPrimitiveBenchmarks(b, c)
	})

	b.Run("sqlite/Memory", func(b *testing.B) {
		c := sqlite.New[string, string](sqlite.WithMemory())

		b.Cleanup(func() { _ = c.Close() })
		runPrimitiveBenchmarks(b, c)
	})

	b.Run("sqlite/File", func(b *testing.B) {
		path := filepath.Join(b.TempDir(), "primitives.db")
		c := sqlite.New[string, string](sqlite.WithPath(path))

		b.Cleanup(func() { _ = c.Close() })
		runPrimitiveBenchmarks(b, c)
	})
}

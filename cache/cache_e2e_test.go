//go:build e2e

package cache_test

import (
	"context"
	"errors"
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

type providerCase struct {
	name string
	fn   func(t *testing.T) cache.Cache[string, string]
}

func runCacheE2E(t *testing.T, providers []providerCase) {
	t.Helper()

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			c := p.fn(t)
			ctx := context.Background()

			t.Run("set_and_get", func(t *testing.T) {
				err := c.Set(ctx, "e2e_key", "e2e_value")
				if err != nil {
					t.Fatal(err)
				}

				got, err := c.Get(ctx, "e2e_key")
				if err != nil {
					t.Fatal(err)
				}
				if got != "e2e_value" {
					t.Fatalf("got %q, want %q", got, "e2e_value")
				}
			})

			t.Run("get_miss", func(t *testing.T) {
				_, err := c.Get(ctx, "e2e_nonexistent")
				if err == nil {
					t.Fatal("expected error for missing key")
				}
			})

			t.Run("delete", func(t *testing.T) {
				_ = c.Set(ctx, "e2e_delete", "to_delete")
				if err := c.Delete(ctx, "e2e_delete"); err != nil {
					t.Fatal(err)
				}
				_, err := c.Get(ctx, "e2e_delete")
				if err == nil {
					t.Fatal("expected error after delete")
				}
			})

			t.Run("delete_idempotent", func(t *testing.T) {
				err := c.Delete(ctx, "e2e_nonexistent_delete")
				if err != nil {
					t.Fatalf("delete of missing key should not error: %v", err)
				}
			})

			t.Run("get_or_set_exists", func(t *testing.T) {
				_ = c.Set(ctx, "e2e_gos", "existing")
				got, err := c.GetOrSet(ctx, "e2e_gos", func() (string, error) {
					return "computed", nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if got != "existing" {
					t.Fatalf("got %q, want %q", got, "existing")
				}
			})

			t.Run("get_or_set_computes", func(t *testing.T) {
				got, err := c.GetOrSet(ctx, "e2e_gos_compute", func() (string, error) {
					return "computed", nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if got != "computed" {
					t.Fatalf("got %q, want %q", got, "computed")
				}
			})

			t.Run("get_or_set_setter_error", func(t *testing.T) {
				_, err := c.GetOrSet(ctx, "e2e_gos_err", func() (string, error) {
					return "", errors.New("setter failed")
				})
				if err == nil {
					t.Fatal("expected error from setter")
				}
			})

			t.Run("ttl_expiry", func(t *testing.T) {
				err := c.Set(ctx, "e2e_ttl", "short-lived", 2*time.Second)
				if err != nil {
					t.Fatal(err)
				}

				time.Sleep(3 * time.Second)

				_, err = c.Get(ctx, "e2e_ttl")
				if err == nil {
					t.Fatal("expected error for expired key")
				}
			})

			t.Run("set_no_ttl_uses_default", func(t *testing.T) {
				err := c.Set(ctx, "e2e_no_ttl", "val")
				if err != nil {
					t.Fatal(err)
				}
				got, err := c.Get(ctx, "e2e_no_ttl")
				if err != nil {
					t.Fatal(err)
				}
				if got != "val" {
					t.Fatalf("got %q, want %q", got, "val")
				}
			})

			t.Run("close", func(t *testing.T) {
				err := c.Close()
				if err != nil {
					t.Fatalf("close should not error: %v", err)
				}
			})
		})
	}
}

func TestCacheE2E_Mem(t *testing.T) {
	runCacheE2E(t, []providerCase{
		{
			name: "in_memory",
			fn: func(t *testing.T) cache.Cache[string, string] {
				return mem.New[string, string]()
			},
		},
	})
}

func TestCacheE2E_Redis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis E2E test in short mode")
	}

	ctx := context.Background()
	redisC, err := tcredis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		tcredis.WithSnapshotting(0, 0),
	)
	if err != nil {
		t.Fatal(err)
	}

	port, err := redisC.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}

	host, err := redisC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	addr := host + ":" + port.Port()

	runCacheE2E(t, []providerCase{
		{
			name: "redis",
			fn: func(t *testing.T) cache.Cache[string, string] {
				return redis.New[string, string](redis.WithAddr(addr))
			},
		},
	})
}

func TestCacheE2E_Valkey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Valkey E2E test in short mode")
	}

	ctx := context.Background()
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
		t.Fatal(err)
	}

	port, err := valkeyC.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatal(err)
	}

	host, err := valkeyC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	addr := host + ":" + port.Port()

	runCacheE2E(t, []providerCase{
		{
			name: "valkey",
			fn: func(t *testing.T) cache.Cache[string, string] {
				return valkey.New[string, string](valkey.WithAddr(addr))
			},
		},
	})
}

func TestCacheE2E_Memcache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Memcache E2E test in short mode")
	}

	ctx := context.Background()
	memcacheC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "memcached:1-alpine",
			ExposedPorts: []string{"11211/tcp"},
			WaitingFor:   wait.ForListeningPort("11211/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	port, err := memcacheC.MappedPort(ctx, "11211")
	if err != nil {
		t.Fatal(err)
	}

	host, err := memcacheC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	addr := host + ":" + port.Port()

	runCacheE2E(t, []providerCase{
		{
			name: "memcache",
			fn: func(t *testing.T) cache.Cache[string, string] {
				return memcache.New[string, string](memcache.WithServers(addr))
			},
		},
	})
}

func TestCacheE2E_Postgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Postgres E2E test in short mode")
	}

	ctx := context.Background()
	pgC, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		tcpostgres.WithDatabase("cache_e2e"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
	)
	if err != nil {
		t.Fatal(err)
	}

	port, err := pgC.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	host, err := pgC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	connStr := "postgres://test:test@" + host + ":" + port.Port() + "/cache_e2e?sslmode=disable"

	// Retry connecting — Postgres may not accept connections immediately.
	var pgCache cache.Cache[string, string]
	var lastErr error
	for retry := 0; retry < 10; retry++ {
		pgCache, lastErr = postgres.New[string, string](postgres.WithConnString(connStr))
		if lastErr == nil {
			break
		}
		pgCache = nil
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		t.Fatal(lastErr)
	}

	runCacheE2E(t, []providerCase{
		{
			name: "postgres",
			fn: func(t *testing.T) cache.Cache[string, string] {
				return pgCache
			},
		},
	})
}

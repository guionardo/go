package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/guionardo/go/cache"
	"golang.org/x/sync/singleflight"
)

// Spike 006: can the spike-005 shared helper (get/set closures only) preserve
// each provider's CURRENT GetOrSet error semantics?
//
// Current provider behavior (from reading source):
//   mem:      setter error returned RAW
//   memcache: setter error returned RAW
//   postgres: setter error returned RAW
//   redis:    setter error WRAPPED  "cache/redis: %w"
//   valkey:   setter error WRAPPED  "cache/valkey: %w"
//   valkey:   initErr short-circuit BEFORE calling the setter on a miss
//
// The 005 helper returns whatever setter() returns, unwrapped. That would change
// redis/valkey behavior (lose the "cache/redis:" prefix). And valkey's initErr
// guard can't be expressed by Get/Set closures at all.

var discoErr = errors.New("disco-init-failed")

type helper[K comparable, V any] struct{ group singleflight.Group }

// exactly the spike-005 helper shape
func (s *helper[K, V]) do(
	ctx context.Context, key K,
	get func(context.Context, K) (V, error),
	set func(context.Context, K, V, ...interface{}) error,
	setter func() (V, error),
) (V, error) {
	var zero V
	if v, err := get(ctx, key); err == nil {
		return v, nil
	}
	v, err, _ := s.group.Do(keyString(key), func() (any, error) {
		if v, err := get(ctx, key); err == nil {
			return v, nil
		}
		c, err := setter()
		if err != nil {
			return zero, err
		}
		if err := set(ctx, key, c); err != nil {
			return zero, err
		}
		return c, nil
	})
	if err != nil {
		return zero, err
	}
	return v.(V), nil
}

func keyString(k interface{}) string {
	if s, ok := k.(string); ok {
		return s
	}
	return fmt.Sprint(k)
}

func main() {
	testRedisRegression()
	testMemPreserved()
	testValkeyInitErrGuard()
}

func testRedisRegression() {
	h := &helper[string, string]{}
	target := errors.New("setter-boom")
	_, err := h.do(context.Background(), "k",
		func(context.Context, string) (string, error) { return "", cache.ErrMiss },
		func(context.Context, string, string, ...interface{}) error { return nil },
		func() (string, error) { return "", target })
	prefix := strings.HasPrefix(err.Error(), "cache/redis:")

	fmt.Printf("[redis] 005-helper setter error: prefix_preserved=%v err=%q\n", prefix, err.Error())

	if prefix {
		panic("005 helper unexpectedly preserved redis prefix — semantics diverged")
	}
	fmt.Println("CASE redis: PASS — helper drops the 'cache/redis:' prefix; wrapping providers must wrap inside the delegated fn")
}

func testMemPreserved() {
	h := &helper[string, string]{}
	target := errors.New("mboom")
	_, err := h.do(context.Background(), "k",
		func(context.Context, string) (string, error) { return "", cache.ErrMiss },
		func(context.Context, string, string, ...interface{}) error { return nil },
		func() (string, error) { return "", target })
	fmt.Printf("[mem] helper setter error raw: err=%q\n", err.Error())
	if err != target {
		panic(fmt.Sprintf("expected raw identical error, got %v", err))
	}
	fmt.Println("CASE mem: PASS — raw providers (mem/memcache/postgres) keep identical error identity")
}

func testValkeyInitErrGuard() {
	// The 005 helper shape has no way to express valkey's initErr guard. That
	// guard MUST stay in the provider's own GetOrSet, wrapping the helper.
	sim := &valkeyCache{initErr: discoErr}
	model := &setterModel{}

	err := sim.getOrSetHelper(&helper[string, string]{}, &model.ran)
	fmt.Printf("[valkey] with initErr, setterRan=%v err=%q (want: initErr, setter NOT run)\n", model.ran, err)
	if model.ran {
		panic("valkey initErr guard must live in provider's own GetOrSet, NOT the helper")
	}
	if !strings.HasPrefix(err.Error(), "cache/valkey:") {
		panic(fmt.Sprintf("expected valkey prefix, got %q", err))
	}
	fmt.Println("CASE valkey: PASS — initErr guard + prefix cannot move into helper; must stay provider-side")
}

type setterModel struct{ ran bool }

type valkeyCache struct{ initErr error }

func (c *valkeyCache) get(_ context.Context, _ string) (string, error) {
	return "", errors.New("cache/valkey: key not found")
}
func (c *valkeyCache) set(_ context.Context, _ string, _ string, _ ...interface{}) error { return nil }

// models real valkey: initErr guard BEFORE delegating to the shared helper.
func (c *valkeyCache) getOrSetHelper(h *helper[string, string], setterRan *bool) error {
	if c.initErr != nil {
		return fmt.Errorf("cache/valkey: %w", c.initErr)
	}
	_, err := h.do(context.Background(), "k", c.get, c.set,
		func() (string, error) { *setterRan = true; return "computed", nil })
	return err
}
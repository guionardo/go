package cache

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

// SingleflightGetOrSet deduplicates concurrent GetOrSet misses on the same
// key: when many callers miss the same key at once, the setter runs exactly
// once and every caller receives the identical value or error. Providers
// embed a value of this type (state persists per provider instance) and
// delegate their GetOrSet to Do, passing their own Get/Set closures.
type SingleflightGetOrSet[K comparable, V any] struct {
	group singleflight.Group
}

// Do blocks until the value for key is available, deduplicating concurrent
// misses on the same key.
//
// The fast-path Get runs OUTSIDE the singleflight group, so a cache hit
// returns immediately without singleflight lock contention. On a miss, the
// group fn re-checks the store (double-check Get) so a stale in-flight
// computation never clobbers a concurrent direct Set, then runs the setter
// exactly once (in the leader's goroutine, with the leader's context) and
// stores the result with the supplied ttl (passed verbatim to set).
//
// When concurrent callers pass different TTLs, the LEADER's ttl wins. Because
// the setter is deduplicated, a caller that joins an in-flight computation
// receives the result "as of when the leader started" (a stale window bounded
// by the setter's duration). Do is context-blind for waiters: cancelling a
// follower's context does not release it — it stays blocked until the leader
// completes. Use DoChan (phase 05-02) when per-caller cancellation is needed.
func (s *SingleflightGetOrSet[K, V]) Do(
	ctx context.Context,
	key K,
	get func(context.Context, K) (V, error),
	set func(context.Context, K, V, ...time.Duration) error,
	setter func(context.Context) (V, error),
	ttl ...time.Duration,
) (V, error) {
	var zero V
	if v, err := get(ctx, key); err == nil {
		return v, nil // fast-path Get stays OUTSIDE the group (SF-02)
	}

	v, err, _ := s.group.Do(fmt.Sprint(key), func() (any, error) {
		// double-check Get: another leader may have Set while we queued (SF-08)
		if v, err := get(ctx, key); err == nil {
			return v, nil
		}
		// setter runs only in the leader's goroutine with the leader's ctx (D-04)
		computed, err := s.callSetter(setter, ctx)
		if err != nil {
			return zero, err
		}
		// Set runs once, inside the fn (D-18 ttl passthrough, D-19 leader wins)
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

// DoChan is the cancel-aware variant of Do, with an identical signature (D-12)
// returning (V, error) only — the singleflight.Result channel never leaks to
// callers (D-12).
//
// A waiter's own context gates only its select: when the waiter's context is
// canceled before the leader's value is ready, DoChan abandons the wait
// promptly (SF-06) and returns an error wrapping both cache.ErrCanceled and
// the waiter's ctx.Err() — errors.Is matches either (D-09/D-10). A canceled
// waiter whose value is already ready receives the shared value instead of the
// error (value-if-ready wins, D-08). Canceling a waiter never cancels the
// shared computation: the leader's fn completes and Set lands regardless
// (D-04).
//
// The caller that IS the leader never receives ErrCanceled: its own
// cancellation surfaces as the setter's raw context error through the fn
// result (D-05/D-11). Like Do, the leader's ttl wins when concurrent callers
// pass different TTLs (D-19).
func (s *SingleflightGetOrSet[K, V]) DoChan(
	ctx context.Context,
	key K,
	get func(context.Context, K) (V, error),
	set func(context.Context, K, V, ...time.Duration) error,
	setter func(context.Context) (V, error),
	ttl ...time.Duration,
) (V, error) {
	var zero V
	if v, err := get(ctx, key); err == nil {
		return v, nil // fast-path Get stays OUTSIDE the group (SF-02)
	}

	// leaderCh signals that the caller is the leader: only the leader's fn
	// executes, so only the leader's signal fires (D-04).
	leaderCh := make(chan struct{})
	ch := s.group.DoChan(fmt.Sprint(key), func() (any, error) {
		close(leaderCh)
		// identical body to Do's fn: double-check Get → callSetter → Set
		if v, err := get(ctx, key); err == nil {
			return v, nil
		}
		computed, err := s.callSetter(setter, ctx)
		if err != nil {
			return zero, err
		}
		if err := set(ctx, key, computed, ttl...); err != nil {
			return zero, err
		}
		return computed, nil
	})

	// Non-blocking first: a value that is already ready wins (D-08).
	select {
	case res := <-ch:
		if res.Err != nil {
			return zero, res.Err
		}
		return res.Val.(V), nil
	default:
	}

	// Blocking select: the caller's cancel abandons its own wait (SF-06).
	select {
	case <-ctx.Done():
		// (1) re-read ch non-blocking — the leader's raw result may already be
		// ready by the time ctx.Done() fires (D-11).
		select {
		case res := <-ch:
			if res.Err != nil {
				return zero, res.Err
			}
			return res.Val.(V), nil
		default:
		}
		// (2) if the caller IS the leader, block on ch and return the raw fn
		// result — the leader's own cancellation is the setter's ctx.Err(),
		// never wrapped in ErrCanceled (D-04/D-11).
		select {
		case <-leaderCh:
			res := <-ch
			if res.Err != nil {
				return zero, res.Err
			}
			return res.Val.(V), nil
		default:
		}
		// (3) otherwise this is a waiter abandoning its wait (D-10).
		return zero, fmt.Errorf("%w: %w", ErrCanceled, ctx.Err())
	case res := <-ch:
		if res.Err != nil {
			return zero, res.Err
		}
		return res.Val.(V), nil
	}
}

// callSetter runs the setter with panic recovery. A recovered panic becomes a
// *cache.Panic wrapping the value and stack (mirrors x/sync's panicError), so
// a panicking setter yields the typed error to every waiter and no panic ever
// escapes to a caller goroutine (SF-05 / D-16 / D-17). It never re-panics.
func (s *SingleflightGetOrSet[K, V]) callSetter(setter func(context.Context) (V, error), ctx context.Context) (v V, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = newPanic(r)
		}
	}()
	return setter(ctx)
}
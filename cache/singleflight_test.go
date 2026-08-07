package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guionardo/go/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSingleflightGetOrSet_SetterRunsOnce verifies that N concurrent misses on
// the same key run the setter exactly once and every caller receives the same
// value (ROADMAP success criterion 1).
func TestSingleflightGetOrSet_SetterRunsOnce(t *testing.T) { //nolint:funlen,unparam
	t.Parallel()

	var mu sync.Mutex

	store := map[string]string{}

	var setterRuns atomic.Int64

	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(_ context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()

		if v, ok := store[k]; ok {
			return v, nil
		}

		return "", cache.ErrMiss
	}
	set := func(_ context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()

		store[k] = v

		return nil
	}
	setter := func(context.Context) (string, error) { //nolint:unparam //nolint:unparam
		setterRuns.Add(1)
		time.Sleep(3 * time.Millisecond)

		return computed, nil
	}

	const callers = 50

	var wg sync.WaitGroup

	results := make([]string, callers)
	for i := range callers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			v, err := sf.Do(t.Context(), "k", get, set, setter)
			if err != nil {
				t.Error(err)
				return
			}

			results[i] = v
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int64(1), setterRuns.Load(), "setter must run exactly once")

	for i, r := range results {
		assert.Equal(t, computed, r, "caller %d must receive the computed value", i)
	}

	// Second phase: the value is now cached; Do must return it WITHOUT
	// re-running the setter (a panic here proves the setter was not invoked).
	v, err := sf.Do(t.Context(), "k", get, set, func(context.Context) (string, error) {
		setterRuns.Add(1)
		panic("setter must not run for a cached key")
	})
	require.NoError(t, err)
	assert.Equal(t, computed, v)
	assert.Equal(t, int64(1), setterRuns.Load(), "cached read must not re-run the setter")
}

// TestSingleflightGetOrSet_PanicRecovery verifies that a panicking setter
// yields a *cache.PanicError to every waiter and no panic escapes the Do call
// (SF-05 / D-16 / D-17).
func TestSingleflightGetOrSet_PanicRecovery(t *testing.T) {
	t.Parallel()

	t.Run("string_panic_value", func(t *testing.T) {
		t.Parallel()

		sf := &cache.SingleflightGetOrSet[string, string]{}
		get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
		set := func(context.Context, string, string, ...time.Duration) error { return nil }
		setter := func(context.Context) (string, error) { //nolint:unparam
			panic("boom")
		}

		const callers = 5

		var wg sync.WaitGroup

		errs := make([]error, callers)
		for i := range callers {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				_, err := sf.Do(t.Context(), "k", get, set, setter)
				errs[i] = err
			}(i)
		}

		wg.Wait()

		for i, err := range errs {
			var p *cache.PanicError
			require.ErrorAs(t, err, &p, "waiter %d must receive a *cache.PanicError", i)
			assert.Contains(t, p.Error(), "boom", "waiter %d: Error() must contain the panic value", i)
			assert.NotEmpty(t, p.Stack, "waiter %d: Stack must be captured", i)
		}
	})

	t.Run("error_panic_value", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("sentinel boom")
		sf := &cache.SingleflightGetOrSet[string, string]{}
		get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
		set := func(context.Context, string, string, ...time.Duration) error { return nil }
		setter := func(context.Context) (string, error) { //nolint:unparam
			panic(sentinel)
		}

		_, err := sf.Do(t.Context(), "k", get, set, setter)

		var p *cache.PanicError
		require.ErrorAs(t, err, &p)
		assert.ErrorIs(t, err, sentinel, "errors.Is must traverse Panic.Unwrap()")
	})
}

// TestSingleflightGetOrSet_ZeroValueSetter verifies that a setter returning
// the zero value with nil error stores and returns it; deduplicated waiters
// receive the identical value.
func TestSingleflightGetOrSet_ZeroValueSetter(t *testing.T) { //nolint:funlen,unparam
	t.Parallel()

	var mu sync.Mutex

	store := map[string]string{}

	var setterRuns atomic.Int64

	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(_ context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()

		if v, ok := store[k]; ok {
			return v, nil
		}

		return "", cache.ErrMiss
	}
	set := func(_ context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()

		store[k] = v

		return nil
	}
	setter := func(context.Context) (string, error) { //nolint:unparam
		setterRuns.Add(1)
		return "", nil
	}

	const callers = 10

	var wg sync.WaitGroup

	results := make([]string, callers)
	for i := range callers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			v, err := sf.Do(t.Context(), "k", get, set, setter)
			if err != nil {
				t.Error(err)
				return
			}

			results[i] = v
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int64(1), setterRuns.Load())

	for i, r := range results {
		assert.Empty(t, r, "waiter %d must receive the identical zero value", i)
	}

	mu.Lock()
	_, stored := store["k"]
	mu.Unlock()
	assert.True(t, stored, "the zero value must have been stored")
}

// TestSingleflightGetOrSet_ClobberGuard verifies that a concurrent direct Set
// is never clobbered by a stale in-flight computation: the double-check Get
// inside the singleflight fn picks up the fresh value and the setter never
// runs (SF-08 / T-05-02).
func TestSingleflightGetOrSet_ClobberGuard(t *testing.T) { //nolint:funlen,unparam
	t.Parallel()

	var mu sync.Mutex

	store := map[string]string{}

	var (
		setterRuns atomic.Int64
		getCalls   atomic.Int64
	)

	setLanded := make(chan struct{})

	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(_ context.Context, k string) (string, error) {
		// The first (fast-path) invocation blocks until the direct Set lands,
		// then artificially misses so Do enters the singleflight group; the
		// double-check Get inside the fn is what must find "fresh".
		if getCalls.Add(1) == 1 {
			<-setLanded
			return "", cache.ErrMiss
		}

		mu.Lock()
		defer mu.Unlock()

		if v, ok := store[k]; ok {
			return v, nil
		}

		return "", cache.ErrMiss
	}
	set := func(_ context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()

		store[k] = v

		return nil
	}
	setter := func(context.Context) (string, error) { //nolint:unparam
		setterRuns.Add(1)
		return "stale", nil
	}

	gotCh := make(chan struct {
		v   string
		err error
	}, 1)
	go func() {
		v, err := sf.Do(t.Context(), "k", get, set, setter)
		gotCh <- struct {
			v   string
			err error
		}{v, err}
	}()

	// Direct Set lands while the leader's fast-path Get is blocked.
	require.NoError(t, set(t.Context(), "k", "fresh"))
	close(setLanded)

	res := <-gotCh
	require.NoError(t, res.err)
	assert.Equal(t, "fresh", res.v, "Do must return the direct Set's value")
	assert.Equal(t, int64(0), setterRuns.Load(), "the setter must never run")

	mu.Lock()
	assert.Equal(t, "fresh", store["k"], "the store must still hold the fresh value")
	mu.Unlock()
}

// TestSingleflightGetOrSet_TTLPassthrough verifies that the leader's ttl is
// passed verbatim to the set closure (D-18) and that the leader's ttl wins
// when concurrent callers pass different TTLs (D-19).
func TestSingleflightGetOrSet_TTLPassthrough(t *testing.T) { //nolint:funlen,unparam
	t.Parallel()

	t.Run("single_leader_ttl", func(t *testing.T) {
		t.Parallel()

		var (
			mu          sync.Mutex
			recordedTTL []time.Duration
		)

		sf := &cache.SingleflightGetOrSet[string, string]{}
		get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
		set := func(_ context.Context, _, v string, ttl ...time.Duration) error {
			mu.Lock()
			defer mu.Unlock()

			recordedTTL = append(recordedTTL, ttl...)

			return nil
		}
		setter := func(context.Context) (string, error) { return computed, nil }

		_, err := sf.Do(t.Context(), "k", get, set, setter, 5*time.Second)
		require.NoError(t, err)

		mu.Lock()
		assert.Equal(t, []time.Duration{5 * time.Second}, recordedTTL)
		mu.Unlock()
	})

	t.Run("leader_ttl_wins", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex

		store := map[string]string{}

		var recordedTTL []time.Duration

		setterStarted := make(chan struct{})
		releaseSetter := make(chan struct{})
		sf := &cache.SingleflightGetOrSet[string, string]{}

		get := func(_ context.Context, k string) (string, error) {
			mu.Lock()
			defer mu.Unlock()

			if v, ok := store[k]; ok {
				return v, nil
			}

			return "", cache.ErrMiss
		}
		set := func(_ context.Context, k, v string, ttl ...time.Duration) error {
			mu.Lock()
			defer mu.Unlock()

			store[k] = v

			recordedTTL = append(recordedTTL, ttl...)

			return nil
		}
		setter := func(context.Context) (string, error) { //nolint:unparam
			close(setterStarted)
			<-releaseSetter

			return computed, nil
		}

		leaderResult := make(chan error, 1)

		go func() {
			_, err := sf.Do(t.Context(), "k", get, set, setter, 5*time.Second)
			leaderResult <- err
		}()

		// Wait until the leader's setter is blocked, then join with a
		// different ttl — the leader is deterministically first.
		<-setterStarted

		followerResult := make(chan error, 1)

		go func() {
			_, err := sf.Do(t.Context(), "k", get, set, setter, 1*time.Second)
			followerResult <- err
		}()

		// Give the follower time to join the in-flight call.
		time.Sleep(20 * time.Millisecond)
		close(releaseSetter)

		require.NoError(t, <-leaderResult)
		require.NoError(t, <-followerResult)

		mu.Lock()
		assert.Equal(t, []time.Duration{5 * time.Second}, recordedTTL, "the leader's ttl must win")
		mu.Unlock()
	})
}

// TestSingleflightGetOrSet_LeaderCancel verifies that canceling the leader's
// context surfaces the raw context error to the Do caller.
func TestSingleflightGetOrSet_LeaderCancel(t *testing.T) {
	t.Parallel()

	sf := &cache.SingleflightGetOrSet[string, string]{}
	get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
	set := func(context.Context, string, string, ...time.Duration) error { return nil }

	ctx, cancel := context.WithCancel(t.Context())
	setterStarted := make(chan struct{})
	setter := func(ctx context.Context) (string, error) {
		close(setterStarted)
		<-ctx.Done()

		return "", ctx.Err()
	}

	result := make(chan error, 1)

	go func() {
		_, err := sf.Do(ctx, "k", get, set, setter)
		result <- err
	}()

	<-setterStarted
	cancel()

	err := <-result
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, ctx.Err(), err, "Do must return the raw context error")
}

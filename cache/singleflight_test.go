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
func TestSingleflightGetOrSet_SetterRunsOnce(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	store := map[string]string{}
	var setterRuns atomic.Int64
	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(ctx context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := store[k]; ok {
			return v, nil
		}
		return "", cache.ErrMiss
	}
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		store[k] = v
		return nil
	}
	setter := func(context.Context) (string, error) {
		setterRuns.Add(1)
		time.Sleep(3 * time.Millisecond)
		return "computed", nil
	}

	const callers = 50
	var wg sync.WaitGroup
	results := make([]string, callers)
	for i := 0; i < callers; i++ {
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
		assert.Equal(t, "computed", r, "caller %d must receive the computed value", i)
	}

	// Second phase: the value is now cached; Do must return it WITHOUT
	// re-running the setter (a panic here proves the setter was not invoked).
	v, err := sf.Do(t.Context(), "k", get, set, func(context.Context) (string, error) {
		setterRuns.Add(1)
		panic("setter must not run for a cached key")
	})
	require.NoError(t, err)
	assert.Equal(t, "computed", v)
	assert.Equal(t, int64(1), setterRuns.Load(), "cached read must not re-run the setter")
}

// TestSingleflightGetOrSet_PanicRecovery verifies that a panicking setter
// yields a *cache.Panic to every waiter and no panic escapes the Do call
// (SF-05 / D-16 / D-17).
func TestSingleflightGetOrSet_PanicRecovery(t *testing.T) {
	t.Parallel()

	t.Run("string_panic_value", func(t *testing.T) {
		t.Parallel()

		sf := &cache.SingleflightGetOrSet[string, string]{}
		get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
		set := func(context.Context, string, string, ...time.Duration) error { return nil }
		setter := func(context.Context) (string, error) {
			panic("boom")
		}

		const callers = 5
		var wg sync.WaitGroup
		errs := make([]error, callers)
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := sf.Do(t.Context(), "k", get, set, setter)
				errs[i] = err
			}(i)
		}
		wg.Wait()

		for i, err := range errs {
			var p *cache.Panic
			require.ErrorAs(t, err, &p, "waiter %d must receive a *cache.Panic", i)
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
		setter := func(context.Context) (string, error) {
			panic(sentinel)
		}

		_, err := sf.Do(t.Context(), "k", get, set, setter)
		var p *cache.Panic
		require.ErrorAs(t, err, &p)
		assert.ErrorIs(t, err, sentinel, "errors.Is must traverse Panic.Unwrap()")
	})
}

// TestSingleflightGetOrSet_ZeroValueSetter verifies that a setter returning
// the zero value with nil error stores and returns it; deduplicated waiters
// receive the identical value.
func TestSingleflightGetOrSet_ZeroValueSetter(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	store := map[string]string{}
	var setterRuns atomic.Int64
	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(ctx context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := store[k]; ok {
			return v, nil
		}
		return "", cache.ErrMiss
	}
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		store[k] = v
		return nil
	}
	setter := func(context.Context) (string, error) {
		setterRuns.Add(1)
		return "", nil
	}

	const callers = 10
	var wg sync.WaitGroup
	results := make([]string, callers)
	for i := 0; i < callers; i++ {
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
		assert.Equal(t, "", r, "waiter %d must receive the identical zero value", i)
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
func TestSingleflightGetOrSet_ClobberGuard(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	store := map[string]string{}
	var setterRuns atomic.Int64
	var getCalls atomic.Int64
	setLanded := make(chan struct{})

	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(ctx context.Context, k string) (string, error) {
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
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		store[k] = v
		return nil
	}
	setter := func(context.Context) (string, error) {
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
func TestSingleflightGetOrSet_TTLPassthrough(t *testing.T) {
	t.Parallel()

	t.Run("single_leader_ttl", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		var recordedTTL []time.Duration
		sf := &cache.SingleflightGetOrSet[string, string]{}
		get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
		set := func(_ context.Context, _, v string, ttl ...time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			recordedTTL = append(recordedTTL, ttl...)
			return nil
		}
		setter := func(context.Context) (string, error) { return "computed", nil }

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

		get := func(ctx context.Context, k string) (string, error) {
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
		setter := func(context.Context) (string, error) {
			close(setterStarted)
			<-releaseSetter
			return "computed", nil
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
// context surfaces the raw context error to the Do caller (D-05/D-11). The
// assertion is raw-error equality with ctx.Err(); cache.ErrCanceled is a
// 05-02-owned symbol and is intentionally not referenced here.
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
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, ctx.Err(), err, "Do must return the raw context error")
}

// TestSingleflightGetOrSet_DoChan_CanceledWaiter verifies that a waiter whose
// context is canceled while the leader's setter is still running abandons its
// own wait promptly and receives an error matching BOTH cache.ErrCanceled and
// context.Canceled (D-10 / SF-06). It also asserts the leader's computation is
// never disturbed (D-04): the leader still completes, stores the value, and a
// subsequent Get hits it.
func TestSingleflightGetOrSet_DoChan_CanceledWaiter(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	store := map[string]string{}
	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(ctx context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := store[k]; ok {
			return v, nil
		}
		return "", cache.ErrMiss
	}
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		store[k] = v
		return nil
	}

	setterStarted := make(chan struct{})
	releaseSetter := make(chan struct{})
	setter := func(context.Context) (string, error) {
		close(setterStarted)
		<-releaseSetter
		return "computed", nil
	}

	// The leader blocks in its setter (channel-gated).
	leaderResult := make(chan error, 1)
	go func() {
		_, err := sf.Do(t.Context(), "k", get, set, setter)
		leaderResult <- err
	}()
	<-setterStarted

	// The waiter joins as a follower with a cancelable context.
	waiterCtx, cancel := context.WithCancel(t.Context())
	type res struct {
		v   string
		err error
	}
	waiterResult := make(chan res, 1)
	go func() {
		v, err := sf.DoChan(waiterCtx, "k", get, set, setter)
		waiterResult <- res{v, err}
	}()
	// Give the waiter time to join the in-flight singleflight call.
	time.Sleep(20 * time.Millisecond)

	// Cancel the waiter's ctx while the setter is STILL blocked.
	cancel()

	select {
	case r := <-waiterResult:
		require.Error(t, r.err, "canceled waiter must return an error")
		assert.ErrorIs(t, r.err, cache.ErrCanceled, "waiter must match cache.ErrCanceled")
		assert.ErrorIs(t, r.err, context.Canceled, "waiter must match context.Canceled")
	case <-time.After(2 * time.Second):
		t.Fatal("canceled waiter did not return promptly (SF-06)")
	}

	// Release the setter; the leader completes and stores despite the waiter's
	// cancel (D-04).
	close(releaseSetter)
	require.NoError(t, <-leaderResult, "leader must still complete after waiter cancel")

	mu.Lock()
	stored := store["k"]
	mu.Unlock()
	assert.Equal(t, "computed", stored, "leader must store the value after a waiter canceled")

	// A subsequent read hits the stored value without re-running the setter.
	v, err := sf.DoChan(t.Context(), "k", get, set, func(context.Context) (string, error) {
		t.Fatal("setter must not re-run for a stored key")
		return "", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "computed", v)
}

// TestSingleflightGetOrSet_DoChan_ValueIfReady verifies the value-if-ready
// semantics (D-08): a waiter whose context is canceled AFTER the leader's value
// is already available returns the shared value, not ErrCanceled.
func TestSingleflightGetOrSet_DoChan_ValueIfReady(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	store := map[string]string{}
	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(ctx context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := store[k]; ok {
			return v, nil
		}
		return "", cache.ErrMiss
	}
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		store[k] = v
		return nil
	}

	setterStarted := make(chan struct{})
	releaseSetter := make(chan struct{})
	setter := func(context.Context) (string, error) {
		close(setterStarted)
		<-releaseSetter
		return "computed", nil
	}

	// Leader blocks in its setter.
	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		_, _ = sf.Do(t.Context(), "k", get, set, setter)
	}()
	<-setterStarted

	// The waiter joins as a follower.
	waiterCtx, cancel := context.WithCancel(t.Context())
	type res struct {
		v   string
		err error
	}
	waiterResult := make(chan res, 1)
	go func() {
		v, err := sf.DoChan(waiterCtx, "k", get, set, setter)
		waiterResult <- res{v, err}
	}()
	time.Sleep(20 * time.Millisecond)

	// Release the leader; the value becomes ready and is stored.
	close(releaseSetter)
	<-leaderDone

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		_, ok := store["k"]
		return ok
	}, time.Second, time.Millisecond)

	// Only now cancel the waiter's ctx — the value is already ready.
	cancel()

	select {
	case r := <-waiterResult:
		require.NoError(t, r.err, "value-if-ready must not produce an error")
		assert.Equal(t, "computed", r.v, "waiter must receive the shared value")
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not return")
	}
}

// TestSingleflightGetOrSet_DoChan_SharedValue verifies that N concurrent DoChan
// callers on the same key run the setter exactly once and all receive the same
// value (D-20).
func TestSingleflightGetOrSet_DoChan_SharedValue(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	store := map[string]string{}
	var setterRuns atomic.Int64
	sf := &cache.SingleflightGetOrSet[string, string]{}

	get := func(ctx context.Context, k string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := store[k]; ok {
			return v, nil
		}
		return "", cache.ErrMiss
	}
	set := func(ctx context.Context, k, v string, _ ...time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		store[k] = v
		return nil
	}
	setter := func(context.Context) (string, error) {
		setterRuns.Add(1)
		time.Sleep(3 * time.Millisecond)
		return "computed", nil
	}

	const callers = 30
	var wg sync.WaitGroup
	results := make([]string, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := sf.DoChan(t.Context(), "k", get, set, setter)
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
		assert.Equal(t, "computed", r, "caller %d must receive the shared value", i)
	}
}

// TestSingleflightGetOrSet_DoChan_SetterError verifies that a leader's setter
// error is shared with DoChan waiters as the identical error object (spike 003).
func TestSingleflightGetOrSet_DoChan_SetterError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	sf := &cache.SingleflightGetOrSet[string, string]{}
	get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
	set := func(context.Context, string, string, ...time.Duration) error { return nil }
	setter := func(context.Context) (string, error) {
		time.Sleep(3 * time.Millisecond)
		return "", sentinel
	}

	const callers = 10
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := sf.DoChan(t.Context(), "k", get, set, setter)
			errs[i] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.ErrorIs(t, err, sentinel, "waiter %d must receive the setter's exact error", i)
	}
}

// TestSingleflightGetOrSet_DoChan_LeaderCancel verifies the leader path never
// delivers cache.ErrCanceled: when the DoChan caller IS the leader and cancels
// its OWN context, the raw context error surfaces (the setter's ctx.Err())
// rather than the waiter wrap (D-04 / D-11).
func TestSingleflightGetOrSet_DoChan_LeaderCancel(t *testing.T) {
	t.Parallel()

	sf := &cache.SingleflightGetOrSet[string, string]{}
	get := func(context.Context, string) (string, error) { return "", cache.ErrMiss }
	set := func(context.Context, string, string, ...time.Duration) error { return nil }

	ctx, cancel := context.WithCancel(t.Context())
	setterStarted := make(chan struct{})
	setter := func(c context.Context) (string, error) {
		close(setterStarted)
		<-c.Done()
		return "", c.Err()
	}

	result := make(chan error, 1)
	go func() {
		_, err := sf.DoChan(ctx, "k", get, set, setter)
		result <- err
	}()

	<-setterStarted
	cancel()

	select {
	case err := <-result:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled, "leader must receive the raw context error")
		assert.NotErrorIs(t, err, cache.ErrCanceled, "the leader path must never deliver ErrCanceled")
	case <-time.After(2 * time.Second):
		t.Fatal("leader did not return after cancel")
	}
}

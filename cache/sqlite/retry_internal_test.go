package sqlite

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeError is a minimal busyErr implementation carrying a raw result code; it
// stands in for the driver's *sqlite.Error in the classifier matrix without
// importing the driver (the classifier must stay interface-based).
type fakeError struct{ code int }

func (f fakeError) Error() string { return fmt.Sprintf("fake %d", f.code) }
func (f fakeError) Code() int     { return f.code }

// TestIsBusyError covers the classifier matrix: primary code 5, the extended
// busy variants the driver can produce (261 SQLITE_BUSY_RECOVERY, 517
// SQLITE_BUSY_SNAPSHOT), one non-busy coded error, a plain error, and a
// wrapped busy error (errors.As must walk the chain).
func TestIsBusyError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "sqlite_busy", err: fakeError{code: 5}, want: true},
		{name: "extended_busy_recovery", err: fakeError{code: 261}, want: true},
		{name: "extended_busy_snapshot", err: fakeError{code: 517}, want: true},
		{name: "sqlite_error", err: fakeError{code: 1}, want: false},
		{name: "plain_error", err: errors.New("plain"), want: false},
		{name: "wrapped_busy", err: fmt.Errorf("wrapped: %w", fakeError{code: 5}), want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, isBusyError(tc.err))
		})
	}
}

// TestRetryBusyWithin drives the deadline/backoff loop with millisecond
// budgets and a fake fn: busy-then-nil, several-busy-then-nil, non-busy
// immediate return, always-busy budget exhaustion, and ctx cancellation.
func TestRetryBusyWithin(t *testing.T) { //nolint:funlen // one behavior matrix per case, mem_test.go precedent
	t.Parallel()

	t.Run("busy_once_then_nil", func(t *testing.T) {
		t.Parallel()

		calls := 0
		fn := func(context.Context) error {
			calls++
			if calls == 1 {
				return fakeError{code: 5}
			}

			return nil
		}

		err := retryBusyWithin(t.Context(), 100*time.Millisecond, time.Millisecond, fn)
		require.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("busy_three_times_then_nil", func(t *testing.T) {
		t.Parallel()

		calls := 0
		fn := func(context.Context) error {
			calls++
			if calls <= 3 {
				return fakeError{code: 517} // extended busy retried like primary
			}

			return nil
		}

		err := retryBusyWithin(t.Context(), 100*time.Millisecond, time.Millisecond, fn)
		require.NoError(t, err)
		assert.Equal(t, 4, calls)
	})

	t.Run("non_busy_error_returns_immediately", func(t *testing.T) {
		t.Parallel()

		calls := 0
		boom := errors.New("boom")
		fn := func(context.Context) error {
			calls++
			return boom
		}

		err := retryBusyWithin(t.Context(), time.Second, time.Millisecond, fn)
		require.ErrorIs(t, err, boom)
		assert.Equal(t, 1, calls, "a non-busy error must never enter the retry loop")
	})

	t.Run("always_busy_exhausts_budget", func(t *testing.T) {
		t.Parallel()

		calls := 0
		fn := func(context.Context) error {
			calls++
			return fakeError{code: 5}
		}

		err := retryBusyWithin(t.Context(), 20*time.Millisecond, time.Millisecond, fn)
		require.Error(t, err)
		assert.True(t, isBusyError(err), "exhaustion must surface the last busy error")
		assert.GreaterOrEqual(t, calls, 2, "a 20 ms budget with a 1 ms backoff must attempt more than once")
	})

	t.Run("canceled_ctx_short_circuits", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		fn := func(context.Context) error { return fakeError{code: 5} }

		go func() {
			time.Sleep(2 * time.Millisecond)
			cancel()
		}()

		err := retryBusyWithin(ctx, 5*time.Second, time.Millisecond, fn)
		require.ErrorIs(t, err, context.Canceled)
	})
}

// TestBusyTimeoutDSNSync mechanically pins the retry budget to the DSN: the
// busy_timeout=NNNN literal parsed out of dsnSuffixFile must equal
// busyTimeout in milliseconds — no silent drift between retry.go and dsn.go.
func TestBusyTimeoutDSNSync(t *testing.T) {
	t.Parallel()

	const marker = "_busy_timeout="

	start := strings.Index(dsnSuffixFile, marker)
	require.Positive(t, start, "dsnSuffixFile must carry the busy_timeout key")

	rest := dsnSuffixFile[start+len(marker):]
	if end := strings.IndexByte(rest, '&'); end >= 0 {
		rest = rest[:end]
	}

	dsnMillis, err := strconv.Atoi(rest)
	require.NoError(t, err)

	assert.Equal(t, int(busyTimeout.Milliseconds()), dsnMillis)
	assert.Equal(t, 5000, dsnMillis)
}

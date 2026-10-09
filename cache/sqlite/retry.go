package sqlite

import (
	"context"
	"errors"
	"time"
)

// busyErr is the minimal interface the driver's error type satisfies
// (modernc.org/sqlite *Error exposes Code() int). Classifying through the
// interface keeps the driver import blank (it stays only in sqlite.go) and
// makes the classifier unit-testable with a fake error (Finding 3).
type busyErr interface{ Code() int }

const (
	// busyTimeout bounds the open-window retry. It mirrors the DSN's
	// _busy_timeout=5000 literal (dsn.go); TestBusyTimeoutDSNSync pins the two
	// together so they cannot drift apart.
	busyTimeout = 5 * time.Second //nolint:mnd

	// busyBackoff is the wait between open-retry attempts (D-06: tiny backoff).
	busyBackoff = 5 * time.Millisecond //nolint:mnd

	// sqliteBusyCode is the primary SQLITE_BUSY result code; sqliteBusyMask
	// isolates the primary code from the extended result codes the driver
	// enables per connection (extended variants such as SQLITE_BUSY_SNAPSHOT
	// 517 and SQLITE_BUSY_RECOVERY 261 keep the primary code in the low byte).
	sqliteBusyCode = 5    //nolint:mnd
	sqliteBusyMask = 0xff //nolint:mnd
)

// isBusyError reports whether err is SQLITE_BUSY by primary code. The driver
// enables extended result codes, so the comparison masks down to the primary
// code — never string matching, which is locale/version-fragile.
func isBusyError(err error) bool {
	var be busyErr
	return errors.As(err, &be) && be.Code()&sqliteBusyMask == sqliteBusyCode
}

// retryBusy runs fn, retrying only busy-classified errors within the
// busyTimeout budget (D-06). It is the single production entry point, used
// exactly once: the open-window PingContext in open().
func retryBusy(ctx context.Context, fn func(context.Context) error) error {
	return retryBusyWithin(ctx, busyTimeout, busyBackoff, fn)
}

// retryBusyWithin calls fn until it succeeds, fails with a non-busy error, or
// the absolute budget expires. The deadline is checked before every attempt, so
// once the budget is spent no further fn call is started; exhaustion returns
// the last busy error. A canceled context short-circuits with ctx.Err(). The
// deadline is absolute, so the total wait is bounded regardless of retry count.
func retryBusyWithin(ctx context.Context, budget, backoff time.Duration, fn func(context.Context) error) error {
	deadline := time.Now().Add(budget)

	var lastErr error

	for {
		if lastErr != nil && time.Now().After(deadline) {
			return lastErr // budget exhausted: surface the last busy error, no extra attempt
		}

		err := fn(ctx)
		if err == nil || !isBusyError(err) || time.Now().After(deadline) {
			return err
		}

		lastErr = err

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}

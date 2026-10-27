package cache

import (
	"bytes"
	"errors"
	"fmt"
	"runtime/debug"
)

// PanicError is returned by SingleflightGetOrSet.Do when a setter panics.
// It wraps the recovered panic value and the stack trace captured at
// recovery time, mirroring x/sync/singleflight's panicError. Error() formats
// the value plus the stack; Unwrap() returns the value when it is an error so
// errors.Is/errors.As traverse through the recovered value.
type PanicError struct {
	// Value is the value recovered from the panic.
	Value any
	// Stack is the stack trace captured when the panic was recovered.
	Stack []byte
}

// Error implements the error interface.
func (p *PanicError) Error() string {
	return fmt.Sprintf("%v\n\n%s", p.Value, p.Stack)
}

// Unwrap returns the recovered value when it is an error, enabling
// errors.Is and errors.As to traverse through the panic value.
func (p *PanicError) Unwrap() error {
	err, ok := p.Value.(error)
	if !ok {
		return nil
	}

	return err
}

var (
	// ErrMiss is returned by Get when the key is not in the cache.
	ErrMiss = errors.New("cache: key not found")

	// ErrClosed is the sentinel for an operation attempted on a closed cache.
	// Providers do not share one closed-error value: cache/sqlite wraps this
	// sentinel (so errors.Is(err, ErrClosed) matches after unwrapping); redis,
	// valkey and postgres surface their client library's own closed error (a
	// closed client or pool); memcache has no closed state at all, its Close is
	// a documented no-op; and mem reports misses rather than a closed state.
	// Compare with errors.Is, never with ==.
	ErrClosed = errors.New("cache: cache is closed")
)

// newPanic builds a *PanicError from a recovered value, trimming the first
// "goroutine N [status]:" line of the stack like x/sync's newPanicError does
// (by recovery time the goroutine may no longer exist and its status may have
// changed).
func newPanic(v any) *PanicError {
	stack := debug.Stack()
	if line := bytes.IndexByte(stack, '\n'); line >= 0 {
		stack = stack[line+1:]
	}

	return &PanicError{Value: v, Stack: stack}
}

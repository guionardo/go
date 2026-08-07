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

	// ErrClosed is returned when operations are attempted on a closed cache.
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

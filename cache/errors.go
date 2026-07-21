package cache

import (
	"errors"
)

var (
	// ErrMiss is returned by Get when the key is not in the cache.
	ErrMiss = errors.New("cache: key not found")

	// ErrClosed is returned when operations are attempted on a closed cache.
	ErrClosed = errors.New("cache: cache is closed")
)

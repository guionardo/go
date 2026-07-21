package mem

import (
	"time"
)

type (
	// entry holds a cached value with optional expiration.
	entry[V any] struct {
		value     V
		expiresAt *time.Time // nil means no expiry
	}
)

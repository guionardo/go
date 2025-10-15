package set

import (
	"database/sql/driver"
	"errors"
)

// Scan will receive data as string or []byte, marshaled as JSON (e.g. '["a","b","c"]')
func (s Set[T]) Scan(value any) error {
	switch v := value.(type) {
	case string:
		return s.UnmarshalJSON([]byte(v))
	case []byte:
		return s.UnmarshalJSON(v)
	default:
		return errors.New("invalid type for scan")
	}
}

// Value will produce []byte JSON from this set
func (s Set[T]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

package timetools

import (
	"errors"
	"sync/atomic"
	"time"
)

// layouts holds all supported Go time layout strings, ordered by
// priority. Successful parses promote the matched template to the
// front of the list for faster matching on subsequent calls.
var (
	layouts atomic.Pointer[[]string]

	// ErrTimeParser is returned when Parse cannot match the input
	// string against any known time layout.
	ErrTimeParser = errors.New("failed to parse time.Time value")
)

func init() {
	layouts.Store(&[]string{
		time.DateTime,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
		time.DateOnly,
		time.TimeOnly,
	})
}

// Parse attempts to parse the input string s using a prioritized list
// of Go time layouts (RFC3339, DateTime, DateOnly, Kitchen, etc.).
// The first successful match is returned. On success, the matched
// layout is promoted to the front of the list so that frequently used
// formats are checked first in future calls.
// Returns ErrTimeParser if no layout matches the input.
func Parse(s string) (time.Time, error) {
	current := *layouts.Load()

	for i := range current {
		t, err := time.Parse(current[i], s)
		if err == nil {
			if i > 0 {
				promoted := make([]string, len(current))
				copy(promoted, current)
				for n := i; n > 0; n-- {
					promoted[n] = promoted[n-1]
				}
				promoted[0] = current[i]
				layouts.Store(&promoted)
			}

			return t, nil
		}
	}

	return time.Time{}, ErrTimeParser
}

// SetLayouts replaces the global layouts list with a new set of time
// format strings. This is useful for customizing which layouts Parse
// will attempt and in what order.
func SetLayouts(newLayouts []string) {
	layouts.Store(&newLayouts)
}

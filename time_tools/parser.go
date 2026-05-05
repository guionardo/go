// Package timetools provides utilities for parsing time.Time values
// using multiple common time formats. The parser auto-prioritizes
// templates based on successful parses, optimizing subsequent calls.
package timetools

import (
	"errors"
	"sync"
	"time"
)

// layouts holds all supported Go time layout strings, ordered by
// priority. Successful parses promote the matched template to the
// front of the list for faster matching on subsequent calls.
var (
	layouts = []string{
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
	}
	layoutsLock sync.RWMutex

	// ErrTimeParser is returned when Parse cannot match the input
	// string against any known time layout.
	ErrTimeParser = errors.New("failed to parse time.Time value")
)

// Parse attempts to parse the input string s using a prioritized list
// of Go time layouts (RFC3339, DateTime, DateOnly, Kitchen, etc.).
// The first successful match is returned. On success, the matched
// layout is promoted to the front of the list (asynchronously) so
// that frequently used formats are checked first in future calls.
// Returns ErrTimeParser if no layout matches the input.
func Parse(s string) (time.Time, error) {
	layoutsLock.RLock()
	defer layoutsLock.RUnlock()

	for index := range layouts {
		t, err := time.Parse(layouts[index], s)
		if err == nil {
			if index > 0 {
				go func(index int, layout string) {
					layoutsLock.Lock()
					defer layoutsLock.Unlock()

					if index >= len(layouts) || layouts[index] != layout {
						return
					}

					for n := index; n > 0; n-- {
						layouts[n] = layouts[n-1]
					}

					layouts[0] = layout
				}(index, layouts[index])
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
	layoutsLock.Lock()
	defer layoutsLock.Unlock()

	layouts = newLayouts
}

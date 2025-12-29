package httptestmock

import (
	"fmt"
	"slices"
	"strings"

	reflecttools "github.com/guionardo/go/reflect_tools"
)

type (
	StringParts []stringPart

	stringPart struct {
		key   string
		value any
	}
)

const defaultStringPartsCapacity = 16

// String returns a human-readable representation of the string parts.

func (s StringParts) String() string {
	var (
		parts = make([]string, 0, len(s))
		value string
	)

	for _, part := range s {
		if reflecttools.IsZeroValue(part.value) {
			continue
		}

		switch v := part.value.(type) {
		case string:
			value = v

		case fmt.Stringer:
			value = v.String()

		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			value = fmt.Sprintf("%d", v)

		case float32, float64, bool:
			value = fmt.Sprintf("%v", v)

		default:
			value = fmt.Sprintf("%+v", v)
		}

		parts = append(parts, fmt.Sprintf("[%s: %s]", part.key, value))
	}

	return strings.Join(parts, " ")
}

func (s StringParts) Set(key string, value any) StringParts {
	if s == nil {
		s = make(StringParts, 0, defaultStringPartsCapacity)
	}

	if p := slices.IndexFunc(s, func(p stringPart) bool {
		return p.key == key
	}); p == -1 {
		s = append(s, stringPart{key: key, value: value})
	} else {
		s[p].value = value
	}

	return s
}

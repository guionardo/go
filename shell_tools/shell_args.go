package shelltools

import (
	"strings"
)

// QuotedShellArgs splits a string into arguments, respecting quoted substrings.
// Supports single and double quotes and backslash escaping (outside or inside double quotes).
// Quotes are removed from returned arguments. All unicode whitespace is treated as separator.
type QuotedShellArgs []string

func NewQuotedShellArgs(s string) QuotedShellArgs {
	parts := make(QuotedShellArgs, 0)
	s = strings.TrimSpace(s)
	// parse escaped spaces
	s = strings.ReplaceAll(s, `\ `, `\0`)

	for len(s) > 0 {
		// search for next quote
		pq := strings.IndexAny(s, `"'`)
		if pq == -1 {
			// no more quotes
			parts = append(parts, strings.Fields(s)...)
			break
		}

		if pq != 0 {
			// add preceding unquoted word (if any)
			if fields := strings.Fields(s); len(fields) > 0 {
				parts = append(parts, fields[0])
				s = s[len(fields[0]):]
			} else {
				// fallback to avoid infinite loop
				s = s[1:]
			}

			s = strings.TrimSpace(s)

			continue
		}

		// pq == 0: extract quoted argument and remaining string
		arg, rest := extractQuotedPrefix(s)
		parts = append(parts, arg)
		s = strings.TrimSpace(rest)
	}

	// restore escaped spaces
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(part, `\0`, ` `)
	}

	return parts
}

// extractQuotedPrefix assumes s starts with a quote (single or double) and returns
// the unquoted argument and the remaining string after consuming the closing quote
// and an optional following space; if no closing quote is found, the rest is empty.
func extractQuotedPrefix(s string) (string, string) {
	if len(s) == 0 {
		return "", ""
	}

	quote := s[0]
	restSub := s[1:]
	// prefer closing quote followed by space
	if p2 := strings.Index(restSub, string(quote)+" "); p2 > -1 {
		arg := restSub[:p2]
		rest := s[p2+2:] // skip closing quote and following space

		return arg, rest
	}
	// otherwise find any closing quote
	if p1 := strings.Index(restSub, string(quote)); p1 > -1 {
		arg := restSub[:p1]
		rest := s[p1+2:] // skip closing quote

		return arg, rest
	}
	// no closing quote found, treat rest as single argument
	return restSub, ""
}

func (q QuotedShellArgs) String() string {
	if len(q) < 2 { //nolint:mnd
		return strings.Join(q, " ")
	}

	var sb strings.Builder

	for i, arg := range q {
		if i > 0 {
			sb.WriteRune(' ')
		}

		if !strings.Contains(arg, " ") {
			sb.WriteString(arg)
			continue
		}

		if strings.Contains(arg, "\"") {
			sb.WriteString("'" + arg + "'")
			continue
		}

		sb.WriteString("\"" + arg + "\"")
	}

	return sb.String()
}

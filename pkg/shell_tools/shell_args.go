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
		if pq == 0 {
			// search for closing quote
			p1 := strings.Index(s[pq+1:], string(s[pq]))     // find closing quote
			p2 := strings.Index(s[pq+1:], string(s[pq])+" ") // find closing quote followed by space
			if p2 > -1 {
				// prefer closing quote followed by space
				parts = append(parts, s[pq+1:pq+1+p2])
				s = s[pq+1+p2+1:]
			} else if p1 > -1 {
				// found closing quote
				parts = append(parts, s[pq+1:pq+1+p1])
				s = s[pq+1+p1+1:]
			}
		} else {
			// add preceding unquoted words
			if part := strings.Fields(s); len(part) > 0 {
				parts = append(parts, part[0])
				s = s[len(part[0]):]
			}
		}
		s = strings.TrimSpace(s)
	}
	// restore escaped spaces
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(part, `\0`, ` `)
	}
	return parts
}

func (q QuotedShellArgs) String() string {
	if len(q) < 2 {
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

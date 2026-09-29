package projectprobe

import (
	"bytes"
	"strings"
)

// readTOMLSection returns the key/values of the named TOML section.
// Supported grammar (D-01): [name] headers (dotted names like
// "tool.poetry" match exactly), bare or "quoted" keys, "="/'=' separator,
// "…"/'…' quoted scalar values, # comments, blank lines. Legal-but-
// unsupported TOML — multi-line strings, dotted keys, arrays, inline
// tables, unquoted values — is never stored, so the caller reads "" for
// those keys (D-02 strict degrade: no panic, no partial data). A missing
// section returns an empty map, never an error (D-03). Content is
// BOM-tolerant (readManifest strips upstream; the raw helper handles it too).
// Skip-state scanning is quote-aware (CR-01): delimiter runs inside bodies
// — a 6-quote close+reopen line, a 4-quote opening line, or a ]/} inside a
// quoted string — never clear a skip state early, so body lines can never
// fabricate headers or keyvals.
func readTOMLSection(content []byte, section string) map[string]string {
	out := make(map[string]string)
	var current string
	var skipDelim string  // multi-line string delimiter when non-empty
	var skipClose rune    // ']' or '}' when in bracket-skip state
	var pendingMLS string // multi-line string delimiter open inside a bracket body
	for _, line := range strings.Split(string(bytes.TrimPrefix(content, utf8BOM)), "\n") {
		line = strings.TrimSpace(line) // handles CRLF (P5)
		switch {
		case skipDelim != "":
			if closesMultiLine(line, skipDelim) {
				skipDelim = ""
			}
			continue
		case skipClose != 0:
			if pendingMLS != "" {
				// a multi-line string opened inside the bracket body: only its
				// own delimiter's runs matter — a ]/} on ANY line inside the
				// string never clears the bracket state (CR-01 fix).
				if closesMultiLine(line, pendingMLS) {
					pendingMLS = ""
				}
				continue
			}
			if delim, open := opensMultiLine(line); open {
				// a multi-line string opener inside the bracket body: every
				// subsequent line stays inside the string until a net-closed line.
				pendingMLS = delim
				continue
			}
			if clearsBracket(line, skipClose) {
				skipClose = 0
			}
			continue
		case strings.HasPrefix(line, "[["): // array-of-tables header
			current = headerName(line[2:], "]]")
		case strings.HasPrefix(line, "["): // table header
			current = headerName(line[1:], "]")
		case !strings.Contains(line, "="):
			continue
		default:
			key, value, ok := parseKeyValue(line)
			skipDelim, skipClose = enterSkip(value) // global skip states (D-disc-4)
			if skipClose != 0 && pendingMLS == "" {
				// a multi-line string opener ON the bracket's opening line
				// (e.g. `authors = [ """`) keeps later lines inside the string.
				if delim, open := opensMultiLine(value[1:]); open {
					pendingMLS = delim
				}
			}
			if !ok || current != section {
				continue
			}
			out[key] = value
		}
	}
	return out
}

// headerName extracts the section name between the first opening bracket
// (already consumed by the caller) and the first closing delimiter, trimmed
// (D-disc-8). A trailing "# comment" after the closing delimiter and inner
// padding ("[ project ]") are handled by construction; quoted header segments
// and inner-dot whitespace forms simply never equal a requested section and
// are documented non-matches (A3).
func headerName(rest, closeDelim string) string {
	if i := strings.Index(rest, closeDelim); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

// parseKeyValue extracts a key/value pair from a keyval line. The key is
// validated FIRST (anti-pattern: naive SplitN before key validation — a
// quoted key may contain "="): bare keys must match [A-Za-z0-9_-]+ (a "."
// is a dotted key, never stored, D-07), quoted keys are stripped of their
// outer quotes. Only the exact names name/version/description are stored.
// Unsupported value forms — multi-line strings, arrays, inline tables,
// unquoted values — return ok=false so the key is never stored (D-02).
func parseKeyValue(line string) (key, value string, ok bool) {
	if line[0] == '"' || line[0] == '\'' {
		q := line[0]
		end := strings.IndexByte(line[1:], q)
		if end < 0 {
			return "", "", false // unterminated quoted key
		}
		key = line[1 : end+1]
		rest := strings.TrimSpace(line[end+2:])
		if !strings.HasPrefix(rest, "=") {
			return "", "", false // not a keyval
		}
		value = strings.TrimSpace(rest[1:])
	} else {
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return "", "", false
		}
		key = strings.TrimSpace(line[:eq])
		value = strings.TrimSpace(line[eq+1:])
	}
	if !isBareKey(key) || key != "name" && key != "version" && key != "description" {
		return key, value, false
	}
	if strings.HasPrefix(value, `"""`) || strings.HasPrefix(value, `'''`) ||
		strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		return key, value, false // multi-line string/array/inline table: never stored
	}
	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
		v, ok := quotedValue(value)
		return key, v, ok
	}
	return key, value, false // unquoted or unspecified value: never stored
}

// isBareKey reports whether key matches the TOML bare-key grammar
// [A-Za-z0-9_-]+. A "." is not in that set, so dotted keys like
// version.workspace fail here and are never stored (D-07).
func isBareKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

// quotedValue extracts the raw interior of a "…" or '…' value — outer
// quotes stripped, interior escapes kept verbatim (D-disc-5, A4). For basic
// strings the character after a backslash is skipped so \" and \\ never
// close the string. After the closing quote the remainder must be whitespace
// or a "# comment", else the key is not stored (D-disc-6 — never partial
// data). An unterminated quote returns ok=false.
func quotedValue(v string) (string, bool) {
	q := v[0]
	for i := 1; i < len(v); i++ {
		c := v[i]
		if c == '\\' && q == '"' {
			i++ // skip the escaped character
			continue
		}
		if c == q {
			rest := strings.TrimSpace(v[i+1:])
			if rest != "" && !strings.HasPrefix(rest, "#") {
				return "", false
			}
			return v[1:i], true
		}
	}
	return "", false // unterminated
}

// enterSkip reports the global skip state a value opens: skipDelim is the
// multi-line string delimiter (""" or ”') when the string spans lines;
// skipClose is the rune (] or }) of a multi-line array or inline table that
// is still open. Constructs closed on the same line enter no state (P4).
// The state is GLOBAL — entered from keyval lines in ANY section (D-disc-4)
// — so string/array/inline-table bodies can never fabricate headers or
// keyvals (SC4, T-12-02). The same-line close checks are quote-aware (CR-01):
// a 4-quote run on the opening line (close+reopen) ENTERS the multi-line
// string state, and a ]/} inside a quoted string on the opening line does
// NOT close the construct — the bracket state is entered.
func enterSkip(value string) (skipDelim string, skipClose rune) {
	switch {
	case strings.HasPrefix(value, `"""`) || strings.HasPrefix(value, `'''`):
		if !closesMultiLine(value[3:], value[:3]) {
			return value[:3], 0
		}
	case strings.HasPrefix(value, "["):
		if !clearsBracket(value[1:], ']') {
			return "", ']'
		}
	case strings.HasPrefix(value, "{"):
		if !clearsBracket(value[1:], '}') {
			return "", '}'
		}
	}
	return "", 0
}

// closesMultiLine reports whether line nets the multi-line string delim
// CLOSED — i.e. the skip state must clear. Scan the line left-to-right
// counting consecutive runs of delim[0]: a run of length 1-2 is content
// (no effect); a run of exactly 3 closes the string (toggles the open
// state); a run of length > 3 closes AND reopens (legal TOML close+reopen —
// the string continues on the next line), so the state stays open. A
// `""""""` body line therefore keeps the skip state (6 = close+reopen) while
// a plain `"""` line clears it (3 = close); a 6-quote run then a later
// 3-quote run nets to closed.
func closesMultiLine(line, delim string) bool {
	q := delim[0]
	open := true // the caller is inside the string when this is called
	for i := 0; i < len(line); i++ {
		if line[i] != q {
			continue
		}
		run := 1
		for i+run < len(line) && line[i+run] == q {
			run++
		}
		switch {
		case run == 3:
			open = !open
		case run > 3:
			open = true // close+reopen nets open
		}
		i += run - 1
	}
	return !open
}

// clearsBracket reports whether line contains closeRune as the construct's
// own closer: a closeRune OUTSIDE any " or ' quoted string at bracket depth
// 0. Walk the line byte-by-byte tracking (1) insideDouble/insideSingle —
// basic strings honor \ escapes (a backslash inside a double-quoted string
// skips the next byte), literal strings have no escapes; (2) bracket depth —
// [ and { outside any string increment, ] and } outside any string decrement
// (a nested array/table inside an inline table, or vice versa, never clears
// the outer state). A closeRune outside a string at depth > 0 is a nested
// close and only decrements; a closeRune inside a quoted string never
// qualifies at any depth and never affects depth, so "Alice ] Bob" and
// version = "0.0.1 }" no longer clear their states.
func clearsBracket(line string, closeRune rune) bool {
	depth := 0
	inDouble := false
	inSingle := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case inDouble:
			if c == '\\' {
				i++ // skip the escaped character
				continue
			}
			if c == '"' {
				inDouble = false
			}
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			if c == byte(closeRune) && depth == 0 {
				return true
			}
			if depth > 0 {
				depth--
			}
		}
	}
	return false
}

// opensMultiLine reports whether line opens a multi-line string that is still
// open at the end of the line (net open), and returns its delimiter. The same
// quote-aware walk as clearsBracket (insideDouble/insideSingle with \ escapes)
// counts consecutive quote runs: a run of >= 3 of the same quote char
// occurring OUTSIDE any basic/literal string opens a multi-line string (a run
// > 3 is close+reopen — still open; a run of 1-2 is a basic/literal string
// toggle, not an opener). Returns ("", false) when no net-open multi-line
// string is found — `"""` alone nets open, `"""text"""` nets closed. While a
// multi-line string is pending, only its own delimiter's runs matter: a ”'
// run inside a pending """ string is content, not an opener.
func opensMultiLine(line string) (string, bool) {
	var delim byte
	open := false
	inDouble := false
	inSingle := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case inDouble:
			if c == '\\' {
				i++ // skip the escaped character
				continue
			}
			if c == '"' {
				inDouble = false
			}
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case c == '"' || c == '\'':
			run := 1
			for i+run < len(line) && line[i+run] == c {
				run++
			}
			if delim != 0 {
				// already inside a multi-line string on this line: only its
				// own delimiter's runs matter.
				if c == delim {
					switch {
					case run == 3:
						open = !open
					case run > 3:
						open = true
					}
				}
				i += run - 1
				continue
			}
			if run >= 3 {
				delim = c
				open = true // run > 3 is close+reopen — still open
				i += run - 1
				continue
			}
			// run of 1-2: basic/literal string toggle, not an opener.
			if c == '"' {
				inDouble = true
			} else {
				inSingle = true
			}
			i += run - 1
		}
	}
	return string(delim), open
}

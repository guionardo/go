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
func readTOMLSection(content []byte, section string) map[string]string {
	out := make(map[string]string)
	var current string
	var skipDelim string // multi-line string delimiter when non-empty
	var skipClose rune   // ']' or '}' when in bracket-skip state
	for _, line := range strings.Split(string(bytes.TrimPrefix(content, utf8BOM)), "\n") {
		line = strings.TrimSpace(line) // handles CRLF (P5)
		switch {
		case skipDelim != "":
			if strings.Contains(line, skipDelim) {
				skipDelim = ""
			}
			continue
		case skipClose != 0:
			if strings.ContainsRune(line, skipClose) {
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
// multi-line string delimiter (""" or ''') when the string spans lines;
// skipClose is the rune (] or }) of a multi-line array or inline table that
// is still open. Constructs closed on the same line enter no state (P4).
// The state is GLOBAL — entered from keyval lines in ANY section (D-disc-4)
// — so string/array/inline-table bodies can never fabricate headers or
// keyvals (SC4, T-12-02).
func enterSkip(value string) (skipDelim string, skipClose rune) {
	switch {
	case strings.HasPrefix(value, `"""`) || strings.HasPrefix(value, `'''`):
		if !strings.Contains(value[3:], value[:3]) {
			return value[:3], 0
		}
	case strings.HasPrefix(value, "["):
		if !strings.ContainsRune(value[1:], ']') {
			return "", ']'
		}
	case strings.HasPrefix(value, "{"):
		if !strings.ContainsRune(value[1:], '}') {
			return "", '}'
		}
	}
	return "", 0
}
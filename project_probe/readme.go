package projectprobe

import (
	"os"
	"strings"
)

const (
	// minUnderlineLen is the shortest accepted underline (D-disc-3): two
	// characters can be prose ("==", "::"), so underlines need at least three.
	minUnderlineLen = 3

	// maxHeadingLevel is the deepest ATX heading (markdown allows 1-6 hashes).
	maxHeadingLevel = 6
)

// readmeCandidates are the README names probed in locked order (D-08).
// Matching is exact-case by contract — case-folding would break D-08
// (Pitfall 7).
var readmeCandidates = []string{"README.md", "README.rst", "README"}

// readmeDescription returns the first real paragraph of the folder's README.
// Candidates are README.md → README.rst → README, exact-case (D-08); every
// candidate is read through readManifest, so the 1 MB cap and BOM strip
// (ROBT-02) apply. The first readable candidate wins; "" when none exists
// (D-10). Exact-case is enforced against the real directory entries —
// os.Open alone resolves case-insensitively on macOS/Windows volumes, which
// would silently break the D-08 contract (Pitfall 7).
func readmeDescription(folder string) string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return "" // unreadable folder → empty, never an error (D-10)
	}
	names := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		names[entry.Name()] = struct{}{}
	}
	for _, name := range readmeCandidates {
		if _, ok := names[name]; !ok {
			continue
		}
		if content, ok := readManifest(folder, name); ok {
			return firstRealParagraph(content)
		}
	}

	return ""
}

// firstRealParagraph returns the first consecutive non-blank text block of
// content after skipping badge lines, TOC links, ATX headings, underline
// heading pairs, and HTML comment preambles (D-09 + D-disc-3/4/5). Blocks are
// joined with a single space; inline markdown markers are preserved verbatim
// (D-disc-4). "" when no block exists.
func firstRealParagraph(content []byte) string {
	var para []string
	var prev string // last non-blank line (candidate heading title)
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			if len(para) > 0 {
				return strings.Join(para, " ") // first block is complete
			}
			prev = ""
		case isUnderline(trimmed):
			// Discard the underline AND the preceding non-blank line —
			// an rst/setext heading pair.
			if prev != "" && len(para) > 0 && para[len(para)-1] == prev {
				para = para[:len(para)-1]
			}
			prev = ""
		case isBadgeLine(trimmed) || isTOCLine(trimmed) || isHeading(trimmed) ||
			(strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->")):
			prev = ""
		default:
			para = append(para, trimmed)
			prev = trimmed
		}
	}
	if len(para) == 0 {
		return ""
	}

	return strings.Join(para, " ")
}

// isUnderline reports whether line is an rst/setext underline: at least
// minUnderlineLen identical characters from the set = - ~ ^ _ * + # ' "
// backtick. "." and ":" are excluded so ellipsis and "::" stay prose
// (D-disc-3).
func isUnderline(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < minUnderlineLen {
		return false
	}
	first := line[0]
	if !strings.ContainsRune("=-~^_*+#'\"`", rune(first)) {
		return false
	}
	for i := 1; i < len(line); i++ {
		if line[i] != first {
			return false
		}
	}

	return true
}

// isBadgeLine reports whether line is an image-only badge: its markdown link
// segments [...] (...) — including the wrapped [![...](...)](...) form — strip
// to empty. A badge followed by real text is NOT a badge line (D-09).
func isBadgeLine(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.Contains(line, "![") {
		return false
	}
	for {
		linkEnd := strings.Index(line, "](")
		if linkEnd < 0 {
			break
		}
		start := strings.LastIndex(line[:linkEnd], "[")
		if start < 0 {
			break
		}
		urlEnd := strings.Index(line[linkEnd+1:], ")")
		if urlEnd < 0 {
			break
		}
		line = line[:start] + line[linkEnd+1+urlEnd+1:]
	}

	return strings.TrimSpace(line) == ""
}

// isTOCLine reports whether line is a markdown TOC link: a bullet link
// ("- [" or "* [") or a numbered link (digits, then "." or ")", then a space,
// then "[").
func isTOCLine(line string) bool {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "- [") || strings.HasPrefix(line, "* [") {
		return true
	}
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(line) {
		return false
	}
	if line[i] != '.' && line[i] != ')' {
		return false
	}
	i++
	if i >= len(line) || line[i] != ' ' {
		return false
	}

	return strings.HasPrefix(line[i+1:], "[")
}

// isHeading reports whether line is an ATX heading: 1-6 leading "#" characters
// followed by a space (D-disc-5).
func isHeading(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '#' {
		return false
	}
	i := 0
	for i < len(line) && i < maxHeadingLevel && line[i] == '#' {
		i++
	}

	return i < len(line) && line[i] == ' '
}
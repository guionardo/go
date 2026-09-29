package projectprobe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeReadme writes content to <folder>/<name>, failing the test on error.
// Inline fixtures via t.TempDir + os.WriteFile — the repo convention
// (D-disc-6, no testdata/ dirs).
func writeReadme(t *testing.T, folder, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(folder, name), []byte(content), 0o600))
}

// TestReadmeDescription_Candidates pins the D-08 candidate chain: README.md →
// README.rst → README, exact-case, first existing wins; a lowercase readme.md
// alone yields "" (Pitfall 7); no README at all yields "" (D-10).
func TestReadmeDescription_Candidates(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) (folder, want string)
	}{
		{
			name: "md_wins",
			setup: func(t *testing.T) (string, string) {
				folder := t.TempDir()
				writeReadme(t, folder, "README.md", "md content paragraph\n")
				writeReadme(t, folder, "README.rst", "rst content paragraph\n")
				writeReadme(t, folder, "README", "plain content paragraph\n")

				return folder, "md content paragraph"
			},
		},
		{
			name: "rst_fallback",
			setup: func(t *testing.T) (string, string) {
				folder := t.TempDir()
				writeReadme(t, folder, "README.rst", "rst content paragraph\n")

				return folder, "rst content paragraph"
			},
		},
		{
			name: "plain_fallback",
			setup: func(t *testing.T) (string, string) {
				folder := t.TempDir()
				writeReadme(t, folder, "README", "plain content paragraph\n")

				return folder, "plain content paragraph"
			},
		},
		{
			name: "exact_case",
			setup: func(t *testing.T) (string, string) {
				folder := t.TempDir()
				writeReadme(t, folder, "readme.md", "lowercase content paragraph\n")

				return folder, ""
			},
		},
		{
			name: "none",
			setup: func(t *testing.T) (string, string) {
				return t.TempDir(), ""
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			folder, want := tt.setup(t)

			assert.Equal(t, want, readmeDescription(folder))
		})
	}
}

// TestReadmeDescription_Realistic runs a badge line, a TOC bullet, an ATX
// heading, an rst underline pair, and a two-line paragraph through the full
// candidate chain — the paragraph must come out joined with a single space
// (D-09 + D-disc-4).
func TestReadmeDescription_Realistic(t *testing.T) {
	t.Parallel()

	folder := t.TempDir()
	writeReadme(t, folder, "README.md", `[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
- [TOC](#toc)
# My Project
Title
=====

First paragraph line one.
First paragraph line two.
`)

	assert.Equal(t, "First paragraph line one. First paragraph line two.", readmeDescription(folder))
}

// TestReadmeDescription_NoiseOnly pins the D-10 empty outcome: a README made
// of only badges, TOC links, and headings has no real paragraph → "".
func TestReadmeDescription_NoiseOnly(t *testing.T) {
	t.Parallel()

	folder := t.TempDir()
	writeReadme(t, folder, "README.md", `[![logo](x)](y)
- [TOC](#toc)
1. [TOC](#toc)
# Heading
`)

	assert.Equal(t, "", readmeDescription(folder))
}

// TestFirstRealParagraph pins the full README extraction matrix (RESEARCH
// Pitfall 5 + Pattern 4): every skip threshold — badge-only vs badge-plus-text,
// bullet and numbered TOC links, rst/setext underline pairs, ATX headings,
// HTML comment preambles, ellipsis/colon-colon non-headings, thematic breaks,
// multi-line joins, empty and whitespace-only inputs, and the 3-char underline
// floor (D-09 + D-disc-3/4/5).
func TestFirstRealParagraph(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"badge_only", "[![logo](x)](y)\n", ""},
		{"badge_plus_text", "[![logo](x)](y) Welcome!\n", "[![logo](x)](y) Welcome!"},
		{"bullet_toc", "- [TOC](#x)\npara\n", "para"},
		{"numbered_toc", "1. [TOC](#x)\npara\n", "para"},
		{"rst_underline_pair", "Title\n=======\npara\n", "para"},
		{"setext_pair", "Title\n---\npara\n", "para"},
		{"atx_heading", "# Title\npara\n", "para"},
		{"ellipsis", "para\n...\n", "para ..."},
		{"colon_colon", "para\n::\n", "para ::"},
		{"thematic_break", "para1\n\n---\n\npara2\n", "para1"},
		{"multi_line_join", "line1\nline2\n", "line1 line2"},
		{"html_comment", "<!-- TOC -->\npara\n", "para"},
		{"empty", "", ""},
		{"whitespace_only", "\n   \n\t\n", ""},
		{"two_char_underline", "para\n==\n", "para =="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, firstRealParagraph([]byte(tt.content)))
		})
	}
}

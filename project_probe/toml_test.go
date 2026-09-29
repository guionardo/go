package projectprobe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The readTOMLSection matrix is pure []byte content — no files, no global
// state — so every test may run in parallel (12-PATTERNS.md toml_test.go row,
// readme_test.go:127-158 precedent).

// TestReadTOMLSection_HappyPath pins the D-01 grammar: a [project] section
// with double-quoted scalar values, a comment line, and a blank line yields
// all three raw values.
func TestReadTOMLSection_HappyPath(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"acme\"\nversion = \"1.2.3\"\ndescription = \"A CLI for acme.\"\n# a comment\n\n")
	want := map[string]string{
		"name":        "acme",
		"version":     "1.2.3",
		"description": "A CLI for acme.",
	}
	assert.Equal(t, want, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_LiteralString pins single-quoted literal scalars
// (D-01): the value is stored raw, quotes stripped.
func TestReadTOMLSection_LiteralString(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nversion = '1.2.3'\n")
	assert.Equal(t, map[string]string{"version": "1.2.3"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_MissingSection pins D-03: no [project] header anywhere
// returns an empty map, never an error.
func TestReadTOMLSection_MissingSection(t *testing.T) {
	t.Parallel()

	content := []byte("[tool.poetry]\nname = \"poet\"\n")
	assert.Equal(t, map[string]string{}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_SectionIsolation pins exact header equality both
// directions (P2): a "project" read sees only [project] keys and a
// "tool.poetry" read only [tool.poetry] keys — never prefix matching.
func TestReadTOMLSection_SectionIsolation(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"acme\"\nversion = \"1.0.0\"\n[tool.poetry]\nname = \"poet\"\nversion = \"2.0.0\"\n")
	assert.Equal(t, map[string]string{"name": "acme", "version": "1.0.0"}, readTOMLSection(content, "project"))
	assert.Equal(t, map[string]string{"name": "poet", "version": "2.0.0"}, readTOMLSection(content, "tool.poetry"))
}

// TestReadTOMLSection_RootKeysExcluded pins that keyvals before any header
// belong to no section and never leak into a section read.
func TestReadTOMLSection_RootKeysExcluded(t *testing.T) {
	t.Parallel()

	content := []byte("name = \"root\"\n[project]\nname = \"acme\"\n")
	assert.Equal(t, map[string]string{"name": "acme"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_DottedKeyDegrade pins the D-07 mechanism: a dotted key
// (version.workspace) is never stored, so the caller reads "" for version.
func TestReadTOMLSection_DottedKeyDegrade(t *testing.T) {
	t.Parallel()

	content := []byte("[package]\nversion.workspace = true\nname = \"crate\"\n")
	assert.Equal(t, map[string]string{"name": "crate"}, readTOMLSection(content, "package"))
}

// TestReadTOMLSection_UnquotedDegrade pins that unquoted values (booleans,
// numbers, barewords) are never stored (D-02).
func TestReadTOMLSection_UnquotedDegrade(t *testing.T) {
	t.Parallel()

	content := []byte("[package]\npublish = false\n")
	assert.Equal(t, map[string]string{}, readTOMLSection(content, "package"))
}

// TestReadTOMLSection_OneLineArray pins P4: one-line arrays are not stored
// AND enter no skip state — a later keyval in the same section still parses.
func TestReadTOMLSection_OneLineArray(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\ndynamic = [\"version\"]\nauthors = [\"a\", \"b\"]\nversion = \"1.2.3\"\n")
	assert.Equal(t, map[string]string{"version": "1.2.3"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_MultiLineStringSkip pins P3/D-disc-4 (SC4): a multi-line
// string body containing a fake keyval and a fake [project] header fabricates
// nothing — the project read stays empty, and the opening key is not stored.
func TestReadTOMLSection_MultiLineStringSkip(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\ndescription = \"\"\"\nname = \"evil\"\n[project]\n\"\"\"\n")
	assert.Equal(t, map[string]string{}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_MultiLineArraySkip pins P4: a multi-line array enters a
// bracket-skip state that clears at the closing bracket — later keyvals parse.
func TestReadTOMLSection_MultiLineArraySkip(t *testing.T) {
	t.Parallel()

	content := []byte("[package]\nauthors = [\n  \"Alice\",\n  \"Bob\",\n]\nversion = \"1.2.3\"\n")
	assert.Equal(t, map[string]string{"version": "1.2.3"}, readTOMLSection(content, "package"))
}

// TestReadTOMLSection_InlineTable pins the inline-table degrade (A6): a
// one-line table is not stored and enters no state; a "{" with the body on
// following lines is skipped until "}" and fabricates nothing.
func TestReadTOMLSection_InlineTable(t *testing.T) {
	t.Parallel()

	oneLine := []byte("[project]\nversion = { file = \"VERSION\" }\nname = \"acme\"\n")
	assert.Equal(t, map[string]string{"name": "acme"}, readTOMLSection(oneLine, "project"))

	multiLine := []byte("[project]\nversion = {\nname = \"evil\"\n}\nname = \"acme\"\n")
	assert.Equal(t, map[string]string{"name": "acme"}, readTOMLSection(multiLine, "project"))
}

// TestReadTOMLSection_EmptyVariants pins the empty-map contract for empty,
// comment-only, and whitespace-only content.
func TestReadTOMLSection_EmptyVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"comment_only", "# just a comment\n# another\n"},
		{"whitespace_only", "\n   \n\t\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, map[string]string{}, readTOMLSection([]byte(tt.content), "project"))
		})
	}
}
package projectprobe

import (
	"fmt"
	"strings"
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

// TestReadTOMLSection_SubtableIsolation pins P2: keys in
// [project.optional-dependencies] never leak into the [project] read — ANY
// [ leading line switches the section.
func TestReadTOMLSection_SubtableIsolation(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"acme\"\n[project.optional-dependencies]\nname = \"evil\"\ntest = [\"pytest\"]\nversion = \"9.9.9\"\n")
	assert.Equal(t, map[string]string{"name": "acme"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_PoetryDepsIsolation pins P2 for the legacy fallback:
// [tool.poetry.dependencies] entries never leak into the [tool.poetry] read.
func TestReadTOMLSection_PoetryDepsIsolation(t *testing.T) {
	t.Parallel()

	content := []byte("[tool.poetry]\nname = \"poet\"\nversion = \"2.0.0\"\n[tool.poetry.dependencies]\nrequests = \"^2.13.0\"\nname = \"evil\"\n")
	assert.Equal(t, map[string]string{"name": "poet", "version": "2.0.0"}, readTOMLSection(content, "tool.poetry"))
}

// TestReadTOMLSection_ArrayOfTables pins P2: a [[array-of-tables]] header
// switches the section and never matches the parent "tool.poetry" read.
func TestReadTOMLSection_ArrayOfTables(t *testing.T) {
	t.Parallel()

	content := []byte("[tool.poetry]\nname = \"poet\"\n[[tool.poetry.source]]\nname = \"evil\"\nversion = \"9.9.9\"\n")
	assert.Equal(t, map[string]string{"name": "poet"}, readTOMLSection(content, "tool.poetry"))
}

// TestReadTOMLSection_HeaderForms pins D-disc-8/A2/A3: a trailing comment and
// inner padding match; quoted header segments and inner-dot whitespace are
// documented non-matches (A3) and degrade to empty reads.
func TestReadTOMLSection_HeaderForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		section string
		want    map[string]string
	}{
		{"trailing_comment", "[project] # comment\nname = \"x\"\n", "project", map[string]string{"name": "x"}},
		{"padded", "[ project ]\nname = \"x\"\n", "project", map[string]string{"name": "x"}},
		{"quoted_segment", "[tool.\"poetry\"]\nname = \"x\"\n", "tool.poetry", map[string]string{}},
		{"inner_dot_space", "[tool . poetry]\nname = \"x\"\n", "tool.poetry", map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, readTOMLSection([]byte(tt.content), tt.section))
		})
	}
}

// TestReadTOMLSection_HashInValue pins P1: "#" inside a quoted value is
// content, never a comment (the spec's "except when inside a string" rule).
func TestReadTOMLSection_HashInValue(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\ndescription = \"hello # world\"\n")
	assert.Equal(t, map[string]string{"description": "hello # world"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_EqualsInValue pins that "=" inside a quoted value is
// content: the first-= split is safe because the key side is validated first.
func TestReadTOMLSection_EqualsInValue(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"a=b\"\n")
	assert.Equal(t, map[string]string{"name": "a=b"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_EscapesRaw pins D-disc-5/A4: basic-string escapes are
// kept verbatim — outer quotes stripped, interior never unescaped.
func TestReadTOMLSection_EscapesRaw(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\ndescription = \"a \\\"b\\\" c\"\n")
	assert.Equal(t, map[string]string{"description": `a \"b\" c`}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_GarbageRemainder pins D-disc-6: a non-whitespace,
// non-comment remainder after the closing quote means the key is not stored —
// never partial data.
func TestReadTOMLSection_GarbageRemainder(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"foo\" garbage\n")
	assert.Equal(t, map[string]string{}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_QuotedKey pins quoted keys (D-01): "name" stores the
// same value as name.
func TestReadTOMLSection_QuotedKey(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\n\"name\" = \"x\"\n")
	assert.Equal(t, map[string]string{"name": "x"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_NonTargetKeysIgnored pins that only the exact target
// names name/version/description are stored — all other keys are ignored.
func TestReadTOMLSection_NonTargetKeysIgnored(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"acme\"\nversion = \"1.0.0\"\ndescription = \"d\"\nauthors = [\"a\"]\nkeywords = [\"k\"]\nhomepage = \"https://example.com\"\nreadme = \"README.md\"\n")
	want := map[string]string{"name": "acme", "version": "1.0.0", "description": "d"}
	assert.Equal(t, want, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_UnspecifiedValue pins that "key =" (no value) is
// invalid per spec and never stored.
func TestReadTOMLSection_UnspecifiedValue(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname =\n")
	assert.Equal(t, map[string]string{}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_DuplicateKeys pins A5: duplicate keys are invalid TOML
// but harmless — the last occurrence wins (mirrors encoding/json).
func TestReadTOMLSection_DuplicateKeys(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\nname = \"first\"\nname = \"second\"\n")
	assert.Equal(t, map[string]string{"name": "second"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_CRLF pins P5: \r\n line endings parse identically to
// \n (TrimSpace per line).
func TestReadTOMLSection_CRLF(t *testing.T) {
	t.Parallel()

	content := []byte("[project]\r\nname = \"acme\"\r\nversion = \"1.2.3\"\r\n")
	want := map[string]string{"name": "acme", "version": "1.2.3"}
	assert.Equal(t, want, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_BOM pins P5: a UTF-8 BOM prefix is tolerated even
// though readManifest strips it upstream (mirrors the Phase 11 BOM rows).
func TestReadTOMLSection_BOM(t *testing.T) {
	t.Parallel()

	content := []byte("\xEF\xBB\xBF[project]\nname = \"acme\"\n")
	assert.Equal(t, map[string]string{"name": "acme"}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_ManyKeys pins that a 200-key section returns all stored
// target keys — no arbitrary limit.
func TestReadTOMLSection_ManyKeys(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	b.WriteString("[project]\nname = \"acme\"\nversion = \"1.0.0\"\ndescription = \"d\"\n")
	for i := 0; i < 197; i++ {
		fmt.Fprintf(&b, "key%d = \"v%d\"\n", i, i)
	}
	want := map[string]string{"name": "acme", "version": "1.0.0", "description": "d"}
	assert.Equal(t, want, readTOMLSection([]byte(b.String()), "project"))
}

// TestReadTOMLSection_CrossSectionSkip pins D-disc-4 globally (the strongest
// SC4 guard): a multi-line string opened in [build-system] suppresses header
// AND keyval parsing everywhere — a fake [project] header and name = "evil"
// inside its body are never parsed.
func TestReadTOMLSection_CrossSectionSkip(t *testing.T) {
	t.Parallel()

	content := []byte("[build-system]\nrequires = [\"setuptools\"]\ndescription = \"\"\"\n[project]\nname = \"evil\"\n\"\"\"\n")
	assert.Equal(t, map[string]string{}, readTOMLSection(content, "project"))
}

// TestReadTOMLSection_Adversarial pins T-12-01: pathological inputs
// (bracket floods, unterminated quotes, a 1 MB jumble) complete without
// panicking — the rows assert only that the call returns a map.
func TestReadTOMLSection_Adversarial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{"bracket_flood", "[project]\n" + strings.Repeat("[", 10000) + "\n"},
		{"unterminated_quote", "[project]\nname = \"abc\n"},
		{"one_mb_jumble", "[project]\n" + strings.Repeat("\"[]=#", 250000) + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.NotNil(t, readTOMLSection([]byte(tt.content), "project"))
		})
	}
}
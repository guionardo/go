package projectprobe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// t.Parallel is safe in this file: every test writes t.TempDir fixtures only
// and never mutates the package-level detectors slice (AGENTS.md global-state
// rule — the registry tests are the ones that must stay serial).

// TestProbe_JavaPomEndToEnd exercises the full chain through Probe: registry →
// detectJavaKotlin → readManifest presence gate → xml.Unmarshal → merge. A
// folder with the canonical namespaced
// <project xmlns="http://maven.apache.org/POM/4.0.0"> pom yields LanguageJava
// with Name/Version/Description verbatim (SC2, D-05, probe row F).
func TestProbe_JavaPomEndToEnd(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pom.xml"),
		[]byte(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.acme</groupId>
  <artifactId>acme-core</artifactId>
  <name>Acme Core</name>
  <version>1.4.2</version>
  <description>Core library for acme.</description>
</project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJava, data.Language)
	assert.Equal(t, "Acme Core", data.Name)
	assert.Equal(t, "1.4.2", data.Version)
	assert.Equal(t, "Core library for acme.", data.Description)
}

// TestProbe_JavaParentVersion pins the SC2 child-module shape: the
// <parent><version> block lives in the CHILD's OWN pom.xml (Maven requires
// the parent block's version in every child POM); the child's own <version>
// is omitted and inherits it — single level, never a sibling/relativePath
// read (D-05, Pitfall 5, probe row G).
func TestProbe_JavaParentVersion(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pom.xml"),
		[]byte(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <parent>
    <groupId>com.acme</groupId>
    <artifactId>acme-parent</artifactId>
    <version>2.0.0</version>
  </parent>
  <artifactId>acme-child</artifactId>
  <name>Acme Child</name>
</project>
`),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJava, data.Language)
	assert.Equal(t, "Acme Child", data.Name)
	assert.Equal(t, "2.0.0", data.Version)
}

// TestProbe_JavaGradleFallback pins the SC3 fallback: a folder with ONLY
// settings.gradle (no pom.xml) yields LanguageJava with the rootProject.name
// literal (D-06).
func TestProbe_JavaGradleFallback(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "settings.gradle"),
		[]byte("rootProject.name = 'acme-tool'\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJava, data.Language)
	assert.Equal(t, "acme-tool", data.Name)
}

// TestProbe_JavaPomWinsOverGradle pins the D-06 asymmetry: when BOTH pom.xml
// and settings.gradle exist, the pom fields win — the gradle fallback fires
// ONLY when no pom.xml is present.
func TestProbe_JavaPomWinsOverGradle(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pom.xml"),
		[]byte(`<project xmlns="http://maven.apache.org/POM/4.0.0">
  <artifactId>acme-pom</artifactId>
  <version>3.1.0</version>
</project>
`),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "settings.gradle"),
		[]byte("rootProject.name = 'gradle-name'\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJava, data.Language)
	assert.Equal(t, "acme-pom", data.Name)
	assert.Equal(t, "3.1.0", data.Version)
}

// TestProbe_JavaGarbagePomSkipsGradle pins the D-09 presence-match asymmetry:
// a present-but-garbage pom.xml still claims Java (folder-base Name via the
// fallback chain), and the settings.gradle fallback does NOT fire — D-06's
// fallback only fires when pom.xml is ABSENT.
func TestProbe_JavaGarbagePomSkipsGradle(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pom.xml"),
		[]byte("<project><name"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "settings.gradle"),
		[]byte("rootProject.name = 'gradle-name'\n"),
		0o600,
	))

	data, err := Probe(folder)
	require.NoError(t, err)
	assert.Equal(t, LanguageJava, data.Language)
	assert.Equal(t, filepath.Base(folder), data.Name)
}

// TestProbe_JavaRootScope pins D-07/SC4: a pom.xml in a subdirectory never
// triggers the parent folder — root-scoped only.
func TestProbe_JavaRootScope(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	sub := filepath.Join(parent, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(sub, "pom.xml"),
		[]byte(`<project><name>Sub</name></project>`),
		0o600,
	))

	data, err := Probe(parent)
	require.NoError(t, err)
	assert.Equal(t, LanguageUnknown, data.Language)
}

// TestDetectJavaKotlin_MissingManifest pins D-09: a folder with neither
// pom.xml nor settings.gradle is a non-match — (ProjectData{}, false).
func TestDetectJavaKotlin_MissingManifest(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()

	data, ok := detectJavaKotlin(folder)
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, data)
}

// TestParseRootProjectName pins the settings.gradle line parser contract
// (D-disc-3/4): exact left-of-= key match, first quoted literal, remainder
// guard — a commented assignment or an 'a' + 'b' concatenation degrades to
// "", and no input (an empty file included) can panic the parser (D-09).
// Pure content — []byte in, string out (the parseGoMod precedent).
func TestParseRootProjectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"single_quotes", "rootProject.name = 'acme-tool'\n", "acme-tool"},
		{"comment_line", "// rootProject.name = 'evil'\n", ""},
		{"concat_expression", "rootProject.name = 'a' + 'b'\n", ""},
		{"empty_content", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, parseRootProjectName([]byte(tt.content)))
		})
	}
}
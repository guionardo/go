package projectprobe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: no t.Parallel() anywhere in this file. The registry tests mutate the
// package-level detectors slice; parallel tests on global state race
// (AGENTS.md mid/collectFuncs precedent, RESEARCH Pitfall 4).

// TestRunDetectors_EmptyRegistry pins the Phase-10 default state: the
// registry is empty, so no folder ever matches.
func TestRunDetectors_EmptyRegistry(t *testing.T) { //nolint:paralleltest // global-state mutation: detectors slice
	original := detectors
	defer func() { detectors = original }()

	pd, ok := runDetectors("")
	assert.False(t, ok)
	assert.Equal(t, ProjectData{}, pd)
}

// TestRunDetectors_OrderAndFirstMatch verifies the cascade contract
// (DETC-01): the first matching detector wins, and a non-matching detector
// falls through to the next one.
func TestRunDetectors_OrderAndFirstMatch(t *testing.T) { //nolint:paralleltest // global-state mutation: detectors slice
	original := detectors
	defer func() { detectors = original }()

	detectors = []detectorFunc{
		func(_ string) (ProjectData, bool) { return ProjectData{}, false }, // non-match falls through
		func(_ string) (ProjectData, bool) { return ProjectData{Language: LanguagePython}, true },
		func(_ string) (ProjectData, bool) { return ProjectData{Language: LanguageRust}, true },
	}

	pd, ok := runDetectors("x")
	assert.True(t, ok)
	assert.Equal(t, LanguagePython, pd.Language)
}

// TestRunDetectors_PanicRecovery verifies D-03 / Pattern 2: a panicking
// detector is swallowed by callDetector and counts as a non-match — the
// panic never escapes and the next detector still runs.
func TestRunDetectors_PanicRecovery(t *testing.T) { //nolint:paralleltest // global-state mutation: detectors slice
	original := detectors
	defer func() { detectors = original }()

	detectors = []detectorFunc{
		func(_ string) (ProjectData, bool) { panic("boom") },
		func(_ string) (ProjectData, bool) { return ProjectData{Language: LanguageGo}, true },
	}

	pd, ok := runDetectors("x")
	assert.True(t, ok)
	assert.Equal(t, LanguageGo, pd.Language)
}

// TestProbe_MergeRule verifies the A7 merge rule: when a detector matches,
// its Language/Name/Version/Description override the Unknown defaults while
// Folder stays owned by Probe (filepath.Clean(as-given)).
func TestProbe_MergeRule(t *testing.T) { //nolint:paralleltest // global-state mutation: detectors slice
	original := detectors
	defer func() { detectors = original }()

	detectors = []detectorFunc{
		func(_ string) (ProjectData, bool) {
			return ProjectData{Language: LanguageGo, Name: "x", Version: "1", Description: "d"}, true
		},
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("hi"), 0o600))

	data, err := Probe(dir)
	require.NoError(t, err)
	assert.Equal(t, LanguageGo, data.Language)
	assert.Equal(t, "x", data.Name)
	assert.Equal(t, "1", data.Version)
	assert.Equal(t, "d", data.Description)
	assert.Equal(t, filepath.Clean(dir), data.Folder)
}

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

// TestRunDetectors_EmptyRegistry pins the empty-cascade behavior: with no
// detector matching (the injected slice below never matches the "" folder),
// runDetectors yields no match. The production registry now holds real
// detectors, but this test replaces the slice wholesale, so it still passes
// (Pitfall 4 — the comment, not the test, was stale).
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

// TestDetectorPositions pins the D-11 order contract: the registry literal
// has exactly 7 positions; Go, JS/TS, and PHP are live at indices 0, 3, and
// 6; the future-phase slots (Python, C#/.NET, Rust, Java) are nil until
// their plans land. A future phase edits slots in place, never reorders —
// this test guards the cascade precedence (T-11-07).
func TestDetectorPositions(t *testing.T) { //nolint:paralleltest // reads global detectors
	assert.Len(t, detectors, 7)
	assert.NotNil(t, detectors[0]) // Go
	assert.NotNil(t, detectors[1]) // Python — Phase 12
	assert.NotNil(t, detectors[2]) // C#/.NET — Phase 13
	assert.NotNil(t, detectors[3]) // JS/TS
	assert.NotNil(t, detectors[4]) // Rust — Phase 12
	assert.Nil(t, detectors[5])    // Java/Kotlin — Phase 13
	assert.NotNil(t, detectors[6]) // PHP
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

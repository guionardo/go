package projectprobe

// detectorFunc reports a project match for folder. It must never return an
// error: a non-match is (ProjectData{}, false). A recovered panic is treated
// as a non-match so Probe can never fail on content (D-03).
type detectorFunc func(folder string) (ProjectData, bool)

// detectors is the ordered cascade. Order is the contract (DETC-01): first
// match wins. Phases 11-13 append entries HERE, in cascade position.
// Go → Python → C#/.NET → JS/TS → Rust → Java/Kotlin → PHP.
var detectors = []detectorFunc{}

// runDetectors dispatches folder to the ordered cascade and returns the
// first match. An empty registry yields no match.
func runDetectors(folder string) (pd ProjectData, ok bool) {
	for _, fn := range detectors {
		if pd, ok := callDetector(fn, folder); ok {
			return pd, true
		}
	}

	return ProjectData{}, false
}

// callDetector runs one detector with panic recovery, mirroring
// cache.SingleflightGetOrSet.callSetter: a recovered panic never escapes and
// the detector counts as a non-match (D-03, Pattern 2).
func callDetector(fn detectorFunc, folder string) (pd ProjectData, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			// treat as non-match (never re-panic)
			pd, ok = ProjectData{}, false
		}
	}()

	return fn(folder)
}

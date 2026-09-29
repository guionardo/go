package projectprobe

// detectorFunc reports a project match for folder. It must never return an
// error: a non-match is (ProjectData{}, false). A recovered panic is treated
// as a non-match so Probe can never fail on content (D-03).
type detectorFunc func(folder string) (ProjectData, bool)

// detectors is the ordered cascade. Order is the contract (DETC-01, D-11):
// first match wins; the 7 positions are permanent — Go → Python → C#/.NET →
// JS/TS → Rust → Java/Kotlin → PHP. Live entries sit at their research-locked
// slots; empty slots stay nil until their phase fills them (Phase 11 fills
// Go at 0, JS/TS at 3, PHP at 6; Python/Rust in Phase 12, C#/.NET/Java in
// Phase 13). nil entries are skipped by runDetectors.
var detectors = []detectorFunc{
	detectGo,  // index 0 — Go (Phase 11)
	nil,       // index 1 — Python (Phase 12)
	nil,       // index 2 — C#/.NET (Phase 13)
	detectJS,  // index 3 — JavaScript/TypeScript (Phase 11)
	nil,       // index 4 — Rust (Phase 12)
	nil,       // index 5 — Java/Kotlin (Phase 13)
	detectPHP, // index 6 — PHP (Phase 11)
}

// runDetectors dispatches folder to the ordered cascade and returns the
// first match. Empty slots (nil entries) are skipped before callDetector —
// a nil entry would panic on the call, be swallowed as a non-match, and spam
// recovery. A registry with no live detectors yields no match.
func runDetectors(folder string) (pd ProjectData, ok bool) {
	for _, fn := range detectors {
		if fn == nil { // empty slot (Phase 12/13) — skip
			continue
		}
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

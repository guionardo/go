---
phase: 12
phase_name: "TOML Subset + Python/Rust Detectors"
project: "github.com/guionardo/go"
generated: "2026-09-30"
counts:
  decisions: 7
  lessons: 5
  patterns: 6
  surprises: 4
missing_artifacts:
  - "12-UAT.md"
---

# Phase 12 Learnings: TOML Subset + Python/Rust Detectors

## Decisions

### Global skip states entered from keyval lines in ANY section
`parseKeyValue` runs for every keyval line, not only the target section — a multi-line string body in `[build-system]` must never fabricate `name = "evil"` into `[project]`.

**Rationale:** D-disc-4 (flagged assumption): the RESEARCH skeleton's section-gated order applies to storage only, not skip detection. Pinned by `TestReadTOMLSection_CrossSectionSkip`.
**Source:** 12-01-SUMMARY.md

### Whole-section precedence for Python metadata
`[project]` non-empty map wins; `[tool.poetry]` is read ONLY when `len(fields)==0` — no per-field mixing across sections for migrated Poetry 2.x files.

**Rationale:** D-disc-3 — the len==0 guard covers both "section absent" and "section present but everything unsupported" (`name = 123` unquoted → empty map → poetry wins). Pinned by ProjectWins + EmptyProjectFallsToPoetry rows.
**Source:** 12-02-SUMMARY.md

### D-07 executed structurally with zero code branches
`version.workspace = true` degrades to Version `""` entirely via the reader's dotted-key classification (`isBareKey` rejects `.`) — no workspace-root resolution, no special-casing, no parent traversal.

**Rationale:** SC3 never-fabricated; the `workspace.package` grep is clean — the reader IS the mechanism, so no detector can regress the contract.
**Source:** 12-03-SUMMARY.md

### Quote-aware skip-state scanning (CR-01 gap closure)
Skip-state clearing uses run-length semantics for multi-line strings (3 quotes = close, >3 = close+reopen) and in-string immunity for brackets (honoring `\` escapes, tracking `[`/`{` depth); a cross-line `pendingMLS` state keeps later `]`/`}` inside a multi-line string opened in a bracket body.

**Rationale:** Whole-line `strings.Contains` detection fabricated `name="evil"` from inside construct bodies in 8/9 verifier probe rows. The fix direction came verbatim from 12-REVIEW.md lines 67-90.
**Source:** 12-04-SUMMARY.md

### No dynamic-field special case
`dynamic = ["version"]` has NO code branch in the detector — the reader degrades the array to an absent key, Version reads `""`.

**Rationale:** DATA-03 never-fabricated; a special case would be an anti-pattern (zero detector code branches).
**Source:** 12-02-SUMMARY.md

### Presence-match asymmetry accepted at cascade level
A malformed pyproject.toml claims Python at index 1 over a valid Cargo.toml at index 4 (and over broken-JS at 3) — the manifest file IS the ecosystem marker; fields degrade, never fabricated.

**Rationale:** T-12-08 accept (D-09 presence vs D-04 parse-success asymmetry); pinned by the `pyproject_toml_and_cargo_toml` cascade row.
**Source:** 12-03-SUMMARY.md

### Under-skip residual accepted as the safe direction
A line that nets the string closed skips bracket scanning, so a construct closer sharing that line is scanned on the next line — extra skipping degrades to empty, never fabricates.

**Rationale:** Documented in the plan's flagged-assumptions table; under-skip is the SAFE direction (empty fields), over-read would fabricate.
**Source:** 12-04-SUMMARY.md

---

## Lessons

### Whole-line skip-state detection fabricates data — the verifier found it
The CR-01 blocker fabricated `name="evil"` from inside multi-line construct bodies in 8/9 verifier probe rows — the whole-line `strings.Contains(line, skipDelim)` approach was fundamentally wrong, not edge-buggy.

**Context:** Code review (12-REVIEW.md CR-01) caught what the 341-line fixture matrix missed; the gap-closure cycle (plan 12-04 via /gsd-plan-phase --gaps) restored SC4. The 12-subtest verifier probe now proves 9/9 shapes produce no fabrication.
**Source:** 12-04-SUMMARY.md

### Reader size overran the budget — correctness first
`readTOMLSection` landed at 172 lines (128 non-comment code) vs the ~80-line CONTEXT target and ~100-110 flagged estimate — the skip states and thorough doc comments account for the size.

**Context:** Flagged assumption OQ-3/A7: the verification explicitly required only documenting the final count, not meeting the budget. Documented, not deviated.
**Source:** 12-01-SUMMARY.md

### `go test -cover` prints no percentage line on this Go version
The coverage figure had to be extracted from `-coverprofile` via `go tool cover -func` (toml.go 97.3% file, project_probe 96.3% package).

**Context:** Environment wrinkle, plans 12-01/12-03 — the rtk wrapper hides the coverage line; `-coverprofile` + `go tool cover -func` is the reliable path.
**Source:** 12-01-SUMMARY.md

### Interim position flips must preserve the next plan's RED
`TestDetectorPositions` changed exactly one assertion per plan (detectors[1] NotNil in 12-02; detectors[4] Nil→NotNil in 12-03) — detectors[4] stayed Nil through 12-02 so plan 12-03's RED step remained meaningful.

**Context:** Pitfall 9 discipline; the flip is the ONLY registry_test.go change allowed per plan prohibition — no comment edits, no parallel markers.
**Source:** 12-02-SUMMARY.md

### Prior-wave edits can leave files non-gofmt at HEAD
The 7-slot registry literal was committed non-gofmt at HEAD (mixed trailing-comment alignment from prior edits); gofmt -w realigned 7 whitespace-only lines as part of the slot-4 edit.

**Context:** Plan 12-03 — formatting a file being modified is the clean end state, not scope creep (verified `gofmt -l` on the HEAD blob).
**Source:** 12-03-SUMMARY.md

---

## Patterns

### Presence-match detector with whole-section fallback
detect_go.go exact analog: readManifest → readTOMLSection → ProjectData{Language}; len==0 whole-section fallback; DATA-02 folder-base then DATA-04 README chains.

**When to use:** Detector families with multiple manifest generations (PEP 621 vs Poetry legacy) — whole-section precedence, never per-field mixing.
**Source:** 12-02-SUMMARY.md

### Probe-level fixture rows with t.TempDir + os.WriteFile
Each e2e row writes its own temp-dir fixture through the production path (Probe → registry → detector → readManifest), `t.Parallel`-safe when no detectors-slice mutation occurs.

**When to use:** End-to-end detector tests — real files through the real pipeline, no mocks; parallel-safe only when the registry is untouched.
**Source:** 12-02-SUMMARY.md

### Regression isolation in RED
New failing rows run with `-run` union of old+new skip tests to prove the old matrix stays green while the new rows fail.

**When to use:** Gap-closure REDs — isolates the fix's blast radius before it lands (12-04 precedent).
**Source:** 12-04-SUMMARY.md

### len==0 map guard for section-absent-or-unsupported
The empty-map guard is the single branch deciding fallback — covers absent sections and present-but-degenerate ones identically.

**When to use:** Multi-generation manifest readers where "no usable data" must be one decision.
**Source:** 12-02-SUMMARY.md

### BOM tolerance reuses the shared utf8BOM constant
`bytes.TrimPrefix(content, utf8BOM)` — single source of truth per Phase 10 precedent; the reader stays tolerant even though readManifest strips upstream (P5 BOM row).

**When to use:** Any pure-content parser that may receive pre-stripped or raw bytes — keep the strip idempotent and shared.
**Source:** 12-01-SUMMARY.md

### Pure-content fixture matrix with adversarial no-panic rows
341-line matrix (29 tests, 39 cases): 10k bracket flood, unterminated quote, 1 MB jumble — pathological input never panics (T-12-01).

**When to use:** Parser validation — a named row per classification plus adversarial rows discharges "never panics" contracts.
**Source:** 12-01-SUMMARY.md

---

## Surprises

### Code review caught a real fabrication bug the test matrix missed
The 341-line fixture matrix passed while the reader fabricated `name="evil"` from 8 of 9 verifier probe shapes — the matrix tested declared behaviors, not adversarial construct-body shapes.

**Impact:** Triggered the milestone's only gap-closure cycle (plan 12-04, 37 min — the longest plan of the phase); the quote-aware rewrite and 13-subtest pinning now prove no-fabrication. The loop worked end-to-end: verifier → gap plan → fix → re-verify.
**Source:** 12-04-SUMMARY.md

### 4-quote runs keep the multi-line state
`""""` (4 quotes) = close+reopen — the run > 3 semantics mean a 4-quote opening line ENTERS the multi-line-string skip state (previously a missed-entry fabrication path).

**Impact:** `enterSkip` uses the same quote-aware scans as clearing; both missed-entry directions (4-quote opens, bracket-in-string opens) are now closed and pinned.
**Source:** 12-04-SUMMARY.md

### The multi-line literal-string branch was never exercised
The `'''` (single-quote triple) path existed but had no test row — IN-03 from review; the SixQuoteBody test now covers both `"""` and `'''` branches.

**Impact:** The unexercised branch was the second fabrication direction; pinning it closed the CR-01 family completely.
**Source:** 12-04-SUMMARY.md

### A malformed pyproject outranks a valid Cargo.toml
The presence-match asymmetry is a real cascade consequence: garbage pyproject.toml + valid Cargo.toml → LanguagePython at index 1, by design (T-12-08 accept).

**Impact:** Mixed-repo precedence is now deterministic and documented; pinned by the cascade row — surprising but intentional.
**Source:** 12-03-SUMMARY.md
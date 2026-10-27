---
status: resolved
trigger: "CI run red on Windows only. Test 1 in UAT (.planning/phases/16-batch-surface-multi-process-hardening/16-UAT.md): 'failed only on windows test'. GitHub Actions run 37863201983 (PR #48, branch gsd/v1.8-sqlite-cache-backend): job 'Run tests (windows-latest)' FAILED; ubuntu-latest and macos-latest succeeded. Only failing test: --- FAIL: TestDownloadUpdate_DigestMismatch (0.01s) — package github.com/guionardo/go/release; release/update_test.go:489 require.NoFileExists — Error: file 'C:\\Users\\RUNNER~1\\AppData\\Local\\Temp\\TestDownloadUpdate_DigestMismatch1350819659\\001\\myapp_windows_amd64.tar.gz' exists. Goal: find_root_cause_only."
created: 2026-10-09T00:15:00Z
updated: 2026-10-27T00:00:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: "CONFIRMED — In release/update.go DownloadUpdate, the asset.Download error path calls os.Remove(filePath) while the os.Create file handle is still open (the deferred f.Close() only runs after DownloadUpdate returns), and discards the os.Remove error. On Windows, Go opens files without FILE_SHARE_DELETE, so DeleteFile fails with ERROR_SHARING_VIOLATION and the empty file remains; require.NoFileExists then fails. On POSIX, unlink of an open file succeeds → same code passes."
test: "COMPLETE. Direct CI job JSON + failed log, local POSIX differential run, Go 1.27/1.26 stdlib source inspection (syscall.Open sharemode, os.Remove→DeleteFile), testify v1.11.1 NoFileExists→os.Lstat, git provenance."
expecting: "All confirmed — see Evidence."
next_action: "NONE — diagnosis complete (goal: find_root_cause_only). Hand to plan-phase --gaps for fix design."

reasoning_checkpoint:
  hypothesis: "DownloadUpdate's digest-mismatch cleanup calls os.Remove while the os.Create handle is still open; on Windows (no FILE_SHARE_DELETE on Go's default open sharemode) DeleteFile fails, the ignored error leaves a 0-byte file behind, and require.NoFileExists fails. POSIX unlink-of-open-file semantics hide the defect."
  confirming_evidence:
    - "release/update.go:138-144 — defer f.Close() registered at line 138; os.Remove(filePath) at line 141 executes BEFORE the deferred close (deferred functions run on return); remove's return value is discarded."
    - "release/release.go:140-143 — Asset.Download checks the digest BEFORE writing: on mismatch it returns without calling w.Write, so the leftover file is the 0-byte file created by os.Create (not a partial payload)."
    - "Go 1.27 stdlib: syscall/syscall_windows.go:395 sharemode = FILE_SHARE_READ|FILE_SHARE_WRITE (no FILE_SHARE_DELETE); os/file_windows.go:158 (openFileNolog→syscall.Open); os/file_windows.go:222-230 (Remove→syscall.DeleteFile). CI Go = go.mod 1.26.4; behavior long-standing."
    - "CI run 37863201983: exactly one '--- FAIL' in --log-failed (the digest test); windows 'Run tests' failed and E2E+race steps skipped; ubuntu/macos jobs fully green (gh JSON)."
    - "Local macOS: go test ./release/ -run TestDownloadUpdate_DigestMismatch → PASS (differential: POSIX unlink succeeds on open file)."
    - "Provenance: d1a3786 (which adds the NoFileExists assertion, blamed to release/update_test.go:489) is NOT on origin/main; origin/main fefec2b update_test.go has no DigestMismatch test. PR #48 is the first CI exposure. Defective update.go code exists on origin/main since 3a13dae (latent)."
  falsification_test: "If the file were closed before remove, or if Go opened files with FILE_SHARE_DELETE, os.Remove would succeed on Windows and the file would not exist — the test would pass. Also falsified if the failing path wrote no file at all (os.Create failed) — but test asserts 'download failed' error was returned, which only happens after os.Create succeeded."
  fix_rationale: "The fix must guarantee close-before-remove ordering on the error path (and ideally surface the cleanup error). Adding FILE_SHARE_DELETE via raw syscall is not portable and not needed; atomic temp+rename is the larger alternative. Both satisfy the minimal missing property."
  blind_spots: "Could not execute on a Windows host from this machine; Windows behavior is established from Go stdlib source + the direct CI failure signature. The pre-fix failure on Windows is deterministic unless external AV interference exists; AV is not needed to explain the observed failure."
  candidate_causes:
    - "code: os.Remove before f.Close on the download-error path (root cause); ignored os.Remove error (masks it)"
    - "environment: Windows DeleteFile requires FILE_SHARE_DELETE for open files; POSIX unlink does not (platform-conditional visibility)"
    - "data: none — the deliberately mismatched digest is the intended test trigger, not a defect"
    - "config/CI: none — Windows matrix leg exists since 2025-07-09 and behaved correctly"
  and_gate: "Single code defect with platform-conditional visibility. The failure requires the code ordering AND Windows delete semantics — but those are one mechanism, not two independent defects. The ignored remove error is a contributing weakness (silent failure), not a co-necessary cause of file persistence. No second independent cause found."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: "Both TestTwoProcessContention and TestTwoProcessCrashRecovery pass on all three OSes; the race step passes on ubuntu and macos and is skipped on windows."
actual: "GitHub Actions run 37863201983 (PR #48, branch gsd/v1.8-sqlite-cache-backend): job 'Run tests (windows-latest)' FAILED; ubuntu-latest and macos-latest succeeded; the 'SQLite two-process E2E' and 'SQLite race detector' steps on windows were SKIPPED (cascade from the failed step). Only failing test: --- FAIL: TestDownloadUpdate_DigestMismatch (0.01s)."
errors: |
  --- FAIL: TestDownloadUpdate_DigestMismatch (0.01s) — package github.com/guionardo/go/release
  release/update_test.go:489: Error: file "C:\\Users\\RUNNER~1\\AppData\\Local\\Temp\\TestDownloadUpdate_DigestMismatch1350819659\\001\\myapp_windows_amd64.tar.gz" exists
  require.NoFileExists failure
reproduction: "Test 1 in UAT (.planning/phases/16-batch-surface-multi-process-hardening/16-UAT.md) — remote GitHub Actions matrix run on PR #48. Exact local analog: go test ./release/ -run TestDownloadUpdate_DigestMismatch (fails on Windows only; passes on POSIX)."
started: "Discovered during UAT — first push of the milestone branch. Latent since 3a13dae (2026-07-21); first CI exposure on PR #48 (2026-10-09) because the asserting test (d1a3786) had never been pushed to a remote branch."

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "The cache/sqlite two-process E2E tests failed on Windows."
  evidence: "CI log shows windows 'SQLite two-process E2E' and 'SQLite race detector' steps were SKIPPED (cascade), not failed; the only '--- FAIL' in the whole failed log is release TestDownloadUpdate_DigestMismatch; cache tests passed on all three OSes."
  timestamp: 2026-10-09T00:20:00Z

- hypothesis: "t.Parallel / t.TempDir cross-test interference or a stale file from another test."
  evidence: "t.TempDir() returns a unique per-test directory (CI path ...\\TestDownloadUpdate_DigestMismatch1350819659\\001\\...); other DownloadUpdate tests in the same package passed; the assertion path is the digest-mismatch cleanup."
  timestamp: 2026-10-09T00:22:00Z

- hypothesis: "Transient external lock (antivirus/indexer) caused the Remove failure — CI flake."
  evidence: "The os.Create handle is provably still open at os.Remove time (defer runs after return) and Go's Windows open sharemode lacks FILE_SHARE_DELETE, making the failure deterministic; the same run's other Windows tests passed and ubuntu/macos were green. AV not required to explain it."
  timestamp: 2026-10-09T00:24:00Z

- hypothesis: "The file was recreated after removal, or a different file is asserted."
  evidence: "No code path recreates the file after os.Remove; DownloadUpdate returns immediately after the failed cleanup; the asserted path is the unique t.TempDir path."
  timestamp: 2026-10-09T00:25:00Z

- hypothesis: "Windows CI workflow/config is broken (step conditions, matrix)."
  evidence: "go.yml matrix [ubuntu, macos, windows] dates to 2025-07-09; windows 'Run tests' step executed and failed on exactly one test; all its other steps/tests behaved normally; ubuntu/macos jobs fully green."
  timestamp: 2026-10-09T00:26:00Z

- hypothesis: "Digest string case/format disagreement caused a spurious mismatch failure."
  evidence: "The mismatch is intentional (server sends 'content', expected digest is of 'differentContent'); the test's require.Error + 'download failed' assertions passed; only the post-cleanup NoFileExists assertion failed."
  timestamp: 2026-10-09T00:27:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-10-09T00:16:00Z
  checked: "release/update.go:123-147 DownloadUpdate (branch HEAD, on-disk)"
  found: "Line 134 os.Create(filePath); line 138 defer func(){ _ = f.Close() }(); lines 140-144 error path: asset.Download(f) error → os.Remove(filePath) [return value discarded] → return error. The deferred Close runs only when DownloadUpdate returns — i.e., AFTER os.Remove."
  implication: "Handle is open at Remove time on every platform; remove error is silently ignored. This is the exact ordering that Windows rejects."

- timestamp: 2026-10-09T00:17:00Z
  checked: "release/release.go:128-147 Asset.Download"
  found: "HTTP GET → io.ReadAll → digest.FromBytes(content) compared with asset.Digest BEFORE w.Write; on mismatch returns error at line 142 without writing any bytes."
  implication: "On the digest-mismatch path the on-disk artifact is the 0-byte file created by os.Create — present but empty; the test correctly expects it removed."

- timestamp: 2026-10-09T00:18:00Z
  checked: "release/update_test.go:458-491 TestDownloadUpdate_DigestMismatch (and blame)"
  found: "Test serves content but sets Digest of differentContent; expects 'download failed' and require.NoFileExists(filepath.Join(dir, rel.Assets[0].Name)) at line 489. Blame: added by d1a3786 (2026-09-30, v1.7 squash). t.TempDir() isolates the directory."
  implication: "The test is a correct oracle for the cleanup contract; it is the first test to exercise the open-handle cleanup on Windows."

- timestamp: 2026-10-09T00:19:00Z
  checked: "Go stdlib Windows semantics (local GOROOT go1.27.0; CI toolchain from go.mod = go 1.26.4)"
  found: "syscall/syscall_windows.go:395 `sharemode := uint32(FILE_SHARE_READ | FILE_SHARE_WRITE)` — FILE_SHARE_DELETE is NOT requested. os/file_windows.go:153-158 openFileNolog → syscall.Open. os/file_windows.go:222-230 os.Remove → syscall.DeleteFile (then RemoveDirectory fallback). No FILE_SHARE_DELETE usage anywhere in os/syscall open paths; only internal/syscall/windows/at_windows.go (lines 151, 223, 252) explicitly adds it where delete/rename semantics need it."
  implication: "While the os.Create handle is open, DeleteFile gets ERROR_SHARING_VIOLATION (32). Deterministic Windows-only behavior. Explains the platform split exactly."

- timestamp: 2026-10-09T00:21:00Z
  checked: "CI run 37863201983 via gh (jobs JSON + --log-failed)"
  found: "windows-latest: 'Run tests' FAILED, 'SQLite two-process E2E' SKIPPED, 'SQLite race detector' SKIPPED. ubuntu-latest and macos-latest: every step success. Failed log contains exactly one '--- FAIL' line — TestDownloadUpdate_DigestMismatch, update_test.go:489, 'file ... exists'. Run: PR #48, base main fefec2b, head 61d1787 (67 commits), created 2026-10-09T00:08:15Z."
  implication: "Direct reproduction evidence: only this test fails; cache/sqlite work is clean; the failure message is exactly the leftover-file signature."

- timestamp: 2026-10-09T00:23:00Z
  checked: "testify v1.11.1 (go.mod) NoFileExists implementation in module cache"
  found: "NoFileExists calls os.Lstat(path); if err != nil → passes; if the stat succeeds → fails with 'file ... exists'."
  implication: "Because DeleteFile failed on Windows, Lstat succeeds and the assertion fails. Confirms the propagation chain."

- timestamp: 2026-10-09T00:28:00Z
  checked: "Local differential run on macOS (darwin)"
  found: "go test ./release/ -run TestDownloadUpdate_DigestMismatch -v -count=1 → 1 passed."
  implication: "POSIX unlink of an open file removes the directory entry immediately → test passes. Same code, platform-conditional outcome — matches the hypothesis."

- timestamp: 2026-10-09T00:31:00Z
  checked: "Provenance: ancestry of d1a3786 vs origin/main (fefec2b), PR #48 metadata, remote branches"
  found: "d1a3786 (adds the NoFileExists test) is NOT an ancestor of origin/main; origin/main release/update_test.go has no DigestMismatch test (grep count 0); origin/main release/update.go has the IDENTICAL defective error path (from 3a13dae, 2026-07-21). PR #48 base fefec2b → head 61d1787 contains 67 commits. Only remote gsd branches: origin/gsd/v1.6-cache-dedup, origin/gsd/v1.8-sqlite-cache-backend. Main-branch Windows CI runs (2026-09-30, 2026-10-08) passed because the test did not exist there."
  implication: "Explains 'started on first push of the milestone branch': PR #48 is the first-ever Windows CI exposure of this test. Defect was latent, not newly introduced by v1.8."

- timestamp: 2026-10-09T00:33:00Z
  checked: "Sibling cleanup patterns across repo (grep os.Remove)"
  found: "release/self_update.go:187-190 correctly closes the lock file BEFORE os.Remove(lock.path). release/swapper/swap_windows.go:16,27 ignore os.Remove(backup) errors but operate on files not held open by this process at that moment. release/update.go:141 is the only remove-while-open instance."
  implication: "The codebase already contains the correct close-before-remove idiom (self_update.go); the fix for update.go can mirror it. Siblings are out of scope for this failure."

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "release/update.go DownloadUpdate's asset.Download error path calls os.Remove(filePath) while the os.Create handle is still open (defer f.Close() runs only after DownloadUpdate returns) and discards the os.Remove error. On Windows, Go opens files without FILE_SHARE_DELETE (syscall.Open sharemode = FILE_SHARE_READ|FILE_SHARE_WRITE), so syscall.DeleteFile fails with ERROR_SHARING_VIOLATION; the ignored error leaves the 0-byte file on disk and testify's require.NoFileExists (os.Lstat) fails. On POSIX, unlink of an open file succeeds, so the same code passes — hence Windows-only red CI. Contributing weakness: the cleanup error is silently discarded, masking the failure mechanism."
fix: ""  # NOT applied — goal: find_root_cause_only. Planning owns the fix.
verification: "Diagnosis established by direct evidence only: CI job JSON + failed log (single FAIL; windows step cascade), local POSIX differential pass, Go stdlib source inspection of the Windows open/remove semantics, testify assertion source, and git provenance. No fix applied per goal."
files_changed: []

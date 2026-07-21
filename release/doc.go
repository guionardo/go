/*
Package release provides a complete self-update mechanism for Go CLI tools
distributed via GitHub Releases.

The package handles the full update lifecycle:
  - Version detection from Go build info (debug.ReadBuildInfo)
  - Checking GitHub Releases for newer versions
  - Downloading and verifying platform-specific release assets
  - Atomic binary replacement via an embedded swapper process
  - Cleanup and rollback on failure

Asset naming convention:

	Assets are matched by the findAsset function, which looks for
	the GOOS and GOARCH strings (lowercase) in the asset name.
	Recommended naming: {binary}_{goos}_{goarch}[.exe|.tar.gz]

	Examples:
	  myapp_linux_amd64
	  myapp_darwin_amd64
	  myapp_darwin_arm64
	  myapp_windows_amd64.exe

Release asset digest:

	Each Asset carries a Digest field in "sha256:<hex>" (go-digest format).
	The release workflow must generate and attach this digest metadata.
	See release/README.md for workflow examples in Go, Python, and .NET.

Architecture (self-update flow):

 1. PerformSelfUpdate reads the current version from build info.

 2. It calls CheckForUpdate to compare against the latest GitHub release.

 3. If a newer version exists, it acquires a file lock (.update.lock).

 4. DownloadUpdate downloads the platform-specific asset and verifies its digest.

 5. ExtractSwapper writes the embedded swapper binary to the executable directory.

 6. The swapper is spawned with --new-binary, --checksum, --target flags.

 7. The parent process exits; the swapper performs the atomic swap:
    - Backs up the old binary (exe.bak)
    - Renames the new binary into place
    - Re-verifies the checksum
    - Restores from backup on failure

 8. The swapper relaunches the replaced binary with original CLI arguments.

    Platform support:
    - Linux (amd64)
    - macOS (amd64, arm64)
    - Windows (amd64)

    On Unix, relaunch uses syscall.Exec (process replacement, no new PID).
    On Windows, relaunch uses os.StartProcess (new process, then exit).
*/
package release

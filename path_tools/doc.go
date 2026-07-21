// Package pathtools provides file and directory path utilities.
//
// Functions:
//   - DirExists: check if a path is an existing directory
//   - FileExists: check if a path is an existing file
//   - CreatePath: recursively create a directory with permissions inherited from its parent
//   - GetRootFolder: find the Go module root by searching upward for go.mod
//   - IsRootDirectory: check if a path is a filesystem root
//   - FindFileInPath: search PATH environment variable for an executable
package pathtools

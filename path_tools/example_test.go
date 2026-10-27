package pathtools_test

import (
	"fmt"
	"os"
	"path/filepath"

	pathtools "github.com/guionardo/go/path_tools"
)

// ExampleCreatePath builds a directory tree inside a temporary directory and
// shows that CreatePath is a no-op once the path already exists.
func ExampleCreatePath() {
	dir, err := os.MkdirTemp("", "pathtools-example")
	if err != nil {
		fmt.Println("temp dir failed:", err)

		return
	}

	defer func() { _ = os.RemoveAll(dir) }()

	target := filepath.Join(dir, "a", "b", "c")

	fmt.Println(pathtools.DirExists(target))
	fmt.Println(pathtools.CreatePath(target))
	fmt.Println(pathtools.DirExists(target))
	fmt.Println(pathtools.CreatePath(target))

	// Output:
	// false
	// <nil>
	// true
	// <nil>
}

// ExampleFileExists contrasts FileExists with DirExists: each one rejects the
// kind of path the other accepts.
func ExampleFileExists() {
	dir, err := os.MkdirTemp("", "pathtools-example")
	if err != nil {
		fmt.Println("temp dir failed:", err)

		return
	}

	defer func() { _ = os.RemoveAll(dir) }()

	file := filepath.Join(dir, "notes.txt")

	fmt.Println(pathtools.FileExists(file))

	if err := os.WriteFile(file, []byte("hello"), 0o600); err != nil {
		fmt.Println("write failed:", err)

		return
	}

	fmt.Println(pathtools.FileExists(file))
	fmt.Println(pathtools.DirExists(file))
	fmt.Println(pathtools.FileExists(dir))

	// Output:
	// false
	// true
	// false
	// false
}

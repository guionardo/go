package projectprobe_test

import (
	"fmt"
	"os"

	projectprobe "github.com/guionardo/go/project_probe"
)

// ExampleProbe probes a temporary folder and prints only deterministic
// fields — the language and the error — never the temp path.
func ExampleProbe() {
	dir, _ := os.MkdirTemp("", "probe")
	defer func() { _ = os.RemoveAll(dir) }()

	data, err := projectprobe.Probe(dir)
	fmt.Printf("language=%s err=%v", data.Language, err)

	// Output:
	// language=unknown err=<nil>
}

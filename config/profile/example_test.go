package profile_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/guionardo/go/config/profile"
)

func ExampleGetScopedProfileContent() {
	dir, err := os.MkdirTemp("", "profile-example")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(dir) //nolint:errcheck // example cleanup

	files := map[string]string{
		"default.yml":    "name: app\nport: 8080\nserver:\n  timeout: 30\n",
		"production.yml": "port: 9090\nserver:\n  timeout: 120\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			panic(err)
		}
	}

	merged, err := profile.GetScopedProfileContent(dir, "default", "production")
	if err != nil {
		panic(err)
	}

	fmt.Print(string(merged))
	// Output:
	// name: app
	// port: 9090
	// server:
	//     timeout: 120
}

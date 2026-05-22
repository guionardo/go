// Package shelltools provides utilities for shell environment interaction,
// including case-insensitive environment variable lookup and quoted shell
// argument parsing.
package shelltools

import (
	"os"
	"strings"
)

// GetEnv returns the value of the environment variable with the given name from os.Getenv
// In some cases, the environment variable is not set, so we need to try to read it from the system environment
// The system environment is the environment variable that is set in the system, not the environment variable that is
// set in the current process
func GetEnv(name string) string {
	value, ok := os.LookupEnv(name)
	if ok && value != "" {
		return value
	}

	// Try to read it from the system environment
	for _, env := range os.Environ() {
		if value, ok := strings.CutPrefix(env, name+"="); ok {
			return value
		}
	}

	return ""
}

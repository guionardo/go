package shelltools

import (
	"os"
)

// GetEnv returns the value of the environment variable with the given name.
func GetEnv(name string) string {
	value, _ := os.LookupEnv(name)
	return value
}

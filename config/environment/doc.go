// Package environment provides utilities for reading configuration
// from environment variables into struct fields.
//
// Uses env and default struct tags. Supports string, int, uint, bool,
// float, time.Duration, and nested struct types. Matching is
// case-insensitive.
//
// Example:
//
//	type Config struct {
//	    Port int    `env:"APP_PORT" default:"8080"`
//	    Host string `env:"APP_HOST" default:"localhost"`
//	}
//
// Functions:
//   - GetEnv: get env var with optional default
//   - ParseEnvironment: populate a struct from env vars using struct tags
package environment

// Package config provides a generic typed configuration provider with
// YAML profile loading, environment variable overrides, and struct validation.
package config

const (
	// DefaultScope is the default configuration scope name used when none is specified.
	DefaultScope = "default"

	// DefaultConfigurationPath is the default directory path for configuration files.
	DefaultConfigurationPath = "./CONFIGS"

	// EnvScope is the environment variable name for overriding the configuration scope.
	EnvScope = "SCOPE"

	// EnvConfigurationLog is the environment variable name for enabling configuration logging.
	EnvConfigurationLog = "CONFIGURATION_LOG"

	// EnvDefaultScope is the environment variable name for overriding the default scope.
	EnvDefaultScope = "DEFAULT_SCOPE"

	// EnvProfilesPath is the environment variable name for overriding the profiles directory path.
	EnvProfilesPath = "PROFILES_PATH"
)

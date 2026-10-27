package config

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/guionardo/go/config/environment"
)

func (p *provider) postInit() *provider {
	// Set the default scope if not set
	p.defaultScope = cmp.Or(p.defaultScope, DefaultScope)

	// Set the scope if not set
	if p.scope == "" {
		p.scope = environment.GetEnv(EnvScope, p.defaultScope)
	}

	// Set the profiles path if not set
	if p.profilesPath == "" {
		p.profilesPath = environment.GetEnv(EnvProfilesPath, DefaultConfigurationPath)
	}

	p.log().Info("configuration provider initialized",
		slog.String("defaultScope", p.defaultScope),
		slog.String("scope", p.scope),
		slog.String("profilesPath", p.profilesPath))

	return p
}

// WithProfilesPath sets the base directory for YAML profile files.
// Panics if the directory does not exist.
func WithProfilesPath(profilesPath string) providerOption {
	if _, err := os.Stat(profilesPath); err != nil {
		panic(fmt.Errorf("profiles path does not exist: %w", err))
	}

	return func(p *provider) {
		p.profilesPath = profilesPath
	}
}

// WithSingleConfigFile loads a single YAML file instead of a default/scope pair.
// configFile is the file name, with or without extension; the extension is
// resolved by the profile loader (.yml, .yaml, .YML, .YAML). Any directory in
// configFile becomes the profiles directory, and a bare name reads from the
// working directory.
//
// The file's base name is used as both the default and the active scope, so no
// second scope file is required and SCOPE is ignored.
// Panics if configFile has no usable file name.
func WithSingleConfigFile(configFile string) providerOption {
	base := filepath.Base(strings.TrimSpace(configFile))
	name := strings.TrimSuffix(base, filepath.Ext(base))

	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		panic(errors.New("WithSingleConfigFile requires a config file name"))
	}

	return func(p *provider) {
		p.profilesPath = filepath.Dir(strings.TrimSpace(configFile))
		p.defaultScope = name
		p.scope = name
	}
}

// WithLogger sets a custom logger for configuration events.
// The logger receives info, debug, warn, and error messages during
// configuration loading and updates.
func WithLogger(logger Logger) providerOption {
	return func(p *provider) {
		p.logger = logger
	}
}

// WithDebugLogger enables debug-level logging for configuration operations.
// This should NOT be used in production as it may log sensitive configuration values.
//
// It turns on the provider's debug messages and logs them through the Logger
// already injected by WithLogger, or through a debug-level slog logger writing
// to stdout when no logger has been injected yet.
func WithDebugLogger() providerOption {
	return func(p *provider) {
		p.debug = true

		if p.logger == nil {
			level := slog.LevelDebug
			p.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
				With("module", "configuration")
		}

		p.log().Warn("DEBUG LOGGER ENABLED - THIS SHOULD NOT BE USED IN PRODUCTION")
	}
}

// WithScope sets the configuration scope name used for profile selection.
// Scope is used to pick a scope-specific YAML file (e.g., "production", "development").
func WithScope(scope string) providerOption {
	return func(p *provider) {
		p.scope = scope
	}
}

// WithDefaultScope sets the fallback configuration scope name.
// The default scope is used when no specific scope is configured.
func WithDefaultScope(defaultScope string) providerOption {
	return func(p *provider) {
		p.defaultScope = defaultScope
	}
}

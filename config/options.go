package config

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"

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
		p.profilesPath = environment.GetEnv(EnvConfigurationLog, DefaultConfigurationPath)
	}

	logger().Info("configuration provider initialized",
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
func WithDebugLogger() providerOption {
	return func(p *provider) {
		level := slog.LevelDebug
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
			With("module", "configuration")
		p.logger = logger
		logger.Warn("DEBUG LOGGER ENABLED - THIS SHOULD NOT BE USED IN PRODUCTION")
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

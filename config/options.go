package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/guionardo/go/config/environment"
)

func (p *provider) postInit() *provider {
	// Set the default scope if not set
	if p.defaultScope == "" {
		p.defaultScope = DefaultScope
	}

	// Set the scope if not set
	if p.scope == "" {
		p.scope = environment.GetEnv(EnvScope, p.defaultScope)
	}

	// Set the profiles path if not set
	if p.profilesPath == "" {
		p.profilesPath = environment.GetEnv(EnvConfigurationLog, DefaultConfigurationPath)
	}

	log().Info("configuration provider initialized",
		slog.String("defaultScope", p.defaultScope),
		slog.String("scope", p.scope),
		slog.String("profilesPath", p.profilesPath))

	return p
}

func WithProfilesPath(profilesPath string) providerOption {
	if _, err := os.Stat(profilesPath); err != nil {
		panic(fmt.Errorf("profiles path does not exist: %w", err))
	}

	return func(p *provider) {
		p.profilesPath = profilesPath
	}
}

func WithLogger(logger Logger) providerOption {
	return func(p *provider) {
		p.logger = logger
	}
}

func WithDebugLogger() providerOption {
	return func(p *provider) {
		level := slog.LevelDebug
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
			With("module", "configuration")
		p.logger = logger
		logger.Warn("DEBUG LOGGER ENABLED - THIS SHOULD NOT BE USED IN PRODUCTION")
	}
}

func WithScope(scope string) providerOption {
	return func(p *provider) {
		p.scope = scope
	}
}

func WithDefaultScope(defaultScope string) providerOption {
	return func(p *provider) {
		p.defaultScope = defaultScope
	}
}

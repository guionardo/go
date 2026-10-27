package config

import (
	"log/slog"

	"github.com/guionardo/go/config/environment"
	"github.com/guionardo/go/config/validation"
)

type (
	provider struct {
		profilesPath string
		logger       Logger
		scope        string
		defaultScope string
		debug        bool
	}
)

// log returns the logger injected through WithLogger, or the package default
// — slog with module=config — when none was injected.
func (p *provider) log() Logger {
	if p.logger != nil {
		return p.logger
	}

	return logger()
}

// debugLog logs at debug level only when WithDebugLogger enabled it.
func (p *provider) debugLog(msg string, args ...any) {
	if p.debug {
		p.log().Debug(msg, args...)
	}
}

func (p *provider) getProfilesPath() string {
	if p.profilesPath == "" {
		p.profilesPath = environment.GetEnv(EnvProfilesPath)
		if p.profilesPath != "" {
			p.log().Warn("profiles path found in environment variables", slog.String("profilesPath", p.profilesPath))
		}
	}

	return p.profilesPath
}

func (p *provider) validateConfiguration(configuration any) error {
	if validator, ok := configuration.(validation.Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return validation.Validate(configuration)
}

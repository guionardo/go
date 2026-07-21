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
	}
)

func (p *provider) getProfilesPath() string {
	if p.profilesPath == "" {
		p.profilesPath = environment.GetEnv(EnvProfilesPath)
		if p.profilesPath != "" {
			slog.Warn("profiles path found in environment variables", slog.String("profilesPath", p.profilesPath))
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

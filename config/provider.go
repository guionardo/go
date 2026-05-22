package config

import (
	"log/slog"
	"reflect"
	"sync"

	"github.com/guionardo/go/config/environment"
	"github.com/guionardo/go/config/profile"
	"gopkg.in/yaml.v3"
)

type (
	providerOption  func(*provider)
	Provider[T any] struct {
		provider

		lock          sync.RWMutex
		configuration T
		loaded        bool
	}
	Logger interface {
		Info(msg string, args ...any)
		Error(msg string, args ...any)
		Debug(msg string, args ...any)
		Warn(msg string, args ...any)
	}
)

func NewProvider[T any](options ...providerOption) *Provider[T] {
	typeOf := reflect.TypeFor[T]()
	if typeOf.Kind() != reflect.Struct {
		panic("configuration type must be a struct")
	}

	provider := &provider{
		defaultScope: DefaultScope,
		scope:        environment.GetEnv(EnvScope, DefaultScope),
		profilesPath: environment.GetEnv(EnvProfilesPath),
	}
	for _, option := range options {
		option(provider)
	}

	return &Provider[T]{provider: *provider.postInit()}
}

func (p *Provider[T]) GetConfiguration() (T, error) {
	if !p.loaded {
		err := p.loadStaticConfiguration()
		return p.configuration, err
	}

	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.configuration, nil
}

func (p *Provider[T]) UpdateConfiguration(configuration T) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.updateConfiguration(configuration)
}

func (p *Provider[T]) updateConfiguration(configuration T) error {
	if err := p.validateConfiguration(configuration); err != nil {
		log().Error("error validating configuration", "error", err)
		return err
	}

	// Compare the configuration with the previous configuration
	if reflect.DeepEqual(p.configuration, configuration) {
		log().Info("configuration is the same as the previous configuration, skipping update")
		return nil
	}

	p.configuration = configuration
	p.loaded = true

	log().Info("configuration updated", getConfigurationLog(configuration))

	return nil
}

// loadStaticConfiguration loads the static configuration from the scope files and the environment variables
func (p *Provider[T]) loadStaticConfiguration() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var configuration T

	if profilesPath := p.getProfilesPath(); profilesPath == "" {
		slog.Info("no profiles path found, skipping profile loading")
	} else {
		content, err := profile.GetScopedProfileContent(p.profilesPath, p.defaultScope, p.scope)
		if err != nil {
			log().Error("error reading profile", "error", err)
		} else if err := yaml.Unmarshal(content, &configuration); err != nil {
			log().Error("error unmarshalling profile", "error", err)
		}
	}

	if err := environment.ParseEnvironment(&configuration, nil); err != nil {
		log().Error("error parsing environment", "error", err)
	}

	return p.updateConfiguration(configuration)
}

package config

import (
	"log/slog"
	"reflect"
	"sync"

	"github.com/guionardo/go/config/environment"
	"github.com/guionardo/go/config/profile"
	"gopkg.in/yaml.v3"
)

// providerOption is a functional option for configuring a Provider.
type providerOption func(*provider)

// Provider is a generic typed configuration provider that loads configuration
// from YAML profiles and environment variables, with struct validation.
//
// The zero value is not usable directly; use NewProvider to create an instance.
type Provider[T any] struct {
	provider

	lock          sync.RWMutex
	configuration T
	loaded        bool
}

// Logger defines the logging interface used by Provider for configuration events.
// Implementations can use log/slog, testing.T.Logf, or any custom logger.
type Logger interface {
	// Info logs a message at info level.
	Info(msg string, args ...any)
	// Error logs a message at error level.
	Error(msg string, args ...any)
	// Debug logs a message at debug level.
	Debug(msg string, args ...any)
	// Warn logs a message at warn level.
	Warn(msg string, args ...any)
}

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

// GetConfiguration returns the current configuration, loading it from YAML
// profiles and environment variables on the first call. Subsequent calls
// return the cached configuration. Safe for concurrent use.
func (p *Provider[T]) GetConfiguration() (T, error) {
	p.lock.RLock()
	if p.loaded {
		defer p.lock.RUnlock()
		return p.configuration, nil
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.loaded {
		if err := p.loadStaticConfiguration(); err != nil {
			return p.configuration, err
		}
	}
	return p.configuration, nil
}

// UpdateConfiguration replaces the current configuration and re-validates it.
// Returns an error if validation fails. Safe for concurrent use.
func (p *Provider[T]) UpdateConfiguration(configuration T) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.updateConfiguration(configuration)
}

func (p *Provider[T]) updateConfiguration(configuration T) error {
	if err := p.validateConfiguration(configuration); err != nil {
		logger().Error("error validating configuration", "error", err)
		return err
	}

	// Compare the configuration with the previous configuration
	if reflect.DeepEqual(p.configuration, configuration) {
		logger().Info("configuration is the same as the previous configuration, skipping update")
		return nil
	}

	p.configuration = configuration
	p.loaded = true

	logger().Info("configuration updated", getConfigurationLog(configuration))

	return nil
}

// loadStaticConfiguration loads the static configuration from the scope files and the environment variables.
// Caller MUST hold p.lock write lock.
func (p *Provider[T]) loadStaticConfiguration() error {

	var configuration T

	if profilesPath := p.getProfilesPath(); profilesPath == "" {
		slog.Info("no profiles path found, skipping profile loading")
	} else {
		content, err := profile.GetScopedProfileContent(p.profilesPath, p.defaultScope, p.scope)
		if err != nil {
			logger().Error("error reading profile", "error", err)
		} else if err := yaml.Unmarshal(content, &configuration); err != nil {
			logger().Error("error unmarshalling profile", "error", err)
		}
	}

	if err := environment.ParseEnvironment(&configuration, nil); err != nil {
		logger().Error("error parsing environment", "error", err)
	}

	return p.updateConfiguration(configuration)
}

package config_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/guionardo/go/config"
)

// appConfig is the typed configuration both examples load.
type appConfig struct {
	Host string `yaml:"host" validate:"required"`
	Port int    `yaml:"port" validate:"required,min=1,max=65535"`
}

// newAppConfigProvider writes a one-file YAML profile to a temporary directory
// and returns a provider bound to it, plus a cleanup function. Examples get no
// *testing.T, so t.TempDir() is not available here.
func newAppConfigProvider() (*config.Provider[appConfig], func()) {
	dir, err := os.MkdirTemp("", "config-example")
	if err != nil {
		panic(err)
	}

	content := "host: example.com\nport: 8080\n"
	if writeErr := os.WriteFile(filepath.Join(dir, "app.yml"), []byte(content), 0o600); writeErr != nil {
		panic(writeErr)
	}

	provider := config.NewProvider[appConfig](
		config.WithSingleConfigFile(filepath.Join(dir, "app.yml")),
	)

	return provider, func() { _ = os.RemoveAll(dir) }
}

// ExampleNewProvider loads the configuration from a single YAML file.
// WithSingleConfigFile uses the file's base name as both scopes, so no second
// file is needed. GetConfiguration loads once and then serves the cached value.
func ExampleNewProvider() {
	provider, cleanup := newAppConfigProvider()
	defer cleanup()

	cfg, err := provider.GetConfiguration()
	fmt.Println(cfg.Host, cfg.Port, err)

	// Output:
	// example.com 8080 <nil>
}

// ExampleProvider_UpdateConfiguration replaces the configuration at runtime.
// A valid value replaces the cached one; an invalid value is rejected by the
// validate tags and leaves the previous configuration in place.
func ExampleProvider_UpdateConfiguration() {
	provider, cleanup := newAppConfigProvider()
	defer cleanup()

	fmt.Println(provider.UpdateConfiguration(appConfig{Host: "db.internal", Port: 5432}))

	cfg, err := provider.GetConfiguration()
	fmt.Println(cfg.Host, cfg.Port, err)

	fmt.Println(provider.UpdateConfiguration(appConfig{Host: "db.internal", Port: 0}) != nil)

	// Output:
	// <nil>
	// db.internal 5432 <nil>
	// true
}

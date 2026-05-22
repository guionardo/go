package config

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Name    string      `yaml:"name" env:"TESTCFG_NAME" validate:"required"`
	Version int         `yaml:"version" env:"TESTCFG_VERSION"`
	Secret  string      `yaml:"secret" safe:"true"`
	Nested  testSubConfig `yaml:"nested"`
}

type testSubConfig struct {
	Enabled bool   `yaml:"enabled" env:"TESTCFG_ENABLED"`
	Tags    string `yaml:"tags"`
}

type testValidatorConfig struct {
	Value string
}

func (v testValidatorConfig) Validate() error {
	if v.Value == "" {
		return errors.New("value is required")
	}
	return nil
}

type testInvalidConfig struct {
	Name string `validate:"required"`
}

func TestLog(t *testing.T) {
	t.Parallel()

	l := log()
	assert.NotNil(t, l)
	assert.True(t, l.Enabled(context.Background(), slog.LevelInfo))
}

func TestFieldPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", fieldPath())
	assert.Equal(t, "a", fieldPath("a"))
	assert.Equal(t, "a.b", fieldPath("a", "b"))
	assert.Equal(t, "a.b.c", fieldPath("a", "b", "c"))
	assert.Equal(t, "a.b.c", fieldPath("", "a", "b", "c"))
}

func TestGetMapFromStruct(t *testing.T) {
	t.Parallel()

	t.Run("non_struct_value", func(t *testing.T) {
		t.Parallel()

		result := getMapFromStruct("hello", "key")
		assert.Equal(t, map[string]any{"key": "hello"}, result)
	})

	t.Run("struct_with_fields", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig{
			Name:    "test",
			Version: 1,
			Secret:  "should_not_appear",
		}
		result := getMapFromStruct(cfg, "")
		assert.Equal(t, "test", result["Name"])
		assert.Equal(t, 1, result["Version"])
		assert.Equal(t, "********", result["Secret"])
	})

	t.Run("nested_struct", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig{
			Name: "test",
			Nested: testSubConfig{
				Enabled: true,
				Tags:    "a,b",
			},
		}
		result := getMapFromStruct(cfg, "")
		assert.Equal(t, "test", result["Name"])
		assert.Equal(t, true, result["Nested.Enabled"])
		assert.Equal(t, "a,b", result["Nested.Tags"])
	})

	t.Run("empty_struct", func(t *testing.T) {
		t.Parallel()

		result := getMapFromStruct(testConfig{}, "")
		assert.Equal(t, "", result["Name"])
		assert.Equal(t, 0, result["Version"])
		assert.Equal(t, "********", result["Secret"])
	})
}

func TestGetConfigurationLog(t *testing.T) {
	t.Parallel()

	cfg := testConfig{Name: "test-name", Version: 42}
	attr := getConfigurationLog(cfg)

	assert.Equal(t, "testConfig", attr.Key)
	require.NotNil(t, attr.Value)
}

func TestProviderNew(t *testing.T) {
	t.Run("default_values", func(t *testing.T) {
		provider := NewProvider[testConfig]()
		require.NotNil(t, provider)
		assert.Equal(t, DefaultScope, provider.defaultScope)
		assert.Equal(t, "default", provider.scope)
		assert.Equal(t, DefaultConfigurationPath, provider.profilesPath)
	})

	t.Run("with_options", func(t *testing.T) {
		tmp := t.TempDir()
		provider := NewProvider[testConfig](
			WithScope("production"),
			WithDefaultScope("base"),
			WithProfilesPath(tmp),
		)
		require.NotNil(t, provider)
		assert.Equal(t, "production", provider.scope)
		assert.Equal(t, "base", provider.defaultScope)
		assert.Equal(t, tmp, provider.profilesPath)
	})

	t.Run("panic_on_non_struct", func(t *testing.T) {
		assert.Panics(t, func() {
			NewProvider[string]()
		})
	})
}

func TestWithProfilesPath(t *testing.T) {
	t.Run("valid_path", func(t *testing.T) {
		tmp := t.TempDir()
		opt := WithProfilesPath(tmp)
		require.NotNil(t, opt)
	})

	t.Run("invalid_path_panics", func(t *testing.T) {
		assert.Panics(t, func() {
			WithProfilesPath("/nonexistent/path")
		})
	})
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	opt := WithLogger(logger)
	require.NotNil(t, opt)

	p := &provider{}
	opt(p)
	assert.Equal(t, logger, p.logger)
}

func TestWithDebugLogger(t *testing.T) {
	opt := WithDebugLogger()
	require.NotNil(t, opt)

	p := &provider{}
	opt(p)
	require.NotNil(t, p.logger)
}

func TestWithScope(t *testing.T) {
	t.Parallel()

	opt := WithScope("staging")
	require.NotNil(t, opt)

	p := &provider{}
	opt(p)
	assert.Equal(t, "staging", p.scope)
}

func TestWithDefaultScope(t *testing.T) {
	t.Parallel()

	opt := WithDefaultScope("base")
	require.NotNil(t, opt)

	p := &provider{}
	opt(p)
	assert.Equal(t, "base", p.defaultScope)
}

func TestProviderGetConfiguration_ProfileError(t *testing.T) {
	t.Run("invalid_yaml_in_profile", func(t *testing.T) {
		tmp := t.TempDir()
		profilePath := path.Join(tmp, "default.yml")
		require.NoError(t, os.WriteFile(profilePath, []byte(":::: invalid yaml ::::"), 0644))

		provider := NewProvider[testConfig](
			WithProfilesPath(tmp),
			WithDefaultScope("default"),
			WithScope("default"),
		)
		cfg, err := provider.GetConfiguration()
		require.Error(t, err)
		require.Empty(t, cfg.Name)
	})
}

func TestProviderGetConfiguration(t *testing.T) {
	t.Run("from_environment", func(t *testing.T) {
		t.Setenv("TESTCFG_NAME", "env-name")
		t.Setenv("TESTCFG_VERSION", "99")

		provider := NewProvider[testConfig]()
		cfg, err := provider.GetConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "env-name", cfg.Name)
		assert.Equal(t, 99, cfg.Version)
	})

	t.Run("caches_result", func(t *testing.T) {
		t.Setenv("TESTCFG_NAME", "first")

		provider := NewProvider[testConfig]()
		cfg1, err := provider.GetConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "first", cfg1.Name)

		t.Setenv("TESTCFG_NAME", "second")

		cfg2, err := provider.GetConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "first", cfg2.Name)
	})

	t.Run("from_profile_yaml", func(t *testing.T) {
		tmp := t.TempDir()
		profilePath := path.Join(tmp, "default.yml")
		err := os.WriteFile(profilePath, []byte("name: profile-name\nversion: 42\nnested:\n  enabled: true\n  tags: x,y"), 0644)
		require.NoError(t, err)

		provider := NewProvider[testConfig](
			WithProfilesPath(tmp),
			WithDefaultScope("default"),
			WithScope("default"),
		)
		cfg, err := provider.GetConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "profile-name", cfg.Name)
		assert.Equal(t, 42, cfg.Version)
		assert.True(t, cfg.Nested.Enabled)
		assert.Equal(t, "x,y", cfg.Nested.Tags)
	})

	t.Run("env_overrides_profile", func(t *testing.T) {
		tmp := t.TempDir()
		profilePath := path.Join(tmp, "default.yml")
		err := os.WriteFile(profilePath, []byte("name: from-profile\nversion: 10"), 0644)
		require.NoError(t, err)

		t.Setenv("TESTCFG_NAME", "from-env")

		provider := NewProvider[testConfig](
			WithProfilesPath(tmp),
			WithDefaultScope("default"),
			WithScope("default"),
		)
		cfg, err := provider.GetConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "from-env", cfg.Name)
		assert.Equal(t, 10, cfg.Version)
	})

	t.Run("concurrent_safe", func(t *testing.T) {
		t.Setenv("TESTCFG_NAME", "concurrent")

		provider := NewProvider[testConfig]()
		_, err := provider.GetConfiguration()
		require.NoError(t, err)

		done := make(chan struct{})
		go func() {
			_, _ = provider.GetConfiguration()
			close(done)
		}()

		_, err = provider.GetConfiguration()
		require.NoError(t, err)
		<-done
	})
}

func TestProviderUpdateConfiguration(t *testing.T) {
	t.Run("successful_update", func(t *testing.T) {
		provider := NewProvider[testConfig]()
		cfg := testConfig{Name: "updated", Version: 5}
		err := provider.UpdateConfiguration(cfg)
		require.NoError(t, err)

		loaded, err := provider.GetConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "updated", loaded.Name)
		assert.Equal(t, 5, loaded.Version)
	})

	t.Run("validation_error", func(t *testing.T) {
		provider := NewProvider[testConfig]()
		cfg := testConfig{Name: ""}
		err := provider.UpdateConfiguration(cfg)
		require.Error(t, err)
	})

	t.Run("same_config_skips", func(t *testing.T) {
		provider := NewProvider[testConfig]()
		cfg := testConfig{Name: "same", Version: 1}
		err := provider.UpdateConfiguration(cfg)
		require.NoError(t, err)

		err = provider.UpdateConfiguration(cfg)
		require.NoError(t, err)
	})
}

func TestValidateConfiguration(t *testing.T) {
	t.Run("valid_struct", func(t *testing.T) {
		p := &provider{}
		err := p.validateConfiguration(testConfig{Name: "valid", Version: 1})
		require.NoError(t, err)
	})

	t.Run("invalid_struct", func(t *testing.T) {
		p := &provider{}
		err := p.validateConfiguration(testConfig{Name: ""})
		require.Error(t, err)
	})

	t.Run("with_validator_interface", func(t *testing.T) {
		p := &provider{}
		err := p.validateConfiguration(testValidatorConfig{Value: "ok"})
		require.NoError(t, err)
	})

	t.Run("with_validator_interface_invalid", func(t *testing.T) {
		p := &provider{}
		err := p.validateConfiguration(testValidatorConfig{Value: ""})
		require.Error(t, err)
	})

	t.Run("nested_struct_validation", func(t *testing.T) {
		p := &provider{}

		type innerStruct struct {
			InnerName string `validate:"required"`
		}
		type outerStruct struct {
			Inner innerStruct
		}

		err := p.validateConfiguration(outerStruct{})
		require.Error(t, err)
	})

	t.Run("nested_struct_validation_valid", func(t *testing.T) {
		p := &provider{}

		type innerStruct struct {
			InnerName string `validate:"required"`
		}
		type outerStruct struct {
			Inner innerStruct
		}

		err := p.validateConfiguration(outerStruct{Inner: innerStruct{InnerName: "ok"}})
		require.NoError(t, err)
	})
}

func TestGetProfilesPath(t *testing.T) {
	t.Run("already_set", func(t *testing.T) {
		p := &provider{profilesPath: "/custom/path"}
		assert.Equal(t, "/custom/path", p.getProfilesPath())
	})

	t.Run("from_env", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("PROFILES_PATH", tmp)
		p := &provider{}
		result := p.getProfilesPath()
		assert.Equal(t, tmp, result)
	})

	t.Run("empty_when_not_set", func(t *testing.T) {
		p := &provider{}
		result := p.getProfilesPath()
		assert.Equal(t, "", result)
	})
}

func TestPostInit(t *testing.T) {
	t.Run("sets_defaults", func(t *testing.T) {
		p := (&provider{}).postInit()
		assert.Equal(t, DefaultScope, p.defaultScope)
		assert.Contains(t, []string{DefaultScope}, p.scope)
		assert.Equal(t, DefaultConfigurationPath, p.profilesPath)
	})

	t.Run("preserves_existing_values", func(t *testing.T) {
		p := (&provider{
			defaultScope: "custom-default",
			scope:        "custom-scope",
			profilesPath: "/custom/path",
		}).postInit()
		assert.Equal(t, "custom-default", p.defaultScope)
		assert.Equal(t, "custom-scope", p.scope)
		assert.Equal(t, "/custom/path", p.profilesPath)
	})
}

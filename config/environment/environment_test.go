package environment_test

import (
	"log/slog"
	"testing"

	"github.com/guionardo/go/config/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	TestStruct struct {
		SubStruct SubStruct
		Int       int     `env:"INT" default:"1"`
		Int8      int8    `env:"INT8"`
		Int16     int16   `env:"INT16" default:"16"`
		Int32     int32   `env:"INT32" default:"32"`
		Int64     int64   `env:"INT64" default:"64"`
		Uint      uint    `env:"UINT"`
		Uint8     uint8   `env:"UINT8"`
		Uint16    uint16  `env:"UINT16"`
		Uint32    uint32  `env:"UINT32"`
		Uint64    uint64  `env:"UINT64"`
		Bool      bool    `env:"BOOL"`
		Float64   float64 `env:"FLOAT64"`
		Float32   float32 `env:"FLOAT32"`
		String    string  `env:"STRING" default:"string"`
	}
	SubStruct struct {
		Name string `env:"NAME" default:"sub_name"`
		Age  int    `env:"AGE" default:"18"`
	}
)

func TestGetEnv(t *testing.T) {
	t.Run("returns_env_value", func(t *testing.T) {
		t.Setenv("TEST_GETENV_EXISTS", "found")
		assert.Equal(t, "found", environment.GetEnv("TEST_GETENV_EXISTS"))
	})

	t.Run("returns_default_when_missing", func(t *testing.T) {
		assert.Equal(t, "fallback", environment.GetEnv("TEST_GETENV_MISSING", "fallback"))
	})

	t.Run("returns_empty_when_no_default", func(t *testing.T) {
		assert.Equal(t, "", environment.GetEnv("TEST_GETENV_EMPTY"))
	})

	t.Run("empty_env_returns_empty", func(t *testing.T) {
		assert.Equal(t, "", environment.GetEnv(""))
	})

	t.Run("exact_case_match", func(t *testing.T) {
		t.Setenv("TESTCASE_ENV", "value")
		assert.Equal(t, "value", environment.GetEnv("TESTCASE_ENV"))
		assert.Empty(t, environment.GetEnv("testcase_env"))
	})
}

func TestParseEnvironmentErrors(t *testing.T) {
	t.Run("non_pointer_returns_error", func(t *testing.T) {
		err := environment.ParseEnvironment(TestStruct{}, nil)
		require.Error(t, err)
	})

	t.Run("invalid_int_value_logs_error", func(t *testing.T) {
		ts := TestStruct{}
		t.Setenv("INT", "not-a-number")
		err := environment.ParseEnvironment(&ts, nil)
		require.Error(t, err)
	})
}

func TestParseEnvironment(t *testing.T) { //nolint:paralleltest
	ts := TestStruct{}

	logger := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	t.Setenv("INT", "2")
	t.Setenv("INT16", "17")
	t.Setenv("INT32", "33")
	t.Setenv("INT64", "65")
	t.Setenv("UINT", "2")
	t.Setenv("UINT16", "17")
	t.Setenv("UINT32", "33")
	t.Setenv("UINT64", "65")
	t.Setenv("BOOL", "true")
	t.Setenv("FLOAT64", "0.1")
	t.Setenv("FLOAT32", "0.2")
	t.Setenv("STRING", "string2")
	t.Setenv("NAME", "sub_name2")
	t.Setenv("AGE", "20")

	require.NoError(t, environment.ParseEnvironment(&ts, nil))
	assert.Equal(t, 2, ts.Int)
	assert.Equal(t, int16(17), ts.Int16)
	assert.Equal(t, int32(33), ts.Int32)
	assert.Equal(t, int64(65), ts.Int64)
	assert.Equal(t, uint(2), ts.Uint)
	assert.Equal(t, uint16(17), ts.Uint16)
	assert.Equal(t, uint32(33), ts.Uint32)
	assert.Equal(t, uint64(65), ts.Uint64)
	assert.True(t, ts.Bool)
	assert.InEpsilon(t, float32(0.1), ts.Float64, 0.000001)
	assert.InEpsilon(t, float64(0.2), ts.Float32, 0.000001)
	assert.Equal(t, "string2", ts.String)

	assert.Equal(t, "sub_name2", ts.SubStruct.Name)
	assert.Equal(t, 20, ts.SubStruct.Age)
}

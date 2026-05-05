package shelltools_test

import (
	"testing"

	shelltools "github.com/guionardo/go/shell_tools"
	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) { //nolint: paralleltest
	t.Run("existing_env_should_return_value", func(t *testing.T) { //nolint: paralleltest
		t.Setenv("TEST_ENV", "test_value")

		got := shelltools.GetEnv("TEST_ENV")
		assert.Equal(t, "test_value", got)
	})
	t.Run("unexisting_env_should_return_empty_string", func(t *testing.T) { //nolint: paralleltest
		got := shelltools.GetEnv("UNEXISTENT_ENV")
		assert.Empty(t, got)
	})
}

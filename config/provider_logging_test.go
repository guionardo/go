package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	// logTestConfig has no env or validate tags, so loading it depends on
	// nothing but the profile directory.
	logTestConfig struct {
		Value string
	}

	// recordingLogger captures every call made through the Logger interface.
	recordingLogger struct {
		mu      sync.Mutex
		levels  []string
		entries []string
	}
)

const (
	levelDebug = "DEBUG"
	levelInfo  = "INFO"
	levelWarn  = "WARN"
	levelError = "ERROR"
)

func (r *recordingLogger) Info(msg string, _ ...any)  { r.record(levelInfo, msg) }
func (r *recordingLogger) Error(msg string, _ ...any) { r.record(levelError, msg) }
func (r *recordingLogger) Debug(msg string, _ ...any) { r.record(levelDebug, msg) }
func (r *recordingLogger) Warn(msg string, _ ...any)  { r.record(levelWarn, msg) }

func (r *recordingLogger) messages() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.entries...)
}

func (r *recordingLogger) levelsSeen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.levels...)
}

func (r *recordingLogger) record(level, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.levels = append(r.levels, level)
	r.entries = append(r.entries, msg)
}

func TestProvider_LogsThroughInjectedLogger(t *testing.T) {
	t.Parallel()

	rec := &recordingLogger{}
	p := NewProvider[logTestConfig](WithLogger(rec), WithProfilesPath(t.TempDir()))

	_, err := p.GetConfiguration()
	require.NoError(t, err)
	require.NoError(t, p.UpdateConfiguration(logTestConfig{Value: "updated"}))

	msgs := rec.messages()
	require.NotEmpty(t, msgs, "injected logger received no log output")
	assert.Contains(t, msgs, "configuration provider initialized")
	assert.Contains(t, msgs, "error reading profile")
	assert.Contains(t, msgs, "configuration updated")
}

func TestProvider_WithoutLoggerStillWorks(t *testing.T) {
	t.Parallel()

	p := NewProvider[logTestConfig](WithProfilesPath(t.TempDir()))

	cfg, err := p.GetConfiguration()
	require.NoError(t, err)
	assert.Equal(t, logTestConfig{}, cfg)

	require.NoError(t, p.UpdateConfiguration(logTestConfig{Value: "v"}))
}

func TestProvider_DebugLogger(t *testing.T) {
	t.Parallel()

	t.Run("debug_is_off_by_default", func(t *testing.T) {
		t.Parallel()

		rec := &recordingLogger{}
		p := NewProvider[logTestConfig](WithLogger(rec), WithProfilesPath(t.TempDir()))

		_, err := p.GetConfiguration()
		require.NoError(t, err)

		assert.NotEmpty(t, rec.messages())
		assert.NotContains(t, rec.levelsSeen(), levelDebug)
	})

	t.Run("debug_logger_enables_debug_on_later_logger", func(t *testing.T) {
		t.Parallel()

		rec := &recordingLogger{}
		p := NewProvider[logTestConfig](
			WithDebugLogger(),
			WithLogger(rec),
			WithProfilesPath(t.TempDir()),
		)

		_, err := p.GetConfiguration()
		require.NoError(t, err)

		assert.Contains(t, rec.levelsSeen(), levelDebug)
	})

	t.Run("debug_logger_keeps_earlier_logger", func(t *testing.T) {
		t.Parallel()

		rec := &recordingLogger{}
		p := NewProvider[logTestConfig](
			WithLogger(rec),
			WithDebugLogger(),
			WithProfilesPath(t.TempDir()),
		)

		_, err := p.GetConfiguration()
		require.NoError(t, err)

		assert.Contains(t, rec.levelsSeen(), levelDebug)
	})
}

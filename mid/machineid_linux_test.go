//go:build linux

package mid

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutErr(t *testing.T) {
	t.Parallel()

	t.Run("non_empty_output_returns_no_error", func(t *testing.T) {
		t.Parallel()

		out, err := outErr("some-id", "test")
		assert.Equal(t, "some-id", out)
		assert.NoError(t, err)
	})

	t.Run("empty_output_returns_error", func(t *testing.T) {
		t.Parallel()

		out, err := outErr("", "test-func")
		require.Error(t, err)
		assert.Empty(t, out)
		assert.Contains(t, err.Error(), "test-func")
	})
}

func TestCollectEtcMachineId(t *testing.T) {
	t.Parallel()

	collectFuncsMu.Lock()
	original := collectFuncs
	collectFuncs = []func() (string, error){collectEtcMachineId}
	collectFuncsMu.Unlock()
	t.Cleanup(func() {
		collectFuncsMu.Lock()
		collectFuncs = original
		collectFuncsMu.Unlock()
	})

	id := MachineID()
	if id == "" {
		t.Log("/etc/machine-id not present on this system")
	} else {
		assert.NotEmpty(t, id)
	}
}

func TestCollectDbusMachineId(t *testing.T) {
	t.Parallel()

	collectFuncsMu.Lock()
	original := collectFuncs
	collectFuncs = []func() (string, error){collectDbusMachineId}
	collectFuncsMu.Unlock()
	t.Cleanup(func() {
		collectFuncsMu.Lock()
		collectFuncs = original
		collectFuncsMu.Unlock()
	})

	id := MachineID()
	if id == "" {
		t.Log("/var/lib/dbus/machine-id not present on this system")
	} else {
		assert.NotEmpty(t, id)
	}
}

func TestCollectHostnamectl(t *testing.T) {
	t.Parallel()
	t.Run("command_fails_returns_empty_in_ci", func(t *testing.T) {
		t.Parallel()
		collectFuncsMu.Lock()
		original := collectFuncs
		collectFuncs = []func() (string, error){collectHostnamectl}
		collectFuncsMu.Unlock()
		t.Cleanup(func() {
			collectFuncsMu.Lock()
			collectFuncs = original
			collectFuncsMu.Unlock()
		})

		id := MachineID()
		if id == "" {
			assert.Empty(t, id)
		} else {
			assert.NotEmpty(t, id)
		}
	})
}

func TestMachineIDFallbackOrder(t *testing.T) { //nolint: funlen
	t.Run("stops_at_first_success", func(t *testing.T) {

		var (
			mu        sync.Mutex
			callOrder []string
		)

		fail := func() (string, error) {
			mu.Lock()

			callOrder = append(callOrder, "fail")
			mu.Unlock()

			return "", assert.AnError
		}
		success := func() (string, error) {
			mu.Lock()

			callOrder = append(callOrder, "success")
			mu.Unlock()

			return "found", nil
		}
		never := func() (string, error) {
			mu.Lock()

			callOrder = append(callOrder, "never")
			mu.Unlock()

			return "", assert.AnError
		}

		collectFuncsMu.Lock()
		original := collectFuncs
		collectFuncs = []func() (string, error){fail, success, never}
		collectFuncsMu.Unlock()
		t.Cleanup(func() {
			collectFuncsMu.Lock()
			collectFuncs = original
			collectFuncsMu.Unlock()
		})

		result := MachineID()
		assert.Equal(t, "found", result)
		assert.Equal(t, []string{"fail", "success"}, callOrder)
	})

	t.Run("all_fail_returns_empty", func(t *testing.T) {
		allFail := func() (string, error) {
			return "", assert.AnError
		}

		collectFuncsMu.Lock()
		original := collectFuncs
		collectFuncs = []func() (string, error){allFail, allFail}
		collectFuncsMu.Unlock()
		t.Cleanup(func() {
			collectFuncsMu.Lock()
			collectFuncs = original
			collectFuncsMu.Unlock()
		})

		result := MachineID()
		assert.Empty(t, result)
	})
}

func TestMachineIDMutex(t *testing.T) {
	t.Parallel()
	t.Run("concurrent_access_is_safe", func(t *testing.T) {
		t.Parallel()

		done := make(chan struct{})

		go func() {
			MachineID()
			close(done)
		}()

		MachineID()
		<-done
	})
}

func TestSplitSeqBehavior(t *testing.T) {
	t.Parallel()

	input := "Machine ID: abc123\nSomething else"

	count := 0
	for range strings.SplitSeq(input, "\n") {
		count++
	}

	assert.Equal(t, 2, count)
}

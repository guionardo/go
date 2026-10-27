package mid

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	collectFuncsMu sync.RWMutex
	collectFuncs   = []func() (string, error){collectHostnamectl, collectEtcMachineId, collectDbusMachineId}
)

// MachineID in linux tries hostnamectl, then /etc/machine-id, then
// /var/lib/dbus/machine-id, returning "" when none of them yields a value.
func MachineID() string {
	collectFuncsMu.RLock()

	funcs := collectFuncs

	collectFuncsMu.RUnlock()

	for _, cf := range funcs {
		if c, err := cf(); err == nil {
			return c
		}
	}

	return ""
}

func outErr(out string, funcName string) (string, error) {
	var err error
	if len(out) == 0 {
		err = errors.New(funcName)
	}

	return out, err
}

func collectHostnamectl() (out string, err error) {
	cmd := exec.Command("hostnamectl", "status")

	output, err := cmd.Output()
	if err == nil {
		for line := range strings.SplitSeq(string(output), "\n") {
			if w := strings.Split(line, ":"); len(w) == 2 && strings.TrimSpace(strings.ToLower(w[0])) == "machine id" {
				out = strings.TrimSpace(w[1])
				break
			}
		}
	}

	return outErr(out, "hostnamectl")
}

func collectDbusMachineId() (out string, err error) {
	if content, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		out = strings.TrimSpace(string(content))
	}

	return outErr(out, "dbus")
}

func collectEtcMachineId() (out string, err error) {
	if content, err := os.ReadFile("/etc/machine-id"); err == nil {
		out = strings.TrimSpace(string(content))
	}

	return outErr(out, "etc")
}

package mid

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

var collectFuncs = []func() (string, error){collectHostnamectl, collectEtcMachineId, collectDbusMachineId}

// MachineID in linux use hostnamectl, /var/lib/dbus/machine-id, or /etc/machine-id
func MachineID() string {
	for _, cf := range collectFuncs {
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
		out = string(content)
	}

	return outErr(out, "dbus")
}

func collectEtcMachineId() (out string, err error) {
	if content, err := os.ReadFile("/etc/machine-id"); err == nil {
		out = string(content)
	}

	return outErr(out, "etc")
}

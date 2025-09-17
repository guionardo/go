package mid

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// MachineID in linux use hostnamectl, /var/lib/dbus/machine-id, or /etc/machine-id
func MachineID() string {
	if c, err := collectHostnamectl(); err == nil {
		return c
	}
	if c, err := collectDbusMachineId(); err == nil {
		return c
	}
	if c, err := collectEtcMachineId(); err == nil {
		return c
	}
	return ""
}

func collectHostnamectl() (string, error) {
	cmd := exec.Command("hostnamectl", "status")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(output), "\n") {
		if w := strings.Split(line, ":"); len(w) == 2 && strings.TrimSpace(strings.ToLower(w[0])) == "machine id" {
			return strings.TrimSpace(w[1]), nil
		}
	}
	return "", errors.New("hostnamectl")
}

func collectDbusMachineId() (string, error) {
	if content, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		return string(content), nil
	}
	return "", errors.New("dbus")
}

func collectEtcMachineId() (string, error) {
	if content, err := os.ReadFile("/etc/machine-id"); err == nil {
		return string(content), nil
	}
	return "", errors.New("etc")
}

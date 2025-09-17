package mid

import (
	"fmt"
	"os/exec"
	"strings"
)

// MachineID - returns a string "{model number}|{serial number}|{hardware uuid}"
func MachineID() string {
	cmd := exec.Command("system_profiler", "SPHardwareDataType", "SPSecureElementDataType")
	if output, err := cmd.Output(); err == nil {
		var modelNumber, serialNumber, hardwareUUID string
		for line := range strings.SplitSeq(string(output), "\n") {
			if w := strings.Split(line, ":"); len(w) == 2 {
				key := strings.TrimSpace(strings.ToLower(w[0]))
				value := strings.TrimSpace(w[1])
				switch key {
				case "model number":
					modelNumber = value
				case "serial number":
					serialNumber = value
				case "hardware uuid":
					hardwareUUID = value
				}
			}
		}
		return fmt.Sprintf("%s|%s|%s", modelNumber, serialNumber, hardwareUUID)
	}
	return ""
}

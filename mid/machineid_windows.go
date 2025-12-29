package mid

import (
	"os/exec"
	"regexp"
)

// reg query HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient

// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient
//     CabSessionAfterSize    REG_DWORD    0x5000
//     WinSqmFirstSessionStartTime    REG_QWORD    0x1db1cffbcda2a23
//     MachineId    REG_SZ    {B43D4F72-6478-46AA-AB85-25686B1FA81D}

// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient\CommonUploader
// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient\IE
// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient\Windows
const pattern = `MachineId\s+REG_SZ\s+\{([A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12})\}`

var re = regexp.MustCompile(pattern)

// MachineID returns the MachineID from registry SQMClient
func MachineID() string {
	cmd := exec.Command("reg", "query", `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SQMClient`)
	if output, err := cmd.Output(); err == nil {
		match := re.FindStringSubmatch(string(output))
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

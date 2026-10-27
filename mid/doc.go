// Package mid provides cross-platform machine identification.
//
// It retrieves a unique machine identifier using OS-specific sources:
//
//   - macOS: system_profiler SPHardwareDataType (model + serial + hardware UUID)
//   - Linux: hostnamectl, then /etc/machine-id, then /var/lib/dbus/machine-id
//     (the first source that returns a non-empty value wins)
//   - Windows: SQMClient registry key
//
// Usage:
//
//	id := mid.MachineID()
package mid

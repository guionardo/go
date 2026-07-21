// Package mid provides cross-platform machine identification.
//
// It retrieves a unique machine identifier using OS-specific sources:
//
//   - macOS: system_profiler SPHardwareDataType (model + serial + hardware UUID)
//   - Linux: hostnamectl, /var/lib/dbus/machine-id, or /etc/machine-id (fallback chain)
//   - Windows: SQMClient registry key
//
// Usage:
//
//	id := mid.MachineID()
package mid

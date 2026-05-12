//go:build !linux

package player

// listPlatformDevices is a no-op on non-Linux platforms. macOS and
// Windows route through oto, which doesn't expose per-device
// enumeration today; the canonical "default (system output)" entry
// listed by ListDevices is the only choice.
func listPlatformDevices() []string { return nil }

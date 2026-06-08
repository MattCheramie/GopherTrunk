package usb

import (
	"strconv"
	"strings"
)

// This file holds the platform-independent, pure-function half of the
// Windows composite-device handling (see child_windows.go for the
// SetupAPI/CfgMgr glue). Keeping these helpers free of any build tag lets
// them compile and be table-tested on every platform — the Windows-only
// syscalls in child_windows.go are thin wrappers around the decisions made
// here.

// parseInstanceID extracts the VID, PID, and interface number from a Windows
// device *instance* ID such as:
//
//	USB\VID_0BDA&PID_2838&MI_00\6&1234abcd&0&0000
//
// Unlike a device-interface path (parsed by parseDevicePath, which is
// "#"-delimited and lower-cased), an instance ID is "\"-delimited and
// conventionally upper-cased; comparison here is case-insensitive to tolerate
// either. The MI_xx token is only present on the child function nodes of a USB
// composite device — hasMI reports whether it was found, which is how callers
// tell a composite child apart from a single-interface dongle's whole-device
// node.
func parseInstanceID(id string) (vid, pid uint16, mi int, hasMI bool) {
	lower := strings.ToLower(id)
	if i := strings.Index(lower, "vid_"); i >= 0 && i+8 <= len(id) {
		if v, err := strconv.ParseUint(id[i+4:i+8], 16, 16); err == nil {
			vid = uint16(v)
		}
	}
	if i := strings.Index(lower, "pid_"); i >= 0 && i+8 <= len(id) {
		if v, err := strconv.ParseUint(id[i+4:i+8], 16, 16); err == nil {
			pid = uint16(v)
		}
	}
	if i := strings.Index(lower, "mi_"); i >= 0 && i+5 <= len(id) {
		if v, err := strconv.ParseUint(id[i+3:i+5], 16, 16); err == nil {
			mi = int(v)
			hasMI = true
		}
	}
	return vid, pid, mi, hasMI
}

// isInterfaceZero reports whether an instance ID names the Interface 0
// (&MI_00) child function node — the RTL-SDR's only data interface and the one
// Zadig binds to WinUSB on a composite dongle.
func isInterfaceZero(id string) bool {
	_, _, mi, hasMI := parseInstanceID(id)
	return hasMI && mi == 0
}

// effectiveCompositeService picks the SPDRP_SERVICE value that best represents
// a device's real driver binding for the `sdr doctor` classifier.
//
// For a USB composite device the GUID_DEVINTERFACE_USB_DEVICE node we enumerate
// is the *parent*, which is always bound to "usbccgp" (correct and normal). The
// SDR's actual driver lives on the Interface 0 (&MI_00) *child* function node.
// When such a child is found we report ITS service so the verdict reflects the
// binding the user actually controls in Zadig — turning a correctly
// WinUSB-bound composite dongle from a false "BAD" into "OK", and giving an
// accurate hint (e.g. "libusbK instead of WinUSB") when they bound the wrong
// driver. With no child found we keep the parent's service so the existing
// composite-parent guidance still fires.
func effectiveCompositeService(parentService, childService string, childFound bool) string {
	if childFound && strings.EqualFold(parentService, "usbccgp") {
		return childService
	}
	return parentService
}

// firstInterfaceGUID returns the first non-empty entry from a child node's
// DeviceInterfaceGUIDs (REG_MULTI_SZ) registry value. Zadig/libwdi normally
// writes exactly one; we tolerate blank padding entries. ok is false when the
// list holds nothing usable.
func firstInterfaceGUID(guids []string) (guid string, ok bool) {
	for _, g := range guids {
		if t := strings.TrimSpace(g); t != "" {
			return t, true
		}
	}
	return "", false
}

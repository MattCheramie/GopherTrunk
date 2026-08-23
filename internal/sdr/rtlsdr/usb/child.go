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

// compositeChildCandidate is one Interface 0 (&MI_00) child function node
// found by the composite-device walk, paired with the instance ID of its
// composite PARENT devnode ("" when the parent could not be resolved). The
// serial that tells two identical dongles apart lives on the parent — a child
// instance ID carries only a bus-position discriminator — so the parent link
// is what makes serial-aware selection possible.
type compositeChildCandidate struct {
	InstanceID       string
	ParentInstanceID string
}

// serialFromInstanceID returns the final "\"-separated component of a device
// instance ID. For a USB composite parent ("USB\VID_0BDA&PID_2838\90000002")
// that is the dongle serial — or the system-synthesized "7&xxxx&0&port"
// discriminator when no serial is programmed. Either form also appears
// verbatim (modulo case) as the serial field of the parent's device-interface
// path, so it compares directly against Descriptor.Serial.
func serialFromInstanceID(id string) string {
	if i := strings.LastIndexByte(id, '\\'); i >= 0 {
		return id[i+1:]
	}
	return ""
}

// pickInterfaceZeroChild selects which &MI_00 child belongs to the dongle
// with the given serial, returning its index or -1.
//
// Policy (issue #1131 — two identical composite dongles on one bus):
//  1. no serial to match → first candidate (callers without a serial can do
//     no better; this is the pre-#1131 behaviour);
//  2. a candidate whose parent's serial matches → that candidate;
//  3. no candidate's parent resolved → first candidate (zero information —
//     behave as before rather than break single-dongle rigs on stacks where
//     the parent lookup fails);
//  4. otherwise → -1: at least one parent DID resolve and none matched, so
//     "the first one" is provably some other dongle. Returning it anyway is
//     the #1131 failure: with that dongle already open, WinUsb_Initialize on
//     its child fails ERROR_NOT_ENOUGH_MEMORY; with it closed, the caller
//     silently opens the wrong radio.
func pickInterfaceZeroChild(cands []compositeChildCandidate, serial string) int {
	if len(cands) == 0 {
		return -1
	}
	if serial == "" {
		return 0
	}
	anyResolved := false
	for i, c := range cands {
		if c.ParentInstanceID == "" {
			continue
		}
		anyResolved = true
		if strings.EqualFold(serialFromInstanceID(c.ParentInstanceID), serial) {
			return i
		}
	}
	if !anyResolved {
		return 0
	}
	return -1
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

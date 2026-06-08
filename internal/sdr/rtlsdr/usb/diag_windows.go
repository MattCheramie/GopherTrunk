//go:build windows && (amd64 || arm64)

package usb

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SetupAPI registry-property selectors (SPDRP_*). Values match the
// public Windows SDK header setupapi.h. SPDRP_SERVICE is the bound
// kernel-mode service (e.g. "WinUSB", "RTL2832UUSB"); SPDRP_DEVICEDESC
// is the human-readable description shown in Device Manager.
const (
	spdrpDeviceDesc = 0x00
	spdrpService    = 0x04
)

// SP_DEVINFO_DATA layout on x64. The build tag at the top of this
// file restricts compilation to amd64 / arm64 so the field sizes line
// up with the kernel-mode header.
type spDevInfoData struct {
	Size      uint32
	ClassGuid windows.GUID
	DevInst   uint32
	Reserved  uintptr
}

var (
	procSetupDiEnumDeviceInfo             = modSetupAPI.NewProc("SetupDiEnumDeviceInfo")
	procSetupDiGetDeviceRegistryPropertyW = modSetupAPI.NewProc("SetupDiGetDeviceRegistryPropertyW")
)

func platformDriverInspector() DriverInspector { return &winDriverInspector{} }

// winDriverInspector enumerates every present USB device interface
// (same SetupAPI walk linuxEnumerator-equivalent winEnumerator uses)
// and reads the SPDRP_SERVICE / SPDRP_DRIVER_DESC registry properties
// to surface which function driver is currently bound. Read-only —
// safe to run as a regular user; never claims or opens a device.
type winDriverInspector struct{}

func (w *winDriverInspector) Inspect(vid, pid uint16) ([]DriverBinding, error) {
	devSet, _, errno := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&guidDevInterfaceUSBDevice)),
		0,
		0,
		uintptr(digcfPresentDeviceInterface),
	)
	if devSet == uintptr(windows.InvalidHandle) || devSet == 0 {
		return nil, fmt.Errorf("winusb: SetupDiGetClassDevs: %w", winErr(errno))
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devSet)

	var out []DriverBinding
	for memberIndex := uint32(0); ; memberIndex++ {
		var iface spDeviceInterfaceData
		iface.Size = uint32(unsafe.Sizeof(iface))
		ret, _, errno := procSetupDiEnumDeviceInterfaces.Call(
			devSet,
			0,
			uintptr(unsafe.Pointer(&guidDevInterfaceUSBDevice)),
			uintptr(memberIndex),
			uintptr(unsafe.Pointer(&iface)),
		)
		if ret == 0 {
			if errno == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			return out, fmt.Errorf("winusb: SetupDiEnumDeviceInterfaces[%d]: %w", memberIndex, winErr(errno))
		}
		path, err := getDeviceInterfacePath(devSet, &iface)
		if err != nil {
			continue
		}
		v, p, serial := parseDevicePath(path)
		if v == 0 && p == 0 {
			continue
		}
		if vid != 0 && v != vid {
			continue
		}
		if pid != 0 && p != pid {
			continue
		}
		// Pair the interface entry with its underlying SP_DEVINFO_DATA
		// at the same member index so we can ask for its registry
		// properties. Most builds keep the indices aligned, but the
		// API doesn't guarantee it; on mismatch we record OK=false
		// with the descriptor and skip the driver lookup.
		var devInfo spDevInfoData
		devInfo.Size = uint32(unsafe.Sizeof(devInfo))
		dret, _, _ := procSetupDiEnumDeviceInfo.Call(
			devSet,
			uintptr(memberIndex),
			uintptr(unsafe.Pointer(&devInfo)),
		)
		desc := Descriptor{VID: v, PID: p, Serial: serial, Path: path}
		if dret == 0 {
			out = append(out, classifyWindows(desc, "", ""))
			continue
		}
		service := readDevRegistryString(devSet, &devInfo, spdrpService)
		descr := readDevRegistryString(devSet, &devInfo, spdrpDeviceDesc)
		// A "usbccgp" service means this GUID_DEVINTERFACE_USB_DEVICE node is a
		// USB composite parent — the SDR's real driver lives on its Interface 0
		// (&MI_00) child. Report the child's binding so a correctly
		// WinUSB-bound composite dongle reads OK instead of a false BAD (and a
		// wrong-driver child still gets an accurate hint).
		if strings.EqualFold(service, "usbccgp") {
			if childSvc, _, derr := findInterfaceZeroChild(v, p); derr == nil && childSvc != "" {
				service = effectiveCompositeService(service, childSvc, true)
			}
		}
		out = append(out, classifyWindows(desc, service, descr))
	}
	return out, nil
}

// readDevRegistryString fetches a single SPDRP_* string property for a
// device-info node. Two calls: one with a nil buffer to learn the
// size, one with a sized buffer. Returns "" on any failure — the
// classifier treats an empty service name as "no driver bound".
func readDevRegistryString(devSet uintptr, dev *spDevInfoData, prop uint32) string {
	var required uint32
	procSetupDiGetDeviceRegistryPropertyW.Call(
		devSet,
		uintptr(unsafe.Pointer(dev)),
		uintptr(prop),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&required)),
	)
	if required == 0 {
		return ""
	}
	buf := make([]byte, required)
	ret, _, _ := procSetupDiGetDeviceRegistryPropertyW.Call(
		devSet,
		uintptr(unsafe.Pointer(dev)),
		uintptr(prop),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(required),
		0,
	)
	if ret == 0 {
		return ""
	}
	utf16Len := len(buf) / 2
	if utf16Len == 0 {
		return ""
	}
	utf16Slice := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[0])), utf16Len)
	end := utf16Len
	for i, c := range utf16Slice {
		if c == 0 {
			end = i
			break
		}
	}
	return windows.UTF16ToString(utf16Slice[:end])
}

// lookupBoundDriver returns (service, desc) for a single device-interface
// path — the lighter-weight counterpart to Inspect, used from
// winEnumerator.Open to embed the bound driver in the error message
// when WinUsb_Initialize fails. Returns ("", "") on miss; the caller
// falls back to the raw HRESULT rather than masking it.
func lookupBoundDriver(devPath string) (service, desc string) {
	target := strings.ToLower(devPath)
	devSet, _, _ := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&guidDevInterfaceUSBDevice)),
		0,
		0,
		uintptr(digcfPresentDeviceInterface),
	)
	if devSet == uintptr(windows.InvalidHandle) || devSet == 0 {
		return "", ""
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devSet)

	for memberIndex := uint32(0); ; memberIndex++ {
		var iface spDeviceInterfaceData
		iface.Size = uint32(unsafe.Sizeof(iface))
		ret, _, errno := procSetupDiEnumDeviceInterfaces.Call(
			devSet,
			0,
			uintptr(unsafe.Pointer(&guidDevInterfaceUSBDevice)),
			uintptr(memberIndex),
			uintptr(unsafe.Pointer(&iface)),
		)
		if ret == 0 {
			if errno == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			return "", ""
		}
		path, err := getDeviceInterfacePath(devSet, &iface)
		if err != nil {
			continue
		}
		if strings.ToLower(path) != target {
			continue
		}
		var devInfo spDevInfoData
		devInfo.Size = uint32(unsafe.Sizeof(devInfo))
		dret, _, _ := procSetupDiEnumDeviceInfo.Call(
			devSet,
			uintptr(memberIndex),
			uintptr(unsafe.Pointer(&devInfo)),
		)
		if dret == 0 {
			return "", ""
		}
		return readDevRegistryString(devSet, &devInfo, spdrpService),
			readDevRegistryString(devSet, &devInfo, spdrpDeviceDesc)
	}
	return "", ""
}

// fallbackString returns s when non-empty, otherwise dflt. Used by
// the WinUsb_Initialize error message so a registry-lookup miss
// surfaces as "unknown" rather than an empty parenthesis.
func fallbackString(s, dflt string) string {
	if s == "" {
		return dflt
	}
	return s
}

// parens wraps s in " (...)" when non-empty; returns "" otherwise.
// Keeps the WinUsb_Initialize error grammatical when only the service
// name is available.
func parens(s string) string {
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}

// classifyWindows maps a SPDRP_SERVICE value (or "" for none) to the
// doctor row. Extracted as a pure function so the unit test can
// table-drive the policy without touching SetupAPI.
func classifyWindows(d Descriptor, service, desc string) DriverBinding {
	b := DriverBinding{
		Descriptor: d,
		DriverName: service,
		DriverDesc: desc,
		Expected:   "WinUSB",
	}
	switch strings.ToLower(service) {
	case "winusb":
		b.OK = true
	case "rtl2832uusb", "rtl28xxbda":
		b.Hint = "Windows in-box DVB-T driver is bound. Run Zadig from the Start Menu, Options → List All Devices, pick this dongle's Bulk-In, Interface (Interface 0) entry, choose WinUSB as the target driver, and click Replace Driver."
	case "libusbk", "libusb0":
		b.Hint = "Zadig installed " + service + " instead of WinUSB. Re-run Zadig and pick WinUSB (not libusbK or libusb-win32) as the target driver."
	case "usbccgp":
		b.Hint = "The composite-device parent is bound, not the SDR interface. In Zadig, pick the child Bulk-In, Interface 0 entry (not the parent USB Composite Device) and bind it to WinUSB."
	case "":
		b.Hint = "No driver is bound. Open Device Manager — if the dongle shows a yellow exclamation mark, run Zadig and install the WinUSB driver."
	default:
		b.Hint = "Unexpected driver " + service + " bound; the transport speaks WinUSB. Re-run Zadig and pick WinUSB as the target driver."
	}
	return b
}

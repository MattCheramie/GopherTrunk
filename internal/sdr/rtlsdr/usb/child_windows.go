//go:build windows && (amd64 || arm64)

package usb

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// errNoWinUSBChild is returned by findInterfaceZeroChild when no Interface 0
// (&MI_00) child function node exists for the target VID/PID — i.e. the device
// is not a composite device, or its child is not present. Callers treat it as
// "fall back to the parent-node behaviour".
var errNoWinUSBChild = errors.New("winusb: no Interface 0 child found")

// findInterfaceZeroChild locates the Interface 0 (&MI_00) child function node
// of the composite USB device VID:PID and reports the driver bound to it
// (childService) and, when that driver is WinUSB, a CreateFile-openable
// device-interface path for it (childPath).
//
// Why this exists: winEnumerator.List/Open and winDriverInspector.Inspect only
// ever walk the GUID_DEVINTERFACE_USB_DEVICE interface, which a composite
// device registers on its usbccgp PARENT node. The SDR's real driver (WinUSB,
// after Zadig) lives on the &MI_00 CHILD node, which registers its own
// per-install device-interface GUID instead of GUID_DEVINTERFACE_USB_DEVICE —
// so the parent walk can never reach it. We walk the raw USB device-node tree
// here, match VID/PID + &MI_00, read the child's SPDRP_SERVICE, and (for
// WinUSB) resolve its symbolic-link path.
//
// childPath is non-empty only when childService is WinUSB. When an &MI_00 child
// exists but is bound to something else (libusbK, the in-box DVB driver, ...)
// we still return childService (with an empty path) so the doctor can give an
// accurate hint. Returns errNoWinUSBChild when no &MI_00 child for VID:PID is
// present, or when children exist but all provably belong to OTHER dongles.
//
// serial disambiguates identical composite dongles (issue #1131): the serial
// lives on the composite PARENT node, not the child instance ID, so each
// candidate child is linked to its parent via DEVPKEY_Device_Parent and
// pickInterfaceZeroChild matches the parent's serial against the caller's.
// Without that, two identical dongles both resolved to the FIRST &MI_00 child:
// the daemon held dongle A open, Open(serial=B) re-resolved to A's child, and
// WinUsb_Initialize failed with ERROR_NOT_ENOUGH_MEMORY — while sequential
// `sdr list --probe` opens never collided. serial may be "" (first match wins,
// the old behaviour).
func findInterfaceZeroChild(vid, pid uint16, serial string) (childService, childPath string, err error) {
	devSet, err := windows.SetupDiGetClassDevsEx(nil, "USB", 0, windows.DIGCF_PRESENT|windows.DIGCF_ALLCLASSES, 0, "")
	if err != nil {
		return "", "", fmt.Errorf("winusb: SetupDiGetClassDevsEx(USB): %w", err)
	}
	defer windows.SetupDiDestroyDeviceInfoList(devSet)

	var cands []compositeChildCandidate
	var infos []*windows.DevInfoData
	for i := 0; ; i++ {
		devInfo, e := windows.SetupDiEnumDeviceInfo(devSet, i)
		if e != nil {
			break // ERROR_NO_MORE_ITEMS (or any walk error) ends the scan
		}
		instID, e := windows.SetupDiGetDeviceInstanceId(devSet, devInfo)
		if e != nil {
			continue
		}
		v, p, _, hasMI := parseInstanceID(instID)
		if !hasMI || v != vid || p != pid || !isInterfaceZero(instID) {
			continue
		}
		cands = append(cands, compositeChildCandidate{
			InstanceID:       instID,
			ParentInstanceID: parentInstanceID(devSet, devInfo),
		})
		infos = append(infos, devInfo)
	}
	idx := pickInterfaceZeroChild(cands, serial)
	if idx < 0 {
		return "", "", errNoWinUSBChild
	}
	devInfo := infos[idx]
	// Read the matched child's bound kernel service.
	svc := ""
	if val, e := windows.SetupDiGetDeviceRegistryProperty(devSet, devInfo, windows.SPDRP_SERVICE); e == nil {
		svc, _ = val.(string)
	}
	if !strings.EqualFold(svc, "winusb") {
		// Child exists but isn't WinUSB-bound: report the service (no
		// openable path) so the doctor hint is accurate.
		return svc, "", nil
	}
	path, e := childInterfacePath(devSet, devInfo, cands[idx].InstanceID)
	if e != nil {
		return svc, "", e
	}
	return svc, path, nil
}

// devpkeyDeviceParent is DEVPKEY_Device_Parent from the Windows SDK's
// devpkey.h — the instance ID of a devnode's parent. x/sys/windows wraps
// SetupDiGetDeviceProperty but ships no predefined DEVPROPKEYs, so the one
// key needed here is defined locally.
var devpkeyDeviceParent = windows.DEVPROPKEY{
	FmtID: windows.DEVPROPGUID{
		Data1: 0x4340a6c5, Data2: 0x93fa, Data3: 0x4706,
		Data4: [8]byte{0x97, 0x2c, 0x7b, 0x64, 0x80, 0x08, 0xa5, 0xa7},
	},
	PID: 8,
}

// parentInstanceID resolves the instance ID of a devnode's parent (for an
// &MI_00 child function node: the composite parent, whose last path component
// is the dongle serial). Best-effort — returns "" on any failure and
// pickInterfaceZeroChild's no-information fallback keeps the old behaviour.
func parentInstanceID(devSet windows.DevInfo, devInfo *windows.DevInfoData) string {
	val, err := windows.SetupDiGetDeviceProperty(devSet, devInfo, &devpkeyDeviceParent)
	if err != nil {
		return ""
	}
	s, _ := val.(string)
	return s
}

// childInterfacePath resolves the CreateFile-openable device-interface path for
// a WinUSB-bound composite child. Zadig/libwdi records the per-install
// interface GUID in the child's "Device Parameters\DeviceInterfaceGUIDs"
// (REG_MULTI_SZ); we read it, then ask CfgMgr for that GUID's live symbolic
// links scoped to this device instance.
func childInterfacePath(devSet windows.DevInfo, devInfo *windows.DevInfoData, instID string) (string, error) {
	hkey, err := windows.SetupDiOpenDevRegKey(devSet, devInfo, windows.DICS_FLAG_GLOBAL, 0, windows.DIREG_DEV, windows.KEY_READ)
	if err != nil {
		return "", fmt.Errorf("winusb: open child Device Parameters: %w", err)
	}
	key := registry.Key(hkey)
	defer key.Close()

	guids, _, err := key.GetStringsValue("DeviceInterfaceGUIDs")
	if err != nil {
		return "", fmt.Errorf("winusb: read DeviceInterfaceGUIDs: %w", err)
	}
	guidStr, ok := firstInterfaceGUID(guids)
	if !ok {
		return "", errors.New("winusb: child node has no DeviceInterfaceGUIDs")
	}
	guid, err := windows.GUIDFromString(guidStr)
	if err != nil {
		return "", fmt.Errorf("winusb: bad interface GUID %q: %w", guidStr, err)
	}
	paths, err := windows.CM_Get_Device_Interface_List(instID, &guid, windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT)
	if err != nil {
		return "", fmt.Errorf("winusb: CM_Get_Device_Interface_List: %w", err)
	}
	for _, p := range paths {
		if strings.TrimSpace(p) != "" {
			return p, nil
		}
	}
	return "", errors.New("winusb: no live device-interface path for child")
}

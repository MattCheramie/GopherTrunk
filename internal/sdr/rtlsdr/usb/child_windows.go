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
// present.
//
// NOTE: matching is by VID/PID + &MI_00 only; the dongle serial lives on the
// parent, not the child instance ID. Two *identical* composite dongles would
// be indistinguishable here — an acceptable limitation today (the pool already
// disambiguates by serial at the List layer, and the single-interface path is
// unaffected).
func findInterfaceZeroChild(vid, pid uint16) (childService, childPath string, err error) {
	devSet, err := windows.SetupDiGetClassDevsEx(nil, "USB", 0, windows.DIGCF_PRESENT|windows.DIGCF_ALLCLASSES, 0, "")
	if err != nil {
		return "", "", fmt.Errorf("winusb: SetupDiGetClassDevsEx(USB): %w", err)
	}
	defer windows.SetupDiDestroyDeviceInfoList(devSet)

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
		// Matched the Interface 0 child. Read its bound kernel service.
		svc := ""
		if val, e := windows.SetupDiGetDeviceRegistryProperty(devSet, devInfo, windows.SPDRP_SERVICE); e == nil {
			svc, _ = val.(string)
		}
		if !strings.EqualFold(svc, "winusb") {
			// Child exists but isn't WinUSB-bound: report the service (no
			// openable path) so the doctor hint is accurate.
			return svc, "", nil
		}
		path, e := childInterfacePath(devSet, devInfo, instID)
		if e != nil {
			return svc, "", e
		}
		return svc, path, nil
	}
	return "", "", errNoWinUSBChild
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

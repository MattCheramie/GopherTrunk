//go:build windows && (amd64 || arm64)

package usb

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Lazy-loaded Win32 entry points. setupapi.dll handles device-info
// enumeration; winusb.dll wraps the bound WinUSB kernel-mode driver.
// Loading is deferred so this package still compiles + imports cleanly
// on Wine / older Windows installs missing the DLLs — callers see a
// runtime error from the first proc call instead of a load-time panic.
var (
	modSetupAPI = windows.NewLazySystemDLL("setupapi.dll")
	modWinUSB   = windows.NewLazySystemDLL("winusb.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procSetupDiGetClassDevsW             = modSetupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = modSetupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = modSetupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = modSetupAPI.NewProc("SetupDiDestroyDeviceInfoList")

	procWinUsbInitialize             = modWinUSB.NewProc("WinUsb_Initialize")
	procWinUsbFree                   = modWinUSB.NewProc("WinUsb_Free")
	procWinUsbGetAssociatedInterface = modWinUSB.NewProc("WinUsb_GetAssociatedInterface")
	procWinUsbControlTransfer        = modWinUSB.NewProc("WinUsb_ControlTransfer")
	procWinUsbGetDescriptor          = modWinUSB.NewProc("WinUsb_GetDescriptor")
	procWinUsbReadPipe               = modWinUSB.NewProc("WinUsb_ReadPipe")
	procWinUsbAbortPipe              = modWinUSB.NewProc("WinUsb_AbortPipe")
	procWinUsbResetPipe              = modWinUSB.NewProc("WinUsb_ResetPipe")
	procWinUsbSetPipePolicy          = modWinUSB.NewProc("WinUsb_SetPipePolicy")
	procWinUsbGetOverlappedResult    = modWinUSB.NewProc("WinUsb_GetOverlappedResult")
)

// GUID_DEVINTERFACE_USB_DEVICE — the class GUID every USB device exposes
// via the usbhub stack, regardless of which function driver is bound.
// Lets us enumerate every USB device on the system, then probe each one
// for WinUSB binding via [winTransport.tryInitialize].
var guidDevInterfaceUSBDevice = windows.GUID{
	Data1: 0xA5DCBF10,
	Data2: 0x6530,
	Data3: 0x11D2,
	Data4: [8]byte{0x90, 0x1F, 0x00, 0xC0, 0x4F, 0xB9, 0x51, 0xED},
}

// WinUSB pipe-policy IDs (see WinUsb_SetPipePolicy).
const (
	policyPipeTransferTimeout = 0x03
	policyRawIO               = 0x07
	policyAllowPartialReads   = 0x05
	policyAutoClearStall      = 0x02
)

// DIGCF_PRESENT | DIGCF_DEVICEINTERFACE — only currently-attached devices.
const digcfPresentDeviceInterface = windows.DIGCF_PRESENT | windows.DIGCF_DEVICEINTERFACE

// spDeviceInterfaceData mirrors SP_DEVICE_INTERFACE_DATA. The Size field
// must be the size of this struct (32 bytes on x64).
type spDeviceInterfaceData struct {
	Size               uint32
	InterfaceClassGuid windows.GUID
	Flags              uint32
	Reserved           uintptr
}

// On 64-bit Windows the SP_DEVICE_INTERFACE_DETAIL_DATA_W header is 8
// bytes (DWORD cbSize + padding). The build tag at the top of this file
// restricts compilation to amd64 / arm64 so this constant is safe to
// hard-code.
const spDeviceInterfaceDetailDataHeaderSize = 8

const winusbIfaceGUIDEnv = "RTLSDR_WINUSB_IFACE_GUID"

func platformEnumerator() Enumerator { return &winEnumerator{} }

// winEnumerator walks every present USB device interface via SetupAPI
// and, on Open, hands ownership to the bound WinUSB driver. Devices that
// aren't WinUSB-bound (the default for an out-of-the-box RTL-SDR before
// Zadig) cause Open to fail with an explicit "no WinUSB driver" error.
type winEnumerator struct{}

func (w *winEnumerator) Name() string { return "winusb" }

func (w *winEnumerator) List(vid, pid uint16) ([]Descriptor, error) {
	ifaceGUID, guidLabel, err := winInterfaceClassGUID()
	if err != nil {
		return nil, err
	}
	debugLogf("winusb", "List start vid=0x%04x pid=0x%04x", vid, pid)
	debugLogf("winusb", "List interface class guid=%s", guidLabel)
	devSet, _, errno := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&ifaceGUID)),
		0,
		0,
		uintptr(digcfPresentDeviceInterface),
	)
	if devSet == uintptr(windows.InvalidHandle) || devSet == 0 {
		return nil, fmt.Errorf("winusb: SetupDiGetClassDevs: %w", winErr(errno))
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devSet)

	var out []Descriptor
	for memberIndex := uint32(0); ; memberIndex++ {
		var iface spDeviceInterfaceData
		iface.Size = uint32(unsafe.Sizeof(iface))
		ret, _, errno := procSetupDiEnumDeviceInterfaces.Call(
			devSet,
			0,
			uintptr(unsafe.Pointer(&ifaceGUID)),
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
			debugLogf("winusb", "List[%d] interface path error: %v", memberIndex, err)
			continue
		}
		v, p, serial := parseDevicePath(path)
		debugLogf("winusb", "List[%d] path=%q parsed vid=0x%04x pid=0x%04x serial=%q", memberIndex, path, v, p, serial)
		if v == 0 && p == 0 {
			continue
		}
		if vid != 0 && v != vid {
			debugLogf("winusb", "List[%d] skip: vid mismatch (want 0x%04x)", memberIndex, vid)
			continue
		}
		if pid != 0 && p != pid {
			debugLogf("winusb", "List[%d] skip: pid mismatch (want 0x%04x)", memberIndex, pid)
			continue
		}
		out = append(out, Descriptor{
			Bus:     0, // Windows doesn't expose libusb-style bus/address;
			Address: 0, // serial number is the disambiguator.
			VID:     v,
			PID:     p,
			Serial:  serial,
			Path:    path,
		})
	}
	debugLogf("winusb", "List done matches=%d", len(out))
	return out, nil
}

// errWinUSBCreateFile tags failures from the CreateFile stage of
// createAndInitWinUSB (as opposed to the WinUsb_Initialize stage). Open keys on
// it to keep CreateFile-level failures — most importantly ERROR_ACCESS_DENIED
// when another process already holds the dongle (issue #333) — surfacing
// verbatim, instead of misattributing them to a driver-binding problem and
// uselessly triggering the composite-child fallback.
var errWinUSBCreateFile = errors.New("winusb: CreateFile")

// createAndInitWinUSB opens a device-interface path and binds it to WinUSB,
// returning the file + WinUSB interface handles on success. A CreateFile-stage
// failure is wrapped with errWinUSBCreateFile; a WinUsb_Initialize-stage
// failure returns winErr(errno) directly, so Open can tell the two apart.
func createAndInitWinUSB(path string) (windows.Handle, uintptr, error) {
	wpath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, fmt.Errorf("%w %q: %w", errWinUSBCreateFile, path, err)
	}
	flags := winCreateFileFlags()
	handle, err := windows.CreateFile(
		wpath,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		flags,
		0,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("%w %q: %w", errWinUSBCreateFile, path, err)
	}
	debugLogf("winusb", "Open CreateFile ok handle=0x%x flags=0x%x", uintptr(handle), flags)
	var ifaceHandle uintptr
	ret, _, errno := procWinUsbInitialize.Call(uintptr(handle), uintptr(unsafe.Pointer(&ifaceHandle)))
	if ret == 0 {
		windows.CloseHandle(handle)
		return 0, 0, winErr(errno)
	}
	// Match libusb's WinUSB backend: explicitly disable the per-pipe
	// transfer timeout on the control endpoint at open time
	// (winusbx_configure_endpoints in libusb/os/windows_winusb.c sets
	// PIPE_TRANSFER_TIMEOUT=0 on pipe 0 as part of claim_interface).
	// WinUSB's documented default is 5000 ms; without this call, the
	// brief window between open and the first ControlOut would inherit
	// that default. The lazy applyControlTimeout still overrides this
	// to CtrlTimeoutMs on the first user-level transfer; this call is
	// parity/hardening, not the issue #395 fix.
	setControlPipeTimeout(ifaceHandle, 0)
	return handle, ifaceHandle, nil
}

func (w *winEnumerator) Open(d Descriptor) (Transport, error) {
	if d.Path == "" {
		return nil, errors.New("winusb: Descriptor.Path empty (re-enumerate)")
	}
	debugLogf("winusb", "Open path=%q serial=%q vid=0x%04x pid=0x%04x", d.Path, d.Serial, d.VID, d.PID)
	handle, ifaceHandle, err := createAndInitWinUSB(d.Path)
	if err == nil {
		debugLogf("winusb", "Open WinUsb_Initialize ok iface=0x%x", ifaceHandle)
		return MaybeWrapDebug(&winTransport{fileHandle: handle, ifaceHandle: ifaceHandle, desc: d}, d), nil
	}
	// A CreateFile-stage failure (e.g. ERROR_ACCESS_DENIED — another process
	// holds the dongle, issue #333) is not a driver-binding problem; surface it
	// verbatim so openUSBHint / isAccessDenied upstream still fire.
	if errors.Is(err, errWinUSBCreateFile) {
		return nil, err
	}
	// WinUsb_Initialize failed. The GUID_DEVINTERFACE_USB_DEVICE node we
	// enumerated may be a USB composite parent (bound to usbccgp), whose
	// WinUSB-capable SDR lives on the Interface 0 (&MI_00) child function node.
	// Find and open that child before surfacing the bind failure — this lets a
	// correctly-bound RTL-SDR V4 (and other composite dongles) open without the
	// user having to chase the right entry in Zadig. Store the child path in the
	// returned transport's descriptor so Reset() re-opens the child, not the
	// parent.
	if _, childPath, derr := findInterfaceZeroChild(d.VID, d.PID); derr == nil && childPath != "" {
		if ch, cif, cerr := createAndInitWinUSB(childPath); cerr == nil {
			cd := d
			cd.Path = childPath
			debugLogf("winusb", "Open child WinUsb_Initialize ok iface=0x%x", cif)
			return MaybeWrapDebug(&winTransport{fileHandle: ch, ifaceHandle: cif, desc: cd}, cd), nil
		}
	}
	svc, descr := lookupBoundDriver(d.Path)
	return nil, fmt.Errorf(
		"winusb: WinUsb_Initialize failed for VID_%04X&PID_%04X (current driver: %s%s — expected WinUSB; run Zadig and rebind Interface 0, see `gophertrunk sdr doctor`): %w",
		d.VID, d.PID, fallbackString(svc, "unknown"), parens(descr), err)
}

func winCreateFileFlags() uint32 {
	flags := uint32(windows.FILE_ATTRIBUTE_NORMAL | windows.FILE_FLAG_OVERLAPPED)
	if os.Getenv("RTLSDR_WINUSB_NO_OVERLAPPED") != "" {
		flags = windows.FILE_ATTRIBUTE_NORMAL
	}
	return flags
}

func winInterfaceClassGUID() (windows.GUID, string, error) {
	raw := strings.TrimSpace(os.Getenv(winusbIfaceGUIDEnv))
	if raw == "" {
		return guidDevInterfaceUSBDevice, guidToString(guidDevInterfaceUSBDevice), nil
	}
	g, err := parseGUID(raw)
	if err != nil {
		return windows.GUID{}, "", fmt.Errorf("winusb: invalid %s=%q: %w", winusbIfaceGUIDEnv, raw, err)
	}
	return g, guidToString(g), nil
}

func parseGUID(s string) (windows.GUID, error) {
	var (
		d1     uint32
		d2, d3 uint16
		d4     [8]uint8
	)
	formats := []string{
		"{%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x}",
		"%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
	}
	for _, f := range formats {
		n, err := fmt.Sscanf(s, f,
			&d1, &d2, &d3,
			&d4[0], &d4[1], &d4[2], &d4[3], &d4[4], &d4[5], &d4[6], &d4[7],
		)
		if err == nil && n == 11 {
			return windows.GUID{Data1: d1, Data2: d2, Data3: d3, Data4: d4}, nil
		}
	}
	return windows.GUID{}, errors.New("expected GUID form {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}")
}

func guidToString(g windows.GUID) string {
	return fmt.Sprintf("{%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x}",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7],
	)
}

// setControlPipeTimeout pushes a PIPE_TRANSFER_TIMEOUT policy on the
// default control endpoint (pipe ID 0). Splitting this out lets both
// Open (libusb-parity seed) and applyControlTimeout (mid-stream
// override) share the same syscall shape without either reaching
// into the other's locking. Errors are ignored — this is best-effort
// hardening, and the next ControlOut would fail loudly anyway if the
// handle were already invalid.
func setControlPipeTimeout(ifaceHandle uintptr, timeoutMs uint32) {
	procWinUsbSetPipePolicy.Call(
		ifaceHandle,
		0,
		policyPipeTransferTimeout,
		uintptr(unsafe.Sizeof(timeoutMs)),
		uintptr(unsafe.Pointer(&timeoutMs)),
	)
}

// winTransport is the WinUSB-backed [Transport].
type winTransport struct {
	fileHandle  windows.Handle
	ifaceHandle uintptr
	desc        Descriptor
	closed      atomic.Bool

	assocMu      sync.Mutex
	assocHandles map[uint8]uintptr

	bulkMu       sync.Mutex
	bulkActive   bool
	bulkEpAddr   byte
	bulkSlots    []*winBulkSlot
	bulkEvents   []windows.Handle
	bulkStopFlag atomic.Int32
	bulkDone     chan struct{}

	timeoutMu      sync.Mutex
	lastCtlTimeout uint32
}

type winBulkSlot struct {
	buf        []byte
	overlapped windows.Overlapped
	event      windows.Handle
}

// winusbSetupPacket is the 8-byte setup packet WinUsb_ControlTransfer
// consumes — identical wire format to the USB 2.0 setup stage.
type winusbSetupPacket struct {
	RequestType uint8
	Request     uint8
	Value       uint16
	Index       uint16
	Length      uint16
}

// Some WinUSB-bound devices expose vendor control requests on interface
// recipient scope (bmRequestType ...|RECIPIENT_INTERFACE) rather than
// device scope. Most devices (including RTL-SDR) accept recipient-device,
// so we try device first and fall back to interface only when the first
// transfer reports ErrDeviceGone.
const interfaceRecipientBit uint8 = 0x01

// pack folds the setup packet into a single uintptr. This is load-bearing
// and the reason vendor control transfers never worked on real Windows
// hardware before: the WinUsb_ControlTransfer prototype takes the
// WINUSB_SETUP_PACKET **by value**, and the x64/arm64 calling convention
// passes an 8-byte struct in one integer register. Go's
// syscall.proc.Call(...uintptr) marshals each argument as exactly one
// register, so the setup packet must arrive as the 8 bytes packed into a
// uintptr — NOT as a pointer to the struct. Passing a pointer made WinUSB
// interpret the pointer's low bytes as bmRequestType/bRequest/wValue/...,
// i.e. a garbage vendor request the device timed out on or rejected with
// ERROR_GEN_FAILURE, while standard descriptor requests (which go through
// WinUsb_GetDescriptor, a different prototype) kept working — exactly the
// split the field diagnostics showed. The byte order below matches the
// struct's little-endian in-memory layout on every supported target
// (amd64, arm64), so it is identical to what a by-value struct push would
// place in the register.
func (p winusbSetupPacket) pack() uintptr {
	return uintptr(uint64(p.RequestType) |
		uint64(p.Request)<<8 |
		uint64(p.Value)<<16 |
		uint64(p.Index)<<32 |
		uint64(p.Length)<<48)
}

func (t *winTransport) applyControlTimeout(timeoutMs int) {
	if timeoutMs <= 0 {
		return
	}
	v := uint32(timeoutMs)
	t.timeoutMu.Lock()
	if t.lastCtlTimeout == v {
		t.timeoutMu.Unlock()
		return
	}
	t.lastCtlTimeout = v
	t.timeoutMu.Unlock()
	setControlPipeTimeout(t.ifaceHandle, v)
}

func (t *winTransport) ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error) {
	if t.closed.Load() {
		return nil, ErrClosed
	}
	if n < 0 || n > 0xFFFF {
		return nil, fmt.Errorf("winusb: control IN length %d out of range", n)
	}
	t.applyControlTimeout(timeoutMs)
	out, err := t.controlInWithType(VendorIn, bRequest, wValue, wIndex, n)
	if err == nil {
		return out, nil
	}
	if !shouldRetryInterfaceRecipient(err) {
		return nil, err
	}
	debugLogf("winusb", "ControlIn retry with interface recipient req=0x%02x val=0x%04x idx=0x%04x", bRequest, wValue, wIndex)
	out, err = t.controlInWithType(VendorIn|interfaceRecipientBit, bRequest, wValue, wIndex, n)
	if err != nil {
		debugLogf("winusb", "ControlIn interface-recipient retry failed: %v", err)
		out, assocErr := t.controlInAssociatedFallback(bRequest, wValue, wIndex, n)
		if assocErr == nil {
			return out, nil
		}
	}
	return out, err
}

func (t *winTransport) controlInWithType(reqType uint8, bRequest uint8, wValue, wIndex uint16, n int) ([]byte, error) {
	return t.controlInWithTypeOnHandle(t.ifaceHandle, reqType, bRequest, wValue, wIndex, n)
}

func (t *winTransport) controlInWithTypeOnHandle(handle uintptr, reqType uint8, bRequest uint8, wValue, wIndex uint16, n int) ([]byte, error) {
	pkt := winusbSetupPacket{
		RequestType: reqType,
		Request:     bRequest,
		Value:       wValue,
		Index:       wIndex,
		Length:      uint16(n),
	}
	var buf []byte
	var bufPtr uintptr
	if n > 0 {
		buf = make([]byte, n)
		bufPtr = uintptr(unsafe.Pointer(&buf[0]))
	}
	var transferred uint32
	ret, _, errno := procWinUsbControlTransfer.Call(
		handle,
		pkt.pack(), // WINUSB_SETUP_PACKET is passed BY VALUE (see pack)
		bufPtr,
		uintptr(n),
		uintptr(unsafe.Pointer(&transferred)),
		0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("winusb: WinUsb_ControlTransfer IN: %w", winErr(errno))
	}
	return buf[:transferred], nil
}

func (t *winTransport) controlInAssociatedFallback(bRequest uint8, wValue, wIndex uint16, n int) ([]byte, error) {
	h, ok, err := t.associatedInterfaceHandle(0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("winusb: no associated interface available")
	}
	debugLogf("winusb", "ControlIn retry with associated interface[0] req=0x%02x val=0x%04x idx=0x%04x", bRequest, wValue, wIndex)
	out, err := t.controlInWithTypeOnHandle(h, VendorIn, bRequest, wValue, wIndex, n)
	if err == nil {
		return out, nil
	}
	out, err = t.controlInWithTypeOnHandle(h, VendorIn|interfaceRecipientBit, bRequest, wValue, wIndex, n)
	if err != nil {
		debugLogf("winusb", "ControlIn associated interface retry failed: %v", err)
	}
	return out, err
}

func (t *winTransport) ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if len(data) > 0xFFFF {
		return fmt.Errorf("winusb: control OUT length %d out of range", len(data))
	}
	t.applyControlTimeout(timeoutMs)
	err := t.controlOutWithType(VendorOut, bRequest, wValue, wIndex, data)
	if err == nil {
		return nil
	}
	if !shouldRetryInterfaceRecipient(err) {
		return err
	}
	debugLogf("winusb", "ControlOut retry with interface recipient req=0x%02x val=0x%04x idx=0x%04x len=%d", bRequest, wValue, wIndex, len(data))
	err = t.controlOutWithType(VendorOut|interfaceRecipientBit, bRequest, wValue, wIndex, data)
	if err != nil {
		debugLogf("winusb", "ControlOut interface-recipient retry failed: %v", err)
		assocErr := t.controlOutAssociatedFallback(bRequest, wValue, wIndex, data)
		if assocErr == nil {
			return nil
		}
	}
	return err
}

func shouldRetryInterfaceRecipient(err error) bool {
	if errors.Is(err, ErrDeviceGone) {
		return true
	}
	// Some WinUSB stacks surface recipient mismatches as ERROR_GEN_FAILURE
	// rather than a disconnect-like code.
	return strings.Contains(err.Error(), "ERROR_GEN_FAILURE")
}

func (t *winTransport) controlOutWithType(reqType uint8, bRequest uint8, wValue, wIndex uint16, data []byte) error {
	return t.controlOutWithTypeOnHandle(t.ifaceHandle, reqType, bRequest, wValue, wIndex, data)
}

func (t *winTransport) controlOutWithTypeOnHandle(handle uintptr, reqType uint8, bRequest uint8, wValue, wIndex uint16, data []byte) error {
	pkt := winusbSetupPacket{
		RequestType: reqType,
		Request:     bRequest,
		Value:       wValue,
		Index:       wIndex,
		Length:      uint16(len(data)),
	}
	var dataPtr uintptr
	if len(data) > 0 {
		dataPtr = uintptr(unsafe.Pointer(&data[0]))
	}
	var transferred uint32
	ret, _, errno := procWinUsbControlTransfer.Call(
		handle,
		pkt.pack(), // WINUSB_SETUP_PACKET is passed BY VALUE (see pack)
		dataPtr,
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&transferred)),
		0,
	)
	if ret == 0 {
		// A vendor write that stalls the control pipe (ERROR_GEN_FAILURE,
		// the clone-dongle cold-boot symptom in issue #395) leaves the
		// default control endpoint halted. Per the USB spec the next
		// SETUP should clear a control-endpoint halt automatically, but
		// some clone RTL2832U firmwares need an explicit CLEAR_FEATURE
		// before they accept the retried write — which is exactly what
		// libusb's WinUSB backend does on a control stall. Clear pipe 0
		// and retry the transfer once before surfacing the error. The
		// bring-up path's sacrificial warmup relies on this so the
		// byte-identical InitBaseband step 0 isn't poisoned by the
		// warmup's stall. Errors from the reset are ignored — the retry
		// (or the returned error) is the real signal.
		if errors.Is(winErr(errno), ErrPipeStalled) {
			procWinUsbResetPipe.Call(t.ifaceHandle, 0)
			ret, _, errno = procWinUsbControlTransfer.Call(
				t.ifaceHandle,
				pkt.pack(), // WINUSB_SETUP_PACKET is passed BY VALUE (see pack)
				dataPtr,
				uintptr(len(data)),
				uintptr(unsafe.Pointer(&transferred)),
				0,
			)
		}
		if ret == 0 {
			return fmt.Errorf("winusb: WinUsb_ControlTransfer OUT: %w", winErr(errno))
		}
	}
	return nil
}

func (t *winTransport) controlOutAssociatedFallback(bRequest uint8, wValue, wIndex uint16, data []byte) error {
	h, ok, err := t.associatedInterfaceHandle(0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("winusb: no associated interface available")
	}
	debugLogf("winusb", "ControlOut retry with associated interface[0] req=0x%02x val=0x%04x idx=0x%04x len=%d", bRequest, wValue, wIndex, len(data))
	err = t.controlOutWithTypeOnHandle(h, VendorOut, bRequest, wValue, wIndex, data)
	if err == nil {
		return nil
	}
	err = t.controlOutWithTypeOnHandle(h, VendorOut|interfaceRecipientBit, bRequest, wValue, wIndex, data)
	if err != nil {
		debugLogf("winusb", "ControlOut associated interface retry failed: %v", err)
	}
	return err
}

func (t *winTransport) associatedInterfaceHandle(index uint8) (uintptr, bool, error) {
	t.assocMu.Lock()
	defer t.assocMu.Unlock()
	if t.assocHandles != nil {
		if h, ok := t.assocHandles[index]; ok {
			return h, true, nil
		}
	}
	var h uintptr
	ret, _, errno := procWinUsbGetAssociatedInterface.Call(
		t.ifaceHandle,
		uintptr(index),
		uintptr(unsafe.Pointer(&h)),
	)
	if ret == 0 {
		if errno == windows.ERROR_NO_MORE_ITEMS {
			debugLogf("winusb", "WinUsb_GetAssociatedInterface[%d]: no more items", index)
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("winusb: WinUsb_GetAssociatedInterface[%d]: %w", index, winErr(errno))
	}
	if t.assocHandles == nil {
		t.assocHandles = make(map[uint8]uintptr)
	}
	t.assocHandles[index] = h
	debugLogf("winusb", "Associated interface[%d] acquired handle=0x%x", index, h)
	return h, true, nil
}

// ClaimInterface is a no-op on Windows: WinUsb_Initialize already gave
// us exclusive access to interface 0. Multi-interface WinUSB devices
// require WinUsb_GetAssociatedInterface to access interfaces > 0, but
// the RTL-SDR exposes only interface 0 — we'd surface that as an error
// rather than silently succeed if num != 0.
func (t *winTransport) ClaimInterface(num int) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if num != 0 {
		return fmt.Errorf("winusb: only interface 0 supported (got %d)", num)
	}
	return nil
}

func (t *winTransport) ReleaseInterface(int) error { return nil }

// Reset mirrors what libusb_reset_device does on Windows: clear-halt
// the control endpoint, then drop the WinUSB handles and re-bind via
// CreateFile + WinUsb_Initialize. The plain clear-halt that earlier
// shipped here (WinUsb_ResetPipe(0)) is enough for a stale endpoint
// halt but does nothing for a wedged firmware state left by a prior
// crashed process — for that you need a fresh device-object handle.
//
// Invariants:
//   - The bring-up retry envelope in purego/driver.go is the only
//     caller, and it runs Reset before any bulk-in stream starts.
//     The bulkMu acquire below is defensive in case a future caller
//     reuses Reset() after streaming begins.
//   - On failure mid-sequence the transport is left in a coherent
//     "closed" state so subsequent ControlOut/ControlIn calls return
//     ErrClosed rather than dereferencing a freed handle.
func (t *winTransport) Reset() error {
	if t.closed.Load() {
		return ErrClosed
	}
	// Best-effort clear-halt before we drop the handle: clears any
	// stalled control transfer so the device firmware sees the
	// CLEAR_FEATURE before the bus re-enumerates us. Errors are
	// ignored — we're about to throw the handle away anyway.
	procWinUsbResetPipe.Call(t.ifaceHandle, 0)

	t.bulkMu.Lock()
	defer t.bulkMu.Unlock()
	oldIface := t.ifaceHandle
	oldFile := t.fileHandle
	t.ifaceHandle = 0
	t.fileHandle = 0
	if oldIface != 0 {
		procWinUsbFree.Call(oldIface)
	}
	if oldFile != 0 {
		windows.CloseHandle(oldFile)
	}

	// Let the WinUSB stack release the file-object and the device
	// firmware finish internal recovery before we re-open. libusb-1.0
	// uses 50ms here for clean closes, but the bring-up envelope only
	// calls Reset() after a wedged-firmware failure (ERROR_GEN_FAILURE
	// / pipe stall) — issue #395 surfaced a Windows clone dongle whose
	// firmware needed longer than 50ms to recover, so we settle for
	// 150ms here. Healthy opens never run this path; the cost only
	// applies to dongles that actually need recovery.
	time.Sleep(150 * time.Millisecond)

	wpath, err := windows.UTF16PtrFromString(t.desc.Path)
	if err != nil {
		t.closed.Store(true)
		return fmt.Errorf("winusb: reset bad path %q: %w", t.desc.Path, err)
	}
	handle, err := windows.CreateFile(
		wpath,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OVERLAPPED,
		0,
	)
	if err != nil {
		t.closed.Store(true)
		return fmt.Errorf("winusb: reset CreateFile %q: %w", t.desc.Path, err)
	}
	var ifaceHandle uintptr
	ret, _, errno := procWinUsbInitialize.Call(uintptr(handle), uintptr(unsafe.Pointer(&ifaceHandle)))
	if ret == 0 {
		windows.CloseHandle(handle)
		t.closed.Store(true)
		return fmt.Errorf("winusb: reset WinUsb_Initialize: %w", winErr(errno))
	}
	t.fileHandle = handle
	t.ifaceHandle = ifaceHandle
	// The new ifaceHandle starts with WinUSB defaults — re-seed the
	// libusb-parity zero timeout on the control endpoint, then force
	// the next applyControlTimeout to re-push our real timeout.
	setControlPipeTimeout(ifaceHandle, 0)
	t.timeoutMu.Lock()
	t.lastCtlTimeout = 0
	t.timeoutMu.Unlock()
	return nil
}

func (t *winTransport) StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte), onStreamDead func(error)) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if ringBufs <= 0 || bufLen <= 0 {
		return fmt.Errorf("winusb: invalid bulk ring geometry (bufs=%d len=%d)", ringBufs, bufLen)
	}
	if ringBufs > 64 {
		// WaitForMultipleObjects caps at MAXIMUM_WAIT_OBJECTS = 64.
		// The default of 32 is well below this — we only guard against
		// callers asking for absurd values.
		return fmt.Errorf("winusb: ringBufs %d exceeds WaitForMultipleObjects limit (64)", ringBufs)
	}
	t.bulkMu.Lock()
	defer t.bulkMu.Unlock()
	if t.bulkActive {
		return ErrBulkActive
	}

	// Enable RAW_IO for max throughput; the RTL2832U IQ stream is
	// always a multiple of maxpacketsize (512 B on full-speed bulk).
	one := uint8(1)
	procWinUsbSetPipePolicy.Call(
		t.ifaceHandle,
		uintptr(epAddr),
		policyRawIO,
		uintptr(unsafe.Sizeof(one)),
		uintptr(unsafe.Pointer(&one)),
	)
	procWinUsbSetPipePolicy.Call(
		t.ifaceHandle,
		uintptr(epAddr),
		policyAllowPartialReads,
		uintptr(unsafe.Sizeof(one)),
		uintptr(unsafe.Pointer(&one)),
	)
	// Disable per-pipe transfer timeout (we manage cancellation via AbortPipe).
	zero := uint32(0)
	procWinUsbSetPipePolicy.Call(
		t.ifaceHandle,
		uintptr(epAddr),
		policyPipeTransferTimeout,
		uintptr(unsafe.Sizeof(zero)),
		uintptr(unsafe.Pointer(&zero)),
	)

	slots := make([]*winBulkSlot, 0, ringBufs)
	events := make([]windows.Handle, 0, ringBufs)
	cleanup := func() {
		for _, s := range slots {
			procWinUsbAbortPipe.Call(t.ifaceHandle, uintptr(epAddr))
			_ = s
		}
		for _, ev := range events {
			windows.CloseHandle(ev)
		}
	}
	for i := 0; i < ringBufs; i++ {
		// Auto-reset event so WaitForMultipleObjects atomically clears
		// it on return — saves a ResetEvent call before re-issuing.
		ev, err := windows.CreateEvent(nil, 0 /*manualReset=false*/, 0, nil)
		if err != nil {
			cleanup()
			return fmt.Errorf("winusb: CreateEvent[%d]: %w", i, err)
		}
		s := &winBulkSlot{
			buf:   make([]byte, bufLen),
			event: ev,
		}
		s.overlapped.HEvent = ev
		if err := t.issueReadPipe(epAddr, s); err != nil {
			windows.CloseHandle(ev)
			cleanup()
			return fmt.Errorf("winusb: ReadPipe[%d]: %w", i, err)
		}
		slots = append(slots, s)
		events = append(events, ev)
	}

	t.bulkEpAddr = epAddr
	t.bulkSlots = slots
	t.bulkEvents = events
	t.bulkActive = true
	t.bulkStopFlag.Store(0)
	t.bulkDone = make(chan struct{})

	go t.reapLoop(onPacket, onStreamDead)
	return nil
}

// issueReadPipe arms one bulk-IN URB on the kernel side. The success path
// is ERROR_IO_PENDING (Go's CreateFile|FILE_FLAG_OVERLAPPED makes every
// pipe op async); the rare synchronous-completion path also returns
// success.
func (t *winTransport) issueReadPipe(epAddr byte, s *winBulkSlot) error {
	var transferred uint32
	ret, _, errno := procWinUsbReadPipe.Call(
		t.ifaceHandle,
		uintptr(epAddr),
		uintptr(unsafe.Pointer(&s.buf[0])),
		uintptr(len(s.buf)),
		uintptr(unsafe.Pointer(&transferred)),
		uintptr(unsafe.Pointer(&s.overlapped)),
	)
	if ret == 0 {
		if errno == windows.ERROR_IO_PENDING {
			return nil
		}
		return winErr(errno)
	}
	return nil
}

func (t *winTransport) reapLoop(onPacket func([]byte), onStreamDead func(error)) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(t.bulkDone)

	// firstErr stashes the first error that took a slot out of rotation.
	// When the loop exits with the stop flag unset (i.e. every slot died
	// of an unrecoverable error rather than being aborted by
	// StopBulkIn), we hand it to onStreamDead so the driver can surface
	// "stream died" to its IQ consumer.
	var firstErr error
	// Active mask — once a slot's I/O is taken to completion under stop
	// we mark it consumed; we exit when no slot is still in flight.
	consumed := make([]bool, len(t.bulkSlots))
	defer func() {
		if t.bulkStopFlag.Load() == 0 && onStreamDead != nil {
			if firstErr == nil {
				firstErr = ErrDeviceGone
			}
			// Dispatch from a fresh goroutine: onStreamDead typically
			// calls StopBulkIn (via the driver's cancel path) which
			// waits on `bulkDone` — which is only closed by this
			// reaper's defer chain (registered earlier, runs after).
			go onStreamDead(firstErr)
		}
	}()
	for {
		// Build a wait list of unfinished events.
		wait := make([]windows.Handle, 0, len(t.bulkSlots))
		idxMap := make([]int, 0, len(t.bulkSlots))
		for i, c := range consumed {
			if !c {
				wait = append(wait, t.bulkEvents[i])
				idxMap = append(idxMap, i)
			}
		}
		if len(wait) == 0 {
			return
		}
		ret, err := windows.WaitForMultipleObjects(wait, false, windows.INFINITE)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("winusb: WaitForMultipleObjects: %w", err)
			}
			return
		}
		raw := int(ret - windows.WAIT_OBJECT_0)
		if raw < 0 || raw >= len(wait) {
			if firstErr == nil {
				firstErr = fmt.Errorf("winusb: WaitForMultipleObjects returned %d outside slot range", ret)
			}
			return
		}
		slotIdx := idxMap[raw]
		slot := t.bulkSlots[slotIdx]
		var transferred uint32
		result, _, _ := procWinUsbGetOverlappedResult.Call(
			t.ifaceHandle,
			uintptr(unsafe.Pointer(&slot.overlapped)),
			uintptr(unsafe.Pointer(&transferred)),
			0, // bWait = FALSE
		)
		stop := t.bulkStopFlag.Load() != 0
		if stop {
			consumed[slotIdx] = true
			continue
		}
		if result != 0 && transferred > 0 {
			onPacket(slot.buf[:transferred])
		}
		if err := t.issueReadPipe(t.bulkEpAddr, slot); err != nil {
			// Slot is dead; mark consumed so we don't wait on its event.
			if firstErr == nil {
				firstErr = fmt.Errorf("winusb: ReadPipe resubmit: %w", err)
			}
			consumed[slotIdx] = true
		}
	}
}

func (t *winTransport) StopBulkIn() error {
	t.bulkMu.Lock()
	if !t.bulkActive {
		t.bulkMu.Unlock()
		return ErrBulkInactive
	}
	t.bulkStopFlag.Store(1)
	epAddr := t.bulkEpAddr
	events := t.bulkEvents
	done := t.bulkDone
	t.bulkActive = false
	t.bulkMu.Unlock()

	// AbortPipe completes every pending read with ERROR_OPERATION_ABORTED;
	// each event signals once, the reaper drains them and exits.
	procWinUsbAbortPipe.Call(t.ifaceHandle, uintptr(epAddr))
	<-done

	t.bulkMu.Lock()
	for _, ev := range events {
		windows.CloseHandle(ev)
	}
	t.bulkSlots = nil
	t.bulkEvents = nil
	t.bulkMu.Unlock()
	return nil
}

func (t *winTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	t.bulkMu.Lock()
	active := t.bulkActive
	t.bulkMu.Unlock()
	if active {
		t.closed.Store(false)
		_ = t.StopBulkIn()
		t.closed.Store(true)
	}
	if t.ifaceHandle != 0 {
		procWinUsbFree.Call(t.ifaceHandle)
		t.ifaceHandle = 0
	}
	t.assocMu.Lock()
	for idx, h := range t.assocHandles {
		procWinUsbFree.Call(h)
		delete(t.assocHandles, idx)
	}
	t.assocMu.Unlock()
	if t.fileHandle != 0 {
		windows.CloseHandle(t.fileHandle)
		t.fileHandle = 0
	}
	return nil
}

// ----------------------------------------------------------------------
// Diagnostics

// USB descriptor type codes (USB 2.0 §9.4) used by getDescriptor.
const (
	descTypeDevice = 0x01
	descTypeConfig = 0x02
)

// Diagnostics returns a multi-line, human-readable dump of everything we
// can learn about this device through WinUSB + SetupAPI: the actually
// bound driver (WinUSB vs libusbK vs the DVB driver vs none), the device
// path, the raw device + configuration descriptors (interfaces +
// endpoints), and a control-IN read probe. The bring-up path appends it
// to a failed Open so a single `gophertrunk sdr list --probe` run
// captures the data needed to triage a dongle that rejects control
// transfers even with WinUSB reportedly bound. Every step is
// best-effort: a failure on one line is reported inline and never aborts
// the rest. Implements the usb.Diagnoser optional interface.
func (t *winTransport) Diagnostics() string {
	var b strings.Builder
	fmt.Fprintf(&b, "device path: %s\n", t.desc.Path)
	fmt.Fprintf(&b, "matched VID:PID %04x:%04x serial=%q\n", t.desc.VID, t.desc.PID, t.desc.Serial)

	// The single most useful datum: which driver is *actually* bound.
	// WinUsb_Initialize can succeed against a libusbK-style filter while
	// the device still rejects raw vendor control writes.
	svc, descr := lookupBoundDriver(t.desc.Path)
	db := classifyWindows(t.desc, svc, descr)
	fmt.Fprintf(&b, "bound driver: service=%q desc=%q (winusb-bound=%t)\n", fallbackString(svc, "<none>"), descr, db.OK)
	if db.Hint != "" {
		fmt.Fprintf(&b, "  driver hint: %s\n", db.Hint)
	}

	if dev, err := t.getDescriptor(descTypeDevice, 0, 18); err != nil {
		fmt.Fprintf(&b, "device descriptor: unavailable (%v)\n", err)
	} else if len(dev) >= 18 {
		fmt.Fprintf(&b, "device descriptor: bcdUSB=%x.%02x class=%d sub=%d proto=%d maxPacket0=%d idVendor=%04x idProduct=%04x bcdDevice=%x.%02x iMfr=%d iProd=%d iSerial=%d numCfg=%d\n",
			dev[3], dev[2], dev[4], dev[5], dev[6], dev[7],
			uint16(dev[8])|uint16(dev[9])<<8, uint16(dev[10])|uint16(dev[11])<<8,
			dev[13], dev[12], dev[14], dev[15], dev[16], dev[17])
	} else {
		fmt.Fprintf(&b, "device descriptor: short read (%d bytes: % x)\n", len(dev), dev)
	}

	// Config descriptor: read the 9-byte header for wTotalLength, then
	// re-read the whole thing so the interface/endpoint walk is complete.
	if hdr, err := t.getDescriptor(descTypeConfig, 0, 9); err != nil {
		fmt.Fprintf(&b, "config descriptor: unavailable (%v)\n", err)
	} else if len(hdr) >= 9 {
		total := int(uint16(hdr[2]) | uint16(hdr[3])<<8)
		if total < 9 || total > 4096 {
			total = 256
		}
		full, err := t.getDescriptor(descTypeConfig, 0, total)
		if err != nil || len(full) < 9 {
			full = hdr
		}
		fmt.Fprintf(&b, "config descriptor: numInterfaces=%d configValue=%d attrs=0x%02x totalLen=%d\n", hdr[4], hdr[5], hdr[7], total)
		describeInterfaces(&b, full)
	}

	// Control-IN read probe: can the device service vendor reads at all?
	// Reading USB_SYSCTL (block=USB, addr=0x2000) mirrors a rtl2832u
	// read_reg. If reads succeed while the USB_SYSCTL *write* fails, the
	// firmware/driver is rejecting OUT transfers specifically — a very
	// different problem from "the device is wedged".
	if in, err := t.ControlIn(0, 0x2000, uint16(1)<<8, 1, 300); err != nil {
		fmt.Fprintf(&b, "control-IN probe (read block=USB addr=0x2000): FAILED (%v)\n", err)
	} else {
		fmt.Fprintf(&b, "control-IN probe (read block=USB addr=0x2000): ok, got % x\n", in)
	}
	return b.String()
}

// getDescriptor wraps WinUsb_GetDescriptor for the standard descriptor
// types, returning the bytes the device actually sent. Best-effort: any
// failure surfaces as an error the caller folds into the diagnostic dump.
func (t *winTransport) getDescriptor(descType, index uint8, length int) ([]byte, error) {
	if t.closed.Load() {
		return nil, ErrClosed
	}
	if length <= 0 {
		return nil, fmt.Errorf("winusb: bad descriptor length %d", length)
	}
	buf := make([]byte, length)
	var transferred uint32
	ret, _, errno := procWinUsbGetDescriptor.Call(
		t.ifaceHandle,
		uintptr(descType),
		uintptr(index),
		0, // LanguageID (0 for device/config descriptors)
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(length),
		uintptr(unsafe.Pointer(&transferred)),
	)
	if ret == 0 {
		return nil, winErr(errno)
	}
	if int(transferred) > len(buf) {
		transferred = uint32(len(buf))
	}
	return buf[:transferred], nil
}

// describeInterfaces walks a raw USB configuration descriptor and emits
// one line per interface and per endpoint. It tolerates truncated /
// malformed input by bailing out the moment a length field doesn't fit.
func describeInterfaces(b *strings.Builder, cfg []byte) {
	for i := 0; i+2 <= len(cfg); {
		l := int(cfg[i])
		if l < 2 || i+l > len(cfg) {
			break
		}
		switch cfg[i+1] {
		case 0x04: // INTERFACE
			if l >= 9 {
				fmt.Fprintf(b, "  interface: num=%d alt=%d numEndpoints=%d class=%d sub=%d proto=%d\n",
					cfg[i+2], cfg[i+3], cfg[i+4], cfg[i+5], cfg[i+6], cfg[i+7])
			}
		case 0x05: // ENDPOINT
			if l >= 7 {
				fmt.Fprintf(b, "    endpoint: addr=0x%02x attrs=0x%02x maxPacket=%d interval=%d\n",
					cfg[i+2], cfg[i+3], int(uint16(cfg[i+4])|uint16(cfg[i+5])<<8), cfg[i+6])
			}
		}
		i += l
	}
}

// ----------------------------------------------------------------------
// SetupAPI helpers

// getDeviceInterfacePath issues SetupDiGetDeviceInterfaceDetailW twice:
// once with NULL to learn the required size, then with a sized buffer.
// The buffer layout is: [DWORD cbSize][padding to 8B][UTF-16 path NUL].
func getDeviceInterfacePath(devSet uintptr, iface *spDeviceInterfaceData) (string, error) {
	var requiredSize uint32
	procSetupDiGetDeviceInterfaceDetailW.Call(
		devSet,
		uintptr(unsafe.Pointer(iface)),
		0,
		0,
		uintptr(unsafe.Pointer(&requiredSize)),
		0,
	)
	if requiredSize < spDeviceInterfaceDetailDataHeaderSize {
		return "", errors.New("winusb: bogus required size from SetupDiGetDeviceInterfaceDetailW")
	}
	buf := make([]byte, requiredSize)
	*(*uint32)(unsafe.Pointer(&buf[0])) = spDeviceInterfaceDetailDataHeaderSize
	ret, _, errno := procSetupDiGetDeviceInterfaceDetailW.Call(
		devSet,
		uintptr(unsafe.Pointer(iface)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(requiredSize),
		0,
		0,
	)
	if ret == 0 {
		return "", fmt.Errorf("winusb: SetupDiGetDeviceInterfaceDetailW: %w", winErr(errno))
	}
	// DevicePath is the UTF-16 string starting at offset 4 (right after
	// the cbSize DWORD). Walk until the NUL.
	const pathOffset = 4
	if len(buf) <= pathOffset {
		return "", errors.New("winusb: empty path")
	}
	tail := buf[pathOffset:]
	// Re-interpret as []uint16.
	utf16Len := len(tail) / 2
	utf16Slice := unsafe.Slice((*uint16)(unsafe.Pointer(&tail[0])), utf16Len)
	end := utf16Len
	for i, c := range utf16Slice {
		if c == 0 {
			end = i
			break
		}
	}
	return windows.UTF16ToString(utf16Slice[:end]), nil
}

// parseDevicePath extracts VID, PID, and serial from a Windows USB
// device path like:
//
//	\\?\usb#vid_0bda&pid_2838#00000001#{a5dcbf10-6530-11d2-901f-00c04fb951ed}
//
// Comparison is case-insensitive (Windows mixes cases between API
// callers); the serial is the third "#"-separated element when present.
// Returns (0,0,"") when neither vid_ nor pid_ tokens match — caller skips
// such entries.
func parseDevicePath(p string) (vid, pid uint16, serial string) {
	lower := strings.ToLower(p)
	if i := strings.Index(lower, "vid_"); i >= 0 && i+8 <= len(p) {
		if v, err := strconv.ParseUint(p[i+4:i+8], 16, 16); err == nil {
			vid = uint16(v)
		}
	}
	if i := strings.Index(lower, "pid_"); i >= 0 && i+8 <= len(p) {
		if v, err := strconv.ParseUint(p[i+4:i+8], 16, 16); err == nil {
			pid = uint16(v)
		}
	}
	parts := strings.Split(p, "#")
	if len(parts) >= 3 {
		serial = parts[2]
	}
	return vid, pid, serial
}

// winErr maps Windows error codes to the package's sentinel errors;
// unmapped errors come through wrapped as-is so callers can still
// inspect the underlying code via errors.As(*windows.Errno).
//
// ERROR_GEN_FAILURE (0x1F) is deliberately NOT folded into
// ErrDeviceGone: it commonly means the device firmware NAK'd the
// request, the pipe stalled, or the wrong function driver is bound
// (e.g. libusbK rather than in-box WinUSB.sys). Conflating it with
// physical disconnect actively misled the issue #270 reporter.
// Instead it wraps both ErrPipeStalled (so the bring-up retry envelope
// in purego/driver.go treats it as resetable, matching the Linux EPIPE
// path) and the underlying windows.Errno (so existing call-sites that
// inspect the Win32 code via errors.Is still work).
func winErr(errno error) error {
	if errno == nil {
		return nil
	}
	switch errno {
	case windows.ERROR_DEVICE_NOT_CONNECTED,
		windows.ERROR_NO_SUCH_DEVICE,
		windows.ERROR_DEV_NOT_EXIST:
		return ErrDeviceGone
	case windows.ERROR_GEN_FAILURE:
		return fmt.Errorf("winusb: device rejected request (ERROR_GEN_FAILURE 0x1F — firmware NAK / stalled pipe / wrong driver bound; try re-binding to WinUSB via Zadig): %w (errno: %w)", ErrPipeStalled, errno)
	case windows.ERROR_SEM_TIMEOUT, windows.ERROR_TIMEOUT:
		return fmt.Errorf("%w (windows errno=%v)", ErrTimeout, errno)
	}
	return errno
}

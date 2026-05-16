//go:build darwin

// Package-level documentation for the macOS USB transport. The IOKit
// + CoreFoundation FFI surface lives in usb_darwin_iokit.go; this
// file contains the higher-level [Enumerator] / [Transport]
// implementations the rest of the rtlsdr driver consumes.
//
// PR-10 of the librtlsdr → pure-Go rewrite. Replaces the
// "ErrMacOSUnsupported" stub from PR-03; closes tracking issue
// https://github.com/MattCheramie/GopherTrunk/issues/82.
//
// Implementation notes:
//
//   - All IOKit / CoreFoundation calls go through purego (see
//     internal/sdr/rtlsdr/usb/usb_darwin_iokit.go). No CGO.
//   - Enumeration walks IOKit's "IOUSBDevice" service registry and
//     reads VID/PID/serial/etc. as IORegistry properties. No device
//     is opened during List.
//   - Open creates an IOUSBDeviceInterface via the standard
//     IOCFPlugIn dance, opens the device, walks its interface
//     iterator, and claims interface 0 (the only one RTL2832U
//     exposes).
//   - Control transfers go through IOUSBDeviceInterface::DeviceRequest
//     with an IOUSBDevRequest struct that mirrors the USB 2.0 setup
//     packet plus a data pointer.
//   - Bulk-IN runs N goroutines, each pinned to its own OS thread
//     via runtime.LockOSThread, doing synchronous ReadPipe calls in
//     a loop. Cancellation is via AbortPipe — pending reads return
//     with kIOReturnAborted, the goroutines see the closed-flag and
//     exit. This sidesteps CFRunLoop callbacks entirely; trade-off
//     is one OS thread per slot (32 by default) instead of one
//     reaper-loop thread for the whole ring.
//
// **Hardware validation status**: this code compiles cleanly under
// `GOOS=darwin GOARCH={amd64,arm64} CGO_ENABLED=0` and the FFI
// surface is structurally complete — but it has not been exercised
// against real RTL-SDR hardware on macOS. Contributors with the
// hardware should diff USB control-transfer captures (Wireshark +
// usbmon-equivalent) against the Linux backend's output to verify
// wire format. See the PR description for the manual-test
// follow-up checklist.

package usb

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// IOKit load is lazy: we don't want any framework-resolution glitch
// to crash the test binary at startup. First call to
// platformEnumerator runs loadIOKit under sync.Once and stashes the
// outcome; subsequent calls return the cached result.
var (
	darwinLoadOnce sync.Once
	darwinLoadErr  error
)

// platformEnumerator returns the IOKit-backed enumerator unless
// IOKit/CoreFoundation failed to load. The load is performed lazily
// here (not in init) so the test binary itself starts cleanly even
// on macOS revisions where purego's Dlopen / fakecgo path
// misbehaves; the failure surfaces from List/Open instead.
func platformEnumerator() Enumerator {
	darwinLoadOnce.Do(func() {
		darwinLoadErr = loadIOKit()
	})
	if darwinLoadErr != nil {
		return loadFailedEnumerator{err: darwinLoadErr}
	}
	return &darwinEnumerator{}
}

// loadFailedEnumerator surfaces the dlopen failure to callers.
type loadFailedEnumerator struct{ err error }

func (e loadFailedEnumerator) Name() string { return "iokit-load-failed" }
func (e loadFailedEnumerator) List(uint16, uint16) ([]Descriptor, error) {
	return nil, fmt.Errorf("usb: IOKit framework load failed: %w", e.err)
}
func (e loadFailedEnumerator) Open(Descriptor) (Transport, error) {
	return nil, fmt.Errorf("usb: IOKit framework load failed: %w", e.err)
}

// darwinEnumerator scans IOKit's IOUSBDevice service registry.
type darwinEnumerator struct{}

func (d *darwinEnumerator) Name() string { return "iokit" }

// List queries IOKit for every USB device the kernel knows about,
// reads VID/PID/serial/etc. via IORegistryEntryCreateCFProperty,
// applies the optional vid/pid filter, and returns a [Descriptor]
// per match. Devices not opened.
//
// On macOS, "Path" is the IOService location-id string (a 32-bit
// hex value Apple guarantees stable across reboots for a given
// USB-port location); we round-trip it through Open to find the
// matching service when claiming.
func (d *darwinEnumerator) List(vid, pid uint16) ([]Descriptor, error) {
	className := append([]byte("IOUSBDevice"), 0)
	matchingDict := ioServiceMatching(&className[0])
	if matchingDict == 0 {
		return nil, errors.New("usb: IOServiceMatching(IOUSBDevice) returned NULL")
	}
	// IOServiceGetMatchingServices CONSUMES the matching dictionary
	// — no defer cfRelease here, the kernel owns it after the call.
	var iter ioIterator
	if rc := ioServiceGetMatchingServices(kIOMasterPortDefault, matchingDict, &iter); rc != kIOReturnSuccess {
		return nil, fmt.Errorf("usb: IOServiceGetMatchingServices: 0x%08x", uint32(rc))
	}
	defer ioObjectRelease(iter)

	var out []Descriptor
	for {
		svc := ioIteratorNext(iter)
		if svc == 0 {
			break
		}
		desc, ok := readServiceDescriptor(svc)
		ioObjectRelease(svc)
		if !ok {
			continue
		}
		if vid != 0 && desc.VID != vid {
			continue
		}
		if pid != 0 && desc.PID != pid {
			continue
		}
		out = append(out, desc)
	}
	return out, nil
}

// readServiceDescriptor pulls the standard USB device-descriptor
// fields out of a IOService's IORegistry properties. Returns
// (Descriptor, true) on success; (zero, false) if the service
// doesn't carry the expected properties (e.g. it's a hub or composite
// child rather than a leaf USB device).
func readServiceDescriptor(svc ioService) (Descriptor, bool) {
	vid, ok := readUSBProperty(svc, "idVendor")
	if !ok {
		return Descriptor{}, false
	}
	pid, ok := readUSBProperty(svc, "idProduct")
	if !ok {
		return Descriptor{}, false
	}
	loc, _ := readUSBProperty(svc, "locationID")
	d := Descriptor{
		VID:  uint16(vid),
		PID:  uint16(pid),
		Path: fmt.Sprintf("0x%08x", loc),
	}
	// Optional string fields — empty when missing. We don't read
	// them via the IOKit string-descriptor path here (that requires
	// opening the device). The IORegistry's "USB Serial Number" /
	// "USB Vendor Name" / "USB Product Name" properties carry the
	// already-decoded strings.
	d.Serial = readUSBString(svc, "USB Serial Number")
	d.Manufacturer = readUSBString(svc, "USB Vendor Name")
	d.Product = readUSBString(svc, "USB Product Name")
	return d, true
}

// readUSBProperty returns a 32-bit integer property from the
// IORegistry, or (0, false) when the key is missing or non-numeric.
func readUSBProperty(svc ioService, key string) (uint32, bool) {
	keyBytes := append([]byte(key), 0)
	cfKey := cfStringCreateWithCString(kCFAllocatorDefault, &keyBytes[0], kCFStringEncodingASCII)
	if cfKey == 0 {
		return 0, false
	}
	defer cfRelease(cfKey)
	cfNum := ioRegistryEntryCreateCFProperty(svc, cfKey, kCFAllocatorDefault, 0)
	if cfNum == 0 {
		return 0, false
	}
	defer cfRelease(cfNum)
	var v int32
	if !cfNumberGetValue(cfNum, kCFNumberSInt32Type, unsafe.Pointer(&v)) {
		return 0, false
	}
	return uint32(v), true
}

// readUSBString reads a UTF-8 string property from the IORegistry.
// IOKit decodes USB string descriptors (UTF-16LE on the wire) into
// UTF-8 CFStringRefs in the property dictionary, so we read them
// back via CFStringGetCString with kCFStringEncodingUTF8.
//
// USB descriptor strings are bounded at 255 chars by the USB spec
// (1 length byte + 254 UTF-16 code units, which can round up to
// ~500 UTF-8 bytes worst case for full BMP characters). A 1024-byte
// stack-allocated buffer covers every realistic device. CFStringGetCString
// returns false when the buffer is too small; in that case we log
// (not yet — silent fallback) and return "" so the caller stays on
// the friendly-name path.
//
// Returns "" when the property is missing, the buffer is too small,
// or the framework hasn't loaded.
func readUSBString(svc ioService, key string) string {
	if cfStringGetCString == nil {
		return ""
	}
	keyBytes := append([]byte(key), 0)
	cfKey := cfStringCreateWithCString(kCFAllocatorDefault, &keyBytes[0], kCFStringEncodingASCII)
	if cfKey == 0 {
		return ""
	}
	defer cfRelease(cfKey)
	cfStr := ioRegistryEntryCreateCFProperty(svc, cfKey, kCFAllocatorDefault, 0)
	if cfStr == 0 {
		return ""
	}
	defer cfRelease(cfStr)
	// 1024 bytes is the practical upper bound for a USB descriptor
	// string converted to UTF-8 (255-char limit × ≤4 UTF-8 bytes per
	// code unit, plus terminator). Stack allocation avoids the GC.
	var buf [1024]byte
	if !cfStringGetCString(cfStr, &buf[0], int64(len(buf)), kCFStringEncodingUTF8) {
		return ""
	}
	// Walk to the NUL terminator and slice. We don't trust the buffer
	// to be entirely NUL-padded beyond the string itself.
	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}
	return string(buf[:end])
}

// Open creates a IOUSBDeviceInterface for the device at d.Path
// (matched by locationID), opens it, walks its interface iterator,
// claims interface 0, and returns a wired-up [darwinTransport].
func (e *darwinEnumerator) Open(d Descriptor) (Transport, error) {
	wantLoc, err := strconv.ParseUint(d.Path, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("usb: bad Descriptor.Path %q: %w", d.Path, err)
	}
	className := append([]byte("IOUSBDevice"), 0)
	matchingDict := ioServiceMatching(&className[0])
	if matchingDict == 0 {
		return nil, errors.New("usb: IOServiceMatching returned NULL")
	}
	var iter ioIterator
	if rc := ioServiceGetMatchingServices(kIOMasterPortDefault, matchingDict, &iter); rc != kIOReturnSuccess {
		return nil, fmt.Errorf("usb: IOServiceGetMatchingServices: 0x%08x", uint32(rc))
	}
	defer ioObjectRelease(iter)

	var svc ioService
	for {
		next := ioIteratorNext(iter)
		if next == 0 {
			break
		}
		loc, ok := readUSBProperty(next, "locationID")
		if ok && uint64(loc) == wantLoc {
			svc = next
			break
		}
		ioObjectRelease(next)
	}
	if svc == 0 {
		return nil, fmt.Errorf("usb: no IOService with locationID %s (device removed?)", d.Path)
	}
	defer ioObjectRelease(svc)

	devIface, err := openDeviceInterface(svc)
	if err != nil {
		return nil, err
	}

	// Open the device — exclusive access for control transfers.
	if rc := vtableCall(devIface, deviceUSBDeviceOpen); rc != kIOReturnSuccess {
		release(devIface)
		return nil, fmt.Errorf("usb: USBDeviceOpen: %w", translateIOReturn(rc))
	}

	t := &darwinTransport{
		devIface: devIface,
		desc:     d,
	}
	return t, nil
}

// openDeviceInterface runs the IOCFPlugIn dance: create the plug-in,
// QueryInterface for the IOUSBDeviceInterface, release the plug-in.
// Returns a usable IOUSBDeviceInterface pointer (a **interface in C
// terms; the caller dereferences via vtableCall).
func openDeviceInterface(svc ioService) (uintptr, error) {
	var plugin *uintptr
	var score int32
	pluginUUID := cfUUIDCreateFromUUIDBytes(kCFAllocatorDefault, uuidIOUSBDeviceUserClientType)
	if pluginUUID == 0 {
		return 0, errors.New("usb: CFUUIDCreateFromUUIDBytes(IOUSBDeviceUserClientType) returned NULL")
	}
	defer cfRelease(pluginUUID)
	ifaceUUID := cfUUIDCreateFromUUIDBytes(kCFAllocatorDefault, uuidIOCFPlugInInterface)
	if ifaceUUID == 0 {
		return 0, errors.New("usb: CFUUIDCreateFromUUIDBytes(IOCFPlugInInterface) returned NULL")
	}
	defer cfRelease(ifaceUUID)
	if rc := ioCreatePlugInInterfaceForService(svc, pluginUUID, ifaceUUID, &plugin, &score); rc != kIOReturnSuccess {
		return 0, fmt.Errorf("usb: IOCreatePlugInInterfaceForService: 0x%08x", uint32(rc))
	}
	if plugin == nil {
		return 0, errors.New("usb: IOCreatePlugInInterfaceForService returned NULL plugin")
	}
	defer release(uintptr(unsafe.Pointer(plugin)))

	devIface, err := queryInterface(uintptr(unsafe.Pointer(plugin)), uuidIOUSBDeviceInterface)
	if err != nil {
		return 0, fmt.Errorf("usb: QueryInterface(IOUSBDeviceInterface): %w", err)
	}
	return devIface, nil
}

// darwinTransport is the IOKit-backed [Transport] for one open
// device. It tracks the device + interface IOUSBInterface handles
// and the per-bulk-IN-slot goroutines that ReadPipe synchronously
// in a loop.
type darwinTransport struct {
	devIface   uintptr // IOUSBDeviceInterface**
	ifaceIface uintptr // IOUSBInterfaceInterface**, populated by ClaimInterface
	desc       Descriptor
	closed     atomic.Bool

	bulkMu       sync.Mutex
	bulkActive   bool
	bulkPipeRef  uint8
	bulkSlots    []*darwinBulkSlot
	bulkStopFlag atomic.Int32
	bulkDone     chan struct{}
}

type darwinBulkSlot struct {
	buf []byte
}

func (t *darwinTransport) ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error) {
	if t.closed.Load() {
		return nil, ErrClosed
	}
	if n < 0 || n > 0xFFFF {
		return nil, fmt.Errorf("usb: control IN length %d out of range", n)
	}
	var buf []byte
	if n > 0 {
		buf = make([]byte, n)
	}
	req := iousbDevRequest{
		BmRequestType: VendorIn,
		BRequest:      bRequest,
		WValue:        wValue,
		WIndex:        wIndex,
		WLength:       uint16(n),
	}
	if n > 0 {
		req.PData = unsafe.Pointer(&buf[0])
	}
	rc := vtableCall(t.devIface, deviceDeviceRequest, uintptr(unsafe.Pointer(&req)))
	if err := translateIOReturn(rc); err != nil {
		return nil, fmt.Errorf("usb: DeviceRequest IN: %w", err)
	}
	_ = timeoutMs // IOUSBDeviceInterface v1.0's DeviceRequest has no timeout; v1.8.2's DeviceRequestTO does. Future enhancement.
	return buf[:req.WLenDone], nil
}

func (t *darwinTransport) ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if len(data) > 0xFFFF {
		return fmt.Errorf("usb: control OUT length %d out of range", len(data))
	}
	req := iousbDevRequest{
		BmRequestType: VendorOut,
		BRequest:      bRequest,
		WValue:        wValue,
		WIndex:        wIndex,
		WLength:       uint16(len(data)),
	}
	if len(data) > 0 {
		req.PData = unsafe.Pointer(&data[0])
	}
	rc := vtableCall(t.devIface, deviceDeviceRequest, uintptr(unsafe.Pointer(&req)))
	if err := translateIOReturn(rc); err != nil {
		return fmt.Errorf("usb: DeviceRequest OUT: %w", err)
	}
	_ = timeoutMs
	return nil
}

// ClaimInterface walks CreateInterfaceIterator and claims the first
// interface (RTL-SDR exposes one). Subsequent calls with num=0 are
// no-ops; num != 0 errors (matches the Windows backend).
func (t *darwinTransport) ClaimInterface(num int) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if num != 0 {
		return fmt.Errorf("usb: only interface 0 supported (got %d)", num)
	}
	if t.ifaceIface != 0 {
		return nil
	}
	req := iousbFindInterfaceRequest{
		BInterfaceClass:    0xFFFF,
		BInterfaceSubClass: 0xFFFF,
		BInterfaceProtocol: 0xFFFF,
		BAlternateSetting:  0xFFFF,
	}
	var iter ioIterator
	rc := vtableCall(t.devIface, deviceCreateInterfaceIterator,
		uintptr(unsafe.Pointer(&req)),
		uintptr(unsafe.Pointer(&iter)),
	)
	if err := translateIOReturn(rc); err != nil {
		return fmt.Errorf("usb: CreateInterfaceIterator: %w", err)
	}
	defer ioObjectRelease(iter)

	svc := ioIteratorNext(iter)
	if svc == 0 {
		return errors.New("usb: device has no USB interfaces")
	}
	defer ioObjectRelease(svc)

	var plugin *uintptr
	var score int32
	pluginUUID := cfUUIDCreateFromUUIDBytes(kCFAllocatorDefault, uuidIOUSBDeviceUserClientType)
	if pluginUUID == 0 {
		return errors.New("usb: CFUUIDCreateFromUUIDBytes(IOUSBDeviceUserClientType) returned NULL")
	}
	defer cfRelease(pluginUUID)
	pluginIfaceUUID := cfUUIDCreateFromUUIDBytes(kCFAllocatorDefault, uuidIOCFPlugInInterface)
	if pluginIfaceUUID == 0 {
		return errors.New("usb: CFUUIDCreateFromUUIDBytes(IOCFPlugInInterface) returned NULL")
	}
	defer cfRelease(pluginIfaceUUID)
	if rc := ioCreatePlugInInterfaceForService(svc, pluginUUID, pluginIfaceUUID, &plugin, &score); rc != kIOReturnSuccess {
		return fmt.Errorf("usb: IOCreatePlugInInterfaceForService(interface): 0x%08x", uint32(rc))
	}
	defer release(uintptr(unsafe.Pointer(plugin)))

	ifaceIface, err := queryInterface(uintptr(unsafe.Pointer(plugin)), uuidIOUSBInterfaceInterface)
	if err != nil {
		return fmt.Errorf("usb: QueryInterface(IOUSBInterfaceInterface): %w", err)
	}
	if rc := vtableCall(ifaceIface, ifaceUSBInterfaceOpen); rc != kIOReturnSuccess {
		release(ifaceIface)
		return fmt.Errorf("usb: USBInterfaceOpen: %w", translateIOReturn(uintptr(rc)))
	}
	t.ifaceIface = ifaceIface
	return nil
}

func (t *darwinTransport) ReleaseInterface(int) error {
	if t.ifaceIface == 0 {
		return nil
	}
	vtableCall(t.ifaceIface, ifaceUSBInterfaceClose)
	release(t.ifaceIface)
	t.ifaceIface = 0
	return nil
}

// Reset issues IOUSBDeviceInterface::ResetDevice — the device
// re-enumerates and the caller must re-Open. Documented as
// best-effort across all backends.
func (t *darwinTransport) Reset() error {
	if t.closed.Load() {
		return ErrClosed
	}
	rc := vtableCall(t.devIface, deviceResetDevice)
	return translateIOReturn(rc)
}

// StartBulkIn spawns one OS-thread-pinned goroutine per slot. Each
// loops in ReadPipe; AbortPipe unblocks them all on Stop.
//
// Trade-off vs. CFRunLoop async: simpler, no callback marshalling
// across the C/Go boundary, no run-loop thread to babysit. Cost is
// ringBufs OS threads (32 default → ~32 MB stack). Acceptable for
// a foreground SDR daemon.
func (t *darwinTransport) StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte)) error {
	if t.closed.Load() {
		return ErrClosed
	}
	if t.ifaceIface == 0 {
		return errors.New("usb: ClaimInterface(0) must be called before StartBulkIn")
	}
	if ringBufs <= 0 || bufLen <= 0 {
		return fmt.Errorf("usb: invalid bulk ring geometry (bufs=%d len=%d)", ringBufs, bufLen)
	}
	t.bulkMu.Lock()
	defer t.bulkMu.Unlock()
	if t.bulkActive {
		return ErrBulkActive
	}
	pipeRef, err := t.findPipeRef(epAddr)
	if err != nil {
		return err
	}
	slots := make([]*darwinBulkSlot, ringBufs)
	for i := range slots {
		slots[i] = &darwinBulkSlot{buf: make([]byte, bufLen)}
	}
	t.bulkPipeRef = pipeRef
	t.bulkSlots = slots
	t.bulkActive = true
	t.bulkStopFlag.Store(0)
	t.bulkDone = make(chan struct{}, ringBufs)
	for _, s := range slots {
		go t.bulkLoop(pipeRef, s, onPacket)
	}
	return nil
}

// findPipeRef walks the interface's pipe list to find the pipeRef
// (1..GetNumEndpoints) that backs the given epAddr. RTL2832U exposes
// a single bulk-IN pipe at endpoint 0x81; on macOS the kernel-level
// "pipeRef" is usually 1, but we discover it via GetPipeProperties to
// stay robust against firmware variants.
func (t *darwinTransport) findPipeRef(epAddr byte) (uint8, error) {
	var nEndpoints uint8
	rc := vtableCall(t.ifaceIface, ifaceGetNumEndpoints, uintptr(unsafe.Pointer(&nEndpoints)))
	if err := translateIOReturn(rc); err != nil {
		return 0, fmt.Errorf("usb: GetNumEndpoints: %w", err)
	}
	for ref := uint8(1); ref <= nEndpoints; ref++ {
		var direction, number, transferType, maxPacketSize, interval uint8
		var maxPacketSize16 uint16
		_ = maxPacketSize
		rc := vtableCall(t.ifaceIface, ifaceGetPipeProperties,
			uintptr(ref),
			uintptr(unsafe.Pointer(&direction)),
			uintptr(unsafe.Pointer(&number)),
			uintptr(unsafe.Pointer(&transferType)),
			uintptr(unsafe.Pointer(&maxPacketSize16)),
			uintptr(unsafe.Pointer(&interval)),
		)
		if err := translateIOReturn(rc); err != nil {
			continue
		}
		// Match: direction=IN (1) + endpoint number == epAddr & 0x7F.
		if direction == 1 && number == (epAddr&0x7F) {
			return ref, nil
		}
	}
	return 0, fmt.Errorf("usb: no IN pipe for endpoint 0x%02x", epAddr)
}

// bulkLoop runs in a dedicated OS-thread-pinned goroutine. ReadPipe
// blocks the kernel side; on AbortPipe + close-flag we exit. Each
// successful read is delivered via onPacket.
func (t *darwinTransport) bulkLoop(pipeRef uint8, slot *darwinBulkSlot, onPacket func([]byte)) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func() { t.bulkDone <- struct{}{} }()
	for {
		if t.bulkStopFlag.Load() != 0 {
			return
		}
		size := uint32(len(slot.buf))
		rc := vtableCall(t.ifaceIface, ifaceReadPipe,
			uintptr(pipeRef),
			uintptr(unsafe.Pointer(&slot.buf[0])),
			uintptr(unsafe.Pointer(&size)),
		)
		if t.bulkStopFlag.Load() != 0 {
			return
		}
		if rc != kIOReturnSuccess {
			// kIOReturnAborted (cancellation) and any other
			// transport error both exit the loop. ResetPipe
			// below would be needed before re-Start.
			return
		}
		if size > 0 {
			onPacket(slot.buf[:size])
		}
	}
}

func (t *darwinTransport) StopBulkIn() error {
	t.bulkMu.Lock()
	if !t.bulkActive {
		t.bulkMu.Unlock()
		return ErrBulkInactive
	}
	t.bulkStopFlag.Store(1)
	pipeRef := t.bulkPipeRef
	slotCount := len(t.bulkSlots)
	t.bulkActive = false
	t.bulkMu.Unlock()

	// AbortPipe makes every blocked ReadPipe return with
	// kIOReturnAborted; goroutines see it and exit.
	if t.ifaceIface != 0 {
		vtableCall(t.ifaceIface, ifaceAbortPipe, uintptr(pipeRef))
		// ResetPipe so subsequent StartBulkIn calls work without a
		// re-Open of the device. AbortPipe leaves the pipe in a
		// halted state until Reset is issued.
		vtableCall(t.ifaceIface, ifaceResetPipe, uintptr(pipeRef))
	}
	// Drain done-channel so we know every goroutine has exited
	// before we return.
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for i := 0; i < slotCount; i++ {
		select {
		case <-t.bulkDone:
		case <-deadline.C:
			return errors.New("usb: bulk reaper goroutines did not exit within 2 s")
		}
	}
	t.bulkMu.Lock()
	t.bulkSlots = nil
	t.bulkMu.Unlock()
	return nil
}

func (t *darwinTransport) Close() error {
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
	if t.ifaceIface != 0 {
		vtableCall(t.ifaceIface, ifaceUSBInterfaceClose)
		release(t.ifaceIface)
		t.ifaceIface = 0
	}
	if t.devIface != 0 {
		vtableCall(t.devIface, deviceUSBDeviceClose)
		release(t.devIface)
		t.devIface = 0
	}
	return nil
}

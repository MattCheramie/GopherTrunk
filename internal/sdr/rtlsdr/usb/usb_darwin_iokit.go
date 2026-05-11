//go:build darwin

package usb

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// IOKit + CoreFoundation function pointers loaded once at package
// init via purego.Dlopen + purego.Dlsym. Keeps the FFI surface in
// one file so the higher-level driver in usb_darwin.go can read
// like ordinary Go code.
//
// References:
//   - Apple's IOKit User Programming Guide:
//     https://developer.apple.com/library/archive/documentation/DeviceDrivers/Conceptual/AccessingHardware/AH_Intro/AH_Intro.html
//   - libusb's macOS backend (proven port of the same calls):
//     https://github.com/libusb/libusb/blob/master/libusb/os/darwin_usb.c

// CoreFoundation type aliases. uintptr stands in for the opaque
// CFTypeRef family — we never inspect them, only pass them through.
type (
	cfTypeRef        = uintptr
	cfStringRef      = uintptr
	cfDictionaryRef  = uintptr
	cfMutableDictRef = uintptr
	cfNumberRef      = uintptr
	cfAllocatorRef   = uintptr
)

// IOKit type aliases. mach_port_t / io_object_t / io_iterator_t /
// io_service_t are all 32-bit kernel handles on macOS.
type (
	machPort     uint32
	ioObject     = machPort
	ioIterator   = machPort
	ioService    = machPort
	ioRegEntry   = machPort
)

// kIOReturn (kern_return_t) status codes. We only check for
// kIOReturnSuccess on the hot path; specific errors get translated
// in translateIOReturn below.
const (
	kIOReturnSuccess     = 0
	kIOReturnNoDevice    = 0xE00002C0
	kIOReturnAborted     = 0xE00002EB
	kIOReturnTimeout     = 0xE00002D6
	kIOReturnExclusive   = 0xE00002C5 // already opened by another process
	kIOReturnNotResponding = 0xE00002ED
)

// CFNumberType for CFNumberGetValue. Only kCFNumberSInt32Type is
// used — VID/PID/serial/etc. live in the IORegistry as 32-bit ints.
const kCFNumberSInt32Type = 3

// IOUSBDeviceUserClientTypeID, IOCFPlugInInterfaceID,
// IOUSBDeviceInterfaceID, IOUSBInterfaceInterfaceID — UUIDs Apple
// publishes as part of the IOKit USB API. Stable across macOS
// versions. Sourced from Apple's IOUSBLib.h.
var (
	uuidIOUSBDeviceUserClientType = cfUUIDBytes{
		0x9D, 0xC7, 0xB7, 0x80, 0x9E, 0xC0, 0x11, 0xD4,
		0xA5, 0x4F, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61,
	}
	uuidIOCFPlugInInterface = cfUUIDBytes{
		0xC2, 0x44, 0xE8, 0x58, 0x10, 0x9C, 0x11, 0xD4,
		0x91, 0xD4, 0x00, 0x50, 0xE4, 0xC6, 0x42, 0x6F,
	}
	uuidIOUSBDeviceInterface = cfUUIDBytes{
		0x5C, 0x81, 0x87, 0xD0, 0x9E, 0xF3, 0x11, 0xD4,
		0x8B, 0x45, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61,
	}
	uuidIOUSBInterfaceInterface = cfUUIDBytes{
		0x73, 0xC9, 0x7A, 0xE8, 0x9E, 0xF3, 0x11, 0xD4,
		0xB1, 0xD0, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61,
	}
)

// cfUUIDBytes mirrors Apple's CFUUIDBytes — 16 raw bytes of a UUID.
// Layout matches the C struct exactly, so the value can be passed
// to CFUUIDCreateFromUUIDBytes by reference.
type cfUUIDBytes [16]byte

// IOCFPlugInInterface vtable indices we need (after the IUnknown
// header at indices 0..3).
const (
	pluginQueryInterface = 1 // IUnknown
	// Plug-in's only interesting method is QueryInterface (above).
)

// IOUSBDeviceInterface vtable indices (post-IUnknown).
const (
	deviceUSBDeviceOpen           = 8
	deviceUSBDeviceClose          = 9
	deviceGetDeviceVendor         = 13
	deviceGetDeviceProduct        = 14
	deviceGetLocationID           = 20
	deviceResetDevice             = 25
	deviceDeviceRequest           = 26
	deviceCreateInterfaceIterator = 28
)

// IOUSBInterfaceInterface vtable indices (post-IUnknown).
const (
	ifaceUSBInterfaceOpen   = 8
	ifaceUSBInterfaceClose  = 9
	ifaceGetNumEndpoints    = 19
	ifaceGetPipeProperties  = 26
	ifaceAbortPipe          = 28
	ifaceResetPipe          = 29
	ifaceReadPipe           = 31
)

// iousbDevRequest mirrors IOUSBDevRequest from IOUSBLib.h. Layout:
// 8-byte setup packet + pData pointer + wLenDone uint32 + 4 bytes
// padding to round up to 8-byte alignment on 64-bit.
type iousbDevRequest struct {
	BmRequestType uint8
	BRequest      uint8
	WValue        uint16
	WIndex        uint16
	WLength       uint16
	PData         unsafe.Pointer
	WLenDone      uint32
	_pad          uint32
}

// iousbFindInterfaceRequest is what CreateInterfaceIterator takes —
// a four-uint16 wildcard (0xFFFF) finds every interface.
type iousbFindInterfaceRequest struct {
	BInterfaceClass    uint16
	BInterfaceSubClass uint16
	BInterfaceProtocol uint16
	BAlternateSetting  uint16
}

// kIOMasterPortDefault — the default mach port for IOKit calls on
// modern macOS. NULL works equivalently; we use 0 explicitly.
const kIOMasterPortDefault machPort = 0

// dyld handles populated at init.
var (
	hCoreFoundation uintptr
	hIOKit          uintptr
)

// CoreFoundation function pointers we hold across the package
// lifetime. RegisterLibFunc populates each at init.
var (
	cfRelease                func(cf cfTypeRef)
	cfStringCreateWithCString func(alloc cfAllocatorRef, str *byte, encoding uint32) cfStringRef
	cfNumberGetValue          func(num cfNumberRef, theType uint32, valuePtr unsafe.Pointer) bool
	cfUUIDCreateFromUUIDBytes func(alloc cfAllocatorRef, bytes cfUUIDBytes) cfTypeRef
	cfUUIDGetUUIDBytes        func(uuid cfTypeRef) cfUUIDBytes
)

// IOKit function pointers.
var (
	ioServiceMatching            func(name *byte) cfMutableDictRef
	ioServiceGetMatchingServices func(masterPort machPort, matching cfDictionaryRef, iter *ioIterator) int32
	ioIteratorNext               func(iter ioIterator) ioObject
	ioObjectRelease              func(obj ioObject) int32
	ioCreatePlugInInterfaceForService func(service ioService, pluginType cfTypeRef, interfaceType cfTypeRef, theInterface **uintptr, score *int32) int32
	ioRegistryEntryCreateCFProperty   func(entry ioRegEntry, key cfStringRef, alloc cfAllocatorRef, options uint32) cfTypeRef
)

// kCFAllocatorDefault — passing 0 yields the default allocator on
// every CoreFoundation call we make.
const kCFAllocatorDefault cfAllocatorRef = 0

// kCFStringEncodingASCII for CFStringCreateWithCString.
const kCFStringEncodingASCII uint32 = 0x0600

// loadIOKit opens IOKit + CoreFoundation and binds the function
// pointers we use. Lazy: called from platformEnumerator on first
// access via sync.Once so the test binary's startup doesn't crash
// if IOKit / purego ever misbehave on a given macOS revision —
// the failure surfaces from List/Open with a clear error instead.
//
// purego.RegisterLibFunc panics on missing symbols; we wrap the
// whole load in a recover so any such panic becomes a returned
// error.
func loadIOKit() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("usb: IOKit load panic: %v", r)
		}
	}()
	const (
		cfPath = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"
		ioPath = "/System/Library/Frameworks/IOKit.framework/IOKit"
	)
	cf, err := purego.Dlopen(cfPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return fmt.Errorf("dlopen %s: %w", cfPath, err)
	}
	io, err := purego.Dlopen(ioPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return fmt.Errorf("dlopen %s: %w", ioPath, err)
	}
	hCoreFoundation = cf
	hIOKit = io
	purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
	purego.RegisterLibFunc(&cfStringCreateWithCString, cf, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&cfNumberGetValue, cf, "CFNumberGetValue")
	purego.RegisterLibFunc(&cfUUIDCreateFromUUIDBytes, cf, "CFUUIDCreateFromUUIDBytes")
	purego.RegisterLibFunc(&cfUUIDGetUUIDBytes, cf, "CFUUIDGetUUIDBytes")
	purego.RegisterLibFunc(&ioServiceMatching, io, "IOServiceMatching")
	purego.RegisterLibFunc(&ioServiceGetMatchingServices, io, "IOServiceGetMatchingServices")
	purego.RegisterLibFunc(&ioIteratorNext, io, "IOIteratorNext")
	purego.RegisterLibFunc(&ioObjectRelease, io, "IOObjectRelease")
	purego.RegisterLibFunc(&ioCreatePlugInInterfaceForService, io, "IOCreatePlugInInterfaceForService")
	purego.RegisterLibFunc(&ioRegistryEntryCreateCFProperty, io, "IORegistryEntryCreateCFProperty")
	return nil
}

// vtableCall reads a function pointer from a COM-style vtable at the
// given index (offsets are in pointers, not bytes — purego sorts out
// arch differences) and dispatches via purego.SyscallN. The first
// argument is always `iface` itself (the IUnknown `this` pointer).
//
// Vtable layout: iface is **interface — a pointer to a pointer to
// the vtable struct. Reading *iface gives the vtable address; the
// function pointer at index N is at vtable + N*sizeof(uintptr).
//
// All pointer manipulation goes through unsafe.Pointer + unsafe.Add
// so go vet's unsafeptr check stays happy. The IOKit pointers are
// allocated by the kernel and never moved by Go's GC.
func vtableCall(iface uintptr, index int, args ...uintptr) uintptr {
	if iface == 0 {
		return uintptr(kIOReturnNoDevice)
	}
	const ptrSize = unsafe.Sizeof(uintptr(0))
	ifacePtr := unsafe.Pointer(&iface)
	// *iface dereferences the **interface to get the vtable address.
	vtable := *(*unsafe.Pointer)(*(**unsafe.Pointer)(ifacePtr))
	// vtable[index] reads the method pointer at the given index.
	fn := *(*uintptr)(unsafe.Add(vtable, uintptr(index)*ptrSize))
	all := append([]uintptr{iface}, args...)
	r1, _, _ := purego.SyscallN(fn, all...)
	return r1
}

// translateIOReturn maps the most common IOKit error codes to the
// package's sentinel errors. Anything else flows through as a raw
// numeric error so callers can still see the kIOReturn code.
func translateIOReturn(rc uintptr) error {
	switch uint32(rc) {
	case kIOReturnSuccess:
		return nil
	case kIOReturnNoDevice, kIOReturnNotResponding:
		return ErrDeviceGone
	case kIOReturnTimeout:
		return ErrTimeout
	case kIOReturnAborted:
		return ErrBulkInactive
	case kIOReturnExclusive:
		return fmt.Errorf("usb: device already opened by another process (kIOReturnExclusive)")
	default:
		return fmt.Errorf("usb: IOKit kern_return 0x%08x", uint32(rc))
	}
}

// queryInterface invokes IOCFPlugInInterface::QueryInterface to
// obtain a typed COM-style interface handle (e.g. IOUSBDeviceInterface)
// from a plug-in. plugin is a **IOCFPlugInInterface (the indirect
// pointer IOCreatePlugInInterfaceForService returns). uuid is one of
// the package-level UUID constants (uuidIOUSBDeviceInterface, etc.).
func queryInterface(plugin uintptr, uuid cfUUIDBytes) (uintptr, error) {
	cfUUID := cfUUIDCreateFromUUIDBytes(kCFAllocatorDefault, uuid)
	if cfUUID == 0 {
		return 0, fmt.Errorf("usb: CFUUIDCreateFromUUIDBytes returned NULL")
	}
	defer cfRelease(cfUUID)
	uuidBytes := cfUUIDGetUUIDBytes(cfUUID)
	var iface uintptr
	rc := vtableCall(plugin, pluginQueryInterface,
		// QueryInterface signature on macOS: (this, REFIID *, void**)
		// REFIID is passed by value but it's a 16-byte CFUUIDBytes.
		// purego splits multi-word values across registers per the
		// SysV ABI used on Darwin amd64 / AAPCS64 on arm64.
		uintptr(unsafe.Pointer(&uuidBytes)),
		uintptr(unsafe.Pointer(&iface)),
	)
	if rc != 0 {
		return 0, fmt.Errorf("usb: QueryInterface: 0x%08x", uint32(rc))
	}
	return iface, nil
}

// release calls the IUnknown::Release method on a COM-style interface,
// dropping the caller's reference. Index 3 in every IOKit vtable.
func release(iface uintptr) {
	if iface == 0 {
		return
	}
	vtableCall(iface, 3 /* IUnknown::Release */)
}

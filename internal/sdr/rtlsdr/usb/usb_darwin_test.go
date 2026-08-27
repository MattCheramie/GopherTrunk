//go:build darwin

package usb

import (
	"errors"
	"testing"
	"unsafe"
)

// TestTranslateIOReturn_PipeStalledMapsToErrPipeStalled pins the fix for
// issue #1038: on macOS a NESDR/R820T tuner init failed with
//
//	tuner init: r82xx init: burst write: rtl2832u: I2CWrite addr=0x34:
//	usb: DeviceRequest OUT: usb: IOKit kern_return 0xe000404f
//
// 0xe000404f is kIOUSBPipeStalled — the recoverable I²C-bridge STALL
// that surfaces as syscall.EPIPE on Linux and ERROR_GEN_FAILURE on
// Windows. The R820T cold-boot burst-write recovery (issue #248) keys
// off usb.ErrPipeStalled via isI2CBurstStall, so translateIOReturn must
// map the IOKit stall to that shared sentinel or the retry never fires
// and tuner detection reports "no supported tuner detected". This is the
// macOS analog of TestWinErr_GenFailureMapsToPipeStalled.
func TestTranslateIOReturn_PipeStalledMapsToErrPipeStalled(t *testing.T) {
	got := translateIOReturn(kIOUSBPipeStalled)
	if !errors.Is(got, ErrPipeStalled) {
		t.Fatalf("translateIOReturn(0x%08x) = %v, want errors.Is(err, ErrPipeStalled) so the R820T burst-write stall recovery fires on macOS", uint32(kIOUSBPipeStalled), got)
	}
}

// TestKIOUSBPipeStalledValue guards the constant against a typo: the
// value is Apple's iokit_usb_err(0x4f) and the whole fix hinges on it
// exactly matching the kern_return the host controller returns.
func TestKIOUSBPipeStalledValue(t *testing.T) {
	const want = 0xE000404F // sys_iokit | sub_iokit_usb | 0x4f
	if kIOUSBPipeStalled != want {
		t.Fatalf("kIOUSBPipeStalled = 0x%08x, want 0x%08x", uint32(kIOUSBPipeStalled), uint32(want))
	}
}

// TestTranslateIOReturn_AbortedMapsToErrTransferAborted pins the fix for
// issue #1135: on macOS with two RTL-SDR dongles on one bus, a control
// DeviceRequest during bring-up intermittently returns kIOReturnAborted
// (0xe00002eb). It used to map to ErrBulkInactive, so the failure surfaced
// as the misleading "usb: DeviceRequest OUT: usb: bulk-IN not active" AND
// was not recognised as reset-recoverable by the bring-up envelope
// (isBringupResetable), so the open aborted instead of retrying the
// transient. It must now map to the dedicated ErrTransferAborted and must
// NOT be ErrBulkInactive, whose meaning is "StopBulkIn called with no
// active stream". This is the macOS analog of the #1038 stall-mapping fix.
func TestTranslateIOReturn_AbortedMapsToErrTransferAborted(t *testing.T) {
	got := translateIOReturn(kIOReturnAborted)
	if !errors.Is(got, ErrTransferAborted) {
		t.Fatalf("translateIOReturn(0x%08x) = %v, want errors.Is(err, ErrTransferAborted) so the bring-up envelope resets and retries the transient", uint32(kIOReturnAborted), got)
	}
	if errors.Is(got, ErrBulkInactive) {
		t.Fatalf("translateIOReturn(0x%08x) = %v, must not be ErrBulkInactive — a control-transfer abort is not a bulk-IN-inactive condition (issue #1135)", uint32(kIOReturnAborted), got)
	}
}

// TestKIOReturnAbortedValue guards the constant against a typo: the whole
// #1135 fix hinges on it matching the kern_return IOKit actually returns.
func TestKIOReturnAbortedValue(t *testing.T) {
	const want = 0xE00002EB // sys_iokit | sub_iokit_common | 0x2eb
	if kIOReturnAborted != want {
		t.Fatalf("kIOReturnAborted = 0x%08x, want 0x%08x", uint32(kIOReturnAborted), uint32(want))
	}
}

// These tests pin compile-time invariants (UUIDs, struct sizes,
// vtable indices) that don't depend on IOKit actually loading at
// runtime. The real IOKit transport's behavior is verified on
// contributor macOS hardware — the FFI surface is too far below
// what unit tests can mock.

func TestDarwinEnumeratorCallable(t *testing.T) {
	// Just confirm the enumerator constructor returns a non-nil
	// Enumerator with a non-empty backend Name(). Both "iokit"
	// (IOKit loaded successfully) and "iokit-load-failed"
	// (framework dlopen failed) are valid outcomes; the test
	// binary must not crash either way.
	e := DefaultEnumerator()
	if e == nil {
		t.Fatal("DefaultEnumerator() returned nil")
	}
	if e.Name() == "" {
		t.Error("backend Name() is empty")
	}
}

func TestDarwinEnumerationClassesIncludesHostClass(t *testing.T) {
	// Regression guard for issue #257: List and Open must match
	// both legacy "IOUSBDevice" and modern "IOUSBHostDevice" IOKit
	// classes. Dropping either silently shrinks the set of dongles
	// visible to sdr list — on Apple Silicon + modern macOS the
	// legacy class can yield zero services, on older Intel hosts
	// the host class may be absent. Pin the order to keep the
	// trace log diff-stable.
	want := []string{"IOUSBDevice", "IOUSBHostDevice"}
	if len(darwinEnumerationClasses) != len(want) {
		t.Fatalf("darwinEnumerationClasses = %v, want %v", darwinEnumerationClasses, want)
	}
	for i, c := range want {
		if darwinEnumerationClasses[i] != c {
			t.Errorf("darwinEnumerationClasses[%d] = %q, want %q", i, darwinEnumerationClasses[i], c)
		}
	}
}

func TestUUIDsMatchAppleConstants(t *testing.T) {
	// Pin Apple's IOKit-USB UUIDs so a typo or table reordering
	// fails noisily rather than silently routing through a wrong
	// COM-style interface.
	cases := []struct {
		name string
		got  cfUUIDBytes
		want cfUUIDBytes
	}{
		{
			name: "IOUSBDeviceUserClientType",
			got:  uuidIOUSBDeviceUserClientType,
			want: cfUUIDBytes{0x9D, 0xC7, 0xB7, 0x80, 0x9E, 0xC0, 0x11, 0xD4, 0xA5, 0x4F, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61},
		},
		{
			name: "IOUSBInterfaceUserClientType",
			got:  uuidIOUSBInterfaceUserClientType,
			want: cfUUIDBytes{0x2D, 0x97, 0x86, 0xC6, 0x9E, 0xF3, 0x11, 0xD4, 0xAD, 0x51, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61},
		},
		{
			name: "IOCFPlugInInterface",
			got:  uuidIOCFPlugInInterface,
			want: cfUUIDBytes{0xC2, 0x44, 0xE8, 0x58, 0x10, 0x9C, 0x11, 0xD4, 0x91, 0xD4, 0x00, 0x50, 0xE4, 0xC6, 0x42, 0x6F},
		},
		{
			name: "IOUSBDeviceInterface",
			got:  uuidIOUSBDeviceInterface,
			want: cfUUIDBytes{0x5C, 0x81, 0x87, 0xD0, 0x9E, 0xF3, 0x11, 0xD4, 0x8B, 0x45, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61},
		},
		{
			name: "IOUSBInterfaceInterface",
			got:  uuidIOUSBInterfaceInterface,
			want: cfUUIDBytes{0x73, 0xC9, 0x7A, 0xE8, 0x9E, 0xF3, 0x11, 0xD4, 0xB1, 0xD0, 0x00, 0x0A, 0x27, 0x05, 0x28, 0x61},
		},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s UUID = %x, want %x", c.name, c.got, c.want)
		}
	}
}

func TestVtableIndicesNonZero(t *testing.T) {
	// Spot-check that the vtable indices we hard-coded are non-zero
	// (the IUnknown header occupies 0..3; every IOKit method we use
	// is at index 8 or higher).
	for name, idx := range map[string]int{
		"USBDeviceOpen":           deviceUSBDeviceOpen,
		"USBDeviceClose":          deviceUSBDeviceClose,
		"DeviceRequest":           deviceDeviceRequest,
		"CreateInterfaceIterator": deviceCreateInterfaceIterator,
		"USBInterfaceOpen":        ifaceUSBInterfaceOpen,
		"AbortPipe":               ifaceAbortPipe,
		"ReadPipe":                ifaceReadPipe,
	} {
		if idx < 4 {
			t.Errorf("%s vtable index %d collides with IUnknown header (0..3)", name, idx)
		}
	}
}

func TestIOUSBDevRequestSize(t *testing.T) {
	// IOUSBDevRequest must be 24 bytes on x64 / arm64 (8-byte
	// setup packet + 8-byte pData pointer + 4-byte WLenDone + 4
	// padding for the trailing union/alignment). Pin the size so a
	// future field reordering surfaces immediately.
	if got, want := unsafe.Sizeof(iousbDevRequest{}), uintptr(24); got != want {
		t.Errorf("sizeof(iousbDevRequest) = %d, want %d", got, want)
	}
}

func TestUUIDByteSize(t *testing.T) {
	if got, want := unsafe.Sizeof(cfUUIDBytes{}), uintptr(16); got != want {
		t.Errorf("sizeof(cfUUIDBytes) = %d, want %d", got, want)
	}
}

// TestReadUSBStringHandlesMissingSymbol confirms readUSBString stays
// safe when the IOKit load is bypassed (test binary running on a
// macOS revision where purego.Dlopen failed). The contract is "empty
// string, no panic" — same as the pre-CFStringGetCString stub.
//
// Constructing a "real service handle that doesn't have the property"
// path would require an actual IOKit-equipped macOS host, and
// invoking IORegistryEntryCreateCFProperty with a synthesised bogus
// handle segfaults inside the framework rather than degrading
// gracefully — so this is the deepest test we can write portably.
// TestLoadIOKitNoPanic is the regression guard for issue #257. The
// macOS USB enumerator panicked inside purego.RegisterLibFunc with
// "unsupported kind array" because two CoreFoundation function
// pointers named the array type cfUUIDBytes in their signatures —
// loadIOKit recovered the panic into an error and every macOS user
// got zero devices. Calling loadIOKit directly (not via the
// platformEnumerator sync.Once) and asserting a nil error fails
// loudly if an array type ever creeps back into a registered
// signature. TestDarwinEnumeratorCallable can't catch this: it
// accepts "iokit-load-failed" as a valid outcome.
func TestLoadIOKitNoPanic(t *testing.T) {
	if err := loadIOKit(); err != nil {
		t.Fatalf("loadIOKit() = %v, want nil", err)
	}
}

func TestReadUSBStringHandlesMissingSymbol(t *testing.T) {
	// Force the unloaded path by saving + nilling the resolved
	// function pointer; restore after the assertion.
	saved := cfStringGetCString
	cfStringGetCString = nil
	defer func() { cfStringGetCString = saved }()

	got := readUSBString(0, "USB Serial Number")
	if got != "" {
		t.Errorf("readUSBString with no resolver = %q, want empty", got)
	}
}

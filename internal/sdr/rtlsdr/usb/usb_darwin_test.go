//go:build darwin

package usb

import (
	"testing"
	"unsafe"
)

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

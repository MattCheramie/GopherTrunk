// Package usb is the platform-abstraction layer that the pure-Go RTL-SDR
// driver speaks to. It exposes the minimal slice of USB the RTL2832U
// demodulator needs — vendor control transfers in both directions and an
// async bulk-IN endpoint — and nothing else.
//
// Each supported OS provides one implementation behind the [Enumerator]
// interface returned by [DefaultEnumerator]:
//
//   - Linux uses USBDEVFS ioctls on /dev/bus/usb/BBB/DDD. See usb_linux.go.
//   - Windows (PR-02) will use WinUSB via the system DLL.
//   - macOS (PR-10) will use IOKit via purego.
//
// A non-platform [MockEnumerator]/[MockTransport] pair lives alongside the
// real backends so the RTL2832U register layer and the tuner drivers can
// be unit-tested without touching real hardware.
package usb

import (
	"errors"
)

// Vendor request-type byte values for libusb-style control transfers.
// RTL2832U firmware only speaks vendor requests; class/standard requests
// are unused by the driver.
const (
	VendorOut uint8 = 0x40 // bmRequestType: host → device, vendor, device recipient
	VendorIn  uint8 = 0xC0 // bmRequestType: device → host, vendor, device recipient
)

// Default async-bulk geometry the RTL-SDR driver uses. Held here so the
// transport implementations can pre-validate caller inputs against a
// known-good baseline; the driver still supplies the values explicitly.
const (
	DefaultRingBuffers = 32
	DefaultBufferLen   = 16 * 1024
)

// Descriptor is the pre-resolved identity of a single USB device. Returned
// by [Enumerator.List] without claiming the device, so enumeration is safe
// to run without root privileges where sysfs permissions allow it.
type Descriptor struct {
	Bus          uint8  // USB bus number
	Address      uint8  // device address on the bus
	VID, PID     uint16 // vendor / product ID
	Serial       string // iSerialNumber, may be empty
	Manufacturer string // iManufacturer, may be empty
	Product      string // iProduct, may be empty
	// Path is the implementation-defined locator (e.g. /dev/bus/usb/001/007
	// on Linux). Treat as opaque; pass back to [Enumerator.Open] verbatim.
	Path string
}

// Transport is a claimed handle on a single USB device. Methods are not
// goroutine-safe by default — the RTL-SDR driver serializes control
// transfers through its own mutex — except that [Transport.StopBulkIn]
// must be safe to call while a foreign goroutine is blocked inside the
// bulk-IN reaper.
type Transport interface {
	// ControlIn issues a vendor-IN control transfer and returns up to n
	// bytes of response. The returned slice's length equals the number
	// of bytes the device actually delivered (≤ n).
	ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error)

	// ControlOut issues a vendor-OUT control transfer.
	ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error

	// ClaimInterface tells the kernel/driver that this transport owns
	// the given USB interface number. Must be called before any I/O.
	ClaimInterface(num int) error

	// ReleaseInterface relinquishes the interface; the device stays
	// open. Idempotent; calling on an unclaimed interface returns nil.
	ReleaseInterface(num int) error

	// StartBulkIn submits a ring of `ringBufs` bulk-IN URBs of `bufLen`
	// bytes each on endpoint epAddr (e.g. 0x81 for RTL-SDR's IQ stream).
	// onPacket is invoked from a dedicated reaper goroutine each time a
	// URB completes; the slice passed to it is owned by the transport
	// and reused once onPacket returns — callers must copy any bytes
	// they wish to retain.
	//
	// onStreamDead, when non-nil, is invoked exactly once when every URB
	// has died of an unrecoverable USB error (host controller hang,
	// ENODEV, EPROTO storm under load, ...) — i.e. the stream ended
	// **without** StopBulkIn being called. Implementations must not
	// invoke it on the normal StopBulkIn path, and must dispatch the
	// callback from a fresh goroutine — never from the reaper itself,
	// because the callback typically calls StopBulkIn (to close the
	// driver's consumer channel) and StopBulkIn waits for the reaper to
	// exit. Drivers use this hook to surface "stream died" to their IQ
	// consumer; without it, a silent reaper exit leaves the consumer
	// blocked forever (issue #345).
	StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte), onStreamDead func(error)) error

	// StopBulkIn cancels every in-flight URB, drains the reaper, and
	// returns once the goroutine has exited. Safe to call concurrently
	// with reaper execution; idempotent.
	StopBulkIn() error

	// Reset performs a USB port reset. The device re-enumerates with
	// the same address (best effort) but configuration is lost.
	Reset() error

	// Close releases the device handle. Implies StopBulkIn and
	// ReleaseInterface for any still-held interfaces.
	Close() error
}

// Enumerator discovers and opens USB devices on the host. One instance
// represents one platform backend; obtained via [DefaultEnumerator].
type Enumerator interface {
	// Name identifies the backend (e.g. "usbdevfs", "winusb", "iokit",
	// "mock"). Used in error messages and logs.
	Name() string

	// List returns every device that matches both vid and pid. A zero
	// value for either acts as a wildcard.
	List(vid, pid uint16) ([]Descriptor, error)

	// Open claims a device identified by a Descriptor previously
	// returned from List.
	Open(d Descriptor) (Transport, error)
}

// Sentinel errors. Implementations may wrap these or return their own
// types; callers compare with errors.Is.
var (
	// ErrUnsupportedPlatform is returned by [DefaultEnumerator] when the
	// running OS has no USB backend compiled in.
	ErrUnsupportedPlatform = errors.New("usb: backend not implemented on this platform")

	// ErrBulkActive is returned by StartBulkIn when a stream is already
	// running on the transport.
	ErrBulkActive = errors.New("usb: bulk-IN already active")

	// ErrBulkInactive is returned by StopBulkIn when no stream is
	// running.
	ErrBulkInactive = errors.New("usb: bulk-IN not active")

	// ErrDeviceGone is returned by any operation after the device has
	// been physically removed or the kernel has decided it's dead.
	// Maps to ENODEV on Linux and ERROR_GEN_FAILURE on Windows.
	ErrDeviceGone = errors.New("usb: device disconnected")

	// ErrTimeout is returned when a control transfer didn't complete
	// within the supplied timeoutMs window.
	ErrTimeout = errors.New("usb: transfer timed out")

	// ErrClosed is returned by methods invoked after Close.
	ErrClosed = errors.New("usb: transport closed")
)

// DefaultEnumerator returns the platform's USB backend. On platforms
// without a backend it returns one whose every method yields
// [ErrUnsupportedPlatform]; this keeps higher layers compilable across
// the OS matrix while the driver is iterated PR-by-PR.
func DefaultEnumerator() Enumerator { return platformEnumerator() }

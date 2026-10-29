package usb

import (
	"fmt"
	"os"
)

// IOKit status codes and their translation to this package's sentinel
// errors. They live in a file with no build tag, although only the macOS
// transport (usb_darwin*.go) produces them, so the mapping is unit-tested
// on every CI runner, not only on usb-macos: the mapping is pure Go, and a
// kern_return that falls through it as a raw number is exactly the class of
// bug that hid the #1038 stall and the #1135 aborts and timeouts.

// kIOReturn (kern_return_t) status codes. We only check for
// kIOReturnSuccess on the hot path; specific errors get translated
// in translateIOReturn below.
const (
	kIOReturnSuccess       = 0
	kIOReturnNoDevice      = 0xE00002C0
	kIOReturnAborted       = 0xE00002EB
	kIOReturnTimeout       = 0xE00002D6
	kIOReturnExclusive     = 0xE00002C5 // already opened by another process
	kIOReturnNotResponding = 0xE00002ED

	// kIOUSBPipeStalled is the USB-family IOReturn the host controller
	// raises when the device STALLs a pipe — "pipe has stalled, error
	// needs to be cleared". Value from IOKit/usb/IOUSBLib.h:
	// iokit_usb_err(0x4f) = sys_iokit(0xE0000000) | sub_iokit_usb(0x4000)
	// | 0x4f = 0xE000404F. This is the macOS analog of the recoverable
	// I²C-bridge stall that surfaces as syscall.EPIPE on Linux and
	// ERROR_GEN_FAILURE on Windows/WinUSB; translateIOReturn maps all
	// three to usb.ErrPipeStalled so the R820T cold-boot burst-write
	// recovery (issue #248) fires identically across the OS matrix
	// (issue #1038: NESDR/R820T tuner-init failed on macOS because this
	// stall fell through as a raw kern_return the retry couldn't match).
	kIOUSBPipeStalled = 0xE000404F

	// kIOUSBTransactionTimeout is iokit_usb_err(0x51) = 0xE0004051, the
	// USB family's "transaction timed out" (IOKit/usb/USB.h). It is what a
	// control DeviceRequest returns when the device never completes it,
	// the macOS analog of Linux ETIMEDOUT and WinUSB ERROR_SEM_TIMEOUT.
	// It is NOT the common kIOReturnTimeout (0xE00002D6), so mapping only
	// that one let it fall through as a raw kern_return (issue #1135,
	// 29 Oct: the daemon open failed on "IOKit kern_return 0xe0004051"
	// without the bring-up envelope ever retrying it).
	kIOUSBTransactionTimeout = 0xE0004051
)

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
	case kIOUSBTransactionTimeout:
		// Keep the raw code: it names the USB-family timeout for the next
		// field log, while errors.Is still matches the shared sentinel the
		// bring-up envelope and runtime recovery key off.
		return fmt.Errorf("%w (IOKit kIOUSBTransactionTimeout 0x%08x)", ErrTimeout, uint32(rc))
	case kIOUSBPipeStalled:
		// Recoverable pipe STALL — same class as Linux EPIPE / Windows
		// ERROR_GEN_FAILURE. Surfacing the shared sentinel lets the
		// tuner burst-write retry (isI2CBurstStall) recover on macOS.
		return ErrPipeStalled
	case kIOReturnAborted:
		// A request the host stack aborted mid-flight. On the bulk-IN
		// path this is the normal StopBulkIn/AbortPipe cancellation, but
		// bulkLoop reads the raw rc there and never routes through here —
		// so every translateIOReturn caller is a control / interface /
		// reset op. For those an abort is a transient the bring-up
		// envelope can reset-and-retry, so surface the dedicated
		// ErrTransferAborted rather than ErrBulkInactive, whose meaning is
		// "StopBulkIn called with no active stream" (issue #1135 — the
		// abort used to be mislabelled "usb: bulk-IN not active" on a
		// control DeviceRequest, and was not treated as resetable).
		return ErrTransferAborted
	case kIOReturnExclusive:
		return fmt.Errorf("usb: device already opened by another process (kIOReturnExclusive)")
	default:
		return fmt.Errorf("usb: IOKit kern_return 0x%08x", uint32(rc))
	}
}

// deviceRequestTOEnv names the opt-out for the macOS DeviceRequestTO
// control path: GT_USB_DEVICE_REQUEST_TO=0 makes Open use the base device
// interface and its timeout-less DeviceRequest, as before issue #1135.
const deviceRequestTOEnv = "GT_USB_DEVICE_REQUEST_TO"

// deviceRequestTOEnabled reports whether the macOS transport should ask
// for the v182 device interface and send control requests with a timeout.
// On by default; only an explicit "0" turns it off.
func deviceRequestTOEnabled() bool {
	return os.Getenv(deviceRequestTOEnv) != "0"
}

// controlTimeoutMs converts a Transport control-call timeout to the
// DeviceRequestTO no-data / completion timeout. A non-positive timeout
// means "no per-call bound" and keeps the plain DeviceRequest path.
func controlTimeoutMs(timeoutMs int) (uint32, bool) {
	if timeoutMs <= 0 {
		return 0, false
	}
	return uint32(timeoutMs), true
}

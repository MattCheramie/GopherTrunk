package usb

import (
	"errors"
	"strings"
	"testing"
)

// TestTranslateIOReturn_TransactionTimeoutMapsToErrTimeout pins the #1135
// follow-up (29 Oct, macOS 27.2, v1.3.5): with two RTL-SDRs on an Apple
// Silicon Mac the daemon's open failed on the very first bring-up write with
//
//	rtlsdr: init baseband: init baseband step 0: rtl2832u: write block=1
//	addr=0x2000 val=0x0009: usb: DeviceRequest OUT: usb: IOKit kern_return 0xe0004051
//
// 0xe0004051 is IOUSBFamily's kIOUSBTransactionTimeout (iokit_usb_err(0x51)),
// a control transfer that timed out, not an access conflict. It fell through
// translateIOReturn as a raw number, so isBringupResetable did not recognise
// it and the open gave up on a transient that the reset+retry envelope
// handles on every other OS (Linux ETIMEDOUT, WinUSB ERROR_SEM_TIMEOUT).
func TestTranslateIOReturn_TransactionTimeoutMapsToErrTimeout(t *testing.T) {
	got := translateIOReturn(0xE0004051)
	if !errors.Is(got, ErrTimeout) {
		t.Fatalf("translateIOReturn(0xe0004051) = %v, want errors.Is(err, ErrTimeout) so the bring-up envelope resets and retries it", got)
	}
	if !strings.Contains(got.Error(), "0xe0004051") {
		t.Errorf("translateIOReturn(0xe0004051) = %q, want the raw kern_return kept in the text for the next field log", got)
	}
}

// TestIOKitStatusValues guards the constants against typos: each fix hinges
// on matching the kern_return IOKit actually returns. iokit_usb_err(x) is
// sys_iokit (0xE0000000) | sub_iokit_usb (0x4000) | x (IOKit/usb/USB.h);
// the common codes are sys_iokit | sub_iokit_common (0) | x (IOReturn.h).
func TestIOKitStatusValues(t *testing.T) {
	for _, tc := range []struct {
		name      string
		got, want uint32
	}{
		{"kIOReturnTimeout", kIOReturnTimeout, 0xE00002D6},
		{"kIOUSBTransactionTimeout", kIOUSBTransactionTimeout, 0xE0000000 | 0x4000 | 0x51},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = 0x%08x, want 0x%08x", tc.name, tc.got, tc.want)
		}
	}
}

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

// TestControlTimeoutMs pins the DeviceRequestTO timeout conversion (issue
// #1135): a positive per-call timeout is passed through unchanged (the
// rtl2832u control transfers use 300 ms, as librtlsdr's CTRL_TIMEOUT), and
// a non-positive one keeps the plain DeviceRequest path.
func TestControlTimeoutMs(t *testing.T) {
	if ms, ok := controlTimeoutMs(300); !ok || ms != 300 {
		t.Errorf("controlTimeoutMs(300) = %d, %v; want 300, true", ms, ok)
	}
	for _, in := range []int{0, -1} {
		if _, ok := controlTimeoutMs(in); ok {
			t.Errorf("controlTimeoutMs(%d) ok = true, want false (no per-call bound)", in)
		}
	}
}

// TestDeviceRequestTOEnabled pins the opt-out: on by default, off only for
// an explicit GT_USB_DEVICE_REQUEST_TO=0.
func TestDeviceRequestTOEnabled(t *testing.T) {
	for _, tc := range []struct {
		val  string
		want bool
	}{{"", true}, {"1", true}, {"0", false}} {
		t.Setenv(deviceRequestTOEnv, tc.val)
		if got := deviceRequestTOEnabled(); got != tc.want {
			t.Errorf("%s=%q: deviceRequestTOEnabled() = %v, want %v", deviceRequestTOEnv, tc.val, got, tc.want)
		}
	}
}

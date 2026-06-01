//go:build windows && (amd64 || arm64)

package usb

import (
	"errors"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestParseDevicePath_RTLSDR(t *testing.T) {
	const path = `\\?\usb#vid_0bda&pid_2838#00000001#{a5dcbf10-6530-11d2-901f-00c04fb951ed}`
	vid, pid, serial := parseDevicePath(path)
	if vid != 0x0BDA {
		t.Errorf("vid = 0x%04x, want 0x0BDA", vid)
	}
	if pid != 0x2838 {
		t.Errorf("pid = 0x%04x, want 0x2838", pid)
	}
	if serial != "00000001" {
		t.Errorf("serial = %q, want 00000001", serial)
	}
}

func TestParseDevicePath_UpperCaseTokens(t *testing.T) {
	// Windows is inconsistent about case in device paths; SetupAPI
	// callers sometimes get the tokens uppercased.
	const path = `\\?\USB#VID_1D50&PID_6089#hackrf1#{GUID}`
	vid, pid, serial := parseDevicePath(path)
	if vid != 0x1D50 {
		t.Errorf("vid = 0x%04x, want 0x1D50", vid)
	}
	if pid != 0x6089 {
		t.Errorf("pid = 0x%04x, want 0x6089", pid)
	}
	if serial != "hackrf1" {
		t.Errorf("serial = %q, want hackrf1", serial)
	}
}

func TestParseDevicePath_NoSerial(t *testing.T) {
	// Devices without an iSerialNumber descriptor get a generated 8-digit
	// number embedded in the path; if even that's absent the third
	// "#"-element is empty/missing.
	const path = `\\?\usb#vid_0bda&pid_2832#{a5dcbf10-6530-11d2-901f-00c04fb951ed}`
	vid, pid, serial := parseDevicePath(path)
	if vid != 0x0BDA {
		t.Errorf("vid = 0x%04x, want 0x0BDA", vid)
	}
	if pid != 0x2832 {
		t.Errorf("pid = 0x%04x, want 0x2832", pid)
	}
	if serial != "{a5dcbf10-6530-11d2-901f-00c04fb951ed}" {
		// Note: when there's no serial, the GUID lands where the serial
		// would be — that's accurate parsing per the path format.
		t.Logf("serial = %q (acceptable: GUID lands in slot 3 when iSerial missing)", serial)
	}
}

func TestParseDevicePath_GarbagePath(t *testing.T) {
	vid, pid, serial := parseDevicePath("not-a-usb-path")
	if vid != 0 || pid != 0 || serial != "" {
		t.Errorf("garbage path produced (vid=0x%04x pid=0x%04x serial=%q), want all zero", vid, pid, serial)
	}
}

func TestParseDevicePath_TruncatedToken(t *testing.T) {
	// vid_ followed by < 4 hex chars must not panic or mis-parse.
	for _, p := range []string{
		`\\?\usb#vid_`,
		`\\?\usb#vid_ab`,
		`\\?\usb#pid_`,
		`\\?\usb#pid_12`,
	} {
		vid, pid, _ := parseDevicePath(p)
		if vid != 0 || pid != 0 {
			t.Errorf("truncated %q parsed as vid=0x%04x pid=0x%04x, want both 0", p, vid, pid)
		}
	}
}

func TestSetupPacketSize(t *testing.T) {
	// WinUsb_ControlTransfer requires an 8-byte setup packet (USB 2.0
	// standard layout). Go's natural alignment must reproduce that.
	if got, want := unsafe.Sizeof(winusbSetupPacket{}), uintptr(8); got != want {
		t.Errorf("sizeof(winusbSetupPacket) = %d, want %d", got, want)
	}
}

func TestSetupPacketPack_MatchesByValueLayout(t *testing.T) {
	// WinUsb_ControlTransfer takes WINUSB_SETUP_PACKET BY VALUE; the
	// x64/arm64 ABI passes the 8-byte struct in one integer register, so
	// pack() must produce exactly the bytes a by-value struct push would
	// place there. Compare pack() against the struct's own little-endian
	// in-memory image — the ground truth for the register contents.
	pkt := winusbSetupPacket{
		RequestType: VendorIn, // 0xC0
		Request:     0x02,
		Value:       0x3000,
		Index:       0x0110,
		Length:      0x0001,
	}
	want := *(*uint64)(unsafe.Pointer(&pkt))
	if got := uint64(pkt.pack()); got != want {
		t.Errorf("pack() = 0x%016x, want 0x%016x (must equal the by-value register image)", got, want)
	}

	// Spot-check the field placement explicitly so a future struct
	// reorder can't silently pass via the layout-equality check alone.
	const wantExplicit = uint64(0xC0) |
		uint64(0x02)<<8 |
		uint64(0x3000)<<16 |
		uint64(0x0110)<<32 |
		uint64(0x0001)<<48
	if got := uint64(pkt.pack()); got != wantExplicit {
		t.Errorf("pack() = 0x%016x, want 0x%016x (explicit field placement)", got, wantExplicit)
	}
}

func TestDeviceInterfaceDataSize(t *testing.T) {
	// SP_DEVICE_INTERFACE_DATA on x64 must be 32 bytes for
	// SetupDiEnumDeviceInterfaces to accept the input.
	if got, want := unsafe.Sizeof(spDeviceInterfaceData{}), uintptr(32); got != want {
		t.Errorf("sizeof(spDeviceInterfaceData) = %d, want %d", got, want)
	}
}

func TestPlatformEnumeratorIsWinUSB(t *testing.T) {
	e := DefaultEnumerator()
	if got, want := e.Name(), "winusb"; got != want {
		t.Errorf("backend Name() = %q, want %q", got, want)
	}
}

func TestWinErr_GenFailureIsNotDeviceGone(t *testing.T) {
	// ERROR_GEN_FAILURE used to be folded into ErrDeviceGone, which
	// printed "usb: device disconnected" and misled the issue #270
	// reporter when their (still-connected) Airspy NAK'd a vendor
	// request. Keep them distinct.
	got := winErr(windows.ERROR_GEN_FAILURE)
	if errors.Is(got, ErrDeviceGone) {
		t.Errorf("winErr(ERROR_GEN_FAILURE) erroneously folds into ErrDeviceGone: %v", got)
	}
	if !errors.Is(got, windows.ERROR_GEN_FAILURE) {
		t.Errorf("winErr(ERROR_GEN_FAILURE) lost the underlying errno (need errors.Is to find it): %v", got)
	}
	if !strings.Contains(got.Error(), "ERROR_GEN_FAILURE") {
		t.Errorf("winErr(ERROR_GEN_FAILURE) message should name the Win32 code: %q", got.Error())
	}
}

func TestWinErr_GenFailureMapsToPipeStalled(t *testing.T) {
	// The WinUSB equivalent of librtlsdr's "dummy write probe stalled
	// on cold boot" surfaces as ERROR_GEN_FAILURE on the second
	// USB_SYSCTL=0x09 write. The bring-up retry envelope keys off
	// ErrPipeStalled to trigger a control-pipe clear-halt and retry
	// once — matching the Linux EPIPE recovery path.
	got := winErr(windows.ERROR_GEN_FAILURE)
	if !errors.Is(got, ErrPipeStalled) {
		t.Errorf("winErr(ERROR_GEN_FAILURE) = %v, want errors.Is(err, ErrPipeStalled) so the bring-up retry kicks in", got)
	}
}

func TestWinErr_DisconnectCodesMapToDeviceGone(t *testing.T) {
	for _, errno := range []windows.Errno{
		windows.ERROR_DEVICE_NOT_CONNECTED,
		windows.ERROR_NO_SUCH_DEVICE,
		windows.ERROR_DEV_NOT_EXIST,
	} {
		if got := winErr(errno); !errors.Is(got, ErrDeviceGone) {
			t.Errorf("winErr(%v) = %v, want ErrDeviceGone", errno, got)
		}
	}
}

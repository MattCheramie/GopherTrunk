package usb

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// debugUSBEnv is the environment variable that toggles the debug
// transport wrapper. Anything non-empty enables it.
const debugUSBEnv = "RTLSDR_DEBUG_USB"

// debugMaxDataBytes caps how many payload bytes the debug log dumps
// per ControlOut. 64 bytes is enough to capture the R820T 27-byte
// init burst (and the chunked variant) in full while keeping log
// lines manageable when callers stream larger blobs.
const debugMaxDataBytes = 64

// debugSink is the io.Writer the debug transport logs to. Defaults
// to os.Stderr; tests swap it via SetDebugSink.
var (
	debugSinkMu sync.Mutex
	debugSink   io.Writer = os.Stderr
)

// SetDebugSink redirects debug-transport output. Returns the previous
// sink so tests can restore it. Intended for tests only.
func SetDebugSink(w io.Writer) io.Writer {
	debugSinkMu.Lock()
	defer debugSinkMu.Unlock()
	prev := debugSink
	debugSink = w
	return prev
}

// debugUSBEnabled reports whether RTLSDR_DEBUG_USB is set in the
// environment. Public-package helpers like the macOS IOKit enumerator
// gate per-iteration FFI calls (e.g. IOObjectGetClass) on this so the
// off-path stays free.
func debugUSBEnabled() bool { return os.Getenv(debugUSBEnv) != "" }

// debugLogf writes one line to the debug sink with the standard
// rtlsdr-usb prefix and a component label, when RTLSDR_DEBUG_USB is
// set. Free function so enumerator paths (not just the per-device
// transport wrapper) can emit traces — used by the macOS IOKit
// enumerator to surface service-iteration failures that would
// otherwise vanish into a silently empty sdr list (issue #257).
func debugLogf(component, format string, args ...interface{}) {
	if !debugUSBEnabled() {
		return
	}
	debugSinkMu.Lock()
	w := debugSink
	debugSinkMu.Unlock()
	fmt.Fprintln(w, "rtlsdr-usb ["+component+"]: "+fmt.Sprintf(format, args...))
}

// MaybeWrapDebug returns t wrapped in a debug-logging Transport when
// RTLSDR_DEBUG_USB is set in the environment; otherwise it returns t
// unchanged. Wrapping is gated at Open time so the rest of the code
// path stays unaffected when debugging is off.
//
// The wrapped transport logs every ControlIn/ControlOut/Reset call
// to stderr (or the sink set via SetDebugSink) in a format that's
// diffable against `LIBUSB_DEBUG=4` traces from osmocom librtlsdr's
// rtl_test:
//
//	rtlsdr-usb: ControlOut bmReqType=0x40 bReq=0x00 wValue=0x0034 wIndex=0x0610 wLength=17 timeout=300ms
//	rtlsdr-usb:   data=83 32 75 c0 ...
//	rtlsdr-usb:   → ok (after 1.2ms)
//
// The label embeds the descriptor's bus/address (or serial when
// available) so multi-dongle traces remain attributable.
func MaybeWrapDebug(t Transport, desc Descriptor) Transport {
	if os.Getenv(debugUSBEnv) == "" {
		return t
	}
	return newDebugTransport(t, desc)
}

// debugTransport wraps another [Transport], logging each control
// transfer + Reset before delegating. Bulk-IN is left untouched —
// the sample stream is the hot path and would flood stderr if
// per-packet hooks were added.
type debugTransport struct {
	inner Transport
	label string
}

func newDebugTransport(inner Transport, desc Descriptor) *debugTransport {
	label := fmt.Sprintf("bus=%d addr=%d", desc.Bus, desc.Address)
	if desc.Serial != "" {
		label += " serial=" + desc.Serial
	}
	return &debugTransport{inner: inner, label: label}
}

func (d *debugTransport) writeLog(line string) {
	debugSinkMu.Lock()
	w := debugSink
	debugSinkMu.Unlock()
	fmt.Fprintln(w, line)
}

func (d *debugTransport) logf(format string, args ...interface{}) {
	d.writeLog("rtlsdr-usb [" + d.label + "]: " + fmt.Sprintf(format, args...))
}

func (d *debugTransport) ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error) {
	d.logf("ControlIn  bmReqType=0x%02x bReq=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d timeout=%dms",
		VendorIn, bRequest, wValue, wIndex, n, timeoutMs)
	start := time.Now()
	out, err := d.inner.ControlIn(bRequest, wValue, wIndex, n, timeoutMs)
	dur := time.Since(start)
	if err != nil {
		d.logf("  -> err=%v (after %s)", err, dur)
		return out, err
	}
	d.logf("  -> data=%s (%d bytes, %s)", hexBytes(out, debugMaxDataBytes), len(out), dur)
	return out, nil
}

func (d *debugTransport) ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	d.logf("ControlOut bmReqType=0x%02x bReq=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d timeout=%dms",
		VendorOut, bRequest, wValue, wIndex, len(data), timeoutMs)
	if len(data) > 0 {
		d.logf("  data=%s", hexBytes(data, debugMaxDataBytes))
	}
	start := time.Now()
	err := d.inner.ControlOut(bRequest, wValue, wIndex, data, timeoutMs)
	dur := time.Since(start)
	if err != nil {
		d.logf("  -> err=%v (after %s)", err, dur)
		return err
	}
	d.logf("  -> ok (after %s)", dur)
	return nil
}

func (d *debugTransport) ClaimInterface(num int) error {
	d.logf("ClaimInterface(%d)", num)
	return d.inner.ClaimInterface(num)
}

func (d *debugTransport) ReleaseInterface(num int) error {
	d.logf("ReleaseInterface(%d)", num)
	return d.inner.ReleaseInterface(num)
}

func (d *debugTransport) StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte)) error {
	d.logf("StartBulkIn ep=0x%02x ring=%d bufLen=%d", epAddr, ringBufs, bufLen)
	return d.inner.StartBulkIn(epAddr, ringBufs, bufLen, onPacket)
}

func (d *debugTransport) StopBulkIn() error {
	d.logf("StopBulkIn")
	return d.inner.StopBulkIn()
}

func (d *debugTransport) Reset() error {
	d.logf("Reset (USBDEVFS_RESET)")
	return d.inner.Reset()
}

func (d *debugTransport) Close() error {
	d.logf("Close")
	return d.inner.Close()
}

// hexBytes returns a space-separated lowercase hex dump of the first
// max bytes of b, with a "... (N more)" suffix when truncated.
func hexBytes(b []byte, max int) string {
	if len(b) == 0 {
		return "(empty)"
	}
	n := len(b)
	truncated := false
	if n > max {
		n = max
		truncated = true
	}
	var sb strings.Builder
	sb.Grow(n*3 + 16)
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(' ')
		}
		const hex = "0123456789abcdef"
		sb.WriteByte(hex[b[i]>>4])
		sb.WriteByte(hex[b[i]&0x0f])
	}
	if truncated {
		fmt.Fprintf(&sb, " ... (%d more)", len(b)-max)
	}
	return sb.String()
}

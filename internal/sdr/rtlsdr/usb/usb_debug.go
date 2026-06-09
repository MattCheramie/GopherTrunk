package usb

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// debugUSBEnv is the environment variable that toggles the debug
// transport wrapper. Anything non-empty enables it.
const debugUSBEnv = "RTLSDR_DEBUG_USB"

// debugUSBCSVEnv enables compact CSV-style lines alongside human-readable
// traces. Useful for diffing against USBPcap/Wireshark exports.
const debugUSBCSVEnv = "RTLSDR_DEBUG_USB_CSV"

// debugUSBCSVHeader is emitted once per process when CSV mode is enabled.
//
// Column order is fixed for script-friendly parsing:
//  1. tx_id           monotonically increasing transfer sequence number
//  2. ts_utc          RFC3339Nano timestamp in UTC
//  3. rel             elapsed time since process-local trace epoch
//  4. phase           request | payload | result
//  5. dir             in | out
//  6. bmReqType       hex or empty
//  7. bRequest        hex or empty
//  8. wValue          hex or empty
//  9. wIndex          hex or empty
//
// 10) wLength         decimal or empty
// 11) timeoutMs       decimal or empty
// 12) status          ok | err or empty
// 13) dataLen         decimal or empty
// 14) payloadHex      quoted hex dump or empty
// 15) err             quoted error text or empty
// 16) duration        elapsed op duration string or empty
const debugUSBCSVHeader = "tx_id,ts_utc,rel,phase,dir,bmReqType,bRequest,wValue,wIndex,wLength,timeoutMs,status,dataLen,payloadHex,err,duration"

// Diagnoser is an optional Transport capability. Implementations that
// can describe the underlying device — the bound driver, USB descriptors,
// a control-IN read probe — expose Diagnostics. The bring-up path
// (purego/driver.go) appends this to a failed Open so a single probe run
// captures everything needed to triage a dongle that rejects control
// transfers. The Windows transport implements it; transports that don't
// simply contribute no extra diagnostics.
type Diagnoser interface {
	Diagnostics() string
}

// debugMaxDataBytes caps how many payload bytes the debug log dumps
// per ControlOut. 64 bytes is enough to capture the R820T 27-byte
// init burst (and the chunked variant) in full while keeping log
// lines manageable when callers stream larger blobs.
const debugMaxDataBytes = 64

// debugSink is the io.Writer the debug transport logs to. Defaults
// to os.Stderr; tests swap it via SetDebugSink.
var (
	debugSinkMu           sync.Mutex
	debugSink             io.Writer = os.Stderr
	debugTrace0                     = time.Now()
	debugTraceSeq         atomic.Uint64
	debugCSVHeaderEmitted atomic.Bool
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

func debugUSBCSVEnabled() bool { return os.Getenv(debugUSBCSVEnv) != "" }

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

// DebugLogf is the exported entry point to the gated debug sink for
// callers outside this package (e.g. the rtlsdr bring-up path in
// purego/driver.go, which logs the swallowed sacrificial-warmup
// failure). Like debugLogf it only emits when RTLSDR_DEBUG_USB is set,
// writes to the same sink (so SetDebugSink works in tests), and uses
// the same "rtlsdr-usb [component]" line prefix — keeping every USB
// diagnostic on one diffable channel.
func DebugLogf(component, format string, args ...interface{}) {
	debugLogf(component, format, args...)
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

func (d *debugTransport) logCSV(format string, args ...interface{}) {
	if !debugUSBCSVEnabled() {
		return
	}
	if !debugCSVHeaderEmitted.Load() {
		if debugCSVHeaderEmitted.CompareAndSwap(false, true) {
			d.writeLog("rtlsdr-usb-csv,#" + debugUSBCSVHeader)
		}
	}
	d.writeLog("rtlsdr-usb-csv," + fmt.Sprintf(format, args...))
}

func nextDebugTrace() (uint64, time.Time, time.Duration) {
	now := time.Now()
	seq := debugTraceSeq.Add(1)
	delta := now.Sub(debugTrace0).Round(time.Microsecond)
	return seq, now, delta
}

func (d *debugTransport) ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error) {
	txID, ts, rel := nextDebugTrace()
	d.logf("tx=%06d ts=%s rel=%s ControlIn  bmReqType=0x%02x bReq=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d timeout=%dms",
		txID, ts.UTC().Format(time.RFC3339Nano), rel, VendorIn, bRequest, wValue, wIndex, n, timeoutMs)
	d.logCSV("%06d,%s,%s,request,in,0x%02x,0x%02x,0x%04x,0x%04x,%d,%d,,,,,", txID, ts.UTC().Format(time.RFC3339Nano), rel, VendorIn, bRequest, wValue, wIndex, n, timeoutMs)
	start := time.Now()
	out, err := d.inner.ControlIn(bRequest, wValue, wIndex, n, timeoutMs)
	dur := time.Since(start)
	if err != nil {
		d.logf("tx=%06d -> err=%v (after %s)", txID, err, dur)
		d.logCSV("%06d,%s,%s,result,in,,,,,,,err,,,%q,%s", txID, ts.UTC().Format(time.RFC3339Nano), rel, err.Error(), dur)
		return out, err
	}
	d.logf("tx=%06d -> data=%s (%d bytes, %s)", txID, hexBytes(out, debugMaxDataBytes), len(out), dur)
	d.logCSV("%06d,%s,%s,result,in,,,,,,,ok,%d,%q,,%s", txID, ts.UTC().Format(time.RFC3339Nano), rel, len(out), hexBytes(out, debugMaxDataBytes), dur)
	return out, nil
}

func (d *debugTransport) ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	txID, ts, rel := nextDebugTrace()
	d.logf("tx=%06d ts=%s rel=%s ControlOut bmReqType=0x%02x bReq=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d timeout=%dms",
		txID, ts.UTC().Format(time.RFC3339Nano), rel, VendorOut, bRequest, wValue, wIndex, len(data), timeoutMs)
	d.logCSV("%06d,%s,%s,request,out,0x%02x,0x%02x,0x%04x,0x%04x,%d,%d,,,,,", txID, ts.UTC().Format(time.RFC3339Nano), rel, VendorOut, bRequest, wValue, wIndex, len(data), timeoutMs)
	if len(data) > 0 {
		d.logf("tx=%06d data=%s", txID, hexBytes(data, debugMaxDataBytes))
		d.logCSV("%06d,%s,%s,payload,out,,,,,,,,,%q,,", txID, ts.UTC().Format(time.RFC3339Nano), rel, hexBytes(data, debugMaxDataBytes))
	}
	start := time.Now()
	err := d.inner.ControlOut(bRequest, wValue, wIndex, data, timeoutMs)
	dur := time.Since(start)
	if err != nil {
		d.logf("tx=%06d -> err=%v (after %s)", txID, err, dur)
		d.logCSV("%06d,%s,%s,result,out,,,,,,,err,,,%q,%s", txID, ts.UTC().Format(time.RFC3339Nano), rel, err.Error(), dur)
		return err
	}
	d.logf("tx=%06d -> ok (after %s)", txID, dur)
	d.logCSV("%06d,%s,%s,result,out,,,,,,,ok,,,,%s", txID, ts.UTC().Format(time.RFC3339Nano), rel, dur)
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

func (d *debugTransport) StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte), onStreamDead func(error)) error {
	d.logf("StartBulkIn ep=0x%02x ring=%d bufLen=%d", epAddr, ringBufs, bufLen)
	wrappedDead := onStreamDead
	if onStreamDead != nil {
		wrappedDead = func(err error) {
			d.logf("onStreamDead: %v", err)
			onStreamDead(err)
		}
	}
	return d.inner.StartBulkIn(epAddr, ringBufs, bufLen, onPacket, wrappedDead)
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

// Diagnostics forwards to the wrapped transport's Diagnostics when it
// implements Diagnoser, so the RTLSDR_DEBUG_USB wrapper doesn't hide the
// rich device dump the bring-up path appends on failure.
func (d *debugTransport) Diagnostics() string {
	if dg, ok := d.inner.(Diagnoser); ok {
		return dg.Diagnostics()
	}
	return ""
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

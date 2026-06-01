package purego

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// DriverName is the [sdr.Driver.Name] this backend registers under
// — the canonical "rtlsdr" string downstream config files,
// `gophertrunk sdr list` output, and Prometheus labels expect.
const DriverName = "rtlsdr"

// Driver implements [sdr.Driver]. The optional Enumerator field lets
// tests inject a [usb.MockEnumerator]; production code leaves it nil
// and falls through to [usb.DefaultEnumerator].
type Driver struct {
	Enumerator usb.Enumerator

	// detectCache stores the descriptors returned by the most recent
	// Enumerate so Open(idx) can map index → descriptor without
	// re-scanning sysfs.
	mu          sync.Mutex
	detectCache []usb.Descriptor
}

// New returns a Driver bound to the given enumerator. Pass nil to use
// the default (platform) enumerator.
func New(e usb.Enumerator) *Driver {
	return &Driver{Enumerator: e}
}

// Name returns "rtlsdr" — the canonical name the SDR registry
// exposes this driver under.
func (d *Driver) Name() string { return DriverName }

func (d *Driver) enumerator() usb.Enumerator {
	if d.Enumerator != nil {
		return d.Enumerator
	}
	return usb.DefaultEnumerator()
}

// Enumerate scans the platform for known RTL-SDR dongles and returns
// one [sdr.Info] per matching descriptor. Per the [sdr.Driver]
// contract, the returned slice is stable across calls — Open uses
// the index into it to find the descriptor to claim.
//
// Devices not in [knownDevices] are silently filtered out so we don't
// claim arbitrary USB devices (e.g. mice, keyboards) sharing a vendor
// ID with a supported chipset. New devices show up by adding rows to
// devices.go.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	all, err := d.enumerator().List(0, 0)
	if err != nil {
		return nil, fmt.Errorf("rtlsdr: enumerate: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.detectCache = d.detectCache[:0]
	out := make([]sdr.Info, 0, len(all))
	for _, desc := range all {
		kd := lookupKnown(desc.VID, desc.PID)
		if kd == nil {
			continue
		}
		// Prefer the device-reported product string over our own
		// friendly name when available; falls back to our table for
		// dongles that don't populate iProduct.
		product := desc.Product
		if product == "" {
			product = kd.Name
		}
		out = append(out, sdr.Info{
			Driver:       DriverName,
			Index:        len(d.detectCache),
			Serial:       desc.Serial,
			Manufacturer: desc.Manufacturer,
			Product:      product,
		})
		d.detectCache = append(d.detectCache, desc)
	}
	return out, nil
}

// Open claims the device at the given index (an offset into the
// most-recent [Driver.Enumerate] result), brings up the demod and
// tuner, and returns an [sdr.Device] handle.
//
// Steps:
//  1. Open the USB transport via the enumerator.
//  2. Claim USB interface 0 (RTL-SDR's only interface).
//  3. Wrap the transport in [rtl2832u.Demod] and run a single
//     librtlsdr-parity USB-sysctl warmup write; if it returns EPIPE,
//     reset the device and retry once (fixes issue #248).
//  4. Run the baseband init flood.
//  5. Probe for a tuner (leaves the I2C repeater ON on success).
//  6. For R820T-family tuners, run the four-write demod prep that
//     librtlsdr emits between detect_tuner and tuner->init.
//  7. Run tuner.Init while the I2C repeater is still on.
//  8. Toggle the I2C repeater off explicitly.
//  9. Populate [sdr.Info] with the tuner's name and gain ladder.
//
// If RTLSDR_DEBUG_USB=1 is set in the environment, the transport is
// wrapped with [usb.NewDebugTransport] so every control transfer is
// logged to stderr — diffable against `LIBUSB_DEBUG=4` traces from
// rtl_test.
//
// Failure at any step closes the transport and returns the error.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	d.mu.Lock()
	if idx < 0 || idx >= len(d.detectCache) {
		d.mu.Unlock()
		return nil, fmt.Errorf("rtlsdr: index %d out of range (cache size %d; call Enumerate first)", idx, len(d.detectCache))
	}
	desc := d.detectCache[idx]
	d.mu.Unlock()

	transport, err := d.enumerator().Open(desc)
	if err != nil {
		return nil, fmt.Errorf("rtlsdr: open USB %s: %w%s", desc.Path, err, openUSBHint(err))
	}
	transport = usb.MaybeWrapDebug(transport, desc)

	dev, err := openDevice(transport, desc, idx)
	if err != nil {
		_ = transport.Close()
		return nil, err
	}
	return dev, nil
}

// openDevice runs the full bring-up sequence: claim interface, warm up
// the USB endpoint, init demod, detect tuner, run any tuner-specific
// demod prep, init the tuner, set IF freq. Factored out of Open so
// tests can drive it directly with a mock transport without going
// through the enumerator.
//
// Each step manages its own I²C repeater state. Detect wraps the
// probe sweep in a single on/off pair (off-on-return); PrepareDemod's
// writes are demod control transfers that don't touch the I²C bridge;
// tuner.Init opens the repeater once at the top of its body and
// closes it on return (the per-public-method pattern shared by every
// tuner driver). The fresh on-toggle inside tuner.Init is
// load-bearing on NESDR v5 silicon (issue #248) — the chip needs the
// explicit "kick" to arm the bridge for the multi-byte OUT that
// follows, even though the demod register already holds the on-value.
//
// The whole sequence (InitBaseband onward) is wrapped in a bounded
// reset+retry envelope on EPIPE / ErrDeviceGone / ErrTimeout /
// ErrPipeStalled. The warmup dummy write itself is no longer part of
// this envelope — its failure is swallowed in runBringup (librtlsdr
// parity; resetting on a warmup NAK only re-arms the first-transfer
// NAK on the next pass). The envelope covers the NESDR v5 cold-boot
// I²C-bridge-stall case (issue #248, the chip rejecting the first
// 17-byte tuner burst even after PR #262's fresh wire toggle is on
// the wire), and the Windows clone-dongle wedge where one
// device-rebind isn't enough to clear a stale firmware state — both
// of which surface from InitBaseband or later. Up to four resets per
// Open with exponential backoff
// (200ms / 400ms / 800ms / 1200ms) between the failing attempt and
// the next retry — gives the WinUSB stack and device firmware time
// to settle before the next bring-up pass. Worst-case open delay
// for a permanently-broken dongle is ~2.6s of sleep plus the
// per-reset rebind cost.
//
// Inter-attempt backoff is the wrong tool for the issue #395
// failure mode (warmup OK → step 0 immediately fails with
// ERROR_GEN_FAILURE because the dongle's firmware can't handle the
// back-to-back identical write at WinUSB-syscall speed); that's
// handled inside runBringup via warmupSettleDuration before the
// first attempt even runs.
//
// Non-resetable errors (validation failures, ErrClosed) return
// immediately — reset is the wrong hammer for them.
func openDevice(transport usb.Transport, desc usb.Descriptor, idx int) (*Device, error) {
	if err := transport.ClaimInterface(0); err != nil {
		return nil, fmt.Errorf("rtlsdr: claim interface 0: %w%s", err, claimBusyHint(err))
	}

	const maxAttempts = 5
	// backoff[attempt] is the sleep that runs BEFORE the (attempt+1)th
	// bring-up pass — exponential with a soft cap at 1200ms so a
	// pathologically wedged dongle still surfaces the error in ~3s
	// rather than dragging out into a 10s tail. Indexed 0..maxAttempts-2;
	// the last attempt's failure surfaces immediately with no further
	// sleep (we're out of retries).
	backoffs := [maxAttempts - 1]time.Duration{
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1200 * time.Millisecond,
	}
	var (
		demod *rtl2832u.Demod
		tuner tuners.Tuner
		err   error
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		demod = rtl2832u.New(transport)
		tuner, err = runBringup(demod)
		if err == nil {
			break
		}
		if attempt == maxAttempts-1 || !isBringupResetable(err) {
			return nil, withTransportDiagnostics(transport, err)
		}
		if resetErr := transport.Reset(); resetErr != nil {
			return nil, fmt.Errorf("rtlsdr: bring-up hit %w; reset failed: %w", err, resetErr)
		}
		time.Sleep(backoffs[attempt])
		_ = transport.ReleaseInterface(0)
		if claimErr := transport.ClaimInterface(0); claimErr != nil {
			return nil, fmt.Errorf("rtlsdr: re-claim interface 0 after reset: %w", claimErr)
		}
	}

	kd := lookupKnown(desc.VID, desc.PID)
	product := desc.Product
	biasTeeGPIO := defaultBiasTeeGPIO
	if kd != nil {
		if product == "" {
			product = kd.Name
		}
		if kd.BiasTeeGPIO != 0 {
			biasTeeGPIO = kd.BiasTeeGPIO
		}
	}
	info := sdr.Info{
		Driver:       DriverName,
		Index:        idx,
		Serial:       desc.Serial,
		Manufacturer: desc.Manufacturer,
		Product:      product,
		TunerName:    tuner.Type().String(),
		Gains:        tuner.Gains(),
	}
	return &Device{
		transport:   transport,
		demod:       demod,
		tuner:       tuner,
		info:        info,
		biasTeeGPIO: biasTeeGPIO,
	}, nil
}

// defaultOpenSampleRateHz mirrors librtlsdr's rtlsdr_open: leave the
// chip at a known-good 2.048 MS/s so consumers that forget to program
// the rate before streaming still get coherent IQ instead of whatever
// the resampler's power-on default happens to be.
const defaultOpenSampleRateHz uint32 = 2_048_000

// warmupSettleDuration is the gap inserted between the WarmupUSBSysctl
// probe and the first InitBaseband write — both of which target
// (block=USB, addr=USBSysctl, val=0x09) on the wire. librtlsdr emits
// the same byte-identical pair through libusb on every OS, but
// libusb's WinUSB backend has multiple stack layers (libusb-1.0.dll
// → winusb.dll → WinUSB.sys) and naturally spaces those two writes
// by hundreds of microseconds. GopherTrunk's direct
// procWinUsbControlTransfer.Call path is several layers thinner, so
// on some clone dongles the second identical write arrives while the
// firmware is still processing the first and gets NAK'd with
// ERROR_GEN_FAILURE (issue #395). 10 ms is enough headroom for the
// observed clone firmware to settle without being noticeable on
// healthy hardware. The Linux / macOS USBDEVFS / IOKit paths have
// their own per-transfer overhead, so the cost is functionally only
// paid once per Open on every platform.
//
// It's a var rather than a const so tests can override it; production
// callers must treat it as read-only.
var warmupSettleDuration = 10 * time.Millisecond

// runBringup executes the librtlsdr-parity init sequence on a fresh
// Demod: USB-sysctl warmup probe → warmupSettleDuration sleep →
// baseband init → tuner detect → R820T-family demod prep (no-op for
// other tuners) → tuner.Init → IF-freq programming (R820T-family
// programs its own IF inside PrepareDemod, so non-R82xx tuners use
// the demod-side path) → default sample-rate program (librtlsdr
// parity; see issue #275). Returns the initialized tuner on success.
// All errors are wrapped with a stage prefix so the outer openDevice
// can spot resetable EPIPE / ErrDeviceGone via errors.Is.
func runBringup(demod *rtl2832u.Demod) (tuners.Tuner, error) {
	// librtlsdr's rtlsdr_open issues this USB-sysctl write purely as a
	// throwaway "dummy write": its only job is to absorb the
	// first-control-transfer NAK that some clone dongles (and devices
	// left dirty by a crashed process) emit right after the interface
	// is claimed. librtlsdr never inspects its result — it proceeds
	// straight to rtlsdr_init_baseband. We match that exactly: a warmup
	// failure is EXPECTED on the affected hardware and must NOT abort
	// bring-up, because the transfer that actually has to land is the
	// *next* one (InitBaseband's byte-identical step 0).
	//
	// Treating warmup as a hard gate — reset + re-issue it on failure,
	// as this code did before — re-arms the same first-transfer NAK on
	// every pass (each transport.Reset re-opens the device), so the
	// dongle never reaches InitBaseband and Open fails after exhausting
	// the retry envelope. That's the Windows ERROR_GEN_FAILURE warmup
	// failure reported against the RTL-SDR clone path (follow-up to the
	// issue #395 settle fix). Genuine stalls are still caught: if the
	// device is truly wedged, InitBaseband's step 0 fails too and the
	// outer openDevice envelope resets + retries the whole sequence.
	if err := demod.WarmupUSBSysctl(); err != nil {
		usb.DebugLogf("bringup",
			"sacrificial warmup write failed (expected on some clone dongles / dirty device state), proceeding to InitBaseband: %v", err)
	}
	// The first write inside InitBaseband is byte-identical to the
	// warmup probe — give the dongle's firmware a moment to settle
	// before sending it. See warmupSettleDuration above for the full
	// reasoning (issue #395).
	time.Sleep(warmupSettleDuration)
	if err := demod.InitBaseband(); err != nil {
		return nil, fmt.Errorf("rtlsdr: init baseband: %w%s", err, tunerBringupHint(err))
	}
	tuner, err := tuners.Detect(demod)
	if err != nil {
		return nil, fmt.Errorf("rtlsdr: tuner detect: %w%s", err, tunerBringupHint(err))
	}
	_, isR82xx := tuner.(*tuners.R82xx)
	if isR82xx {
		if err := tuner.(*tuners.R82xx).PrepareDemod(); err != nil {
			return nil, fmt.Errorf("rtlsdr: tuner prep: %w%s", err, tunerBringupHint(err))
		}
	}
	if err := tuner.Init(); err != nil {
		return nil, fmt.Errorf("rtlsdr: tuner init: %w%s", err, tunerBringupHint(err))
	}
	if !isR82xx {
		if err := demod.SetIFFreq(tuner.IFFreqHz()); err != nil {
			return nil, fmt.Errorf("rtlsdr: set IF freq: %w%s", err, tunerBringupHint(err))
		}
	}
	actual, err := demod.SetSampleRate(defaultOpenSampleRateHz)
	if err != nil {
		return nil, fmt.Errorf("rtlsdr: default sample rate: %w%s", err, tunerBringupHint(err))
	}
	if err := tuner.SetBandwidth(actual); err != nil {
		return nil, fmt.Errorf("rtlsdr: default tuner bandwidth: %w%s", err, tunerBringupHint(err))
	}
	return tuner, nil
}

// isBringupResetable reports whether err is one of the four error
// classes a USB port reset is the right hammer for during open-time
// bring-up: EPIPE (Linux/USBDEVFS cold-boot stall, the librtlsdr "dummy
// write probe" case), ErrDeviceGone (the chip dropped off the bus
// mid-bringup), ErrTimeout (Windows/WinUSB cold-boot — same root cause
// as the Linux EPIPE, but WinUsb_ControlTransfer surfaces it as
// ERROR_SEM_TIMEOUT instead of stalling the pipe), or ErrPipeStalled
// (Windows/WinUSB clone-dongle cold-boot — the chip latches the first
// USB_SYSCTL write and then NAKs the next identical write with
// ERROR_GEN_FAILURE). The retry is bounded to four resets per Open
// (200ms + 400ms + 800ms + 1200ms exponential backoff between passes)
// so even if all four resets are wasted on a non-cold-boot stall the
// cost is capped at ~2.6s of sleep before the original error surfaces.
// The previous 100ms/200ms envelope (#393) wasn't long enough for the
// Windows clone-dongle in issue #395 — widening to four resets gives
// the WinUSB stack and firmware enough settle time for the wedged-
// state recovery without penalising healthy opens (zero delay on
// attempt 0). Other errors (ErrClosed, validation failures) stay
// non-resetable.
func isBringupResetable(err error) bool {
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, usb.ErrDeviceGone) ||
		errors.Is(err, usb.ErrTimeout) ||
		errors.Is(err, usb.ErrPipeStalled)
}

// withTransportDiagnostics appends a USB diagnostic dump to err when the
// transport can produce one (the Windows WinUSB transport today, via
// usb.Diagnoser). Gathered once, on the final bring-up failure, so a
// single `gophertrunk sdr list --probe` run captures the bound driver,
// the device + configuration descriptors, and a control-IN read probe —
// the data needed to triage a dongle that rejects control transfers even
// with WinUSB reportedly bound. No-op on transports that don't implement
// Diagnoser or that return an empty dump, so non-Windows error messages
// stay unchanged.
func withTransportDiagnostics(transport usb.Transport, err error) error {
	dg, ok := transport.(usb.Diagnoser)
	if !ok {
		return err
	}
	diag := dg.Diagnostics()
	if diag == "" {
		return err
	}
	return fmt.Errorf("%w\n--- USB diagnostics (gophertrunk; attach this to the issue) ---\n%s------------------------------------------------------------", err, diag)
}

// claimBusyHint returns a parenthesized, space-prefixed remediation
// string when err is an EBUSY that survived the USB transport's
// kernel-driver auto-detach. The transport already detaches the DVB-T
// driver (dvb_usb_rtl28xxu) and retries, so a surviving EBUSY means the
// interface is held by another *user-space* process — a second
// gophertrunk, rtl_test, SDR++, GQRX — which detaching a kernel driver
// cannot clear. Returns "" for unrelated errors so the wrapped message
// stays clean.
func claimBusyHint(err error) string {
	if errors.Is(err, syscall.EBUSY) {
		return " (hint: the dongle is already claimed by another process —" +
			" close any other SDR app holding it (rtl_test, SDR++, GQRX," +
			" or a second gophertrunk instance) and retry." +
			" See https://gophertrunk.org/install-linux.html#troubleshooting)"
	}
	return ""
}

// tunerBringupHint returns a parenthesized, space-prefixed remediation
// string when err looks like one of the two known unrecoverable cold-
// boot classes (issue #248):
//
//   - EPIPE / ErrDeviceGone — the Linux/USBDEVFS cold-boot stall: the
//     tuner did not ACK on the I²C bridge. Common causes: DVB kernel
//     driver still bound, marginal USB power, or a flaky cable/hub.
//
//   - ErrTimeout — the Windows/WinUSB cold-boot equivalent: the device
//     accepted the control transfer but never responded within the
//     CTRL_TIMEOUT window. Common causes: WinUSB driver not bound (the
//     Zadig step was skipped or installed against the wrong device),
//     marginal USB power, or a flaky cable/hub.
//
// Both branches link to the docs troubleshooting table. Returns "" for
// unrelated errors so the wrapped message stays clean.
func tunerBringupHint(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, usb.ErrDeviceGone) {
		return " (hint: tuner did not respond on the I2C bus — common causes:" +
			" DVB kernel driver still bound (run `sudo modprobe -r dvb_usb_rtl28xxu` and re-plug)," +
			" marginal USB power, or a flaky cable/hub." +
			" See https://gophertrunk.org/install-linux.html#troubleshooting)"
	}
	if errors.Is(err, usb.ErrTimeout) {
		return " (hint: USB control transfer timed out — common causes:" +
			" on Windows, the WinUSB driver is not bound to the device (re-run Zadig and select the RTL-SDR);" +
			" on any OS, marginal USB power, a flaky cable/hub, or another process already holding the device." +
			" See https://gophertrunk.org/install-windows.html#troubleshooting)"
	}
	if errors.Is(err, usb.ErrPipeStalled) {
		return " (hint: the dongle's USB control pipe stalled" +
			" (Windows ERROR_GEN_FAILURE) — on Windows, the bring-up path" +
			" already inserts a 10 ms settle between the warmup probe and" +
			" the first InitBaseband write to absorb the clone-dongle" +
			" firmware timing quirk reported in issue #395. If the failure" +
			" persists, try a different USB controller (front vs rear ports" +
			" often use different host controllers) and confirm" +
			" `gophertrunk sdr doctor` reports WinUSB bound to interface 0." +
			" Avoid USB hubs and powered extension cables." +
			" See https://gophertrunk.org/install-windows.html#troubleshooting)"
	}
	return ""
}

// openUSBHint returns a parenthesized, space-prefixed remediation
// string when err looks like a Windows ERROR_ACCESS_DENIED from
// CreateFile — the symptom seen when another user-space process (SDR
// Sharp, AbracaDABra, rtl_test, a second gophertrunk) already holds
// the dongle, or when the same serial is listed twice in sdr.devices
// so the pool tries to open the same physical device for two roles
// (issue #333). Returns "" for unrelated errors so the wrapped
// message stays clean. The platform-specific check lives in
// driver_windows.go / driver_other.go.
func openUSBHint(err error) string {
	if isAccessDenied(err) {
		return " (hint: another process is holding the dongle —" +
			" close any other SDR app (SDR Sharp, AbracaDABra, rtl_test," +
			" or a second gophertrunk instance) and retry." +
			" If only one gophertrunk is running, check that the same serial" +
			" is not listed twice in sdr.devices — one physical SDR cannot" +
			" serve two roles.)"
	}
	return ""
}

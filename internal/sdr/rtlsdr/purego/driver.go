package purego

import (
	"errors"
	"fmt"
	"sync"
	"syscall"

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
		return nil, fmt.Errorf("rtlsdr: open USB %s: %w", desc.Path, err)
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
func openDevice(transport usb.Transport, desc usb.Descriptor, idx int) (*Device, error) {
	if err := transport.ClaimInterface(0); err != nil {
		return nil, fmt.Errorf("rtlsdr: claim interface 0: %w", err)
	}

	demod := rtl2832u.New(transport)
	if err := warmupWithReset(transport, demod); err != nil {
		return nil, err
	}
	if err := demod.InitBaseband(); err != nil {
		return nil, fmt.Errorf("rtlsdr: init baseband: %w", err)
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
			return nil, fmt.Errorf("rtlsdr: set IF freq: %w", err)
		}
	}

	kd := lookupKnown(desc.VID, desc.PID)
	product := desc.Product
	if product == "" && kd != nil {
		product = kd.Name
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
		transport: transport,
		demod:     demod,
		tuner:     tuner,
		info:      info,
	}, nil
}

// warmupWithReset mirrors librtlsdr's rtlsdr_open dummy-write probe: it
// issues one USB-sysctl write to confirm the endpoint is healthy, and
// on EPIPE / ErrDeviceGone it resets the device and retries once before
// giving up. Dongles left half-initialised by a crashed previous
// session (or by a DVB kernel driver that was unbound mid-flight) tend
// to ack control reads fine but stall the first multi-byte OUT — which
// in the pre-fix code path showed up as "r82xx init: burst write:
// I2CWrite addr=0x34: broken pipe" (issue #248). Running the probe
// here means InitBaseband / tuner.Init see a clean device.
//
// USBDEVFS_RESET on Linux invalidates the prior interface claim, so the
// retry path explicitly re-claims interface 0. On macOS, IOKit
// ResetDevice may invalidate the device handle entirely; if the
// re-claim fails we return that error verbatim rather than spinning.
// On Windows, transport.Reset is a no-op — the retry will EPIPE
// identically and we surface the wrapped error with the existing hint.
func warmupWithReset(transport usb.Transport, demod *rtl2832u.Demod) error {
	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := demod.WarmupUSBSysctl()
		if err == nil {
			return nil
		}
		if attempt == maxAttempts-1 {
			return fmt.Errorf("rtlsdr: USB warmup: %w%s", err, tunerBringupHint(err))
		}
		if !errors.Is(err, syscall.EPIPE) && !errors.Is(err, usb.ErrDeviceGone) {
			return fmt.Errorf("rtlsdr: USB warmup: %w", err)
		}
		if resetErr := transport.Reset(); resetErr != nil {
			return fmt.Errorf("rtlsdr: USB warmup hit %w; reset failed: %w", err, resetErr)
		}
		_ = transport.ReleaseInterface(0)
		if claimErr := transport.ClaimInterface(0); claimErr != nil {
			return fmt.Errorf("rtlsdr: re-claim interface 0 after reset: %w", claimErr)
		}
	}
	return nil
}

// tunerBringupHint returns a parenthesized, space-prefixed remediation
// string when err looks like the tuner-not-acking-on-I2C case
// (issue #248): EPIPE from the first I2C burst, or the device
// disappearing mid-bringup. The hint covers the three known root
// causes — DVB kernel driver still bound, marginal USB power, or a
// half-initialized tuner state — and links to the docs troubleshooting
// table. Returns "" for unrelated errors so the wrapped message stays
// clean.
func tunerBringupHint(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, usb.ErrDeviceGone) {
		return " (hint: tuner did not respond on the I2C bus — common causes:" +
			" DVB kernel driver still bound (run `sudo modprobe -r dvb_usb_rtl28xxu` and re-plug)," +
			" marginal USB power, or a flaky cable/hub." +
			" See https://mattcheramie.github.io/GopherTrunk/install-linux.html#troubleshooting)"
	}
	return ""
}

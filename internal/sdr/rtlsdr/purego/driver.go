package purego

import (
	"fmt"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// DriverName is the [sdr.Driver.Name] this backend registers under.
// Distinct from "rtlsdr" so the pure-Go and CGO drivers can coexist
// during the transition (PR-08 will swap names).
const DriverName = "rtlsdr-go"

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

// Name returns "rtlsdr-go".
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
		return nil, fmt.Errorf("rtlsdr-go: enumerate: %w", err)
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
//  3. Wrap the transport in [rtl2832u.Demod] and run the baseband
//     init flood.
//  4. Probe for an R820T-family tuner; set up the [tuners.Tuner]
//     and run its init.
//  5. Program the tuner's IF frequency on the demod.
//  6. Populate [sdr.Info] with the tuner's name and gain ladder.
//
// Failure at any step closes the transport and returns the error.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	d.mu.Lock()
	if idx < 0 || idx >= len(d.detectCache) {
		d.mu.Unlock()
		return nil, fmt.Errorf("rtlsdr-go: index %d out of range (cache size %d; call Enumerate first)", idx, len(d.detectCache))
	}
	desc := d.detectCache[idx]
	d.mu.Unlock()

	transport, err := d.enumerator().Open(desc)
	if err != nil {
		return nil, fmt.Errorf("rtlsdr-go: open USB %s: %w", desc.Path, err)
	}

	dev, err := openDevice(transport, desc, idx)
	if err != nil {
		_ = transport.Close()
		return nil, err
	}
	return dev, nil
}

// openDevice runs the full bring-up sequence: claim interface, init
// demod, detect + init tuner, set IF freq. Factored out of Open so
// tests can drive it directly with a mock transport without going
// through the enumerator.
func openDevice(transport usb.Transport, desc usb.Descriptor, idx int) (*Device, error) {
	if err := transport.ClaimInterface(0); err != nil {
		return nil, fmt.Errorf("rtlsdr-go: claim interface 0: %w", err)
	}

	demod := rtl2832u.New(transport)
	if err := demod.InitBaseband(); err != nil {
		return nil, fmt.Errorf("rtlsdr-go: init baseband: %w", err)
	}

	i2cAddr, chipType, err := tuners.Detect(demod)
	if err != nil {
		return nil, fmt.Errorf("rtlsdr-go: tuner detect: %w", err)
	}
	tuner := tuners.NewR82xx(demod, i2cAddr, chipType)
	if err := tuner.Init(); err != nil {
		return nil, fmt.Errorf("rtlsdr-go: tuner init: %w", err)
	}
	if err := demod.SetIFFreq(tuner.IFFreqHz()); err != nil {
		return nil, fmt.Errorf("rtlsdr-go: set IF freq: %w", err)
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
		TunerName:    chipType.String(),
		Gains:        tuner.Gains(),
	}
	return &Device{
		transport: transport,
		demod:     demod,
		tuner:     tuner,
		info:      info,
	}, nil
}

// Package airspy is a pure-Go driver for the Airspy R2 / Airspy Mini
// software-defined radios, implementing [sdr.Driver] and [sdr.Device].
//
// It speaks the libairspy USB vendor protocol directly over the shared
// pure-Go USB transport at internal/sdr/rtlsdr/usb — the same transport
// the RTL-SDR driver uses — so no CGO and no libairspy are pulled into
// the build. Real-hardware validation against an attached Airspy is a
// documented follow-up; the in-package tests exercise the wire
// protocol against a usb.MockTransport.
//
// Sample format: this driver pins the device to INT16_IQ — signed
// 16-bit interleaved IQ pairs (4 bytes per sample) — and converts each
// pair to a complex64 with components in [-1, 1].
package airspy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// USB IDs. Airspy R2 and Airspy Mini both enumerate on this pair.
const (
	vidAirspy uint16 = 0x1d50
	pidAirspy uint16 = 0x60a1
)

// libairspy vendor request opcodes (subset).
const (
	reqReceiverMode   uint8 = 1
	reqSetSamplerate  uint8 = 12
	reqSetFreq        uint8 = 13
	reqSetLNAGain     uint8 = 14
	reqSetMixerGain   uint8 = 15
	reqSetVGAGain     uint8 = 16
	reqSetLNAAGC      uint8 = 17
	reqSetMixerAGC    uint8 = 18
	reqGPIOWrite      uint8 = 21
	reqGetSamplerates uint8 = 25
)

// Sample-type values for reqSetSampleType.
const (
	sampleTypeFloat32IQ uint16 = 0
	sampleTypeInt16IQ   uint16 = 2
)

const (
	receiverModeOff uint16 = 0
	receiverModeOn  uint16 = 1
	biasTeeGPIOPort uint16 = 1
	biasTeeGPIOPin  uint16 = 13

	bulkInEP   byte = 0x81
	driverName      = "airspy"

	defaultVGAGain   = 10
	controlTimeoutMs = 1000
	minSamplerateHz  = 1_000_000
)

var openRetryBackoff = 250 * time.Millisecond

// Driver implements sdr.Driver for Airspy.
type Driver struct {
	enum usb.Enumerator

	mu     sync.Mutex
	cached []usb.Descriptor
}

// New returns a Driver backed by enum (nil → platform default).
func New(enum usb.Enumerator) *Driver {
	if enum == nil {
		enum = usb.DefaultEnumerator()
	}
	return &Driver{enum: enum}
}

// Name implements sdr.Driver.
func (d *Driver) Name() string { return driverName }

// Enumerate finds every Airspy on the bus and caches the descriptor
// list so a subsequent Open reuses the same ordering.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	descs, err := d.enum.List(vidAirspy, pidAirspy)
	if err != nil {
		return nil, fmt.Errorf("airspy: enumerate: %w", err)
	}
	d.mu.Lock()
	d.cached = descs
	d.mu.Unlock()

	out := make([]sdr.Info, len(descs))
	for i, desc := range descs {
		serial := desc.Serial
		if serial == "" {
			serial = fmt.Sprintf("airspy-%02d", i)
		}
		out[i] = sdr.Info{
			Driver:       driverName,
			Index:        i,
			Serial:       serial,
			Manufacturer: desc.Manufacturer,
			Product:      desc.Product,
			TunerName:    tunerNameFor(desc.Product),
			// Indicative tenth-dB presets; the driver also accepts
			// a free-form SetGain value.
			Gains: []int{0, 100, 200, 300, 400, 500},
		}
	}
	return out, nil
}

// tunerNameFor distinguishes Airspy R2 from Airspy Mini for the
// TunerName surface. Both share VID:PID 0x1d50:0x60a1 and the same
// R820T tuner — the USB descriptor's Product string is the only
// observable difference at enumeration time. A missing Product
// falls back to the R2 label since R2 is the older, more common
// variant.
func tunerNameFor(product string) string {
	if strings.Contains(strings.ToUpper(product), "MINI") {
		return "R820T (Airspy Mini)"
	}
	return "R820T (Airspy R2)"
}

// Open claims the device at idx and returns an sdr.Device. The
// returned device defaults to INT16_IQ sample mode and the highest
// rate the firmware advertises.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	d.mu.Lock()
	cached := d.cached
	d.mu.Unlock()
	if cached == nil {
		if _, err := d.Enumerate(); err != nil {
			return nil, err
		}
		d.mu.Lock()
		cached = d.cached
		d.mu.Unlock()
	}
	if idx < 0 || idx >= len(cached) {
		return nil, fmt.Errorf("airspy: index %d out of range", idx)
	}
	desc := cached[idx]
	serial := fallbackSerial(desc.Serial, idx)

	// Windows hosts occasionally surface a transient ErrDeviceGone on early
	// post-open control transfers even though enumeration + WinUsb_Initialize
	// just succeeded. Retry with a fresh open (and descriptor refresh by
	// serial/path) before failing daemon startup.
	const maxOpenAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxOpenAttempts; attempt++ {
		dev, err := d.openDevice(desc, idx, serial)
		if err == nil {
			return dev, nil
		}
		lastErr = err
		if !errors.Is(err, usb.ErrDeviceGone) || attempt == maxOpenAttempts {
			break
		}
		if refreshed, ok := d.refreshDescriptor(desc); ok {
			desc = refreshed
		}
		if openRetryBackoff > 0 {
			time.Sleep(openRetryBackoff)
		}
	}
	return nil, lastErr
}

func (d *Driver) openDevice(desc usb.Descriptor, idx int, serial string) (*Device, error) {
	t, err := d.enum.Open(desc)
	if err != nil {
		return nil, fmt.Errorf("airspy: open %s: %w", desc.Path, err)
	}
	if err := t.ClaimInterface(0); err != nil {
		_ = t.Close()
		return nil, fmt.Errorf("airspy: claim interface 0: %w", err)
	}
	dev := &Device{
		t: t,
		info: sdr.Info{
			Driver:       driverName,
			Index:        idx,
			Serial:       serial,
			Manufacturer: desc.Manufacturer,
			Product:      desc.Product,
			TunerName:    tunerNameFor(desc.Product),
		},
	}
	_ = t.ControlOut(reqReceiverMode, receiverModeOff, 0, nil, controlTimeoutMs)
	// Read the supported-samplerate table so SetSampleRate can match
	// the requested rate against an index.
	rates, err := dev.fetchSampleRates()
	if err != nil {
		// Non-fatal: keep the device usable; SetSampleRate will fall
		// back to index 0.
		dev.rates = nil
	} else {
		dev.rates = rates
	}
	return dev, nil
}

func setSampleTypeInt16(usb.Transport) error {
	// libairspy no longer issues a USB command here; it keeps sample type
	// as host-side conversion state.
	return nil
}

func (d *Driver) refreshDescriptor(current usb.Descriptor) (usb.Descriptor, bool) {
	list, err := d.enum.List(vidAirspy, pidAirspy)
	if err != nil || len(list) == 0 {
		return current, false
	}
	if current.Serial != "" {
		for _, cand := range list {
			if cand.Serial == current.Serial {
				return cand, true
			}
		}
	}
	if current.Path != "" {
		for _, cand := range list {
			if cand.Path == current.Path {
				return cand, true
			}
		}
	}
	return list[0], true
}

func fallbackSerial(s string, idx int) string {
	if s != "" {
		return s
	}
	return fmt.Sprintf("airspy-%02d", idx)
}

// Device is one opened Airspy.
type Device struct {
	t    usb.Transport
	info sdr.Info

	mu        sync.Mutex
	closed    bool
	streaming bool
	rates     []uint32 // supported sample rates, Hz, descending order
}

// Info implements sdr.Device.
func (d *Device) Info() sdr.Info { return d.info }

// SetCenterFreq programs the R820T to hz Hz.
func (d *Device) SetCenterFreq(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, hz)
	return d.t.ControlOut(reqSetFreq, 0, 0, payload, controlTimeoutMs)
}

// SetSampleRate follows libairspy semantics:
// - exact known rates are sent by index
// - otherwise values >= 1 MHz are encoded as (rate_hz*2)/1000 for IQ modes
// and sent as wIndex on an IN vendor request.
func (d *Device) SetSampleRate(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	param := d.sampleRateCommandParam(hz)
	_, err := d.t.ControlIn(reqSetSamplerate, 0, param, 1, controlTimeoutMs)
	return err
}

func (d *Device) sampleRateCommandParam(hz uint32) uint16 {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()

	if hz >= minSamplerateHz {
		for i, r := range rates {
			if hz == r {
				return uint16(i)
			}
		}
		return uint16((hz * 2) / 1000)
	}
	return uint16(hz)
}

// closestRateIndex returns the index of the supported sample rate
// nearest hz. If no table is known, it returns 0.
func (d *Device) closestRateIndex(hz uint32) int {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()
	if len(rates) == 0 {
		return 0
	}
	best, bestDiff := 0, ^uint32(0)
	for i, r := range rates {
		diff := r
		if hz > r {
			diff = hz - r
		} else {
			diff = r - hz
		}
		if diff < bestDiff {
			best, bestDiff = i, diff
		}
	}
	return best
}

// SetGain accepts a single tenth-dB target and distributes it across
// the Airspy's three R820T stages (LNA, mixer, VGA), each 0–15. A
// negative value enables LNA + mixer AGC, matching libairspy's
// "auto" behaviour, and fixes the VGA at a sensible mid-band value.
func (d *Device) SetGain(tenthDB int) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	if tenthDB < 0 {
		if err := d.setLNAAGC(true); err != nil {
			return err
		}
		if err := d.setMixerAGC(true); err != nil {
			return err
		}
		return d.setVGAGain(defaultVGAGain)
	}
	if err := d.setLNAAGC(false); err != nil {
		return err
	}
	if err := d.setMixerAGC(false); err != nil {
		return err
	}
	lna, mixer, vga := splitAirspyGain(tenthDB)
	if err := d.setLNAGain(lna); err != nil {
		return err
	}
	if err := d.setMixerGain(mixer); err != nil {
		return err
	}
	return d.setVGAGain(vga)
}

// splitAirspyGain distributes a tenth-dB target across the three R820T
// gain stages. Each step covers roughly 3 dB; remaining gain rolls
// from LNA → mixer → VGA. Values clamp to 0–15 per stage.
func splitAirspyGain(tenthDB int) (lna, mixer, vga int) {
	const step = 30 // tenths of dB per stage unit
	lna = clamp15(tenthDB / step)
	mixer = clamp15((tenthDB - lna*step) / step)
	vga = clamp15((tenthDB - lna*step - mixer*step) / step)
	return
}

func clamp15(v int) int {
	if v < 0 {
		return 0
	}
	if v > 15 {
		return 15
	}
	return v
}

func (d *Device) setLNAGain(v int) error {
	_, err := d.t.ControlIn(reqSetLNAGain, 0, uint16(v), 1, controlTimeoutMs)
	return err
}
func (d *Device) setMixerGain(v int) error {
	_, err := d.t.ControlIn(reqSetMixerGain, 0, uint16(v), 1, controlTimeoutMs)
	return err
}
func (d *Device) setVGAGain(v int) error {
	_, err := d.t.ControlIn(reqSetVGAGain, 0, uint16(v), 1, controlTimeoutMs)
	return err
}
func (d *Device) setLNAAGC(on bool) error {
	v := uint16(0)
	if on {
		v = 1
	}
	_, err := d.t.ControlIn(reqSetLNAAGC, 0, v, 1, controlTimeoutMs)
	return err
}
func (d *Device) setMixerAGC(on bool) error {
	v := uint16(0)
	if on {
		v = 1
	}
	_, err := d.t.ControlIn(reqSetMixerAGC, 0, v, 1, controlTimeoutMs)
	return err
}

// SetPPM is a no-op for Airspy — the Si5351C reference clock is
// internally trimmed and the libairspy protocol carries no PPM.
func (d *Device) SetPPM(int) error { return nil }

// SetBiasTee toggles the bias-T on the antenna SMA.
func (d *Device) SetBiasTee(enable bool) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	v := uint16(0)
	if enable {
		v = 1
	}
	portPin := (biasTeeGPIOPort << 5) | biasTeeGPIOPin
	return d.t.ControlOut(reqGPIOWrite, v, portPin, nil, controlTimeoutMs)
}

// StreamIQ flips the receiver on and starts the bulk-IN reaper,
// delivering one complex64 chunk per URB.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, usb.ErrClosed
	}
	if d.streaming {
		d.mu.Unlock()
		return nil, errors.New("airspy: stream already active")
	}
	d.streaming = true
	d.mu.Unlock()

	// Apply INT16_IQ at stream start rather than open time. This keeps
	// Open resilient on hosts that reject early vendor OUT transfers,
	// while still pinning the wire format before bulk IQ starts.
	if err := setSampleTypeInt16(d.t); err != nil {
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		return nil, fmt.Errorf("airspy: set sample type: %w", err)
	}

	if err := d.setReceiver(receiverModeOn); err != nil {
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		return nil, fmt.Errorf("airspy: receiver on: %w", err)
	}

	out := make(chan []complex64, 8)
	onPacket := func(buf []byte) {
		samples := decodeInt16IQ(buf)
		select {
		case out <- samples:
		case <-ctx.Done():
		}
	}
	// streamDead fires (exactly once, via streamDeadOnce) when the USB
	// reaper exits without StopBulkIn being called — every URB died of
	// an unrecoverable error. The cleanup goroutine below treats it
	// just like a ctx-cancel: tear the stream down and close `out`
	// so the IQ consumer sees a real EOF instead of hanging forever
	// (issue #345).
	streamDead := make(chan struct{})
	var streamDeadOnce sync.Once
	onStreamDead := func(error) {
		streamDeadOnce.Do(func() { close(streamDead) })
	}
	if err := d.t.StartBulkIn(bulkInEP, usb.DefaultRingBuffers, usb.DefaultBufferLen, onPacket, onStreamDead); err != nil {
		_ = d.setReceiver(receiverModeOff)
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		return nil, fmt.Errorf("airspy: start bulk-in: %w", err)
	}

	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case <-streamDead:
		}
		_ = d.t.StopBulkIn()
		_ = d.setReceiver(receiverModeOff)
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
	}()
	return out, nil
}

// Close stops any active stream and releases the USB handle.
func (d *Device) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.mu.Unlock()
	if d.streaming {
		_ = d.t.StopBulkIn()
		_ = d.setReceiver(receiverModeOff)
	}
	_ = d.t.ReleaseInterface(0)
	return d.t.Close()
}

func (d *Device) isClosed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

func (d *Device) setReceiver(mode uint16) error {
	return d.t.ControlOut(reqReceiverMode, mode, 0, nil, controlTimeoutMs)
}

// fetchSampleRates reads the firmware's supported-rate table. libairspy's
// protocol: GET_SAMPLERATES with wIndex=0 returns a u32 count;
// GET_SAMPLERATES with wIndex=count returns count×u32 rates.
func (d *Device) fetchSampleRates() ([]uint32, error) {
	cntBytes, err := d.t.ControlIn(reqGetSamplerates, 0, 0, 4, controlTimeoutMs)
	if err != nil {
		return nil, err
	}
	if len(cntBytes) < 4 {
		return nil, fmt.Errorf("airspy: short samplerate count (%d bytes)", len(cntBytes))
	}
	count := binary.LittleEndian.Uint32(cntBytes)
	if count == 0 || count > 32 {
		return nil, fmt.Errorf("airspy: implausible samplerate count %d", count)
	}
	listBytes, err := d.t.ControlIn(reqGetSamplerates, 0, uint16(count), int(count*4), controlTimeoutMs)
	if err != nil {
		return nil, err
	}
	if len(listBytes) < int(count*4) {
		return nil, fmt.Errorf("airspy: short samplerate list (%d/%d bytes)", len(listBytes), count*4)
	}
	rates := make([]uint32, count)
	for i := range rates {
		rates[i] = binary.LittleEndian.Uint32(listBytes[4*i:])
	}
	return rates, nil
}

// decodeInt16IQ converts a libairspy INT16_IQ payload (interleaved
// little-endian signed 16-bit I,Q) into normalised complex64.
func decodeInt16IQ(buf []byte) []complex64 {
	n := len(buf) / 4
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
		qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
	return out
}

// Package airspyhf is a pure-Go driver for the Airspy HF+ family
// (Discovery, Dual Port, and the legacy HF+), implementing
// [sdr.Driver] and [sdr.Device].
//
// It speaks the libairspyhf USB vendor protocol directly over the
// shared pure-Go USB transport at internal/sdr/rtlsdr/usb — the same
// transport the RTL-SDR, HackRF, and Airspy R2/Mini drivers use — so
// no CGO and no libairspyhf are pulled into the build. Real-hardware
// validation against an attached HF+ is a documented follow-up; the
// in-package tests exercise the wire protocol against a
// usb.MockTransport.
//
// Coverage: 9 kHz–31 MHz HF + 60 MHz–260 MHz VHF. All three known
// variants (HF+, HF+ Discovery, HF+ Dual Port) enumerate on the same
// VID:PID (0x03eb:0x800c); the USB Product string distinguishes them.
//
// Sample format: the HF+ delivers interleaved int16 IQ pairs (4 bytes
// per sample) on bulk endpoint 0x81 once RECEIVER_MODE has been
// flipped to receive. Each pair is decoded to a complex64 with
// components in [-1, 1].
package airspyhf

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// USB IDs. The HF+ family shares one VID:PID across Discovery, Dual
// Port, and the legacy unit — the descriptor's Product string is the
// only observable model identifier.
const (
	vidAirspyHF uint16 = 0x03eb
	pidAirspyHF uint16 = 0x800c
)

// libairspyhf vendor request opcodes (subset — only the calls this
// driver actually issues; matches libairspyhf/src/airspyhf_commands.h).
// Note that these are NOT the same opcodes as the R2/Mini's
// libairspy protocol — sibling devices, sibling protocols.
const (
	reqReceiverMode      uint8 = 1
	reqSetFreq           uint8 = 2
	reqGetSamplerates    uint8 = 3
	reqSetSamplerate     uint8 = 4
	reqGetSerialBoardID  uint8 = 7
	reqGetVersionString  uint8 = 9
	reqSetHFAGC          uint8 = 10
	reqSetHFAGCThreshold uint8 = 11
	reqSetHFATT          uint8 = 12
	reqSetHFLNA          uint8 = 13
	reqSetBiasTee        uint8 = 17
)

const (
	receiverModeOff uint16 = 0
	receiverModeOn  uint16 = 1

	bulkInEP   byte = 0x81
	driverName      = "airspyhf"

	// HF_ATT covers 0..48 dB in 6 dB steps (step 0..8).
	hfATTStepTenthDB = 60
	hfATTMaxStep     = 8

	// HF_LNA adds a fixed +6 dB preamp when enabled.
	hfLNAGainTenthDB = 60

	controlTimeoutMs = 1000
)

// Driver implements sdr.Driver for the Airspy HF+ family.
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

// Enumerate finds every Airspy HF+ on the bus and caches the
// descriptor list so a subsequent Open reuses the same ordering.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	descs, err := d.enum.List(vidAirspyHF, pidAirspyHF)
	if err != nil {
		return nil, fmt.Errorf("airspyhf: enumerate: %w", err)
	}
	d.mu.Lock()
	d.cached = descs
	d.mu.Unlock()

	out := make([]sdr.Info, len(descs))
	for i, desc := range descs {
		serial := desc.Serial
		if serial == "" {
			serial = fmt.Sprintf("airspyhf-%02d", i)
		}
		variant := variantName(desc.Product)
		out[i] = sdr.Info{
			Driver:       driverName,
			Index:        i,
			Serial:       serial,
			Manufacturer: desc.Manufacturer,
			Product:      variant,
			TunerName:    variant,
			// 0–48 dB attenuation in 6 dB steps, plus an optional
			// +6 dB LNA preamp on top.
			Gains: []int{0, 60, 120, 180, 240, 300, 360, 420, 480, 540},
		}
	}
	return out, nil
}

// variantName returns the canonical HF+ product label for the USB
// descriptor's Product string. Unrecognised strings fall back to
// "Airspy HF+" — that's also what the legacy (pre-Discovery) units
// report.
func variantName(product string) string {
	p := strings.ToUpper(product)
	switch {
	case strings.Contains(p, "DISCOVERY"):
		return "Airspy HF+ Discovery"
	case strings.Contains(p, "DUAL"):
		return "Airspy HF+ Dual Port"
	default:
		return "Airspy HF+"
	}
}

// Open claims the device at idx and returns an sdr.Device. The
// returned device has its firmware-advertised sample-rate table
// pre-loaded so SetSampleRate can match an index. Failures reading
// the table are non-fatal — SetSampleRate falls back to index 0.
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
		return nil, fmt.Errorf("airspyhf: index %d out of range", idx)
	}
	desc := cached[idx]
	t, err := d.enum.Open(desc)
	if err != nil {
		return nil, fmt.Errorf("airspyhf: open %s: %w", desc.Path, err)
	}
	if err := t.ClaimInterface(0); err != nil {
		_ = t.Close()
		return nil, fmt.Errorf("airspyhf: claim interface 0: %w", err)
	}
	variant := variantName(desc.Product)
	dev := &Device{
		t: t,
		info: sdr.Info{
			Driver:       driverName,
			Index:        idx,
			Serial:       fallbackSerial(desc.Serial, idx),
			Manufacturer: desc.Manufacturer,
			Product:      variant,
			TunerName:    variant,
		},
	}
	// Best-effort: read firmware version and append it to TunerName.
	// HF+ firmware older than v1.5 has the opcode but may NAK on some
	// shipments — ignore failures.
	if version, vErr := fetchVersionString(t); vErr == nil && version != "" {
		dev.info.TunerName = fmt.Sprintf("%s (fw %s)", variant, version)
	}
	// Read the firmware's supported-samplerate table so SetSampleRate
	// can pick an index. Discovery exposes 192k/256k/384k/768k;
	// Dual Port typically just 768k. Non-fatal on error.
	if rates, rErr := fetchSampleRates(t); rErr == nil {
		dev.rates = rates
	}
	return dev, nil
}

func fallbackSerial(s string, idx int) string {
	if s != "" {
		return s
	}
	return fmt.Sprintf("airspyhf-%02d", idx)
}

// Device is one opened Airspy HF+.
type Device struct {
	t    usb.Transport
	info sdr.Info

	mu        sync.Mutex
	closed    bool
	streaming bool
	rates     []uint32 // supported sample rates, Hz
}

// Info implements sdr.Device.
func (d *Device) Info() sdr.Info { return d.info }

// SetCenterFreq programs the synthesizer to hz Hz. libairspyhf carries
// the value as a 4-byte little-endian uint32 in the control transfer
// data stage; wValue and wIndex are unused. The firmware accepts
// frequencies outside the device's coverage windows (it just won't
// lock), so the driver forwards verbatim — matching libairspyhf.
func (d *Device) SetCenterFreq(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, hz)
	return d.t.ControlOut(reqSetFreq, 0, 0, payload, controlTimeoutMs)
}

// SetSampleRate selects the firmware-advertised rate closest to hz.
// If the supported-rate table is unavailable, index 0 is used (which
// is always the highest rate the unit supports).
func (d *Device) SetSampleRate(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	idx := d.closestRateIndex(hz)
	return d.t.ControlOut(reqSetSamplerate, uint16(idx), 0, nil, controlTimeoutMs)
}

func (d *Device) closestRateIndex(hz uint32) int {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()
	if len(rates) == 0 {
		return 0
	}
	best, bestDiff := 0, ^uint32(0)
	for i, r := range rates {
		var diff uint32
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
// the HF+'s gain chain: HF_AGC (LNA + mixer AGC managed by firmware),
// HF_ATT (0..48 dB attenuator, 6 dB steps), and HF_LNA (+0 or +6 dB
// preamp). A negative value enables HF_AGC and zeros out the
// attenuator + LNA — that matches libairspyhf's default.
//
// For positive values the driver disables HF_AGC, computes the
// attenuator step, and turns on the LNA when the requested gain has
// not yet been satisfied by attenuator reduction alone.
//
// Note: "gain" on the HF+ is mostly *attenuation reduction* — the
// front-end has a fixed conversion stage and what the operator
// controls is how much signal to push into the ADC. tenthDB here is
// interpreted as "headroom to reduce", consistent with libairspyhf.
func (d *Device) SetGain(tenthDB int) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	if tenthDB < 0 {
		if err := d.setHFAGC(true); err != nil {
			return err
		}
		if err := d.setHFLNA(false); err != nil {
			return err
		}
		return d.setHFATT(0)
	}
	if err := d.setHFAGC(false); err != nil {
		return err
	}
	att := tenthDB / hfATTStepTenthDB
	if att > hfATTMaxStep {
		att = hfATTMaxStep
	}
	remaining := tenthDB - att*hfATTStepTenthDB
	lnaOn := remaining >= hfLNAGainTenthDB
	if err := d.setHFATT(att); err != nil {
		return err
	}
	return d.setHFLNA(lnaOn)
}

// SetPPM is a no-op for the HF+ — the device's TCXO is factory-trimmed
// and the libairspyhf wire protocol carries no PPM register.
func (d *Device) SetPPM(int) error { return nil }

// SetBiasTee toggles the bias-T on the antenna SMA. Dual Port units
// only expose this on the SMA-1 / HF port; SMA-2 (VHF) is unaffected.
func (d *Device) SetBiasTee(enable bool) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	v := uint16(0)
	if enable {
		v = 1
	}
	return d.t.ControlOut(reqSetBiasTee, v, 0, nil, controlTimeoutMs)
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
		return nil, errors.New("airspyhf: stream already active")
	}
	d.streaming = true
	d.mu.Unlock()

	if err := d.setReceiver(receiverModeOn); err != nil {
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		return nil, fmt.Errorf("airspyhf: receiver on: %w", err)
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
		return nil, fmt.Errorf("airspyhf: start bulk-in: %w", err)
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

func (d *Device) setHFAGC(on bool) error {
	v := uint16(0)
	if on {
		v = 1
	}
	return d.t.ControlOut(reqSetHFAGC, v, 0, nil, controlTimeoutMs)
}

func (d *Device) setHFLNA(on bool) error {
	v := uint16(0)
	if on {
		v = 1
	}
	return d.t.ControlOut(reqSetHFLNA, v, 0, nil, controlTimeoutMs)
}

func (d *Device) setHFATT(step int) error {
	if step < 0 {
		step = 0
	}
	if step > hfATTMaxStep {
		step = hfATTMaxStep
	}
	return d.t.ControlOut(reqSetHFATT, uint16(step), 0, nil, controlTimeoutMs)
}

// fetchSampleRates reads the firmware's supported-rate table. The
// libairspyhf protocol mirrors libairspy here: GET_SAMPLERATES with
// wIndex=0 returns a u32 count; GET_SAMPLERATES with wIndex=count
// returns count×u32 rates.
func fetchSampleRates(t usb.Transport) ([]uint32, error) {
	cntBytes, err := t.ControlIn(reqGetSamplerates, 0, 0, 4, controlTimeoutMs)
	if err != nil {
		return nil, err
	}
	if len(cntBytes) < 4 {
		return nil, fmt.Errorf("airspyhf: short samplerate count (%d bytes)", len(cntBytes))
	}
	count := binary.LittleEndian.Uint32(cntBytes)
	if count == 0 || count > 32 {
		return nil, fmt.Errorf("airspyhf: implausible samplerate count %d", count)
	}
	listBytes, err := t.ControlIn(reqGetSamplerates, 0, uint16(count), int(count*4), controlTimeoutMs)
	if err != nil {
		return nil, err
	}
	if len(listBytes) < int(count*4) {
		return nil, fmt.Errorf("airspyhf: short samplerate list (%d/%d bytes)", len(listBytes), count*4)
	}
	rates := make([]uint32, count)
	for i := range rates {
		rates[i] = binary.LittleEndian.Uint32(listBytes[4*i:])
	}
	return rates, nil
}

// fetchVersionString reads the firmware's ASCII version string. Padding
// NULs and any non-printable bytes are stripped.
func fetchVersionString(t usb.Transport) (string, error) {
	reply, err := t.ControlIn(reqGetVersionString, 0, 0, 255, controlTimeoutMs)
	if err != nil {
		return "", err
	}
	return cleanVersionString(reply), nil
}

func cleanVersionString(buf []byte) string {
	end := len(buf)
	for i, b := range buf {
		if b == 0 {
			end = i
			break
		}
	}
	out := make([]byte, 0, end)
	for _, b := range buf[:end] {
		if b >= 0x20 && b < 0x7f {
			out = append(out, b)
		}
	}
	return strings.TrimSpace(string(out))
}

// decodeInt16IQ converts an HF+ bulk-IN payload (interleaved
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

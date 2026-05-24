// Package hackrf is a pure-Go driver for the Great Scott Gadgets HackRF
// One software-defined radio, implementing the [sdr.Driver] and
// [sdr.Device] interfaces.
//
// It speaks the libhackrf USB vendor protocol directly over the shared
// pure-Go USB transport at internal/sdr/rtlsdr/usb — the same transport
// that backs the RTL-SDR driver — so no CGO and no libhackrf are
// pulled into the build. Real-hardware validation against an attached
// HackRF One is a documented follow-up; the in-package tests exercise
// the wire protocol against a usb.MockTransport.
//
// Sample format: HackRF delivers signed 8-bit interleaved IQ
// (I,Q,I,Q,…) on bulk endpoint 0x81 once SET_TRANSCEIVER_MODE has been
// flipped to receive. Each pair is converted to a complex64 sample
// with components in roughly [-1, 1].
package hackrf

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

// USB VIDs/PIDs. The HackRF firmware enumerates the device on this VID
// across several PID variants — One, Jawbreaker prototype, Rad1o.
const (
	vidHackRF       uint16 = 0x1d50
	pidHackRFOne    uint16 = 0x6089
	pidHackRFJawbrk uint16 = 0x604b
	pidHackRFRad1o  uint16 = 0xcc15
)

// libhackrf vendor request opcodes (subset — only the calls this
// driver actually issues; matches host/libhackrf/src/hackrf.h).
const (
	reqSetTransceiverMode  uint8 = 1
	reqSampleRateSet       uint8 = 6
	reqBasebandFilterBwSet uint8 = 7
	reqBoardIDRead         uint8 = 14
	reqVersionStringRead   uint8 = 15
	reqSetFreq             uint8 = 16
	reqAmpEnable           uint8 = 17
	reqSetLNAGain          uint8 = 19
	reqSetVGAGain          uint8 = 20
	reqAntennaEnable       uint8 = 23
)

// pidProductNames maps each known HackRF USB PID to the canonical
// product label we surface in sdr.Info. USB descriptor strings vary
// across vendors and firmware builds; the PID is the stable identity.
var pidProductNames = map[uint16]string{
	pidHackRFOne:    "HackRF One",
	pidHackRFJawbrk: "HackRF Jawbreaker",
	pidHackRFRad1o:  "Rad1o",
}

// boardIDNames mirrors libhackrf's enum hackrf_board_id values. Read
// from the firmware via reqBoardIDRead at open-time; takes precedence
// over the PID-based lookup so a flashed-with-the-wrong-firmware unit
// still reports what's actually running on the board.
var boardIDNames = map[uint8]string{
	0:   "Jellybean",
	1:   "HackRF Jawbreaker",
	2:   "HackRF One",
	3:   "Rad1o",
	255: "Invalid",
}

const (
	transceiverModeOff     uint16 = 0
	transceiverModeReceive uint16 = 1

	bulkInEP   byte = 0x81
	driverName      = "hackrf"

	defaultSampleRateHz uint32 = 8_000_000
	defaultLNAGainDB           = 16
	defaultVGAGainDB           = 20
	controlTimeoutMs           = 1000
)

// Driver implements sdr.Driver for HackRF.
type Driver struct {
	enum usb.Enumerator

	mu     sync.Mutex
	cached []usb.Descriptor
}

// New returns a Driver that enumerates through the supplied USB
// backend. Pass nil to use the platform default enumerator.
func New(enum usb.Enumerator) *Driver {
	if enum == nil {
		enum = usb.DefaultEnumerator()
	}
	return &Driver{enum: enum}
}

// Name implements sdr.Driver.
func (d *Driver) Name() string { return driverName }

// Enumerate scans every HackRF PID variant and returns an sdr.Info per
// detected device. The descriptor list is cached so a subsequent Open
// reuses the same ordering.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	descs := make([]usb.Descriptor, 0)
	for _, pid := range []uint16{pidHackRFOne, pidHackRFJawbrk, pidHackRFRad1o} {
		found, err := d.enum.List(vidHackRF, pid)
		if err != nil {
			return nil, fmt.Errorf("hackrf: enumerate pid=%04x: %w", pid, err)
		}
		descs = append(descs, found...)
	}
	d.mu.Lock()
	d.cached = descs
	d.mu.Unlock()

	out := make([]sdr.Info, len(descs))
	for i, desc := range descs {
		serial := desc.Serial
		if serial == "" {
			serial = fmt.Sprintf("hackrf-%02d", i)
		}
		out[i] = sdr.Info{
			Driver:       driverName,
			Index:        i,
			Serial:       serial,
			Manufacturer: desc.Manufacturer,
			Product:      productForPID(desc.PID, desc.Product),
			TunerName:    "MAX2839+MAX5864",
			// Tenth-dB presets the operator can pick from in the UI;
			// the driver also accepts a free-form value via SetGain.
			Gains: []int{0, 80, 160, 240, 320, 400, 480, 560},
		}
	}
	return out, nil
}

// productForPID returns the canonical HackRF product label for pid,
// falling back to the USB descriptor's Product string when the PID
// isn't in our table (defensive against future firmware revisions
// that ship under a new PID).
func productForPID(pid uint16, fallback string) string {
	if name, ok := pidProductNames[pid]; ok {
		return name
	}
	if fallback != "" {
		return fallback
	}
	return "HackRF"
}

// Open claims the device at idx and returns an sdr.Device. The caller
// is responsible for calling Close.
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
		return nil, fmt.Errorf("hackrf: index %d out of range", idx)
	}
	desc := cached[idx]
	t, err := d.enum.Open(desc)
	if err != nil {
		return nil, fmt.Errorf("hackrf: open %s: %w", desc.Path, err)
	}
	if err := t.ClaimInterface(0); err != nil {
		_ = t.Close()
		return nil, fmt.Errorf("hackrf: claim interface 0: %w", err)
	}
	serial := desc.Serial
	if serial == "" {
		serial = fmt.Sprintf("hackrf-%02d", idx)
	}
	// Best-effort: ask the firmware which board it actually is and
	// what version it's running. Either readback may fail on older
	// firmware — fall through to the PID-derived product name and a
	// plain TunerName when that happens.
	product := productForPID(desc.PID, desc.Product)
	if bid, err := readBoardID(t); err == nil {
		if name, ok := boardIDNames[bid]; ok && name != "" {
			product = name
		}
	}
	tuner := "MAX2839+MAX5864"
	if version, err := readVersionString(t); err == nil && version != "" {
		if isPortaPack(version) {
			product += " + PortaPack"
		}
		tuner = fmt.Sprintf("MAX2839+MAX5864 (fw %s)", version)
	}
	return &Device{
		t: t,
		info: sdr.Info{
			Driver:       driverName,
			Index:        idx,
			Serial:       serial,
			Manufacturer: desc.Manufacturer,
			Product:      product,
			TunerName:    tuner,
		},
	}, nil
}

// readBoardID issues a BOARD_ID_READ vendor request and returns the
// firmware's self-reported board enum value. A single byte is enough
// — we tolerate longer replies for forward compatibility.
func readBoardID(t usb.Transport) (uint8, error) {
	reply, err := t.ControlIn(reqBoardIDRead, 0, 0, 1, controlTimeoutMs)
	if err != nil {
		return 0, err
	}
	if len(reply) == 0 {
		return 0, fmt.Errorf("hackrf: empty board-id reply")
	}
	return reply[0], nil
}

// readVersionString fetches the firmware's printable version string
// (typically a git describe like "git-2024.02.1" or, on Mayhem builds,
// something containing "portapack"/"mayhem"). Trailing NUL padding
// and non-printable bytes are stripped.
func readVersionString(t usb.Transport) (string, error) {
	reply, err := t.ControlIn(reqVersionStringRead, 0, 0, 255, controlTimeoutMs)
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

// isPortaPack reports whether v looks like a PortaPack/Mayhem firmware
// string. Both spellings appear in the wild — official PortaPack
// builds tag themselves, and the community Mayhem fork keeps its own
// name. We match either.
func isPortaPack(v string) bool {
	lv := strings.ToLower(v)
	return strings.Contains(lv, "portapack") || strings.Contains(lv, "mayhem")
}

// Device is one opened HackRF.
type Device struct {
	t    usb.Transport
	info sdr.Info

	mu         sync.Mutex
	closed     bool
	streaming  bool
	sampleRate uint32
}

// Info implements sdr.Device.
func (d *Device) Info() sdr.Info { return d.info }

// SetCenterFreq programs the synthesizer to the requested frequency,
// in Hz. libhackrf splits the value into MHz + Hz-remainder octets.
func (d *Device) SetCenterFreq(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:4], hz/1_000_000)
	binary.LittleEndian.PutUint32(payload[4:8], hz%1_000_000)
	return d.t.ControlOut(reqSetFreq, 0, 0, payload, controlTimeoutMs)
}

// SetSampleRate programs the baseband sampler. libhackrf encodes the
// rate as a numerator/divider pair; this driver always uses a divider
// of 1 so the host sees exactly the requested rate.
func (d *Device) SetSampleRate(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	if hz == 0 {
		hz = defaultSampleRateHz
	}
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:4], hz)
	binary.LittleEndian.PutUint32(payload[4:8], 1)
	if err := d.t.ControlOut(reqSampleRateSet, 0, 0, payload, controlTimeoutMs); err != nil {
		return err
	}
	// The baseband-filter cutoff tracks the sample rate; matching it
	// to ~75 % of the rate avoids aliasing.
	bw := uint32(float64(hz) * 0.75)
	bwBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bwBytes, bw)
	_ = d.t.ControlOut(reqBasebandFilterBwSet,
		uint16(bw&0xFFFF), uint16(bw>>16), nil, controlTimeoutMs)
	d.mu.Lock()
	d.sampleRate = hz
	d.mu.Unlock()
	return nil
}

// SetGain accepts a single tenth-dB target and distributes it across
// the HackRF's three gain stages (RF amp on/off, LNA 0–40 dB in 8 dB
// steps, VGA 0–62 dB in 2 dB steps). A negative value selects a
// hardware-friendly preset (amp off, LNA 16 dB, VGA 20 dB) — the
// HackRF has no true AGC, so "auto" maps to a sane fixed split.
func (d *Device) SetGain(tenthDB int) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	lna, vga, amp := splitGain(tenthDB)
	if err := d.amp(amp); err != nil {
		return err
	}
	if err := d.setLNAGain(lna); err != nil {
		return err
	}
	return d.setVGAGain(vga)
}

// splitGain maps an SDR-interface tenth-dB target to the HackRF's
// (LNA, VGA, AMP-on) triple. LNA must be a multiple of 8; VGA a
// multiple of 2.
func splitGain(tenthDB int) (lna, vga int, amp bool) {
	if tenthDB < 0 {
		return defaultLNAGainDB, defaultVGAGainDB, false
	}
	target := tenthDB / 10
	lna = (target / 8) * 8
	if lna > 40 {
		lna = 40
	}
	rem := target - lna
	vga = (rem / 2) * 2
	if vga > 62 {
		vga = 62
	}
	return lna, vga, false
}

func (d *Device) setLNAGain(db int) error {
	if db < 0 {
		db = 0
	}
	if db > 40 {
		db = 40
	}
	// libhackrf returns 1 byte of status (0=fail, 1=ok). We don't
	// fail on a missing status byte — the request itself succeeding
	// means the gain landed.
	_, err := d.t.ControlIn(reqSetLNAGain, 0, uint16(db), 1, controlTimeoutMs)
	return err
}

func (d *Device) setVGAGain(db int) error {
	if db < 0 {
		db = 0
	}
	if db > 62 {
		db = 62
	}
	_, err := d.t.ControlIn(reqSetVGAGain, 0, uint16(db), 1, controlTimeoutMs)
	return err
}

func (d *Device) amp(on bool) error {
	v := uint16(0)
	if on {
		v = 1
	}
	return d.t.ControlOut(reqAmpEnable, v, 0, nil, controlTimeoutMs)
}

// SetPPM is a no-op for HackRF — the Si5351C reference clock is
// internally trimmed and the protocol carries no PPM correction.
func (d *Device) SetPPM(int) error { return nil }

// SetBiasTee toggles the +3.3 V antenna-port bias for external LNAs.
func (d *Device) SetBiasTee(enable bool) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	v := uint16(0)
	if enable {
		v = 1
	}
	return d.t.ControlOut(reqAntennaEnable, v, 0, nil, controlTimeoutMs)
}

// StreamIQ flips the HackRF into receive mode and reaps bulk-IN URBs,
// converting each int8 IQ pair into a complex64 sample on the returned
// channel. Close (or ctx cancellation) stops the stream and returns
// the device to off-mode.
func (d *Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, usb.ErrClosed
	}
	if d.streaming {
		d.mu.Unlock()
		return nil, errors.New("hackrf: stream already active")
	}
	d.streaming = true
	d.mu.Unlock()

	if err := d.setMode(transceiverModeReceive); err != nil {
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		return nil, fmt.Errorf("hackrf: set receive mode: %w", err)
	}

	out := make(chan []complex64, 8)
	stopped := make(chan struct{})

	onPacket := func(buf []byte) {
		samples := decodeInt8IQ(buf)
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
		_ = d.setMode(transceiverModeOff)
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		return nil, fmt.Errorf("hackrf: start bulk-in: %w", err)
	}

	go func() {
		defer close(out)
		defer close(stopped)
		select {
		case <-ctx.Done():
		case <-streamDead:
		}
		_ = d.t.StopBulkIn()
		_ = d.setMode(transceiverModeOff)
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
		_ = d.setMode(transceiverModeOff)
	}
	_ = d.t.ReleaseInterface(0)
	return d.t.Close()
}

func (d *Device) isClosed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

func (d *Device) setMode(mode uint16) error {
	return d.t.ControlOut(reqSetTransceiverMode, mode, 0, nil, controlTimeoutMs)
}

// decodeInt8IQ converts a HackRF bulk-IN payload (signed 8-bit
// interleaved IQ) into normalised complex64 samples in [-1, 1].
func decodeInt8IQ(buf []byte) []complex64 {
	n := len(buf) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := float32(int8(buf[2*i])) / 128
		qv := float32(int8(buf[2*i+1])) / 128
		out[i] = complex(iv, qv)
	}
	return out
}

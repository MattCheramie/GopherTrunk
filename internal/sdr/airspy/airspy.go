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
// Sample format: the Airspy R2 / Mini are real-sampling receivers — the
// firmware streams bare ADC samples (unpacked little-endian uint16, 12-bit,
// DC at 2048) at twice the configured IQ rate. This driver converts that real
// stream to complex64 baseband on the host via the [iqConverter] (an Fs/4
// translation plus a half-band Hilbert pair, decimating by two), matching what
// libairspy does in its IQ sample modes.
package airspy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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

// gainPresetsTenthDB are indicative tenth-dB gain values surfaced in
// sdr.Info.Gains so `sdr list --probe` and the web UI can offer the
// operator a ladder to pick from (0–50 dB). SetGain also accepts any
// free-form tenth-dB value between these rungs. Shared by Enumerate
// and openDevice so a probed (opened) device reports the same ladder
// an enumerated one does — otherwise `--probe` shows an empty list
// (issue: airspy/hackrf probe gains came back []).
var gainPresetsTenthDB = []int{0, 100, 200, 300, 400, 500}

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
			Gains:        gainPresetsTenthDB,
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
	// When RTLSDR_DEBUG_USB is set, trace every vendor control transfer
	// (SET_SAMPLERATE / SET_FREQ / RECEIVER_MODE / gain). The RTL-SDR driver
	// already wraps its transport this way; the Airspy shares the same USB
	// transport but never wrapped it, so its control setup was invisible even
	// with debugging on. No-op when the env var is unset. (Bulk-IN is left
	// untouched — the stall watchdog + telemetry live inside the transport.)
	t = usb.MaybeWrapDebug(t, desc)
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
			// Carry the gain ladder onto the opened device so
			// dev.Info() (used by `sdr list --probe`) reports it
			// instead of an empty list.
			Gains: gainPresetsTenthDB,
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
	// streamDone is closed when the current stream's teardown goroutine
	// finishes (after `streaming` is cleared). A new StreamIQ waits on it
	// so a fast retune can't race the previous stream's async cleanup
	// (#686). Non-nil whenever streaming is true.
	streamDone chan struct{}
	rates      []uint32     // supported sample rates, Hz, descending order
	cnv        *iqConverter // real-to-IQ converter, fresh per stream
	// lastReqRate is the IQ rate (Hz) from the most recent SetSampleRate,
	// remembered so ActualSampleRate can report the rate the device will
	// actually deliver after snapping to the firmware's fixed rate table.
	lastReqRate uint32
	// dropped counts IQ chunks the reaper discarded because the primary
	// consumer fell behind (see StreamIQ's onPacket). Mirrors the RTL-SDR
	// purego driver's Device.dropped so a live overrun is visible rather
	// than silently wedging the USB reaper.
	dropped atomic.Uint64
	// streamDeadErr holds the USB error that killed the most recent bulk-IN
	// stream — the error the transport hands to onStreamDead (usb.ErrStreamStalled
	// when the stall watchdog aborted a frozen endpoint, usb.ErrDeviceGone on a
	// disconnect, or a wrapped per-URB error). It was previously discarded, so an
	// operator only ever saw the generic "IQ stream closed unexpectedly" with no
	// cause; StreamDeadCause surfaces it so widebandt2/ccdecoder can name the real
	// reason in the daemon's "IQ stream died" log. Reset to nil at the start of
	// each StreamIQ so a prior death never leaks into a later clean shutdown.
	streamDeadErr atomic.Pointer[error]
}

// StreamDeadCause reports the USB error that terminated the most recent bulk-IN
// stream, or nil if the stream ended cleanly (ctx cancel / Close) or has not
// died. The daemon's IQ-stream retry loop (via widebandt2 / ccdecoder) wraps it
// into ErrIQStreamClosed so the "IQ stream died" log line names the concrete
// cause instead of a generic message. Satisfies the optional
// interface{ StreamDeadCause() error } those consumers probe for.
func (d *Device) StreamDeadCause() error {
	if p := d.streamDeadErr.Load(); p != nil {
		return *p
	}
	return nil
}

// Info implements sdr.Device.
func (d *Device) Info() sdr.Info { return d.info }

// FreqRange reports the Airspy R2/Mini's documented 24 MHz .. 1.8 GHz
// tuning span (sdr.FreqRanger), so a whole-device hunt can sweep it.
func (d *Device) FreqRange() (minHz, maxHz uint32) { return 24_000_000, 1_800_000_000 }

// SetCenterFreq programs the R820T to hz Hz.
func (d *Device) SetCenterFreq(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, hz)
	return d.t.ControlOut(reqSetFreq, 0, 0, payload, controlTimeoutMs)
}

// SetSampleRate programs the firmware so the host-side IQ stream comes
// out at the requested IQ rate (hz).
//
// The Airspy is a real-sampling receiver: the host converter (iqconverter.go)
// translates by Fs/4 and decimates by two. The firmware's rate table
// (fetchSampleRates, i.e. libairspy GET_SAMPLERATES) is in *IQ output* rates —
// what the device delivers after that decimation: an Airspy R2 advertises
// {10 MSPS, 2.5 MSPS}, a Mini {6 MSPS, 3 MSPS}. Selecting a table entry makes
// the firmware stream the real ADC at 2× that rate, which the converter
// decimates back down, so the delivered IQ rate equals the table entry. We
// therefore match the requested IQ rate against the table directly.
//
// The prior code treated the table as 2× "device" rates and multiplied hz by
// two before matching, so ActualSampleRate reported half the rate the hardware
// truly delivered. The daemon (effectiveStreamRate, issue #402) then built its
// down-converters at half rate, the symbol clock ran 2× off, and nothing
// decoded — exactly the Airspy R2-at-10-MS/s no-decode report.
func (d *Device) SetSampleRate(hz uint32) error {
	if d.isClosed() {
		return usb.ErrClosed
	}
	param := d.sampleRateCommandParam(hz)
	_, err := d.t.ControlIn(reqSetSamplerate, 0, param, 1, controlTimeoutMs)
	if err == nil {
		d.mu.Lock()
		d.lastReqRate = hz
		d.mu.Unlock()
	}
	return err
}

// ActualSampleRate reports the IQ rate the device will actually deliver for the
// most recent SetSampleRate. The Airspy only streams the discrete rates its
// firmware advertises, so a requested rate the table can't honour (e.g. 6 MS/s
// on an R2 that does 2.5/10) is snapped to the nearest one by
// sampleRateCommandParam — silently changing the delivered rate. Implementing
// this optional sdr.Device extension lets the daemon (effectiveStreamRate) warn
// on the mismatch and build its down-converters from the rate that actually
// arrives, keeping the symbol clock aligned (issue #402). Returns the requested
// rate unchanged when it maps exactly, so correctly-configured rates stay quiet.
func (d *Device) ActualSampleRate() (uint32, error) {
	d.mu.Lock()
	last := d.lastReqRate
	d.mu.Unlock()
	if last == 0 {
		// SetSampleRate hasn't run yet; let the caller fall back to its
		// requested value rather than inventing one.
		return 0, nil
	}
	return d.resolvedIQRate(last), nil
}

// resolvedIQRate returns the IQ rate the firmware delivers for a requested IQ
// rate, mirroring the index decision in sampleRateCommandParam: the advertised
// table holds IQ output rates, so it returns the requested rate when it matches
// a table entry exactly, otherwise the nearest advertised rate. With no rate
// table or a sub-MHz request it returns iqHz unchanged (best effort).
func (d *Device) resolvedIQRate(iqHz uint32) uint32 {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()
	if iqHz < minSamplerateHz || len(rates) == 0 {
		return iqHz
	}
	for _, r := range rates {
		if iqHz == r {
			return iqHz
		}
	}
	return rates[d.closestRateIndex(iqHz)]
}

// sampleRateCommandParam maps a requested IQ rate to the wIndex the firmware
// expects: the index of the advertised IQ rate that matches exactly, the
// nearest table index when it doesn't, or a by-value encoding (iqHz/1000) when
// no table was read.
func (d *Device) sampleRateCommandParam(iqHz uint32) uint16 {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()

	if iqHz >= minSamplerateHz {
		for i, r := range rates {
			if iqHz == r {
				return uint16(i)
			}
		}
		if len(rates) > 0 {
			// Snap to the nearest advertised IQ rate rather than emitting a
			// by-value encoding the firmware may reject.
			return uint16(d.closestRateIndex(iqHz))
		}
		return uint16(iqHz / 1000)
	}
	return uint16(iqHz)
}

// closestRateIndex returns the index of the advertised IQ sample rate nearest
// iqHz. The Airspy's GET_SAMPLERATES table holds IQ output rates directly (the
// firmware streams the real ADC at twice that rate and the host converter
// decimates by two; see SetSampleRate). If no table is known, it returns 0.
func (d *Device) closestRateIndex(iqHz uint32) int {
	d.mu.Lock()
	rates := d.rates
	d.mu.Unlock()
	if len(rates) == 0 {
		return 0
	}
	best, bestDiff := 0, ^uint32(0)
	for i, r := range rates {
		var diff uint32
		if iqHz > r {
			diff = iqHz - r
		} else {
			diff = r - iqHz
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
	for {
		if d.closed {
			d.mu.Unlock()
			return nil, usb.ErrClosed
		}
		if !d.streaming {
			break
		}
		// A previous stream is still tearing down (its ctx was cancelled
		// by a retune). Wait it out instead of failing fast with
		// "stream already active" (#686). The hardware is single-stream
		// and the prior teardown completes promptly; the wait is bounded
		// by this call's ctx so shutdown can't hang here.
		done := d.streamDone
		d.mu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		d.mu.Lock()
	}
	d.streaming = true
	done := make(chan struct{})
	d.streamDone = done
	d.mu.Unlock()

	// Clear any cause recorded by a previous stream's death so it can't be
	// misattributed to this session (e.g. a later clean ctx-cancel shutdown).
	d.streamDeadErr.Store(nil)

	// Fresh real-to-IQ converter per stream so filter memory never carries
	// over from a previous session. The device streams bare real samples;
	// the converter turns them into complex baseband (see iqconverter.go).
	d.cnv = newIQConverter()

	if err := d.setReceiver(receiverModeOn); err != nil {
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		close(done)
		return nil, fmt.Errorf("airspy: receiver on: %w", err)
	}

	out := make(chan []complex64, 8)
	onPacket := func(buf []byte) {
		samples := d.cnv.processRaw(buf)
		select {
		case out <- samples:
		default:
			// Consumer fell behind. Drop the chunk rather than block the USB
			// reaper: each bulkLoop goroutine posts its next ReadPipe only after
			// onPacket returns, so a blocked send stops posting reads. Stall all
			// of them (macOS runs DefaultRingBuffers=32 concurrent reapers) and
			// no reads are outstanding — the device FIFO overflows and the whole
			// stream silently wedges with no error, no EOF, no log (the Airspy
			// freeze). Shed the chunk and surface it instead: the daemon's IQ-drop
			// observer bumps iq_underruns_total and emits a rate-limited "host
			// can't keep up — lower sdr.sample_rate" WARN.
			d.dropped.Add(1)
			sdr.NotifyIQDrop(d.info)
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
	onStreamDead := func(cause error) {
		streamDeadOnce.Do(func() {
			// Record the killing error (before closing streamDead so it is
			// visible by the time the cleanup goroutine closes `out` and the
			// IQ consumer observes EOF). StreamDeadCause exposes it so the
			// daemon log names the real cause instead of a generic message.
			if cause != nil {
				d.streamDeadErr.Store(&cause)
			}
			close(streamDead)
		})
	}
	if err := d.t.StartBulkIn(bulkInEP, usb.DefaultRingBuffers, usb.DefaultBufferLen, onPacket, onStreamDead); err != nil {
		_ = d.setReceiver(receiverModeOff)
		d.mu.Lock()
		d.streaming = false
		d.mu.Unlock()
		close(done)
		return nil, fmt.Errorf("airspy: start bulk-in: %w", err)
	}

	go func() {
		defer close(out)
		defer close(done)
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

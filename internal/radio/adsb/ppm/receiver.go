// Package ppm is the native 1090 MHz Mode-S / ADS-B receiver: it
// demodulates the pulse-position-modulated downlink directly from an
// IQ stream, with no external dump1090 / readsb required. The pipeline
// is:
//
//	IQ chunks (Fs Hz, complex64)
//	  → resample to 2 Msps (2 samples/µs; skipped if already 2 Msps)
//	  → magnitude² envelope (I² + Q²)
//	  → Mode-S preamble correlation (8 µs pattern: pulses at
//	    0, 1, 3.5, 4.5 µs)
//	  → PPM bit slice (1 µs/bit: "1" = high-then-low half, "0" =
//	    low-then-high half)
//	  → frame-length from DF (56- vs 112-bit) → 7- or 14-byte frame
//	  → adsb.ProcessFrame (decode → CPR track → AircraftReport)
//	  → events.KindAircraftReport on the bus
//
// The decode → track → publish stage is the exact path the BEAST
// upstream client uses (adsb.ProcessFrame), so a frame recovered off
// the air and the same frame relayed from dump1090 produce identical
// reports. This package owns only IQ-to-frame.
//
// Operators pin one SDR to 1090 MHz (a 1090 MHz SAW filter + LNA is
// strongly recommended) and add it under adsb.channels. dump1090 users
// stay on the BEAST upstream; this is for greenfield setups that want
// GopherTrunk to own the whole chain.
//
// The preamble detector and PPM slicer follow the well-trodden
// dump1090 baseline (fixed 2 Msps, magnitude-domain). Phase-corrected
// re-detection and 2.4 Msps operation are refinements left for later;
// this baseline locks cleanly on strong signals, which is what a
// filtered + amplified 1090 MHz chain delivers.
package ppm

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/adsb"
)

// SampleRateHz is the fixed internal processing rate: 2 Msps gives
// 2 samples per microsecond, the minimum to resolve a PPM half-bit.
const SampleRateHz = 2_000_000

// samplesPerBit is one Mode-S bit period (1 µs) in samples at the
// internal rate.
const samplesPerBit = SampleRateHz / 1_000_000

// preambleSamples is the 8 µs Mode-S preamble in samples.
const preambleSamples = 8 * samplesPerBit // 16

// shortBits / longBits are the two Mode-S frame lengths.
const (
	shortBits = 56  // DF 0/4/5/11 — 7 bytes
	longBits  = 112 // DF 16/17/18/19/20/21/24 — 14 bytes
)

// frameSpan is the worst-case sample footprint of preamble + a long
// frame; scanning needs this many samples available to attempt a
// decode without running off the buffer.
const frameSpan = preambleSamples + longBits*samplesPerBit // 240

// Options configures a Receiver.
type Options struct {
	// InputRateHz is the IQ sample rate the broker is feeding at.
	// When it is exactly SampleRateHz the magnitude envelope is taken
	// directly; otherwise the IQ is resampled to SampleRateHz first.
	InputRateHz uint32

	// SourceName is stamped on log lines and surfaces in metrics.
	SourceName string

	// Bus is required — reports publish onto KindAircraftReport.
	Bus *events.Bus

	// Log is optional; defaults to slog.Default.
	Log *slog.Logger
}

// Receiver runs a native Mode-S PPM decode pipeline against a stream
// of IQ chunks. One Receiver per 1090 MHz SDR.
type Receiver struct {
	source  string
	log     *slog.Logger
	bus     *events.Bus
	tracker *adsb.Tracker

	rs *dsp.Resampler // nil when InputRateHz == SampleRateHz

	// Scratch + carry buffers reused across chunks. mag accumulates
	// the magnitude envelope; up to frameSpan-1 samples are retained
	// across calls so a preamble straddling a chunk boundary still
	// decodes.
	rsBuf []complex64
	mag   []float32

	samplesSeen   atomic.Uint64
	preamblesSeen atomic.Uint64
	framesDecoded atomic.Uint64
	framesEmitted atomic.Uint64
}

// New constructs a Receiver. Returns an error if opts.Bus is nil or
// InputRateHz is unset / below the 2 Msps the PPM slicer needs.
func New(opts Options) (*Receiver, error) {
	if opts.Bus == nil {
		return nil, errors.New("adsb/ppm: Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("adsb/ppm: InputRateHz is required")
	}
	if opts.InputRateHz < SampleRateHz {
		return nil, errors.New("adsb/ppm: InputRateHz must be at least 2 Msps")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	r := &Receiver{
		source:  opts.SourceName,
		log:     log,
		bus:     opts.Bus,
		tracker: adsb.NewTracker(),
	}
	if opts.InputRateHz != SampleRateHz {
		g := gcd(SampleRateHz, opts.InputRateHz)
		L := int(SampleRateHz / g)
		M := int(opts.InputRateHz / g)
		r.rs = dsp.NewResampler(L, M, 16, 7.0)
	}
	return r, nil
}

// Process pumps IQ chunks from in through the decode pipeline until
// ctx cancels or in closes.
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("adsb/ppm: input channel is nil")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			r.processChunk(chunk)
		}
	}
}

func (r *Receiver) processChunk(chunk []complex64) {
	r.samplesSeen.Add(uint64(len(chunk)))
	samples := chunk
	if r.rs != nil {
		r.rsBuf = r.rs.Process(r.rsBuf, chunk)
		samples = r.rsBuf
	}
	for _, c := range samples {
		i, q := real(c), imag(c)
		r.mag = append(r.mag, i*i+q*q)
	}
	r.scan()
}

// scan walks the magnitude buffer hunting Mode-S preambles, demodulates
// the frame that follows each detection, and retains the unprocessed
// tail (a frame may straddle the next chunk).
func (r *Receiver) scan() {
	i := 0
	for i+frameSpan <= len(r.mag) {
		if !detectPreamble(r.mag[i : i+preambleSamples+2]) {
			i++
			continue
		}
		r.preamblesSeen.Add(1)
		frame, nbits := r.demodFrame(r.mag[i+preambleSamples:])
		if nbits == 0 {
			i++
			continue
		}
		r.handleFrame(frame)
		// Skip past the consumed frame.
		i += preambleSamples + nbits*samplesPerBit
	}
	// Retain the tail for the next chunk.
	if i > 0 {
		n := copy(r.mag, r.mag[i:])
		r.mag = r.mag[:n]
	}
}

// demodFrame slices the PPM data that follows a detected preamble.
// data points at the first data sample. It reads the 5-bit DF to pick
// the 56- or 112-bit length, then packs the bits MSB-first into bytes.
// Returns the frame bytes and the bit count (0 if the magnitude buffer
// is too short).
func (r *Receiver) demodFrame(data []float32) ([]byte, int) {
	// Read DF (first 5 bits) to determine length.
	if len(data) < 5*samplesPerBit {
		return nil, 0
	}
	df := 0
	for b := 0; b < 5; b++ {
		df = df<<1 | sliceBit(data, b)
	}
	nbits := shortBits
	if longFrame(df) {
		nbits = longBits
	}
	if len(data) < nbits*samplesPerBit {
		return nil, 0
	}
	out := make([]byte, nbits/8)
	for b := 0; b < nbits; b++ {
		if sliceBit(data, b) == 1 {
			out[b/8] |= 1 << uint(7-(b%8))
		}
	}
	return out, nbits
}

// handleFrame runs a demodulated frame through the shared
// decode → track → publish path and emits the report.
func (r *Receiver) handleFrame(frame []byte) {
	rep, ok := adsb.ProcessFrame(frame, r.tracker, time.Now())
	if !ok {
		return
	}
	r.framesDecoded.Add(1)
	r.bus.Publish(events.Event{Kind: events.KindAircraftReport, Payload: rep})
	r.framesEmitted.Add(1)
}

// sliceBit slices one PPM bit at bit index b within data (which starts
// at the first data sample). Mode-S PPM: "1" is high-then-low across
// the two half-bit samples, "0" is low-then-high.
func sliceBit(data []float32, b int) int {
	first := data[b*samplesPerBit]
	second := data[b*samplesPerBit+1]
	if first > second {
		return 1
	}
	return 0
}

// longFrame reports whether a downlink format uses the 112-bit frame.
func longFrame(df int) bool {
	switch df {
	case 16, 17, 18, 19, 20, 21, 24:
		return true
	}
	return false
}

// detectPreamble applies the dump1090 magnitude-domain preamble test
// to a window starting at the candidate preamble. m must hold at least
// preambleSamples samples. The pattern is high pulses at sample
// offsets 0, 2, 7, 9 (µs 0, 1, 3.5, 4.5 at 2 Msps) with quiet gaps and
// a quiet zone before the data.
func detectPreamble(m []float32) bool {
	if len(m) < preambleSamples {
		return false
	}
	if !(m[0] > m[1] &&
		m[1] < m[2] &&
		m[2] > m[3] &&
		m[3] < m[0] &&
		m[4] < m[0] &&
		m[5] < m[0] &&
		m[6] < m[0] &&
		m[7] > m[8] &&
		m[8] < m[9] &&
		m[9] > m[6]) {
		return false
	}
	// Average pulse level. The quiet samples around and after the
	// preamble must sit clearly below it, which rejects random noise
	// that happens to satisfy the relative-ordering test above.
	high := (m[0] + m[2] + m[7] + m[9]) / 6
	if m[5] >= high || m[6] >= high {
		return false
	}
	for j := 10; j < preambleSamples; j++ {
		if m[j] >= high {
			return false
		}
	}
	return true
}

// Stats reports cumulative counters for /metrics + debugging.
type Stats struct {
	IQSamplesSeen uint64
	PreamblesSeen uint64
	FramesDecoded uint64
	FramesEmitted uint64
	TrackerSize   int
}

func (r *Receiver) Stats() Stats {
	return Stats{
		IQSamplesSeen: r.samplesSeen.Load(),
		PreamblesSeen: r.preamblesSeen.Load(),
		FramesDecoded: r.framesDecoded.Load(),
		FramesEmitted: r.framesEmitted.Load(),
		TrackerSize:   r.tracker.Size(),
	}
}

// gcd computes the greatest common divisor via Euclid's algorithm.
func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

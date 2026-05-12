package conventional

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// DCS — Digital-Coded Squelch, also called DPL (Digital Private Line).
// A 134.4 baud sub-audible NRZ stream carrying a continuously-cycled
// 23-bit Golay(23,12,7) codeword. The 12 information bits decompose
// as 9 code bits (three octal digits, e.g. "023" → 000 010 011) plus
// a 3-bit fixed sync field ("100"). The transmitter loops the
// codeword indefinitely so a receiver can lock onto any of the 23
// cyclic rotations.
//
// Detection strategy in this file: FM discriminator → single-pole
// IIR low-pass at ~250 Hz (rejects audio band) → bit-rate integrator
// → 23-bit sliding window → compare against the 46 precomputed
// rotations (23 cyclic shifts × 2 polarities; some markets transmit
// DCS with inverted polarity) and declare a match when Hamming
// distance ≤ 2. The Golay(23,12,7) primitive lives in
// internal/radio/framing — same one used by P25 Phase 1 IMBE channel
// coding — so the codeword math is shared with the rest of the
// project.

// DCSDetector matches a single DCS code on a stream of IQ chunks.
// Construct via NewDCSDetector. Stateful — owns the demod / bit-
// recovery / sliding-window state across IQ chunks. Not safe for
// concurrent use; the conv scanner owns one detector per channel.
type DCSDetector struct {
	// FM discriminator state.
	last complex64

	// Single-pole IIR low-pass to roll off the audio band before
	// the bit integrator sees it.
	lpfAlpha float64
	lpfState float64

	// Bit-rate integrator. We accumulate the discriminator
	// output over a window of ~ samplesPerBit samples, then
	// slice on the sign of the integrated value.
	samplesPerBit  float64
	phaseInBit     float64 // 0..1, advances by 1/samplesPerBit each sample
	bitAccumulator float64

	// 23-bit sliding window of recovered bits. Bits enter at the
	// LSB; older bits shift into higher positions. Masked to 23
	// bits after every shift.
	bitWindow uint32
	bitsHave  int // how many bits received so far (saturates at 23)

	// Precomputed targets — both polarities × 23 rotations of the
	// expected codeword. We check every entry per new bit; popcount
	// is one CPU instruction and 46 comparisons per bit at 134.4
	// baud is invisible in the profiler.
	targets []uint32

	// distanceThreshold is the maximum Hamming distance from any
	// target rotation that still counts as a match. 2 is a sweet
	// spot empirically — tight enough that pure noise doesn't
	// false-trigger across the 46 targets, loose enough that a
	// real DCS signal with a single demod glitch still locks.
	distanceThreshold int

	present bool
	code    string
}

// DCSConfig holds the IQ sample rate + the DCS code to detect.
type DCSConfig struct {
	// SampleHz is the IQ sample rate (typically 2.4e6 for RTL-SDR).
	SampleHz float64
	// Code is the 3-digit octal DCS code (e.g. "023", "754"). The
	// 38 EIA codes are widely deployed; the standard accepts any
	// 3-digit octal value.
	Code string
	// AudioCutoffHz sets the single-pole IIR low-pass cutoff.
	// Defaults to 250 Hz — well above the 134.4 baud Nyquist
	// rate and below any voice formant.
	AudioCutoffHz float64
}

// NewDCSDetector builds a detector for the configured DCS code.
// Returns nil on bad config (empty code, non-octal digits, missing
// sample rate) — the scanner falls back to power-only squelch when
// the constructor returns nil.
func NewDCSDetector(cfg DCSConfig) *DCSDetector {
	if cfg.SampleHz <= 0 {
		return nil
	}
	target, err := dcsCodewordFromOctal(cfg.Code)
	if err != nil {
		return nil
	}
	if cfg.AudioCutoffHz <= 0 {
		cfg.AudioCutoffHz = 250
	}
	dt := 1.0 / cfg.SampleHz
	rc := 1.0 / (2 * math.Pi * cfg.AudioCutoffHz)
	alpha := dt / (rc + dt)
	const bitsPerSecond = 134.4
	return &DCSDetector{
		last:              complex(1, 0),
		lpfAlpha:          alpha,
		samplesPerBit:     cfg.SampleHz / bitsPerSecond,
		targets:           dcsRotations(target),
		distanceThreshold: 2,
		code:              cfg.Code,
	}
}

// SetDistanceThreshold tunes the match tolerance. Lower = fewer
// false positives at the cost of slower lock under noise; higher
// = quicker lock but more false alarms.
func (d *DCSDetector) SetDistanceThreshold(n int) {
	if n < 0 {
		n = 0
	}
	d.distanceThreshold = n
}

// Code returns the configured 3-digit octal DCS code.
func (d *DCSDetector) Code() string { return d.code }

// Present reports the latest detection state.
func (d *DCSDetector) Present() bool { return d != nil && d.present }

// Reset clears all internal state. Called by the scanner whenever
// it retunes.
func (d *DCSDetector) Reset() {
	if d == nil {
		return
	}
	d.last = complex(1, 0)
	d.lpfState = 0
	d.phaseInBit = 0
	d.bitAccumulator = 0
	d.bitWindow = 0
	d.bitsHave = 0
	d.present = false
}

// Process feeds an IQ chunk. Returns the most-recent Present()
// value as a convenience for callers that gate on a single call.
func (d *DCSDetector) Process(iq []complex64) bool {
	if d == nil || len(iq) == 0 {
		return d != nil && d.present
	}
	step := 1.0 / d.samplesPerBit
	for _, s := range iq {
		// FM discriminator: arg(z[n]·conj(z[n-1])).
		ar := real(s)*real(d.last) + imag(s)*imag(d.last)
		ai := imag(s)*real(d.last) - real(s)*imag(d.last)
		demod := math.Atan2(float64(ai), float64(ar))
		d.last = s

		// Audio-band rejection.
		d.lpfState = d.lpfState + d.lpfAlpha*(demod-d.lpfState)

		// Integrate over the bit window.
		d.bitAccumulator += d.lpfState
		d.phaseInBit += step
		if d.phaseInBit < 1.0 {
			continue
		}
		// One bit completed. Slice on the integrator sign.
		var bit uint32
		if d.bitAccumulator > 0 {
			bit = 1
		}
		d.bitWindow = ((d.bitWindow << 1) | bit) & dcsCodewordMask
		d.bitAccumulator = 0
		d.phaseInBit -= 1.0
		if d.bitsHave < 23 {
			d.bitsHave++
		}
		if d.bitsHave < 23 {
			continue
		}
		d.present = d.checkTargets()
	}
	return d.present
}

// checkTargets walks the precomputed rotation table and returns true
// when any entry is within distanceThreshold Hamming bits of the
// current window. Cheap: 46 XOR + popcount ops per bit decision.
func (d *DCSDetector) checkTargets() bool {
	for _, t := range d.targets {
		if bits.OnesCount32(d.bitWindow^t) <= d.distanceThreshold {
			return true
		}
	}
	return false
}

// --- internal helpers ---

const dcsCodewordMask uint32 = 0x7F_FF_FF // 23 bits

// dcsCodewordFromOctal converts a 3-digit octal DCS code (e.g. "023")
// into the 23-bit Golay(23,12,7) codeword that a transmitter would
// cycle on air. Layout of the 12 info bits passed to the Golay
// encoder: [octal-digit-1(3) | octal-digit-2(3) | octal-digit-3(3) |
// sync(3 = "100")] with octal-digit-1 in bits 11..9 and sync "100"
// in bits 2..0. The 11 parity bits land in the low end after Golay
// encoding (the framing package's GolayEncode23_12 emits
// [data | parity]).
func dcsCodewordFromOctal(code string) (uint32, error) {
	if len(code) != 3 {
		return 0, fmt.Errorf("dcs: code %q must be 3 octal digits", code)
	}
	var info uint16
	for i, r := range code {
		if r < '0' || r > '7' {
			return 0, fmt.Errorf("dcs: code %q must be octal 0..7", code)
		}
		d, err := strconv.ParseUint(string(r), 10, 8)
		if err != nil {
			return 0, fmt.Errorf("dcs: parse %q: %w", code, err)
		}
		info |= uint16(d) << (9 - (i+1)*3)
	}
	// Append the standard "100" sync trailer in the low 3 bits of
	// the 12-bit info field — bit 11..3 hold the octal-digit-1..3
	// pattern, bit 2 is fixed '1', bits 1..0 are '0'.
	info = (info << 3) | 0b100
	return framing.GolayEncode23_12(info), nil
}

// dcsRotations precomputes every cyclic rotation of cw (23 of them)
// plus the bit-inverse of each rotation (so the detector tolerates
// inverted-polarity transmitters without doubling the runtime check
// cost). Returns a 46-element slice — small enough that linear
// scanning beats any hashing scheme.
func dcsRotations(cw uint32) []uint32 {
	out := make([]uint32, 0, 46)
	r := cw & dcsCodewordMask
	for i := 0; i < 23; i++ {
		out = append(out, r)
		out = append(out, (^r)&dcsCodewordMask)
		// Cyclic left-shift by one bit inside a 23-bit field.
		msb := (r >> 22) & 1
		r = ((r << 1) | msb) & dcsCodewordMask
	}
	return out
}

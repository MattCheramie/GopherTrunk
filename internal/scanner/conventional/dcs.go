package conventional

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"time"
)

// DCS — Digital-Coded Squelch, also called DPL (Digital Private Line).
// A 134.4 baud sub-audible NRZ stream carrying a continuously-cycled
// 23-bit Golay(23,12,7) word. On air, bit by bit: the 9-bit code (the
// three octal digits, least significant bit first), the fixed 0 0 1,
// then 11 parity bits. The transmitter loops the word indefinitely, so
// a receiver can lock onto any of its 23 cyclic rotations.
//
// Detection: toneFrontEnd (decimate to ~48 kHz, channel filter, FM
// discriminator) → single-pole IIR low-pass at ~250 Hz → a slicing level
// halfway between the recent high and low (the discriminator carries the
// carrier offset as DC, which a sign slicer would read as all-ones) →
// integrate-and-dump over one bit period at dcsPhases staggered clock
// phases → per phase, a 23-bit sliding window compared against the 46
// precomputed rotations (23 cyclic shifts × 2 polarities) at Hamming
// distance ≤ 2. A match must then hold for dcsConfirmBits consecutive
// bits before the gate opens: a real DCS stream matches at every bit
// shift, random data does not.
//
// Issue #1184: the previous detector never matched a real radio. Its
// codeword put the code in the high bits and the sync as "100" in the
// low bits, and it slid bits in MSB-first; its test synthesizer
// transmitted that same invented layout, so every unit test passed.
// It also ran the discriminator on the whole 2.4 MHz SDR band and
// sliced on the raw sign, so a carrier offset alone pinned every bit.
// The codeword is now pinned against the published parity equations
// (dcs_reference_test.go), independently of this file.

// dcsPhases is how many staggered bit-clock phases are integrated in
// parallel. There is no clock recovery: one of four phases is always
// within 1/8 bit of the transmitter's, which is plenty at 134.4 baud.
const dcsPhases = 4

// dcsConfirmBits is how many consecutive bits a match must hold before
// the gate opens (~120 ms). A true stream matches at every bit; noise
// that happens to match one window keeps matching with probability ~1/2
// per bit, so 16 bits cuts the false-open rate by ~2^16.
const dcsConfirmBits = 16

// dcsMinDwell covers the detector's first report: half a word to set
// the slicing level, a full 23-bit window, dcsConfirmBits, and the
// front end's filter delay.
const dcsMinDwell = 600 * time.Millisecond

// dcsBitRate is the DCS signalling rate in bits per second.
const dcsBitRate = 134.4

// DCSDetector matches a single DCS code on a stream of IQ chunks.
// Construct via NewDCSDetector. Stateful — owns the demod / bit-
// recovery / sliding-window state across IQ chunks. Not safe for
// concurrent use; the conv scanner owns one detector per channel.
type DCSDetector struct {
	fe *toneFrontEnd

	// Single-pole IIR low-pass to roll off the audio band before
	// the bit integrator sees it.
	lpfAlpha float64
	lpfState float64

	// Slicing level: halfway between the high and low of the last two
	// half-word blocks. blkN counts samples into the current block.
	blkLen     int
	blkN       int
	blkHi      float64
	blkLo      float64
	prevHi     float64
	prevLo     float64
	prevValid  bool
	mid        float64
	levelValid bool

	samplesPerBit float64
	phases        [dcsPhases]dcsPhase

	// Precomputed targets — both polarities × 23 rotations of the
	// expected word.
	targets []uint32

	// distanceThreshold is the maximum Hamming distance from any
	// target rotation that still counts as a match.
	distanceThreshold int

	// sinceMatch counts samples since any phase last matched; the gate
	// closes once it exceeds one word.
	sinceMatch int
	wordLen    int

	present bool
	// inverted records the polarity of the match that opened the gate:
	// false when a 1 bit arrived as positive frequency deviation.
	inverted bool
	code     string

	// polarity is the NRZ sense the gate accepts. The other polarity
	// is still tracked, but only to flag a likely N/I misconfiguration
	// (oppositeHeard); it never opens the gate.
	polarity      DCSPolarity
	oppositeHeard bool
}

// DCSPolarity selects which NRZ sense of the configured code opens the
// gate. On air (#1184, reporter's Kenwood NX-300/NX-5000 and service
// monitor): a radio set to "N" (normal, e.g. D025N) sends the code's 1
// bits as POSITIVE frequency deviation, and "I" (inverted, D025I) as
// negative. That is what normal/inverted mean here.
type DCSPolarity int

const (
	// DCSPolarityNormal accepts only the normal (N) sense — the default,
	// matching a radio's plain "D023" / "D023N" setting.
	DCSPolarityNormal DCSPolarity = iota
	// DCSPolarityInverted accepts only the inverted (I) sense.
	DCSPolarityInverted
	// DCSPolarityBoth accepts either sense. A code's inverted word is a
	// rotation of a different code's normal word (023I ≡ 047N), so
	// "both" on 023 also opens for 047N.
	DCSPolarityBoth
)

// ParseDCSPolarity maps the config spelling ("" / "normal" / "n",
// "inverted" / "i", "both") to a DCSPolarity. "" is normal.
func ParseDCSPolarity(s string) (DCSPolarity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "normal", "n":
		return DCSPolarityNormal, nil
	case "inverted", "i":
		return DCSPolarityInverted, nil
	case "both":
		return DCSPolarityBoth, nil
	}
	return 0, fmt.Errorf("dcs polarity %q must be normal|inverted|both", s)
}

// String returns the config spelling.
func (p DCSPolarity) String() string {
	switch p {
	case DCSPolarityInverted:
		return "inverted"
	case DCSPolarityBoth:
		return "both"
	}
	return "normal"
}

// accepts reports whether a match of the given sense may open the gate.
func (p DCSPolarity) accepts(inverted bool) bool {
	switch p {
	case DCSPolarityBoth:
		return true
	case DCSPolarityInverted:
		return inverted
	}
	return !inverted
}

// dcsPhase is one integrate-and-dump bit clock and its bit window.
type dcsPhase struct {
	pos    float64 // position in the current bit, 0..1
	acc    float64 // integrated (sample − slicing level)
	window uint32  // last 23 bits, newest at bit 22 (on-air order)
	have   int     // bits received, saturating at 23
	streak int     // consecutive matching bits (accepted polarity)
	// oppStreak counts consecutive matches of the code in the polarity
	// the gate does NOT accept — evidence of an N/I mismatch.
	oppStreak int
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
	// Defaults to 250 Hz — above the 134.4 baud NRZ fundamental
	// and below the voice band.
	AudioCutoffHz float64
	// Polarity selects the NRZ sense that opens the gate. The zero
	// value is DCSPolarityNormal.
	Polarity DCSPolarity
}

// NewDCSDetector builds a detector for the configured DCS code.
// Returns nil on bad config (empty code, non-octal digits, missing
// sample rate) — the scanner falls back to power-only squelch when
// the constructor returns nil.
//
// Only cfg.Polarity opens the gate (normal by default). A code's
// inverted word is a rotation of a different code's normal word
// (023I ≡ 047N, and so 023N ≡ 047I), so every gate also opens on its
// alias — inherent in DCS, not tellable apart on air — and a "both"
// gate on 023 opens for 023N/047I and 023I/047N alike.
func NewDCSDetector(cfg DCSConfig) *DCSDetector {
	if cfg.SampleHz <= 0 {
		return nil
	}
	if cfg.Polarity < DCSPolarityNormal || cfg.Polarity > DCSPolarityBoth {
		return nil
	}
	target, err := dcsCodewordFromOctal(cfg.Code)
	if err != nil {
		return nil
	}
	if cfg.AudioCutoffHz <= 0 {
		cfg.AudioCutoffHz = 250
	}
	fe := newToneFrontEnd(cfg.SampleHz)
	spb := fe.rate / dcsBitRate
	d := &DCSDetector{
		fe:                fe,
		lpfAlpha:          onePoleAlpha(cfg.AudioCutoffHz, fe.rate),
		blkLen:            int(spb * 23 / 2),
		samplesPerBit:     spb,
		targets:           dcsRotations(target),
		distanceThreshold: 2,
		wordLen:           int(spb * 23),
		code:              cfg.Code,
		polarity:          cfg.Polarity,
	}
	d.Reset()
	return d
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
	d.fe.reset()
	d.lpfState = 0
	d.blkN = 0
	d.blkHi, d.blkLo = math.Inf(-1), math.Inf(1)
	d.prevValid = false
	d.levelValid = false
	d.mid = 0
	for i := range d.phases {
		d.phases[i] = dcsPhase{pos: float64(i) / dcsPhases}
	}
	d.sinceMatch = 0
	d.present = false
	d.inverted = false
	d.oppositeHeard = false
}

// Process feeds an IQ chunk. Returns the most-recent Present()
// value as a convenience for callers that gate on a single call.
func (d *DCSDetector) Process(iq []complex64) bool {
	if d == nil || len(iq) == 0 {
		return d != nil && d.present
	}
	step := 1.0 / d.samplesPerBit
	for _, demod := range d.fe.process(iq) {
		d.lpfState += d.lpfAlpha * (demod - d.lpfState)
		x := d.lpfState
		d.trackLevel(x)
		if !d.levelValid {
			continue
		}
		d.sinceMatch++
		for i := range d.phases {
			p := &d.phases[i]
			p.acc += x - d.mid
			p.pos += step
			if p.pos < 1 {
				continue
			}
			p.pos--
			var bit uint32
			if p.acc > 0 {
				bit = 1
			}
			p.acc = 0
			// On-air order: the oldest bit ends up in bit 0, so a window
			// aligned to a word boundary reads exactly the codeword.
			p.window = (p.window >> 1) | bit<<22
			if p.have < 23 {
				p.have++
				continue
			}
			match, opposite, inverted := d.checkTargets(p.window)
			if !match {
				p.streak = 0
				if opposite {
					p.oppStreak++
					if p.oppStreak >= dcsConfirmBits {
						d.oppositeHeard = true
					}
				} else {
					p.oppStreak = 0
				}
				continue
			}
			p.oppStreak = 0
			p.streak++
			d.sinceMatch = 0
			if p.streak >= dcsConfirmBits && !d.present {
				d.present = true
				d.inverted = inverted
			}
		}
		if d.present && d.sinceMatch > d.wordLen {
			d.present = false
		}
	}
	return d.present
}

// trackLevel maintains the slicing level: the midpoint of the high and
// low over the last one-to-two half-word blocks. A DCS word always holds
// both bit values, so the extremes bracket the NRZ levels whatever DC
// the carrier offset adds.
func (d *DCSDetector) trackLevel(x float64) {
	d.blkHi = math.Max(d.blkHi, x)
	d.blkLo = math.Min(d.blkLo, x)
	d.blkN++
	if d.blkN < d.blkLen {
		return
	}
	hi, lo := d.blkHi, d.blkLo
	if d.prevValid {
		hi = math.Max(hi, d.prevHi)
		lo = math.Min(lo, d.prevLo)
	}
	d.mid = (hi + lo) / 2
	d.levelValid = true
	d.prevHi, d.prevLo, d.prevValid = d.blkHi, d.blkLo, true
	d.blkHi, d.blkLo = math.Inf(-1), math.Inf(1)
	d.blkN = 0
}

// checkTargets reports whether w is within distanceThreshold Hamming
// bits of a target rotation in an accepted polarity (match, with that
// rotation's sense in inverted — odd entries of dcsRotations are the
// complemented polarity), or failing that, of one in the polarity the
// gate does not accept (opposite). Accepted rotations are checked in
// full first, so a word near both never reads as a mismatch. Cheap:
// 46 XOR + popcount ops.
func (d *DCSDetector) checkTargets(w uint32) (match, opposite, inverted bool) {
	for i, t := range d.targets {
		inv := i%2 == 1
		if d.polarity.accepts(inv) && bits.OnesCount32(w^t) <= d.distanceThreshold {
			return true, false, inv
		}
	}
	for i, t := range d.targets {
		if !d.polarity.accepts(i%2 == 1) && bits.OnesCount32(w^t) <= d.distanceThreshold {
			return false, true, false
		}
	}
	return false, false, false
}

// Polarity returns the NRZ sense this detector accepts.
func (d *DCSDetector) Polarity() DCSPolarity { return d.polarity }

// TakeOppositeHeard reports whether, since the last call or Reset, the
// configured code was received in the polarity the gate does NOT accept
// (held for the same dcsConfirmBits a gate-open needs) — the signature
// of a channel configured N for a radio sending I, or vice versa. The
// flag is cleared by the call.
func (d *DCSDetector) TakeOppositeHeard() bool {
	if d == nil || !d.oppositeHeard {
		return false
	}
	d.oppositeHeard = false
	return true
}

// Inverted reports the polarity of the match that opened the gate:
// false when the code's 1 bits arrived as positive frequency deviation,
// true when they arrived as negative. Meaningful only while Present.
//
// On air a radio's "N" setting reads false and "I" reads true (#1184:
// D025N → false, D025I → true on the reporter's Kenwoods).
func (d *DCSDetector) Inverted() bool { return d != nil && d.inverted }

// --- internal helpers ---

const dcsCodewordMask uint32 = 0x7F_FF_FF // 23 bits

// dcsCodewordFromOctal converts a 3-digit octal DCS code (e.g. "023")
// into the 23-bit word a transmitter cycles on air, with bit i the i-th
// bit sent: bits 0..8 the code (least significant bit first), bits
// 9..11 the fixed 0 0 1, bits 12..22 the Golay parity. The parity is
// the remainder of the systematic Golay(23,12) encoding with generator
// x^11+x^9+x^7+x^6+x^5+x+1 (0xAE3), taken over the on-air bit order;
// dcs_reference_test.go pins it against the published parity equations
// for all 512 codes (023 → 0x763813).
func dcsCodewordFromOctal(code string) (uint32, error) {
	if len(code) != 3 {
		return 0, fmt.Errorf("dcs: code %q must be 3 octal digits", code)
	}
	for _, r := range code {
		if r < '0' || r > '7' {
			return 0, fmt.Errorf("dcs: code %q must be octal 0..7", code)
		}
	}
	c, err := strconv.ParseUint(code, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("dcs: parse %q: %w", code, err)
	}
	data := uint32(c) | 1<<11 // code + the fixed 0 0 1
	// Systematic cyclic encoding over the transmit order: with bit i the
	// coefficient of x^(22-i), the data occupies x^22..x^11 and the
	// parity x^10..x^0 is (data · x^11) mod g(x).
	var reg uint32
	for i := 0; i < 12; i++ {
		fb := (data>>uint(i))&1 ^ (reg>>10)&1
		reg = (reg << 1) & 0x7FF
		if fb != 0 {
			reg ^= dcsGolayPoly & 0x7FF
		}
	}
	w := data
	for i := 0; i < 11; i++ {
		w |= ((reg >> uint(10-i)) & 1) << uint(12+i)
	}
	return w, nil
}

// dcsGolayPoly is the Golay(23,12) generator x^11+x^9+x^7+x^6+x^5+x+1.
const dcsGolayPoly = 0xAE3

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

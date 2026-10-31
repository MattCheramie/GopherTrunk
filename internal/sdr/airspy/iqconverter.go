package airspy

// The Airspy R2 / Mini are real-sampling receivers: the firmware streams the
// bare ADC output — unpacked little-endian uint16 real samples (12-bit, DC at
// 2048) — at twice the advertised IQ rate. Producing complex baseband is a
// host-side job. This file implements that conversion from first principles
// using the standard real-to-IQ technique for a real-IF front end:
//
//  1. remove the ADC DC bias with a leaky high-pass,
//  2. translate by Fs/4 (a free, multiplier-less mix by (-j)^n),
//  3. split into the two polyphase half-rate streams and apply a half-band
//     filter pair — a symmetric low-pass on the in-phase branch, a matched
//     integer delay on the quadrature branch — which together reject the
//     mirror image, and
//  4. decimate by two.
//
// N real input samples therefore yield N/2 complex64 outputs at half the
// hardware rate, i.e. the configured IQ sample rate. The converter is
// stateful: its filter memory and DC tracker carry across USB packets so
// there is no discontinuity at buffer boundaries.
//
// The half-band coefficients below are a standard 47-tap half-band design
// (zero on the odd taps except the unity-ish centre); only the 24 non-trivial
// taps participate in the in-phase FIR, with the 0.5 centre tap folded into
// the quadrature branch during the Fs/4 mix.
//
// Cost matters here: a Mini streams 6 M real samples/s, and this runs on one
// goroutine (see Device.StreamIQ). The in-phase FIR evaluates the 24 taps as
// 12 symmetric pairs over a mirrored (wrap-free) window, the four-sample mix
// cycle is unrolled so the phase switch is decided once per cycle, and the
// uint16 decode is inline — about 4× the throughput of the per-sample,
// ring-buffer form it replaced on x86, which on a Raspberry Pi 4 was a sizeable
// share of a core at 3 MS/s (#1321). The output is numerically the same
// filter; only the float32 summation order differs (TestIQConverterMatchesReference
// bounds it).

const (
	hbTaps = 24   // non-trivial half-band taps used by the in-phase FIR
	hbHalf = 12   // quadrature delay length (hbTaps / 2)
	dcLeak = 0.01 // leaky DC-blocker coefficient

	// defaultADCBits is the Airspy's (and old HydraSDR firmware's) ADC
	// width: 12-bit unsigned, DC at 2048, carried in a little-endian uint16.
	defaultADCBits = 12
)

// hbKernel is the symmetric half-band low-pass applied to the in-phase branch
// (the even-indexed taps of the canonical 47-tap half-band; the largest taps
// sit in the middle, giving the filter its group delay).
var hbKernel = [hbTaps]float32{
	-0.000998606272947510, 0.001695637278417295, -0.003054430179754289,
	0.005055504379767936, -0.007901319195893647, 0.011873357051047719,
	-0.017411159379930066, 0.025304817427568772, -0.037225225204559217,
	0.057533286997004301, -0.102327462004259350, 0.317034472508947400,
	0.317034472508947400, -0.102327462004259350, 0.057533286997004301,
	-0.037225225204559217, 0.025304817427568772, -0.017411159379930066,
	0.011873357051047719, -0.007901319195893647, 0.005055504379767936,
	-0.003054430179754289, 0.001695637278417295, -0.000998606272947510,
}

// iqConverter holds the running state of the real-to-IQ conversion.
//
// processRaw is called from exactly one goroutine — the per-stream converter
// goroutine in Device.StreamIQ, which drains the reaper's rawCh and runs the
// conversion in packet order. It is therefore NOT safe for concurrent use, and
// the driver must never call it from more than one goroutine. (Earlier the
// macOS transport's 32 concurrent bulk-IN reapers each called processRaw on one
// shared converter, so a mutex was needed to stop them racing on the mutable
// filter state below — and that inline, mutex-serialised conversion on the read
// hot path was what starved the USB read queue and made macOS abort the pipe at
// 10 MS/s. StreamIQ now moves conversion off the reapers onto this single
// goroutine, so the lock is gone.)
type iqConverter struct {
	// mid is the unsigned ADC midpoint, 2^(bits-1): it is both the DC bias
	// removed from each sample and the scale that normalises to roughly
	// [-1, 1). byteSamples marks an 8-bit ADC, which the HydraSDR firmware
	// streams one byte per sample instead of one uint16 (libhydrasdr's
	// iqconverter_process_u8 path).
	mid         float32
	invMid      float32 // 1/mid; exact, mid is a power of two
	byteSamples bool

	avg   float32 // leaky DC-blocker accumulator
	phase int     // Fs/4 mix phase, 0..3, persists across packets
	// iwin is the in-phase FIR window stored twice (mirrored ring): the newest
	// hbTaps samples are always the contiguous run iwin[iwinPos : iwinPos+hbTaps],
	// oldest first, so the FIR never wraps.
	iwin    [2 * hbTaps]float32
	iwinPos int             // ring write position, 0..hbTaps-1
	lastI   float32         // in-phase output awaiting its paired quadrature
	qline   [hbHalf]float32 // quadrature delay line
	qpos    int             // delay-line read/write cursor
}

// newIQConverter returns a converter for the 12-bit Airspy ADC in its reset
// state (silent filter memory, phase 0).
func newIQConverter() *iqConverter { return newIQConverterADC(defaultADCBits) }

// newIQConverterADC returns a converter for an unsigned ADC of the given
// width. The HydraSDR firmware reports 8, 10, 12, 14 or 16 bits per sample
// rate; anything outside that set (including 0, "not reported") means the
// 12-bit default.
func newIQConverterADC(bits uint8) *iqConverter {
	switch bits {
	case 8, 10, 12, 14, 16:
	default:
		bits = defaultADCBits
	}
	mid := float32(int(1) << (bits - 1))
	return &iqConverter{mid: mid, invMid: 1 / mid, byteSamples: bits == 8}
}

// processRaw decodes one bulk-IN payload of real ADC samples (little-endian
// uint16, or one byte each for an 8-bit ADC) into complex64 baseband. Two
// real samples collapse to one complex sample, so a uint16 payload yields
// len(buf)/4 complex samples.
// iqSamplesPerChunk is how many IQ samples processRaw yields from a raw
// payload of n bytes: a real sample is 2 bytes (1 on an 8-bit ADC) and the
// Fs/4 mix halves the count.
func (c *iqConverter) iqSamplesPerChunk(n int) int {
	if c.byteSamples {
		return n / 2
	}
	return n / 4
}

func (c *iqConverter) processRaw(buf []byte) []complex64 {
	nReal := len(buf) / 2
	if c.byteSamples {
		nReal = len(buf)
	}
	out := make([]complex64, 0, nReal/2+1)
	i := 0
	// Bring the phase to 0 one sample at a time (a payload is normally a
	// multiple of four samples, so this runs only after an odd-sized one).
	for ; i < nReal && c.phase != 0; i++ {
		out = c.step(out, c.decode(buf, i))
	}
	// Unrolled four-sample mix cycle at phase 0: I(-), Q(-), I(+), Q(+).
	for ; i+4 <= nReal; i += 4 {
		c.pushI(-c.decode(buf, i))
		c.lastI = c.firI()
		qOut := c.pushQ(-0.5 * c.decode(buf, i+1))
		out = append(out, complex(c.lastI, qOut))
		c.pushI(c.decode(buf, i+2))
		c.lastI = c.firI()
		qOut = c.pushQ(0.5 * c.decode(buf, i+3))
		out = append(out, complex(c.lastI, qOut))
	}
	for ; i < nReal; i++ {
		out = c.step(out, c.decode(buf, i))
	}
	return out
}

// decode reads real sample i (a little-endian uint16, or one byte for an
// 8-bit ADC), removes the ADC midpoint, scales to ~[-1, 1) and runs the leaky
// DC blocker. For the 12-bit default this is (raw - 2048) * (1/2048), exactly
// the constant form it replaced.
func (c *iqConverter) decode(buf []byte, i int) float32 {
	var raw float32
	if c.byteSamples {
		raw = float32(buf[i])
	} else {
		raw = float32(uint16(buf[2*i]) | uint16(buf[2*i+1])<<8)
	}
	x := (raw - c.mid) * c.invMid
	x -= c.avg
	c.avg += dcLeak * x
	return x
}

// step runs one real sample through the mix / filter at the current phase and
// advances it: the generic per-sample path the unrolled loop is equivalent to.
func (c *iqConverter) step(out []complex64, x float32) []complex64 {
	switch c.phase {
	case 0, 2:
		// In-phase branch. The Fs/4 mix alternates the sign of the even
		// samples; push into the FIR and recompute its output.
		if c.phase == 0 {
			x = -x
		}
		c.pushI(x)
		c.lastI = c.firI()
	case 1, 3:
		// Quadrature branch. The mix alternates sign and folds in the 0.5
		// half-band centre tap; the matched delay aligns it with the in-phase
		// FIR's group delay, then we emit one sample.
		q := 0.5 * x
		if c.phase == 1 {
			q = -q
		}
		out = append(out, complex(c.lastI, c.pushQ(q)))
	}
	if c.phase++; c.phase >= 4 {
		c.phase = 0
	}
	return out
}

// pushI stores the newest in-phase sample in the mirrored FIR window.
func (c *iqConverter) pushI(v float32) {
	c.iwin[c.iwinPos] = v
	c.iwin[c.iwinPos+hbTaps] = v
	if c.iwinPos++; c.iwinPos >= hbTaps {
		c.iwinPos = 0
	}
}

// pushQ writes q into the quadrature delay line and returns the sample that
// falls out of it (hbHalf samples old).
func (c *iqConverter) pushQ(q float32) float32 {
	qOut := c.qline[c.qpos]
	c.qline[c.qpos] = q
	if c.qpos++; c.qpos >= hbHalf {
		c.qpos = 0
	}
	return qOut
}

// firI evaluates the symmetric half-band FIR over the current window:
// hbKernel[0] weights the newest sample and hbKernel[hbTaps-1] the oldest, and
// since hbKernel[j] == hbKernel[hbTaps-1-j] each coefficient multiplies the sum
// of its two samples — 12 multiplies for 24 taps.
func (c *iqConverter) firI() float32 {
	w := c.iwin[c.iwinPos : c.iwinPos+hbTaps] // oldest … newest
	var acc float32
	for j := 0; j < hbHalf; j++ {
		// newest-side sample for tap j is w[hbTaps-1-j]; oldest-side is w[j].
		acc += hbKernel[j] * (w[hbTaps-1-j] + w[j])
	}
	return acc
}

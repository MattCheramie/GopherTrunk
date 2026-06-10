package airspy

import "encoding/binary"

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

const (
	hbTaps = 24   // non-trivial half-band taps used by the in-phase FIR
	hbHalf = 12   // quadrature delay length (hbTaps / 2)
	dcLeak = 0.01 // leaky DC-blocker coefficient
	dcBias = 2048 // 12-bit unsigned ADC midpoint
	dcFull = 2048 // normalisation to roughly [-1, 1)
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
type iqConverter struct {
	avg     float32         // leaky DC-blocker accumulator
	phase   int             // Fs/4 mix phase, 0..3, persists across packets
	iwin    [hbTaps]float32 // in-phase FIR window (newest written at iwinPos)
	iwinPos int             // next write position in iwin
	lastI   float32         // in-phase output awaiting its paired quadrature
	qline   [hbHalf]float32 // quadrature delay line
	qpos    int             // delay-line read/write cursor
}

// newIQConverter returns a converter in its reset state. The zero value is a
// valid starting point (silent filter memory, phase 0).
func newIQConverter() *iqConverter { return &iqConverter{} }

// processRaw decodes one bulk-IN payload of little-endian uint16 real samples
// into complex64 baseband. It returns len(buf)/4 complex samples (two real
// samples collapse to one complex sample).
func (c *iqConverter) processRaw(buf []byte) []complex64 {
	nReal := len(buf) / 2
	out := make([]complex64, 0, nReal/2+1)
	for i := 0; i < nReal; i++ {
		x := (float32(binary.LittleEndian.Uint16(buf[2*i:])) - dcBias) / dcFull

		// Leaky DC blocker: removes the residual ADC bias before the mix.
		x -= c.avg
		c.avg += dcLeak * x

		switch c.phase {
		case 0, 2:
			// In-phase branch. The Fs/4 mix alternates the sign of the
			// even samples; push into the FIR and recompute its output.
			if c.phase == 0 {
				x = -x
			}
			c.pushI(x)
			c.lastI = c.firI()
		case 1, 3:
			// Quadrature branch. The mix alternates sign and folds in the
			// 0.5 half-band centre tap; the matched delay aligns it with
			// the in-phase FIR's group delay, then we emit one sample.
			q := 0.5 * x
			if c.phase == 1 {
				q = -q
			}
			qOut := c.qline[c.qpos]
			c.qline[c.qpos] = q
			if c.qpos++; c.qpos >= hbHalf {
				c.qpos = 0
			}
			out = append(out, complex(c.lastI, qOut))
		}
		if c.phase++; c.phase >= 4 {
			c.phase = 0
		}
	}
	return out
}

// pushI stores the newest in-phase sample in the circular FIR window.
func (c *iqConverter) pushI(v float32) {
	c.iwin[c.iwinPos] = v
	if c.iwinPos++; c.iwinPos >= hbTaps {
		c.iwinPos = 0
	}
}

// firI evaluates the symmetric half-band FIR over the current window, with
// hbKernel[0] weighting the newest sample and hbKernel[hbTaps-1] the oldest.
func (c *iqConverter) firI() float32 {
	var acc float32
	idx := c.iwinPos - 1
	if idx < 0 {
		idx += hbTaps
	}
	for j := 0; j < hbTaps; j++ {
		acc += hbKernel[j] * c.iwin[idx]
		if idx--; idx < 0 {
			idx += hbTaps
		}
	}
	return acc
}

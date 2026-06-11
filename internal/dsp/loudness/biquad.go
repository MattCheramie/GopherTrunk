// Package loudness implements ITU-R BS.1770-4 / EBU R128 integrated
// loudness measurement and the gain math used for per-call loudness
// normalization. It is pure Go (no cgo, no ffmpeg) and operates on mono
// float64 PCM in the range [-1, 1].
//
// The measurement chain is the standard one: a two-stage "K-weighting"
// pre-filter, 400 ms mean-square gating blocks at 75% overlap, an
// absolute -70 LUFS gate and a relative -10 LU gate, then the gated mean
// expressed in LUFS. See EBU Tech 3341 and ITU-R BS.1770-4.
package loudness

import "math"

// biquad is a second-order IIR section in Direct Form II transposed.
// The coefficients follow the convention
//
//	y[n] = b0*x[n] + b1*x[n-1] + b2*x[n-2] - a1*y[n-1] - a2*y[n-2]
type biquad struct {
	b0, b1, b2 float64
	a1, a2     float64
	z1, z2     float64
}

// process filters s in place.
func (f *biquad) process(s []float64) {
	b0, b1, b2 := f.b0, f.b1, f.b2
	a1, a2 := f.a1, f.a2
	z1, z2 := f.z1, f.z2
	for i, x := range s {
		y := b0*x + z1
		z1 = b1*x - a1*y + z2
		z2 = b2*x - a2*y
		s[i] = y
	}
	f.z1, f.z2 = z1, z2
}

// kWeighting returns the two BS.1770 K-weighting stages designed for the
// given sample rate: stage 1 is the head-related high-shelf "pre-filter"
// (~+4 dB above ~1.5 kHz) and stage 2 is the RLB high-pass (~38 Hz).
//
// The coefficients are derived by bilinear transform of the documented
// analog prototypes (the same derivation libebur128 uses), so they
// reproduce the published 48 kHz reference coefficients and remain valid
// at GopherTrunk's 8 kHz voice rate.
func kWeighting(sampleRate int) (shelf, hpf biquad) {
	fs := float64(sampleRate)

	// Stage 1: high-shelf pre-filter.
	{
		const (
			f0 = 1681.974450955533
			g  = 3.999843853973347
			q  = 0.7071752369554196
		)
		k := math.Tan(math.Pi * f0 / fs)
		vh := math.Pow(10.0, g/20.0)
		vb := math.Pow(vh, 0.4996667741545416)
		a0 := 1.0 + k/q + k*k
		shelf = biquad{
			b0: (vh + vb*k/q + k*k) / a0,
			b1: 2.0 * (k*k - vh) / a0,
			b2: (vh - vb*k/q + k*k) / a0,
			a1: 2.0 * (k*k - 1.0) / a0,
			a2: (1.0 - k/q + k*k) / a0,
		}
	}

	// Stage 2: RLB high-pass.
	{
		const (
			f0 = 38.13547087602444
			q  = 0.5003270373238773
		)
		k := math.Tan(math.Pi * f0 / fs)
		a0 := 1.0 + k/q + k*k
		hpf = biquad{
			b0: 1.0,
			b1: -2.0,
			b2: 1.0,
			a1: 2.0 * (k*k - 1.0) / a0,
			a2: (1.0 - k/q + k*k) / a0,
		}
	}
	return shelf, hpf
}

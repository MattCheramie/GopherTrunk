package demod

import (
	"math"
	"math/rand"
)

// Impairments describes the front-end / RF-path degradations a real
// SDR capture carries that a synthetic modulator output does not. The
// modulators in this package emit a mathematically ideal IQ stream;
// ApplyImpairments degrades it the way an RTL-SDR + antenna chain
// would, so a decoder test can exercise the failure modes that only
// surface on live hardware.
//
// A zero-valued Impairments is a no-op — ApplyImpairments copies the
// input through unchanged. Each field is independent; combine them to
// model a realistic capture.
type Impairments struct {
	// FreqOffsetHz is a residual carrier frequency offset — tuner
	// crystal ppm error, or a channel that is not exactly centred in
	// the SDR passband. Applied as a per-sample phase ramp.
	FreqOffsetHz float64
	// DCOffset is a constant complex term added to every sample: the
	// R820T2 / E4000 "DC spike". Magnitude is relative to the
	// unit-scale modulator output (0.1 ≈ -20 dBFS).
	DCOffset complex64
	// IQGainImbalance is the Q-channel gain relative to I (1.0, or the
	// zero value, means no imbalance) — quadrature amplitude mismatch
	// from an imperfect analog downconverter.
	IQGainImbalance float64
	// IQPhaseSkewRad is the quadrature phase error in radians: how far
	// the Q axis sits off perfect 90° from I (0 = none).
	IQPhaseSkewRad float64
	// SNRdB is the target signal-to-noise ratio for additive white
	// Gaussian noise, measured against the clean input signal power.
	// Zero or negative adds no noise.
	SNRdB float64
	// Seed makes the AWGN draw reproducible across runs.
	Seed int64
}

// ApplyImpairments returns a new IQ slice degraded by imp; the input is
// not mutated. Stages run in physical capture order: IQ imbalance (the
// analog quadrature mixer) → DC offset (summed at the ADC) → carrier
// frequency offset (LO error) → AWGN (thermal noise, added last).
func ApplyImpairments(iq []complex64, sampleRateHz float64, imp Impairments) []complex64 {
	out := make([]complex64, len(iq))
	copy(out, iq)
	if len(out) == 0 {
		return out
	}

	// IQ gain / phase imbalance: keep I pure, distort Q.
	gain := imp.IQGainImbalance
	if gain == 0 {
		gain = 1
	}
	if gain != 1 || imp.IQPhaseSkewRad != 0 {
		sinSkew := math.Sin(imp.IQPhaseSkewRad)
		cosSkew := math.Cos(imp.IQPhaseSkewRad)
		for i, s := range out {
			iCh := float64(real(s))
			qCh := float64(imag(s))
			out[i] = complex(
				float32(iCh),
				float32(gain*(qCh*cosSkew+iCh*sinSkew)),
			)
		}
	}

	// DC offset: a constant complex term.
	if imp.DCOffset != 0 {
		for i := range out {
			out[i] += imp.DCOffset
		}
	}

	// Carrier frequency offset: a per-sample phase ramp.
	if imp.FreqOffsetHz != 0 && sampleRateHz > 0 {
		w := 2 * math.Pi * imp.FreqOffsetHz / sampleRateHz
		for i := range out {
			ph := w * float64(i)
			out[i] *= complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
		}
	}

	// AWGN scaled to SNRdB against the clean input signal power.
	if imp.SNRdB > 0 {
		var power float64
		for _, s := range iq {
			power += float64(real(s))*float64(real(s)) + float64(imag(s))*float64(imag(s))
		}
		power /= float64(len(iq))
		if power > 0 {
			noisePower := power / math.Pow(10, imp.SNRdB/10)
			// Complex Gaussian: split the variance across I and Q.
			sigma := math.Sqrt(noisePower / 2)
			rng := rand.New(rand.NewSource(imp.Seed))
			for i := range out {
				out[i] += complex(
					float32(rng.NormFloat64()*sigma),
					float32(rng.NormFloat64()*sigma),
				)
			}
		}
	}

	return out
}

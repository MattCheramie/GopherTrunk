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
	// Multipath is a complex channel FIR modelling a multipath /
	// simulcast propagation path: the received signal becomes
	// y[n] = Σ Multipath[k]·x[n-k], where Multipath[k] is the complex
	// gain of the copy arriving k samples late (Multipath[0] is the
	// main path). Nil or empty applies no multipath. This is the
	// inter-symbol-interference source a single-transmitter flat-fading
	// model cannot produce — it reproduces what the overlapping,
	// differently-delayed transmitters of a simulcast system do to a
	// control channel.
	Multipath []complex64
	// FreqOffsetHz is a residual carrier frequency offset — tuner
	// crystal ppm error, or a channel that is not exactly centred in
	// the SDR passband. Applied as a per-sample phase ramp.
	FreqOffsetHz float64
	// FreqDriftHzPerSec is a linear carrier-frequency drift in Hz per
	// second, added on top of FreqOffsetHz: the instantaneous offset is
	// FreqOffsetHz + FreqDriftHzPerSec·t. It models a tuner whose LO
	// wanders as it warms up — the slow drift a fixed coarse seed cannot
	// track and a fine carrier loop must follow (issue #492). Zero = none.
	FreqDriftHzPerSec float64
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
	// Scale multiplies the whole IQ stream by a constant amplitude
	// factor, modelling the SDR front-end / ADC gain. The synthetic
	// modulators emit a fixed unit-ish amplitude; a real capture's level
	// swings with the tuner gain setting. Zero (the zero value) is
	// treated as 1.0 — no scaling.
	Scale float64
}

// ApplyImpairments returns a new IQ slice degraded by imp; the input is
// not mutated. Stages run in physical capture order: multipath channel
// → IQ imbalance (the analog quadrature mixer) → DC offset (summed at
// the ADC) → carrier frequency offset (LO error) → AWGN (thermal noise)
// → front-end gain scale, applied last so signal and noise scale
// together.
func ApplyImpairments(iq []complex64, sampleRateHz float64, imp Impairments) []complex64 {
	out := make([]complex64, len(iq))
	copy(out, iq)
	if len(out) == 0 {
		return out
	}

	// Multipath / simulcast channel: convolve with the channel FIR
	// before any receiver-side impairment (multipath is a propagation
	// effect, upstream of the LO and front-end). out currently mirrors
	// iq; rebuild it as y[n] = Σ Multipath[k]·iq[n-k].
	if len(imp.Multipath) > 0 {
		for n := range out {
			var acc complex64
			for k, tap := range imp.Multipath {
				if n-k < 0 {
					break
				}
				acc += tap * iq[n-k]
			}
			out[n] = acc
		}
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

	// Carrier frequency offset (+ optional linear drift). A constant offset
	// is a per-sample phase ramp; a drift d (Hz/s) makes the instantaneous
	// frequency FreqOffsetHz + d·t, whose integral adds a quadratic phase
	// term ½·(2π·d)·t² (t = i/sampleRateHz).
	if (imp.FreqOffsetHz != 0 || imp.FreqDriftHzPerSec != 0) && sampleRateHz > 0 {
		for i := range out {
			t := float64(i) / sampleRateHz
			ph := 2 * math.Pi * (imp.FreqOffsetHz*t + 0.5*imp.FreqDriftHzPerSec*t*t)
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

	// Front-end / ADC gain: a constant amplitude scale applied last, so
	// signal and noise scale together and the SNR is preserved.
	if imp.Scale != 0 && imp.Scale != 1 {
		s := float32(imp.Scale)
		for i := range out {
			out[i] *= complex(s, 0)
		}
	}

	return out
}

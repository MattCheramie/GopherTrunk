package demod

import "math"

// P25 C4FM pulse-shaping filters, per TIA-102.BAAA (cross-checked against
// the op25 reference implementation, c4fm_const.py).
//
// P25 C4FM is NOT a root-raised-cosine matched-pair system. The transmit
// baseband filter is a raised-cosine (α=0.2) cascaded with an inverse-sinc
// compensation; the receive filter is a sinc (a one-symbol-period
// integrate-and-dump). The transmit inverse-sinc and the receive sinc
// cancel, so the cascade transmit×receive is a plain raised-cosine —
// ISI-free at the symbol instants.
//
// GopherTrunk previously modelled C4FM as an RRC matched pair on both
// ends; that is self-consistent in a synthetic harness but does not match
// a real P25 transmitter, leaving residual inter-symbol interference on
// real captures (issue #275).

const p25C4FMSymbolRate = 4800.0

// p25C4FMSpan is the span, in symbols, of the generated P25 C4FM filters
// (ntaps ≈ sps·span). 13 matches the op25 reference and is long enough
// for the raised-cosine + inverse-sinc impulse response to settle.
const p25C4FMSpan = 13

// transferFunctionTxP25 returns the P25 C4FM transmit filter's frequency
// response, sampled at 1 Hz spacing. Index i is the response at i Hz;
// the response is zero above 2880 Hz.
//
//	H(f): raised-cosine α=0.2 — flat to 1920 Hz, cosine roll-off to 2880 Hz
//	P(f): inverse-sinc compensation (πf/Rs)/sin(πf/Rs)
func transferFunctionTxP25() []float64 {
	xfer := make([]float64, 2881)
	for f := 0; f <= 2880; f++ {
		var hf float64
		if f < 1920 {
			hf = 1.0
		} else {
			hf = 0.5 + 0.5*math.Cos(2*math.Pi*float64(f)/1920.0)
		}
		t := math.Pi * float64(f) / p25C4FMSymbolRate
		pf := 1.0
		if t >= 1e-6 {
			pf = t / math.Sin(t)
		}
		xfer[f] = pf * hf
	}
	return xfer
}

// transferFunctionRxP25 returns the P25 C4FM receive filter's frequency
// response: D(f) = sinc(f/Rs), a one-symbol integrate-and-dump that
// undoes the transmit inverse-sinc compensation.
func transferFunctionRxP25() []float64 {
	rate := int(p25C4FMSymbolRate)
	xfer := make([]float64, rate)
	for f := 0; f < rate; f++ {
		t := math.Pi * float64(f) / p25C4FMSymbolRate
		df := 1.0
		if t >= 1e-6 {
			df = math.Sin(t) / t
		}
		xfer[f] = df
	}
	return xfer
}

// p25C4FMFilterTaps converts a 1 Hz-spaced frequency response into a
// real, linear-phase FIR of ntaps taps at sampleRate, normalised so the
// taps sum to dcGain. Mirrors op25's c4fm_taps.generate(): inverse-FFT
// the transfer function, centre it on its peak, take the central ntaps.
// The transfer function is real and even, so the inverse FFT reduces to
// a cosine sum and only the central taps need to be evaluated.
func p25C4FMFilterTaps(xfer []float64, sampleRate, dcGain float64, ntaps int) []float32 {
	n := math.Round(sampleRate)
	half := (ntaps - 1) / 2
	taps := make([]float32, ntaps)
	var sum float64
	for i := 0; i < ntaps; i++ {
		m := float64(i - half) // tap offset from the centre (peak at m=0)
		// h[m] = (1/N)·( X[0] + 2·Σ_{k≥1} X[k]·cos(2πkm/N) )
		acc := xfer[0]
		for k := 1; k < len(xfer); k++ {
			acc += 2 * xfer[k] * math.Cos(2*math.Pi*float64(k)*m/n)
		}
		h := acc / n
		taps[i] = float32(h)
		sum += h
	}
	if sum != 0 {
		g := float32(dcGain / sum)
		for i := range taps {
			taps[i] *= g
		}
	}
	return taps
}

// P25C4FMTxTaps builds the P25 C4FM transmit pulse-shaping filter for
// the given sample rate, normalised to unit DC gain.
func P25C4FMTxTaps(sampleRate float64) []float32 {
	sps := int(math.Round(sampleRate / p25C4FMSymbolRate))
	return p25C4FMFilterTaps(transferFunctionTxP25(), sampleRate, 1.0, (sps*p25C4FMSpan)|1)
}

// P25C4FMRxTaps builds the P25 C4FM receive filter for the given sample
// rate. It is normalised to a DC gain of sps so that, fed an
// FM-discriminator output, the matched-filtered symbol centres land at
// ±(2π·deviation/sampleRate) — the level the C4FM slicer's thresholds
// are calibrated to (a one-symbol impulse carries 1/sps of the DC,
// which the sps DC gain restores).
func P25C4FMRxTaps(sampleRate float64) []float32 {
	sps := int(math.Round(sampleRate / p25C4FMSymbolRate))
	return p25C4FMFilterTaps(transferFunctionRxP25(), sampleRate, float64(sps), (sps*p25C4FMSpan)|1)
}

// NewC4FMP25 builds a C4FM demodulator with the spec P25 receive filter
// (P25C4FMRxTaps) — the correct matched filter for a real P25 Phase 1
// C4FM transmitter, which NewC4FM's root-raised-cosine is not.
func NewC4FMP25(sampleRate, deviation float64) *C4FM {
	return NewC4FMWithTaps(P25C4FMRxTaps(sampleRate), deviation)
}

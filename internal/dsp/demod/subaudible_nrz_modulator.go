package demod

import "math"

// SubAudibleNRZModulator synthesises a sub-audible NRZ bit stream
// FM-modulated onto an IQ stream. Pairs with the LTR receiver's
// FM-demod → narrow-LPF → MM clock recovery → zero-threshold
// slicer chain so integration tests and offline harnesses can
// produce IQ the production LTR receiver locks on.
//
// Signal chain (TX side):
//
//	bit → bipolar symbol (-1 / +1)
//	    → upsample × sps (one symbol per `sampleRate / symbolRate`
//	      audio samples, no pulse shaping)
//	    → FM modulator (each output sample's phase advances by
//	      audioAmp · symbol — receiver's FM discriminator
//	      recovers the same bipolar audio)
//	    → IQ[n] = exp(j · rf_phase[n])
//
// The audio amplitude is set well below the receiver's LPF
// cutoff and well within the FM-discriminator's linear range so
// the LPF doesn't introduce ringing and the slicer (at zero
// threshold) recovers bits cleanly. LTR's sub-audible signaling
// is NRZ at 300 baud below 300 Hz; the modulator generates that
// directly. Manchester-encoded variants are a follow-up — the
// production receiver's Manchester support (PR #142) already
// handles them once the modulator wraps the bits in a Manchester
// pre-encode.
//
// Stateful across Modulate calls: the RF phase carries forward so
// long streams stay phase-continuous. Reset clears it. Single-
// shot callers can use ModulateSubAudibleNRZ.
type SubAudibleNRZModulator struct {
	sampleRate float64
	symbolRate float64
	audioAmp   float64

	spsAudio int

	rfPhase float64
}

// NewSubAudibleNRZModulator constructs a modulator at the given
// audio sample rate and symbol rate. The audio amplitude
// (audioAmp, in rad/sample at the symbol's NRZ level) sets the
// FM modulation depth — pick a value comfortably below π but
// large enough that the receiver's LPF + slicer hit the
// transitions cleanly. 0.05 is well-tuned for LTR's 300-baud
// signaling at 48 kHz IQ with a 300 Hz LPF cutoff.
//
// Panics if any argument is non-positive or if sampleRate /
// symbolRate < 2.
func NewSubAudibleNRZModulator(sampleRate, symbolRate, audioAmp float64) *SubAudibleNRZModulator {
	if sampleRate <= 0 || symbolRate <= 0 || audioAmp <= 0 {
		panic("demod: NewSubAudibleNRZModulator requires positive sampleRate, symbolRate, audioAmp")
	}
	spsAudio := int(sampleRate/symbolRate + 0.5)
	if spsAudio < 2 {
		panic("demod: NewSubAudibleNRZModulator requires sampleRate/symbolRate >= 2")
	}
	return &SubAudibleNRZModulator{
		sampleRate: sampleRate,
		symbolRate: symbolRate,
		audioAmp:   audioAmp,
		spsAudio:   spsAudio,
	}
}

// Reset clears the RF phase accumulator so the next Modulate call
// starts a fresh, phase-zero stream.
func (m *SubAudibleNRZModulator) Reset() {
	m.rfPhase = 0
}

// Modulate converts a bit sequence (each entry 0 or 1) to
// len(bits) × spsAudio IQ samples. Subsequent calls continue the
// RF-phase accumulator so long streams can be chunked.
func (m *SubAudibleNRZModulator) Modulate(bits []byte) []complex64 {
	out := make([]complex64, len(bits)*m.spsAudio)
	for bi, b := range bits {
		// Bipolar mapping: bit 1 → +audioAmp, bit 0 → -audioAmp.
		// The receiver's slicer thresholds at zero so positive
		// FM-demod output → bit 1.
		sym := m.audioAmp
		if b&1 == 0 {
			sym = -m.audioAmp
		}
		for k := 0; k < m.spsAudio; k++ {
			// FM-modulate: each sample's phase advances by the
			// bipolar audio level. Receiver's FM discriminator
			// recovers arg(z[n] · conj(z[n-1])) = sym.
			m.rfPhase += sym
			if m.rfPhase >= 2*math.Pi || m.rfPhase < -2*math.Pi {
				m.rfPhase = math.Mod(m.rfPhase, 2*math.Pi)
			}
			out[bi*m.spsAudio+k] = complex(
				float32(math.Cos(m.rfPhase)),
				float32(math.Sin(m.rfPhase)),
			)
		}
	}
	return out
}

// ModulateSubAudibleNRZ is the convenience wrapper for
// single-shot callers: constructs a fresh modulator, runs
// Modulate once, and returns the IQ buffer.
func ModulateSubAudibleNRZ(bits []byte, sampleRate, symbolRate, audioAmp float64) []complex64 {
	return NewSubAudibleNRZModulator(sampleRate, symbolRate, audioAmp).Modulate(bits)
}

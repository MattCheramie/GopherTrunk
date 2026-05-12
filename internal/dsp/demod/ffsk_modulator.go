package demod

import "math"

// FFSKModulator synthesises an audio-band Fast-FSK signal
// FM-modulated onto an IQ stream. Pairs with the existing FFSK
// demodulator in this package so integration tests and offline
// harnesses can produce IQ the production MPT 1327 (and other
// audio-FSK) receivers actually lock on.
//
// Signal chain (TX side):
//
//	bit → tone select (mark / space)
//	    → continuous-phase audio sinusoid at the tone frequency
//	    → audio amplitude × cos(audio_phase)
//	    → FM modulator (phase accumulator integrates audio)
//	    → IQ[n] = exp(j · rf_phase[n])
//
// The receiver pipeline runs the inverse: FM discriminator
// produces the per-sample phase difference, which equals the
// integrated audio (a sinusoid at the mark or space frequency);
// the FFSK tone discriminator complex-mixes by the midpoint,
// low-pass filters, and FM-discriminates the complex baseband to
// recover the slicer's soft bit.
//
// Stateful across Modulate calls: audio phase + RF phase carry
// forward so long streams can be chunked. Reset clears both.
// Single-shot callers can use the ModulateFFSK convenience
// function below.
//
// MPT 1327 parameters (CCIR FFSK): sampleRate ≥ 9600 Hz,
// symbolRate = 1200 baud, markHz = 1200, spaceHz = 1800. The audio
// amplitude scaling (audioAmp = 0.5) keeps the FM modulation depth
// well under π rad/sample at typical sample rates so the modulated
// IQ stays in the receiver's expected linear range.
type FFSKModulator struct {
	sampleRate float64
	symbolRate float64
	markHz     float64
	spaceHz    float64
	audioAmp   float64

	spsAudio int

	audioPhase float64 // accumulates across bits for phase-continuous audio
	rfPhase    float64 // accumulates across bits for phase-continuous FM
}

// NewFFSKModulator constructs a modulator at the given audio
// sample rate, symbol rate (bits/sec), and mark / space tone
// frequencies. Panics if any argument is non-positive, if
// markHz == spaceHz, or if sampleRate < 2 × max(markHz, spaceHz).
func NewFFSKModulator(sampleRate, symbolRate, markHz, spaceHz float64) *FFSKModulator {
	if sampleRate <= 0 || symbolRate <= 0 || markHz <= 0 || spaceHz <= 0 {
		panic("demod: NewFFSKModulator requires positive sampleRate, symbolRate, markHz, spaceHz")
	}
	if markHz == spaceHz {
		panic("demod: NewFFSKModulator requires markHz != spaceHz")
	}
	maxTone := markHz
	if spaceHz > maxTone {
		maxTone = spaceHz
	}
	if sampleRate < 2*maxTone {
		panic("demod: NewFFSKModulator requires sampleRate >= 2 * max(markHz, spaceHz)")
	}
	spsAudio := int(sampleRate/symbolRate + 0.5)
	if spsAudio < 2 {
		panic("demod: NewFFSKModulator requires sampleRate/symbolRate >= 2")
	}
	return &FFSKModulator{
		sampleRate: sampleRate,
		symbolRate: symbolRate,
		markHz:     markHz,
		spaceHz:    spaceHz,
		audioAmp:   0.5,
		spsAudio:   spsAudio,
	}
}

// Reset clears the audio + RF phase accumulators so the next
// Modulate call starts a fresh, phase-zero stream.
func (m *FFSKModulator) Reset() {
	m.audioPhase = 0
	m.rfPhase = 0
}

// Modulate converts a bit sequence (each entry 0 or 1; CCIR FFSK
// convention is mark = binary 1) to len(bits) × spsAudio IQ
// samples. Subsequent calls continue the phase accumulators from
// where the previous call left off, so long streams can be chunked.
func (m *FFSKModulator) Modulate(bits []byte) []complex64 {
	out := make([]complex64, len(bits)*m.spsAudio)
	for bi, b := range bits {
		tone := m.markHz
		if b&1 == 0 {
			tone = m.spaceHz
		}
		audioStep := 2 * math.Pi * tone / m.sampleRate
		for k := 0; k < m.spsAudio; k++ {
			audio := m.audioAmp * math.Cos(m.audioPhase)
			m.audioPhase += audioStep
			// Wrap audio phase periodically to keep float64
			// precision well-conditioned.
			if m.audioPhase >= 2*math.Pi {
				m.audioPhase -= 2 * math.Pi
			}
			// FM-modulate: integrate audio into RF phase. The
			// receiver's FM discriminator recovers the audio as
			// arg(z[n] · conj(z[n-1])) = audio[n]; the FFSK
			// helper then tone-discriminates that audio.
			m.rfPhase += audio
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

// ModulateFFSK is the convenience wrapper for single-shot callers:
// constructs a fresh modulator, runs Modulate once, and returns the
// IQ buffer.
func ModulateFFSK(bits []byte, sampleRate, symbolRate, markHz, spaceHz float64) []complex64 {
	return NewFFSKModulator(sampleRate, symbolRate, markHz, spaceHz).Modulate(bits)
}

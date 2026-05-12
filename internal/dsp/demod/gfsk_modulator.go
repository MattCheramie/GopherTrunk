package demod

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// GFSKModulator synthesises a continuous-phase Gaussian-FSK IQ
// stream from a bit sequence. Pairs with the existing GFSK
// demodulator in this package so integration tests and offline
// harnesses can produce IQ the production EDACS / Motorola Type II
// receivers actually lock on.
//
// Signal chain (TX side):
//
//	bit → bipolar symbol (-1 / +1)
//	    → upsample × sps (impulse train)
//	    → Gaussian premod filter (matches the receiver's Gaussian
//	      matched filter; unit-sum normalised so a sustained NRZ
//	      level passes through at the symbol centre unchanged)
//	    → frequency-modulate via phase accumulation
//	      dφ/dn = 2π · shaped(n) · deviation / Fs
//	    → IQ[n] = exp(j · φ[n])
//
// The receiver pipeline runs the inverse: FM discriminator
// produces the per-sample phase difference, which equals the
// shaped symbol train (scaled by ±deviation); the GFSK matched
// filter convolves with the same Gaussian giving a wider
// raised-Gaussian composite; the slicer thresholds at zero and
// maps each sample to {0, 1}.
//
// One Modulator instance is stateful between Modulate calls so a
// long stream can be assembled chunk-by-chunk. Reset clears the
// filter history and the integrator. Single-shot callers can use
// the ModulateGFSK convenience function below.
type GFSKModulator struct {
	sps        int
	gauss      []float32
	deviation  float64
	sampleRate float64

	shapeHist []float32
	histPos   int
	phase     float64
}

// NewGFSKModulator constructs a modulator. sps is samples per
// symbol (sampleRate / symbol_rate); span is the Gaussian
// half-span in symbols (typical 4 for EDACS / GE-Marc); bt is
// the Gaussian premod bandwidth-time product (0.3 for EDACS /
// GE-Marc, 0.5 for Bluetooth-class); deviation is the peak
// frequency deviation in Hz at the steady-state symbol value
// (±2.4 kHz nominal on EDACS).
//
// Panics if sps or span are non-positive — deterministic
// programmer errors, not runtime configuration.
func NewGFSKModulator(sps, span int, bt, sampleRate, deviation float64) *GFSKModulator {
	if sps <= 0 {
		panic("demod: GFSKModulator sps must be positive")
	}
	if span <= 0 {
		panic("demod: GFSKModulator span must be positive")
	}
	g := filter.Gaussian(sps, span, bt)
	return &GFSKModulator{
		sps:        sps,
		gauss:      g,
		deviation:  deviation,
		sampleRate: sampleRate,
		shapeHist:  make([]float32, len(g)),
	}
}

// Reset clears the FIR history and the phase accumulator so the
// next Modulate call starts a fresh, phase-zero stream.
func (m *GFSKModulator) Reset() {
	for i := range m.shapeHist {
		m.shapeHist[i] = 0
	}
	m.histPos = 0
	m.phase = 0
}

// Modulate converts a bit sequence (each entry 0 or 1) to
// len(bits) * sps IQ samples. Subsequent calls continue the phase
// accumulator from where the previous call left off, so long
// streams can be chunked.
func (m *GFSKModulator) Modulate(bits []byte) []complex64 {
	out := make([]complex64, len(bits)*m.sps)
	N := len(m.gauss)
	for bi, b := range bits {
		// Bipolar mapping: 0 → -1, 1 → +1.
		sym := float32(2*int(b&1) - 1)
		for k := 0; k < m.sps; k++ {
			// Impulse-train sample: symbol value at slot 0, zero
			// elsewhere within the symbol period.
			var x float32
			if k == 0 {
				x = sym
			}
			m.shapeHist[m.histPos] = x
			m.histPos = (m.histPos + 1) % N

			// FIR convolve: y[n] = Σ gauss[k] · hist[n-k].
			var shaped float32
			idx := m.histPos - 1
			if idx < 0 {
				idx = N - 1
			}
			for k := 0; k < N; k++ {
				shaped += m.gauss[k] * m.shapeHist[idx]
				idx--
				if idx < 0 {
					idx = N - 1
				}
			}

			// FM integrator. The Gaussian is unit-sum normalised so
			// a sustained ±1 NRZ level passes through unchanged at
			// the symbol centre — no need to divide by an alphabet
			// scale here (C4FM divides by 3 because its alphabet
			// peaks at ±3; GFSK's peaks at ±1).
			m.phase += 2 * math.Pi * float64(shaped) * m.deviation / m.sampleRate
			out[bi*m.sps+k] = complex(
				float32(math.Cos(m.phase)),
				float32(math.Sin(m.phase)),
			)
		}
	}
	return out
}

// ModulateGFSK is the convenience wrapper for single-shot
// callers: constructs a fresh GFSKModulator, runs Modulate once,
// and returns the IQ buffer. Useful in tests + offline harnesses
// that don't need cross-call phase continuity.
func ModulateGFSK(bits []byte, sps, span int, bt, sampleRate, deviation float64) []complex64 {
	return NewGFSKModulator(sps, span, bt, sampleRate, deviation).Modulate(bits)
}

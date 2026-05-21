package demod

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// C4FMModulator synthesises a P25-Phase-1-style C4FM IQ stream
// from a dibit sequence. Pairs with the existing C4FM demodulator
// in this package (and the wider P25 Phase 1 receiver chain) so
// integration tests and offline harnesses can produce IQ the
// production decoder actually locks on.
//
// Signal chain (TX side):
//
//	dibit → symbol value (-3 / -1 / +1 / +3)
//	     → upsample × sps (impulse train)
//	     → RRC pulse-shape filter (matches the receiver's RRC
//	       matched filter; unit-energy normalised)
//	     → frequency-modulate via phase accumulation
//	       dφ/dn = 2π · shaped_symbol(n) · deviation / 3 / Fs
//	     → IQ[n] = exp(j · φ[n])
//
// The receiver pipeline runs the inverse: FM discriminator
// produces the per-sample phase difference, which equals the
// shaped symbol train (scaled by deviation); the RRC matched
// filter convolves with the same RRC giving a raised-cosine
// composite that's ISI-free at symbol centres; the slicer maps
// the symbol-centre samples to ±1 / ±3 and SymbolToDibit packs
// them back into dibits.
//
// One Modulator instance is stateful between Modulate calls so a
// long stream can be assembled chunk-by-chunk. Calls to Reset
// clear the filter history and the integrator. Single-shot
// callers can use the convenience function ModulateC4FM below.
type C4FMModulator struct {
	sps        int
	shapeTaps  []float32
	deviation  float64
	sampleRate float64

	// shapeHist holds the last len(rrc)-1 input samples
	// (impulses + zero-fill) the FIR convolution needs to span
	// the pulse across symbol boundaries.
	shapeHist []float32
	histPos   int

	// phase is the current FM accumulator value in radians;
	// kept across Modulate calls so chunked output is
	// phase-continuous.
	phase float64
}

// NewC4FMModulator constructs a modulator. sps is samples per
// symbol (sampleRate / symbol_rate); span is the RRC half-span in
// symbols (matches the receiver's PulseSpanSymbols, typically 8);
// alpha is the RRC roll-off (typically 0.2 for P25 Phase 1);
// deviation is the peak frequency deviation in Hz (1800 for
// P25 Phase 1).
//
// Panics if sps or span are non-positive — these are deterministic
// programmer errors, not runtime configuration.
func NewC4FMModulator(sps, span int, alpha, sampleRate, deviation float64) *C4FMModulator {
	if span <= 0 {
		panic("demod: C4FMModulator span must be positive")
	}
	return NewC4FMModulatorWithTaps(sps, filter.RootRaisedCosine(sps, span, alpha), sampleRate, deviation)
}

// NewC4FMModulatorWithTaps constructs a modulator whose pulse-shaping
// filter is the supplied FIR taps, instead of the default
// root-raised-cosine (NewC4FMModulator). P25 Phase 1 uses this with the
// spec C4FM transmit filter (P25C4FMTxTaps) so a synthesized stream
// matches a real P25 transmitter — see ModulateP25C4FM and c4fm_p25.go.
func NewC4FMModulatorWithTaps(sps int, shapeTaps []float32, sampleRate, deviation float64) *C4FMModulator {
	if sps <= 0 {
		panic("demod: C4FMModulator sps must be positive")
	}
	if len(shapeTaps) == 0 {
		panic("demod: C4FMModulator shapeTaps must be non-empty")
	}
	return &C4FMModulator{
		sps:        sps,
		shapeTaps:  shapeTaps,
		deviation:  deviation,
		sampleRate: sampleRate,
		shapeHist:  make([]float32, len(shapeTaps)),
	}
}

// Reset clears the FIR history and the phase accumulator so the
// next Modulate call starts a fresh, phase-zero stream.
func (m *C4FMModulator) Reset() {
	for i := range m.shapeHist {
		m.shapeHist[i] = 0
	}
	m.histPos = 0
	m.phase = 0
}

// Modulate converts a dibit sequence to len(dibits)*sps IQ
// samples. Subsequent calls continue the phase accumulator from
// where the previous call left off, so long streams can be
// chunked.
func (m *C4FMModulator) Modulate(dibits []uint8) []complex64 {
	out := make([]complex64, len(dibits)*m.sps)
	N := len(m.shapeTaps)
	for di, d := range dibits {
		sym := dibitToC4FMSymbol(d)
		for k := 0; k < m.sps; k++ {
			// Impulse-train sample: symbol value at slot 0,
			// zero elsewhere within the symbol period.
			var x float32
			if k == 0 {
				x = float32(sym)
			}
			m.shapeHist[m.histPos] = x
			m.histPos = (m.histPos + 1) % N

			// FIR convolve: y[n] = Σ rrc[k] · hist[n-k].
			var shaped float32
			idx := m.histPos - 1
			if idx < 0 {
				idx = N - 1
			}
			for k := 0; k < N; k++ {
				shaped += m.shapeTaps[k] * m.shapeHist[idx]
				idx--
				if idx < 0 {
					idx = N - 1
				}
			}

			// FM integrator. The shaped value spans roughly
			// ±3 at peaks, so we divide by 3 to normalise to
			// ±1 before scaling to ±deviation. The factor of 2π
			// converts cycles to radians.
			m.phase += 2 * math.Pi * float64(shaped) * m.deviation / 3.0 / m.sampleRate
			out[di*m.sps+k] = complex(
				float32(math.Cos(m.phase)),
				float32(math.Sin(m.phase)),
			)
		}
	}
	return out
}

// ModulateC4FM is the convenience wrapper for single-shot
// callers: constructs a fresh C4FMModulator, runs Modulate once,
// and returns the IQ buffer. Useful in tests + offline harnesses
// that don't need cross-call phase continuity.
func ModulateC4FM(dibits []uint8, sps, span int, alpha, sampleRate, deviation float64) []complex64 {
	return NewC4FMModulator(sps, span, alpha, sampleRate, deviation).Modulate(dibits)
}

// ModulateP25C4FM synthesises a spec-faithful P25 Phase 1 C4FM stream:
// it shapes with the real P25 transmit filter (raised-cosine α=0.2
// cascaded with the inverse-sinc compensation, P25C4FMTxTaps) rather
// than the root-raised-cosine ModulateC4FM uses. A real P25 transmitter
// uses this shaping, so this is what the receiver's spec matched filter
// (NewC4FMP25) must be tested against. sps is derived as sampleRate/4800.
func ModulateP25C4FM(dibits []uint8, sampleRate, deviation float64) []complex64 {
	sps := int(math.Round(sampleRate / p25C4FMSymbolRate))
	return NewC4FMModulatorWithTaps(sps, P25C4FMTxTaps(sampleRate), sampleRate, deviation).Modulate(dibits)
}

// dibitToC4FMSymbol is the inverse of phase1.SymbolToDibit: maps
// a 0..3 dibit value to the {-3, -1, +1, +3} C4FM symbol value
// per TIA-102.BAAA-A §6.1.1.
//
//	dibit 0 → +1   dibit 1 → +3   dibit 2 → -1   dibit 3 → -3
//
// The mapping lives here rather than in phase1 so the modulator
// stays in the dsp/demod package alongside its inverse and the
// rest of the C4FM tooling.
func dibitToC4FMSymbol(dibit uint8) int {
	switch dibit & 3 {
	case 0:
		return +1
	case 1:
		return +3
	case 2:
		return -1
	case 3:
		return -3
	}
	return 0
}

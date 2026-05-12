package demod

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// PiOver4DQPSKModulator synthesises a π/4-shifted differential
// QPSK IQ stream from a dibit sequence. Pairs with the existing
// PiOver4DQPSK demodulator in this package so integration tests
// and offline harnesses can produce IQ the production TETRA
// (rotation = π/4) and P25 Phase 2 H-DQPSK (rotation = π/8)
// receivers actually lock on.
//
// Signal chain (TX side):
//
//	dibit → raw phase increment ∈ {0, π/2, π, -π/2}
//	      → +rotation per symbol (π/4 for TETRA, π/8 for P25 P2)
//	      → cumulative phase φ[k]
//	      → complex symbol exp(j · φ[k])
//	      → impulse train × sps (symbol at slot 0, zero
//	        elsewhere within each symbol period)
//	      → RRC pulse-shape filter (matches the receiver's RRC
//	        matched filter; unit-energy normalised)
//	      → IQ
//
// The receiver pipeline runs the inverse: RRC matched filter →
// clock recovery → arg(s · conj(s_prev)) - rotation, sliced into
// one of four quadrants to recover the dibit.
//
// Stateful across Modulate calls: the cumulative phase + RRC FIR
// history carry forward so long streams can be chunked. Reset
// clears both. Single-shot callers can use the ModulatePiOver4DQPSK
// convenience function below.
type PiOver4DQPSKModulator struct {
	sps      int
	rrc      []float32
	rotation float64

	// FIR history is complex — symbol pulses live in I/Q,
	// unlike the real-valued C4FM modulator.
	shapeHist []complex64
	histPos   int

	// phase is the cumulative differential phase, carried
	// across Modulate calls.
	phase float64
}

// NewPiOver4DQPSKModulator constructs a modulator. sps is samples
// per symbol; span is the RRC half-span in symbols; alpha is the
// RRC roll-off; rotation is the per-symbol constellation offset
// (math.Pi/4 for TETRA, math.Pi/8 for P25 Phase 2 H-DQPSK).
//
// Panics if any of sps, span, alpha is non-positive — deterministic
// programmer errors, not runtime configuration.
func NewPiOver4DQPSKModulator(sps, span int, alpha, rotation float64) *PiOver4DQPSKModulator {
	if sps <= 0 || span <= 0 || alpha <= 0 {
		panic("demod: PiOver4DQPSKModulator requires positive sps, span, alpha")
	}
	rrc := filter.RootRaisedCosine(sps, span, alpha)
	return &PiOver4DQPSKModulator{
		sps:       sps,
		rrc:       rrc,
		rotation:  rotation,
		shapeHist: make([]complex64, len(rrc)),
	}
}

// Reset clears the FIR history and the differential phase
// accumulator so the next Modulate call starts a fresh,
// phase-zero stream.
func (m *PiOver4DQPSKModulator) Reset() {
	for i := range m.shapeHist {
		m.shapeHist[i] = 0
	}
	m.histPos = 0
	m.phase = 0
}

// Modulate converts a dibit sequence (each entry 0..3) to
// len(dibits) * sps IQ samples. Subsequent calls continue the
// phase accumulator and FIR history from where the previous call
// left off, so long streams can be chunked.
func (m *PiOver4DQPSKModulator) Modulate(dibits []uint8) []complex64 {
	out := make([]complex64, len(dibits)*m.sps)
	N := len(m.rrc)
	for di, d := range dibits {
		// Advance the differential phase by rotation + raw
		// dibit phase. The mapping matches the receiver's
		// quadrant decode (DQPSK.Decode):
		//   dibit 00 → +0
		//   dibit 01 → +π/2
		//   dibit 11 → -π/2
		//   dibit 10 → +π (or -π — symmetric)
		m.phase += m.rotation + dibitToDQPSKPhase(d)
		// Wrap into a tight numeric range to avoid float drift
		// over long streams.
		for m.phase >= 2*math.Pi {
			m.phase -= 2 * math.Pi
		}
		for m.phase < 0 {
			m.phase += 2 * math.Pi
		}
		sym := complex(
			float32(math.Cos(m.phase)),
			float32(math.Sin(m.phase)),
		)
		for k := 0; k < m.sps; k++ {
			// Impulse-train sample: complex symbol at slot 0,
			// zero elsewhere within the symbol period.
			var x complex64
			if k == 0 {
				x = sym
			}
			m.shapeHist[m.histPos] = x
			m.histPos = (m.histPos + 1) % N

			// FIR convolve (real-valued taps × complex history):
			//   y[n] = Σ rrc[k] · hist[n-k]
			var accI, accQ float32
			idx := m.histPos - 1
			if idx < 0 {
				idx = N - 1
			}
			for k := 0; k < N; k++ {
				accI += m.rrc[k] * real(m.shapeHist[idx])
				accQ += m.rrc[k] * imag(m.shapeHist[idx])
				idx--
				if idx < 0 {
					idx = N - 1
				}
			}
			out[di*m.sps+k] = complex(accI, accQ)
		}
	}
	return out
}

// ModulatePiOver4DQPSK is the convenience wrapper for single-shot
// callers: constructs a fresh modulator, runs Modulate once, and
// returns the IQ buffer.
func ModulatePiOver4DQPSK(dibits []uint8, sps, span int, alpha, rotation float64) []complex64 {
	return NewPiOver4DQPSKModulator(sps, span, alpha, rotation).Modulate(dibits)
}

// dibitToDQPSKPhase returns the raw differential phase
// increment encoded by a 0..3 dibit value. Inverse of the
// quadrant decode in DQPSK.Decode (after the rotation offset is
// subtracted).
func dibitToDQPSKPhase(dibit uint8) float64 {
	switch dibit & 3 {
	case 0b00:
		return 0
	case 0b01:
		return math.Pi / 2
	case 0b11:
		return -math.Pi / 2
	case 0b10:
		return math.Pi
	}
	return 0
}

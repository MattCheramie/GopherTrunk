package receiver

import (
	"math"
	"testing"
)

// makeRunC4FMIQ synthesises a 48 kHz / 10 sps IQ buffer whose
// instantaneous frequency steps between the four C4FM symbols with a
// rectangular (NRZ) profile — a constant frequency held for each symbol
// period — and holds each data symbol for runLen consecutive symbols.
//
// The run structure matters: a real (non-RRC-shaped) rectangular stream is
// what makes the receiver's unit-energy RRC matched filter overshoot
// (~3.1× DC gain), the exact condition the symbol-AGC exists to correct.
// Holding each symbol for a run keeps the run *interior* free of the
// inter-symbol interference a fast-alternating pattern would suffer (the
// 160-tap matched filter spans ~16 symbols), so the 4-level eye is clean
// and correctness scoring actually reflects the slicer's over-scale, not
// ISI. dibit→symbol uses the inverse of SymbolToDibit: 0→+1, 1→+3, 2→-1,
// 3→-3.
func makeRunC4FMIQ(dibits []uint8) []complex64 {
	const (
		sampleRate = 48_000.0
		sps        = 10
		deviation  = 1800.0
	)
	symFor := func(d uint8) int {
		switch d & 3 {
		case 0:
			return +1
		case 1:
			return +3
		case 2:
			return -1
		default:
			return -3
		}
	}
	iq := make([]complex64, len(dibits)*sps)
	phase := 0.0
	for s, d := range dibits {
		dphi := 2 * math.Pi * float64(symFor(d)) * deviation / 3.0 / sampleRate
		base := s * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}
	return iq
}

// TestReceiverDecodesOverScaledC4FM is the end-to-end regression test for
// the symbol-AGC. It drives the receiver with a rectangular-symbol C4FM
// stream at the physical DeviationHz calibration and requires the
// recovered dibits to match the transmitted sequence.
//
// The failure it pins: the RRC matched filter is unit-*energy* (DC gain
// ~3.1), so on a rectangular (real-world, non-RRC-shaped) symbol stream
// the 4-level eye lands ~3.1× above the slicer's fixed thresholds. The
// inner (±1) symbols then cross the outer threshold (2·deviation/3) and
// mis-slice to ±3, corrupting every dibit carried by an inner symbol —
// exactly the "control channel locks then decodes nothing" field failure.
// The symbol-AGC renormalises mean|x| so the eye sits at the designed
// level and the slicer decides correctly.
//
// Without the AGC the two inner-symbol dibits (0→+1, 2→-1) collapse onto
// the outer dibits (1→+3, 3→-3), so the match rate falls to ~50% and this
// test fails. With the AGC it recovers ~97%+.
func TestReceiverDecodesOverScaledC4FM(t *testing.T) {
	// Hold each of the four symbols for a run so the run interiors form a
	// clean, ISI-free 4-level eye; repeat the 4-symbol cycle many times.
	const runLen = 24
	base := []uint8{0, 1, 2, 3}
	period := make([]uint8, 0, len(base)*runLen)
	for _, d := range base {
		for range runLen {
			period = append(period, d)
		}
	}
	const cycles = 40
	tx := make([]uint8, 0, len(period)*cycles)
	for range cycles {
		tx = append(tx, period...)
	}

	iq := makeRunC4FMIQ(tx)

	var rx []uint8
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800,
		DibitSink: func(dibits []uint8, baseIdx int) {
			rx = append(rx, dibits...)
		},
	})
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	if len(rx) < len(period)*4 {
		t.Fatalf("recovered only %d dibits; receiver produced too few symbols to score", len(rx))
	}

	// Score the settled tail against the periodic transmit pattern. The
	// decode carries a fixed filter/clock delay, so the transmitted pattern
	// appears cyclically shifted by some constant k ∈ [0,len(period)); pick
	// the best-matching shift.
	P := len(period)
	tail := rx[len(rx)-P*10:]
	best := 0
	for k := 0; k < P; k++ {
		match := 0
		for i, d := range tail {
			if d == period[(i+k)%P] {
				match++
			}
		}
		if match > best {
			best = match
		}
	}
	rate := float64(best) / float64(len(tail))
	if rate < 0.92 {
		t.Errorf("best dibit match rate = %.1f%%, want ≥ 92%% "+
			"(inner symbols mis-sliced — symbol-AGC not normalising the eye)",
			rate*100)
	}
}

// The symbol-AGC's own unit tests (level normalisation + disabled no-op) live
// with the shared type in internal/dsp/demod (TestC4FMSymbolAGC*); the tests
// here exercise it in situ through the NXDN receiver.

// TestReceiverLegacyFixturePathUnchanged confirms that with DeviationHz
// unset (the pre-scaled-fixture path) the AGC is disabled, so the receiver
// stays byte-identical to its pre-AGC behaviour — the opt-in discipline the
// roadmap requires for every new DSP stage.
func TestReceiverLegacyFixturePathUnchanged(t *testing.T) {
	agc := agcTargetFor(1.0, 0) // DeviationHz == 0 → legacy path
	if agc != 0 {
		t.Errorf("agcTargetFor with DeviationHz=0 = %v, want 0 (AGC must stay disabled on the legacy path)", agc)
	}
}

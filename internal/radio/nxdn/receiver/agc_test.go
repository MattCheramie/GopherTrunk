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

// TestC4FMSymbolAGCNormalisesLevel pins the symbol-AGC that lets the
// receiver decode real captures: the RRC matched filter is unit-energy
// (DC gain ~3.1), so on a real FM-discriminator stream the 4-level
// symbol centres land well above the slicer's fixed thresholds and the
// inner symbols collapse onto the outer rails. The AGC must pull the
// running mean|x| back to its target (slicerScale·2/3) regardless of the
// absolute input level, while preserving the relative 4-level structure
// the slicer decides on.
func TestC4FMSymbolAGCNormalisesLevel(t *testing.T) {
	const target = 0.17
	a := c4fmSymbolAGC{target: target, rate: 1.0 / 256.0}

	// A balanced 4-level stream at ~3.1× the level the slicer expects —
	// the over-scale a real matched-filter output carries.
	const scale = 3.1
	levels := []float32{scale * target, scale * target / 3, -scale * target / 3, -scale * target}
	buf := make([]float32, 4096)
	for i := range buf {
		buf[i] = levels[i%4]
	}
	a.process(buf)

	// After the EMA settles, mean|x| over the tail should track target.
	var sum float64
	tail := buf[len(buf)-1024:]
	for _, x := range tail {
		sum += math.Abs(float64(x))
	}
	got := sum / float64(len(tail))
	if math.Abs(got-target) > 0.02*target {
		t.Errorf("settled mean|x| = %.4f, want ≈ %.4f (AGC not normalising)", got, target)
	}

	// Relative structure preserved: outer/inner magnitude ratio stays 3:1.
	outer := math.Abs(float64(tail[0]))
	inner := math.Abs(float64(tail[1]))
	if r := outer / inner; math.Abs(r-3.0) > 0.05 {
		t.Errorf("outer/inner ratio = %.3f, want ≈ 3.0 (AGC distorted the eye)", r)
	}
}

// TestC4FMSymbolAGCDisabled confirms the legacy pre-scaled-fixture path
// (target<=0) is a no-op so existing fixtures stay byte-identical.
func TestC4FMSymbolAGCDisabled(t *testing.T) {
	a := c4fmSymbolAGC{target: 0}
	buf := []float32{0.5, -0.3, 1.2, -0.9}
	want := append([]float32(nil), buf...)
	a.process(buf)
	for i := range buf {
		if buf[i] != want[i] {
			t.Errorf("buf[%d] = %v, want %v (disabled AGC must be a no-op)", i, buf[i], want[i])
		}
	}
}

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

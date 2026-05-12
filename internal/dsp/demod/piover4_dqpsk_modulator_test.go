package demod

import (
	"math"
	"testing"
)

// TestPiOver4DQPSKModulatorRoundTripThroughDemod: a random dibit
// stream is modulated to IQ, then run back through the RRC
// matched filter + π/4-DQPSK quadrant decoder, and recovered
// dibits are checked against the source. The TX + RX RRC cascade
// is ISI-free at symbol centres, so each symbol-period sample
// recovers the source dibit exactly.
func TestPiOver4DQPSKModulatorRoundTripThroughDemod(t *testing.T) {
	const (
		sps      = 8
		span     = 8
		alpha    = 0.35       // TETRA RRC roll-off
		rotation = math.Pi / 4
	)

	src := make([]uint8, 200)
	for i := range src {
		src[i] = uint8((i*7 + 3) & 3)
	}

	iq := ModulatePiOver4DQPSK(src, sps, span, alpha, rotation)
	if len(iq) != len(src)*sps {
		t.Fatalf("IQ length = %d, want %d", len(iq), len(src)*sps)
	}

	// Receiver: RRC matched filter then DQPSK decode at symbol
	// centres. The TX + RX RRC cascade peak lives at the centre
	// of the (2·span+1)·sps support, so for source symbol i the
	// matched-filter peak sample appears at:
	//
	//   centre = i*sps + 2 * sps * span
	rx := NewPiOver4DQPSK(sps, span, alpha, rotation)
	matched := rx.MatchedFilter(nil, iq)

	// Decimate to one sample per symbol at the symbol-centre
	// offset, then run DQPSK.Decode over the resulting symbol
	// stream.
	offset := 2 * sps * span
	var centres []complex64
	for i := range src {
		idx := i*sps + offset
		if idx >= len(matched) {
			break
		}
		centres = append(centres, matched[idx])
	}

	dq := NewDQPSK()
	dq.SetRotation(rotation)
	got := dq.Decode(nil, centres)

	var mismatches int
	for i, want := range src {
		if i >= len(got) {
			break
		}
		if got[i] != want {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("symbol %d: decoded=%d, want=%d (at centre=%v)",
					i, got[i], want, centres[i])
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d/%d dibits failed round-trip", mismatches, len(src))
	}
}

// TestPiOver4DQPSKModulatorEmitsPhaseContinuousStream: chunked
// Modulate calls must produce the same IQ as a single big call.
func TestPiOver4DQPSKModulatorEmitsPhaseContinuousStream(t *testing.T) {
	const (
		sps      = 8
		span     = 8
		alpha    = 0.35
		rotation = math.Pi / 4
	)

	src := make([]uint8, 120)
	for i := range src {
		src[i] = uint8((i*11 + 5) & 3)
	}

	whole := ModulatePiOver4DQPSK(src, sps, span, alpha, rotation)
	mod := NewPiOver4DQPSKModulator(sps, span, alpha, rotation)
	a := mod.Modulate(src[:60])
	b := mod.Modulate(src[60:])
	stitched := append(a, b...)

	if len(whole) != len(stitched) {
		t.Fatalf("length mismatch: whole=%d, stitched=%d", len(whole), len(stitched))
	}
	for i := range whole {
		dr := real(whole[i]) - real(stitched[i])
		di := imag(whole[i]) - imag(stitched[i])
		if math.Abs(float64(dr)) > 1e-5 || math.Abs(float64(di)) > 1e-5 {
			t.Errorf("sample %d diverges: whole=%v, stitched=%v", i, whole[i], stitched[i])
			break
		}
	}
}

// TestPiOver4DQPSKModulatorResetClearsState: after Reset the
// modulator must behave as if newly constructed.
func TestPiOver4DQPSKModulatorResetClearsState(t *testing.T) {
	src := []uint8{0, 1, 2, 3, 0, 1, 2, 3}
	first := ModulatePiOver4DQPSK(src, 8, 8, 0.35, math.Pi/4)

	mod := NewPiOver4DQPSKModulator(8, 8, 0.35, math.Pi/4)
	_ = mod.Modulate([]uint8{3, 2, 1, 0, 3, 2, 1, 0}) // dirty state
	mod.Reset()
	second := mod.Modulate(src)

	for i := range first {
		dr := real(first[i]) - real(second[i])
		di := imag(first[i]) - imag(second[i])
		if math.Abs(float64(dr)) > 1e-5 || math.Abs(float64(di)) > 1e-5 {
			t.Errorf("post-Reset divergence at sample %d", i)
			break
		}
	}
}

// TestDibitToDQPSKPhase pins the encoder mapping as inverse of
// the receiver's quadrant decode. If the receiver's mapping
// changes for spec compliance, this test surfaces the desync.
func TestDibitToDQPSKPhase(t *testing.T) {
	cases := []struct {
		dibit uint8
		want  float64
	}{
		{0b00, 0},
		{0b01, math.Pi / 2},
		{0b11, -math.Pi / 2},
		{0b10, math.Pi},
	}
	for _, tc := range cases {
		if got := dibitToDQPSKPhase(tc.dibit); got != tc.want {
			t.Errorf("dibitToDQPSKPhase(%02b) = %g, want %g", tc.dibit, got, tc.want)
		}
	}
}

package demod

import (
	"math"
	"testing"
)

// TestGFSKModulatorRoundTripThroughDemod is the anchor test: a
// deterministic bit stream is GFSK-modulated, then run back
// through the FM discriminator + Gaussian matched filter +
// zero-threshold slicer, and the recovered bits are checked
// against the source. The Gaussian cascade (TX Gauss × RX Gauss)
// is a wider Gaussian; at symbol centres the soft value is well
// away from zero (positive for source = 1, negative for source =
// 0) so the slicer recovers bits exactly.
func TestGFSKModulatorRoundTripThroughDemod(t *testing.T) {
	const (
		sampleRate = 96_000.0
		sps        = 10
		span       = 4
		bt         = 0.3
		deviation  = 2400.0
	)

	src := make([]byte, 200)
	for i := range src {
		src[i] = byte((i*7 + 3) & 1)
	}

	iq := ModulateGFSK(src, sps, span, bt, sampleRate, deviation)
	if len(iq) != len(src)*sps {
		t.Fatalf("IQ length = %d, want %d", len(iq), len(src)*sps)
	}

	fm := NewFM()
	disc := fm.Process(nil, iq)

	g := NewGFSK(sps, span, bt)
	matched := g.MatchedFilter(nil, disc)

	// Centre offset: the TX + RX Gaussian cascade peaks at
	// time 0 (centre of the combined kernel), so for source bit
	// i the matched-filter peak appears at index:
	//
	//   centre = i*sps + sps*span (TX delay)
	//                  + sps*span (RX delay)
	//                  + 1        (FM discriminator z[n]·conj(z[n-1]))
	offset := 2*sps*span + 1
	var mismatches int
	for i, b := range src {
		centre := i*sps + offset
		if centre >= len(matched) {
			break
		}
		got := g.Slice(matched[centre])
		want := int(b & 1)
		if got != want {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("bit %d: sliced=%d, want=%d, soft=%g",
					i, got, want, matched[centre])
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d/%d bits failed slicer round-trip", mismatches, len(src))
	}
}

// TestGFSKModulatorEmitsPhaseContinuousStream: stitching two
// Modulate calls back-to-back must produce the same IQ as one
// big call. Cross-call phase + FIR-history continuity is
// required for chunked streaming.
func TestGFSKModulatorEmitsPhaseContinuousStream(t *testing.T) {
	const (
		sampleRate = 96_000.0
		sps        = 10
		span       = 4
		bt         = 0.3
		deviation  = 2400.0
	)

	src := make([]byte, 120)
	for i := range src {
		src[i] = byte((i*11 + 5) & 1)
	}

	whole := ModulateGFSK(src, sps, span, bt, sampleRate, deviation)
	mod := NewGFSKModulator(sps, span, bt, sampleRate, deviation)
	a := mod.Modulate(src[:60])
	b := mod.Modulate(src[60:])
	stitched := append(a, b...)

	if len(whole) != len(stitched) {
		t.Fatalf("length mismatch: whole=%d, stitched=%d", len(whole), len(stitched))
	}
	for i := range whole {
		dr := real(whole[i]) - real(stitched[i])
		di := imag(whole[i]) - imag(stitched[i])
		if math.Abs(float64(dr)) > 1e-6 || math.Abs(float64(di)) > 1e-6 {
			t.Errorf("sample %d diverges: whole=%v, stitched=%v", i, whole[i], stitched[i])
			break
		}
	}
}

// TestGFSKModulatorIQMagnitudeStaysUnity: GFSK is constant-
// envelope (CPM). Drift would indicate a bug in the phase
// accumulator or the cos/sin pairing.
func TestGFSKModulatorIQMagnitudeStaysUnity(t *testing.T) {
	src := []byte{0, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0}
	iq := ModulateGFSK(src, 10, 4, 0.3, 96_000.0, 2400.0)
	for i, s := range iq {
		mag := math.Hypot(float64(real(s)), float64(imag(s)))
		if math.Abs(mag-1.0) > 1e-6 {
			t.Errorf("sample %d: |IQ| = %g, want 1.0", i, mag)
			break
		}
	}
}

// TestGFSKModulatorResetClearsState: after Reset the modulator
// must behave as if newly constructed.
func TestGFSKModulatorResetClearsState(t *testing.T) {
	src := []byte{0, 1, 1, 0, 1, 0, 0, 1}
	first := ModulateGFSK(src, 10, 4, 0.3, 96_000.0, 2400.0)

	mod := NewGFSKModulator(10, 4, 0.3, 96_000.0, 2400.0)
	_ = mod.Modulate([]byte{1, 1, 0, 0, 1, 1, 0, 0}) // dirty state
	mod.Reset()
	second := mod.Modulate(src)

	for i := range first {
		dr := real(first[i]) - real(second[i])
		di := imag(first[i]) - imag(second[i])
		if math.Abs(float64(dr)) > 1e-6 || math.Abs(float64(di)) > 1e-6 {
			t.Errorf("post-Reset divergence at sample %d", i)
			break
		}
	}
}

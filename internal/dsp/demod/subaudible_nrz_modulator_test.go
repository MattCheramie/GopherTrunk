package demod

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// TestSubAudibleNRZModulatorRoundTripThroughDemod: a deterministic
// bit stream is sub-audible-modulated, then run back through the
// FM discriminator + narrow-LPF chain the LTR receiver uses, and
// recovered bits are checked against the source. The LPF
// smooths the NRZ transitions but the symbol-centre samples land
// well away from zero, so the slicer recovers each bit exactly.
func TestSubAudibleNRZModulatorRoundTripThroughDemod(t *testing.T) {
	const (
		sampleRate = 48_000.0
		symbolRate = 300.0
		audioAmp   = 0.05
		lpfCutoff  = 300.0
		lpfLen     = 101
		lpfBeta    = 8.6
	)
	const sps = int(sampleRate / symbolRate) // 160

	src := make([]byte, 100)
	for i := range src {
		src[i] = byte((i*7 + 3) & 1)
	}

	iq := ModulateSubAudibleNRZ(src, sampleRate, symbolRate, audioAmp)
	if len(iq) != len(src)*sps {
		t.Fatalf("IQ length = %d, want %d", len(iq), len(src)*sps)
	}

	fm := NewFM()
	audio := fm.Process(nil, iq)

	lpf := filter.NewRealFIR(filter.LowpassKaiser(lpfLen, lpfCutoff/sampleRate, lpfBeta))
	low := lpf.Process(nil, audio)

	// Sample one symbol per bit at the centre. Combined delay:
	// FM discriminator (1 sample) + LPF group delay ((lpfLen-1)/2).
	delay := 1 + (lpfLen-1)/2
	startBit := (delay + sps/2) / sps
	var mismatches, compared int
	for i := startBit; i < len(src); i++ {
		idx := i*sps + sps/2 + delay
		if idx >= len(low) {
			break
		}
		got := 0
		if low[idx] > 0 {
			got = 1
		}
		want := int(src[i] & 1)
		compared++
		if got != want {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("bit %d: sliced=%d, want=%d, soft=%g",
					i, got, want, low[idx])
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d/%d bits failed round-trip", mismatches, compared)
	}
}

// TestSubAudibleNRZModulatorEmitsPhaseContinuousStream: chunked
// Modulate calls must produce the same IQ as a single big call.
func TestSubAudibleNRZModulatorEmitsPhaseContinuousStream(t *testing.T) {
	const (
		sampleRate = 48_000.0
		symbolRate = 300.0
		audioAmp   = 0.05
	)
	src := make([]byte, 60)
	for i := range src {
		src[i] = byte((i*11 + 5) & 1)
	}

	whole := ModulateSubAudibleNRZ(src, sampleRate, symbolRate, audioAmp)
	mod := NewSubAudibleNRZModulator(sampleRate, symbolRate, audioAmp)
	a := mod.Modulate(src[:30])
	b := mod.Modulate(src[30:])
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

// TestSubAudibleNRZModulatorIQMagnitudeStaysUnity: FM modulation
// is constant-envelope; |IQ| must stay at 1.
func TestSubAudibleNRZModulatorIQMagnitudeStaysUnity(t *testing.T) {
	src := []byte{0, 1, 1, 0, 1, 0, 0, 1}
	iq := ModulateSubAudibleNRZ(src, 48_000.0, 300.0, 0.05)
	for i, s := range iq {
		mag := math.Hypot(float64(real(s)), float64(imag(s)))
		if math.Abs(mag-1.0) > 1e-6 {
			t.Errorf("sample %d: |IQ| = %g, want 1.0", i, mag)
			break
		}
	}
}

// TestSubAudibleNRZModulatorResetClearsState: after Reset the
// modulator must behave as if newly constructed.
func TestSubAudibleNRZModulatorResetClearsState(t *testing.T) {
	src := []byte{0, 1, 1, 0, 1, 0, 0, 1}
	first := ModulateSubAudibleNRZ(src, 48_000.0, 300.0, 0.05)

	mod := NewSubAudibleNRZModulator(48_000.0, 300.0, 0.05)
	_ = mod.Modulate([]byte{1, 1, 0, 0, 1, 1, 0, 0})
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

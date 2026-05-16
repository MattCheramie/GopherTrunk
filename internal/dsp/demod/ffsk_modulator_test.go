package demod

import (
	"math"
	"testing"
)

// TestFFSKModulatorRoundTripThroughDemod: a deterministic bit
// stream is FFSK-modulated, then run back through the FM
// discriminator + FFSK tone-discriminator chain, and the
// recovered bits are checked against the source after the
// LPF + tone-discriminator group delay. CCIR FFSK convention:
// mark (1200 Hz) = binary 1, space (1800 Hz) = binary 0.
func TestFFSKModulatorRoundTripThroughDemod(t *testing.T) {
	const (
		sampleRate = 48_000.0
		symbolRate = 1200.0
		markHz     = 1200.0
		spaceHz    = 1800.0
	)
	const sps = int(sampleRate / symbolRate) // 40 samples per bit

	src := make([]byte, 200)
	for i := range src {
		src[i] = byte((i*7 + 3) & 1)
	}

	iq := ModulateFFSK(src, sampleRate, symbolRate, markHz, spaceHz)
	if len(iq) != len(src)*sps {
		t.Fatalf("IQ length = %d, want %d", len(iq), len(src)*sps)
	}

	// FM discriminator → audio.
	fm := NewFM()
	audio := fm.Process(nil, iq)

	// FFSK tone discriminator → soft bit per sample.
	f := NewFFSK(sampleRate, markHz, spaceHz)
	soft := f.Discriminate(nil, audio)

	// Sample one symbol per bit at the symbol centre. Total
	// pipeline delay = FM discriminator (1 sample) + FFSK LPF
	// (Delay()) + tone-discriminator FM (1 sample). The MM clock
	// recovery normally handles this, but for a deterministic
	// round-trip we sample directly at the per-bit centre + group
	// delay. Skip the first few bits while the LPF history is
	// still empty.
	centre := sps / 2
	delay := f.Delay() + 2
	startBit := (delay + sps/2) / sps
	var mismatches, compared int
	for i := startBit; i < len(src); i++ {
		idx := i*sps + centre + delay
		if idx >= len(soft) {
			break
		}
		got := f.Slice(soft[idx])
		want := int(src[i] & 1)
		compared++
		if got != want {
			mismatches++
			if mismatches <= 5 {
				t.Errorf("bit %d: sliced=%d, want=%d, soft=%g",
					i, got, want, soft[idx])
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d/%d bits failed round-trip", mismatches, compared)
	}
}

// TestFFSKModulatorEmitsPhaseContinuousStream: chunked Modulate
// calls must produce the same IQ as a single big call. Both the
// audio phase and the FM-integrator's RF phase must carry across
// call boundaries.
func TestFFSKModulatorEmitsPhaseContinuousStream(t *testing.T) {
	const (
		sampleRate = 48_000.0
		symbolRate = 1200.0
		markHz     = 1200.0
		spaceHz    = 1800.0
	)
	src := make([]byte, 120)
	for i := range src {
		src[i] = byte((i*11 + 5) & 1)
	}

	whole := ModulateFFSK(src, sampleRate, symbolRate, markHz, spaceHz)
	mod := NewFFSKModulator(sampleRate, symbolRate, markHz, spaceHz)
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

// TestFFSKModulatorIQMagnitudeStaysUnity: FM modulation is
// constant-envelope; |IQ| must stay at 1 throughout the stream.
func TestFFSKModulatorIQMagnitudeStaysUnity(t *testing.T) {
	src := []byte{0, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0}
	iq := ModulateFFSK(src, 48_000.0, 1200.0, 1200.0, 1800.0)
	for i, s := range iq {
		mag := math.Hypot(float64(real(s)), float64(imag(s)))
		if math.Abs(mag-1.0) > 1e-6 {
			t.Errorf("sample %d: |IQ| = %g, want 1.0", i, mag)
			break
		}
	}
}

// TestFFSKModulatorResetClearsState: after Reset the modulator
// must behave as if newly constructed.
func TestFFSKModulatorResetClearsState(t *testing.T) {
	src := []byte{0, 1, 1, 0, 1, 0, 0, 1}
	first := ModulateFFSK(src, 48_000.0, 1200.0, 1200.0, 1800.0)

	mod := NewFFSKModulator(48_000.0, 1200.0, 1200.0, 1800.0)
	_ = mod.Modulate([]byte{1, 1, 0, 0, 1, 1, 0, 0}) // dirty state
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

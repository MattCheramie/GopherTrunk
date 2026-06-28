package voice

import (
	"math"
	"testing"
)

func tone(n int, bin float64) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = math.Sin(2 * math.Pi * bin * float64(i) / float64(n))
	}
	return s
}

func maxAbsDiff(a, b []float64) float64 {
	d := 0.0
	for i := range a {
		if x := math.Abs(a[i] - b[i]); x > d {
			d = x
		}
	}
	return d
}

func TestSplitBandInvertSelfInverse(t *testing.T) {
	t.Parallel()
	in := tone(256, 10)
	for i := range in {
		in[i] += 0.3 * math.Sin(2*math.Pi*40*float64(i)/256) // a second component
	}
	got := SplitBandInvert(SplitBandInvert(in, 0.5), 0.5)
	if d := maxAbsDiff(got, in); d > 1e-9 {
		t.Fatalf("split-band not self-inverse: maxdiff %g", d)
	}
}

func TestBandInvertMovesEnergyWithinLowBand(t *testing.T) {
	t.Parallel()
	// A tone near the bottom of the low sub-band should move toward the top of
	// that sub-band after a split-band inversion (energy stays in the low half
	// but flips within it).
	const n = 256
	in := tone(n, 8) // low-frequency tone (bin 8, well inside [0,0.5])
	lowBottom := BandEnergy(in, 0.0, 0.1)
	lowTop := BandEnergy(in, 0.4, 0.5)
	if lowBottom <= lowTop {
		t.Fatalf("setup: tone not at band bottom (bottom=%g top=%g)", lowBottom, lowTop)
	}
	out := SplitBandInvert(in, 0.5)
	// Energy must remain in the low half...
	if BandEnergy(out, 0, 0.5) <= BandEnergy(out, 0.5, 1.0) {
		t.Fatal("split-band inversion leaked energy into the high band")
	}
	// ...but flipped to the top of the low sub-band.
	if BandEnergy(out, 0.4, 0.5) <= BandEnergy(out, 0.0, 0.1) {
		t.Fatal("low-band tone did not invert within its sub-band")
	}
}

func TestRollingInvertRoundTrip(t *testing.T) {
	t.Parallel()
	in := make([]float64, 1024)
	for i := range in {
		in[i] = math.Sin(2*math.Pi*float64(i)*0.05) + 0.5*math.Sin(2*math.Pi*float64(i)*0.2)
	}
	sched := []float64{0.5, 0.7, 1.0, 0.3}
	scrambled := RollingInvert(in, 256, sched)
	got := RollingInvert(scrambled, 256, sched)
	if d := maxAbsDiff(got, in); d > 1e-9 {
		t.Fatalf("rolling invert not self-inverse with a fixed schedule: maxdiff %g", d)
	}
	// Scrambling actually changed the signal.
	if maxAbsDiff(scrambled, in) < 1e-3 {
		t.Fatal("rolling invert did not change the signal")
	}
}

func TestDetectInversionFlagsInvertedFrames(t *testing.T) {
	t.Parallel()
	frame := 256
	// Two frames of low-frequency "speech"; invert the second full-band.
	a := tone(frame, 12)
	b := tone(frame, 12)
	bInv := BandInvert(b, 0, 1)
	sig := append(append([]float64{}, a...), bInv...)
	flags := DetectInversion(sig, frame)
	if len(flags) != 2 || flags[0] || !flags[1] {
		t.Fatalf("detector flags = %v, want [false true]", flags)
	}
	// RollingAuto should undo the inverted second frame.
	out, _ := RollingAuto(sig, frame)
	if BandEnergy(out[frame:], 0, 0.5) <= BandEnergy(out[frame:], 0.5, 1.0) {
		t.Fatal("RollingAuto did not restore the inverted frame to low-band")
	}
}

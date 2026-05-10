package imbe

import (
	"math"
	"testing"
)

func TestAmplitudesFromLog2MlZeroLogIsUnit(t *testing.T) {
	var log2M [57]float64
	for l := 1; l <= 30; l++ {
		log2M[l] = 0
	}
	var dst [57]float64
	AmplitudesFromLog2Ml(&log2M, 30, &dst)
	for l := 1; l <= 30; l++ {
		if !almostEqual(dst[l], 1.0) {
			t.Errorf("dst[%d] = %v, want 1.0 (2^0)", l, dst[l])
		}
	}
}

func TestAmplitudesFromLog2MlPositivePowers(t *testing.T) {
	var log2M [57]float64
	log2M[1] = 1  // 2^1 = 2
	log2M[2] = 2  // 2^2 = 4
	log2M[3] = -1 // 2^-1 = 0.5
	log2M[4] = -3 // 2^-3 = 0.125
	var dst [57]float64
	AmplitudesFromLog2Ml(&log2M, 4, &dst)
	want := []float64{0, 2, 4, 0.5, 0.125}
	for l := 1; l <= 4; l++ {
		if !almostEqual(dst[l], want[l]) {
			t.Errorf("dst[%d] = %v, want %v", l, dst[l], want[l])
		}
	}
}

// TestAmplitudesFromLog2MlClearsTail confirms that AmplitudesFromLog2Ml
// zeroes the indices past L — the contract that lets synthesis loops
// iterate over a fixed [1..56] range without reading stale values
// from a previous frame's longer L.
func TestAmplitudesFromLog2MlClearsTail(t *testing.T) {
	var log2M [57]float64
	dst := [57]float64{}
	for i := range dst {
		dst[i] = 999 // sentinel
	}
	AmplitudesFromLog2Ml(&log2M, 5, &dst)
	for l := 6; l <= 56; l++ {
		if dst[l] != 0 {
			t.Errorf("dst[%d] = %v, want 0 (cleared past L=5)", l, dst[l])
		}
	}
	if dst[0] != 0 {
		t.Errorf("dst[0] = %v, want 0 (1-indexed convention)", dst[0])
	}
}

// TestAmplitudesFromLog2MlSilentLPath: L < 1 should leave dst all-zero
// (the function still scrubs the tail). Lets callers pass a Silent
// frame's L = 0 without an extra guard.
func TestAmplitudesFromLog2MlSilentLPath(t *testing.T) {
	var log2M [57]float64
	log2M[1] = 5 // arbitrary
	dst := [57]float64{}
	for i := range dst {
		dst[i] = 999
	}
	AmplitudesFromLog2Ml(&log2M, 0, &dst)
	for l := 0; l <= 56; l++ {
		if dst[l] != 0 {
			t.Errorf("dst[%d] = %v, want 0 (Silent path)", l, dst[l])
		}
	}
}

// TestAmplitudesFromLog2MlClampsHighL: an L > 56 (caller bug or
// future-proofing) should clamp at 56, not blow past the array
// bounds. The struct extents are pinned at [57] so this protects
// the contract.
func TestAmplitudesFromLog2MlClampsHighL(t *testing.T) {
	var log2M [57]float64
	for l := 1; l <= 56; l++ {
		log2M[l] = 0 // → 1.0
	}
	var dst [57]float64
	AmplitudesFromLog2Ml(&log2M, 99, &dst)
	for l := 1; l <= 56; l++ {
		if !almostEqual(dst[l], 1.0) {
			t.Errorf("dst[%d] = %v, want 1.0 (L clamped to 56)", l, dst[l])
		}
	}
}

func TestFrameEnergyUnitAmplitudesEqualsL(t *testing.T) {
	var M [57]float64
	for l := 1; l <= 20; l++ {
		M[l] = 1.0
	}
	got := FrameEnergy(&M, 20)
	if !almostEqual(got, 20.0) {
		t.Errorf("FrameEnergy = %v, want 20.0 (unit Ml × L=20)", got)
	}
}

func TestFrameEnergyKnownValues(t *testing.T) {
	var M [57]float64
	M[1] = 1
	M[2] = 2
	M[3] = 3
	got := FrameEnergy(&M, 3)
	want := 1.0 + 4.0 + 9.0
	if !almostEqual(got, want) {
		t.Errorf("FrameEnergy = %v, want %v", got, want)
	}
}

func TestFrameEnergyIgnoresPastL(t *testing.T) {
	var M [57]float64
	M[1] = 1
	M[2] = 1
	M[3] = 99 // should be ignored
	got := FrameEnergy(&M, 2)
	if !almostEqual(got, 2.0) {
		t.Errorf("FrameEnergy = %v, want 2.0 (M[3] past L)", got)
	}
}

func TestFrameEnergyZeroL(t *testing.T) {
	var M [57]float64
	M[1] = 99
	if got := FrameEnergy(&M, 0); got != 0 {
		t.Errorf("FrameEnergy(L=0) = %v, want 0", got)
	}
}

func TestVoicingFractionAllVoiced(t *testing.T) {
	var Vl [57]int
	for l := 1; l <= 12; l++ {
		Vl[l] = 1
	}
	got := VoicingFraction(&Vl, 12)
	if !almostEqual(got, 1.0) {
		t.Errorf("VoicingFraction = %v, want 1.0", got)
	}
}

func TestVoicingFractionAllUnvoiced(t *testing.T) {
	var Vl [57]int
	got := VoicingFraction(&Vl, 12)
	if got != 0 {
		t.Errorf("VoicingFraction = %v, want 0", got)
	}
}

func TestVoicingFractionMixed(t *testing.T) {
	var Vl [57]int
	// 4 voiced of 10 → 0.4
	Vl[1] = 1
	Vl[3] = 1
	Vl[5] = 1
	Vl[7] = 1
	got := VoicingFraction(&Vl, 10)
	if !almostEqual(got, 0.4) {
		t.Errorf("VoicingFraction = %v, want 0.4", got)
	}
}

func TestVoicingFractionIgnoresPastL(t *testing.T) {
	var Vl [57]int
	Vl[1] = 1
	Vl[2] = 1
	Vl[3] = 1 // past L
	got := VoicingFraction(&Vl, 2)
	if !almostEqual(got, 1.0) {
		t.Errorf("VoicingFraction = %v, want 1.0 (Vl[3] past L)", got)
	}
}

func TestVoicingFractionZeroL(t *testing.T) {
	var Vl [57]int
	Vl[1] = 1
	if got := VoicingFraction(&Vl, 0); got != 0 {
		t.Errorf("VoicingFraction(L=0) = %v, want 0", got)
	}
}

// TestVoicingFractionRejectsBogusVl: Vl entries outside {0, 1} count
// as zero (defensive — upstream only writes 0/1, so anything else
// is a programming bug and we fail closed rather than over-counting).
func TestVoicingFractionRejectsBogusVl(t *testing.T) {
	var Vl [57]int
	Vl[1] = 2 // bogus
	Vl[2] = -1
	Vl[3] = 1
	got := VoicingFraction(&Vl, 3)
	if !almostEqual(got, 1.0/3.0) {
		t.Errorf("VoicingFraction = %v, want 1/3 (bogus Vl ignored)", got)
	}
}

// TestSpectralCosineSumAtZeroOmega: ω₀ = 0 ⇒ cos(0) = 1 ⇒ R_M1
// equals R_M0. A handy degeneracy check for the §6.2 weight
// formula — the enhancement weight collapses cleanly at the limit.
func TestSpectralCosineSumAtZeroOmega(t *testing.T) {
	var M [57]float64
	for l := 1; l <= 5; l++ {
		M[l] = float64(l) // 1, 2, 3, 4, 5
	}
	rm0 := FrameEnergy(&M, 5)
	rm1 := SpectralCosineSum(&M, 5, 0)
	if !almostEqual(rm0, rm1) {
		t.Errorf("R_M0 = %v, R_M1 = %v at ω₀=0; want equal", rm0, rm1)
	}
}

// TestSpectralCosineSumAtPiOmega: ω₀ = π ⇒ cos(πl) alternates
// (-1)^l ⇒ R_M1 = Σ Ml² · (-1)^l. With M[l] = 1 for l in 1..4,
// R_M1 = -1 + 1 - 1 + 1 = 0. Pins the alternating-cosine path.
func TestSpectralCosineSumAtPiOmega(t *testing.T) {
	var M [57]float64
	for l := 1; l <= 4; l++ {
		M[l] = 1
	}
	got := SpectralCosineSum(&M, 4, math.Pi)
	if !almostEqual(got, 0) {
		t.Errorf("R_M1 = %v, want 0 (alternating with even count)", got)
	}
}

func TestSpectralCosineSumZeroL(t *testing.T) {
	var M [57]float64
	M[1] = 99
	if got := SpectralCosineSum(&M, 0, 1.0); got != 0 {
		t.Errorf("SpectralCosineSum(L=0) = %v, want 0", got)
	}
}

// TestEndToEndPathFromTl: integration between the just-shipped
// step 4a (PredictLog2Ml) and this step's helpers. A first frame
// with Tl[l] = log2(l+1) goes through PredictLog2Ml (no prev-state
// shortcut) → AmplitudesFromLog2Ml → FrameEnergy. Confirms the
// pipeline shape the synthesizer will use one frame at a time.
func TestEndToEndPathFromTl(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 4, K: 2}}
	for l := 1; l <= p.L; l++ {
		p.Tl[l] = math.Log2(float64(l + 1))
	}
	var log2M [57]float64
	PredictLog2Ml(&s, p, &log2M)

	var M [57]float64
	AmplitudesFromLog2Ml(&log2M, p.L, &M)

	want := []float64{0, 2, 3, 4, 5}
	for l := 1; l <= p.L; l++ {
		if !almostEqual(M[l], want[l]) {
			t.Errorf("M[%d] = %v, want %v (2^Tl[%d] for first frame)",
				l, M[l], want[l], l)
		}
	}

	// Energy = 4 + 9 + 16 + 25 = 54
	got := FrameEnergy(&M, p.L)
	if !almostEqual(got, 54) {
		t.Errorf("FrameEnergy = %v, want 54", got)
	}
}

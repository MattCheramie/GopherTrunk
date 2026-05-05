package diversity

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestSelectionPicksLoudestBranch(t *testing.T) {
	branches := [][]complex64{
		{complex(0.1, 0), complex(0.5, 0), complex(0.0, 0)},
		{complex(0.9, 0), complex(0.4, 0), complex(0.7, 0)},
		{complex(0.2, 0), complex(0.8, 0), complex(0.1, 0)},
	}
	out, err := NewSelection().Combine(branches)
	if err != nil {
		t.Fatal(err)
	}
	want := []complex64{complex(0.9, 0), complex(0.8, 0), complex(0.7, 0)}
	for i := range want {
		if real(out[i]) != real(want[i]) || imag(out[i]) != imag(want[i]) {
			t.Errorf("out[%d] = %v, want %v", i, out[i], want[i])
		}
	}
}

func TestSelectionSingleBranchPassThrough(t *testing.T) {
	in := []complex64{complex(1, 2), complex(3, 4)}
	out, err := NewSelection().Combine([][]complex64{in})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("len(out) = %d", len(out))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("out[%d] = %v, want %v", i, out[i], in[i])
		}
	}
	// Mutating the output must not affect the input (separate buffer).
	out[0] = 0
	if in[0] == 0 {
		t.Error("Selection returned a shared buffer, want a copy")
	}
}

func TestSelectionRejectsLengthMismatch(t *testing.T) {
	_, err := NewSelection().Combine([][]complex64{
		{complex(1, 0), complex(2, 0)},
		{complex(3, 0)},
	})
	if err == nil {
		t.Fatal("expected error on mismatched branch lengths")
	}
}

func TestSelectionRequiresOneBranch(t *testing.T) {
	if _, err := NewSelection().Combine(nil); err == nil {
		t.Fatal("expected error on zero branches")
	}
}

func TestMRCFavorsHighPowerBranch(t *testing.T) {
	mrc := NewMRC(2, 0.5)
	// Branch 0: small constant. Branch 1: 10× larger.
	n := 256
	b0 := make([]complex64, n)
	b1 := make([]complex64, n)
	for i := range b0 {
		b0[i] = complex(0.1, 0)
		b1[i] = complex(1.0, 0)
	}
	// Run several chunks so the EMA settles.
	var out []complex64
	for k := 0; k < 8; k++ {
		var err error
		out, err = mrc.Combine([][]complex64{b0, b1})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(out) != n {
		t.Fatalf("len(out) = %d", len(out))
	}
	// After convergence, output should track branch 1 much more
	// closely than branch 0. Threshold: > 0.7.
	if math.Abs(float64(real(out[0])-1.0)) > 0.3 {
		t.Errorf("steady-state output real = %g, want ≈ 1.0", real(out[0]))
	}
	powers := mrc.PowerEstimates()
	if powers[1] <= powers[0] {
		t.Errorf("PowerEstimates = %v, expected branch 1 > branch 0", powers)
	}
}

func TestMRCSilentBranchTreatedAsLowWeight(t *testing.T) {
	mrc := NewMRC(2, 0.5)
	n := 128
	b0 := make([]complex64, n)
	b1 := make([]complex64, n)
	for i := range b0 {
		b0[i] = complex(0.0, 0)
		b1[i] = complex(0.7, 0)
	}
	for k := 0; k < 8; k++ {
		_, err := mrc.Combine([][]complex64{b0, b1})
		if err != nil {
			t.Fatal(err)
		}
	}
	powers := mrc.PowerEstimates()
	if powers[0] != 0 {
		t.Errorf("silent branch power = %g, want 0", powers[0])
	}
	if powers[1] == 0 {
		t.Errorf("active branch power = 0, want > 0")
	}
}

func TestMRCAllSilentReturnsFlatSum(t *testing.T) {
	mrc := NewMRC(2, 0.5)
	n := 4
	b0 := make([]complex64, n)
	b1 := make([]complex64, n)
	out, err := mrc.Combine([][]complex64{b0, b1})
	if err != nil {
		t.Fatal(err)
	}
	for i := range out {
		if out[i] != 0 {
			t.Errorf("out[%d] = %v, want 0 on silent input", i, out[i])
		}
	}
}

func TestMRCPilotModeUsesProvidedGain(t *testing.T) {
	mrc := NewMRC(2, 0.5)
	mrc.SetGain(0, complex(1, 0))
	mrc.SetGain(1, complex(0, 1)) // 90° offset
	in := []complex64{complex(1, 0), complex(0, 1)}
	branches := [][]complex64{
		{in[0]},
		{in[1]},
	}
	out, err := mrc.Combine(branches)
	if err != nil {
		t.Fatal(err)
	}
	// Weight vector: conj(1) = 1, conj(j) = -j.
	// y = (1 * (1+0j) + (-j) * (0+1j)) / (|1|^2 + |j|^2)
	//   = (1 + (-j)(j)) / 2
	//   = (1 + 1) / 2 = 1.
	expected := complex(1, 0)
	got := complex128(out[0])
	if cmplx.Abs(got-complex128(expected)) > 1e-5 {
		t.Errorf("pilot-mode out = %v, want %v", got, expected)
	}
}

func TestMRCResetClearsPower(t *testing.T) {
	mrc := NewMRC(2, 0.5)
	branch := []complex64{complex(1, 0), complex(1, 0)}
	for k := 0; k < 4; k++ {
		_, _ = mrc.Combine([][]complex64{branch, branch})
	}
	mrc.Reset()
	for _, p := range mrc.PowerEstimates() {
		if p != 0 {
			t.Errorf("power after Reset = %g, want 0", p)
		}
	}
}

func TestMRCRejectsBranchCountMismatch(t *testing.T) {
	mrc := NewMRC(2, 0.5)
	_, err := mrc.Combine([][]complex64{{complex(1, 0)}})
	if err == nil {
		t.Fatal("expected error on branch-count mismatch")
	}
}

package acelp

import (
	"math"
	"math/cmplx"
	"testing"
)

// orderedLSPs is a valid strictly-decreasing cosine-domain LSP set (the
// reference decoder's reset value).
var orderedLSPs = [lpcOrder]int16{30000, 26000, 21000, 15000, 8000, 0, -8000, -15000, -21000, -26000}

// evalPoly evaluates sum coef[m]*e^{-j m w} at frequency w.
func evalPoly(coef []float64, w float64) complex128 {
	var s complex128
	for m, c := range coef {
		s += complex(c, 0) * cmplx.Exp(complex(0, -float64(m)*w))
	}
	return s
}

// TestLspAzRootProperty is an independent correctness check of the LSP→LPC
// conversion: reconstruct A(z) from the LSPs, form the symmetric/antisymmetric
// LSP polynomials F1(z)=A(z)+z^-(p+1)A(1/z) and F2(z)=A(z)-z^-(p+1)A(1/z), and
// verify that every LSP frequency ω_i = acos(lsp_i) is a root of one of them
// (min(|F1|,|F2|) ≈ 0). This validates the algorithm without re-deriving it.
func TestLspAzRootProperty(t *testing.T) {
	a := lspAz(orderedLSPs[:])

	// af[0..10] = a[i]/4096 (Q12 -> float), af[11] = 0.
	af := make([]float64, 12)
	for i := 0; i <= lpcOrder; i++ {
		af[i] = float64(a[i]) / 4096.0
	}
	// c(z) = z^-(p+1) A(1/z): c[m] = af[11-m], m=1..11.
	f1 := make([]float64, 12)
	f2 := make([]float64, 12)
	for m := 0; m <= 11; m++ {
		c := 0.0
		if m >= 1 {
			c = af[11-m]
		}
		f1[m] = af[m] + c
		f2[m] = af[m] - c
	}

	for i, q := range orderedLSPs {
		w := math.Acos(float64(q) / 32768.0)
		m1 := cmplx.Abs(evalPoly(f1, w))
		m2 := cmplx.Abs(evalPoly(f2, w))
		if min := math.Min(m1, m2); min > 0.02 {
			t.Errorf("LSP %d (ω=%.3f): min(|F1|,|F2|)=%.4f, not a root", i, w, min)
		}
	}
}

// TestLspAzHeader checks the fixed leading coefficient.
func TestLspAzHeader(t *testing.T) {
	a := lspAz(orderedLSPs[:])
	if a[0] != 4096 {
		t.Errorf("a[0] = %d, want 4096 (1.0 in Q12)", a[0])
	}
}

// TestFacPond checks the spectral-expansion factors are gamma^(i+1).
func TestFacPond(t *testing.T) {
	const gamma = 24576 // 0.75 in Q15
	fac := facPond(gamma)
	for i := 0; i < lpcOrder; i++ {
		want := math.Pow(0.75, float64(i+1))
		got := float64(fac[i]) / 32768.0
		if math.Abs(got-want) > 0.001 {
			t.Errorf("fac[%d] = %.4f, want %.4f", i, got, want)
		}
	}
}

// TestPondAi checks bandwidth-expanded coefficients equal a[i]*fac[i-1].
func TestPondAi(t *testing.T) {
	a := lspAz(orderedLSPs[:])
	fac := facPond(24576)
	aExp := pondAi(a[:], fac)
	if aExp[0] != a[0] {
		t.Errorf("aExp[0] = %d, want %d", aExp[0], a[0])
	}
	for i := 1; i <= lpcOrder; i++ {
		want := etsiRound(lMult(a[i], fac[i-1]))
		if aExp[i] != want {
			t.Errorf("aExp[%d] = %d, want %d", i, aExp[i], want)
		}
		// magnitude should shrink (expansion pulls poles inward)
		if absS(aExp[i]) > absS(a[i]) {
			t.Errorf("aExp[%d]=%d larger than a[%d]=%d", i, aExp[i], i, a[i])
		}
	}
}

// TestIntLpc4 checks the four-subframe interpolation endpoints: the last
// subframe uses the new LSPs exactly, and all four coefficient sets start with
// 4096.
func TestIntLpc4(t *testing.T) {
	lspOld := orderedLSPs
	lspNew := [lpcOrder]int16{31000, 25000, 20000, 14000, 7000, -1000, -9000, -16000, -22000, -27000}
	a := intLpc4(lspOld, lspNew)

	last := lspAz(lspNew[:])
	for i := 0; i <= lpcOrder; i++ {
		if a[33+i] != last[i] {
			t.Errorf("last subframe a[33+%d]=%d, want %d (lspAz(new))", i, a[33+i], last[i])
		}
	}
	for s := 0; s < numSubfr; s++ {
		if a[s*(lpcOrder+1)] != 4096 {
			t.Errorf("subframe %d: leading coef %d, want 4096", s, a[s*(lpcOrder+1)])
		}
	}

	// First subframe should equal lspAz of the 3/4-old + 1/4-new interpolation.
	var lsp0 [lpcOrder]int16
	for i := 0; i < lpcOrder; i++ {
		t0 := lMult(lspOld[i], 24576)
		t0 = lMac(t0, lspNew[i], 8192)
		lsp0[i] = extractH(t0)
	}
	first := lspAz(lsp0[:])
	for i := 0; i <= lpcOrder; i++ {
		if a[i] != first[i] {
			t.Errorf("first subframe a[%d]=%d, want %d", i, a[i], first[i])
		}
	}
}

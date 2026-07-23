package acelp

import "testing"

// unitA is the trivial LPC filter A(z)=1 (1/A(z) is identity).
func unitA() []int16 {
	a := make([]int16, lpcOrder+1)
	a[0] = 4096
	return a
}

// synFiltFloat is an independent float reference of the synthesis recursion
// y[i] = x[i] - sum_{j=1}^{p} (a[j]/4096) * y[i-j], with zero initial memory.
func synFiltFloat(a []int16, x []int16) []float64 {
	y := make([]float64, len(x))
	for i := range x {
		s := float64(x[i])
		for j := 1; j <= lpcOrder; j++ {
			if i-j >= 0 {
				s -= float64(a[j]) / 4096.0 * y[i-j]
			}
		}
		y[i] = s
	}
	return y
}

// TestSynFiltIdentity checks 1/A(z) with A(z)=1 passes the input through.
func TestSynFiltIdentity(t *testing.T) {
	x := []int16{100, -200, 300, -50, 1234, -1000, 7, 0, -9, 500}
	y := make([]int16, len(x))
	mem := make([]int16, lpcOrder)
	synFilt(unitA(), x, y, len(x), mem, false)
	for i := range x {
		if y[i] != x[i] {
			t.Errorf("identity: y[%d]=%d, want %d", i, y[i], x[i])
		}
	}
}

// TestSynFiltVsFloat checks the fixed-point synthesis filter tracks the float
// recursion within a small tolerance for a real (stable) LPC filter.
func TestSynFiltVsFloat(t *testing.T) {
	a := lspAz(orderedLSPs[:]) // a real, stable A(z)
	x := make([]int16, 60)
	for i := range x {
		x[i] = int16(((i*37+11)%400 - 200)) // deterministic pseudo-signal
	}
	y := make([]int16, len(x))
	mem := make([]int16, lpcOrder)
	synFilt(a[:], x, y, len(x), mem, false)

	ref := synFiltFloat(a[:], x)
	for i := range x {
		if d := float64(y[i]) - ref[i]; d < -8 || d > 8 {
			t.Errorf("y[%d]=%d, float ref %.1f (diff %.1f)", i, y[i], ref[i], d)
		}
	}
}

// TestSynFiltMemoryContinuity checks that filtering in two calls with memory
// update matches filtering the whole signal in one call.
func TestSynFiltMemoryContinuity(t *testing.T) {
	a := lspAz(orderedLSPs[:])
	x := make([]int16, 40)
	for i := range x {
		x[i] = int16((i*13+5)%200 - 100)
	}

	whole := make([]int16, len(x))
	mem := make([]int16, lpcOrder)
	synFilt(a[:], x, whole, len(x), mem, false)

	split := make([]int16, len(x))
	mem2 := make([]int16, lpcOrder)
	synFilt(a[:], x[:20], split[:20], 20, mem2, true)
	synFilt(a[:], x[20:], split[20:], 20, mem2, true)

	for i := range x {
		if whole[i] != split[i] {
			t.Errorf("continuity mismatch at %d: whole %d, split %d", i, whole[i], split[i])
		}
	}
}

// TestLpcGainUnit checks the impulse-response energy for A(z)=1: the response is
// just the input impulse (1024 in Q10), so energy = 1024^2.
func TestLpcGainUnit(t *testing.T) {
	got := lpcGain(unitA())
	want := int32(1024 * 1024)
	if got != want {
		t.Errorf("lpcGain(unit) = %d, want %d", got, want)
	}
}

// TestLpcGainResonant checks a real LPC filter has larger impulse-response
// energy than the flat filter (resonances amplify).
func TestLpcGainResonant(t *testing.T) {
	a := lspAz(orderedLSPs[:])
	got := lpcGain(a[:])
	if got <= 1024*1024 {
		t.Errorf("resonant lpcGain = %d, expected > %d", got, 1024*1024)
	}
}

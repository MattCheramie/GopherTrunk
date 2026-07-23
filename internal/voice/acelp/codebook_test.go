package acelp

import "testing"

// TestPredLtInteger checks the integer-lag (frac==0) adaptive codebook path:
// the excitation is copied from delay T0.
func TestPredLtInteger(t *testing.T) {
	const base, T0 = 200, 50
	exc := make([]int16, base+subfrLen)
	for i := 0; i < base; i++ {
		exc[i] = int16((i*13 + 7) % 100)
	}
	predLt(exc, base, T0, 0, subfrLen)
	for i := 0; i < subfrLen; i++ {
		if exc[base+i] != exc[base+i-T0] {
			t.Fatalf("frac=0: exc[%d]=%d, want %d", base+i, exc[base+i], exc[base+i-T0])
		}
	}
}

// TestInter32DCGain checks the fractional interpolators preserve a DC signal:
// interpolating a constant returns (approximately) that constant, i.e. the
// filter gain is ~1.0. This validates the tap tables without re-deriving them.
func TestInter32DCGain(t *testing.T) {
	const c, val = 40, int16(1000)
	exc := make([]int16, 80)
	for i := range exc {
		exc[i] = val
	}
	for name, got := range map[string]int16{
		"+1/3": inter32_1_3(exc, c),
		"-1/3": inter32M1_3(exc, c),
	} {
		if diff := int(got) - int(val); diff < -3 || diff > 3 {
			t.Errorf("inter32 %s DC: got %d, want ~%d", name, got, val)
		}
	}
}

// TestAlgebraicPulses checks the pulse-position decoding for index 0 and a
// spread index, matching the D4i60 grid.
func TestAlgebraicPulses(t *testing.T) {
	p0, p1, p2, p3 := algebraicPulses(0)
	if p0 != 0 || p1 != 2 || p2 != 4 || p3 != 6 {
		t.Errorf("index 0 pulses = %d,%d,%d,%d want 0,2,4,6", p0, p1, p2, p3)
	}
	// index with one bit set in each field: bit0, bit5, bit8, bit11.
	idx := int16(1 | (1 << 5) | (1 << 8) | (1 << 11))
	p0, p1, p2, p3 = algebraicPulses(idx)
	if p0 != 2 || p1 != (32>>2)+2 || p2 != (256>>5)+4 || p3 != (2048>>8)+6 {
		t.Errorf("spread pulses = %d,%d,%d,%d", p0, p1, p2, p3)
	}
}

// TestDD4i60Impulse drives dD4i60 with an impulse noise filter so the output
// is exactly the 4-pulse codeword: pulse 0 scaled by sqrt(2) (Q11 2896),
// pulses 1..3 at ±2048 (=1.0 in Q11) with the +,-,+,- pattern, negated when
// sign is set.
func TestDD4i60Impulse(t *testing.T) {
	const Fbase = 64
	F := make([]int16, Fbase+subfrLen)
	F[Fbase] = 2048 // impulse of value 1.0 in Q11 at F[0]

	cod := make([]int16, subfrLen)
	dD4i60(0, 0, 0, F, Fbase, cod) // positions 0,2,4,6; shift 0; sign +

	want := map[int]int16{0: 2896, 2: -2048, 4: 2048, 6: -2048}
	for i := 0; i < subfrLen; i++ {
		if cod[i] != want[i] { // want[i] is 0 for unlisted positions
			t.Errorf("cod[%d] = %d, want %d", i, cod[i], want[i])
		}
	}

	// sign=1 negates the whole codeword.
	dD4i60(0, 1, 0, F, Fbase, cod)
	for i, w := range want {
		if cod[i] != -w {
			t.Errorf("sign=1 cod[%d] = %d, want %d", i, cod[i], -w)
		}
	}
}

// TestDD4i60Shift checks the pulse shift moves every pulse one sample later.
func TestDD4i60Shift(t *testing.T) {
	const Fbase = 64
	F := make([]int16, Fbase+subfrLen)
	F[Fbase] = 2048

	cod := make([]int16, subfrLen)
	dD4i60(0, 0, 1, F, Fbase, cod) // shift 1: positions 1,3,5,7

	want := map[int]int16{1: 2896, 3: -2048, 5: 2048, 7: -2048}
	for i := 0; i < subfrLen; i++ {
		if cod[i] != want[i] {
			t.Errorf("shifted cod[%d] = %d, want %d", i, cod[i], want[i])
		}
	}
}

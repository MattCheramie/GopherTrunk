package acelp

import "testing"

// gainInputs builds a plausible (a, prdLt, code) triple for the dequantiser.
func gainInputs() (a, prdLt, code []int16) {
	av := lspAz(orderedLSPs[:])
	a = av[:]
	prdLt = make([]int16, subfrLen)
	code = make([]int16, subfrLen)
	for i := range prdLt {
		prdLt[i] = int16((i*29+7)%600 - 300)
		code[i] = int16((i*53+3)%400 - 200)
	}
	return a, prdLt, code
}

// TestDecEnerBounds checks the dequantised gains stay in range across all 64
// energy indices: 0 <= gainPit <= 4915 (1.2 in Q12), gainCod >= 0.
func TestDecEnerBounds(t *testing.T) {
	a, prdLt, code := gainInputs()
	for idx := int16(0); idx < enerQuaEntries; idx++ {
		e := &enerPredictor{lastEnerPit: 2000, lastEnerCod: 2000}
		gp, gc := e.decEner(idx, false, a, prdLt, code, subfrLen)
		if gp < 0 || gp > 4915 {
			t.Errorf("index %d: gainPit %d out of [0,4915]", idx, gp)
		}
		if gc < 0 {
			t.Errorf("index %d: gainCod %d negative", idx, gc)
		}
	}
}

// TestDecEnerBfiDecay checks that on erased frames the stored energies decay by
// 0.5 (128 in Q8) per call down to zero, and the pitch gain is non-increasing.
func TestDecEnerBfiDecay(t *testing.T) {
	a, prdLt, code := gainInputs()
	e := &enerPredictor{lastEnerPit: 500, lastEnerCod: 500}
	prevPit := e.lastEnerPit
	prevGain := int16(4915)
	for call := 0; call < 6; call++ {
		wantPit := prevPit - 128
		if wantPit < 0 {
			wantPit = 0
		}
		gp, _ := e.decEner(0, true, a, prdLt, code, subfrLen)
		if e.lastEnerPit != wantPit {
			t.Errorf("call %d: lastEnerPit %d, want %d", call, e.lastEnerPit, wantPit)
		}
		if gp > prevGain {
			t.Errorf("call %d: gainPit rose %d -> %d under BFI decay", call, prevGain, gp)
		}
		prevPit = e.lastEnerPit
		prevGain = gp
	}
	if e.lastEnerPit != 0 {
		t.Errorf("energy did not decay to 0: %d", e.lastEnerPit)
	}
}

// TestDecEnerIndexMonotonic checks that, from the same prediction state, a
// larger quantised pitch energy yields a pitch gain that is not smaller.
func TestDecEnerIndexMonotonic(t *testing.T) {
	a, prdLt, code := gainInputs()

	// Find two indices whose pitch-energy table entries differ noticeably.
	var lo, hi int16 = -1, -1
	for idx := int16(0); idx < enerQuaEntries; idx++ {
		if enerQua[2*idx] < enerQua[0]+50 {
			lo = idx
		}
		if enerQua[2*idx] > enerQua[0]+1500 {
			hi = idx
		}
	}
	if lo < 0 || hi < 0 {
		t.Skip("no suitable index pair")
	}

	eLo := &enerPredictor{lastEnerPit: 1500, lastEnerCod: 1500}
	gLo, _ := eLo.decEner(lo, false, a, prdLt, code, subfrLen)
	eHi := &enerPredictor{lastEnerPit: 1500, lastEnerCod: 1500}
	gHi, _ := eHi.decEner(hi, false, a, prdLt, code, subfrLen)

	if gHi < gLo {
		t.Errorf("higher energy index gave smaller gain: lo(idx %d)=%d, hi(idx %d)=%d", lo, gLo, hi, gHi)
	}
}

// TestDecEnerState checks the predictor updates its state (not a no-op) on a
// good frame and that the same index is reproducible from the same state.
func TestDecEnerState(t *testing.T) {
	a, prdLt, code := gainInputs()
	e1 := &enerPredictor{lastEnerPit: 1000, lastEnerCod: 1000}
	gp1, gc1 := e1.decEner(20, false, a, prdLt, code, subfrLen)
	e2 := &enerPredictor{lastEnerPit: 1000, lastEnerCod: 1000}
	gp2, gc2 := e2.decEner(20, false, a, prdLt, code, subfrLen)
	if gp1 != gp2 || gc1 != gc2 {
		t.Errorf("not reproducible: (%d,%d) vs (%d,%d)", gp1, gc1, gp2, gc2)
	}
	if e1.lastEnerPit == 1000 && e1.lastEnerCod == 1000 {
		t.Errorf("state not updated on good frame")
	}
}

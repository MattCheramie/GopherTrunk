package acelp

// Gain (energy) dequantisation for the TETRA ACELP decoder, ETSI EN 300 395-2.
// Per subframe the pitch and innovation-code gains are recovered from a 6-bit
// energy index. The decoder works in the log2 energy domain: it computes the
// energy of the 1/A(z) impulse response, the adaptive-codebook vector and the
// innovation code, predicts the pitch/code energies from the previous
// subframe (MA prediction), adds the dequantised energy correction, and
// converts back to a linear gain via pow2. Clean-room port of Dec_Ener.

// enerPredictor carries the inter-subframe energy-prediction state
// (last_ener_pit / last_ener_cod, Q8), which the MA predictor and the bad-frame
// decay both read and update.
type enerPredictor struct {
	lastEnerPit int16
	lastEnerCod int16
}

// decEner dequantises the pitch and code gains for one subframe. index is the
// 6-bit energy VQ index; bfi marks an erased frame; a is the p+1 LPC
// coefficients; prdLt the adaptive-codebook vector; code the innovation code;
// subfr the subframe length. gainPit is Q12 (clamped to 1.2), gainCod is Q0.
func (e *enerPredictor) decEner(index int16, bfi bool, a, prdLt, code []int16, subfr int) (gainPit, gainCod int16) {
	// Energy of the 1/A(z) impulse response.
	L := lpcGain(a)
	expLpc := normL(L)
	enerLpc := extractH(lShl(L, expLpc))

	// Energy of the adaptive (pitch) codebook vector, in log2.
	L = 1 // avoid all-zeros
	for i := 0; i < subfr; i++ {
		L = lMac0(L, prdLt[i], prdLt[i])
	}
	expPlt := normL(L)
	enerPlt16 := extractH(lShl(L, expPlt))

	L = lMult0(enerPlt16, enerLpc)
	expPlt = addOp(expPlt, expLpc)
	exp, frac := log2fp(L)
	L = loadSh16(exp)
	L = addSh(L, frac, 1)
	L = subSh16(L, expPlt)
	L = addSh(L, 1710, 8) // +6.68 (Q16) scaling
	L = lShr(L, 8)        // Q16 -> Q8
	enerPlt := extractL(L)

	// Energy of the innovation code, in log2.
	L = 0
	for i := 0; i < subfr; i++ {
		L = lMac0(L, code[i], code[i])
	}
	enerC16 := extractH(L)
	L = lMult0(enerC16, enerLpc)
	exp, frac = log2fp(L)
	L = loadSh16(exp)
	L = addSh(L, frac, 1)
	L = subSh16(L, expLpc)
	L = subSh(L, 4434, 8) // -17.32 (Q16) scaling
	L = lShr(L, 8)
	enerC := extractL(L)

	if bfi {
		e.lastEnerPit = subOp(e.lastEnerPit, 128) // -0.5 in Q8
		if e.lastEnerPit < 0 {
			e.lastEnerPit = 0
		}
		e.lastEnerCod = subOp(e.lastEnerCod, 128)
		if e.lastEnerCod < 0 {
			e.lastEnerCod = 0
		}
	} else {
		// Predicted pitch energy: 0.5*last_pit + 0.25*last_cod - 3.0.
		Lp := loadSh(e.lastEnerPit, 8)
		Lp = addSh(Lp, e.lastEnerCod, 7)
		Lp = subSh(Lp, 768, 9) // -3.0 in Q9
		if Lp < 0 {
			Lp = 0
		}
		predPit := storeHi(Lp, 7) // Q8

		// Predicted code energy: 0.5*last_cod + 0.25*last_pit - 3.0.
		Lc := loadSh(e.lastEnerCod, 8)
		Lc = addSh(Lc, e.lastEnerPit, 7)
		Lc = subSh(Lc, 768, 9)
		if Lc < 0 {
			Lc = 0
		}
		predCod := storeHi(Lc, 7)

		j := shl(index, 1)
		e.lastEnerPit = addOp(enerQua[j], predPit)
		e.lastEnerCod = addOp(enerQua[j+1], predCod)

		if e.lastEnerPit > 6912 { // 27 in Q8
			e.lastEnerPit = 6912
		}
		if e.lastEnerCod > 6400 { // 25 in Q8
			e.lastEnerCod = 6400
		}
	}

	// Pitch gain = pow2(0.5*(last_ener_pit - ener_plt)), clamped to 1.2 (Q12).
	L = loadSh(e.lastEnerPit, 6)
	L = subSh(L, enerPlt, 6)
	L = addSh(L, 12, 15) // gain in Q12
	exp, frac = lExtract(L)
	L = pow2fp(exp, frac)
	if lSub(L, 4915) > 0 { // 1.2 in Q12
		L = 4915
	}
	gainPit = extractL(L)

	// Code gain = pow2(0.5*(last_ener_cod - ener_c)).
	L = loadSh(e.lastEnerCod, 6)
	L = subSh(L, enerC, 6)
	exp, frac = lExtract(L)
	L = pow2fp(exp, frac)
	gainCod = extractL(L)

	return gainPit, gainCod
}

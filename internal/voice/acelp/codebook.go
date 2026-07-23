package acelp

// Adaptive (pitch) and algebraic (innovation) codebooks for the TETRA ACELP
// decoder, ETSI EN 300 395-2. Per subframe the decoder builds the excitation
// from two contributions:
//
//   - the adaptive codebook (predLt): the past excitation delayed by the pitch
//     lag T0 (+ a 1/3-sample fractional offset via a 32-tap interpolation
//     filter), reconstructing the periodic component;
//   - the algebraic codebook (dD4i60): a 4-pulse innovation sequence convolved
//     with the perceptual noise filter F[], reconstructing the stochastic
//     component.
//
// Clean-room ports of the reference Pred_Lt / Inter32_1_3 / Inter32_M1_3 /
// D_D4i60. The interpolation filter taps and the sqrt(2) impulse gain are
// numeric constants defined by the standard (see package doc for provenance).

const (
	subfrLen  = 60   // L_subfr — samples per subframe
	q11GainI0 = 2896 // sqrt(2) in Q11 — gain of the first algebraic pulse
)

// inter32Coef1_3 / inter32Coef_M1_3 are the 32-tap fractional-delay filters for
// the +1/3 and -1/3 sample pitch offsets (EN 300 395-2 long-term prediction).
var inter32Coef1_3 = [32]int16{
	-47, 59, -84, 125, -183, 263, -366, 500, -672, 893,
	-1185, 1587, -2182, 3179, -5287, 13496, 27072, -6669, 3688, -2452,
	1758, -1304, 981, -739, 553, -407, 294, -207, 142, -96,
	66, -49,
}

var inter32CoefM1_3 = [32]int16{
	-49, 66, -96, 142, -207, 294, -407, 553, -739,
	981, -1304, 1758, -2452, 3688, -6669, 27072, 13496, -5287, 3179,
	-2182, 1587, -1185, 893, -672, 500, -366, 263, -183, 125,
	-84, 59, -47,
}

// inter32_1_3 interpolates the +1/3-sample value centred at exc[c], reading
// exc[c-16 .. c+15]. Result is the interpolated excitation sample (Word16).
func inter32_1_3(exc []int16, c int) int16 {
	var s int32
	for i := 0; i < 32; i++ {
		s = lMac0(s, exc[c+i-16], inter32Coef1_3[i])
	}
	s = lAdd(s, s)
	return etsiRound(s)
}

// inter32M1_3 interpolates the -1/3-sample value centred at exc[c], reading
// exc[c-15 .. c+16].
func inter32M1_3(exc []int16, c int) int16 {
	var s int32
	for i := 0; i < 32; i++ {
		s = lMac0(s, exc[c+i-15], inter32CoefM1_3[i])
	}
	s = lAdd(s, s)
	return etsiRound(s)
}

// predLt fills the adaptive-codebook excitation for one subframe: for i in
// [0,subfr), exc[base+i] = the past excitation at delay (T0 - frac/3). frac is
// 0, +1 or -1 (thirds of a sample). The caller guarantees exc has at least
// pitMax+lInter samples of valid history before base. Mirrors Pred_Lt.
func predLt(exc []int16, base int, T0, frac, subfr int) {
	switch frac {
	case 0:
		for i := 0; i < subfr; i++ {
			exc[base+i] = exc[base+i-T0]
		}
	case 1:
		for i := 0; i < subfr; i++ {
			exc[base+i] = inter32_1_3(exc, base+i-T0)
		}
	case -1:
		for i := 0; i < subfr; i++ {
			exc[base+i] = inter32M1_3(exc, base+i-T0)
		}
	}
}

// algebraicPulses returns the four impulse positions encoded in a 14-bit
// algebraic codebook index (D_D4i60 pulse decoding). Positions are in
// [0,subfrLen) and pairwise even/offset per the D4i60 grid.
func algebraicPulses(index int16) (pos0, pos1, pos2, pos3 int) {
	pos0 = int((index & 31) << 1)
	pos1 = int((index&224)>>2) + 2
	pos2 = int((index&1792)>>5) + 4
	pos3 = int((index&14336)>>8) + 6
	return
}

// dD4i60 decodes the 4-pulse algebraic (innovation) codeword and convolves it
// with the noise filter F[] into cod[0:subfrLen]. F must carry 64 zero samples
// of headroom before Fbase (F[Fbase-64 .. Fbase-1] == 0) so the negative pulse
// offsets read valid memory. sign flips the whole codeword; shift is the 0/1
// pulse shift. cod is Q12. Mirrors D_D4i60.
func dD4i60(index, sign, shift int16, F []int16, Fbase int, cod []int16) {
	pos0, pos1, pos2, pos3 := algebraicPulses(index)

	// F -= shift; p0 = F - pos0; ... — all as offsets into F around Fbase.
	b := Fbase - int(shift)
	p0 := b - pos0
	p1 := b - pos1
	p2 := b - pos2
	p3 := b - pos3

	for i := 0; i < subfrLen; i++ {
		L := lMult0(F[p0+i], q11GainI0)
		L = subSh(L, F[p1+i], 11)
		L = addSh(L, F[p2+i], 11)
		L = subSh(L, F[p3+i], 11)
		if sign != 0 {
			L = lNegate(L)
		}
		cod[i] = storeHi(L, 5)
	}
}

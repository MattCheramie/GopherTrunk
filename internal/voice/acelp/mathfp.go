package acelp

// Fixed-point log2 / pow2 / double-precision-split helpers for the TETRA ACELP
// decoder's gain-dequantisation energy arithmetic (EN 300 395-2). log2 and
// pow2 are piecewise-linear table interpolations over a 33-point grid; the
// decoder works in the log-energy domain, so a gain is recovered as
// pow2(energy - reference_energy). L_extract splits a 32-bit value into a
// high/low fixed-point pair.
//
// These are the standard ITU-T fixed-point kernels (the TETRA reference reuses
// them, with log2.tab/pow2.tab identical to the G.729 tables). Clean-room from
// the algorithm (normalise, table lookup, linear interpolation); the 33-entry
// tables are numeric constants (see package doc for provenance).

// tabLog2[i] = round(2^15 * log2(1 + i/32)), i in [0,32].
var tabLog2 = [33]int16{
	0, 1455, 2866, 4236, 5568, 6863, 8124, 9352, 10549, 11716,
	12855, 13967, 15054, 16117, 17156, 18172, 19167, 20142, 21097, 22033,
	22951, 23852, 24735, 25603, 26455, 27291, 28113, 28922, 29716, 30497,
	31266, 32023, 32767,
}

// tabPow2[i] = round(2^14 * 2^(i/32)), i in [0,32].
var tabPow2 = [33]int16{
	16384, 16743, 17109, 17484, 17867, 18258, 18658, 19066, 19484, 19911,
	20347, 20792, 21247, 21713, 22188, 22674, 23170, 23678, 24196, 24726,
	25268, 25821, 26386, 26964, 27554, 28158, 28774, 29405, 30048, 30706,
	31379, 32066, 32767,
}

// log2fp computes log2(L_x) as an integer exponent and a Q15 fraction, so that
// L_x ≈ 2^(exponent + fraction/2^15). Returns (0,0) for L_x <= 0.
func log2fp(x int32) (exponent, fraction int16) {
	if x <= 0 {
		return 0, 0
	}
	exp := normL(x)
	x = lShl(x, exp)
	exponent = subOp(30, exp)

	x = lShr(x, 9)
	i := extractH(x) // integer table index in [32,63]
	x = lShr(x, 1)
	a := extractL(x) & 0x7fff // interpolation fraction
	i = subOp(i, 32)

	L := lDepositH(tabLog2[i])
	tmp := subOp(tabLog2[i], tabLog2[i+1])
	L = lMsu(L, tmp, a)
	fraction = extractH(L)
	return exponent, fraction
}

// pow2fp computes 2^(exponent + fraction/2^15) as a Q0..Q31 value (the shift is
// implied by exponent), inverse of log2fp. Mirrors the standard Pow2.
func pow2fp(exponent, fraction int16) int32 {
	L := lMult(fraction, 32)
	i := extractH(L) // integer table index in [0,31]
	L = lShr(L, 1)
	a := extractL(L) & 0x7fff

	L = lDepositH(tabPow2[i])
	tmp := subOp(tabPow2[i], tabPow2[i+1])
	L = lMsu(L, tmp, a)

	exp := subOp(30, exponent)
	return lShrR(L, exp)
}

// lExtract splits a 32-bit value into a high word and a low word in the
// double-precision format used by the LPC routines: hi = L>>16, and lo carries
// the residual so that L ≈ (hi<<16) + (lo<<1). Mirrors L_Extract.
func lExtract(l int32) (hi, lo int16) {
	hi = extractH(l)
	lo = extractL(lMsu(lShr(l, 1), hi, 16384))
	return hi, lo
}

// lComp reconstructs a 32-bit value from the double-precision (hi,lo) pair,
// inverse of lExtract. Mirrors L_Comp.
func lComp(hi, lo int16) int32 {
	return lMac(lDepositH(hi), lo, 1)
}

// load_sh16 / add_sh16 / sub_sh16 are the shift-by-16 specialisations of the
// shift-accumulate operators (Load_sh16/add_sh16/sub_sh16): they place a 16-bit
// value in the high half of the accumulator.
func loadSh16(v int16) int32           { return lDepositH(v) }
func addSh16(acc int32, v int16) int32 { return lAdd(acc, lDepositH(v)) }
func subSh16(acc int32, v int16) int32 { return lSub(acc, lDepositH(v)) }

package acelp

// Additional fixed-point operators used by the codebook and synthesis blocks:
// the no-shift long multiply/accumulate (L_mult0/L_mac0) and the shift-and-
// accumulate helpers (Load_sh/add_sh/sub_sh/store_hi). Faithful ports of the
// ETSI TETRA reference operators (see package doc for provenance); built from
// the base operators in ops.go so their saturation behaviour matches.

// lMult0 is the 32-bit product without the Q15 left shift (L_mult0). The only
// overflow case, -32768 × -32768 = 0x40000000, fits in Word32, so no saturation
// is applied — matching the reference.
func lMult0(a, b int16) int32 { return int32(a) * int32(b) }

// lMac0 accumulates a no-shift product with saturating 32-bit add (L_mac0).
func lMac0(acc int32, a, b int16) int32 { return lAdd(acc, lMult0(a, b)) }

// loadSh loads a 16-bit value into a 32-bit accumulator shifted left by sh
// (Load_sh).
func loadSh(v int16, sh int16) int32 { return lShl(lDepositL(v), sh) }

// addSh adds v<<sh to the accumulator with saturation (add_sh).
func addSh(acc int32, v, sh int16) int32 { return lAdd(acc, loadSh(v, sh)) }

// subSh subtracts v<<sh from the accumulator with saturation (sub_sh).
func subSh(acc int32, v, sh int16) int32 { return lSub(acc, loadSh(v, sh)) }

// storeHi shifts the accumulator left by sh and returns its high 16 bits
// (store_hi) — the standard way the reference rescales a 32-bit result down to
// a fixed-point Word16.
func storeHi(l int32, sh int16) int16 { return extractH(lShl(l, sh)) }

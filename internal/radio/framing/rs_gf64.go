package framing

import "errors"

// Reed-Solomon codes over GF(2^6) per TIA-102.BAAA-A §5.9 — the
// three shortened RS codes Project 25 uses for voice information
// and link control protection, which P25 Phase 2 also reuses on
// top of the 4-state ½-rate trellis code that protects MAC PDU
// channel dibits.
//
// All three codes share the same Galois field: GF(2^6) generated
// by the primitive characteristic polynomial α^6 + α + 1 = 0.
// Field elements are represented as 6-bit values 0..63 where bit i
// corresponds to α^i; α itself is the value 0b000010 = 2.
//
// The codes are shortened from the natural (63, K, D) form by
// deleting the leftmost (63 - 24) or (63 - 36) information symbols.
// Encoding is systematic: the first K codeword symbols are the K
// information symbols verbatim; the trailing R = N - K symbols are
// parity computed by matrix-multiplying the information vector by
// the right-hand parity columns of the spec's K × N generator
// matrix.
//
//	Code             | K  | N  | D  | t   | Use in P25
//	-----------------+----+----+----+-----+----------------------------
//	RS(24, 12, 13)   | 12 | 24 | 13 | 6   | Link Control words (LCW),
//	                 |    |    |    |     | Phase 2 SACCH outer FEC
//	RS(24, 16,  9)   | 16 | 24 |  9 | 4   | Encryption Sync (ES),
//	                 |    |    |    |     | Phase 2 FACCH outer FEC
//	RS(36, 20, 17)   | 20 | 36 | 17 | 8   | Header Data Unit (HDU)
//
// This file implements the encoder, syndrome verifier, AND a
// bounded-distance error corrector (Berlekamp-Massey for the error
// locator, Chien search for the error positions, Forney for the error
// magnitudes) for all three codes. The corrector cleans up the residual
// symbol errors the inner Hamming/Golay/trellis layer leaves behind —
// notably the P25 Link Control talkgroup and Encryption Sync algorithm
// fields, whose inner-only decode is otherwise corrupted under marginal
// SNR.

// rsGF64 is the singleton GF(2^6) field. Built at init time.
var rsGF64 *rsField64

type rsField64 struct {
	exp [126]byte // exp[i] = α^i for i in 0..62 (and repeated for 63..125 to avoid mod in hot loops)
	log [64]byte  // log[a] = i such that α^i = a (log[0] unused)
}

func init() {
	f := &rsField64{}
	x := byte(1)
	for i := 0; i < 63; i++ {
		f.exp[i] = x
		f.log[x] = byte(i)
		// Multiply by α: shift left, reduce mod α^6 + α + 1 if bit 6 set.
		// The primitive polynomial in bit form is 0b1000011 (= 0x43). If
		// the pre-shift value had bit 5 set the post-shift value has bit 6
		// set, so we XOR with 0b1000011 then mask to 6 bits — equivalent
		// to XOR with the reduction polynomial 0b000011 = 3 after masking.
		preShift := f.exp[i]
		x = byte((int(x) << 1) & 0x3F)
		if preShift&0x20 != 0 {
			x ^= 0x03
		}
	}
	// Double-period table so multiplication can index exp[(la+lb)] without
	// taking mod 63 in every call.
	for i := 63; i < 126; i++ {
		f.exp[i] = f.exp[i-63]
	}
	rsGF64 = f
}

// gf64Mul multiplies two GF(2^6) elements via the log/exp tables.
func gf64Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return rsGF64.exp[int(rsGF64.log[a])+int(rsGF64.log[b])]
}

// gf64Pow returns α^n with n folded into the [0, 62] range.
func gf64Pow(n int) byte {
	n %= 63
	if n < 0 {
		n += 63
	}
	return rsGF64.exp[n]
}

// gf64Inv returns the multiplicative inverse of a in GF(2^6). Since
// α^63 = 1, a^-1 = α^(63 - log a). a must be nonzero.
func gf64Inv(a byte) byte {
	return rsGF64.exp[63-int(rsGF64.log[a])]
}

// rsParity24_12 is the right-hand 12 × 12 parity sub-matrix of the
// GLC generator matrix for the shortened RS(24, 12, 13) code per
// TIA-102.BAAA-A §5.9 (Table "GLC matrix"). Entries are in the
// octal representation the spec uses; each value is a 6-bit GF(2^6)
// element where octal digit d_i contributes bits (3*i+2 .. 3*i) of
// the element.
//
// Indexing: rsParity24_12[i][j] is the parity contribution of the
// i-th information symbol to the j-th parity symbol (j = 0 is the
// leftmost parity column, immediately after the K identity columns).
var rsParity24_12 = [12][12]byte{
	{0o62, 0o44, 0o03, 0o25, 0o14, 0o16, 0o27, 0o03, 0o53, 0o04, 0o36, 0o47},
	{0o11, 0o12, 0o11, 0o11, 0o16, 0o64, 0o67, 0o55, 0o01, 0o76, 0o26, 0o73},
	{0o03, 0o01, 0o05, 0o75, 0o14, 0o06, 0o20, 0o44, 0o66, 0o06, 0o70, 0o66},
	{0o21, 0o70, 0o27, 0o45, 0o16, 0o67, 0o23, 0o64, 0o73, 0o33, 0o44, 0o21},
	{0o30, 0o22, 0o03, 0o75, 0o15, 0o15, 0o33, 0o15, 0o51, 0o03, 0o53, 0o50},
	{0o01, 0o41, 0o27, 0o56, 0o76, 0o64, 0o21, 0o53, 0o04, 0o25, 0o01, 0o12},
	{0o61, 0o76, 0o21, 0o55, 0o76, 0o01, 0o63, 0o35, 0o30, 0o13, 0o64, 0o70},
	{0o24, 0o22, 0o71, 0o56, 0o21, 0o35, 0o73, 0o42, 0o57, 0o74, 0o43, 0o76},
	{0o72, 0o42, 0o05, 0o20, 0o43, 0o47, 0o33, 0o56, 0o01, 0o16, 0o13, 0o76},
	{0o72, 0o14, 0o65, 0o54, 0o35, 0o25, 0o41, 0o16, 0o15, 0o40, 0o71, 0o26},
	{0o73, 0o65, 0o36, 0o61, 0o42, 0o22, 0o17, 0o04, 0o44, 0o20, 0o25, 0o05},
	{0o71, 0o05, 0o55, 0o03, 0o71, 0o34, 0o60, 0o11, 0o74, 0o02, 0o41, 0o50},
}

// rsParity24_16 is the right-hand 16 × 8 parity sub-matrix of the
// GES generator matrix for the shortened RS(24, 16, 9) code per
// TIA-102.BAAA-A §5.9 (Table "GES matrix").
var rsParity24_16 = [16][8]byte{
	{0o51, 0o45, 0o67, 0o15, 0o64, 0o67, 0o52, 0o12},
	{0o57, 0o25, 0o63, 0o73, 0o71, 0o22, 0o40, 0o15},
	{0o05, 0o01, 0o31, 0o04, 0o16, 0o54, 0o25, 0o76},
	{0o73, 0o07, 0o47, 0o14, 0o41, 0o77, 0o47, 0o11},
	{0o75, 0o15, 0o51, 0o51, 0o17, 0o67, 0o17, 0o57},
	{0o20, 0o32, 0o14, 0o42, 0o75, 0o42, 0o70, 0o54},
	{0o02, 0o75, 0o43, 0o05, 0o01, 0o40, 0o12, 0o64},
	{0o24, 0o74, 0o15, 0o72, 0o24, 0o26, 0o74, 0o61},
	{0o42, 0o64, 0o07, 0o22, 0o61, 0o20, 0o40, 0o65},
	{0o32, 0o32, 0o55, 0o41, 0o57, 0o66, 0o21, 0o77},
	{0o65, 0o36, 0o25, 0o07, 0o50, 0o16, 0o40, 0o51},
	{0o64, 0o06, 0o54, 0o32, 0o76, 0o46, 0o14, 0o36},
	{0o62, 0o63, 0o74, 0o70, 0o05, 0o27, 0o37, 0o46},
	{0o55, 0o43, 0o34, 0o71, 0o57, 0o76, 0o50, 0o64},
	{0o24, 0o23, 0o23, 0o05, 0o50, 0o70, 0o42, 0o23},
	{0o67, 0o75, 0o45, 0o60, 0o57, 0o24, 0o06, 0o26},
}

// rsParity36_20 is the right-hand 20 × 16 parity sub-matrix of the
// PHDR generator matrix for the shortened RS(36, 20, 17) code per
// TIA-102.BAAA-A §5.9 (Table "PHDR matrix").
var rsParity36_20 = [20][16]byte{
	{0o74, 0o37, 0o34, 0o06, 0o02, 0o07, 0o44, 0o64, 0o26, 0o14, 0o26, 0o44, 0o54, 0o13, 0o77, 0o05},
	{0o04, 0o17, 0o50, 0o24, 0o11, 0o05, 0o30, 0o57, 0o33, 0o03, 0o02, 0o02, 0o15, 0o16, 0o25, 0o26},
	{0o07, 0o23, 0o37, 0o46, 0o56, 0o75, 0o43, 0o45, 0o55, 0o21, 0o50, 0o31, 0o45, 0o27, 0o71, 0o62},
	{0o26, 0o05, 0o07, 0o63, 0o63, 0o27, 0o63, 0o40, 0o06, 0o04, 0o40, 0o45, 0o47, 0o30, 0o75, 0o07},
	{0o23, 0o73, 0o73, 0o41, 0o72, 0o34, 0o21, 0o51, 0o67, 0o16, 0o31, 0o74, 0o11, 0o21, 0o12, 0o21},
	{0o24, 0o51, 0o25, 0o23, 0o22, 0o41, 0o74, 0o66, 0o74, 0o65, 0o70, 0o36, 0o67, 0o45, 0o64, 0o01},
	{0o52, 0o33, 0o14, 0o02, 0o20, 0o06, 0o14, 0o25, 0o52, 0o23, 0o35, 0o74, 0o75, 0o75, 0o43, 0o27},
	{0o55, 0o62, 0o56, 0o25, 0o73, 0o60, 0o15, 0o30, 0o13, 0o17, 0o20, 0o02, 0o70, 0o55, 0o14, 0o47},
	{0o54, 0o51, 0o32, 0o65, 0o77, 0o12, 0o54, 0o13, 0o35, 0o32, 0o56, 0o12, 0o75, 0o01, 0o72, 0o63},
	{0o74, 0o41, 0o30, 0o41, 0o43, 0o22, 0o51, 0o06, 0o64, 0o33, 0o03, 0o47, 0o27, 0o12, 0o55, 0o47},
	{0o54, 0o70, 0o11, 0o03, 0o13, 0o22, 0o16, 0o57, 0o03, 0o45, 0o72, 0o31, 0o30, 0o56, 0o35, 0o22},
	{0o51, 0o07, 0o72, 0o30, 0o65, 0o54, 0o06, 0o21, 0o36, 0o63, 0o50, 0o61, 0o64, 0o52, 0o01, 0o60},
	{0o01, 0o65, 0o32, 0o70, 0o13, 0o44, 0o73, 0o24, 0o12, 0o52, 0o21, 0o55, 0o12, 0o35, 0o14, 0o72},
	{0o11, 0o70, 0o05, 0o10, 0o65, 0o24, 0o15, 0o77, 0o22, 0o24, 0o24, 0o74, 0o07, 0o44, 0o07, 0o46},
	{0o06, 0o02, 0o65, 0o11, 0o41, 0o20, 0o45, 0o42, 0o46, 0o54, 0o35, 0o12, 0o40, 0o64, 0o65, 0o33},
	{0o34, 0o31, 0o01, 0o15, 0o44, 0o64, 0o16, 0o24, 0o52, 0o16, 0o06, 0o62, 0o20, 0o13, 0o55, 0o57},
	{0o63, 0o43, 0o25, 0o44, 0o77, 0o63, 0o17, 0o17, 0o64, 0o14, 0o40, 0o74, 0o31, 0o72, 0o54, 0o06},
	{0o71, 0o21, 0o70, 0o44, 0o56, 0o04, 0o30, 0o74, 0o04, 0o23, 0o71, 0o70, 0o63, 0o45, 0o56, 0o43},
	{0o02, 0o01, 0o53, 0o74, 0o02, 0o14, 0o52, 0o74, 0o12, 0o57, 0o24, 0o63, 0o15, 0o42, 0o52, 0o33},
	{0o34, 0o35, 0o02, 0o23, 0o21, 0o27, 0o22, 0o33, 0o64, 0o42, 0o05, 0o73, 0o51, 0o46, 0o73, 0o60},
}

// EncodeRS24_12 produces the 24-symbol codeword for 12 information
// symbols using the GLC generator matrix per TIA-102.BAAA-A §5.9.
// The first 12 codeword symbols are the information symbols
// verbatim; the trailing 12 are parity. Each input/output symbol
// is a 6-bit GF(2^6) element (only the low 6 bits are meaningful).
func EncodeRS24_12(info [12]byte) [24]byte {
	var cw [24]byte
	copy(cw[:12], info[:])
	for j := 0; j < 12; j++ {
		var p byte
		for i := 0; i < 12; i++ {
			p ^= gf64Mul(info[i], rsParity24_12[i][j])
		}
		cw[12+j] = p
	}
	return cw
}

// EncodeRS24_16 produces the 24-symbol codeword for 16 information
// symbols using the GES generator matrix per TIA-102.BAAA-A §5.9.
func EncodeRS24_16(info [16]byte) [24]byte {
	var cw [24]byte
	copy(cw[:16], info[:])
	for j := 0; j < 8; j++ {
		var p byte
		for i := 0; i < 16; i++ {
			p ^= gf64Mul(info[i], rsParity24_16[i][j])
		}
		cw[16+j] = p
	}
	return cw
}

// EncodeRS36_20 produces the 36-symbol codeword for 20 information
// symbols using the PHDR generator matrix per TIA-102.BAAA-A §5.9.
func EncodeRS36_20(info [20]byte) [36]byte {
	var cw [36]byte
	copy(cw[:20], info[:])
	for j := 0; j < 16; j++ {
		var p byte
		for i := 0; i < 20; i++ {
			p ^= gf64Mul(info[i], rsParity36_20[i][j])
		}
		cw[20+j] = p
	}
	return cw
}

// VerifyRS24_12 reports whether the supplied 24-symbol codeword
// satisfies the RS(24, 12, 13) parity equations. Returns false if
// len(cw) != 24 or any input symbol has bits outside the GF(2^6)
// range.
//
// The generator polynomial roots are α^1 through α^12; a valid
// codeword evaluates to zero at each of those points. Polynomial
// convention: cw[0] is the highest-degree coefficient (x^23) and
// cw[23] is the constant term, so c(α^j) is computed by Horner
// from the leading coefficient down.
func VerifyRS24_12(cw []byte) bool {
	return verifyRSGF64(cw, 24, 12)
}

// VerifyRS24_16 reports whether the supplied 24-symbol codeword
// satisfies the RS(24, 16, 9) parity equations.
func VerifyRS24_16(cw []byte) bool {
	return verifyRSGF64(cw, 24, 8)
}

// VerifyRS36_20 reports whether the supplied 36-symbol codeword
// satisfies the RS(36, 20, 17) parity equations.
func VerifyRS36_20(cw []byte) bool {
	return verifyRSGF64(cw, 36, 16)
}

func verifyRSGF64(cw []byte, n, r int) bool {
	if len(cw) != n {
		return false
	}
	for _, s := range cw {
		if s&^0x3F != 0 {
			return false
		}
	}
	for _, s := range syndromesGF64(cw, n, r) {
		if s != 0 {
			return false
		}
	}
	return true
}

// ErrRSUncorrectable is returned by the RS decoders when the received
// codeword carries more symbol errors than the code can correct
// (t = (n-k)/2), so the original information cannot be recovered. The
// caller should treat the inner-decoded content as low-confidence.
var ErrRSUncorrectable = errors.New("framing: Reed-Solomon codeword exceeds error-correction radius")

// DecodeRS24_12 corrects up to t=6 symbol errors in a received
// RS(24,12,13) codeword and returns the 12 recovered information
// symbols, the number of symbol errors corrected, and ErrRSUncorrectable
// when the codeword is beyond the correction radius. The codeword
// convention matches EncodeRS24_12 / VerifyRS24_12 (roots α^1..α^12,
// cw[0] highest degree). This protects P25 Link Control words.
func DecodeRS24_12(cw []byte) ([12]byte, int, error) {
	var info [12]byte
	corrected, nErr, err := decodeRSGF64(cw, 24, 12)
	if err != nil {
		return info, 0, err
	}
	copy(info[:], corrected[:12])
	return info, nErr, nil
}

// DecodeRS24_16 corrects up to t=4 symbol errors in a received
// RS(24,16,9) codeword and returns the 16 recovered information symbols.
// This protects P25 Encryption Sync words.
func DecodeRS24_16(cw []byte) ([16]byte, int, error) {
	var info [16]byte
	corrected, nErr, err := decodeRSGF64(cw, 24, 16)
	if err != nil {
		return info, 0, err
	}
	copy(info[:], corrected[:16])
	return info, nErr, nil
}

// DecodeRS36_20 corrects up to t=8 symbol errors in a received
// RS(36,20,17) codeword and returns the 20 recovered information
// symbols. This protects the P25 Header Data Unit.
func DecodeRS36_20(cw []byte) ([20]byte, int, error) {
	var info [20]byte
	corrected, nErr, err := decodeRSGF64(cw, 36, 20)
	if err != nil {
		return info, 0, err
	}
	copy(info[:], corrected[:20])
	return info, nErr, nil
}

// syndromesGF64 returns the r = n-k syndromes S_1..S_r of cw, where
// S_j = c(α^j) evaluated by Horner from the leading coefficient (cw[0]
// is the x^(n-1) term). The slice is indexed 0..r-1 holding S_1..S_r.
func syndromesGF64(cw []byte, n, r int) []byte {
	s := make([]byte, r)
	for j := 1; j <= r; j++ {
		alphaJ := gf64Pow(j)
		var acc byte
		for i := 0; i < n; i++ {
			acc = gf64Mul(acc, alphaJ) ^ cw[i]
		}
		s[j-1] = acc
	}
	return s
}

// decodeRSGF64 performs bounded-distance decoding of a shortened RS code
// over GF(2^6) with generator roots α^1..α^r (r = n-k). It returns the
// corrected n-symbol codeword, the number of symbol errors corrected, and
// ErrRSUncorrectable if the error count exceeds t = r/2. A defensive
// re-syndrome after correction guarantees a returned codeword is always a
// valid member of the code (so a >t error pattern fails cleanly rather
// than silently miscorrecting).
func decodeRSGF64(cw []byte, n, k int) ([]byte, int, error) {
	if len(cw) != n {
		return nil, 0, ErrRSUncorrectable
	}
	for _, s := range cw {
		if s&^0x3F != 0 {
			return nil, 0, ErrRSUncorrectable
		}
	}
	r := n - k
	t := r / 2
	synd := syndromesGF64(cw, n, r)

	allZero := true
	for _, s := range synd {
		if s != 0 {
			allZero = false
			break
		}
	}
	out := append([]byte(nil), cw...)
	if allZero {
		return out, 0, nil
	}

	// Berlekamp-Massey: synthesise the shortest LFSR (error-locator
	// polynomial Λ) that generates the syndrome sequence.
	lambda := []byte{1}
	bPoly := []byte{1}
	L := 0
	m := 1
	b := byte(1)
	for i := 0; i < r; i++ {
		// Discrepancy delta = S_{i+1} + Σ_{j=1..L} Λ_j · S_{i+1-j}.
		delta := synd[i]
		for j := 1; j <= L; j++ {
			if j < len(lambda) {
				delta ^= gf64Mul(lambda[j], synd[i-j])
			}
		}
		switch {
		case delta == 0:
			m++
		case 2*L <= i:
			tmp := append([]byte(nil), lambda...)
			coef := gf64Mul(delta, gf64Inv(b))
			lambda = polyAddShiftScaledGF64(lambda, bPoly, m, coef)
			L = i + 1 - L
			bPoly = tmp
			b = delta
			m = 1
		default:
			coef := gf64Mul(delta, gf64Inv(b))
			lambda = polyAddShiftScaledGF64(lambda, bPoly, m, coef)
			m++
		}
	}

	if L > t {
		return nil, 0, ErrRSUncorrectable
	}

	// Chien search: a symbol at codeword index i (polynomial degree
	// p = n-1-i) is in error iff Λ(α^-p) = 0. Collect roots → positions.
	type errPos struct {
		idx int  // codeword index
		x   byte // error locator X_l = α^p
	}
	var positions []errPos
	for p := 0; p < n; p++ {
		root := gf64Pow(-p) // α^-p
		if polyEvalGF64(lambda, root) == 0 {
			positions = append(positions, errPos{idx: n - 1 - p, x: gf64Pow(p)})
		}
	}
	if len(positions) != L {
		// Roots fell outside the transmitted positions, or fewer roots
		// than the locator degree → more than t errors.
		return nil, 0, ErrRSUncorrectable
	}

	// Forney: error magnitude e_l = Ω(X_l^-1) / Λ'(X_l^-1) for roots
	// starting at α^1 (b=1, so the X_l^(1-b) factor is unity). Ω is the
	// error-evaluator Ω(x) = S(x)·Λ(x) mod x^r.
	omega := polyMulModGF64(syndromePolyGF64(synd), lambda, r)
	lambdaPrime := formalDerivativeGF64(lambda)
	for _, ep := range positions {
		xInv := gf64Inv(ep.x)
		num := polyEvalGF64(omega, xInv)
		den := polyEvalGF64(lambdaPrime, xInv)
		if den == 0 {
			return nil, 0, ErrRSUncorrectable
		}
		out[ep.idx] ^= gf64Mul(num, gf64Inv(den))
	}

	// Defensive re-syndrome: a genuine >t error pattern that happened to
	// produce a degree-≤t locator is rejected here rather than returned
	// as a plausible-but-wrong codeword.
	for _, s := range syndromesGF64(out, n, r) {
		if s != 0 {
			return nil, 0, ErrRSUncorrectable
		}
	}
	return out, len(positions), nil
}

// syndromePolyGF64 builds S(x) = Σ_{j=1..r} S_j x^(j-1) from the syndrome
// slice (synd[j-1] = S_j).
func syndromePolyGF64(synd []byte) []byte {
	return append([]byte(nil), synd...)
}

// polyAddShiftScaledGF64 returns a(x) + coef·x^shift·b(x) over GF(2^6).
func polyAddShiftScaledGF64(a, b []byte, shift int, coef byte) []byte {
	size := len(a)
	if len(b)+shift > size {
		size = len(b) + shift
	}
	out := make([]byte, size)
	copy(out, a)
	for i, bi := range b {
		out[i+shift] ^= gf64Mul(coef, bi)
	}
	return out
}

// polyEvalGF64 evaluates poly (poly[i] = coefficient of x^i) at x.
func polyEvalGF64(poly []byte, x byte) byte {
	var acc byte
	var xp byte = 1
	for _, c := range poly {
		acc ^= gf64Mul(c, xp)
		xp = gf64Mul(xp, x)
	}
	return acc
}

// polyMulModGF64 returns a(x)·b(x) mod x^deg over GF(2^6).
func polyMulModGF64(a, b []byte, deg int) []byte {
	out := make([]byte, deg)
	for i, ai := range a {
		if ai == 0 || i >= deg {
			continue
		}
		for j, bj := range b {
			if i+j >= deg {
				break
			}
			out[i+j] ^= gf64Mul(ai, bj)
		}
	}
	return out
}

// formalDerivativeGF64 returns Λ'(x): over GF(2^m) the even-degree terms
// vanish (multiplied by an even integer ≡ 0), leaving the odd-degree
// coefficients shifted down by one.
func formalDerivativeGF64(poly []byte) []byte {
	if len(poly) <= 1 {
		return []byte{0}
	}
	out := make([]byte, len(poly)-1)
	for i := 1; i < len(poly); i++ {
		if i%2 == 1 {
			out[i-1] = poly[i]
		}
	}
	return out
}

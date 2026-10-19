package framing

// RS(63,35) over GF(2^6) with errors-and-erasures decoding: the outer code of
// P25 Phase 2 TDMA signalling (FACCH, SACCH, LCCH) and of the voice bursts'
// ESS. Same field (α^6 + α + 1), same generator roots α^1..α^28 and the same
// symbol order (cw[0] is the x^62 coefficient) as the shortened Phase 1 codes
// above; OP25 decodes it with ezpwd::RS<63,35> (poly 0x43, FCR 1, PRIM 1),
// whose codewords this decoder accepts unchanged (pinned by literal ezpwd
// output in rs63_35_test.go).
//
// Phase 2 transmits the code shortened and punctured: the leading data
// symbols are zero and never sent, the trailing parity symbols are not sent
// either. The caller zero-fills the shortened positions and names the
// punctured ones as erasures.

const (
	// RS6335N and RS6335K are the mother code's length and dimension.
	RS6335N = 63
	RS6335K = 35
)

// DecodeRS63_35 corrects cw (63 hexbits) given the indexes of erased
// symbols. It returns the corrected codeword and how many received symbols
// were in error (filling in an erasure is not counted). It fails with
// ErrRSUncorrectable when 2·errors + erasures exceeds the 28 parity symbols
// or the result is not a codeword.
func DecodeRS63_35(cw []byte, erasures []int) ([]byte, int, error) {
	return decodeRSGF64Erasures(cw, RS6335N, RS6335K, erasures)
}

// EncodeRS63_35 appends the 28 parity hexbits to 35 data hexbits.
func EncodeRS63_35(data []byte) ([]byte, error) {
	if len(data) != RS6335K {
		return nil, ErrRSUncorrectable
	}
	r := RS6335N - RS6335K
	// Generator g(x) = Π_{j=1..r} (x + α^j), coefficients highest degree first.
	g := []byte{1}
	for j := 1; j <= r; j++ {
		root := gf64Pow(j)
		next := make([]byte, len(g)+1)
		for i, c := range g {
			next[i] ^= c
			next[i+1] ^= gf64Mul(c, root)
		}
		g = next
	}
	// Systematic encode: remainder of data(x)·x^r divided by g(x).
	rem := make([]byte, r)
	for _, d := range data {
		if d&^0x3F != 0 {
			return nil, ErrRSUncorrectable
		}
		fb := d ^ rem[0]
		copy(rem, rem[1:])
		rem[r-1] = 0
		if fb != 0 {
			for i := 0; i < r; i++ {
				rem[i] ^= gf64Mul(fb, g[i+1])
			}
		}
	}
	return append(append([]byte(nil), data...), rem...), nil
}

// decodeRSGF64Erasures is decodeRSGF64 with erasures, by the Forney-syndrome
// method: the erasure locator Γ is folded into the syndromes, Berlekamp-Massey
// finds the error locator σ from the folded sequence, and the Chien search and
// Forney algorithm run over the full errata locator Λ = σ·Γ.
func decodeRSGF64Erasures(cw []byte, n, k int, erasures []int) ([]byte, int, error) {
	if len(cw) != n {
		return nil, 0, ErrRSUncorrectable
	}
	for _, s := range cw {
		if s&^0x3F != 0 {
			return nil, 0, ErrRSUncorrectable
		}
	}
	r := n - k
	e := len(erasures)
	if e > r {
		return nil, 0, ErrRSUncorrectable
	}
	synd := syndromesGF64(cw, n, r)
	out := append([]byte(nil), cw...)
	allZero := true
	for _, s := range synd {
		if s != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return out, 0, nil
	}

	// Erasure locator Γ(x) = Π (1 + X_k x), X_k = α^(n-1-pos).
	gamma := []byte{1}
	seen := make(map[int]bool, e)
	for _, pos := range erasures {
		if pos < 0 || pos >= n || seen[pos] {
			return nil, 0, ErrRSUncorrectable
		}
		seen[pos] = true
		gamma = polyAddShiftScaledGF64(gamma, gamma, 1, gf64Pow(n-1-pos))
	}

	// Forney syndromes: Ξ(x) = S(x)·Γ(x) mod x^r; its coefficients e..r-1 are
	// free of the erasures and drive Berlekamp-Massey.
	xi := polyMulModGF64(syndromePolyGF64(synd), gamma, r)
	seq := xi[e:]
	sigma := []byte{1}
	bPoly := []byte{1}
	L, m := 0, 1
	b := byte(1)
	for i := range seq {
		delta := seq[i]
		for j := 1; j <= L && j < len(sigma); j++ {
			delta ^= gf64Mul(sigma[j], seq[i-j])
		}
		switch {
		case delta == 0:
			m++
		case 2*L <= i:
			tmp := append([]byte(nil), sigma...)
			sigma = polyAddShiftScaledGF64(sigma, bPoly, m, gf64Mul(delta, gf64Inv(b)))
			L = i + 1 - L
			bPoly, b, m = tmp, delta, 1
		default:
			sigma = polyAddShiftScaledGF64(sigma, bPoly, m, gf64Mul(delta, gf64Inv(b)))
			m++
		}
	}
	if 2*L+e > r {
		return nil, 0, ErrRSUncorrectable
	}

	// Errata locator Λ = σ·Γ.
	lambda := make([]byte, len(sigma)+len(gamma)-1)
	for i, si := range sigma {
		for j, gj := range gamma {
			lambda[i+j] ^= gf64Mul(si, gj)
		}
	}
	type pos struct {
		idx int
		x   byte
	}
	var roots []pos
	for p := 0; p < n; p++ {
		if polyEvalGF64(lambda, gf64Pow(-p)) == 0 {
			roots = append(roots, pos{idx: n - 1 - p, x: gf64Pow(p)})
		}
	}
	if len(roots) != L+e {
		return nil, 0, ErrRSUncorrectable
	}
	omega := polyMulModGF64(syndromePolyGF64(synd), lambda, r)
	lambdaPrime := formalDerivativeGF64(lambda)
	changed := 0
	for _, rt := range roots {
		xInv := gf64Inv(rt.x)
		den := polyEvalGF64(lambdaPrime, xInv)
		if den == 0 {
			return nil, 0, ErrRSUncorrectable
		}
		if v := gf64Mul(polyEvalGF64(omega, xInv), gf64Inv(den)); v != 0 {
			out[rt.idx] ^= v
			if !seen[rt.idx] {
				changed++
			}
		}
	}
	for _, s := range syndromesGF64(out, n, r) {
		if s != 0 {
			return nil, 0, ErrRSUncorrectable
		}
	}
	return out, changed, nil
}

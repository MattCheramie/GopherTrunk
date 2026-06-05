package framing

// Algebraic decoder for the BCH(63,16,11) NID code: syndrome
// computation → Berlekamp-Massey → Chien search. BCH(63,16) is a binary
// code, so every error magnitude is 1 — Chien yields the error bit
// positions directly and no Forney step is needed. This replaces the
// O(2^16) minimum-distance search that BCHDecode63_16 used to run across
// searchNID's full alignment grid on every FSW hit (issue #492): the
// algebraic path decodes — and rejects uncorrectable words — in O(t^2)
// Galois-field operations rather than a 65,536-codeword table walk.
//
// The field is the shared GF(2^6) singleton in rs_gf64.go (primitive
// polynomial α^6 + α + 1, element 2 = α) — the same field the BCH
// generator is defined over — via gf64Mul / gf64Pow / gf64Inv.
//
// Conventions. The codeword is packed with bit p (p = 0..62) carrying
// the coefficient of x^p, matching how callers build it (nid.go). The
// syndromes are S_j = c(α^j); the generator's roots are the minimal
// polynomials of α^1, α^3, …, α^21, whose binary conjugates fill in the
// even powers, so the code has the 2t = 22 consecutive roots α^1..α^22.
// Berlekamp-Massey on S_1..S_22 yields the error-locator Λ(x) =
// Π_l (1 − α^{p_l} x); its roots are α^{−p_l}, so a Chien evaluation of
// Λ(α^{−p}) = 0 marks an error at bit p.

const bch6316T = 11              // correction capacity
const bch6316Synd = 2 * bch6316T // 22 consecutive syndromes S_1..S_22

// bchSyndromes6316 returns S_j = c(α^j) for j = 1..22 in synd[1..22]
// (synd[0] is unused) and whether every syndrome is zero (a valid
// codeword). cw must already be masked to its low 63 bits.
func bchSyndromes6316(cw uint64) (synd [bch6316Synd + 1]byte, allZero bool) {
	allZero = true
	for j := 1; j <= bch6316Synd; j++ {
		aj := gf64Pow(j)
		// Horner over coefficients x^62 .. x^0.
		var s byte
		for p := 62; p >= 0; p-- {
			s = gf64Mul(s, aj)
			if cw&(uint64(1)<<uint(p)) != 0 {
				s ^= 1
			}
		}
		synd[j] = s
		if s != 0 {
			allZero = false
		}
	}
	return synd, allZero
}

// bchBerlekampMassey runs the Berlekamp-Massey algorithm over the 22
// syndromes and returns the error-locator polynomial Λ (lambda[0] = 1)
// and its degree L = number of errors. A returned L > bch6316T means the
// word is uncorrectable.
func bchBerlekampMassey(synd [bch6316Synd + 1]byte) (lambda []byte, L int) {
	// Generous fixed sizes avoid bounds juggling; the meaningful degree
	// is at most bch6316T, and the working shift index i+m stays well
	// under this length for any 22-syndrome run.
	const sz = 2*bch6316Synd + 2
	C := make([]byte, sz) // current locator Λ(x)
	B := make([]byte, sz) // last locator before the previous length change
	C[0], B[0] = 1, 1
	b := byte(1) // discrepancy at that previous length change
	m := 1       // steps since that change

	for n := 0; n < bch6316Synd; n++ {
		// Discrepancy δ = S_{n+1} + Σ_{i=1}^{L} C_i · S_{n+1-i}.
		d := synd[n+1]
		for i := 1; i <= L; i++ {
			d ^= gf64Mul(C[i], synd[n+1-i])
		}
		if d == 0 {
			m++
			continue
		}
		coef := gf64Mul(d, gf64Inv(b))
		if 2*L <= n {
			T := append([]byte(nil), C...)
			for i := 0; i < sz-m; i++ {
				if B[i] != 0 {
					C[i+m] ^= gf64Mul(coef, B[i])
				}
			}
			L = n + 1 - L
			B = T
			b = d
			m = 1
		} else {
			for i := 0; i < sz-m; i++ {
				if B[i] != 0 {
					C[i+m] ^= gf64Mul(coef, B[i])
				}
			}
			m++
		}
	}
	return C, L
}

// bchChienSearch returns the bit positions p (0..62) at which
// Λ(α^{-p}) = 0, i.e. the error locations. lambda[0..L] are the locator
// coefficients.
func bchChienSearch(lambda []byte, L int) []int {
	var pos []int
	for p := 0; p < 63; p++ {
		// Evaluate Λ(α^{-p}) = Σ_i Λ_i · α^{-p·i}.
		var sum byte
		for i := 0; i <= L; i++ {
			if lambda[i] != 0 {
				sum ^= gf64Mul(lambda[i], gf64Pow(-p*i))
			}
		}
		if sum == 0 {
			pos = append(pos, p)
		}
	}
	return pos
}

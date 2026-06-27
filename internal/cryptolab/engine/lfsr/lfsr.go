// Package lfsr provides stream-cipher / LFSR analysis primitives relevant
// to RF scramblers and keystream study: Berlekamp–Massey recovery of the
// shortest LFSR that produces an observed bit sequence, Fibonacci-LFSR
// generation, and known-plaintext keystream extraction.
//
// All bit sequences are represented as []byte holding 0/1 values (one bit
// per element), which keeps the algorithms readable; BitsFromBytes/
// BytesToBits convert to and from packed bytes (MSB-first).
package lfsr

import "fmt"

// BitsFromBytes expands packed bytes into a 0/1 bit slice, MSB first.
func BitsFromBytes(data []byte) []byte {
	out := make([]byte, 0, len(data)*8)
	for _, b := range data {
		for i := 7; i >= 0; i-- {
			out = append(out, (b>>uint(i))&1)
		}
	}
	return out
}

// BitsToBytes packs a 0/1 bit slice into bytes, MSB first. Trailing bits
// that do not fill a byte are left-aligned (zero-padded on the right).
func BitsToBytes(bits []byte) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, bit := range bits {
		if bit&1 == 1 {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

// Result is the outcome of Berlekamp–Massey: the linear complexity L (the
// length of the shortest LFSR) and the connection polynomial C(x) =
// 1 + c1·x + ... + cL·x^L as coefficients [1, c1, ..., cL] over GF(2).
type Result struct {
	LinearComplexity int
	Polynomial       []byte // length L+1, Polynomial[0] == 1
}

// Taps returns the feedback tap positions (1-based powers of x with a
// non-zero coefficient), i.e. the indices into the connection polynomial
// excluding the constant term.
func (r Result) Taps() []int {
	var taps []int
	for i := 1; i < len(r.Polynomial); i++ {
		if r.Polynomial[i] == 1 {
			taps = append(taps, i)
		}
	}
	return taps
}

// BerlekampMassey returns the shortest LFSR (over GF(2)) consistent with the
// bit sequence s (each element 0 or 1). The linear complexity is a strong
// fingerprint: a short LFSR means a simple scrambler; complexity ≈ len(s)/2
// means no short linear generator (the sequence looks random to a linear
// model).
func BerlekampMassey(s []byte) Result {
	n := len(s)
	c := make([]byte, n+1)
	b := make([]byte, n+1)
	c[0], b[0] = 1, 1
	var l, m int = 0, 1

	for i := 0; i < n; i++ {
		// discrepancy d = s[i] + sum_{j=1..L} c[j]*s[i-j]  (mod 2)
		d := s[i] & 1
		for j := 1; j <= l; j++ {
			d ^= c[j] & s[i-j]
		}
		if d == 0 {
			m++
			continue
		}
		t := make([]byte, n+1)
		copy(t, c)
		// c(x) <- c(x) - x^m * b(x)  (XOR over GF(2))
		for j := 0; j+m <= n; j++ {
			c[j+m] ^= b[j]
		}
		if 2*l <= i {
			l = i + 1 - l
			copy(b, t)
			m = 1
		} else {
			m++
		}
	}

	poly := make([]byte, l+1)
	copy(poly, c[:l+1])
	return Result{LinearComplexity: l, Polynomial: poly}
}

// Generate produces n output bits from a Fibonacci LFSR with the given
// 1-based tap positions and an initial state seed (seed[0] is the first bit
// shifted out). The register length is the max tap. Output bit = the bit
// leaving the register; feedback = XOR of the tapped stages. seed length
// must be at least the register length.
func Generate(taps []int, seed []byte, n int) ([]byte, error) {
	deg := 0
	for _, t := range taps {
		if t > deg {
			deg = t
		}
		if t < 1 {
			return nil, fmt.Errorf("lfsr: tap %d out of range", t)
		}
	}
	if deg == 0 {
		return nil, fmt.Errorf("lfsr: no taps")
	}
	if len(seed) < deg {
		return nil, fmt.Errorf("lfsr: seed needs >= %d bits, got %d", deg, len(seed))
	}
	state := make([]byte, deg)
	copy(state, seed[:deg])

	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = state[deg-1] & 1 // bit shifting out the high end
		var fb byte
		for _, t := range taps {
			fb ^= state[t-1] & 1
		}
		// shift towards the high end; inject feedback at stage 0
		copy(state[1:], state[:deg-1])
		state[0] = fb
	}
	return out, nil
}

// Keystream returns plaintext XOR ciphertext over the common prefix — the
// keystream of an additive stream cipher under known plaintext. The result
// length is min(len(pt), len(ct)).
func Keystream(pt, ct []byte) []byte {
	n := len(pt)
	if len(ct) < n {
		n = len(ct)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = pt[i] ^ ct[i]
	}
	return out
}

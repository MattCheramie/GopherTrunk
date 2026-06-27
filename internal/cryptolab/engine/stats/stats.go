// Package stats holds the cipher-agnostic statistical primitives the
// toolkit uses to triage a captured payload: is it plaintext, a weak
// classical cipher, an obfuscation, or strong encryption? Plus the
// repeating-key-XOR key-length estimators (Friedman/Hamming) and a
// crib-dragging helper for many-time-pad / keystream-reuse analysis.
package stats

import (
	"math"
	"math/bits"
	"sort"
)

// ShannonEntropy returns the per-byte Shannon entropy in bits (0..8). High
// values (~8) indicate compression or strong encryption; low values
// indicate structure (plaintext, weak ciphers, obfuscation).
func ShannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	var counts [256]int
	for _, b := range data {
		counts[b]++
	}
	n := float64(len(data))
	var h float64
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h
}

// IndexOfCoincidence returns the probability that two bytes drawn at random
// (without replacement) are equal. English text over the 26-letter
// alphabet sits near 0.066; uniform random over 256 values near 1/256.
func IndexOfCoincidence(data []byte) float64 {
	n := len(data)
	if n < 2 {
		return 0
	}
	var counts [256]int
	for _, b := range data {
		counts[b]++
	}
	var sum float64
	for _, c := range counts {
		sum += float64(c) * float64(c-1)
	}
	return sum / (float64(n) * float64(n-1))
}

// ByteFrequency returns the count of each byte value.
func ByteFrequency(data []byte) [256]int {
	var counts [256]int
	for _, b := range data {
		counts[b]++
	}
	return counts
}

// ChiSquareUniform returns the chi-square statistic of the byte
// distribution against a uniform 256-value model. Near (256-1)=255 for
// uniform/random data; large for structured data.
func ChiSquareUniform(data []byte) float64 {
	n := len(data)
	if n == 0 {
		return 0
	}
	counts := ByteFrequency(data)
	expected := float64(n) / 256.0
	var chi float64
	for _, c := range counts {
		d := float64(c) - expected
		chi += d * d / expected
	}
	return chi
}

// HammingDistance returns the bit-difference count between equal-length
// byte slices. It panics if the lengths differ — callers slice first.
func HammingDistance(a, b []byte) int {
	if len(a) != len(b) {
		panic("stats: HammingDistance on unequal lengths")
	}
	var d int
	for i := range a {
		d += bits.OnesCount8(a[i] ^ b[i])
	}
	return d
}

// KeyLenScore pairs a candidate repeating-XOR key length with its
// normalized average Hamming distance (bits/byte); the true length tends to
// minimize it.
type KeyLenScore struct {
	Length int
	Score  float64
}

// GuessXORKeyLength ranks candidate key lengths for a repeating-key XOR
// ciphertext by averaging the normalized Hamming distance between adjacent
// key-length blocks (the Friedman/Kasiski idea). Lower score = more likely.
// Results are sorted best-first.
func GuessXORKeyLength(ct []byte, maxLen int) []KeyLenScore {
	if maxLen < 1 {
		maxLen = 40
	}
	var out []KeyLenScore
	for k := 1; k <= maxLen; k++ {
		// Need at least a few blocks to average over.
		blocks := len(ct) / k
		if blocks < 2 {
			break
		}
		pairs := blocks - 1
		if pairs > 8 {
			pairs = 8 // averaging the first several pairs is enough and stable
		}
		var total float64
		for p := 0; p < pairs; p++ {
			a := ct[p*k : (p+1)*k]
			b := ct[(p+1)*k : (p+2)*k]
			total += float64(HammingDistance(a, b)) / float64(k)
		}
		out = append(out, KeyLenScore{Length: k, Score: total / float64(pairs)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score < out[j].Score })
	return out
}

// DragHit is one crib-drag position and the plaintext fragment it reveals
// in the other message when two ciphertexts share a keystream (c1 ^ c2 ^
// crib).
type DragHit struct {
	Offset    int
	Revealed  []byte
	Printable bool
}

// CribDrag slides crib across a^b (where a and b are two ciphertexts that
// reused the same keystream, so a^b = p_a ^ p_b) and reports, at each
// offset, the bytes the crib reveals in the partner plaintext. Offsets that
// reveal all-printable fragments are flagged — those are likely real.
func CribDrag(a, b, crib []byte) []DragHit {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if len(crib) == 0 || len(crib) > n {
		return nil
	}
	xored := make([]byte, n)
	for i := 0; i < n; i++ {
		xored[i] = a[i] ^ b[i]
	}
	var out []DragHit
	for off := 0; off+len(crib) <= n; off++ {
		rev := make([]byte, len(crib))
		printable := true
		for i := range crib {
			rev[i] = xored[off+i] ^ crib[i]
			if rev[i] < 0x20 || rev[i] > 0x7e {
				printable = false
			}
		}
		out = append(out, DragHit{Offset: off, Revealed: rev, Printable: printable})
	}
	return out
}

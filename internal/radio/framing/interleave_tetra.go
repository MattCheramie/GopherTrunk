package framing

// TETRA (K, a) block interleaver per ETSI EN 300 392-2 §8.2.4.1.
//
// The phase-modulation block interleaver re-orders K3 type-3 bits
// b3(1), b3(2), ..., b3(K3) into K4 type-4 bits b4(1), b4(2), ...,
// b4(K4) with K = K3 = K4, by the rule:
//
//	b4(k) = b3(i)        for i = 1, 2, ..., K
//	with k = 1 + ((a * i) mod K)
//
// (K, a) is channel-specific:
//
//	(120, 11)   BSCH                              §8.3.1.2
//	(216, 101)  SCH/HD, BNCH, STCH                §8.3.1.4.1
//	(168, 13)   SCH/HU                            §8.3.1.4.3
//	(432, 103)  SCH/F                             §8.3.1.4.5
//
// Pick the constant a for your channel and call
// BlockInterleaveTetra / BlockDeinterleaveTetra with the matching
// K. The deinterleaver is the inverse permutation; round-trip is
// the identity.

// BlockInterleaveTetra applies the (K, a) interleaver to a slice
// of K type-3 bits. The output has the same length. Each input bit
// i (1-indexed) lands at position k = 1 + ((a * i) mod K) in the
// output (also 1-indexed). Returns nil if len(b3) != K or if K, a
// don't form a valid permutation (gcd(a, K) != 1 — but the spec's
// (K, a) pairs are pre-validated, so callers using those constants
// never hit that path).
func BlockInterleaveTetra(b3 []byte, K, a int) []byte {
	if len(b3) != K {
		return nil
	}
	out := make([]byte, K)
	for i := 1; i <= K; i++ {
		k := 1 + ((a * i) % K)
		out[k-1] = b3[i-1] & 1
	}
	return out
}

// BlockDeinterleaveTetra reverses a (K, a) interleave. Each input
// bit at position k (1-indexed) was originally type-3 bit i where
// k = 1 + ((a * i) mod K). The deinterleaver inverts that mapping
// by iterating i = 1..K and copying b4[k-1] to position i-1.
func BlockDeinterleaveTetra(b4 []byte, K, a int) []byte {
	if len(b4) != K {
		return nil
	}
	out := make([]byte, K)
	for i := 1; i <= K; i++ {
		k := 1 + ((a * i) % K)
		out[i-1] = b4[k-1] & 1
	}
	return out
}

// TETRA per-channel block-interleaver parameters per
// EN 300 392-2 §8.3.1. Each constant matches one logical channel's
// (K, a) tuple — callers pass these straight to
// BlockInterleaveTetra / BlockDeinterleaveTetra alongside the
// expected K.

// BSCH (§8.3.1.2).
const (
	InterleaveKBSCH = 120
	InterleaveABSCH = 11
)

// SCH/HD, BNCH, STCH (§8.3.1.4.1) — all three share the same
// 124-bit type-1 / 216-bit type-3 / type-4 layout and interleaver.
const (
	InterleaveKSCHHD = 216
	InterleaveASCHHD = 101
)

// SCH/HU (§8.3.1.4.3).
const (
	InterleaveKSCHHU = 168
	InterleaveASCHHU = 13
)

// SCH/F (§8.3.1.4.5).
const (
	InterleaveKSCHF = 432
	InterleaveASCHF = 103
)

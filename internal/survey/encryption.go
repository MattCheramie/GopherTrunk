package survey

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/stats"
)

// minEncryptBytes is the shortest recovered payload worth an entropy verdict.
// Below it the Shannon estimate is too noisy to separate random from structured.
const minEncryptBytes = 64

// EncryptionTriage reports whether a recovered byte payload looks encrypted,
// with a short label. It is a heuristic, not a decode: a strong cipher's output
// is statistically indistinguishable from random, so a payload with near-maximal
// (sample-size-normalized) Shannon entropy AND a near-uniform byte distribution
// (low index of coincidence) reads as encrypted. Structured traffic — plaintext,
// a repeating scrambler, a low-rate codec — falls below the bar and is not
// flagged. It is the survey's "is this flow encrypted?" for a carrier no decoder
// identified, mirroring the rfscope entropy analyzer; CryptoLab is where the
// verdict is confirmed and the construction attacked.
func EncryptionTriage(data []byte) (encrypted bool, label string) {
	if len(data) < minEncryptBytes {
		return false, ""
	}
	// Measured Shannon entropy is capped at log2(#samples); normalize by the
	// achievable maximum so a short-but-random payload still scores near 1.
	span := len(data)
	if span > 256 {
		span = 256
	}
	maxH := math.Log2(float64(span))
	if maxH <= 0 {
		return false, ""
	}
	// Normalized entropy is a loose floor: a finite random sample underestimates
	// the full 8 bits/byte (Miller bias), so the byte-uniformity test below,
	// which is sample-size robust, does the real work.
	norm := stats.ShannonEntropy(data) / maxH

	// Random/encrypted bytes are near-uniform: IC ≈ 1/256 ≈ 0.0039. English text
	// sits ~0.066; a repeating scrambler or a low-rate codec is far higher. This
	// cleanly separates random-looking (encrypted) from structured payloads.
	ic := stats.IndexOfCoincidence(data)

	if norm >= 0.85 && ic < 0.02 {
		return true, "high-entropy (likely encrypted)"
	}
	return false, ""
}

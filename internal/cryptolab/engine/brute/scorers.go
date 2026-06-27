package brute

import (
	"bytes"
	"math"
)

// Printable scores the fraction of bytes that are printable ASCII or common
// whitespace. Cheap and language-agnostic — a good first filter.
type Printable struct{}

func (Printable) Score(pt []byte) float64 {
	if len(pt) == 0 {
		return 0
	}
	var good int
	for _, b := range pt {
		if (b >= 0x20 && b <= 0x7e) || b == '\n' || b == '\r' || b == '\t' {
			good++
		}
	}
	return float64(good) / float64(len(pt))
}

// englishFreq holds approximate English letter frequencies (plus space),
// used by the log-likelihood English scorer.
var englishFreq = map[byte]float64{
	' ': 0.182, 'e': 0.103, 't': 0.075, 'a': 0.065, 'o': 0.062, 'i': 0.057,
	'n': 0.057, 's': 0.053, 'r': 0.050, 'h': 0.050, 'l': 0.033, 'd': 0.032,
	'u': 0.023, 'c': 0.022, 'm': 0.020, 'f': 0.020, 'w': 0.018, 'g': 0.016,
	'y': 0.016, 'p': 0.015, 'b': 0.013, 'v': 0.008, 'k': 0.006, 'x': 0.002,
	'j': 0.001, 'q': 0.001, 'z': 0.001,
}

// English scores a plaintext by average per-byte log-likelihood under an
// English letter-frequency model (case-folded). Non-letters that are
// printable get a small floor; non-printable bytes are penalized heavily,
// so gibberish keys score far below readable text.
type English struct{}

func (English) Score(pt []byte) float64 {
	if len(pt) == 0 {
		return math.Inf(-1)
	}
	const floor = 1e-5
	var sum float64
	for _, b := range pt {
		c := b
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if f, ok := englishFreq[c]; ok {
			sum += math.Log(f)
		} else if b >= 0x20 && b <= 0x7e {
			sum += math.Log(floor)
		} else {
			sum += math.Log(floor) * 5 // heavy penalty for control bytes
		}
	}
	return sum / float64(len(pt))
}

// Crib rewards plaintexts containing a known substring (known-plaintext).
// It composes a base scorer with a large bonus when the crib appears.
type Crib struct {
	Crib []byte
	Base Scorer
}

func (c Crib) Score(pt []byte) float64 {
	var base float64
	if c.Base != nil {
		base = c.Base.Score(pt)
	}
	if len(c.Crib) > 0 && bytes.Contains(pt, c.Crib) {
		return base + 1000
	}
	return base
}

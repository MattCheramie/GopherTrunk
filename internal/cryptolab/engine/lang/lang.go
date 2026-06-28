// Package lang holds a small embedded English language model used to score
// candidate plaintexts during cipher solving (e.g. the monoalphabetic
// substitution hill-climb). The model is built at init from an embedded
// public-domain English corpus (corpus.txt) — no network, no external data.
package lang

import (
	_ "embed"
	"math"
	"sort"
	"sync"
)

//go:embed corpus.txt
var corpus string

// Letters maps a byte to a 0..25 letter index, or -1 for non-letters
// (case-folded).
func LetterIndex(b byte) int {
	switch {
	case b >= 'A' && b <= 'Z':
		return int(b - 'A')
	case b >= 'a' && b <= 'z':
		return int(b - 'a')
	default:
		return -1
	}
}

// Clean returns the A..Z letter indices of s (case-folded, non-letters dropped).
func Clean(s []byte) []int {
	out := make([]int, 0, len(s))
	for _, b := range s {
		if i := LetterIndex(b); i >= 0 {
			out = append(out, i)
		}
	}
	return out
}

// Model is an n-gram log-probability scorer over the 26-letter alphabet.
type Model struct {
	n     int
	logp  map[uint32]float64
	floor float64
}

func ngramKey(idx []int) uint32 {
	var k uint32
	for _, v := range idx {
		k = k*26 + uint32(v)
	}
	return k
}

// NewModel builds an n-gram model from the embedded corpus.
func NewModel(n int) *Model {
	idx := Clean([]byte(corpus))
	counts := map[uint32]int{}
	total := 0
	for i := 0; i+n <= len(idx); i++ {
		counts[ngramKey(idx[i:i+n])]++
		total++
	}
	m := &Model{n: n, logp: make(map[uint32]float64, len(counts))}
	for k, c := range counts {
		m.logp[k] = math.Log10(float64(c) / float64(total))
	}
	// Unseen n-grams get a small floor probability (Practical-Cryptography
	// style): markedly worse than any observed n-gram.
	m.floor = math.Log10(0.01 / float64(total))
	return m
}

// Score sums the n-gram log-probabilities of text (case-folded, letters only).
// Higher is more English-like. Texts shorter than n score 0.
func (m *Model) Score(text []byte) float64 {
	idx := Clean(text)
	if len(idx) < m.n {
		return 0
	}
	var s float64
	for i := 0; i+m.n <= len(idx); i++ {
		if lp, ok := m.logp[ngramKey(idx[i:i+m.n])]; ok {
			s += lp
		} else {
			s += m.floor
		}
	}
	return s
}

var (
	triOnce sync.Once
	triMod  *Model
)

// EnglishTrigram returns the cached trigram model.
func EnglishTrigram() *Model {
	triOnce.Do(func() { triMod = NewModel(3) })
	return triMod
}

var (
	freqOnce  sync.Once
	freqOrder []int
)

// FreqOrder returns the 26 letter indices ordered from most to least frequent
// in the corpus. Used to seed a frequency-matched starting key.
func FreqOrder() []int {
	freqOnce.Do(func() {
		var counts [26]int
		for _, v := range Clean([]byte(corpus)) {
			counts[v]++
		}
		order := make([]int, 26)
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(a, b int) bool { return counts[order[a]] > counts[order[b]] })
		freqOrder = order
	})
	out := make([]int, 26)
	copy(out, freqOrder)
	return out
}

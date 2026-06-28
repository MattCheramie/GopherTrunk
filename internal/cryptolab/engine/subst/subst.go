// Package subst solves monoalphabetic substitution ciphers by hill-climbing
// over the 26! key space, scored by an embedded English n-gram model. It
// seeds from a frequency match and refines with random key-swaps under random
// restarts — the standard approach for this class of cipher.
package subst

import (
	"math/rand"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lang"
)

// Key is a substitution key: Key[c] is the plaintext letter index (0..25) for
// ciphertext letter index c. It is always a permutation of 0..25.
type Key [26]byte

// Decrypt applies the key, preserving case and passing non-letters through.
func (k Key) Decrypt(ct []byte) []byte {
	out := make([]byte, len(ct))
	for i, b := range ct {
		li := lang.LetterIndex(b)
		if li < 0 {
			out[i] = b
			continue
		}
		p := k[li]
		if b >= 'a' && b <= 'z' {
			out[i] = 'a' + p
		} else {
			out[i] = 'A' + p
		}
	}
	return out
}

// letterFreqOrder returns ciphertext letter indices ordered most→least frequent.
func letterFreqOrder(ct []byte) []int {
	var counts [26]int
	for _, b := range ct {
		if i := lang.LetterIndex(b); i >= 0 {
			counts[i]++
		}
	}
	order := make([]int, 26)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return counts[order[a]] > counts[order[b]] })
	return order
}

// freqSeedKey maps the most frequent ciphertext letters onto the most frequent
// English letters — a strong starting point for the hill-climb.
func freqSeedKey(ct []byte) Key {
	ctOrder := letterFreqOrder(ct)
	engOrder := lang.FreqOrder()
	var k Key
	for i := 0; i < 26; i++ {
		k[ctOrder[i]] = byte(engOrder[i])
	}
	return k
}

func randKey(r *rand.Rand) Key {
	var k Key
	for i := range k {
		k[i] = byte(i)
	}
	r.Shuffle(26, func(i, j int) { k[i], k[j] = k[j], k[i] })
	return k
}

// Options tune the search. Zero values get sensible defaults.
type Options struct {
	Restarts int // independent hill-climbs (default 25)
	MaxFails int // consecutive non-improving swaps before a restart ends (default 1500)
}

// Solve recovers the most English-like substitution key for ct using model m
// and randomness r. It returns the best key, its decryption, and its score.
func Solve(ct []byte, m *lang.Model, r *rand.Rand, opt Options) (Key, []byte, float64) {
	if opt.Restarts <= 0 {
		opt.Restarts = 25
	}
	if opt.MaxFails <= 0 {
		opt.MaxFails = 1500
	}

	bestKey := freqSeedKey(ct)
	bestScore := m.Score(bestKey.Decrypt(ct))

	for restart := 0; restart < opt.Restarts; restart++ {
		key := bestKey
		if restart > 0 {
			key = randKey(r) // diversify after the freq-seeded first climb
		}
		score := m.Score(key.Decrypt(ct))

		fails := 0
		for fails < opt.MaxFails {
			i, j := r.Intn(26), r.Intn(26)
			if i == j {
				continue
			}
			key[i], key[j] = key[j], key[i]
			s := m.Score(key.Decrypt(ct))
			if s > score {
				score = s
				fails = 0
			} else {
				key[i], key[j] = key[j], key[i] // revert
				fails++
			}
		}
		if score > bestScore {
			bestScore, bestKey = score, key
		}
	}
	return bestKey, bestKey.Decrypt(ct), bestScore
}

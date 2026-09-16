package main

import (
	"math/rand"
	"testing"
)

// The replay harness's verdict rests on pitch continuity, so pin what it
// separates: a slowly moving fundamental (speech) versus uniformly random
// b0 indices (ciphertext through the vocoder), with silence frames ignored.
func TestPitchContinuitySeparatesSpeechFromCiphertext(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	// b0 sits at payload bits 0..3 (its top four) and 37..39 (mbelib
	// ambe3600x2450.c); everything else is noise here.
	mk := func(b0 int) []byte {
		f := make([]byte, 49)
		for i := range f {
			f[i] = byte(rng.Intn(2))
		}
		for i := 0; i < 4; i++ {
			f[i] = byte(b0>>uint(6-i)) & 1
		}
		for i := 0; i < 3; i++ {
			f[37+i] = byte(b0>>uint(2-i)) & 1
		}
		return f
	}
	var speech, random [][]byte
	p := 60
	for i := 0; i < 400; i++ {
		p += rng.Intn(7) - 3 // a fundamental drifting a few steps per 20 ms
		if p < 20 {
			p = 20
		}
		if p > 110 {
			p = 110
		}
		speech = append(speech, mk(p))
		if i%50 == 0 {
			speech = append(speech, mk(124)) // an interspersed silence frame
		}
		random = append(random, mk(rng.Intn(128)))
	}
	if c, n := pitchContinuity(speech); c < 0.9 || n < 350 {
		t.Fatalf("speech-like track: continuity %.2f over %d pairs", c, n)
	}
	if c, _ := pitchContinuity(random); c > 0.3 {
		t.Fatalf("random track: continuity %.2f, want ≈0.16", c)
	}
	if c, n := pitchContinuity(nil); c != 0 || n != 0 {
		t.Fatalf("empty: %.2f/%d", c, n)
	}
}
